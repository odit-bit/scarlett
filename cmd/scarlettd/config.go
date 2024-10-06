package main

import (
	"flag"
	"fmt"
	"log"
	"net"
	"path/filepath"
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
}

func loadConfig() Config {
	var nodeDir string
	// var httpAddr string
	var raftAddr string
	var joinAddr string
	var nodeID string
	var bootstrap, purge bool

	flag.StringVar(&nodeID, "name", "", "unique name to identified this node")
	// flag.StringVar(&httpAddr, "http", "localhost:8080", "http port listen")
	flag.StringVar(&raftAddr, "addr", "localhost:8080", "bind address")

	flag.BoolVar(&bootstrap, "bootstrap", false, "start with bootstrap node")
	flag.BoolVar(&purge, "purge", false, "purge raft node data")

	flag.StringVar(&joinAddr, "join", "", "node address to join")
	flag.StringVar(&nodeDir, "raft-dir", "", "dir use for raft persistence")
	flag.Parse()

	// httpAddr = loadHttpAddr(httpAddr)
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
		ID:       nodeID,
		StoreDir: storeDir,
		RaftDir:  nodeDir,
		// HttpAddr:     httpAddr,
		RaftAddr:     raftAddr,
		RaftJoinAddr: joinAddr,
		IsBoostrap:   bootstrap,
		IsPurge:      purge,
	}
}
