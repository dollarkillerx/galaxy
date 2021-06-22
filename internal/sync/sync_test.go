package sync

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"testing"
	"time"

	"github.com/dollarkillerx/galaxy/internal/storage"
	"github.com/dollarkillerx/galaxy/pkg"
	"github.com/pingcap/errors"
)

func TestSync(t *testing.T) {
	log.SetFlags(log.Llongfile | log.LstdFlags)

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

	time.Sleep(time.Second * 3000)

	// ALTER TABLE test.casbin_rule CHANGE `number` num float(20,6) NULL
	// ALTER TABLE test.casbin_rule MODIFY COLUMN num decimal(20,6) NULL+
	// ALTER TABLE 表名 DROP [COLUMN] 字段名 ;
}

func TestSync2(t *testing.T) {
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

	//err = sync.Monitor()
	//if err != nil {
	//	log.Fatalln(err)
	//}
	schema, err := sync.queryTableSchema("test", "china_saic_registration_records")
	if err == nil {
		marshal, err := json.Marshal(schema)
		if err != nil {
			log.Fatalln(err)
		}
		fmt.Println(string(marshal))
	}

	time.Sleep(time.Second * 3000)

	// ALTER TABLE test.casbin_rule CHANGE `number` num float(20,6) NULL
	// ALTER TABLE test.casbin_rule MODIFY COLUMN num decimal(20,6) NULL+
	// ALTER TABLE 表名 DROP [COLUMN] 字段名 ;
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

func TestSchema(t *testing.T) {
	sql := []string{
		"/* applicationname=dbeaver 21.1.0 - sqleditor <script-50.sql> */alter table test.casbin_rule change a22g22e2x vx int(64) default 20 null",
		//"alter table test.casbin_rule modify column a22g22e2x int(64) default 20 null",
		//"alter table test.casbin_rule drop column v1",
		//"/* applicationname=dbeaver 21.1.0 - main */ alter table test.casbin_rule add xxs varchar(100) null",
		//"alter table test.casbin_rule add age int(4) default 20 after v0",
		//"alter table test_table add test int (5) default 4  first",
		//"alter table casbin_rule add age22x int(4) default 20 after v0",
	}

	for _, v := range sql {
		err := updateSchema("test", v)
		if err != nil {
			log.Fatalln(err)
		}
	}
}

// 去除注释 /* /*

// ALTER TABLE test.casbin_rule CHANGE v2_v2 v2 varchar(100) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NULL
// ALTER TABLE test.casbin_rule MODIFY COLUMN v2 varchar(300) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NULL
// ALTER TABLE test.casbin_rule DROP COLUMN num
// ALTER TABLE test.casbin_rule ADD ps varchar(100) NULL

// 添加到某列后面
// alter table tset_table add age int(4) default 20 after id;
// 将age添加到表test_table 中id的后面 其中default 为默认值
// 如果想将某列添加为第一列
// alter table test_table add test int (5) default 4  first

func updateSchema(schema string, query string) (err error) {
	if query == "BEGIN" {
		return nil
	}

	alterTable := "alter table"
	index := strings.Index(query, alterTable)
	if index == -1 {
		return nil
	}

	defer func() {
		if er := recover(); er != nil {
			err = errors.New(fmt.Sprintf("%v", err))
		}
	}()

	query = strings.TrimSpace(query[index:])
	qKv := strings.Split(query, " ")

	table := qKv[2]
	action := qKv[3]

	if strings.Index(table, ".") != -1 {
		split := strings.Split(table, ".")
		table = split[1]
	}

	switch action {
	case "modify":
		return nil
	case "drop":
		byTable, err := storage.Storage.GetSchemasByTable(schema, table)
		if err != nil {
			return err
		}
		delCol := ""
		if qKv[4] == "column" {
			delCol = qKv[5]
		} else {
			delCol = qKv[4]
		}

		var newCol []pkg.Columns
		for i, v := range byTable.Deltas.Def.Columns {
			// TODO: TEST
			if delCol == v.Name {
				if i == len(byTable.Deltas.Def.Columns)-1 {
					newCol = append(newCol[:i])
				} else {
					newCol = append(newCol[:i], newCol[i+1:]...)
				}
			}
		}

		old := byTable.Deltas.Def
		def := byTable.Deltas.Def
		def.Columns = newCol
		byTable.Deltas.Def = def
		byTable.Deltas.Old = old

		err = storage.Storage.UpdateSchema(schema, table, *byTable)
		if err != nil {
			return err
		}
	case "add":
		byTable, err := storage.Storage.GetSchemasByTable(schema, table)
		if err != nil {
			return err
		}
		delCol := ""
		if qKv[4] == "column" {
			delCol = qKv[5]
		} else {
			delCol = qKv[4]
		}

		tvEnd := qKv[len(qKv)-1]

		var newCol []pkg.Columns
		if tvEnd == "first" { // 如果在最前面
			newCol = append(newCol, pkg.Columns{
				Name: delCol,
			})
			newCol = append(newCol, byTable.Deltas.Def.Columns...)
		} else if qKv[len(qKv)-2] == "after" { // 放在什么什么的后面
			for _, v := range byTable.Deltas.Def.Columns {
				newCol = append(newCol, v)
				if v.Name == tvEnd {
					newCol = append(newCol, pkg.Columns{
						Name: delCol,
					})
				}
			}
		} else { // default 放到最后
			newCol = append(newCol, byTable.Deltas.Def.Columns...)
			newCol = append(newCol, pkg.Columns{
				Name: delCol,
			})
		}

		old := byTable.Deltas.Def
		def := byTable.Deltas.Def
		def.Columns = newCol
		byTable.Deltas.Def = def
		byTable.Deltas.Old = old
		err = storage.Storage.UpdateSchema(schema, table, *byTable)
		if err != nil {
			return err
		}
	}

	return nil
}

//func TestTreetMap(t *testing.T) {
//	comparator := treemap.NewWithIntComparator()
//	comparator.Put("a", "helloworld")
//	comparator.Put("a1", "helloworld1")
//	comparator.Put("a2", "helloworld2")
//	comparator.Put("a4", "helloworld3")
//	comparator.Put("a5", "helloworld4")
//	comparator.Put("a6", "helloworld5")
//
//	marshal, err := json.Marshal(comparator)
//	if err == nil {
//		fmt.Println(string(marshal))
//	}
//}
