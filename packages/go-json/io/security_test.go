package io

import (
	"testing"
)

func TestValidateHTTPRequest_BlockedHosts(t *testing.T) {
	sc := DefaultSecurityConfig()

	tests := []struct {
		url     string
		blocked bool
	}{
		{"http://localhost:8080/api", true},
		{"http://127.0.0.1:3000/data", true},
		{"http://[::1]/api", true},
		{"http://0.0.0.0/api", true},
		{"http://169.254.169.254/metadata", true},
		{"http://169.254.169.253/metadata", true},
		{"https://api.example.com/users", false},
	}

	for _, tt := range tests {
		err := sc.ValidateHTTPRequest(tt.url)
		if tt.blocked && err == nil {
			t.Errorf("expected %s to be blocked", tt.url)
		}
		if !tt.blocked && err != nil {
			t.Errorf("expected %s to be allowed, got: %s", tt.url, err.Error())
		}
	}
}

func TestValidateHTTPRequest_AllowedHosts(t *testing.T) {
	sc := DefaultSecurityConfig()
	sc.HTTP.AllowedHosts = []string{"api.example.com", "*.internal.com"}

	tests := []struct {
		url     string
		allowed bool
	}{
		{"https://api.example.com/users", true},
		{"https://service.internal.com/data", true},
		{"https://other.com/api", false},
	}

	for _, tt := range tests {
		err := sc.ValidateHTTPRequest(tt.url)
		if tt.allowed && err != nil {
			t.Errorf("expected %s to be allowed, got: %s", tt.url, err.Error())
		}
		if !tt.allowed && err == nil {
			t.Errorf("expected %s to be blocked", tt.url)
		}
	}
}

func TestValidateHTTPRequest_WildcardHost(t *testing.T) {
	sc := DefaultSecurityConfig()
	sc.HTTP.AllowedHosts = []string{"*.internal.com"}

	err := sc.ValidateHTTPRequest("https://api.internal.com/data")
	if err != nil {
		t.Errorf("wildcard should match: %s", err.Error())
	}

	err = sc.ValidateHTTPRequest("https://internal.com/data")
	if err == nil {
		t.Error("wildcard *.internal.com should not match internal.com")
	}
}

func TestValidateFilePath_PathTraversal(t *testing.T) {
	sc := DefaultSecurityConfig()
	sc.FS.AllowedPaths = []string{"/tmp/go-json/"}

	err := sc.ValidateFilePath("../../etc/passwd", false)
	if err == nil {
		t.Error("path traversal should be blocked")
	}
}

func TestValidateFilePath_WriteDisabled(t *testing.T) {
	sc := DefaultSecurityConfig()
	sc.FS.AllowWrite = false

	err := sc.ValidateFilePath("/tmp/test.txt", true)
	if err == nil {
		t.Error("write should be blocked when AllowWrite is false")
	}
}

func TestValidateFilePath_WriteEnabled(t *testing.T) {
	sc := DefaultSecurityConfig()
	sc.FS.AllowWrite = true
	sc.FS.AllowedPaths = nil
	sc.FS.BlockedPaths = nil

	err := sc.ValidateFilePath("/tmp/test.txt", true)
	if err != nil {
		t.Errorf("write should be allowed: %s", err.Error())
	}
}

func TestValidateCommand_DeniedCommands(t *testing.T) {
	sc := DefaultSecurityConfig()
	sc.Exec.AllowedCommands = []string{"rm", "echo"}

	err := sc.ValidateCommand("rm")
	if err == nil {
		t.Error("rm should be permanently blocked even if in AllowedCommands")
	}
}

func TestValidateCommand_AllowedCommands(t *testing.T) {
	sc := DefaultSecurityConfig()
	sc.Exec.AllowedCommands = []string{"echo", "pandoc"}

	err := sc.ValidateCommand("echo")
	if err != nil {
		t.Errorf("echo should be allowed: %s", err.Error())
	}

	err = sc.ValidateCommand("curl")
	if err == nil {
		t.Error("curl should not be allowed")
	}
}

func TestValidateCommand_NoWhitelist(t *testing.T) {
	sc := DefaultSecurityConfig()
	sc.Exec.AllowedCommands = nil

	err := sc.ValidateCommand("echo")
	if err == nil {
		t.Error("should fail when no commands are whitelisted")
	}
}

