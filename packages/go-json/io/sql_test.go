package io

import (
	"strings"
	"testing"

	_ "modernc.org/sqlite"
)

func testSecurityConfig() *SecurityConfig {
	security := DefaultSecurityConfig()
	security.SQL.AllowedDrivers = []string{"sqlite"}
	security.SQL.DefaultDSN = "file::memory:?cache=shared"
	security.SQL.BlockedKeywords = []string{}
	return security
}

func TestSQLModule_Query(t *testing.T) {
	security := testSecurityConfig()

	m := NewSQLModule(security)
	defer m.Close()

	// Create table
	_, err := m.sqlExecute("CREATE TABLE users (id INTEGER PRIMARY KEY, name TEXT, age INTEGER)")
	if err != nil {
		t.Fatalf("CREATE TABLE failed: %v", err)
	}

	// Insert data
	_, err = m.sqlExecute("INSERT INTO users (name, age) VALUES (?, ?)", []any{"Alice", 30})
	if err != nil {
		t.Fatalf("INSERT failed: %v", err)
	}

	// Query
	result, err := m.sqlQuery("SELECT * FROM users WHERE name = ?", []any{"Alice"})
	if err != nil {
		t.Fatalf("SELECT failed: %v", err)
	}

	resultMap, ok := result.(map[string]any)
	if !ok {
		t.Fatalf("Expected map result, got %T", result)
	}

	rows, ok := resultMap["rows"].([]any)
	if !ok {
		t.Fatalf("Expected rows array, got %T", resultMap["rows"])
	}

	if len(rows) != 1 {
		t.Fatalf("Expected 1 row, got %d", len(rows))
	}

	row := rows[0].(map[string]any)
	if row["name"] != "Alice" {
		t.Errorf("Expected name=Alice, got %v", row["name"])
	}

	count := resultMap["count"].(int)
	if count != 1 {
		t.Errorf("Expected count=1, got %d", count)
	}
}

func TestSQLModule_QueryEmpty(t *testing.T) {
	security := testSecurityConfig()

	m := NewSQLModule(security)
	defer m.Close()

	// Create table
	_, err := m.sqlExecute("CREATE TABLE empty_table (id INTEGER PRIMARY KEY)")
	if err != nil {
		t.Fatalf("CREATE TABLE failed: %v", err)
	}

	// Query empty table
	result, err := m.sqlQuery("SELECT * FROM empty_table")
	if err != nil {
		t.Fatalf("SELECT failed: %v", err)
	}

	resultMap := result.(map[string]any)
	rows := resultMap["rows"].([]any)

	if len(rows) != 0 {
		t.Errorf("Expected 0 rows, got %d", len(rows))
	}

	count := resultMap["count"].(int)
	if count != 0 {
		t.Errorf("Expected count=0, got %d", count)
	}

	columns := resultMap["columns"].([]any)
	if len(columns) != 1 || columns[0] != "id" {
		t.Errorf("Expected columns=[id], got %v", columns)
	}
}

func TestSQLModule_Execute(t *testing.T) {
	security := testSecurityConfig()

	m := NewSQLModule(security)
	defer m.Close()

	// Create table
	_, err := m.sqlExecute("CREATE TABLE products (id INTEGER PRIMARY KEY, name TEXT, price REAL)")
	if err != nil {
		t.Fatalf("CREATE TABLE failed: %v", err)
	}

	// INSERT
	result, err := m.sqlExecute("INSERT INTO products (name, price) VALUES (?, ?)", []any{"Widget", 19.99})
	if err != nil {
		t.Fatalf("INSERT failed: %v", err)
	}

	resultMap := result.(map[string]any)
	rowsAffected := resultMap["rows_affected"].(int64)
	lastInsertID := resultMap["last_insert_id"].(int64)

	if rowsAffected != 1 {
		t.Errorf("Expected rows_affected=1, got %d", rowsAffected)
	}
	if lastInsertID != 1 {
		t.Errorf("Expected last_insert_id=1, got %d", lastInsertID)
	}

	// UPDATE
	result, err = m.sqlExecute("UPDATE products SET price = ? WHERE name = ?", []any{24.99, "Widget"})
	if err != nil {
		t.Fatalf("UPDATE failed: %v", err)
	}

	resultMap = result.(map[string]any)
	rowsAffected = resultMap["rows_affected"].(int64)

	if rowsAffected != 1 {
		t.Errorf("Expected rows_affected=1, got %d", rowsAffected)
	}
}

