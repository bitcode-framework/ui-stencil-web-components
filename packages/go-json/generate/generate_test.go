package generate

import (
	"encoding/json"
	"strings"
	"testing"
)

// ---------------------------------------------------------------------------
// helpers
// ---------------------------------------------------------------------------

func makeTestTableInfo() *TableInfo {
	return &TableInfo{
		Name:       "products",
		PrimaryKey: []string{"id"},
		Columns: []ColumnInfo{
			{Name: "id", DBType: "INTEGER", GoJSONType: "int", OpenAPIType: "integer", IsAutoIncr: true},
			{Name: "name", DBType: "TEXT", GoJSONType: "string", OpenAPIType: "string"},
			{Name: "price", DBType: "REAL", GoJSONType: "float", OpenAPIType: "number"},
			{Name: "active", DBType: "BOOLEAN", GoJSONType: "bool", OpenAPIType: "boolean"},
			{Name: "created_at", DBType: "TIMESTAMP", GoJSONType: "string", OpenAPIType: "string"},
			{Name: "updated_at", DBType: "TIMESTAMP", GoJSONType: "string", OpenAPIType: "string"},
		},
	}
}

// ---------------------------------------------------------------------------
// ParseManualFields
// ---------------------------------------------------------------------------

func TestParseManualFields_Basic(t *testing.T) {
	info := ParseManualFields("products", "name:string,price:float")

	if info.Name != "products" {
		t.Errorf("expected table name 'products', got %q", info.Name)
	}
	if len(info.PrimaryKey) != 1 || info.PrimaryKey[0] != "id" {
		t.Errorf("expected primary key [id], got %v", info.PrimaryKey)
	}

	// id + name + price + created_at + updated_at = 5
	if len(info.Columns) != 5 {
		t.Fatalf("expected 5 columns, got %d: %v", len(info.Columns), columnNames(info.Columns))
	}

	// First column should be auto-generated id
	id := info.Columns[0]
	if id.Name != "id" || !id.IsAutoIncr || id.GoJSONType != "int" {
		t.Errorf("id column mismatch: %+v", id)
	}

	// User-defined columns
	nameCol := info.Columns[1]
	if nameCol.Name != "name" || nameCol.GoJSONType != "string" || nameCol.OpenAPIType != "string" || nameCol.DBType != "TEXT" {
		t.Errorf("name column mismatch: %+v", nameCol)
	}

	priceCol := info.Columns[2]
	if priceCol.Name != "price" || priceCol.GoJSONType != "float" || priceCol.OpenAPIType != "number" || priceCol.DBType != "REAL" {
		t.Errorf("price column mismatch: %+v", priceCol)
	}

	// Trailing columns
	if info.Columns[3].Name != "created_at" {
		t.Errorf("expected created_at at index 3, got %q", info.Columns[3].Name)
	}
	if info.Columns[4].Name != "updated_at" {
		t.Errorf("expected updated_at at index 4, got %q", info.Columns[4].Name)
	}
}

func TestParseManualFields_AllTypes(t *testing.T) {
	info := ParseManualFields("t", "a:int,b:float,c:bool,d:string")

	expected := []struct {
		name, goJSON, openAPI, db string
	}{
		{"a", "int", "integer", "INTEGER"},
		{"b", "float", "number", "REAL"},
		{"c", "bool", "boolean", "BOOLEAN"},
		{"d", "string", "string", "TEXT"},
	}

	// Skip id (index 0), check user columns (1..4)
	for i, exp := range expected {
		col := info.Columns[i+1]
		if col.Name != exp.name {
			t.Errorf("column %d: expected name %q, got %q", i, exp.name, col.Name)
		}
		if col.GoJSONType != exp.goJSON {
			t.Errorf("column %s: expected GoJSONType %q, got %q", exp.name, exp.goJSON, col.GoJSONType)
		}
		if col.OpenAPIType != exp.openAPI {
			t.Errorf("column %s: expected OpenAPIType %q, got %q", exp.name, exp.openAPI, col.OpenAPIType)
		}
		if col.DBType != exp.db {
			t.Errorf("column %s: expected DBType %q, got %q", exp.name, exp.db, col.DBType)
		}
	}
}

func TestParseManualFields_SkipsMalformed(t *testing.T) {
	info := ParseManualFields("t", "good:string,bad,also:int")
	// id + good + also + created_at + updated_at = 5 (bad is skipped)
	if len(info.Columns) != 5 {
		t.Errorf("expected 5 columns (skipping malformed), got %d: %v", len(info.Columns), columnNames(info.Columns))
	}
	if info.Columns[1].Name != "good" {
		t.Errorf("expected first user column 'good', got %q", info.Columns[1].Name)
	}
	if info.Columns[2].Name != "also" {
		t.Errorf("expected second user column 'also', got %q", info.Columns[2].Name)
	}
}

