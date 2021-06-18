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
	"log"
	"math/rand"
	"os"
	"time"
)

// TODO： 退出模块设计

// 同步模块
type Sync struct {
	sharedSync   *pkg.SharedSync
	db           *sql.DB
	binlogSyncer *replication.BinlogSyncer
}

func New(sharedSync *pkg.SharedSync) (*Sync, error) {
	if sharedSync.ServerID == 0 {
		rand.Seed(time.Now().UnixNano())
		sharedSync.ServerID = rand.Uint32()
	}
	sync := &Sync{sharedSync: sharedSync}

	return sync, sync.connMysql()
}

func (s *Sync) Monitor() error {
	cfg := replication.BinlogSyncerConfig{
		ServerID:   s.sharedSync.ServerID,
		Flavor:     "mysql",
		Host:       s.sharedSync.Task.MySqlConfig.Host,
		Port:       s.sharedSync.Task.MySqlConfig.Port,
		User:       s.sharedSync.Task.MySqlConfig.User,
		Password:   s.sharedSync.Task.MySqlConfig.Password,
		UseDecimal: true,
	}

	s.binlogSyncer = replication.NewBinlogSyncer(cfg)

	var pos mysql.Position
	var err error
	// 使用最新值
	if s.sharedSync.Task.StartTime == 0 {
		s.sharedSync.PositionPos = 1
	}
	if s.sharedSync.PositionPos == 1 { // 使用最新的
		pos, err = s.GetMasterPos()
		if err != nil {
			return err
		}
	} else if s.sharedSync.PositionPos != 0 { // 使用设定值
		pos, err = s.tryPosition(s.sharedSync.PositionName, s.sharedSync.PositionPos)
		if err != nil {
			return err
		}
	} else { // 重0开始
		pos = mysql.Position{}
	}
	s.sharedSync.PositionName = pos.Name
	s.sharedSync.PositionPos = pos.Pos
	sync, err := s.binlogSyncer.StartSync(pos)
	if err != nil {
		return errors.WithStack(err)
	}

	go func() {
	loop:
		for {
			select {
			case <-s.sharedSync.Context.Done():
				err := s.close()
				if err != nil {
					log.Println(err)
				}
				break loop
			default:
				err := s.syncMySQL(sync)
				if err != nil {
					log.Println(err)
					continue
				}
			}
		}
	}()

	return nil
}

func (s *Sync) syncMySQL(sync *replication.BinlogStreamer) error {
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
			if s.sharedSync.Task.StartTime != 0 {
				if event.Header.Timestamp < s.sharedSync.Task.StartTime {
					return nil
				}
			}

			ex1, ok := event.Event.(*replication.RowsEvent)
			if ok {
				fmt.Printf("LogPos: %d time: %d table: %s action: %s  TableID: %d  \n", event.Header.LogPos, event.Header.Timestamp, ex1.Table.Table, action, ex1.Table.TableID)
			}

			// 定义Schema 和 TableID 关系
			ex2, ok := event.Event.(*replication.TableMapEvent)
			if ok {
				fmt.Printf("LogPos: %d Schema: %s TableID: %d \n", event.Header.LogPos, ex2.Schema, ex2.TableID)
			}

			// 当模型schema更新时会调用当前
			ex3, ok := event.Event.(*replication.QueryEvent)
			if ok {
				// TODO: 添加对模型更新
				// ALTER TABLE oauth.gorm_client_store_items MODIFY
				fmt.Printf("Query: %s\n", ex3.Query)
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
	// master.000005, bin.000737
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
	db, err := sql.Open("mysql", fmt.Sprintf("%s:%s@(%s:%d)/mysql", s.sharedSync.Task.MySqlConfig.User, s.sharedSync.Task.MySqlConfig.Password, s.sharedSync.Task.MySqlConfig.Host, s.sharedSync.Task.MySqlConfig.Port))
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

func (s *Sync) close() error {
	err := s.db.Close()
	if err != nil {
		return err
	}

	s.binlogSyncer.Close()
	return nil
}