func TestSQLModule_Transaction_Commit(t *testing.T) {
	security := testSecurityConfig()

	m := NewSQLModule(security)
	defer m.Close()

	// Create table
	_, err := m.sqlExecute("CREATE TABLE accounts (id INTEGER PRIMARY KEY, balance REAL)")
	if err != nil {
		t.Fatalf("CREATE TABLE failed: %v", err)
	}

	// Begin transaction
	_, err = m.sqlBegin()
	if err != nil {
		t.Fatalf("BEGIN failed: %v", err)
	}

	// Insert within transaction
	_, err = m.sqlExecute("INSERT INTO accounts (balance) VALUES (?)", []any{100.0})
	if err != nil {
		t.Fatalf("INSERT failed: %v", err)
	}

	// Commit
	_, err = m.sqlCommit()
	if err != nil {
		t.Fatalf("COMMIT failed: %v", err)
	}

	// Verify data persisted
	result, err := m.sqlQuery("SELECT COUNT(*) as cnt FROM accounts")
	if err != nil {
		t.Fatalf("SELECT failed: %v", err)
	}

	resultMap := result.(map[string]any)
	rows := resultMap["rows"].([]any)
	row := rows[0].(map[string]any)

	if row["cnt"].(int64) != 1 {
		t.Errorf("Expected count=1, got %v", row["cnt"])
	}
}

func TestSQLModule_Transaction_Rollback(t *testing.T) {
	security := testSecurityConfig()

	m := NewSQLModule(security)
	defer m.Close()

	// Create table
	_, err := m.sqlExecute("CREATE TABLE logs (id INTEGER PRIMARY KEY, message TEXT)")
	if err != nil {
		t.Fatalf("CREATE TABLE failed: %v", err)
	}

	// Begin transaction
	_, err = m.sqlBegin()
	if err != nil {
		t.Fatalf("BEGIN failed: %v", err)
	}

	// Insert within transaction
	_, err = m.sqlExecute("INSERT INTO logs (message) VALUES (?)", []any{"test"})
	if err != nil {
		t.Fatalf("INSERT failed: %v", err)
	}

	// Rollback
	_, err = m.sqlRollback()
	if err != nil {
		t.Fatalf("ROLLBACK failed: %v", err)
	}

	// Verify data NOT persisted
	result, err := m.sqlQuery("SELECT COUNT(*) as cnt FROM logs")
	if err != nil {
		t.Fatalf("SELECT failed: %v", err)
	}

	resultMap := result.(map[string]any)
	rows := resultMap["rows"].([]any)
	row := rows[0].(map[string]any)

	if row["cnt"].(int64) != 0 {
		t.Errorf("Expected count=0 after rollback, got %v", row["cnt"])
	}
}

func TestSQLModule_Transaction_Nested(t *testing.T) {
	security := testSecurityConfig()

	m := NewSQLModule(security)
	defer m.Close()

	// Create table
	_, err := m.sqlExecute("CREATE TABLE nested (id INTEGER PRIMARY KEY, value TEXT)")
	if err != nil {
		t.Fatalf("CREATE TABLE failed: %v", err)
	}

	// Begin outer transaction
	_, err = m.sqlBegin()
	if err != nil {
		t.Fatalf("BEGIN outer failed: %v", err)
	}

	// Insert in outer
	_, err = m.sqlExecute("INSERT INTO nested (value) VALUES (?)", []any{"outer"})
	if err != nil {
		t.Fatalf("INSERT outer failed: %v", err)
	}

	// Begin nested transaction (savepoint)
	_, err = m.sqlBegin()
	if err != nil {
		t.Fatalf("BEGIN nested failed: %v", err)
	}

	// Insert in nested
	_, err = m.sqlExecute("INSERT INTO nested (value) VALUES (?)", []any{"nested"})
	if err != nil {
		t.Fatalf("INSERT nested failed: %v", err)
	}

	// Commit nested (release savepoint)
	_, err = m.sqlCommit()
	if err != nil {
		t.Fatalf("COMMIT nested failed: %v", err)
	}

	// Commit outer
	_, err = m.sqlCommit()
	if err != nil {
		t.Fatalf("COMMIT outer failed: %v", err)
	}

	// Verify both rows persisted
	result, err := m.sqlQuery("SELECT COUNT(*) as cnt FROM nested")
	if err != nil {
		t.Fatalf("SELECT failed: %v", err)
	}

	resultMap := result.(map[string]any)
	rows := resultMap["rows"].([]any)
	row := rows[0].(map[string]any)

	if row["cnt"].(int64) != 2 {
		t.Errorf("Expected count=2, got %v", row["cnt"])
	}
}

