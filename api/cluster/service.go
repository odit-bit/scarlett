package cluster

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"os"
	"time"

	clusterProto "github.com/odit-bit/scarlett/api/cluster/proto"
	"github.com/odit-bit/scarlett/store"
	storeproto "github.com/odit-bit/scarlett/store/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
	// "google.golang.org/grpc/status"
)

var _ clusterProto.ClusterServer = (*Service)(nil)
var _ storeproto.StorerServer = (*Service)(nil)

type Service struct {
	nodeHttpAddr string
	store        *store.Store
	logger       *slog.Logger
	srv          *grpc.Server
	errCh        chan error

	clusterProto.UnimplementedClusterServer
	storeproto.UnimplementedStorerServer
}

func NewService(httpAddr string, store *store.Store) *Service {
	return &Service{
		nodeHttpAddr:               httpAddr,
		store:                      store,
		logger:                     slog.New(slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{})),
		srv:                        &grpc.Server{},
		errCh:                      make(chan error),
		UnimplementedClusterServer: clusterProto.UnimplementedClusterServer{},
		UnimplementedStorerServer:  storeproto.UnimplementedStorerServer{},
	}
}

func credsFromConfig(tc *tls.Config) credentials.TransportCredentials {
	var credential credentials.TransportCredentials
	credential = insecure.NewCredentials()
	if tc != nil {
		credential = credentials.NewTLS(tc)
	}
	return credential

}

func (s *Service) Run(l net.Listener, tc *tls.Config) error {
	return s.runTLS(l, tc)
}

func (s *Service) runTLS(l net.Listener, tc *tls.Config) error {
	//option
	cred := grpc.Creds(credsFromConfig(tc))
	interceptor := grpc.ChainUnaryInterceptor(UnaryLogInterceptor)
	//server
	s.srv = grpc.NewServer(cred, interceptor)
	clusterProto.RegisterClusterServer(s.srv, s)
	storeproto.RegisterStorerServer(s.srv, s)

	s.errCh <- s.srv.Serve(l)
	close(s.errCh)
	return nil
}

func (s *Service) Stop() error {
	var err error
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	go func() {
		s.srv.GracefulStop()
		cancel()
	}()

	select {
	case <-ctx.Done():
		e := ctx.Err()
		if e != nil {
			err = errors.Join(err, ctx.Err())
			s.srv.Stop()
		}
	case e := <-s.errCh:
		err = errors.Join(err, e)
	}

	s.logger.Info(err.Error())
	return err

}

func (s *Service) SetLogger(l *slog.Logger) {
	if l == nil {
		return
	}
	s.logger = l.With("scope", "cluster")
	s.logger.Info("use custom logger")

}

func (s *Service) Join(ctx context.Context, in *clusterProto.JoinRequest) (*clusterProto.JoinResponse, error) {
	if err := s.store.AddNode(in.Id, in.Address); err != nil {
		return nil, err
	}

	return &clusterProto.JoinResponse{Msg: "success"}, nil
}

// GetNodeApiUrl implements proto.ClusterServer.
func (s *Service) GetNodeApiUrl(ctx context.Context, in *clusterProto.Request) (*clusterProto.Response, error) {
	return &clusterProto.Response{
		Url: s.nodeHttpAddr,
	}, nil
}

func (s *Service) Command(ctx context.Context, in *storeproto.CmdRequest) (*storeproto.CmdResponse, error) {
	out := new(storeproto.CmdResponse)
	if err := s.store.CommandRPC(in, out, 1*time.Second); err != nil {
		switch err {
		case store.ErrTimeout:
			return nil, status.Errorf(codes.DeadlineExceeded, err.Error())
		default:
			return nil, err
		}
	}

	return out, nil
}

// Query implements proto.StorerServer.
func (s *Service) Query(ctx context.Context, in *storeproto.QueryRequest) (*storeproto.QueryResponse, error) {
	var cmd store.QueryType
	switch in.Query {
	case storeproto.QueryType_Get:
		cmd = store.GET_QUERY_TYPE
	default:
		return nil, fmt.Errorf("invalid query type %s", in.Query)

	}
	resp, err := s.store.Query(ctx, cmd, in.Args...)
	if err != nil {
		return nil, err
	}

	return &storeproto.QueryResponse{
		Value: resp.Value,
	}, nil

}

func (s *Service) GetNodeRaftAddr(ctx context.Context, in *clusterProto.NodeRaftAddrRequest) (*clusterProto.NodeRaftAddrResponse, error) {
	addr, _ := s.store.GetLeader()
	return &clusterProto.NodeRaftAddrResponse{Addr: addr}, nil
}
