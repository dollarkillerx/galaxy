package storage

import (
	"github.com/dgraph-io/badger/v3"
	"github.com/pingcap/errors"

	"log"
	"time"
)

type storage struct {
	db *badger.DB
}

var Storage *storage

func init() {
	open, err := badger.Open(badger.DefaultOptions("./galaxy_data"))
	if err != nil {
		log.Fatalln(err)
	}

	Storage = &storage{db: open}
}

func (s *storage) SetNX(key string, value []byte, timeout time.Duration) error {
	return s.db.Update(func(txn *badger.Txn) error {
		ttl := badger.NewEntry([]byte(key), value)
		if timeout > 0 {
			ttl = ttl.WithTTL(timeout)
		}
		return txn.SetEntry(ttl)
	})
}

func (s *storage) Get(key string) (value []byte, err error) {
	return value, s.db.View(func(txn *badger.Txn) error {
		item, err := txn.Get([]byte(key))
		if err != nil {
			return err
		}

		return item.Value(func(val []byte) error {
			if val == nil {
				return errors.New("val is nil")
			}

			value = val
			return nil
		})
	})
}
