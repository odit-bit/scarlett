package store

import (
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	"github.com/hashicorp/go-hclog"
	"github.com/hashicorp/raft"
	raftboltdb "github.com/hashicorp/raft-boltdb/v2"
)

// wrap the underlying raft implementation
type raftWrapper struct {
	applyTimeout time.Duration

	*raft.Raft
}

func newRaftNode(id, dir string, timeout time.Duration, kv raft.FSM, tr raft.Transport) (*raftWrapper, error) {

	raftLogger := hclog.New(&hclog.LoggerOptions{
		Name:   "scarlett.raft",
		Output: os.Stderr,
		Level:  hclog.Error,
	})

	// dir for persist raft data
	snapshotLogger := raftLogger.Named("snapshot")
	snapshot, err := raft.NewFileSnapshotStoreWithLogger(dir, 2, snapshotLogger)
	if err != nil {
		return nil, err
	}

	// raft parameter
	// log and stable store use bolt
	logPath := filepath.Join(dir, "log")
	if err := os.MkdirAll(logPath, 0755); err != nil {
		return nil, err
	}

	stableBolt, err := raftboltdb.NewBoltStore(filepath.Join(logPath, "stable_store"))
	if err != nil {
		return nil, err
	}

	//logger
	l := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{}))
	_ = l.With("scope", "store")

	// raft
	nodeLogger := raftLogger.Named("node")
	rconf := raft.DefaultConfig()
	rconf.LocalID = raft.ServerID(id)
	rconf.Logger = nodeLogger
	rconf.LogLevel = hclog.Info.String()

	node, err := raft.NewRaft(rconf, kv, stableBolt, stableBolt, snapshot, tr)
	if err != nil {
		return nil, err
	}

	rw := raftWrapper{
		applyTimeout: timeout,
		Raft:         node,
	}

	return &rw, nil
}

func (rw *raftWrapper) Close() error {
	var err error

	f := rw.Shutdown()

	if f.Error() != nil {
		err = errors.Join(err, f.Error())
	}

	return err
}

// add remote node to the cluster (should leader)
func (rw *raftWrapper) AddNode(id, addr string) error {

	configFuture := rw.GetConfiguration()
	if err := configFuture.Error(); err != nil {
		return err
	}

	for _, srv := range configFuture.Configuration().Servers {
		// If a node already exists with either the joining node's ID or address,
		// that node may need to be removed from the config first.
		if srv.ID == raft.ServerID(id) || srv.Address == raft.ServerAddress(addr) {
			// However if *both* the ID and the address are the same, then nothing -- not even
			// a join operation -- is needed.
			if srv.Address == raft.ServerAddress(addr) && srv.ID == raft.ServerID(id) {
				return nil
			}

			future := rw.RemoveServer(srv.ID, 0, 2*time.Second)
			if err := future.Error(); err != nil {
				return fmt.Errorf("error removing existing node %s at %s: %s", id, addr, err)
			}
		}
	}

	f := rw.AddVoter(raft.ServerID(id), raft.ServerAddress(addr), 0, rw.applyTimeout)
	if f.Error() != nil {
		return fmt.Errorf("failed to add node: %s addr: %s err: %s", id, addr, f.Error().Error())
	}
	return nil
}

func (rw *raftWrapper) bootstraping(id, addr string) error {
	// log.Println("bootstrap CALLED")
	bootstrapConfig := raft.Configuration{
		Servers: []raft.Server{
			{
				ID:      raft.ServerID(id),
				Address: raft.ServerAddress(addr),
			},
		},
	}
	f := rw.BootstrapCluster(bootstrapConfig)
	bErr := f.Error()
	if bErr != nil {
		if bErr.Error() != raft.ErrCantBootstrap.Error() {
			return bErr
		}
	}

	ok := false
	count := 0
	for !ok && count != 3 {
		addr, _ := rw.LeaderWithID()
		if addr == "" {
			time.Sleep(1 * time.Second)
			//wait for vote
		} else {
			// log.Println("LEADER:", addr)
			ok = true
		}
		count++
	}

	return nil
}
