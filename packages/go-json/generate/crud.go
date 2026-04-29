package generate

import (
	"encoding/json"
	"fmt"
	"strings"
)

// CRUDOptions configures CRUD generation.
type CRUDOptions struct {
	TableName string
	Fields    string
	DSN       string
	Auth      bool
	Pattern   string
}

// GenerateCRUD generates a go-json server program with CRUD routes for a table.
func GenerateCRUD(info *TableInfo, opts CRUDOptions) (map[string]any, error) {
	tableName := info.Name
	singular := singularize(tableName)

	program := map[string]any{
		"name":    tableName + "-api",
		"go_json": "1",
		"server": map[string]any{
			"port": 3000,
		},
	}

	if opts.Auth {
		program["server"].(map[string]any)["jwt"] = map[string]any{
			"secret_env": "JWT_SECRET",
			"algorithm":  "HS256",
			"expiry":     "24h",
		}
		program["middleware"] = []string{"logger", "recover", "cors"}
	} else {
		program["middleware"] = []string{"logger", "recover"}
	}

	program["import"] = map[string]any{
		"db": "io:sql",
	}

	routes := []map[string]any{
		{"method": "GET", "path": "/" + tableName, "handler": "list" + capitalize(tableName)},
		{"method": "POST", "path": "/" + tableName, "handler": "create" + capitalize(singular)},
		{"method": "GET", "path": "/" + tableName + "/:id", "handler": "get" + capitalize(singular)},
		{"method": "PUT", "path": "/" + tableName + "/:id", "handler": "update" + capitalize(singular)},
		{"method": "DELETE", "path": "/" + tableName + "/:id", "handler": "delete" + capitalize(singular)},
	}

	if opts.Auth {
		for i := range routes {
			if routes[i]["method"] != "GET" {
				routes[i]["middleware"] = []string{"jwt"}
			}
		}
	}

	program["routes"] = routes

	functions := make(map[string]any)

	functions["list"+capitalize(tableName)] = buildListFunction(info)
	functions["create"+capitalize(singular)] = buildCreateFunction(info)
	functions["get"+capitalize(singular)] = buildGetFunction(info)
	functions["update"+capitalize(singular)] = buildUpdateFunction(info)
	functions["delete"+capitalize(singular)] = buildDeleteFunction(info)

	program["functions"] = functions

	return program, nil
}

// GenerateCRUDJSON generates CRUD as formatted JSON string.
func GenerateCRUDJSON(info *TableInfo, opts CRUDOptions) (string, error) {
	program, err := GenerateCRUD(info, opts)
	if err != nil {
		return "", err
	}
	data, err := json.MarshalIndent(program, "", "  ")
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func buildListFunction(info *TableInfo) map[string]any {
	return map[string]any{
		"steps": []map[string]any{
			{
				"let":  "page",
				"expr": "int(request.query.page ?? '1')",
			},
			{
				"let":  "limit",
				"expr": "int(request.query.limit ?? '20')",
			},
			{
				"let":  "offset",
				"expr": "(page - 1) * limit",
			},
			{
				"let":  "rows",
				"call": "db.query",
				"with": map[string]string{
					"query": fmt.Sprintf("SELECT * FROM %s LIMIT ? OFFSET ?", info.Name),
					"args":  "[limit, offset]",
				},
			},
			{
				"let":  "total",
				"call": "db.query",
				"with": map[string]string{
					"query": fmt.Sprintf("SELECT COUNT(*) as count FROM %s", info.Name),
				},
			},
			{
				"return": map[string]any{
					"status": 200,
					"body": map[string]string{
						"data":  "rows.rows",
						"total": "total.rows[0].count",
						"page":  "page",
					},
				},
			},
		},
	}
}

func buildCreateFunction(info *TableInfo) map[string]any {
	writableColumns := getWritableColumns(info)
	colNames := make([]string, len(writableColumns))
	placeholders := make([]string, len(writableColumns))
	argExprs := make([]string, len(writableColumns))

	for i, col := range writableColumns {
		colNames[i] = col.Name
		placeholders[i] = "?"
		argExprs[i] = "request.body." + col.Name
	}

	return map[string]any{
		"steps": []map[string]any{
			{
				"let":  "result",
				"call": "db.execute",
				"with": map[string]string{
					"query": fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s)",
						info.Name, strings.Join(colNames, ", "), strings.Join(placeholders, ", ")),
					"args": "[" + strings.Join(argExprs, ", ") + "]",
				},
			},
			{
				"return": map[string]any{
					"status": 201,
					"body": map[string]string{
						"id": "result.last_insert_id",
					},
				},
			},
		},
	}
}