func TestSQLModule_PoolReuse(t *testing.T) {
	security := testSecurityConfig()
	dsn := "file::memory:?cache=shared"

	m := NewSQLModule(security)
	defer m.Close()

	// First query creates pool
	_, err := m.sqlQuery("SELECT 1", nil, dsn)
	if err != nil {
		t.Fatalf("First query failed: %v", err)
	}

	// Second query should reuse pool
	_, err = m.sqlQuery("SELECT 2", nil, dsn)
	if err != nil {
		t.Fatalf("Second query failed: %v", err)
	}

	// Verify only one pool created
	m.poolsMu.Lock()
	poolCount := len(m.pools)
	m.poolsMu.Unlock()

	if poolCount != 1 {
		t.Errorf("Expected 1 pool, got %d", poolCount)
	}
}

func TestSQLModule_PoolLimit(t *testing.T) {
	security := testSecurityConfig()
	security.SQL.MaxPools = 2

	m := NewSQLModule(security)
	defer m.Close()

	// Create 2 pools (should succeed)
	_, err := m.sqlQuery("SELECT 1", nil, "file::memory:?cache=shared&_db1")
	if err != nil {
		t.Fatalf("First pool failed: %v", err)
	}

	_, err = m.sqlQuery("SELECT 1", nil, "file::memory:?cache=shared&_db2")
	if err != nil {
		t.Fatalf("Second pool failed: %v", err)
	}

	// Third pool should fail
	_, err = m.sqlQuery("SELECT 1", nil, "file::memory:?cache=shared&_db3")
	if err == nil {
		t.Fatal("Expected error when exceeding MaxPools, got nil")
	}

	if err.Error() != "sql: max pool limit reached (2)" {
		t.Errorf("Expected pool limit error, got: %v", err)
	}
}

func TestSQLModule_Close(t *testing.T) {
	security := testSecurityConfig()

	m := NewSQLModule(security)

	// Create table and start transaction
	_, err := m.sqlExecute("CREATE TABLE test (id INTEGER PRIMARY KEY)")
	if err != nil {
		t.Fatalf("CREATE TABLE failed: %v", err)
	}

	_, err = m.sqlBegin()
	if err != nil {
		t.Fatalf("BEGIN failed: %v", err)
	}

	// Close should rollback transaction and close pools
	err = m.Close()
	if err != nil {
		t.Fatalf("Close failed: %v", err)
	}

	// Verify pools closed
	m.poolsMu.Lock()
	poolCount := len(m.pools)
	m.poolsMu.Unlock()

	if poolCount != 0 {
		t.Errorf("Expected 0 pools after Close, got %d", poolCount)
	}

	// Verify transaction rolled back
	m.mu.Lock()
	hasTx := m.tx != nil
	m.mu.Unlock()

	if hasTx {
		t.Error("Expected transaction to be nil after Close")
	}
}

func TestSQLModule_Cleanup(t *testing.T) {
	security := testSecurityConfig()

	m := NewSQLModule(security)
	defer m.Close()

	// Create table and start transaction
	_, err := m.sqlExecute("CREATE TABLE test (id INTEGER PRIMARY KEY)")
	if err != nil {
		t.Fatalf("CREATE TABLE failed: %v", err)
	}

	_, err = m.sqlBegin()
	if err != nil {
		t.Fatalf("BEGIN failed: %v", err)
	}

	// Cleanup should rollback transaction but NOT close pools
	m.Cleanup()

	// Verify transaction rolled back
	m.mu.Lock()
	hasTx := m.tx != nil
	m.mu.Unlock()

	if hasTx {
		t.Error("Expected transaction to be nil after Cleanup")
	}

	// Verify pools still open
	m.poolsMu.Lock()
	poolCount := len(m.pools)
	m.poolsMu.Unlock()

	if poolCount == 0 {
		t.Error("Expected pools to remain open after Cleanup")
	}
}

