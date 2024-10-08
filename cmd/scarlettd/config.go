package main

import (
	"flag"
	"fmt"
	"log"
	"net"
	"path/filepath"
)

const (
	_Default_Max_Conn = 100
)

func loadRaftAddr(raftAddr string) string {

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
func loadHttpAddr(httpAddr string) string {

	host, port, err := net.SplitHostPort(httpAddr)
	if err != nil {
		log.Fatal(err)
	}

	if port == "" {
		log.Fatal("http port cannot be empty")
	}

	if host == "" || host == "localhost" {

		addrs, err := net.LookupHost(host)
		if err != nil {
			log.Fatal(err)
		}
		host = addrs[0]
	}

	httpAddr = fmt.Sprintf("%s:%s", host, port)
	return httpAddr
}

type Config struct {
	ID       string
	StoreDir string
	RaftDir  string

	// HttpAddr     string
	RaftAddr     string
	RaftJoinAddr string

	IsBoostrap bool
	IsPurge    bool

	//max conn for incoming traffic conn
	MaxConn int
}

func loadConfig() Config {
	var nodeDir string
	// var httpAddr string
	var host string
	var port int
	var joinAddr string
	var nodeID string
	var bootstrap, purge bool

	flag.StringVar(&nodeID, "name", "", "unique name to identified this node")
	// flag.StringVar(&httpAddr, "http", "localhost:8080", "http port listen")
	flag.StringVar(&host, "host", "localhost", "bind address")
	flag.IntVar(&port, "port", 8001, "bind address")

	flag.BoolVar(&bootstrap, "bootstrap", false, "start with bootstrap node")
	flag.BoolVar(&purge, "purge", false, "purge raft node data")

	flag.StringVar(&joinAddr, "join", "", "node address to join")
	flag.StringVar(&nodeDir, "raft-dir", "", "dir use for raft persistence")

	maxConn := flag.Int("limit", _Default_Max_Conn, "maximum incoming conn")
	flag.Parse()

	raftAddr := fmt.Sprintf("%s:%d", host, port)
	raftAddr = loadRaftAddr(raftAddr)
	if joinAddr != "" {
		joinAddr = loadRaftAddr(joinAddr)
	}

	storeDir := ""
	if nodeDir == "" {
		baseDir := filepath.Join("./data", nodeID)
		storeDir = filepath.Join(baseDir, "badger")
		nodeDir = filepath.Join(baseDir, "raft")
	}

	return Config{
		ID:           nodeID,
		StoreDir:     storeDir,
		RaftDir:      nodeDir,
		RaftAddr:     raftAddr,
		RaftJoinAddr: joinAddr,
		IsBoostrap:   bootstrap,
		IsPurge:      purge,
		MaxConn:      *maxConn,
	}
}
