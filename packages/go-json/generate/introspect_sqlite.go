package generate

import (
	"database/sql"
	"fmt"
	"strings"
)

type SQLiteIntrospector struct{}

func (s *SQLiteIntrospector) ListTables(db *sql.DB) ([]string, error) {
	rows, err := db.Query("SELECT name FROM sqlite_master WHERE type='table' AND name NOT LIKE 'sqlite_%' ORDER BY name")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tables []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, err
		}
		tables = append(tables, name)
	}
	return tables, nil
}

func (s *SQLiteIntrospector) IntrospectTable(db *sql.DB, tableName string) (*TableInfo, error) {
	info := &TableInfo{Name: tableName}

	rows, err := db.Query(fmt.Sprintf("PRAGMA table_info(%s)", tableName))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var cid int
		var name, colType string
		var notNull, pk int
		var dflt sql.NullString
		if err := rows.Scan(&cid, &name, &colType, &notNull, &dflt, &pk); err != nil {
			return nil, err
		}

		col := ColumnInfo{
			Name:        name,
			DBType:      colType,
			GoJSONType:  MapDBTypeToGoJSON(colType),
			OpenAPIType: MapDBTypeToOpenAPI(colType),
			Nullable:    notNull == 0,
			HasDefault:  dflt.Valid,
		}
		if dflt.Valid {
			col.DefaultValue = dflt.String
		}
		if pk > 0 {
			info.PrimaryKey = append(info.PrimaryKey, name)
			if strings.ToUpper(colType) == "INTEGER" {
				col.IsAutoIncr = true
			}
		}
		info.Columns = append(info.Columns, col)
	}

	fkRows, err := db.Query(fmt.Sprintf("PRAGMA foreign_key_list(%s)", tableName))
	if err == nil {
		defer fkRows.Close()
		for fkRows.Next() {
			var id, seq int
			var table, from, to, onUpdate, onDelete, match string
			if err := fkRows.Scan(&id, &seq, &table, &from, &to, &onUpdate, &onDelete, &match); err != nil {
				continue
			}
			info.ForeignKeys = append(info.ForeignKeys, ForeignKey{
				Columns:    []string{from},
				RefTable:   table,
				RefColumns: []string{to},
			})
		}
	}

	return info, nil
}
