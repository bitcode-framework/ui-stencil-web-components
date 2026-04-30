package bridge

import (
	"testing"
)

// mockDB implements DB interface for testing.
type mockDB struct {
	queryResult []map[string]any
	execResult  *ExecDBResult
}

func (m *mockDB) Query(sql string, args ...any) ([]map[string]any, error) {
	return m.queryResult, nil
}
func (m *mockDB) Execute(sql string, args ...any) (*ExecDBResult, error) {
	return m.execResult, nil
}

// mockHTTPClient implements HTTPClient for testing.
type mockHTTPClient struct {
	response *HTTPResponse
}

func (m *mockHTTPClient) Get(url string, opts *HTTPOptions) (*HTTPResponse, error)    { return m.response, nil }
func (m *mockHTTPClient) Post(url string, opts *HTTPOptions) (*HTTPResponse, error)   { return m.response, nil }
func (m *mockHTTPClient) Put(url string, opts *HTTPOptions) (*HTTPResponse, error)    { return m.response, nil }
func (m *mockHTTPClient) Patch(url string, opts *HTTPOptions) (*HTTPResponse, error)  { return m.response, nil }
func (m *mockHTTPClient) Delete(url string, opts *HTTPOptions) (*HTTPResponse, error) { return m.response, nil }

// mockCache implements Cache for testing.
type mockCache struct {
	store map[string]any
}

func (m *mockCache) Get(key string) (any, error)                    { return m.store[key], nil }
func (m *mockCache) Set(key string, value any, opts *CacheSetOptions) error { m.store[key] = value; return nil }
func (m *mockCache) Del(key string) error                           { delete(m.store, key); return nil }

// mockFS implements FS for testing.
type mockFS struct{}

func (m *mockFS) Read(path string) (string, error)    { return "file content", nil }
func (m *mockFS) Write(path, content string) error    { return nil }
func (m *mockFS) Exists(path string) (bool, error)    { return true, nil }
func (m *mockFS) List(path string) ([]string, error)  { return []string{"a.txt", "b.txt"}, nil }
func (m *mockFS) Mkdir(path string) error             { return nil }
func (m *mockFS) Remove(path string) error            { return nil }

// mockEnv implements EnvReader for testing.
type mockEnv struct{ vars map[string]string }

func (m *mockEnv) Get(key string) (string, error) {
	if v, ok := m.vars[key]; ok {
		return v, nil
	}
	return "", NewError(ErrEnvAccessDenied, "env key not found: "+key)
}

// mockConfig implements ConfigReader for testing.
type mockConfig struct{ data map[string]any }

func (m *mockConfig) Get(key string) any { return m.data[key] }

// mockEmitter implements EventEmitter for testing.
type mockEmitter struct{ events []string }

func (m *mockEmitter) Emit(event string, data map[string]any) error {
	m.events = append(m.events, event)
	return nil
}

// mockCaller implements ProcessCaller for testing.
type mockCaller struct{}

func (m *mockCaller) Call(process string, input map[string]any) (any, error) {
	return map[string]any{"called": process}, nil
}

// mockExecer implements CommandExecutor for testing.
type mockExecer struct{}

func (m *mockExecer) Exec(cmd string, args []string, opts *ExecOptions) (*ExecResult, error) {
	return &ExecResult{Stdout: "output", ExitCode: 0}, nil
}

// mockLogger implements Logger for testing.
type mockLogger struct{ messages []string }

func (m *mockLogger) Log(level, msg string, data ...map[string]any) {
	m.messages = append(m.messages, level+": "+msg)
}

// mockEmail implements EmailSender for testing.
type mockEmail struct{ sent bool }

func (m *mockEmail) Send(opts EmailOptions) error { m.sent = true; return nil }

// mockNotify implements Notifier for testing.
type mockNotify struct{}

func (m *mockNotify) Send(opts NotifyOptions) error                  { return nil }
func (m *mockNotify) Broadcast(channel string, data map[string]any) error { return nil }

// mockStorage implements Storage for testing.
type mockStorage struct{}

func (m *mockStorage) Upload(opts UploadOptions) (*Attachment, error) {
	return &Attachment{ID: "att-1", URL: "http://example.com/file.pdf"}, nil
}
func (m *mockStorage) URL(id string) (string, error)      { return "http://example.com/" + id, nil }
func (m *mockStorage) Download(id string) ([]byte, error) { return []byte("data"), nil }
func (m *mockStorage) Delete(id string) error             { return nil }

// mockI18N implements I18N for testing.
type mockI18N struct{}

