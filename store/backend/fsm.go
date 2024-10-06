package backend

//this package exist implementation of key-value db and raft state machine (FSM) backed by badger.

import (
	"errors"
	"fmt"
	"io"
	"log"
	"sync"

	"github.com/dgraph-io/badger/v4"
	"github.com/hashicorp/raft"
	storeproto "github.com/odit-bit/scarlett/store/proto"
	pb "google.golang.org/protobuf/proto"
)

var ErrItemExpired = errors.New("item is expired")

var _ raft.FSM = (*badgerStore)(nil)

type badgerStore struct {
	path string
	db   *badger.DB
	mx   sync.Mutex
}

func NewBadgerStore(path string) (*badgerStore, error) {
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

// Apply implements raft.FSM.
func (x *badgerStore) Apply(l *raft.Log) interface{} {
	cmd := storeproto.CmdRequestPayload{}
	if err := pb.Unmarshal(l.Data, &cmd); err != nil {
		return storeproto.CmdResponse{
			Msg: "",
			Err: err.Error(),
		}
	}

	res := &storeproto.CmdResponse{}
	var oErr error
	switch cmd.Cmd {
	case storeproto.Command_Type_Set:
		oErr = x.Set(cmd.Key, cmd.Value, 0)

	case storeproto.Command_Type_Delete:
		oErr = x.Delete(cmd.Key)

	default:
		res = &storeproto.CmdResponse{Err: fmt.Sprintf("unknown command %v", cmd.Cmd)}
	}
	if oErr != nil {
		res.Err = oErr.Error()
	}
	res.Msg = "success"
	return res
}

// Restore implements raft.FSM.
func (x *badgerStore) Restore(snapshot io.ReadCloser) error {
	log.Println("RESTORE")
	if err := x.Load(snapshot); err != nil {
		return err
	}
	return snapshot.Close()
}

// Snapshot implements raft.FSM.
func (x *badgerStore) Snapshot() (raft.FSMSnapshot, error) {
	// opts := badger.DefaultOptions(x.path)
	// opts.ReadOnly = true
	// db, err := badger.Open(opts)
	// if err != nil {
	// 	return nil, err
	// }
	return &xfsm{
		snape: x,
	}, nil
}
