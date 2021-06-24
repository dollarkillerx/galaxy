package sync_server

import (
	"github.com/dollarkillerx/galaxy/internal/mq_manager"
	"github.com/dollarkillerx/galaxy/internal/storage"
	"github.com/dollarkillerx/galaxy/pkg"
	"github.com/dollarkillerx/go-mysql/canal"
	"github.com/dollarkillerx/go-mysql/mysql"
	"github.com/dollarkillerx/go-mysql/replication"
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

// Sync 同步模块
type Sync struct {
	sharedSync   *pkg.SharedSync
	db           *sql.DB
	binlogSyncer *replication.BinlogSyncer
	mq           mq_manager.MQ

	sync *replication.BinlogStreamer
	cfg  replication.BinlogSyncerConfig
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

	s.cfg = cfg
	var err error
	s.binlogSyncer, err = replication.NewBinlogSyncer(cfg, storage.Storage.GetDB(), s.sharedSync.Task.TaskID)
	if err != nil {
		s.sharedSync.ErrorMsg = err.Error()
		return errors.WithStack(err)
	}

	var pos mysql.Position
	// 使用最新值
	if s.sharedSync.PositionPos == 0 { // 使用最新的
		pos, err = s.GetMasterPos()
		if err != nil {
			s.sharedSync.ErrorMsg = err.Error()
			return err
		}
		fmt.Println("Latest Pos", "   taskID: ", s.sharedSync.Task.TaskID)
	} else if s.sharedSync.PositionPos != 0 { // 使用设定值
		pos, err = s.tryPosition(s.sharedSync.PositionName, s.sharedSync.PositionPos)
		if err != nil {
			s.sharedSync.ErrorMsg = err.Error()
			return err
		}

		fmt.Println("Pos Recovery", pos.Name, pos.Pos, "   taskID: ", s.sharedSync.Task.TaskID)
	}

	s.sharedSync.PositionName = pos.Name
	s.sharedSync.PositionPos = pos.Pos

	log.Println("Start BinlogSyncer: ", pos, "   taskID: ", s.sharedSync.Task.TaskID)
	s.sync, err = s.binlogSyncer.StartSync(pos)
	if err != nil {
		s.sharedSync.ErrorMsg = err.Error()
		return errors.WithStack(err)
	}

	mq, err := mq_manager.Manager.Get(s.sharedSync.Task.TaskID)
	if err != nil {
		s.sharedSync.ErrorMsg = err.Error()
		return errors.WithStack(err)
	}
	s.mq = mq

	go func() {
	loop:
		for {
			select {
			case <-s.sharedSync.Context.Done():
				log.Println("Monitor Close: ", s.sharedSync.Task.TaskID)
				err := s.close()
				if err != nil {
					log.Println(err)
				}

				break loop
			default:
				// 现阶段数据处理采用单线程 多线程处理需要维护许多状态点 复杂度指数级提高 未来优化
				err := s.syncMySQL()
				if err != nil {
					log.Printf("id: %s err: %s \n", s.sharedSync.Task.TaskID, err.Error())
					//os.Exit(0)
					continue
				}
			}
		}
	}()

	log.Println("Monitor Init Success: ", s.sharedSync.Task.TaskID)
	return nil
}

func (s *Sync) syncMySQL() error {
	event, err := s.sync.GetEvent(context.Background())
	if err != nil {
		// Try to output all left events
		events := s.sync.DumpEvents()
		for _, e := range events {
			e.Dump(os.Stdout)
		}

		fmt.Printf("ID: %s Get event error: %s\n", s.sharedSync.Task.TaskID, errors.ErrorStack(err))
		return err
	}

	//event.Dump(os.Stdout)
	if event.Header != nil {
		if event.Event != nil {
			var action string
			switch event.Header.EventType {
			case replication.WRITE_ROWS_EVENTv1, replication.WRITE_ROWS_EVENTv2:
				action = canal.InsertAction
			case replication.DELETE_ROWS_EVENTv1, replication.DELETE_ROWS_EVENTv2:
				action = canal.DeleteAction
			case replication.UPDATE_ROWS_EVENTv1, replication.UPDATE_ROWS_EVENTv2:
				action = canal.UpdateAction
			}

			// insert del update 操作
			rowsEvent, ok := event.Event.(*replication.RowsEvent)
			if ok {
				err := s.RowsEventProcess(action, event, rowsEvent)
				if err != nil {
					return err
				}
			}

			// 修改模型schema 当模型schema更新时会调用当前
			queryEvent, ok := event.Event.(*replication.QueryEvent)
			if ok {
				err := s.QueryEventProcess(event, queryEvent)
				if err != nil {
					return err
				}
			}

			// TODO: 处理offset
			rotateEvent, ok := event.Event.(*replication.RotateEvent)
			if ok {
				err := s.RotateEventProcess(event, rotateEvent)
				if err != nil {
					return err
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
	// master.000005, bin.000737
	s.binlogSyncer.Close()
	s.binlogSyncer, err = replication.NewBinlogSyncer(s.cfg, storage.Storage.GetDB(), s.sharedSync.Task.TaskID)
	return ps, err
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

	return mq_manager.Manager.Close(s.sharedSync.Task.TaskID)
}