func (m *mockI18N) Translate(locale, key string) string { return "translated:" + key }

// mockSecurity implements SecurityChecker for testing.
type mockSecurity struct{}

func (m *mockSecurity) Permissions(modelName string) (*ModelPermissions, error) {
	return &ModelPermissions{Read: true, Write: true, Create: true, Delete: true}, nil
}
func (m *mockSecurity) HasGroup(groupName string) (bool, error) { return true, nil }
func (m *mockSecurity) Groups() ([]string, error)               { return []string{"admin"}, nil }

// mockAudit implements AuditLogger for testing.
type mockAudit struct{ logged bool }

func (m *mockAudit) Log(opts AuditOptions) error { m.logged = true; return nil }

// mockCrypto implements Crypto for testing.
type mockCrypto struct{}

func (m *mockCrypto) Encrypt(plaintext string) (string, error)    { return "enc:" + plaintext, nil }
func (m *mockCrypto) Decrypt(ciphertext string) (string, error)   { return "dec:" + ciphertext, nil }
func (m *mockCrypto) Hash(value string) (string, error)           { return "hash:" + value, nil }
func (m *mockCrypto) Verify(value, hash string) (bool, error)     { return hash == "hash:"+value, nil }

// mockExecution implements ExecutionLog for testing.
type mockExecution struct{}

func (m *mockExecution) Search(opts ExecutionSearchOptions) ([]map[string]any, error) {
	return []map[string]any{{"id": "exec-1"}}, nil
}
func (m *mockExecution) Get(id string, opts ...GetOptions) (map[string]any, error) {
	return map[string]any{"id": id, "status": "completed"}, nil
}
func (m *mockExecution) Current() *ExecutionInfo {
	return &ExecutionInfo{ID: "current-exec"}
}
func (m *mockExecution) Retry(id string) (map[string]any, error) {
	return map[string]any{"id": id, "retried": true}, nil
}
func (m *mockExecution) Cancel(id string) error { return nil }

// mockModelFactory implements ModelFactory for testing.
type mockModelFactory struct{}

func (m *mockModelFactory) Model(name string, session Session, sudo bool) ModelHandle {
	return &mockModelHandle{name: name, sudo: sudo}
}

// mockModelHandle implements ModelHandle for testing.
type mockModelHandle struct {
	name string
	sudo bool
}

func (m *mockModelHandle) Search(opts SearchOptions) ([]map[string]any, error) {
	return []map[string]any{{"id": "1", "name": "test"}}, nil
}
func (m *mockModelHandle) Get(id string, opts ...GetOptions) (map[string]any, error) {
	return map[string]any{"id": id, "name": "test"}, nil
}
func (m *mockModelHandle) Create(data map[string]any) (map[string]any, error) {
	data["id"] = "new-1"
	return data, nil
}
func (m *mockModelHandle) Write(id string, data map[string]any) error { return nil }
func (m *mockModelHandle) Delete(id string) error                     { return nil }
func (m *mockModelHandle) Count(opts SearchOptions) (int64, error)    { return 5, nil }
func (m *mockModelHandle) Sum(field string, opts SearchOptions) (float64, error) {
	return 100.0, nil
}
func (m *mockModelHandle) Upsert(data map[string]any, uniqueFields []string) (map[string]any, error) {
	data["id"] = "upsert-1"
	return data, nil
}
func (m *mockModelHandle) CreateMany(records []map[string]any) ([]map[string]any, error) {
	return records, nil
}
func (m *mockModelHandle) WriteMany(ids []string, data map[string]any) (*BulkResult, error) {
	return &BulkResult{Affected: int64(len(ids))}, nil
}
func (m *mockModelHandle) DeleteMany(ids []string) (*BulkResult, error) {
	return &BulkResult{Affected: int64(len(ids))}, nil
}
func (m *mockModelHandle) UpsertMany(records []map[string]any, uniqueFields []string) ([]map[string]any, error) {
	return records, nil
}
func (m *mockModelHandle) AddRelation(id, field string, relatedIDs []string) error    { return nil }
func (m *mockModelHandle) RemoveRelation(id, field string, relatedIDs []string) error { return nil }
func (m *mockModelHandle) SetRelation(id, field string, relatedIDs []string) error    { return nil }
func (m *mockModelHandle) LoadRelation(id, field string) ([]map[string]any, error) {
	return []map[string]any{{"id": "rel-1"}}, nil
}
func (m *mockModelHandle) Sudo() SudoModelHandle { return nil }

