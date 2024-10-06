package store

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"os"
	"path/filepath"
	"time"

	"github.com/hashicorp/go-hclog"
	"github.com/hashicorp/raft"
	raftBolt "github.com/hashicorp/raft-boltdb/v2"
	storeproto "github.com/odit-bit/scarlett/store/proto"
	"google.golang.org/protobuf/proto"
)

type KV interface {
	Get(key []byte) ([]byte, bool, error)
	// Set(key, value []byte, expired time.Duration) error
	// Delete(key []byte) error

	// Backup(w io.Writer) error
	// Load(r io.Reader) error

	raft.FSM
}

// type CLusterClient interface {
// 	Join(ctx context.Context, joinAddr, nodeAddr, nodeID string) error
// }

type FSM interface {
	raft.FSM
}

// Store wrap raft consensus.
type Store struct {
	id       string
	raftAddr string

	raft        *raft.Raft
	tr          raft.Transport
	logStore    raft.LogStore
	snapStore   raft.SnapshotStore
	stableStore raft.StableStore

	// cluster CLusterClient

	logger *slog.Logger

	db KV

	bootstrap bool
	isClosed  bool
	closers   []io.Closer
}

type Config struct {

	// id for identified this node in cluster
	RaftID string
	// addr listen for raft protocol in cluster
	RaftAddr string
	// addr for persist raft data like 'log' or 'snapshot'
	RaftDir string

	// does this node bootstraped as cluster
	IsBootstrap bool

	// purge raft data when shutdown
	Purge bool
}

func New(kv KV, listener net.Listener, config Config) (Store, error) {

	sl := StreamLayer{
		Listener: listener,
	}

	raftLogger := hclog.New(&hclog.LoggerOptions{
		Name:   "scarlett-raft",
		Output: os.Stderr,
		Level:  hclog.Debug,
	})
	//transport
	transport := raft.NewNetworkTransportWithLogger(&sl, 5, 10*time.Second, raftLogger)
	// dir for persist raft data
	snapshot, err := raft.NewFileSnapshotStore(config.RaftDir, 2, os.Stderr)
	if err != nil {
		return Store{}, err
	}

	// raft parameter
	// log and stable store use bolt
	bolt, err := raftBolt.NewBoltStore(filepath.Join(config.RaftDir, "raft_bolt"))
	if err != nil {
		return Store{}, err
	}

	//logger
	l := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{}))
	l = l.With("scope", "store")

	//purge
	var purger dirPurger
	if config.Purge {
		purger.dir = config.RaftDir
	}

	// raft
	rconf := raft.DefaultConfig()
	rconf.LocalID = raft.ServerID(config.RaftID)
	rconf.Logger = raftLogger

	rf, err := raft.NewRaft(rconf, kv, bolt, bolt, snapshot, transport)
	if err != nil {
		return Store{}, err
	}

	// instance store
	s := Store{
		id:          config.RaftID,
		raftAddr:    config.RaftAddr,
		raft:        rf,
		tr:          transport,
		logStore:    bolt,
		snapStore:   snapshot,
		stableStore: bolt,
		logger:      l,
		db:          kv,
		bootstrap:   config.IsBootstrap,
		isClosed:    false,
		closers:     []io.Closer{bolt, &purger, transport},
	}

	if err := s.bootstraping(); err != nil {
		return Store{}, err
	}

	return s, nil
}

func (s *Store) bootstraping() error {
	if s.bootstrap {
		bootstrapConfig := raft.Configuration{
			Servers: []raft.Server{
				{
					ID:      raft.ServerID(s.id),
					Address: s.tr.LocalAddr(),
				},
			},
		}
		f := s.raft.BootstrapCluster(bootstrapConfig)
		bErr := f.Error()
		if bErr != nil {
			if bErr.Error() != raft.ErrCantBootstrap.Error() {
				return bErr
			}
		}
		// if err := s.AddNode(s.id, s.raftAddr); err != nil {
		// 	return err
		// }
		ok := false
		count := 0
		for !ok && count != 3 {
			addr, id := s.raft.LeaderWithID()
			if addr == "" {
				s.logger.Info("cluster not have leader")
				time.Sleep(1 * time.Second)
				//wait for vote
			} else {
				s.logger.Info("cluster leader", "addr", addr, "id", id)
				ok = true
			}
			count++
		}

	}

	s.logger.Info("cluster", "server", s.raft.GetConfiguration().Configuration().Servers)
	return nil
}

func (s *Store) AddNonVoter(id, addr string) error {
	f := s.raft.AddNonvoter(raft.ServerID(id), raft.ServerAddress(addr), 0, 0)
	if f.Error() != nil {
		return f.Error()
	}
	return nil
}

