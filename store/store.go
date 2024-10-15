package store

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"time"

	// clusterpb "github.com/odit-bit/scarlett/store/api/cluster/proto"

	rpcTransport "github.com/Jille/raft-grpc-transport"
	"github.com/hashicorp/raft"
	"github.com/odit-bit/scarlett/store/backend"
	"github.com/odit-bit/scarlett/store/storepb"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/reflection"
	"google.golang.org/protobuf/proto"
)

var (
	ErrTimeout = errors.New("raft apply log to node is timeout")
)

type Backend interface {
	Get(key []byte) ([]byte, bool, error)
	io.Closer
}

// Store provide Key-value db service.
type Store struct {
	opts *storeOptions
	node *raftWrapper
	db   Backend

	raftAddr string
	httpAddr string

	errC       chan error
	nodeServer *grpc.Server
	tr         *rpcTransport.Manager
	isClosed   bool
}

func New(id, addr, httpAddr string, fn ...Options) (*Store, error) {
	opts := defaultOption
	for _, opt := range fn {
		opt(&opts)
	}

	// fsm implementation
	kv, err := backend.NewBadgerStore(opts.dir)
	if err != nil {
		return nil, err
	}
	opts.logger.Debug("setup kv store is done")

	// tranport
	tr := rpcTransport.New(
		raft.ServerAddress(addr),
		[]grpc.DialOption{grpc.WithTransportCredentials(insecure.NewCredentials())},
	)
	opts.logger.Debug("setup raft transport is done")

	node, err := newRaftNode(id, opts.dir, opts.applyTimeout, kv, tr.Transport())
	if err != nil {
		return nil, err
	}
	opts.logger.Debug("setup raft node is done")
	s := Store{
		opts:       &opts,
		node:       node,
		db:         kv,
		raftAddr:   addr,
		httpAddr:   httpAddr,
		errC:       make(chan error),
		nodeServer: nil,
		tr:         tr,
		isClosed:   false,
	}

	if opts.isBootstrap {
		s.opts.logger.Debug("bootstraping")
		if err := node.bootstraping(id, addr); err != nil {
			return nil, err
		}
	}
	opts.logger.Debug("setup store is done")

	go func() {
		listener, err := net.Listen("tcp", addr)
		if err != nil {
			s.opts.logger.Error(err.Error())
		}
		s.runGRPCServer(listener)
		if err := listener.Close(); err != nil {
			s.opts.logger.Error(err.Error())
		}
	}()
	// go s.runHttpServer(listener)
	opts.logger.Debug("setup node-rpc is done")

	return &s, nil
}

func (s *Store) runGRPCServer(listener net.Listener) {
	credential := insecure.NewCredentials()
	svr := grpc.NewServer(grpc.Creds(credential))
	nrh := nodeRpcHandler{
		node:                       s,
		cluster:                    s,
		UnimplementedCommandServer: storepb.UnimplementedCommandServer{},
		UnimplementedClusterServer: storepb.UnimplementedClusterServer{},
	}

	//grpc handler
	storepb.RegisterCommandServer(svr, &nrh)
	storepb.RegisterClusterServer(svr, &nrh)

	//grpc intra node transport
	s.tr.Register(svr)
	reflection.Register(svr)

	//run server
	s.nodeServer = svr
	s.errC <- svr.Serve(listener)
	close(s.errC)
}

// add node into cluster
func (s *Store) Join(nodeID, addr string) error {
	return s.node.AddNode(nodeID, addr)
}

func (s *Store) Remove(nodeID string) error {
	f := s.node.RemoveServer(raft.ServerID(nodeID), 0, s.opts.applyTimeout)
	if f.Error() != nil {
		return f.Error()
	}
	return nil
}

func (s *Store) Leader() (string, string) {
	addr, id := s.node.LeaderWithID()
	return string(addr), string(id)
}

// return listener addr
func (s *Store) Addr() string {
	return s.raftAddr
}

func (s *Store) Close() error {
	var err error
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	go func() {
		s.nodeServer.GracefulStop()
		cancel()
	}()

	select {
	case <-ctx.Done():
		e := ctx.Err()
		if e != context.Canceled {
			err = errors.Join(err, ctx.Err())
		}

	case e := <-s.errC:
		if e != nil {
			err = errors.Join(err, e)
		}
	}

	s.nodeServer.Stop()
	s.opts.logger.Debug("store-grpc-api closed")

	if xerr := s.node.Close(); xerr != nil {
		err = errors.Join(err, xerr)
	}
	s.opts.logger.Debug("store-node closed")

	if xerr := s.db.Close(); xerr != nil {
		err = errors.Join(err, xerr)
	}
	s.opts.logger.Debug("store-db closed")

	if xerr := s.tr.Close(); xerr != nil {
		err = errors.Join(err, xerr)
	}
	s.opts.logger.Debug("store-transport closed")

	if s.opts.isPurge {
		s.opts.logger.Info("purging")
		xerr := os.RemoveAll(s.opts.dir)
		if xerr != nil {
			err = errors.Join(err, fmt.Errorf("purge error:%s", xerr))
		}
	}
	return err
}

// command use to mutate the state of node in cluster
// error return is from raft system.
func (s *Store) Command(_ context.Context, cmd CMDType, args ...[]byte) (*CommandResponse, error) {
	// node use protobuf  wire format to exchange data inside cluster
	cr := &storepb.CmdRequestPayload{}
	switch cmd {
	case SET_CMD_TYPE:
		if len(args) != 2 {
			return nil, fmt.Errorf("wrong args count, expected 2")
		}
		cr.Cmd = storepb.Command_Type_Set
		cr.Key = args[0]
		cr.Value = args[1]

	case DELETE_CMD_TYPE:
		if len(args) < 1 {
			return nil, fmt.Errorf("wrong args count, expected 1")
		}
		cr.Cmd = storepb.Command_Type_Delete
		cr.Key = args[0]

	default:
		return nil, fmt.Errorf("invalid command %v", cmd)
	}

	//encode cmd into protobuf
	b, err := proto.Marshal(cr)
	if err != nil {
		return nil, err
	}

	f := s.node.Apply(b, s.opts.applyTimeout)
	if f.Error() != nil {
		return nil, f.Error()
	}

	// response should pb response
	res, ok := f.Response().(*storepb.CmdResponse)
	if !ok {
		return nil, fmt.Errorf("wrong command response type [%T]", f.Response())
	}

	// convert into expected response from func argument
	v := &CommandResponse{}
	v.Message = res.Msg
	return v, err
}

func (s *Store) Get(ctx context.Context, key []byte) ([]byte, bool, error) {
	return s.db.Get(key)
}

// // query use to get the current value directly from node
// func (s *Store) Query(ctx context.Context, cmd QueryType, args ...[]byte) (*QueryResponse, error) {
// 	var res QueryResponse
// 	switch cmd {
// 	case "get":
// 		b, ok, err := s.db.Get([]byte(args[0]))
// 		if err != nil {
// 			return nil, err
// 		}
// 		if !ok {
// 			return nil, fmt.Errorf("key not found")
// 		} else {
// 			res.Value = b
// 		}
// 	default:
// 		return nil, fmt.Errorf("invalid query command: %s", cmd)
// 	}

// 	return &res, nil
// }
