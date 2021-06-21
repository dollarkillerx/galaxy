package storage

import (
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/dgraph-io/badger/v3"
	"github.com/dollarkillerx/galaxy/pkg"
	"github.com/pingcap/errors"
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

// GetSchemasByTable 获取 HistorySchemas
func (s *storage) GetSchemasByTable(db string, table string) (*pkg.HistorySchemas, error) {
	resp, err := s.Get(getSchemaID(db, table))
	if err != nil {
		return nil, errors.WithStack(err)
	}

	var historySchemas pkg.HistorySchemas
	err = json.Unmarshal(resp, &historySchemas)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	return &historySchemas, nil
}

// UpdateSchema 获取 更新 UpdateSchema
func (s *storage) UpdateSchema(db string, table string, schema pkg.HistorySchemas) error {
	id := getSchemaID(db, table)
	marshal, err := json.Marshal(schema)
	if err != nil {
		return errors.WithStack(err)
	}

	return s.SetNX(id, marshal, 0)
}

func getSchemaID(db string, table string) string {
	return fmt.Sprintf("scheam.%s.%s", db, table)
}
