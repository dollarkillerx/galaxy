package sync

import (
	"context"
	"github.com/dollarkillerx/galaxy/pkg"
	"log"
	"time"

	"fmt"
	"testing"
)

func TestSync(t *testing.T) {
	sync, err := New(&pkg.SharedSync{
		ServerID: 10120,
		Task: &pkg.Task{
			TaskBaseData: pkg.TaskBaseData{
				TaskID: "xxxx",
				MySqlConfig: pkg.MySQLConfig{
					User:     "root",
					Password: "root",
					Host:     "192.168.88.11",
					Port:     3307,
				},
			},
		},
		Context: context.Background(),
	})

	if err != nil {
		log.Fatalln(err)
	}

	err = sync.Monitor()
	if err != nil {
		log.Fatalln(err)
	}

	time.Sleep(time.Second * 100)

	//
	//if err != nil {
	//	log.Fatalln(err)
	//}
	//
	//pos, err := sync.GetMasterPos()
	//if err != nil {
	//	log.Fatalln(err)
	//}
	//fmt.Println(pos)
	//
	//schema, err := sync.tableSchema("news", "gorm_client_store_items")
	//if err != nil {
	//	log.Fatalln(err)
	//}
	//
	//fmt.Println(schema)
}

func TestPx(t *testing.T) {
	cancel, cancelFunc := context.WithCancel(context.Background())

	go func() {
		time.Sleep(time.Second * 3)
		cancelFunc()
	}()

loop:
	for {
		select {
		case <-cancel.Done():
			break loop
		default:
			fmt.Println("hello world")
		}
	}
}

//func TestM2(t *testing.T) {
//	cfg := replication.BinlogSyncerConfig{
//		ServerID:   1155,
//		Flavor:     "mysql",
//		Host:       "192.168.88.11",
//		Port:       3307,
//		User:       "root",
//		Password:   "root",
//		UseDecimal: true,
//	}
//
//}
