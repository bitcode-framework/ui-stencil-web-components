package generate

import (
	"database/sql"
)

type PostgresIntrospector struct{}

func (p *PostgresIntrospector) ListTables(db *sql.DB) ([]string, error) {
	rows, err := db.Query("SELECT table_name FROM information_schema.tables WHERE table_schema = 'public' AND table_type = 'BASE TABLE' ORDER BY table_name")
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

func (p *PostgresIntrospector) IntrospectTable(db *sql.DB, tableName string) (*TableInfo, error) {
	info := &TableInfo{Name: tableName}

	rows, err := db.Query(`
		SELECT column_name, data_type, is_nullable, column_default, character_maximum_length,
		       is_identity, identity_generation
		FROM information_schema.columns
		WHERE table_name = $1 AND table_schema = 'public'
		ORDER BY ordinal_position`, tableName)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var name, dataType, isNullable string
		var colDefault, isIdentity sql.NullString
		var maxLength sql.NullInt64
		var identityGen sql.NullString
		if err := rows.Scan(&name, &dataType, &isNullable, &colDefault, &maxLength, &isIdentity, &identityGen); err != nil {
			return nil, err
		}

		col := ColumnInfo{
			Name:        name,
			DBType:      dataType,
			GoJSONType:  MapDBTypeToGoJSON(dataType),
			OpenAPIType: MapDBTypeToOpenAPI(dataType),
			Nullable:    isNullable == "YES",
			HasDefault:  colDefault.Valid,
		}
		if colDefault.Valid {
			col.DefaultValue = colDefault.String
		}
		if maxLength.Valid {
			col.MaxLength = int(maxLength.Int64)
		}
		if isIdentity.Valid && isIdentity.String == "YES" {
			col.IsAutoIncr = true
		}
		if identityGen.Valid {
			col.IsGenerated = true
		}
		info.Columns = append(info.Columns, col)
	}

	pkRows, err := db.Query(`
		SELECT a.attname
		FROM pg_index i
		JOIN pg_attribute a ON a.attrelid = i.indrelid AND a.attnum = ANY(i.indkey)
		WHERE i.indrelid = $1::regclass AND i.indisprimary`, tableName)
	if err == nil {
		defer pkRows.Close()
		for pkRows.Next() {
			var col string
			if err := pkRows.Scan(&col); err == nil {
				info.PrimaryKey = append(info.PrimaryKey, col)
			}
		}
	}

	fkRows, err := db.Query(`
		SELECT kcu.column_name, ccu.table_name, ccu.column_name
		FROM information_schema.table_constraints tc
		JOIN information_schema.key_column_usage kcu ON tc.constraint_name = kcu.constraint_name
		JOIN information_schema.constraint_column_usage ccu ON ccu.constraint_name = tc.constraint_name
		WHERE tc.constraint_type = 'FOREIGN KEY' AND tc.table_name = $1`, tableName)
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
