package backend

//this package exist implementation of key-value db and raft state machine (FSM) backed by badger.

import (
	"fmt"
	"io"
	"path/filepath"
	"sync"
	"time"

	"github.com/dgraph-io/badger/v4"
	"github.com/hashicorp/raft"
	"github.com/odit-bit/scarlett/store/storepb"
	pb "google.golang.org/protobuf/proto"
)

func NewBadgerStore(path string) (*badgerStore, error) {
	path = filepath.Join(path, "fsm")
	path, err := filepath.Abs(path)
	if err != nil {
		return nil, err
	}

	opt := badger.DefaultOptions(path)
	opt = opt.WithLoggingLevel(badger.ERROR)
	db, err := badger.Open(opt)
	if err != nil {
		return nil, err
	}
	return &badgerStore{
		path: path,
		db:   db,
		mx:   sync.Mutex{},
	}, nil
}

func (b *badgerStore) Get(key []byte) ([]byte, bool, error) {
	var result []byte
	var ok bool

	err := b.db.View(func(txn *badger.Txn) error {
		item, err := txn.Get([]byte(key))
		if err != nil {
			return err
		}
		result = make([]byte, item.ValueSize())
		result, err = item.ValueCopy(result)
		ok = true
		return err
	})

	return result, ok, err
}

func (b *badgerStore) Set(key, value []byte, expired time.Duration) error {
	expireAt := int64(0)

	err := b.db.Update(func(txn *badger.Txn) error {
		err := txn.SetEntry(&badger.Entry{
			Key:       key,
			Value:     value,
			ExpiresAt: uint64(expireAt),
		})
		return err
	})

	return err
}

func (b *badgerStore) Delete(key []byte) error {
	return b.db.Update(func(txn *badger.Txn) error {
		_, err := txn.Get(key)

		if err != nil {
			return err
		}
		return txn.Delete(key)
	})
}

func (b *badgerStore) Backup(w io.Writer) error {
	b.mx.Lock()
	defer b.mx.Unlock()
	_, err := b.db.Backup(w, 0)
	return err
}

func (b *badgerStore) Load(r io.Reader) error {
	b.mx.Lock()
	defer b.mx.Unlock()
	// if err := b.db.DropAll(); err != nil {
	// 	return err
	// }
	return b.db.Load(r, 1)
}

func (b *badgerStore) Close() error {
	return b.db.Close()
}

var _ raft.FSM = (*badgerStore)(nil)

type badgerStore struct {
	path string
	db   *badger.DB
	mx   sync.Mutex
}

// Apply implements raft.FSM.
func (x *badgerStore) Apply(l *raft.Log) interface{} {
	cmd := storepb.CmdRequestPayload{}
	if err := pb.Unmarshal(l.Data, &cmd); err != nil {
		return storepb.CmdResponse{
			Msg: "",
			Err: err.Error(),
		}
	}

	res := &storepb.CmdResponse{}
	var oErr error
	switch cmd.Cmd {
	case storepb.Command_Type_Set:
		oErr = x.Set(cmd.Key, cmd.Value, 0)

	case storepb.Command_Type_Delete:
		oErr = x.Delete(cmd.Key)

	default:
		res = &storepb.CmdResponse{Err: fmt.Sprintf("unknown command %v", cmd.Cmd)}
	}
	if oErr != nil {
		res.Msg = oErr.Error()
	} else {
		res.Msg = "success"
	}

	return res
}

// Restore implements raft.FSM.
func (x *badgerStore) Restore(snapshot io.ReadCloser) error {
	if err := x.Load(snapshot); err != nil {
		return err
	}
	return snapshot.Close()
}

// Snapshot implements raft.FSM.
func (x *badgerStore) Snapshot() (raft.FSMSnapshot, error) {
	return &xfsm{
		snape: x,
	}, nil
}

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
