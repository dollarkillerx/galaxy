package sync

import (
	"fmt"

	"github.com/dollarkillerx/galaxy/pkg"
	"github.com/pingcap/errors"
)

// tableSchema 获取表schema
func (s *Sync) tableSchema(db string, table string) ([]pkg.MySQLSchema, error) {
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
	}

	return schemas, nil
}