func buildGetFunction(info *TableInfo) map[string]any {
	pk := "id"
	if len(info.PrimaryKey) > 0 {
		pk = info.PrimaryKey[0]
	}
	return map[string]any{
		"steps": []map[string]any{
			{
				"let":  "rows",
				"call": "db.query",
				"with": map[string]string{
					"query": fmt.Sprintf("SELECT * FROM %s WHERE %s = ?", info.Name, pk),
					"args":  "[request.params.id]",
				},
			},
			{
				"if":   "len(rows.rows) == 0",
				"then": []map[string]any{{"return": map[string]any{"status": 404, "body": map[string]string{"error": "'Not found'"}}}},
			},
			{
				"return": map[string]any{
					"status": 200,
					"body":   "rows.rows[0]",
				},
			},
		},
	}
}

func buildUpdateFunction(info *TableInfo) map[string]any {
	writableColumns := getWritableColumns(info)
	setClauses := make([]string, len(writableColumns))
	argExprs := make([]string, len(writableColumns))

	for i, col := range writableColumns {
		setClauses[i] = col.Name + " = ?"
		argExprs[i] = "request.body." + col.Name
	}
	argExprs = append(argExprs, "request.params.id")

	pk := "id"
	if len(info.PrimaryKey) > 0 {
		pk = info.PrimaryKey[0]
	}

	return map[string]any{
		"steps": []map[string]any{
			{
				"let":  "result",
				"call": "db.execute",
				"with": map[string]string{
					"query": fmt.Sprintf("UPDATE %s SET %s WHERE %s = ?",
						info.Name, strings.Join(setClauses, ", "), pk),
					"args": "[" + strings.Join(argExprs, ", ") + "]",
				},
			},
			{
				"return": map[string]any{
					"status": 200,
					"body": map[string]string{
						"affected": "result.rows_affected",
					},
				},
			},
		},
	}
}

func buildDeleteFunction(info *TableInfo) map[string]any {
	pk := "id"
	if len(info.PrimaryKey) > 0 {
		pk = info.PrimaryKey[0]
	}
	return map[string]any{
		"steps": []map[string]any{
			{
				"let":  "result",
				"call": "db.execute",
				"with": map[string]string{
					"query": fmt.Sprintf("DELETE FROM %s WHERE %s = ?", info.Name, pk),
					"args":  "[request.params.id]",
				},
			},
			{
				"return": map[string]any{
					"status": 200,
					"body": map[string]string{
						"affected": "result.rows_affected",
					},
				},
			},
		},
	}
}

func getWritableColumns(info *TableInfo) []ColumnInfo {
	var result []ColumnInfo
	pkSet := make(map[string]bool)
	for _, pk := range info.PrimaryKey {
		pkSet[pk] = true
	}
	for _, col := range info.Columns {
		if pkSet[col.Name] || col.IsAutoIncr || col.IsGenerated {
			continue
		}
		if col.Name == "created_at" || col.Name == "updated_at" {
			continue
		}
		result = append(result, col)
	}
	return result
}

func singularize(s string) string {
	if strings.HasSuffix(s, "ies") {
		return s[:len(s)-3] + "y"
	}
	if strings.HasSuffix(s, "ses") || strings.HasSuffix(s, "xes") {
		return s[:len(s)-2]
	}
	if strings.HasSuffix(s, "s") && !strings.HasSuffix(s, "ss") {
		return s[:len(s)-1]
	}
	return s
}

func capitalize(s string) string {
	if len(s) == 0 {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}
