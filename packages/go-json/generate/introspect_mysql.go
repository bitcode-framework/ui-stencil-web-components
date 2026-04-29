package generate

import (
	"database/sql"
)

type MySQLIntrospector struct{}

func (m *MySQLIntrospector) ListTables(db *sql.DB) ([]string, error) {
	rows, err := db.Query("SHOW TABLES")
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

func (m *MySQLIntrospector) IntrospectTable(db *sql.DB, tableName string) (*TableInfo, error) {
	info := &TableInfo{Name: tableName}

	rows, err := db.Query(`
		SELECT COLUMN_NAME, DATA_TYPE, IS_NULLABLE, COLUMN_DEFAULT, CHARACTER_MAXIMUM_LENGTH,
		       EXTRA, COLUMN_KEY
		FROM information_schema.COLUMNS
		WHERE TABLE_NAME = ? AND TABLE_SCHEMA = DATABASE()
		ORDER BY ORDINAL_POSITION`, tableName)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var name, dataType, isNullable string
		var colDefault sql.NullString
		var maxLength sql.NullInt64
		var extra, columnKey string
		if err := rows.Scan(&name, &dataType, &isNullable, &colDefault, &maxLength, &extra, &columnKey); err != nil {
			return nil, err
		}

		col := ColumnInfo{
			Name:        name,
			DBType:      dataType,
			GoJSONType:  MapDBTypeToGoJSON(dataType),
			OpenAPIType: MapDBTypeToOpenAPI(dataType),
			Nullable:    isNullable == "YES",
			HasDefault:  colDefault.Valid,
			IsAutoIncr:  extra == "auto_increment",
		}
		if colDefault.Valid {
			col.DefaultValue = colDefault.String
		}
		if maxLength.Valid {
			col.MaxLength = int(maxLength.Int64)
		}
		if columnKey == "PRI" {
			info.PrimaryKey = append(info.PrimaryKey, name)
		}
		info.Columns = append(info.Columns, col)
	}

	fkRows, err := db.Query(`
		SELECT COLUMN_NAME, REFERENCED_TABLE_NAME, REFERENCED_COLUMN_NAME
		FROM information_schema.KEY_COLUMN_USAGE
		WHERE TABLE_NAME = ? AND TABLE_SCHEMA = DATABASE() AND REFERENCED_TABLE_NAME IS NOT NULL`, tableName)
	if err == nil {
		defer fkRows.Close()
		for fkRows.Next() {
			var col, refTable, refCol string
			if err := fkRows.Scan(&col, &refTable, &refCol); err == nil {
				info.ForeignKeys = append(info.ForeignKeys, ForeignKey{
					Columns:    []string{col},
					RefTable:   refTable,
					RefColumns: []string{refCol},
				})
			}
		}
	}

	return info, nil
}