// mockTxManager implements TxManager for testing.
type mockTxManager struct{}

func (m *mockTxManager) RunTx(parent *Context, fn func(tx *Context) error) error {
	return fn(parent)
}

func newTestContext() *Context {
	return NewContext(ContextDeps{
		TxManager: &mockTxManager{},
		Model:     &mockModelFactory{},
		DB:        &mockDB{queryResult: []map[string]any{{"id": 1, "name": "Alice"}}, execResult: &ExecDBResult{RowsAffected: 1}},
		HTTP:      &mockHTTPClient{response: &HTTPResponse{Status: 200, Headers: map[string]string{"Content-Type": "application/json"}, Body: map[string]any{"ok": true}}},
		Cache:     &mockCache{store: map[string]any{"key1": "value1"}},
		FS:        &mockFS{},
		Session:   Session{UserID: "user-1", Username: "admin", Email: "admin@test.com", TenantID: "t1", Groups: []string{"admin"}, Locale: "en"},
		Config:    &mockConfig{data: map[string]any{"app.name": "TestApp"}},
		Env:       &mockEnv{vars: map[string]string{"APP_ENV": "test"}},
		Emitter:   &mockEmitter{},
		Caller:    &mockCaller{},
		Execer:    &mockExecer{},
		Logger:    &mockLogger{},
		Email:     &mockEmail{},
		Notify:    &mockNotify{},
		Storage:   &mockStorage{},
		I18N:      &mockI18N{},
		Security:  &mockSecurity{},
		Audit:     &mockAudit{},
		Crypto:    &mockCrypto{},
		Execution: &mockExecution{},
	})
}

func TestBuildGoJSONExtension_Structure(t *testing.T) {
	bc := newTestContext()
	ext := BuildGoJSONExtension(bc)

	if ext.Name != "bitcode" {
		t.Errorf("expected extension name 'bitcode', got %q", ext.Name)
	}

	expectedKeys := []string{
		"model", "db", "http", "cache", "fs", "env", "config", "session",
		"log", "emit", "call", "exec", "email", "notify", "storage",
		"t", "security", "audit", "crypto", "execution",
	}

	for _, key := range expectedKeys {
		if _, ok := ext.Functions[key]; !ok {
			t.Errorf("missing expected key %q in extension functions", key)
		}
	}

	if len(ext.Functions) != len(expectedKeys) {
		t.Errorf("expected %d functions, got %d", len(expectedKeys), len(ext.Functions))
	}
}

func TestBuildGoJSONExtension_NestedNamespaces(t *testing.T) {
	bc := newTestContext()
	ext := BuildGoJSONExtension(bc)

	dbNS, ok := ext.Functions["db"].(map[string]any)
	if !ok {
		t.Fatal("db should be a nested map[string]any")
	}
	if _, ok := dbNS["query"]; !ok {
		t.Error("db namespace missing 'query'")
	}
	if _, ok := dbNS["execute"]; !ok {
		t.Error("db namespace missing 'execute'")
	}

	httpNS, ok := ext.Functions["http"].(map[string]any)
	if !ok {
		t.Fatal("http should be a nested map[string]any")
	}
	for _, method := range []string{"get", "post", "put", "patch", "delete"} {
		if _, ok := httpNS[method]; !ok {
			t.Errorf("http namespace missing '%s'", method)
		}
	}

	cacheNS, ok := ext.Functions["cache"].(map[string]any)
	if !ok {
		t.Fatal("cache should be a nested map[string]any")
	}
	for _, method := range []string{"get", "set", "del"} {
		if _, ok := cacheNS[method]; !ok {
			t.Errorf("cache namespace missing '%s'", method)
		}
	}

	fsNS, ok := ext.Functions["fs"].(map[string]any)
	if !ok {
		t.Fatal("fs should be a nested map[string]any")
	}
	for _, method := range []string{"read", "write", "exists", "list", "mkdir", "remove"} {
		if _, ok := fsNS[method]; !ok {
			t.Errorf("fs namespace missing '%s'", method)
		}
	}

	secNS, ok := ext.Functions["security"].(map[string]any)
	if !ok {
		t.Fatal("security should be a nested map[string]any")
	}
	for _, method := range []string{"permissions", "hasGroup", "groups"} {
		if _, ok := secNS[method]; !ok {
			t.Errorf("security namespace missing '%s'", method)
		}
	}

	cryptoNS, ok := ext.Functions["crypto"].(map[string]any)
	if !ok {
		t.Fatal("crypto should be a nested map[string]any")
	}
	for _, method := range []string{"encrypt", "decrypt", "hash", "verify"} {
		if _, ok := cryptoNS[method]; !ok {
			t.Errorf("crypto namespace missing '%s'", method)
		}
	}
}