func TestParseManualFields_WhitespaceHandling(t *testing.T) {
	info := ParseManualFields("t", " name : string , age : int ")
	// id + name + age + created_at + updated_at = 5
	if len(info.Columns) != 5 {
		t.Fatalf("expected 5 columns, got %d", len(info.Columns))
	}
	if info.Columns[1].Name != "name" {
		t.Errorf("expected trimmed name 'name', got %q", info.Columns[1].Name)
	}
	if info.Columns[2].Name != "age" {
		t.Errorf("expected trimmed name 'age', got %q", info.Columns[2].Name)
	}
}

// ---------------------------------------------------------------------------
// GenerateCRUD
// ---------------------------------------------------------------------------

func TestGenerateCRUD_BasicStructure(t *testing.T) {
	info := makeTestTableInfo()
	opts := CRUDOptions{TableName: "products"}

	program, err := GenerateCRUD(info, opts)
	if err != nil {
		t.Fatalf("GenerateCRUD error: %v", err)
	}

	if program["name"] != "products-api" {
		t.Errorf("expected name 'products-api', got %v", program["name"])
	}
	if program["go_json"] != "1" {
		t.Errorf("expected go_json '1', got %v", program["go_json"])
	}

	server, ok := program["server"].(map[string]any)
	if !ok {
		t.Fatal("server is not map[string]any")
	}
	if server["port"] != 3000 {
		t.Errorf("expected port 3000, got %v", server["port"])
	}

	// Without auth, no JWT config
	if _, hasJWT := server["jwt"]; hasJWT {
		t.Error("expected no JWT config without auth")
	}
}

func TestGenerateCRUD_Routes(t *testing.T) {
	info := makeTestTableInfo()
	opts := CRUDOptions{TableName: "products"}

	program, err := GenerateCRUD(info, opts)
	if err != nil {
		t.Fatalf("GenerateCRUD error: %v", err)
	}

	routes, ok := program["routes"].([]map[string]any)
	if !ok {
		t.Fatal("routes is not []map[string]any")
	}

	if len(routes) != 5 {
		t.Fatalf("expected 5 routes, got %d", len(routes))
	}

	expectedRoutes := []struct {
		method, path, handler string
	}{
		{"GET", "/products", "listProducts"},
		{"POST", "/products", "createProduct"},
		{"GET", "/products/:id", "getProduct"},
		{"PUT", "/products/:id", "updateProduct"},
		{"DELETE", "/products/:id", "deleteProduct"},
	}

	for i, exp := range expectedRoutes {
		r := routes[i]
		if r["method"] != exp.method {
			t.Errorf("route %d: expected method %q, got %v", i, exp.method, r["method"])
		}
		if r["path"] != exp.path {
			t.Errorf("route %d: expected path %q, got %v", i, exp.path, r["path"])
		}
		if r["handler"] != exp.handler {
			t.Errorf("route %d: expected handler %q, got %v", i, exp.handler, r["handler"])
		}
	}
}

func TestGenerateCRUD_Functions(t *testing.T) {
	info := makeTestTableInfo()
	opts := CRUDOptions{TableName: "products"}

	program, err := GenerateCRUD(info, opts)
	if err != nil {
		t.Fatalf("GenerateCRUD error: %v", err)
	}

	functions, ok := program["functions"].(map[string]any)
	if !ok {
		t.Fatal("functions is not map[string]any")
	}

	expectedFunctions := []string{
		"listProducts", "createProduct", "getProduct", "updateProduct", "deleteProduct",
	}
	for _, name := range expectedFunctions {
		fn, exists := functions[name]
		if !exists {
			t.Errorf("missing function %q", name)
			continue
		}
		fnMap, ok := fn.(map[string]any)
		if !ok {
			t.Errorf("function %q is not map[string]any", name)
			continue
		}
		if _, hasSteps := fnMap["steps"]; !hasSteps {
			t.Errorf("function %q has no steps", name)
		}
	}
}

func TestGenerateCRUD_MiddlewareWithoutAuth(t *testing.T) {
	info := makeTestTableInfo()
	opts := CRUDOptions{TableName: "products", Auth: false}

	program, err := GenerateCRUD(info, opts)
	if err != nil {
		t.Fatalf("GenerateCRUD error: %v", err)
	}

	mw, ok := program["middleware"].([]string)
	if !ok {
		t.Fatal("middleware is not []string")
	}
	if len(mw) != 2 || mw[0] != "logger" || mw[1] != "recover" {
		t.Errorf("expected [logger, recover], got %v", mw)
	}
}