func TestValidateSQLDriver(t *testing.T) {
	sc := DefaultSecurityConfig()
	sc.SQL.AllowedDrivers = []string{"sqlite3", "postgres"}

	err := sc.ValidateSQLDriver("sqlite3")
	if err != nil {
		t.Errorf("sqlite3 should be allowed: %s", err.Error())
	}

	err = sc.ValidateSQLDriver("mysql")
	if err == nil {
		t.Error("mysql should not be allowed")
	}
}

func TestValidateSQLDriver_NoRestriction(t *testing.T) {
	sc := DefaultSecurityConfig()
	sc.SQL.AllowedDrivers = nil

	err := sc.ValidateSQLDriver("anything")
	if err != nil {
		t.Errorf("should allow any driver when no restriction: %s", err.Error())
	}
}

func TestStripEngineSecrets(t *testing.T) {
	env := map[string]string{
		"PATH":                   "/usr/bin",
		"HOME":                   "/home/user",
		"JWT_SECRET":             "secret123",
		"DB_PASSWORD":            "pass123",
		"ENCRYPTION_KEY":         "key123",
		"SMTP_PASSWORD":          "smtp123",
		"STORAGE_S3_SECRET_KEY":  "s3secret",
		"STORAGE_S3_ACCESS_KEY":  "s3access",
		"CUSTOM_VAR":             "value",
	}

	result := StripEngineSecrets(env)

	if _, ok := result["JWT_SECRET"]; ok {
		t.Error("JWT_SECRET should be stripped")
	}
	if _, ok := result["DB_PASSWORD"]; ok {
		t.Error("DB_PASSWORD should be stripped")
	}
	if _, ok := result["PATH"]; !ok {
		t.Error("PATH should be preserved")
	}
	if _, ok := result["CUSTOM_VAR"]; !ok {
		t.Error("CUSTOM_VAR should be preserved")
	}
	if len(result) != 3 {
		t.Errorf("expected 3 remaining vars, got %d", len(result))
	}
}

func TestIsModuleEnabled(t *testing.T) {
	sc := &SecurityConfig{EnabledModules: []string{"http", "fs"}}

	if !sc.IsModuleEnabled("http") {
		t.Error("http should be enabled")
	}
	if !sc.IsModuleEnabled("fs") {
		t.Error("fs should be enabled")
	}
	if sc.IsModuleEnabled("sql") {
		t.Error("sql should not be enabled")
	}
}

func TestMatchHost(t *testing.T) {
	tests := []struct {
		hostname string
		pattern  string
		expected bool
	}{
		{"api.example.com", "api.example.com", true},
		{"api.example.com", "*.example.com", true},
		{"example.com", "*.example.com", false},
		{"other.com", "api.example.com", false},
	}

	for _, tt := range tests {
		result := matchHost(tt.hostname, tt.pattern)
		if result != tt.expected {
			t.Errorf("matchHost(%q, %q) = %v, want %v", tt.hostname, tt.pattern, result, tt.expected)
		}
	}
}

func TestIsPathBlocked(t *testing.T) {
	tests := []struct {
		path    string
		blocked []string
		want    bool
	}{
		{"/etc/passwd", []string{"/etc/"}, true},
		{"/tmp/test.txt", []string{"/etc/"}, false},
		{"/etc", []string{"/etc/"}, true},
	}

	for _, tt := range tests {
		got := isPathBlocked(tt.path, tt.blocked)
		if got != tt.want {
			t.Errorf("isPathBlocked(%q, %v) = %v, want %v", tt.path, tt.blocked, got, tt.want)
		}
	}
}

func TestSQLValidateQuery_BlockedKeywords(t *testing.T) {
	m := NewSQLModule(DefaultSecurityConfig())

	tests := []struct {
		query   string
		blocked bool
	}{
		{"SELECT * FROM users", false},
		{"INSERT INTO users (name) VALUES (?)", false},
		{"DROP TABLE users", true},
		{"TRUNCATE TABLE users", true},
		{"ALTER TABLE users ADD COLUMN age INT", true},
	}

	for _, tt := range tests {
		err := m.validateQuery(tt.query)
		if tt.blocked && err == nil {
			t.Errorf("expected query to be blocked: %s", tt.query)
		}
		if !tt.blocked && err != nil {
			t.Errorf("expected query to be allowed: %s, got: %s", tt.query, err.Error())
		}
	}
}