func TestSQLModule_DDLProtection(t *testing.T) {
	security := testSecurityConfig()
	security.SQL.BlockedKeywords = []string{"DROP", "TRUNCATE", "ALTER"}

	m := NewSQLModule(security)
	defer m.Close()

	_, err := m.sqlExecute("CREATE TABLE test (id INTEGER PRIMARY KEY)")
	if err != nil {
		t.Fatalf("CREATE TABLE failed: %v", err)
	}

	// DROP TABLE should be blocked
	_, err = m.sqlExecute("DROP TABLE test")
	if err == nil {
		t.Fatal("Expected error for DROP TABLE, got nil")
	}

	if err.Error() != "sql: query contains blocked keyword 'DROP'" {
		t.Errorf("Expected DROP blocked error, got: %v", err)
	}

	// TRUNCATE should be blocked
	_, err = m.sqlExecute("TRUNCATE TABLE test")
	if err == nil {
		t.Fatal("Expected error for TRUNCATE, got nil")
	}

	if err.Error() != "sql: query contains blocked keyword 'TRUNCATE'" {
		t.Errorf("Expected TRUNCATE blocked error, got: %v", err)
	}
}

func TestSQLModule_MaxQueryLength(t *testing.T) {
	security := DefaultSecurityConfig()
	security.SQL.AllowedDrivers = []string{"sqlite"}
	security.SQL.DefaultDSN = "file::memory:?cache=shared"
	security.SQL.MaxQueryLength = 50

	m := NewSQLModule(security)
	defer m.Close()

	// Query exceeding max length
	longQuery := "SELECT * FROM users WHERE name = 'this is a very long query that exceeds the limit'"
	_, err := m.sqlQuery(longQuery)

	if err == nil {
		t.Fatal("Expected error for query exceeding max length, got nil")
	}

	if !strings.Contains(err.Error(), "query exceeds max length") {
		t.Errorf("Expected max length error, got: %v", err)
	}
}

func TestSQLModule_ParameterizedQueries(t *testing.T) {
	security := testSecurityConfig()

	m := NewSQLModule(security)
	defer m.Close()

	// Create table
	_, err := m.sqlExecute("CREATE TABLE users (id INTEGER PRIMARY KEY, name TEXT, email TEXT)")
	if err != nil {
		t.Fatalf("CREATE TABLE failed: %v", err)
	}

	// Insert with positional parameters
	_, err = m.sqlExecute("INSERT INTO users (name, email) VALUES (?, ?)", []any{"Alice", "alice@example.com"})
	if err != nil {
		t.Fatalf("INSERT failed: %v", err)
	}

	// Query with positional parameters
	result, err := m.sqlQuery("SELECT * FROM users WHERE name = ? AND email = ?", []any{"Alice", "alice@example.com"})
	if err != nil {
		t.Fatalf("SELECT failed: %v", err)
	}

	resultMap := result.(map[string]any)
	rows := resultMap["rows"].([]any)

	if len(rows) != 1 {
		t.Fatalf("Expected 1 row, got %d", len(rows))
	}

	row := rows[0].(map[string]any)
	if row["name"] != "Alice" || row["email"] != "alice@example.com" {
		t.Errorf("Expected Alice/alice@example.com, got %v/%v", row["name"], row["email"])
	}
}

func TestSQLModule_NullHandling(t *testing.T) {
	security := testSecurityConfig()

	m := NewSQLModule(security)
	defer m.Close()

	// Create table with nullable column
	_, err := m.sqlExecute("CREATE TABLE nullable (id INTEGER PRIMARY KEY, value TEXT)")
	if err != nil {
		t.Fatalf("CREATE TABLE failed: %v", err)
	}

	// Insert NULL value
	_, err = m.sqlExecute("INSERT INTO nullable (value) VALUES (NULL)")
	if err != nil {
		t.Fatalf("INSERT failed: %v", err)
	}

	// Query
	result, err := m.sqlQuery("SELECT * FROM nullable")
	if err != nil {
		t.Fatalf("SELECT failed: %v", err)
	}

	resultMap := result.(map[string]any)
	rows := resultMap["rows"].([]any)
	row := rows[0].(map[string]any)

	// NULL should be represented as nil
	if row["value"] != nil {
		t.Errorf("Expected nil for NULL value, got %v", row["value"])
	}
}

