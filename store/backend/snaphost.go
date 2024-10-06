package backend

import (
	"io"

	"github.com/hashicorp/raft"
)

var _ raft.FSMSnapshot = (*xfsm)(nil)

type Snapshoter interface {
	Backup(w io.Writer) error
}

type xfsm struct {
	snape Snapshoter
}

// Persist implements raft.FSMSnapshot.
func (x *xfsm) Persist(sink raft.SnapshotSink) error {

	if err := x.snape.Backup(sink); err != nil {
		sink.Cancel()
		return err
	}
	return sink.Close()
}

// Release implements raft.FSMSnapshot.
func (x *xfsm) Release() {}