func TestGenerateCRUD_WithAuth(t *testing.T) {
	info := makeTestTableInfo()
	opts := CRUDOptions{TableName: "products", Auth: true}

	program, err := GenerateCRUD(info, opts)
	if err != nil {
		t.Fatalf("GenerateCRUD error: %v", err)
	}

	// JWT config should be present
	server := program["server"].(map[string]any)
	jwt, hasJWT := server["jwt"]
	if !hasJWT {
		t.Fatal("expected JWT config with auth")
	}
	jwtMap := jwt.(map[string]any)
	if jwtMap["secret_env"] != "JWT_SECRET" {
		t.Errorf("expected secret_env JWT_SECRET, got %v", jwtMap["secret_env"])
	}

	// Middleware should include cors
	mw := program["middleware"].([]string)
	if len(mw) != 3 || mw[2] != "cors" {
		t.Errorf("expected [logger, recover, cors], got %v", mw)
	}

	// Write routes (POST, PUT, DELETE) should have jwt middleware
	routes := program["routes"].([]map[string]any)
	for _, r := range routes {
		method := r["method"].(string)
		if method == "GET" {
			if _, hasMW := r["middleware"]; hasMW {
				t.Errorf("GET route %v should not have middleware", r["path"])
			}
		} else {
			routeMW, hasMW := r["middleware"]
			if !hasMW {
				t.Errorf("%s route %v should have jwt middleware", method, r["path"])
				continue
			}
			mwSlice := routeMW.([]string)
			if len(mwSlice) != 1 || mwSlice[0] != "jwt" {
				t.Errorf("%s route %v: expected [jwt], got %v", method, r["path"], mwSlice)
			}
		}
	}
}

func TestGenerateCRUD_WritableColumnsExcluded(t *testing.T) {
	info := makeTestTableInfo()
	opts := CRUDOptions{TableName: "products"}

	program, err := GenerateCRUD(info, opts)
	if err != nil {
		t.Fatalf("GenerateCRUD error: %v", err)
	}

	// Verify the create function's SQL doesn't include id, created_at, updated_at
	functions := program["functions"].(map[string]any)
	createFn := functions["createProduct"].(map[string]any)
	steps := createFn["steps"].([]map[string]any)

	// First step should be the INSERT
	insertStep := steps[0]
	withMap := insertStep["with"].(map[string]string)
	query := withMap["query"]

	if strings.Contains(query, "id,") || strings.Contains(query, ", id") {
		t.Error("INSERT should not include auto-increment id column")
	}
	if strings.Contains(query, "created_at") {
		t.Error("INSERT should not include created_at")
	}
	if strings.Contains(query, "updated_at") {
		t.Error("INSERT should not include updated_at")
	}
	if !strings.Contains(query, "name") || !strings.Contains(query, "price") || !strings.Contains(query, "active") {
		t.Errorf("INSERT should include writable columns, got: %s", query)
	}
}

func TestGenerateCRUD_ImportSection(t *testing.T) {
	info := makeTestTableInfo()
	opts := CRUDOptions{TableName: "products"}

	program, err := GenerateCRUD(info, opts)
	if err != nil {
		t.Fatalf("GenerateCRUD error: %v", err)
	}

	imports, ok := program["import"].(map[string]any)
	if !ok {
		t.Fatal("import is not map[string]any")
	}
	if imports["db"] != "io:sql" {
		t.Errorf("expected import db=io:sql, got %v", imports["db"])
	}
}

// ---------------------------------------------------------------------------
// GenerateCRUDJSON
// ---------------------------------------------------------------------------

func TestGenerateCRUDJSON_ValidJSON(t *testing.T) {
	info := makeTestTableInfo()
	opts := CRUDOptions{TableName: "products"}

	jsonStr, err := GenerateCRUDJSON(info, opts)
	if err != nil {
		t.Fatalf("GenerateCRUDJSON error: %v", err)
	}

	if jsonStr == "" {
		t.Fatal("expected non-empty JSON string")
	}

	var parsed map[string]any
	if err := json.Unmarshal([]byte(jsonStr), &parsed); err != nil {
		t.Fatalf("output is not valid JSON: %v\n%s", err, jsonStr)
	}

	if parsed["name"] != "products-api" {
		t.Errorf("expected name 'products-api', got %v", parsed["name"])
	}
}

func TestGenerateCRUDJSON_Indented(t *testing.T) {
	info := makeTestTableInfo()
	opts := CRUDOptions{TableName: "products"}

	jsonStr, err := GenerateCRUDJSON(info, opts)
	if err != nil {
		t.Fatalf("GenerateCRUDJSON error: %v", err)
	}

	// Indented JSON should contain newlines and spaces
	if !strings.Contains(jsonStr, "\n") {
		t.Error("expected indented JSON with newlines")
	}
	if !strings.Contains(jsonStr, "  ") {
		t.Error("expected indented JSON with spaces")
	}
}

// ---------------------------------------------------------------------------
// GenerateAuth
// ---------------------------------------------------------------------------

func TestGenerateAuth_ValidJSON(t *testing.T) {
	jsonStr, err := GenerateAuth()
	if err != nil {
		t.Fatalf("GenerateAuth error: %v", err)
	}

	var parsed map[string]any
	if err := json.Unmarshal([]byte(jsonStr), &parsed); err != nil {
		t.Fatalf("output is not valid JSON: %v", err)
	}

	if parsed["name"] != "auth-api" {
		t.Errorf("expected name 'auth-api', got %v", parsed["name"])
	}
}

