package store_test

import (
	"context"
	"io"
	"log"
	"net"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/hashicorp/raft"
	"github.com/odit-bit/scarlett/store"
	sp "github.com/odit-bit/scarlett/store/proto"
	"github.com/soheilhy/cmux"
)

var _ store.KV = (*kvMock)(nil)
var _ raft.FSMSnapshot = (*kvMock)(nil)

type kvMock struct{}

// Persist implements raft.FSMSnapshot.
func (k *kvMock) Persist(sink raft.SnapshotSink) error {
	return nil
}

// Release implements raft.FSMSnapshot.
func (k *kvMock) Release() {
}

// Apply implements store.KV.
func (k *kvMock) Apply(*raft.Log) interface{} {
	return &sp.CmdResponse{
		Msg: "ok",
		Err: "",
	}
}

// Restore implements store.KV.
func (k *kvMock) Restore(snapshot io.ReadCloser) error {
	panic("unimplemented")
}

// Snapshot implements store.KV.
func (k *kvMock) Snapshot() (raft.FSMSnapshot, error) {
	return k, nil
}

// Backup implements store.KV.
func (k *kvMock) Backup(w io.Writer) error {
	return nil
}

// Delete implements store.KV.
func (k *kvMock) Delete(key []byte) error {
	return nil
}

// Get implements store.KV.
func (k *kvMock) Get(key []byte) ([]byte, bool, error) {
	return nil, false, nil
}

// Load implements store.KV.
func (k *kvMock) Load(r io.Reader) error {
	return nil
}

// Set implements store.KV.
func (k *kvMock) Set(key []byte, value []byte, expired time.Duration) error {
	return nil
}

func Test_(t *testing.T) {
	path, err := os.MkdirTemp("./", "x")
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		os.RemoveAll(path)
	}()
	var wg sync.WaitGroup
	defer func() {
		wg.Wait()
	}()

	// NODE-1
	listener, err := net.Listen("tcp", ":9000")
	if err != nil {
		log.Fatal(err)
	}
	defer listener.Close()

	mux := cmux.New(listener)
	raftL := mux.Match(cmux.Any())
	wg.Add(1)
	go func() {
		mux.Serve()
		wg.Done()
	}()

	node1_path := filepath.Join(path, "/node-1")
	kv := &kvMock{}
	s, err := store.New(kv, raftL, store.Config{
		RaftID:      "test-node-1",
		RaftAddr:    listener.Addr().String(),
		RaftDir:     node1_path,
		IsBootstrap: true,
		Purge:       true,
	})
	if err != nil {
		t.Fatal(err)
	}
	_, id := s.GetLeader()
	if id != "test-node-1" {
		t.Fatal("node should be leader")
	}
	defer s.Close()

	//NODE-2
	listener2, err := net.Listen("tcp", ":10000")
	if err != nil {
		log.Fatal(err)
	}
	mux2 := cmux.New(listener2)
	raftL2 := mux2.Match(cmux.Any())
	wg.Add(1)
	go func() {
		mux2.Serve()
		wg.Done()
	}()

	defer listener2.Close()
	kv2 := &kvMock{}
	node2_path := filepath.Join(path, "/node-2")
	s2, err := store.New(kv2, raftL2, store.Config{
		RaftID:      "test-node-2",
		RaftAddr:    listener2.Addr().String(),
		RaftDir:     node2_path,
		IsBootstrap: false,
		Purge:       true,
	})
	if err != nil {
		t.Fatal(err)
	}
	defer s2.Close()
	//leader(node-1) add node-2 to cluster
	if err := s.AddNode("test-node-2", "127.0.0.1:10000"); err != nil {
		t.Fatal(err)
	}

	// command
	cmd := store.Command{
		Cmd:   "set",
		Key:   "my-key",
		Value: "my-value",
	}
	var res store.CommandResponse
	assertCmd(&s, &cmd, &res, t)

	cmd.Cmd = "delete"
	assertCmd(&s, &cmd, &res, t)

	cmd.Cmd = "get"
	assertQuery(&s, &cmd, t)

}

func assertCmd(s *store.Store, cmd *store.Command, res *store.CommandResponse, t *testing.T) {
	err := s.Command(store.CMDType(cmd.Cmd), []byte(cmd.Key), []byte(cmd.Value), res)
	if err != nil {
		t.Fatal(err)
	}
	if res.Err != nil {
		t.Fatal(res.Err)
	}

}

func assertQuery(s *store.Store, cmd *store.Command, t *testing.T) {
	res, err := s.Query(context.Background(), store.QueryType(cmd.Cmd), []byte(cmd.Key))
	if err != nil {
		t.Fatal(err)
	}
	if res.Err != nil {
		t.Fatal(res.Err)
	}
}
