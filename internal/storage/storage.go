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

func (s *storage) GetDB() *badger.DB {
	return s.db
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

func (s *storage) Del(key string) error {
	return s.db.Update(func(txn *badger.Txn) error {
		return txn.Delete([]byte(key))
	})
}

func (s *storage) Prefix(prefix string) (map[string][]byte, error) {
	result := make(map[string][]byte)

	err := s.db.View(func(txn *badger.Txn) error {
		it := txn.NewIterator(badger.DefaultIteratorOptions)
		defer it.Close()
		for it.Seek([]byte(prefix)); it.ValidForPrefix([]byte(prefix)); it.Next() {
			item := it.Item()
			k := item.Key()
			item.Value(func(val []byte) error {
				result[string(k)] = val
				return nil
			})
		}

		return nil
	})

	return result, err
}

//var once sync_server.Once
//func (s *storage) Test() {
//	s.Del(getSchemaID("test", "casbin_rule"))
//	//s.GetSchemasByTable("test", "casbin_rule")
//	log.Println("init tes success")
//}

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

	//marshal, err := json.Marshal(historySchemas)
	//if err == nil {
	//	fmt.Println("GetSchemasByTable: ", db, "   table: ", table, " ", string(marshal))
	//	fmt.Println()
	//}

	return &historySchemas, nil
}

// UpdateSchema 获取 更新 UpdateSchema
func (s *storage) UpdateSchema(db string, table string, schema pkg.HistorySchemas) error {
	id := getSchemaID(db, table)
	marshal, err := json.Marshal(schema)
	if err != nil {
		return errors.WithStack(err)
	}

	//fmt.Println("UpdateSchema: ", db, "   table: ", table, " ", string(marshal))
	//fmt.Println()

	return s.SetNX(id, marshal, 0)
}

func getSchemaID(db string, table string) string {
	return fmt.Sprintf("scheam.%s.%s", db, table)
}

func getTaskID(taskID string) string {
	return fmt.Sprintf("galaxy_task_%s", taskID)
}

// UpdateTask 持久化TASK
func (s *storage) UpdateTask(taskID string, task pkg.SharedSync) error {
	marshal, err := json.Marshal(task)
	if err != nil {
		return errors.WithStack(err)
	}

	return s.SetNX(getTaskID(taskID), marshal, 0)
}

// GetTasks 获取所有TASK
func (s *storage) GetTasks() ([]*pkg.SharedSync, error) {
	prefix, err := s.Prefix("galaxy_task_")
	if err != nil {
		return nil, errors.WithStack(err)
	}

	var result []*pkg.SharedSync

	for _, v := range prefix {
		var item pkg.SharedSync
		err := json.Unmarshal(v, &item)
		if err != nil {
			return nil, errors.WithStack(err)
		}

		result = append(result, &item)
	}

	return result, nil
}

// DelTask 删除持久化TASK
func (s *storage) DelTask(key string) error {
	return s.Del(getTaskID(key))
}

//func (s *storage) GetConcurrentlyTaskManagerID(taskID string) string {
//	return fmt.Sprintf("concurrently.task.manager.%s", taskID)
//}
//
//func (s *storage) UpdateConcurrentlyTaskManager(taskID string, data []byte) error {
//	return s.SetNX(s.GetConcurrentlyTaskManagerID(taskID), data, 0)
//}