func TestGenerateAuth_HasAllEndpoints(t *testing.T) {
	jsonStr, err := GenerateAuth()
	if err != nil {
		t.Fatalf("GenerateAuth error: %v", err)
	}

	var parsed map[string]any
	if err := json.Unmarshal([]byte(jsonStr), &parsed); err != nil {
		t.Fatalf("output is not valid JSON: %v", err)
	}

	routes, ok := parsed["routes"].([]any)
	if !ok {
		t.Fatal("routes is not []any")
	}

	if len(routes) != 5 {
		t.Fatalf("expected 5 auth routes, got %d", len(routes))
	}

	expectedPaths := []string{
		"/auth/register",
		"/auth/login",
		"/auth/refresh",
		"/auth/me",
		"/auth/password",
	}

	for i, exp := range expectedPaths {
		route := routes[i].(map[string]any)
		if route["path"] != exp {
			t.Errorf("route %d: expected path %q, got %v", i, exp, route["path"])
		}
	}
}

func TestGenerateAuth_HasAllFunctions(t *testing.T) {
	jsonStr, err := GenerateAuth()
	if err != nil {
		t.Fatalf("GenerateAuth error: %v", err)
	}

	var parsed map[string]any
	if err := json.Unmarshal([]byte(jsonStr), &parsed); err != nil {
		t.Fatalf("output is not valid JSON: %v", err)
	}

	functions, ok := parsed["functions"].(map[string]any)
	if !ok {
		t.Fatal("functions is not map[string]any")
	}

	expectedFunctions := []string{"register", "login", "refreshToken", "getProfile", "changePassword"}
	for _, name := range expectedFunctions {
		if _, exists := functions[name]; !exists {
			t.Errorf("missing auth function %q", name)
		}
	}
}

func TestGenerateAuth_JWTConfig(t *testing.T) {
	jsonStr, err := GenerateAuth()
	if err != nil {
		t.Fatalf("GenerateAuth error: %v", err)
	}

	var parsed map[string]any
	if err := json.Unmarshal([]byte(jsonStr), &parsed); err != nil {
		t.Fatalf("output is not valid JSON: %v", err)
	}

	server := parsed["server"].(map[string]any)
	jwt, ok := server["jwt"].(map[string]any)
	if !ok {
		t.Fatal("server.jwt is missing or not map[string]any")
	}
	if jwt["secret_env"] != "JWT_SECRET" {
		t.Errorf("expected secret_env JWT_SECRET, got %v", jwt["secret_env"])
	}
	if jwt["algorithm"] != "HS256" {
		t.Errorf("expected algorithm HS256, got %v", jwt["algorithm"])
	}
	if jwt["expiry"] != "24h" {
		t.Errorf("expected expiry 24h, got %v", jwt["expiry"])
	}
}

func TestGenerateAuth_ProtectedRoutes(t *testing.T) {
	jsonStr, err := GenerateAuth()
	if err != nil {
		t.Fatalf("GenerateAuth error: %v", err)
	}

	var parsed map[string]any
	if err := json.Unmarshal([]byte(jsonStr), &parsed); err != nil {
		t.Fatalf("output is not valid JSON: %v", err)
	}

	routes := parsed["routes"].([]any)

	// /auth/refresh, /auth/me, /auth/password should have jwt middleware
	protectedPaths := map[string]bool{
		"/auth/refresh":  true,
		"/auth/me":       true,
		"/auth/password": true,
	}

	for _, r := range routes {
		route := r.(map[string]any)
		path := route["path"].(string)
		if protectedPaths[path] {
			mw, hasMW := route["middleware"]
			if !hasMW {
				t.Errorf("route %s should have jwt middleware", path)
				continue
			}
			mwSlice := mw.([]any)
			if len(mwSlice) != 1 || mwSlice[0] != "jwt" {
				t.Errorf("route %s: expected [jwt], got %v", path, mwSlice)
			}
		}
	}
}

// ---------------------------------------------------------------------------
// GenerateAuthSQL
// ---------------------------------------------------------------------------

func TestGenerateAuthSQL_ContainsCreateTable(t *testing.T) {
	sql := GenerateAuthSQL()

	if !strings.Contains(sql, "CREATE TABLE IF NOT EXISTS users") {
		t.Error("expected CREATE TABLE IF NOT EXISTS users")
	}
}

func TestGenerateAuthSQL_HasRequiredColumns(t *testing.T) {
	sql := GenerateAuthSQL()

	requiredColumns := []string{"id", "email", "password_hash", "created_at", "updated_at"}
	for _, col := range requiredColumns {
		if !strings.Contains(sql, col) {
			t.Errorf("SQL should contain column %q", col)
		}
	}
}