// add node into cluster
func (s *Store) AddNode(nodeID, addr string) error {
	s.logger.Info(fmt.Sprintf("received join request for remote node %s at %s", nodeID, addr))

	configFuture := s.raft.GetConfiguration()
	if err := configFuture.Error(); err != nil {
		s.logger.Error(fmt.Sprintf("failed to get raft configuration: %v", err))
		return err
	}

	for _, srv := range configFuture.Configuration().Servers {
		// If a node already exists with either the joining node's ID or address,
		// that node may need to be removed from the config first.
		if srv.ID == raft.ServerID(nodeID) || srv.Address == raft.ServerAddress(addr) {
			// However if *both* the ID and the address are the same, then nothing -- not even
			// a join operation -- is needed.
			if srv.Address == raft.ServerAddress(addr) && srv.ID == raft.ServerID(nodeID) {
				s.logger.Error(fmt.Sprintf("node %s at %s already member of cluster, ignoring join request", nodeID, addr))
				return nil
			}

			future := s.raft.RemoveServer(srv.ID, 0, 2*time.Second)
			if err := future.Error(); err != nil {
				return fmt.Errorf("error removing existing node %s at %s: %s", nodeID, addr, err)
			} else {
				s.logger.Info(fmt.Sprintf("remove-server id:%s, addr:%s \n", srv.ID, srv.Address))
			}
		}
	}

	future := s.raft.AddVoter(raft.ServerID(nodeID), raft.ServerAddress(addr), 0, 2*time.Second)
	if future.Error() != nil {
		return future.Error()
	}

	s.logger.Info(fmt.Sprintf("node %s at addr %s join succesfull", nodeID, addr))
	return nil
}

func (s *Store) GetLeader() (string, string) {
	addr, id := s.raft.LeaderWithID()
	return string(addr), string(id)
}

type dirPurger struct {
	dir string
}

func (p *dirPurger) Close() error {
	if p.dir == "" {
		return nil
	}

	return os.RemoveAll(p.dir)
}

func (s *Store) SetLogger(sl *slog.Logger) {
	s.logger = sl
}

func (s *Store) Addr() string {
	return s.raftAddr
}

func (s *Store) ID() string {
	return s.id
}

func (s *Store) Close() error {

	if !s.isClosed {
		var err error

		// // remove all servers
		// // todo: make non-leader node to send this request to leader
		// if s.raft.State() == raft.Leader {
		// 	servers := s.raft.GetConfiguration().Configuration().Servers
		// 	for _, server := range servers {
		// 		f := s.raft.RemoveServer(server.ID, 0, 1*time.Second)
		// 		err = errors.Join(err, f.Error())
		// 	}
		// }

		err = errors.Join(s.raft.Shutdown().Error())
		for _, c := range s.closers {
			errors.Join(err, c.Close())
		}

		s.isClosed = true
		if err != nil {
			s.logger.Debug(err.Error())
		}
		return err
	} else {
		return fmt.Errorf("store closed")
	}
}

func (s *Store) CommandRPC(in *storeproto.CmdRequest, out *storeproto.CmdResponse) error {

	f := s.raft.Apply(in.Payload, 2*time.Second)
	if f.Error() != nil {
		return f.Error()
	}

	// response should pb response
	res, ok := f.Response().(*storeproto.CmdResponse)
	if !ok {
		return fmt.Errorf("wrong command response type [%T]", f.Response())
	}

	out.Msg = res.Msg
	out.Err = res.Err
	return nil
}

// command use to mutate the state of node in cluster
// error return is from raft system.
func (s *Store) Command(cmd CMDType, key, value []byte, v *CommandResponse) error {
	if v == nil {
		return fmt.Errorf("v cannot be nil")
	}

	// node use protobuf  wire format to exchange data inside cluster
	cr := &storeproto.CmdRequestPayload{
		Key:   key,
		Value: value,
	}
	switch cmd {
	case "set":
		cr.Cmd = storeproto.Command_Type_Set
	case "delete":
		cr.Cmd = storeproto.Command_Type_Delete

	default:
		return fmt.Errorf("unknown command %v", cmd)
	}

	//encode cmd into protobuf
	b, err := proto.Marshal(cr)
	if err != nil {
		return err
	}
	f := s.raft.Apply(b, 2*time.Second)
	if f.Error() != nil {
		return f.Error()
	}

	// response should pb response
	res, ok := f.Response().(*storeproto.CmdResponse)
	if !ok {
		return fmt.Errorf("wrong command response type [%T]", f.Response())
	}

	// convert into expected response from func argument
	v.Cmd = "set"
	v.Message = res.Msg
	if res.Err != "" {
		v.Err = errors.New(res.Err)
	}
	return nil
}

// query use to get the current value directly from node
func (s *Store) Query(ctx context.Context, cmd QueryType, args ...[]byte) (*QueryResponse, error) {
	var res QueryResponse
	switch cmd {
	case "get":
		b, ok, err := s.db.Get([]byte(args[0]))
		if err != nil {
			res.Err = err
		}
		if !ok {
			res.Message = "key not found"
		} else {
			res.Value = b
		}
	default:
		res.Err = fmt.Errorf("invalid query command: %s", cmd)
	}

	return &res, nil
}

// func (s *Store) Join(ctx context.Context) error {
// 	addr, id := s.GetLeader()
// 	if addr != "" {
// 		if err := s.cluster.Join(ctx, addr, s.raftAddr, s.id); err != nil {
// 			return fmt.Errorf("store failed join cluster at node %v, address %v", id, addr)
// 		}
// 	}

// }
