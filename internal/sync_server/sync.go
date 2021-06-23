package sync_server

import (
	"github.com/dollarkillerx/galaxy/internal/mq_manager"
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
	"strings"
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
	s.binlogSyncer, err = replication.NewBinlogSyncer(cfg, s.sharedSync.Task.TaskID)
	if err != nil {
		return errors.WithStack(err)
	}

	var pos mysql.Position
	// 使用最新值
	if s.sharedSync.PositionPos == 0 { // 使用最新的
		pos, err = s.GetMasterPos()
		if err != nil {
			return err
		}
		fmt.Println("Latest Pos", "   taskID: ", s.sharedSync.Task.TaskID)
	} else if s.sharedSync.PositionPos != 0 { // 使用设定值
		pos, err = s.tryPosition(s.sharedSync.PositionName, s.sharedSync.PositionPos)
		if err != nil {
			return err
		}

		fmt.Println("Pos Recovery", pos.Name, pos.Pos, "   taskID: ", s.sharedSync.Task.TaskID)
	}

	s.sharedSync.PositionName = pos.Name
	s.sharedSync.PositionPos = pos.Pos

	log.Println("Start BinlogSyncer: ", pos, "   taskID: ", s.sharedSync.Task.TaskID)
	s.sync, err = s.binlogSyncer.StartSync(pos)
	if err != nil {
		return errors.WithStack(err)
	}

	mq, err := mq_manager.Manager.Get(s.sharedSync.Task.TaskID)
	if err != nil {
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
		if strings.Contains(err.Error(), "no corresponding table map event") {
			return nil
		}
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
				schema := string(rowsEvent.Table.Schema)
				table := string(rowsEvent.Table.Table)
				if rowsEvent.Table == nil {
					return nil
				}

				// 处理 table
				var database string
				var tables []string
				var excludeTables []string
				var tablesMap map[string]struct{}
				var excludeTablesMap map[string]struct{}
				{
					s.sharedSync.Rw.RLock()

					tables = s.sharedSync.Task.TaskBaseData.Tables
					tablesMap = s.sharedSync.Task.TaskBaseData.TablesMap
					excludeTables = s.sharedSync.Task.TaskBaseData.ExcludeTable
					excludeTablesMap = s.sharedSync.Task.TaskBaseData.ExcludeTableMap
					database = s.sharedSync.Task.TaskBaseData.Database

					s.sharedSync.Rw.RUnlock()
				}

				if database != schema {
					return nil
				}
				if len(tables) != 0 {
					_, ex := tablesMap[table]
					if !ex {
						return nil
					}
				}
				if len(excludeTables) != 0 {
					_, ex := excludeTablesMap[table]
					if ex {
						return nil
					}
				}

				//fmt.Printf("LogPos: %d time: %d table: %s action: %s  TableID: %d Schema: %s  \n", event.Header.LogPos, event.Header.Timestamp, rowsEvent.Table.Table, action, rowsEvent.Table.TableID, rowsEvent.Table.Schema)

				// 处理 table 结束
				tableSchema, err := s.tableSchema(schema, table)
				if err != nil {
					return errors.WithStack(err)
				}

				var sendEvents []pkg.MQEvent

				switch action {
				case canal.UpdateAction:
					if len(rowsEvent.Rows) < 2 || len(rowsEvent.Rows)%2 != 0 {
						return errors.New("UpdateAction rowsEvent.Rows < 2")
					}

					for i := 0; i < len(rowsEvent.Rows); i += 2 {
						if len(rowsEvent.Rows[i]) != len(tableSchema.Deltas.Def.Columns) ||
							len(rowsEvent.Rows[i+1]) != len(tableSchema.Deltas.Def.Columns) {
							return errors.New(fmt.Sprintf("UpdateAction rowsEvent.Rows[0]: %d  %v rowsEvent.Rows[1]: %d  %v  != tableSchema.Deltas.Def.Columns %v \n", len(rowsEvent.Rows[0]), rowsEvent.Rows[0], len(rowsEvent.Rows[1]), rowsEvent.Rows[1], tableSchema.Deltas.Def.Columns))
						}

						sendEvent := pkg.MQEvent{
							Database: schema,
							Table:    table,
							Action:   action,
							OrgRow:   [][]interface{}{rowsEvent.Rows[i], rowsEvent.Rows[i+1]},
							EventHeader: pkg.EventHeader{
								Timestamp: event.Header.Timestamp,
								LogPos:    event.Header.LogPos,
							},
						}

						after := make(map[string]interface{})
						before := make(map[string]interface{})
						for k, v := range tableSchema.Deltas.Def.Columns {
							after[v.Name] = rowsEvent.Rows[i][k]
							before[v.Name] = rowsEvent.Rows[i+1][k]
						}
						sendEvent.After = after
						sendEvent.Before = before

						sendEvents = append(sendEvents, sendEvent)
					}
				case canal.DeleteAction:
					if len(rowsEvent.Rows) < 1 {
						return errors.New("DeleteAction rowsEvent.Rows < 1")
					}

					for _, vv := range rowsEvent.Rows {
						sendEvent := pkg.MQEvent{
							Database: schema,
							Table:    table,
							Action:   action,
							OrgRow:   [][]interface{}{vv},
							EventHeader: pkg.EventHeader{
								Timestamp: event.Header.Timestamp,
								LogPos:    event.Header.LogPos,
							},
						}

						before := make(map[string]interface{})
						if len(vv) != len(tableSchema.Deltas.Def.Columns) {
							return errors.New("DeleteAction rowsEvent.Rows[0] != tableSchema.Deltas.Def.Columns")
						}

						for k, v := range tableSchema.Deltas.Def.Columns {
							before[v.Name] = vv[k]
						}
						sendEvent.Before = before

						sendEvents = append(sendEvents, sendEvent)
					}
				case canal.InsertAction:
					if len(rowsEvent.Rows) < 1 {
						return errors.New("InsertAction rowsEvent.Rows < 1")
					}

					for _, vv := range rowsEvent.Rows {
						if len(vv) != len(tableSchema.Deltas.Def.Columns) {
							return errors.New("InsertAction rowsEvent.Rows[0] != tableSchema.Deltas.Def.Columns")
						}
						sendEvent := pkg.MQEvent{
							Database: schema,
							Table:    table,
							Action:   action,
							OrgRow:   [][]interface{}{vv},
							EventHeader: pkg.EventHeader{
								Timestamp: event.Header.Timestamp,
								LogPos:    event.Header.LogPos,
							},
						}

						after := make(map[string]interface{})
						for k, v := range tableSchema.Deltas.Def.Columns {
							after[v.Name] = vv[k]
						}
						sendEvent.After = after

						sendEvents = append(sendEvents, sendEvent)
					}
				}

				for _, v := range sendEvents {
					err := s.mq.SendMSG(v)
					if err != nil {
						log.Println(err)
					}
				}

				s.sharedSync.PositionPos = event.Header.LogPos
				s.sharedSync.SaveShared <- s.sharedSync.Task.TaskID // 更新
			}

			// 修改模型schema 当模型schema更新时会调用当前
			queryEvent, ok := event.Event.(*replication.QueryEvent)
			if ok {
				if string(queryEvent.Query) == "BEGIN" {
					return nil
				}
				log.Println("QueryEvent: ", string(queryEvent.Query))
				// 添加对模型更新
				if queryEvent.ErrorCode == 0 {
					schema := string(queryEvent.Schema)
					if schema != "" {
						if schema != s.sharedSync.Task.Database {
							return nil
						}
					}
					err := s.updateSchema(schema, string(queryEvent.Query))
					if err != nil {
						log.Printf("%+v\n", err)
					}

					s.sharedSync.PositionPos = event.Header.LogPos
					s.sharedSync.SaveShared <- s.sharedSync.Task.TaskID // 更新
				}
			}

			// TODO: 处理offset
			offsetEvent, ok := event.Event.(*replication.RotateEvent)
			if ok {
				nm := string(offsetEvent.NextLogName)
				log.Println("RotateEvent: ", nm, " pos: ", offsetEvent.Position)

				if offsetEvent.Position != 0 {
					if nm != s.sharedSync.PositionName {
						s.sharedSync.PositionName = nm
						s.sharedSync.PositionPos = event.Header.LogPos
						s.sharedSync.PositionPos = uint32(offsetEvent.Position)
						s.sharedSync.SaveShared <- s.sharedSync.Task.TaskID // 发送更新信号
					}
				}

				if nm != s.sharedSync.PositionName {
					s.sharedSync.PositionName = nm
					s.sharedSync.PositionPos = event.Header.LogPos
					s.sharedSync.SaveShared <- s.sharedSync.Task.TaskID // 发送更新信号
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
	fmt.Println("开始关闭   binlogSyncer")
	s.binlogSyncer.Close()
	fmt.Println("开始关闭   binlogSyncer End")
	s.binlogSyncer, err = replication.NewBinlogSyncer(s.cfg, s.sharedSync.Task.TaskID)
	return ps, err
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

	return mq_manager.Manager.Close(s.sharedSync.Task.TaskID)
}