func TestGenerateAuthSQL_HasConstraints(t *testing.T) {
	sql := GenerateAuthSQL()

	if !strings.Contains(sql, "PRIMARY KEY") {
		t.Error("SQL should contain PRIMARY KEY")
	}
	if !strings.Contains(sql, "UNIQUE") {
		t.Error("SQL should contain UNIQUE constraint on email")
	}
	if !strings.Contains(sql, "NOT NULL") {
		t.Error("SQL should contain NOT NULL constraints")
	}
}

// ---------------------------------------------------------------------------
// GenerateProject
// ---------------------------------------------------------------------------

func TestGenerateProject_BasicFiles(t *testing.T) {
	files := GenerateProject("myapp", false)

	expectedFiles := []string{
		"api.json",
		".env.example",
		"README.md",
	}

	for _, name := range expectedFiles {
		if _, exists := files[name]; !exists {
			t.Errorf("missing file %q", name)
		}
	}
}

func TestGenerateProject_HasTemplateDir(t *testing.T) {
	files := GenerateProject("myapp", false)

	hasTemplates := false
	hasPublic := false
	for path := range files {
		if strings.HasPrefix(path, "templates") || strings.Contains(path, "templates") {
			hasTemplates = true
		}
		if strings.HasPrefix(path, "public") || strings.Contains(path, "public") {
			hasPublic = true
		}
	}

	if !hasTemplates {
		t.Error("expected files in templates/ directory")
	}
	if !hasPublic {
		t.Error("expected files in public/ directory")
	}
}

func TestGenerateProject_APIJsonValid(t *testing.T) {
	files := GenerateProject("myapp", false)

	apiJSON, exists := files["api.json"]
	if !exists {
		t.Fatal("missing api.json")
	}

	var parsed map[string]any
	if err := json.Unmarshal([]byte(apiJSON), &parsed); err != nil {
		t.Fatalf("api.json is not valid JSON: %v", err)
	}

	if parsed["name"] != "myapp" {
		t.Errorf("expected name 'myapp', got %v", parsed["name"])
	}
	if parsed["go_json"] != "1" {
		t.Errorf("expected go_json '1', got %v", parsed["go_json"])
	}
}

func TestGenerateProject_WithAuth(t *testing.T) {
	files := GenerateProject("myapp", true)

	// api.json should have JWT config
	apiJSON := files["api.json"]
	var parsed map[string]any
	if err := json.Unmarshal([]byte(apiJSON), &parsed); err != nil {
		t.Fatalf("api.json is not valid JSON: %v", err)
	}

	server := parsed["server"].(map[string]any)
	if _, hasJWT := server["jwt"]; !hasJWT {
		t.Error("expected JWT config in api.json when auth is enabled")
	}

	// .env.example should have JWT_SECRET
	envExample := files[".env.example"]
	if !strings.Contains(envExample, "JWT_SECRET") {
		t.Error("expected JWT_SECRET in .env.example when auth is enabled")
	}

	// Should have migrations directory with users SQL
	hasMigration := false
	for path, content := range files {
		if strings.Contains(path, "migrations") {
			hasMigration = true
			if !strings.Contains(content, "CREATE TABLE") {
				t.Errorf("migration file %s should contain CREATE TABLE", path)
			}
		}
	}
	if !hasMigration {
		t.Error("expected migration file when auth is enabled")
	}
}

func TestGenerateProject_WithoutAuth_NoJWT(t *testing.T) {
	files := GenerateProject("myapp", false)

	apiJSON := files["api.json"]
	var parsed map[string]any
	if err := json.Unmarshal([]byte(apiJSON), &parsed); err != nil {
		t.Fatalf("api.json is not valid JSON: %v", err)
	}

	server := parsed["server"].(map[string]any)
	if _, hasJWT := server["jwt"]; hasJWT {
		t.Error("expected no JWT config when auth is disabled")
	}

	envExample := files[".env.example"]
	if strings.Contains(envExample, "JWT_SECRET") {
		t.Error("expected no JWT_SECRET in .env.example when auth is disabled")
	}
}

func TestGenerateProject_ReadmeContainsName(t *testing.T) {
	files := GenerateProject("cool-project", false)

	readme := files["README.md"]
	if !strings.Contains(readme, "cool-project") {
		t.Error("README.md should contain the project name")
	}
}

func TestGenerateProject_PublicIndexContainsName(t *testing.T) {
	files := GenerateProject("cool-project", false)

	found := false
	for path, content := range files {
		if strings.Contains(path, "public") && strings.Contains(path, "index.html") {
			found = true
			if !strings.Contains(content, "cool-project") {
				t.Error("public/index.html should contain the project name")
			}
		}
	}
	if !found {
		t.Error("expected public/index.html in project files")
	}
}

// ---------------------------------------------------------------------------
// MapDBTypeToGoJSON
// ---------------------------------------------------------------------------

