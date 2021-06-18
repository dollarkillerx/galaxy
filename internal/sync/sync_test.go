package sync

import (
	"context"
	"github.com/dollarkillerx/galaxy/pkg"
	"time"

	"fmt"
	"log"
	"testing"
)

func TestSync(t *testing.T) {
	sync, err := New(&pkg.TaskBaseData{
		MySqlConfig: pkg.MySQLConfig{
			User:     "root",
			Password: "root",
			Host:     "192.168.88.11",
			Port:     3306,
		},
	})

	if err != nil {
		log.Fatalln(err)
	}

	pos, err := sync.GetMasterPos()
	if err != nil {
		log.Fatalln(err)
	}
	fmt.Println(pos)

	schema, err := sync.tableSchema("news", "gorm_client_store_items")
	if err != nil {
		log.Fatalln(err)
	}

	fmt.Println(schema)
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