func TestBuildGoJSONExtension_DBQuery(t *testing.T) {
	bc := newTestContext()
	ext := BuildGoJSONExtension(bc)

	dbNS := ext.Functions["db"].(map[string]any)
	queryFn := dbNS["query"].(func(params ...any) (any, error))

	result, err := queryFn("SELECT * FROM users")
	if err != nil {
		t.Fatalf("db.query error: %v", err)
	}

	rows, ok := result.([]any)
	if !ok {
		t.Fatalf("expected []any, got %T", result)
	}
	if len(rows) != 1 {
		t.Errorf("expected 1 row, got %d", len(rows))
	}
}

func TestBuildGoJSONExtension_DBQuery_TypeValidation(t *testing.T) {
	bc := newTestContext()
	ext := BuildGoJSONExtension(bc)

	dbNS := ext.Functions["db"].(map[string]any)
	queryFn := dbNS["query"].(func(params ...any) (any, error))

	_, err := queryFn(123)
	if err == nil {
		t.Error("expected error for non-string SQL")
	}
}

func TestBuildGoJSONExtension_Env(t *testing.T) {
	bc := newTestContext()
	ext := BuildGoJSONExtension(bc)

	envFn := ext.Functions["env"].(func(key string) (any, error))
	val, err := envFn("APP_ENV")
	if err != nil {
		t.Fatalf("env error: %v", err)
	}
	if val != "test" {
		t.Errorf("expected 'test', got %v", val)
	}
}

func TestBuildGoJSONExtension_Config(t *testing.T) {
	bc := newTestContext()
	ext := BuildGoJSONExtension(bc)

	configFn := ext.Functions["config"].(func(key string) any)
	val := configFn("app.name")
	if val != "TestApp" {
		t.Errorf("expected 'TestApp', got %v", val)
	}
}

func TestBuildGoJSONExtension_T(t *testing.T) {
	bc := newTestContext()
	ext := BuildGoJSONExtension(bc)

	tFn := ext.Functions["t"].(func(key string) any)
	val := tFn("hello.world")
	if val != "translated:hello.world" {
		t.Errorf("expected 'translated:hello.world', got %v", val)
	}
}

func TestBuildGoJSONExtension_Model(t *testing.T) {
	bc := newTestContext()
	ext := BuildGoJSONExtension(bc)

	modelFn := ext.Functions["model"].(func(name string) any)
	proxy := modelFn("users")

	proxyMap, ok := proxy.(map[string]any)
	if !ok {
		t.Fatalf("model proxy should be map[string]any, got %T", proxy)
	}

	expectedMethods := []string{
		"search", "get", "create", "write", "delete", "count", "sum", "upsert",
		"createMany", "writeMany", "deleteMany", "upsertMany",
		"addRelation", "removeRelation", "setRelation", "loadRelation",
		"sudo",
	}
	for _, method := range expectedMethods {
		if _, ok := proxyMap[method]; !ok {
			t.Errorf("model proxy missing method '%s'", method)
		}
	}
}

func TestBuildGoJSONExtension_ModelSearch(t *testing.T) {
	bc := newTestContext()
	ext := BuildGoJSONExtension(bc)

	modelFn := ext.Functions["model"].(func(name string) any)
	proxy := modelFn("users").(map[string]any)

	searchFn := proxy["search"].(func(params ...any) (any, error))
	result, err := searchFn(map[string]any{})
	if err != nil {
		t.Fatalf("model.search error: %v", err)
	}
	if result == nil {
		t.Error("expected non-nil result")
	}
}

func TestBuildGoJSONExtension_CacheGetSet(t *testing.T) {
	bc := newTestContext()
	ext := BuildGoJSONExtension(bc)

	cacheNS := ext.Functions["cache"].(map[string]any)
	getFn := cacheNS["get"].(func(params ...any) (any, error))

	val, err := getFn("key1")
	if err != nil {
		t.Fatalf("cache.get error: %v", err)
	}
	if val != "value1" {
		t.Errorf("expected 'value1', got %v", val)
	}
}

