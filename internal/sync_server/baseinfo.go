package sync_server

import (
	"fmt"
	"github.com/dollarkillerx/go-mysql/mysql"
	"log"
	"strings"

	"github.com/dollarkillerx/galaxy/internal/storage"
	"github.com/dollarkillerx/galaxy/pkg"
	"github.com/pingcap/errors"
)

// queryTableSchema 获取表schema
func (s *Sync) queryTableSchema(db string, table string) ([]pkg.MySQLSchema, error) {
	sql := fmt.Sprintf("show full columns from `%s`.`%s`", db, table)
	var schemas []pkg.MySQLSchema

	rows, err := s.db.Query(sql)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	defer rows.Close()

	for rows.Next() {
		var schema pkg.MySQLSchema
		err := rows.Scan(&schema.Field, &schema.Type, &schema.Collation, &schema.Null, &schema.Key, &schema.Default, &schema.Extra, &schema.Privileges, &schema.Comment)
		if err != nil {
			return nil, errors.WithStack(err)
		}

		schemas = append(schemas, schema)
	}

	return schemas, nil
}

// tableSchema 获取历史schema 如果不存在 则设置schema
func (s *Sync) tableSchema(db string, table string) (*pkg.HistorySchemas, error) {
	// 如果不存在 则 获取当前schema 反之, 获取

	hs, err := storage.Storage.GetSchemasByTable(db, table)
	if err != nil {
		// storage
		schema, err := s.initSchema(db, table)
		if err != nil {
			return nil, errors.WithStack(err)
		}

		err = storage.Storage.UpdateSchema(db, table, *schema)
		if err != nil {
			log.Printf("%+v\n", errors.WithStack(err))
		}
		return schema, nil
	}

	return hs, nil
}

// initSchema 初始化 schema
func (s *Sync) initSchema(db string, table string) (*pkg.HistorySchemas, error) {
	schema, err := s.queryTableSchema(db, table)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	hs := pkg.HistorySchemas{
		Db:    db,
		Table: table,
		Deltas: pkg.Deltas{
			Def: pkg.DeltasItem{
				Database: db,
				Table:    table,
			},
		},
	}

	col := make([]pkg.Columns, 0)
	for _, v := range schema {
		ci := pkg.Columns{
			Type: v.Type,
			Name: v.Field,
		}
		switch v.Null {
		case "Yes":
			ci.NotNull = false
		case "No":
			ci.NotNull = true
		}

		col = append(col, ci)
	}

	hs.Deltas.Def.Columns = col

	return &hs, nil
}

// updateSchema 当表结构发生变化时更新表
func (s *Sync) updateSchema(schema string, query string) (err error) {
	fmt.Println(query)
	if query == "BEGIN" {
		return nil
	}
	bakQuery := strings.ToLower(query)
	alterTable := "alter table"
	index := strings.Index(bakQuery, alterTable)
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
	action := strings.ToLower(qKv[3])

	if strings.Index(table, ".") != -1 {
		split := strings.Split(table, ".")
		schema = split[0]
		table = split[1]
	}

	if action == "modify" {
		return nil
	}

	byTable, err := storage.Storage.GetSchemasByTable(schema, table)
	if err != nil {
		byTable, err = s.tableSchema(schema, table)
		if err != nil {
			log.Println(err, "  ", schema, "  ", table)
			return err
		}
	}
	if byTable == nil {
		return nil
	}

	switch action {
	case "drop":
		delCol := ""
		if strings.ToLower(qKv[4]) == "column" {
			delCol = qKv[5]
		} else {
			delCol = qKv[4]
		}

		var newCol []pkg.Columns
		for _, v := range byTable.Deltas.Def.Columns {
			if delCol == v.Name {
				continue
			}

			newCol = append(newCol, v)
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
		delCol := ""
		if strings.ToLower(qKv[4]) == "column" {
			delCol = qKv[5]
		} else {
			delCol = qKv[4]
		}

		// 如果存在 则 不再添加
		for _, v := range byTable.Deltas.Def.Columns {
			if v.Name == delCol {
				return nil
			}
		}

		tvEnd := qKv[len(qKv)-1]

		var newCol []pkg.Columns
		if strings.ToLower(tvEnd) == "first" { // 如果在最前面
			newCol = append(newCol, pkg.Columns{
				Name: delCol,
			})
			newCol = append(newCol, byTable.Deltas.Def.Columns...)
		} else if strings.ToLower(qKv[len(qKv)-2]) == "after" { // 放在什么什么的后面
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
	case "change":
		oldName := qKv[4]
		newName := qKv[5]

		var newCol []pkg.Columns
		for _, v := range byTable.Deltas.Def.Columns {
			if v.Name == oldName {
				v.Name = newName
			}
			newCol = append(newCol, v)
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

// GetMasterPos 获取 Master Pos信息
func (s *Sync) GetMasterPos() (mysql.Position, error) {
	var status pkg.MySQLStatus
	err := s.db.QueryRow("SHOW MASTER STATUS").Scan(&status.File, &status.Position, &status.Binlog_Do_DB, &status.Binlog_lgnore_DB, &status.Executed_Gtid_Set)
	if err != nil {
		return mysql.Position{}, errors.Trace(err)
	}

	return mysql.Position{Name: status.File, Pos: status.Position}, nil
}