func TestMapDBTypeToGoJSON(t *testing.T) {
	tests := []struct {
		input, expected string
	}{
		{"INT", "int"},
		{"INTEGER", "int"},
		{"BIGINT", "int"},
		{"SMALLINT", "int"},
		{"TINYINT", "int"},
		{"FLOAT", "float"},
		{"DOUBLE", "float"},
		{"DECIMAL(10,2)", "float"},
		{"NUMERIC", "float"},
		{"REAL", "float"},
		{"BOOL", "bool"},
		{"BOOLEAN", "bool"},
		{"DATE", "string"},
		{"DATETIME", "string"},
		{"TIMESTAMP", "string"},
		{"TIME", "string"},
		{"JSON", "any"},
		{"JSONB", "any"},
		{"TEXT", "string"},
		{"VARCHAR(255)", "string"},
		{"CHAR(10)", "string"},
		{"BLOB", "string"},
	}

	for _, tt := range tests {
		result := MapDBTypeToGoJSON(tt.input)
		if result != tt.expected {
			t.Errorf("MapDBTypeToGoJSON(%q) = %q, want %q", tt.input, result, tt.expected)
		}
	}
}

func TestMapDBTypeToGoJSON_CaseInsensitive(t *testing.T) {
	tests := []struct {
		input, expected string
	}{
		{"int", "int"},
		{"Integer", "int"},
		{"float", "float"},
		{"bool", "bool"},
		{"json", "any"},
		{"text", "string"},
	}

	for _, tt := range tests {
		result := MapDBTypeToGoJSON(tt.input)
		if result != tt.expected {
			t.Errorf("MapDBTypeToGoJSON(%q) = %q, want %q", tt.input, result, tt.expected)
		}
	}
}

// ---------------------------------------------------------------------------
// MapDBTypeToOpenAPI
// ---------------------------------------------------------------------------

func TestMapDBTypeToOpenAPI(t *testing.T) {
	tests := []struct {
		input, expected string
	}{
		{"INT", "integer"},
		{"INTEGER", "integer"},
		{"BIGINT", "integer"},
		{"SMALLINT", "integer"},
		{"FLOAT", "number"},
		{"DOUBLE", "number"},
		{"DECIMAL(10,2)", "number"},
		{"NUMERIC", "number"},
		{"REAL", "number"},
		{"BOOL", "boolean"},
		{"BOOLEAN", "boolean"},
		{"TEXT", "string"},
		{"VARCHAR(255)", "string"},
		{"DATE", "string"},
		{"TIMESTAMP", "string"},
		{"JSON", "string"},
		{"BLOB", "string"},
	}

	for _, tt := range tests {
		result := MapDBTypeToOpenAPI(tt.input)
		if result != tt.expected {
			t.Errorf("MapDBTypeToOpenAPI(%q) = %q, want %q", tt.input, result, tt.expected)
		}
	}
}

func TestMapDBTypeToOpenAPI_CaseInsensitive(t *testing.T) {
	tests := []struct {
		input, expected string
	}{
		{"int", "integer"},
		{"float", "number"},
		{"bool", "boolean"},
		{"text", "string"},
	}

	for _, tt := range tests {
		result := MapDBTypeToOpenAPI(tt.input)
		if result != tt.expected {
			t.Errorf("MapDBTypeToOpenAPI(%q) = %q, want %q", tt.input, result, tt.expected)
		}
	}
}

// ---------------------------------------------------------------------------
// BuiltinPatterns
// ---------------------------------------------------------------------------

func TestBuiltinPatterns(t *testing.T) {
	patterns := BuiltinPatterns()

	expected := []string{"simple", "service-layer", "ddd", "hexagonal"}

	if len(patterns) != len(expected) {
		t.Fatalf("expected %d patterns, got %d: %v", len(expected), len(patterns), patterns)
	}

	for i, exp := range expected {
		if patterns[i] != exp {
			t.Errorf("pattern %d: expected %q, got %q", i, exp, patterns[i])
		}
	}
}

func TestBuiltinPatterns_NotEmpty(t *testing.T) {
	patterns := BuiltinPatterns()
	if len(patterns) == 0 {
		t.Error("BuiltinPatterns should return at least one pattern")
	}
}

// ---------------------------------------------------------------------------
// singularize (unexported — tested directly since same package)
// ---------------------------------------------------------------------------

func TestSingularize(t *testing.T) {
	tests := []struct {
		input, expected string
	}{
		{"products", "product"},
		{"users", "user"},
		{"categories", "category"},
		{"companies", "company"},
		{"boxes", "box"},
		{"addresses", "address"},
		{"buses", "bus"},
		{"foxes", "fox"},
		{"class", "class"},       // ends in ss, should not strip
		{"boss", "boss"},         // ends in ss, should not strip
		{"product", "product"},   // already singular
		{"data", "data"},         // no trailing s
	}

	for _, tt := range tests {
		result := singularize(tt.input)
		if result != tt.expected {
			t.Errorf("singularize(%q) = %q, want %q", tt.input, result, tt.expected)
		}
	}
}

