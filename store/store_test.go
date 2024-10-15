package store_test

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/hashicorp/raft"
	"github.com/odit-bit/scarlett/store"
	sp "github.com/odit-bit/scarlett/store/storepb"
)

var _ store.Backend = (*kvMock)(nil)
var _ raft.FSMSnapshot = (*kvMock)(nil)

type kvMock struct{}

// Close implements store.KV.
func (k *kvMock) Close() error {
	return nil
}

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

	node1_path := filepath.Join(path, "/node-1")
	// kv := &kvMock{}
	s, err := store.New(
		"test-node-1",
		"localhost:9000",
		"",
		store.CustomDir(node1_path),
		store.WithPurge(true),
		store.WithBootstraping(true))
	if err != nil {
		t.Fatal(err)
	}
	_, id := s.Leader()
	if id != "test-node-1" {
		t.Fatal("node should be leader")
	}
	defer s.Close()

	//NODE-2
	// kv2 := &kvMock{}
	node2_path := filepath.Join(path, "/node-2")
	// s2, err := store.New2(kv2, raftL2, store.Config{
	// 	RaftID:      "test-node-2",
	// 	RaftAddr:    listener2.Addr().String(),
	// 	RaftDir:     node2_path,
	// 	IsBootstrap: false,
	// 	Purge:       true,
	// })
	s2, err := store.New(
		"test-node-2",
		"localhost:10000",
		"",
		store.WithPurge(true),
		store.CustomDir(node2_path),
	)
	if err != nil {
		t.Fatal(err)
	}
	defer s2.Close()
	//leader(node-1) add node-2 to cluster
	if err := s.Join("test-node-2", "127.0.0.1:10000"); err != nil {
		t.Fatal(err)
	}

	// command
	cmd := store.Command{
		Cmd:  "set",
		Args: [][]byte{[]byte("my-key"), []byte("my-value")},
	}
	var res store.CommandResponse
	assertCmd(s, &cmd, &res, t)

	cmd.Cmd = "delete"
	assertCmd(s, &cmd, &res, t)

	cmd.Cmd = "get"
	assertQuery(s, &cmd, t)

}

func assertCmd(s *store.Store, cmd *store.Command, res *store.CommandResponse, t *testing.T) {
	v, err := s.Command(context.Background(), store.CMDType(cmd.Cmd), []byte(cmd.Args[0]), []byte(cmd.Args[1]))
	if err != nil {
		t.Fatal(err)
	}
	*res = *v
}

func assertQuery(s *store.Store, cmd *store.Command, t *testing.T) {
	_, _, err := s.Get(context.Background(), []byte(cmd.Args[0]))
	if err.Error() != "key not found" {
		t.Fatal(err)
	}

}