func TestConvertToAny_Primitives(t *testing.T) {
	tests := []struct {
		input    any
		expected any
	}{
		{"hello", "hello"},
		{42, 42},
		{int64(100), int64(100)},
		{3.14, 3.14},
		{true, true},
		{nil, nil},
	}

	for _, tt := range tests {
		result := convertToAny(tt.input)
		if result != tt.expected {
			t.Errorf("convertToAny(%v) = %v, want %v", tt.input, result, tt.expected)
		}
	}
}

func TestConvertToAny_MapStringAny(t *testing.T) {
	input := map[string]any{"name": "Alice", "age": 30}
	result := convertToAny(input)
	m, ok := result.(map[string]any)
	if !ok {
		t.Fatalf("expected map[string]any, got %T", result)
	}
	if m["name"] != "Alice" {
		t.Errorf("expected name=Alice, got %v", m["name"])
	}
}

func TestConvertToAny_MapStringString(t *testing.T) {
	input := map[string]string{"key": "value", "foo": "bar"}
	result := convertToAny(input)
	m, ok := result.(map[string]any)
	if !ok {
		t.Fatalf("expected map[string]any, got %T", result)
	}
	if m["key"] != "value" {
		t.Errorf("expected key=value, got %v", m["key"])
	}
	if m["foo"] != "bar" {
		t.Errorf("expected foo=bar, got %v", m["foo"])
	}
}

func TestConvertToAny_Struct(t *testing.T) {
	input := struct {
		Name string `json:"name"`
		Age  int    `json:"age"`
	}{Name: "Bob", Age: 25}

	result := convertToAny(input)
	m, ok := result.(map[string]any)
	if !ok {
		t.Fatalf("expected map[string]any from struct, got %T", result)
	}
	if m["name"] != "Bob" {
		t.Errorf("expected name=Bob, got %v", m["name"])
	}
}

func TestBuildGoJSONExtension_Session(t *testing.T) {
	bc := newTestContext()
	ext := BuildGoJSONExtension(bc)

	sessionFn := ext.Functions["session"].(func() any)
	sess := sessionFn()

	sessMap, ok := sess.(map[string]any)
	if !ok {
		t.Fatalf("session should return map[string]any, got %T", sess)
	}
	if sessMap["userId"] != "user-1" {
		t.Errorf("expected userId='user-1', got %v", sessMap["userId"])
	}
	if sessMap["username"] != "admin" {
		t.Errorf("expected username='admin', got %v", sessMap["username"])
	}
	if sessMap["email"] != "admin@test.com" {
		t.Errorf("expected email='admin@test.com', got %v", sessMap["email"])
	}
	if sessMap["tenantId"] != "t1" {
		t.Errorf("expected tenantId='t1', got %v", sessMap["tenantId"])
	}
}

func TestBuildGoJSONExtension_Crypto(t *testing.T) {
	bc := newTestContext()
	ext := BuildGoJSONExtension(bc)

	cryptoNS := ext.Functions["crypto"].(map[string]any)

	encryptFn := cryptoNS["encrypt"].(func(params ...any) (any, error))
	encrypted, err := encryptFn("secret")
	if err != nil {
		t.Fatalf("crypto.encrypt error: %v", err)
	}
	if encrypted != "enc:secret" {
		t.Errorf("expected 'enc:secret', got %v", encrypted)
	}

	hashFn := cryptoNS["hash"].(func(params ...any) (any, error))
	hashed, err := hashFn("password")
	if err != nil {
		t.Fatalf("crypto.hash error: %v", err)
	}
	if hashed != "hash:password" {
		t.Errorf("expected 'hash:password', got %v", hashed)
	}

	verifyFn := cryptoNS["verify"].(func(params ...any) (any, error))
	valid, err := verifyFn("password", "hash:password")
	if err != nil {
		t.Fatalf("crypto.verify error: %v", err)
	}
	if valid != true {
		t.Errorf("expected true, got %v", valid)
	}
}

func TestBuildGoJSONExtension_Execution(t *testing.T) {
	bc := newTestContext()
	ext := BuildGoJSONExtension(bc)

	execNS := ext.Functions["execution"].(map[string]any)

	currentFn := execNS["current"].(func(params ...any) (any, error))
	current, err := currentFn()
	if err != nil {
		t.Fatalf("execution.current error: %v", err)
	}
	if current == nil {
		t.Error("expected non-nil execution info")
	}
}

func TestBuildGoJSONExtension_NoTxFunction(t *testing.T) {
	bc := newTestContext()
	ext := BuildGoJSONExtension(bc)

	if _, ok := ext.Functions["tx"]; ok {
		t.Error("tx function should NOT be in extension (removed per 4.5c-fix Task 11)")
	}
}
