package main

import (
	"context"
	"fmt"
	"log"
	"log/slog"
	"net"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/odit-bit/scarlett/store"
	"github.com/odit-bit/scarlett/store/api/rest"
	"github.com/odit-bit/scarlett/store/storepb"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func main() {

	//config
	config := loadConfig()

	// logger
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		AddSource: false,
		Level:     slogLevelFromInt(config.LogLevel),
	}))
	mainLogger := logger.With("scope", "main")

	path, err := filepath.Abs(config.BaseDir)
	if err != nil {
		log.Fatal(err)
	}
	mainLogger.Info(fmt.Sprintf("dir path: %s", path))

	//setup store
	s, err := store.New(
		config.ID,
		config.RaftAddr,
		config.HttpAddr,
		store.WithBootstraping(config.IsBoostrap),
		store.WithSlog(slogLevelFromInt(config.LogLevel)),
		store.WithPurge(config.IsPurge),
		store.CustomDir(config.BaseDir),
	)

	if err != nil {
		mainLogger.Error(err.Error())
		return
	}

	//should node join cluster
	if config.RaftJoinAddr != "" {
		if err := joinCluster(config.RaftJoinAddr, config.RaftAddr, config.ID); err != nil {
			logger.Error(err.Error())
			return
		}
		mainLogger.Info("Success join cluster")
	}

	/* service */

	hSrv := rest.NewServer(s, logger)
	rest.AddStore(&hSrv)
	l, err := net.Listen("tcp", config.HttpAddr)
	if err != nil {
		mainLogger.Error(err.Error())
		return
	}
	go func() {
		if err := hSrv.Serve(l, nil); err != nil {
			mainLogger.Error(err.Error())
		}
	}()
	defer hSrv.Stop()

	sigC := make(chan os.Signal, 1)
	signal.Notify(sigC, syscall.SIGTERM, syscall.SIGINT)

	<-sigC
	mainLogger.Debug("closing...")
	if err := s.Close(); err != nil {
		mainLogger.Error(fmt.Sprintf("store-raft: %s", err.Error()))
	}

	close(sigC)
	mainLogger.Info("shutdown")

}

func joinCluster(remoteJoinAddr, nodeAddr, nodeID string) error {
	conn, err := grpc.NewClient(remoteJoinAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return err
	}
	defer conn.Close()
	cc := storepb.NewClusterClient(conn)
	if _, err := cc.Join(context.Background(), &storepb.JoinRequest{
		Address: nodeAddr,
		Id:      nodeID,
	}); err != nil {
		return err
	} else {
		return nil
	}

}
