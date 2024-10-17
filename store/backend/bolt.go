package backend

import (
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"sync"
	"syscall"
	"time"

	"github.com/hashicorp/raft"
	"github.com/odit-bit/scarlett/store/storepb"
	"go.etcd.io/bbolt"
	"google.golang.org/protobuf/proto"
)

const (
	defaultOpenTimeout time.Duration = 5 * time.Second
)

type boltStore struct {
	bolt *bbolt.DB
	mx   sync.Mutex
}

func NewBoltStore(path string) (*boltStore, error) {

	path = filepath.Join(path, "fsm")
	path, err := filepath.Abs(path)
	if err != nil {
		return nil, err
	}

	if err := os.MkdirAll(path, 0755); err != nil {
		return nil, fmt.Errorf("bolt-store err: %s", err)
	}

	// fpath := filepath.Join(path, "kv-bolt")

	bolt, err := bbolt.Open(filepath.Join(path, "fsm-bolt"), 0755, &bbolt.Options{
		Timeout: defaultOpenTimeout,
		NoSync:  true,
	})
	if err != nil {
		return nil, err
	}

	bs := boltStore{
		bolt: bolt,
		mx:   sync.Mutex{},
	}

	if err := bs.initBucket(); err != nil {
		return nil, err
	}

	return &bs, err
}

func (b *boltStore) initBucket() error {
	tx, err := b.bolt.Begin(true)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	_, err = tx.CreateBucketIfNotExists(_bolt_kv)
	if err != nil {
		return fmt.Errorf("create bucket: %s", err)
	}

	if err := tx.Commit(); err != nil {
		return err
	}
	return nil
}

func (b *boltStore) Close() error {
	return b.bolt.Close()
}

func (b *boltStore) Get(key []byte) ([]byte, bool, error) {
	var res []byte
	var ok bool
	err := b.bolt.View(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket(_bolt_kv)
		if bucket == nil {
			return fmt.Errorf("bucket did not exist")
		}
		v := bucket.Get(key)
		if len(v) == 0 {
			return ErrKeyNotFound
		}
		res = make([]byte, len(v))
		copy(res, v)
		ok = true
		return nil
	})
	return res, ok, err
}

func (b *boltStore) Set(key []byte, value []byte) error {
	b.mx.Lock()
	defer b.mx.Unlock()
	err := b.bolt.Update(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket(_bolt_kv)
		err := bucket.Put(key, value)
		return err
	})
	return err
}

func (b *boltStore) Delete(key []byte) error {
	b.mx.Lock()
	defer b.mx.Unlock()
	return b.bolt.Update(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket(_bolt_kv)
		return bucket.Delete(key)
	})
}

var _ raft.FSM = (*boltStore)(nil)

// Apply implements raft.FSM.
func (b *boltStore) Apply(l *raft.Log) interface{} {
	req := storepb.CmdRequestPayload{}
	resp := &storepb.CmdResponse{}
	if err := proto.Unmarshal(l.Data, &req); err != nil {
		resp.Err = err.Error()
		return &resp
	}

	var err error
	switch req.Cmd {
	case storepb.Command_Type_Set:
		err = b.Set(req.Key, req.Value)
	case storepb.Command_Type_Delete:
		err = b.Delete(req.Key)
	default:
		err = errors.Join(ErrInvalidCmd, fmt.Errorf("got %v", req.Cmd.Descriptor()))
	}

	if err != nil {
		resp.Err = err.Error()
	} else {
		resp.Msg = "ok"
	}
	return resp
}

// Restore implements raft.FSM.
func (b *boltStore) Restore(snapshot io.ReadCloser) error {
	// should mutex
	log.Println("RESTORE")
	b.mx.Lock()
	defer b.mx.Unlock()
	defer snapshot.Close()
	defer log.Println("RESTORE FINISH")

	path := b.bolt.Path()

	// beware, close underlying bolt !!
	if err := b.bolt.Close(); err != nil {
		return err
	}

	//open file
	f, err := os.OpenFile(path, os.O_RDWR, 0755)
	if err != nil {
		return err
	}
	if _, err := f.Seek(0, 0); err != nil {
		return err
	}
	// copy from snapshot
	if _, err := io.Copy(f, snapshot); err != nil {
		f.Close()
		return err
	}
	f.Close()

	//open with bolt
	bolt, err := bbolt.Open(path, 0755, &bbolt.Options{
		Timeout: defaultOpenTimeout,
		NoSync:  true})
	if err != nil {
		return err
	}
	b.bolt = bolt
	return nil
}

// Snapshot implements raft.FSM.
func (b *boltStore) Snapshot() (raft.FSMSnapshot, error) {
	return &boltSnapshot{
		bolt: b.bolt,
	}, nil
}

var _ raft.FSMSnapshot = (*boltSnapshot)(nil)

type boltSnapshot struct {
	bolt *bbolt.DB // read only mode
}

// Persist implements raft.FSMSnapshot.
func (bs *boltSnapshot) Persist(sink raft.SnapshotSink) error {
	err := bs.bolt.View(func(tx *bbolt.Tx) error {
		tx.WriteFlag = syscall.O_DIRECT
		if _, err := tx.WriteTo(sink); err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		sink.Cancel()
		return err
	}
	sink.Close()
	return nil
}

// Release implements raft.FSMSnapshot.
func (bs *boltSnapshot) Release() {
	// bs.bolt.Close()
}
