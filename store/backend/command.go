package backend

import (
	"io"
	"time"

	"github.com/dgraph-io/badger/v4"
)

////// kv operation

func (b *badgerStore) Get(key []byte) ([]byte, bool, error) {
	var result []byte
	var ok bool

	// todo:the raft-log not knowing if this object expired , make item exist again when node restart.
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

	// todo:the raft-log not knowing if this object expired , make it show up when node restart.
	// _ = expireAt
	// if expired > 0 {
	// 	expireAt = time.Now().Add(expired).Unix()
	// }
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
