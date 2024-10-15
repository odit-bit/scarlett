package main

import (
	"flag"
	"fmt"
	"log"
	"log/slog"
	"net"
	"path/filepath"
)

const (
	_Default_Max_Conn = 100
)

func parseAddress(raftAddr string) string {

	host, port, err := net.SplitHostPort(raftAddr)
	if err != nil {
		log.Fatal(err)
	}

	if port == "" {
		log.Fatal("raft port cannot be empty ", host, port)
	}

	if host == "" || host == "localhost" {
		addrs, err := net.LookupHost("localhost")
		if err != nil {
			log.Fatal(err)
		}
		host = addrs[0]
	}

	nodeAddr := fmt.Sprintf("%s:%s", host, port)
	return nodeAddr
}

type Config struct {
	ID      string
	BaseDir string
	// StoreDir string
	// RaftDir  string

	HttpAddr     string
	RaftAddr     string
	RaftJoinAddr string

	IsBoostrap bool
	IsPurge    bool

	//max conn for incoming traffic conn
	LogLevel int
	MaxConn  int
}

func slogLevelFromInt(n int) slog.Level {
	if n < 0 && n > 3 {
		n = 3
	}
	switch n {
	case 2:
		return slog.LevelInfo
	case 1:
		return slog.LevelWarn
	case 0:
		return slog.LevelError
	default:
		return slog.LevelDebug
	}
}

func loadConfig() Config {
	var nodeDir string

	var host string
	var raftPort int
	var httpPort int
	var joinAddr string

	var nodeID string
	var bootstrap, purge bool

	var logLevel int

	flag.StringVar(&nodeID, "name", "", "unique name to identified this node")
	// flag.StringVar(&httpAddr, "http", "localhost:8080", "http port listen")
	flag.StringVar(&host, "host", "localhost", "bind address")
	flag.IntVar(&raftPort, "raft", 8001, "bind port for rpc")
	flag.IntVar(&httpPort, "http", 9001, "bind port for http ")

	flag.StringVar(&joinAddr, "join", "", "node address to join")
	flag.StringVar(&nodeDir, "raft-dir", "/tmp/scarlett/", "dir use for raft persistence")

	flag.BoolVar(&bootstrap, "bootstrap", false, "start with bootstrap node")
	flag.BoolVar(&purge, "purge", false, "purge raft node data")

	flag.IntVar(&logLevel, "log", 2, "log level verbose 0 - 3, default 2 (most verbose)")

	maxConn := flag.Int("limit", _Default_Max_Conn, "maximum incoming conn")
	flag.Parse()

	raftAddr := fmt.Sprintf("%s:%d", host, raftPort)
	raftAddr = parseAddress(raftAddr)

	httpAddr := fmt.Sprintf("%s:%d", host, httpPort)
	httpAddr = parseAddress(httpAddr)

	if joinAddr != "" {
		joinAddr = parseAddress(joinAddr)
	}

	if nodeID == "" {
		nodeID = raftAddr
	}
	baseDir := filepath.Join(nodeDir, nodeID)

	return Config{
		ID:           nodeID,
		BaseDir:      baseDir,
		HttpAddr:     httpAddr,
		RaftAddr:     raftAddr,
		RaftJoinAddr: joinAddr,
		IsBoostrap:   bootstrap,
		IsPurge:      purge,
		LogLevel:     logLevel,
		MaxConn:      *maxConn,
	}
}
