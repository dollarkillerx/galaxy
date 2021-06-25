package sync_server

import (
	"fmt"
	"log"

	"github.com/dollarkillerx/galaxy/pkg"
	"github.com/dollarkillerx/go-mysql/canal"
	"github.com/dollarkillerx/go-mysql/replication"
	"github.com/pingcap/errors"
)

func (s *Sync) RowsEventProcess(action string, event *replication.BinlogEvent, rowsEvent *replication.RowsEvent, posName string) error {
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
		err := s.rowEventUpdate(event, rowsEvent, tableSchema, &sendEvents, schema, table, action)
		if err != nil {
			return err
		}
	case canal.DeleteAction:
		err := s.rowEventDelete(event, rowsEvent, tableSchema, &sendEvents, schema, table, action)
		if err != nil {
			return err
		}
	case canal.InsertAction:
		err := s.rowEventInsert(event, rowsEvent, tableSchema, &sendEvents, schema, table, action)
		if err != nil {
			return err
		}
	}

	for _, v := range sendEvents {
		err := s.mq.SendMSG(v)
		if err != nil {
			log.Println(err)
		}
	}

	if event.Header.LogPos > s.sharedSync.PositionPos {
		s.sharedSync.PositionPos = event.Header.LogPos
	}
	//s.sharedSync.SaveShared <- s.sharedSync.Task.TaskID // 更新
	s.concurrentlyTaskManager.MissionComplete(posName, event.Header.LogPos) // 记录 并更新
	return nil
}

func (s *Sync) rowEventUpdate(event *replication.BinlogEvent, rowsEvent *replication.RowsEvent, tableSchema *pkg.HistorySchemas, sendEvents *[]pkg.MQEvent, schema, table, action string) error {
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

		*sendEvents = append(*sendEvents, sendEvent)
	}

	return nil
}

func (s *Sync) rowEventDelete(event *replication.BinlogEvent, rowsEvent *replication.RowsEvent, tableSchema *pkg.HistorySchemas, sendEvents *[]pkg.MQEvent, schema, table, action string) error {
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

		*sendEvents = append(*sendEvents, sendEvent)
	}

	return nil
}

func (s *Sync) rowEventInsert(event *replication.BinlogEvent, rowsEvent *replication.RowsEvent, tableSchema *pkg.HistorySchemas, sendEvents *[]pkg.MQEvent, schema, table, action string) error {
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

		*sendEvents = append(*sendEvents, sendEvent)
	}

	return nil
}

func (s *Sync) QueryEventProcess(event *replication.BinlogEvent, queryEvent *replication.QueryEvent) error {
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
			return err
		}

		s.sharedSync.PositionPos = event.Header.LogPos
		s.sharedSync.SaveShared <- s.sharedSync.Task.TaskID // 更新
	}
	return nil
}

func (s *Sync) RotateEventProcess(event *replication.BinlogEvent, rotateEvent *replication.RotateEvent) error {
	nm := string(rotateEvent.NextLogName)
	log.Println("RotateEvent: ", nm, " pos: ", rotateEvent.Position)

	if rotateEvent.Position != 0 {
		if nm != s.sharedSync.PositionName {
			s.sharedSync.PositionName = nm
			s.sharedSync.PositionPos = event.Header.LogPos
			s.sharedSync.PositionPos = uint32(rotateEvent.Position)
			s.sharedSync.SaveShared <- s.sharedSync.Task.TaskID // 发送更新信号
		}
	}

	if nm != s.sharedSync.PositionName {
		s.sharedSync.PositionName = nm
		s.sharedSync.PositionPos = event.Header.LogPos
		s.sharedSync.SaveShared <- s.sharedSync.Task.TaskID // 发送更新信号
	}
	return nil
}
