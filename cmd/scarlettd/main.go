package main

import (
	"context"
	"log/slog"
	"net"
	"os"
	"os/signal"
	"syscall"

	"github.com/odit-bit/scarlett/api/cluster"
	"github.com/odit-bit/scarlett/api/rest"
	"github.com/odit-bit/scarlett/store"
	"github.com/odit-bit/scarlett/store/backend"
	"github.com/odit-bit/scarlett/tcp"
	"github.com/soheilhy/cmux"
)

func main() {
	sigC := make(chan os.Signal, 1)
	signal.Notify(sigC, syscall.SIGTERM, syscall.SIGINT)

	config := loadConfig()

	/*setup app*/
	// logger
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		AddSource: false,
		Level:     slog.LevelInfo,
	}))

	// Listener
	// todo: add tls/security
	// tls config defined for future needed.
	tc := tcp.DefaultTLSConfig()
	listener, err := net.Listen("tcp", config.RaftAddr)
	if err != nil {
		logger.Error(err.Error())
		return
	}
	defer listener.Close()

	// multiplex listener for rpc
	// cluster service and raft will listen the same port but with their own protocol,
	// when listener accept, mux will match the protocol and create connection.
	mux := cmux.New(listener)
	httpL := mux.Match(cmux.HTTP1Fast()) // http api
	grpcL := mux.Match(cmux.HTTP2())     // cluster-node api service
	raftL := mux.Match(cmux.Any())       // raft-consensus

	//setup store
	kv, err := backend.NewBadgerStore(config.StoreDir)
	if err != nil {
		logger.Error(err.Error())
		return
	}
	defer kv.Close()

	// TODO: for consistency hold incoming connection until the raft synced
	s, err := createStore(raftL, kv, logger, store.Config{
		RaftID:      config.ID,
		RaftAddr:    config.RaftAddr,
		RaftDir:     config.RaftDir,
		IsBootstrap: config.IsBoostrap,
		Purge:       config.IsPurge,
	})
	if err != nil {
		logger.Error(err.Error())
		return
	}
	defer s.Close()

	//should node join cluster
	if config.RaftJoinAddr != "" {
		if err := joinCluster(config.RaftJoinAddr, config.RaftAddr, config.ID); err != nil {
			logger.Error(err.Error())
			return
		}
		logger.Info("Success join cluster")
	}

	/* service */

	clstr, err := cluster.NewClient()
	if err != nil {
		logger.Error(err.Error())
		return
	}
	defer clstr.Close()

	//node http server
	hs := rest.NewServer(s, logger, clstr)
	rest.AddStore(&hs)
	go hs.Serve(httpL, tc)
	defer hs.Stop()

	//cluster grpc server
	cs := cluster.NewService(config.RaftAddr, s)
	cs.SetLogger(logger)
	go cs.Run(grpcL, nil)
	defer cs.Stop()

	// connection listener
	go func() {
		if err := mux.Serve(); err != nil {
			logger.Error(err.Error())
		}
	}()

	<-sigC
	close(sigC)
	mux.Close()
	logger.Info("shutdown")

}

func createStore(raftL net.Listener, kv store.KV, logger *slog.Logger, config store.Config) (*store.Store, error) {
	s, err := store.New(kv, raftL, config)
	if err != nil {
		logger.Error(err.Error())
		return nil, err
	}
	if logger != nil {
		s.SetLogger(logger)
	}

	return &s, nil

}

func joinCluster(remoteJoinAddr, nodeAddr, nodeID string) error {
	// log.Println("joining ", remoteJoinAddr, "node ", nodeID, "addr ", nodeAddr)
	cc, err := cluster.NewClient()
	if err != nil {
		return err
	}
	return cc.Join(context.Background(), remoteJoinAddr, nodeAddr, nodeID)
}