func TestSingularize_EmptyString(t *testing.T) {
	result := singularize("")
	if result != "" {
		t.Errorf("singularize('') = %q, want ''", result)
	}
}

// ---------------------------------------------------------------------------
// capitalize (unexported — tested directly since same package)
// ---------------------------------------------------------------------------

func TestCapitalize(t *testing.T) {
	tests := []struct {
		input, expected string
	}{
		{"hello", "Hello"},
		{"world", "World"},
		{"a", "A"},
		{"ABC", "ABC"},
		{"already", "Already"},
		{"products", "Products"},
	}

	for _, tt := range tests {
		result := capitalize(tt.input)
		if result != tt.expected {
			t.Errorf("capitalize(%q) = %q, want %q", tt.input, result, tt.expected)
		}
	}
}

func TestCapitalize_EmptyString(t *testing.T) {
	result := capitalize("")
	if result != "" {
		t.Errorf("capitalize('') = %q, want ''", result)
	}
}

// ---------------------------------------------------------------------------
// Integration: singularize used in route naming
// ---------------------------------------------------------------------------

func TestSingularize_InRouteNaming(t *testing.T) {
	// Verify singularize works correctly for CRUD route handler naming
	info := &TableInfo{
		Name:       "categories",
		PrimaryKey: []string{"id"},
		Columns: []ColumnInfo{
			{Name: "id", DBType: "INTEGER", GoJSONType: "int", OpenAPIType: "integer", IsAutoIncr: true},
			{Name: "name", DBType: "TEXT", GoJSONType: "string", OpenAPIType: "string"},
		},
	}
	opts := CRUDOptions{TableName: "categories"}

	program, err := GenerateCRUD(info, opts)
	if err != nil {
		t.Fatalf("GenerateCRUD error: %v", err)
	}

	routes := program["routes"].([]map[string]any)

	// POST handler should use singularized name: createCategory (not createCategorie)
	postRoute := routes[1]
	if postRoute["handler"] != "createCategory" {
		t.Errorf("expected handler 'createCategory', got %v", postRoute["handler"])
	}

	// GET /:id handler should use singularized name
	getOneRoute := routes[2]
	if getOneRoute["handler"] != "getCategory" {
		t.Errorf("expected handler 'getCategory', got %v", getOneRoute["handler"])
	}
}

// ---------------------------------------------------------------------------
// GenerateCRUD with custom primary key
// ---------------------------------------------------------------------------

func TestGenerateCRUD_CustomPrimaryKey(t *testing.T) {
	info := &TableInfo{
		Name:       "items",
		PrimaryKey: []string{"item_id"},
		Columns: []ColumnInfo{
			{Name: "item_id", DBType: "INTEGER", GoJSONType: "int", OpenAPIType: "integer", IsAutoIncr: true},
			{Name: "title", DBType: "TEXT", GoJSONType: "string", OpenAPIType: "string"},
		},
	}
	opts := CRUDOptions{TableName: "items"}

	program, err := GenerateCRUD(info, opts)
	if err != nil {
		t.Fatalf("GenerateCRUD error: %v", err)
	}

	// The get function should use item_id in WHERE clause
	functions := program["functions"].(map[string]any)
	getFn := functions["getItem"].(map[string]any)
	steps := getFn["steps"].([]map[string]any)
	withMap := steps[0]["with"].(map[string]string)
	query := withMap["query"]

	if !strings.Contains(query, "WHERE item_id = ?") {
		t.Errorf("expected WHERE item_id = ?, got: %s", query)
	}
}

// ---------------------------------------------------------------------------
// GenerateCRUD JSON round-trip
// ---------------------------------------------------------------------------

func TestGenerateCRUDJSON_RoundTrip(t *testing.T) {
	info := makeTestTableInfo()
	opts := CRUDOptions{TableName: "products", Auth: true}

	jsonStr, err := GenerateCRUDJSON(info, opts)
	if err != nil {
		t.Fatalf("GenerateCRUDJSON error: %v", err)
	}

	// Parse the JSON
	var parsed map[string]any
	if err := json.Unmarshal([]byte(jsonStr), &parsed); err != nil {
		t.Fatalf("JSON round-trip failed: %v", err)
	}

	// Verify key fields survived serialization
	if parsed["name"] != "products-api" {
		t.Errorf("name mismatch after round-trip: %v", parsed["name"])
	}

	routes, ok := parsed["routes"].([]any)
	if !ok {
		t.Fatal("routes not preserved in JSON round-trip")
	}
	if len(routes) != 5 {
		t.Errorf("expected 5 routes after round-trip, got %d", len(routes))
	}
}

// ---------------------------------------------------------------------------
// getWritableColumns (unexported)
// ---------------------------------------------------------------------------

