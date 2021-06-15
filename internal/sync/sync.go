package sync

import (
	"github.com/dollarkillerx/galaxy/pkg"
	"github.com/go-mysql-org/go-mysql/canal"
	"github.com/go-mysql-org/go-mysql/mysql"
	"github.com/go-mysql-org/go-mysql/replication"
	_ "github.com/go-sql-driver/mysql"
	"github.com/pingcap/errors"

	"context"
	"database/sql"
	"fmt"
	"math/rand"
	"os"
	"time"
)

// TODO： 退出模块设计

// 同步模块
type Sync struct {
	BaseData     *pkg.TaskBaseData
	db           *sql.DB
	binlogSyncer *replication.BinlogSyncer
}

func New(BaseData *pkg.TaskBaseData) (*Sync, error) {
	if BaseData.ServerID == 0 {
		rand.Seed(time.Now().UnixNano())
		BaseData.ServerID = rand.Uint32()
	}
	sync := &Sync{BaseData: BaseData}

	return sync, sync.connMysql()
}

func (s *Sync) Monitor() error {
	cfg := replication.BinlogSyncerConfig{
		ServerID:   s.BaseData.ServerID,
		Flavor:     "mysql",
		Host:       s.BaseData.MySqlConfig.Host,
		Port:       s.BaseData.MySqlConfig.Port,
		User:       s.BaseData.MySqlConfig.User,
		Password:   s.BaseData.MySqlConfig.Password,
		UseDecimal: true,
	}

	s.binlogSyncer = replication.NewBinlogSyncer(cfg)

	var pos mysql.Position
	var err error
	// 使用最新值
	if s.BaseData.StartTime == 1 {
		s.BaseData.PositionPos = 1
	}
	if s.BaseData.PositionPos != 0 {
		pos, err = s.tryPosition(s.BaseData.PositionName, s.BaseData.PositionPos)
		if err != nil {
			return err
		}
	} else if s.BaseData.PositionPos == 1 { // 使用最新的
		pos, err = s.GetMasterPos()
		if err != nil {
			return err
		}
	} else { // 从0开始
		pos = mysql.Position{}
	}

	s.BaseData.PositionName = pos.Name
	s.BaseData.PositionPos = pos.Pos
	sync, err := s.binlogSyncer.StartSync(pos)
	if err != nil {
		return errors.WithStack(err)
	}

	for {
		event, err := sync.GetEvent(context.Background())
		if err != nil {
			// Try to output all left events
			events := sync.DumpEvents()
			for _, e := range events {
				e.Dump(os.Stdout)
			}
			fmt.Printf("Get event error: %v\n", errors.ErrorStack(err))
			return err
		}

		if event.Header != nil {
			var action string
			switch event.Header.EventType {
			case replication.WRITE_ROWS_EVENTv1, replication.WRITE_ROWS_EVENTv2:
				action = canal.InsertAction
			case replication.DELETE_ROWS_EVENTv1, replication.DELETE_ROWS_EVENTv2:
				action = canal.DeleteAction
			case replication.UPDATE_ROWS_EVENTv1, replication.UPDATE_ROWS_EVENTv2:
				action = canal.UpdateAction
			}

			if event.Event != nil {
				if s.BaseData.StartTime != 0 {
					if event.Header.Timestamp < s.BaseData.StartTime {
						continue
					}
				}

				ex1, ok := event.Event.(*replication.RowsEvent)
				if ok {
					fmt.Printf("LogPos: %d time: %d table: %s action: %s  TableID: %d  \n", event.Header.LogPos, event.Header.Timestamp, ex1.Table.Table, action, ex1.Table.TableID)
				}

				ex2, ok := event.Event.(*replication.TableMapEvent)
				if ok {
					fmt.Printf("LogPos: %d Schema: %s TableID: %d \n", event.Header.LogPos, ex2.Schema, ex2.TableID)
				}

				ex3, ok := event.Event.(*replication.QueryEvent)
				if ok {
					// TODO: 添加对模型更新
					// ALTER TABLE oauth.gorm_client_store_items MODIFY
					fmt.Printf("Query: %s\n", ex3.Query)
				}
			}
		}
	}

	return nil
}

// TODO: 完善尝试逻辑
func (s *Sync) tryPosition(file string, pos uint32) (mysql.Position, error) {
	// 尝试 链接
	ps := mysql.Position{Name: file, Pos: pos}
	sync, err := s.binlogSyncer.StartSync(ps)
	if err != nil {
		return mysql.Position{}, errors.WithStack(err)
	}

	_, err = sync.GetEvent(context.Background())

	return mysql.Position{}, err
}

func (s *Sync) GetMasterPos() (mysql.Position, error) {
	var status pkg.MySQLStatus
	err := s.db.QueryRow("SHOW MASTER STATUS").Scan(&status.File, &status.Position, &status.Binlog_Do_DB, &status.Binlog_lgnore_DB, &status.Executed_Gtid_Set)
	if err != nil {
		return mysql.Position{}, errors.Trace(err)
	}

	return mysql.Position{Name: status.File, Pos: status.Position}, nil
}

func (s *Sync) connMysql() error {
	db, err := sql.Open("mysql", fmt.Sprintf("%s:%s@(%s:%d)/mysql", s.BaseData.MySqlConfig.User, s.BaseData.MySqlConfig.Password, s.BaseData.MySqlConfig.Host, s.BaseData.MySqlConfig.Port))
	if err != nil {
		return errors.WithStack(err)
	}
	// See "Important settings" section.
	db.SetConnMaxLifetime(time.Minute * 3)
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)
	if err := db.Ping(); err != nil {
		return errors.WithStack(err)
	}

	s.db = db
	return nil
}
