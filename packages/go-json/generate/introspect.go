package generate

import (
	"database/sql"
	"fmt"
	"strings"
)

// TableInfo describes a database table's structure.
type TableInfo struct {
	Name        string
	Columns     []ColumnInfo
	PrimaryKey  []string
	ForeignKeys []ForeignKey
	Uniques     [][]string
	Comment     string
}

// ColumnInfo describes a single column in a table.
type ColumnInfo struct {
	Name         string
	DBType       string
	GoJSONType   string
	OpenAPIType  string
	Nullable     bool
	HasDefault   bool
	DefaultValue string
	MaxLength    int
	IsAutoIncr   bool
	IsGenerated  bool
	EnumValues   []string
	Comment      string
}

// ForeignKey describes a foreign key relationship.
type ForeignKey struct {
	Columns    []string
	RefTable   string
	RefColumns []string
}

// Introspector extracts table metadata from a database.
type Introspector interface {
	ListTables(db *sql.DB) ([]string, error)
	IntrospectTable(db *sql.DB, tableName string) (*TableInfo, error)
}

// IntrospectDB connects to a database and introspects the specified tables.
func IntrospectDB(dsn string, tables []string) ([]*TableInfo, error) {
	driver := detectDriver(dsn)
	db, err := sql.Open(driver, dsn)
	if err != nil {
		return nil, fmt.Errorf("connect: %w", err)
	}
	defer db.Close()

	introspector := getIntrospector(driver)
	if introspector == nil {
		return nil, fmt.Errorf("unsupported driver: %s", driver)
	}

	if len(tables) == 0 {
		tables, err = introspector.ListTables(db)
		if err != nil {
			return nil, fmt.Errorf("list tables: %w", err)
		}
	}

	var result []*TableInfo
	for _, name := range tables {
		info, err := introspector.IntrospectTable(db, name)
		if err != nil {
			return nil, fmt.Errorf("introspect %s: %w", name, err)
		}
		result = append(result, info)
	}

	return result, nil
}

func detectDriver(dsn string) string {
	lower := strings.ToLower(dsn)
	if strings.HasPrefix(lower, "postgres://") || strings.Contains(lower, "host=") {
		return "postgres"
	}
	if strings.Contains(lower, "@tcp(") || strings.HasPrefix(lower, "mysql://") {
		return "mysql"
	}
	return "sqlite3"
}

func getIntrospector(driver string) Introspector {
	switch driver {
	case "sqlite3", "sqlite":
		return &SQLiteIntrospector{}
	case "postgres":
		return &PostgresIntrospector{}
	case "mysql":
		return &MySQLIntrospector{}
	default:
		return nil
	}
}

// MapDBTypeToGoJSON converts a database type to a go-json type.
func MapDBTypeToGoJSON(dbType string) string {
	upper := strings.ToUpper(dbType)
	switch {
	case strings.Contains(upper, "INT"):
		return "int"
	case strings.Contains(upper, "FLOAT") || strings.Contains(upper, "DOUBLE") || strings.Contains(upper, "DECIMAL") || strings.Contains(upper, "NUMERIC") || strings.Contains(upper, "REAL"):
		return "float"
	case strings.Contains(upper, "BOOL"):
		return "bool"
	case strings.Contains(upper, "DATE") || strings.Contains(upper, "TIME"):
		return "string"
	case strings.Contains(upper, "JSON"):
		return "any"
	default:
		return "string"
	}
}

// MapDBTypeToOpenAPI converts a database type to an OpenAPI type.
func MapDBTypeToOpenAPI(dbType string) string {
	upper := strings.ToUpper(dbType)
	switch {
	case strings.Contains(upper, "INT"):
		return "integer"
	case strings.Contains(upper, "FLOAT") || strings.Contains(upper, "DOUBLE") || strings.Contains(upper, "DECIMAL") || strings.Contains(upper, "NUMERIC") || strings.Contains(upper, "REAL"):
		return "number"
	case strings.Contains(upper, "BOOL"):
		return "boolean"
	default:
		return "string"
	}
}

// ParseManualFields parses a "name:type,name:type" string into a TableInfo.
func ParseManualFields(tableName, fieldsSpec string) *TableInfo {
	info := &TableInfo{
		Name:       tableName,
		PrimaryKey: []string{"id"},
		Columns: []ColumnInfo{
			{Name: "id", DBType: "INTEGER", GoJSONType: "int", OpenAPIType: "integer", IsAutoIncr: true},
		},
	}

	for _, field := range strings.Split(fieldsSpec, ",") {
		parts := strings.SplitN(strings.TrimSpace(field), ":", 2)
		if len(parts) != 2 {
			continue
		}
		name := strings.TrimSpace(parts[0])
		typ := strings.TrimSpace(parts[1])

		col := ColumnInfo{
			Name:        name,
			GoJSONType:  typ,
			OpenAPIType: goJSONToOpenAPI(typ),
			DBType:      goJSONToDB(typ),
		}
		info.Columns = append(info.Columns, col)
	}

	info.Columns = append(info.Columns,
		ColumnInfo{Name: "created_at", DBType: "TIMESTAMP", GoJSONType: "string", OpenAPIType: "string"},
		ColumnInfo{Name: "updated_at", DBType: "TIMESTAMP", GoJSONType: "string", OpenAPIType: "string"},
	)

	return info
}

func goJSONToOpenAPI(t string) string {
	switch t {
	case "int":
		return "integer"
	case "float":
		return "number"
	case "bool":
		return "boolean"
	default:
		return "string"
	}
}

func goJSONToDB(t string) string {
	switch t {
	case "int":
		return "INTEGER"
	case "float":
		return "REAL"
	case "bool":
		return "BOOLEAN"
	default:
		return "TEXT"
	}
}