func TestSQLValidateQuery_MaxLength(t *testing.T) {
	sec := DefaultSecurityConfig()
	sec.SQL.MaxQueryLength = 50
	m := NewSQLModule(sec)

	shortQuery := "SELECT * FROM users"
	if err := m.validateQuery(shortQuery); err != nil {
		t.Errorf("short query should pass: %s", err.Error())
	}

	longQuery := "SELECT * FROM users WHERE name = 'very long query that exceeds the limit'"
	if err := m.validateQuery(longQuery); err == nil {
		t.Error("long query should be blocked")
	}
}

func TestDetectDriverFromDSN(t *testing.T) {
	tests := []struct {
		dsn    string
		driver string
	}{
		{"postgres://localhost/db", "postgres"},
		{"postgresql://localhost/db", "postgres"},
		{"mysql://localhost/db", "mysql"},
		{"sqlserver://localhost/db", "sqlserver"},
		{"oracle://localhost/db", "oracle"},
		{"sqlite3://test.db", "sqlite3"},
		{"sqlite://test.db", "sqlite3"},
		{"file:test.db", "sqlite3"},
		{"test.db", "sqlite3"},
	}

	for _, tt := range tests {
		got := detectDriverFromDSN(tt.dsn)
		if got != tt.driver {
			t.Errorf("detectDriverFromDSN(%q) = %q, want %q", tt.dsn, got, tt.driver)
		}
	}
}

func TestMongoValidateOperation_BlockedOps(t *testing.T) {
	m := NewMongoModule(DefaultSecurityConfig())

	filter := map[string]any{
		"$where": "this.age > 18",
	}
	if err := m.validateOperation(filter); err == nil {
		t.Error("$where should be blocked")
	}

	safeFilter := map[string]any{
		"age": map[string]any{"$gt": 18},
	}
	if err := m.validateOperation(safeFilter); err != nil {
		t.Errorf("safe filter should pass: %s", err.Error())
	}
}

func TestRedisValidateKey_MaxLength(t *testing.T) {
	sec := DefaultSecurityConfig()
	sec.Redis.MaxKeyLength = 10
	m := NewRedisModule(sec)

	if err := m.validateKey("short"); err != nil {
		t.Errorf("short key should pass: %s", err.Error())
	}

	if err := m.validateKey("this-is-a-very-long-key"); err == nil {
		t.Error("long key should be blocked")
	}
}

func TestRedisValidateCommand_Blocked(t *testing.T) {
	m := NewRedisModule(DefaultSecurityConfig())

	if err := m.validateCommand("GET"); err != nil {
		t.Errorf("GET should be allowed: %s", err.Error())
	}

	if err := m.validateCommand("FLUSHALL"); err == nil {
		t.Error("FLUSHALL should be blocked")
	}

	if err := m.validateCommand("CONFIG"); err == nil {
		t.Error("CONFIG should be blocked")
	}
}

func TestRedisPrefixKey(t *testing.T) {
	sec := DefaultSecurityConfig()
	sec.Redis.KeyPrefix = "tenant1:"
	m := NewRedisModule(sec)

	if got := m.prefixKey("user:1"); got != "tenant1:user:1" {
		t.Errorf("expected 'tenant1:user:1', got '%s'", got)
	}

	m2 := NewRedisModule(DefaultSecurityConfig())
	if got := m2.prefixKey("user:1"); got != "user:1" {
		t.Errorf("expected 'user:1' without prefix, got '%s'", got)
	}
}

func TestRedisAutoSerialize(t *testing.T) {
	m := NewRedisModule(DefaultSecurityConfig())

	s, err := m.autoSerialize("hello")
	if err != nil || s != "hello" {
		t.Errorf("string should pass through: %s, %v", s, err)
	}

	s, err = m.autoSerialize(map[string]any{"key": "value"})
	if err != nil {
		t.Errorf("map should serialize: %v", err)
	}
	if s != `{"key":"value"}` {
		t.Errorf("expected JSON, got: %s", s)
	}
}

func TestRedisAutoDeserialize(t *testing.T) {
	m := NewRedisModule(DefaultSecurityConfig())

	result := m.autoDeserialize(`{"key":"value"}`)
	if m, ok := result.(map[string]any); !ok || m["key"] != "value" {
		t.Errorf("expected map, got: %v", result)
	}

	result = m.autoDeserialize("plain string")
	if result != "plain string" {
		t.Errorf("expected plain string, got: %v", result)
	}
}