func TestGetWritableColumns(t *testing.T) {
	info := makeTestTableInfo()
	writable := getWritableColumns(info)

	// Should exclude: id (auto-incr + PK), created_at, updated_at
	// Should include: name, price, active
	if len(writable) != 3 {
		t.Fatalf("expected 3 writable columns, got %d: %v", len(writable), columnNames(writable))
	}

	names := columnNames(writable)
	for _, expected := range []string{"name", "price", "active"} {
		found := false
		for _, n := range names {
			if n == expected {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("expected writable column %q, not found in %v", expected, names)
		}
	}
}

func TestGetWritableColumns_ExcludesGenerated(t *testing.T) {
	info := &TableInfo{
		Name:       "t",
		PrimaryKey: []string{"id"},
		Columns: []ColumnInfo{
			{Name: "id", IsAutoIncr: true},
			{Name: "computed", IsGenerated: true},
			{Name: "normal"},
		},
	}

	writable := getWritableColumns(info)
	if len(writable) != 1 || writable[0].Name != "normal" {
		t.Errorf("expected only 'normal', got %v", columnNames(writable))
	}
}

// ---------------------------------------------------------------------------
// detectDriver (unexported)
// ---------------------------------------------------------------------------

func TestDetectDriver(t *testing.T) {
	tests := []struct {
		dsn, expected string
	}{
		{"postgres://user:pass@localhost/db", "postgres"},
		{"host=localhost dbname=test", "postgres"},
		{"user:pass@tcp(localhost:3306)/db", "mysql"},
		{"mysql://user:pass@localhost/db", "mysql"},
		{"./app.db", "sqlite3"},
		{"file:test.db", "sqlite3"},
		{":memory:", "sqlite3"},
	}

	for _, tt := range tests {
		result := detectDriver(tt.dsn)
		if result != tt.expected {
			t.Errorf("detectDriver(%q) = %q, want %q", tt.dsn, result, tt.expected)
		}
	}
}

// ---------------------------------------------------------------------------
// goJSONToOpenAPI / goJSONToDB (unexported)
// ---------------------------------------------------------------------------

func TestGoJSONToOpenAPI(t *testing.T) {
	tests := []struct {
		input, expected string
	}{
		{"int", "integer"},
		{"float", "number"},
		{"bool", "boolean"},
		{"string", "string"},
		{"any", "string"},
		{"unknown", "string"},
	}

	for _, tt := range tests {
		result := goJSONToOpenAPI(tt.input)
		if result != tt.expected {
			t.Errorf("goJSONToOpenAPI(%q) = %q, want %q", tt.input, result, tt.expected)
		}
	}
}

func TestGoJSONToDB(t *testing.T) {
	tests := []struct {
		input, expected string
	}{
		{"int", "INTEGER"},
		{"float", "REAL"},
		{"bool", "BOOLEAN"},
		{"string", "TEXT"},
		{"any", "TEXT"},
		{"unknown", "TEXT"},
	}

	for _, tt := range tests {
		result := goJSONToDB(tt.input)
		if result != tt.expected {
			t.Errorf("goJSONToDB(%q) = %q, want %q", tt.input, result, tt.expected)
		}
	}
}

// ---------------------------------------------------------------------------
// Edge cases
// ---------------------------------------------------------------------------

func TestGenerateCRUD_NoPrimaryKey(t *testing.T) {
	info := &TableInfo{
		Name:       "logs",
		PrimaryKey: nil,
		Columns: []ColumnInfo{
			{Name: "message", DBType: "TEXT", GoJSONType: "string"},
			{Name: "level", DBType: "TEXT", GoJSONType: "string"},
		},
	}
	opts := CRUDOptions{TableName: "logs"}

	program, err := GenerateCRUD(info, opts)
	if err != nil {
		t.Fatalf("GenerateCRUD error: %v", err)
	}

	// Should default to "id" for WHERE clauses
	functions := program["functions"].(map[string]any)
	getFn := functions["getLog"].(map[string]any)
	steps := getFn["steps"].([]map[string]any)
	withMap := steps[0]["with"].(map[string]string)
	query := withMap["query"]

	if !strings.Contains(query, "WHERE id = ?") {
		t.Errorf("expected default WHERE id = ?, got: %s", query)
	}
}

func TestGenerateProject_DifferentNames(t *testing.T) {
	names := []string{"my-app", "test_project", "api"}
	for _, name := range names {
		files := GenerateProject(name, false)
		apiJSON := files["api.json"]

		var parsed map[string]any
		if err := json.Unmarshal([]byte(apiJSON), &parsed); err != nil {
			t.Fatalf("api.json for %q is not valid JSON: %v", name, err)
		}
		if parsed["name"] != name {
			t.Errorf("expected name %q, got %v", name, parsed["name"])
		}
	}
}

// ---------------------------------------------------------------------------
// helpers for tests
// ---------------------------------------------------------------------------

func columnNames(cols []ColumnInfo) []string {
	names := make([]string, len(cols))
	for i, c := range cols {
		names[i] = c.Name
	}
	return names
}
