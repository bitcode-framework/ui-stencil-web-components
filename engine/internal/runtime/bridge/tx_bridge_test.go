package bridge

import (
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func setupTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: logger.Discard,
	})
	if err != nil {
		t.Fatalf("failed to open test db: %v", err)
	}
	return db
}

func TestTxBridge_BeginCommit(t *testing.T) {
	db := setupTestDB(t)
	txb := newTxBridge(db)

	if err := txb.Begin(); err != nil {
		t.Fatalf("Begin failed: %v", err)
	}
	if !txb.IsActive() {
		t.Fatal("expected active after Begin")
	}

	if err := txb.Commit(); err != nil {
		t.Fatalf("Commit failed: %v", err)
	}
	if txb.IsActive() {
		t.Fatal("expected inactive after Commit")
	}
}

func TestTxBridge_BeginRollback(t *testing.T) {
	db := setupTestDB(t)
	txb := newTxBridge(db)

	if err := txb.Begin(); err != nil {
		t.Fatalf("Begin failed: %v", err)
	}

	if err := txb.Rollback(); err != nil {
		t.Fatalf("Rollback failed: %v", err)
	}
	if txb.IsActive() {
		t.Fatal("expected inactive after Rollback")
	}
}

func TestTxBridge_DoubleBegin_Error(t *testing.T) {
	db := setupTestDB(t)
	txb := newTxBridge(db)

	txb.Begin()
	err := txb.Begin()
	if err == nil {
		t.Fatal("expected error on double Begin")
	}
	if err.Error() != "transaction already active — commit or rollback first" {
		t.Errorf("unexpected error: %v", err)
	}
	txb.Rollback()
}

func TestTxBridge_CommitWithoutBegin_Error(t *testing.T) {
	db := setupTestDB(t)
	txb := newTxBridge(db)

	err := txb.Commit()
	if err == nil {
		t.Fatal("expected error on Commit without Begin")
	}
	if err.Error() != "no active transaction to commit" {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestTxBridge_RollbackWithoutBegin_Error(t *testing.T) {
	db := setupTestDB(t)
	txb := newTxBridge(db)

	err := txb.Rollback()
	if err == nil {
		t.Fatal("expected error on Rollback without Begin")
	}
	if err.Error() != "no active transaction to rollback" {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestTxBridge_EffectiveDB_ReturnsNormalWhenNoTx(t *testing.T) {
	db := setupTestDB(t)
	txb := newTxBridge(db)

	effective := txb.EffectiveDB()
	if effective != db {
		t.Error("expected normal DB when no tx active")
	}
}

func TestTxBridge_EffectiveDB_ReturnsTxWhenActive(t *testing.T) {
	db := setupTestDB(t)
	txb := newTxBridge(db)

	txb.Begin()
	effective := txb.EffectiveDB()
	if effective == db {
		t.Error("expected tx handle, not raw db")
	}
	txb.Rollback()
}

func TestTxBridge_Cleanup_RollsBackDangling(t *testing.T) {
	db := setupTestDB(t)
	txb := newTxBridge(db)

	txb.Begin()
	if !txb.IsActive() {
		t.Fatal("expected active")
	}

	txb.Cleanup()
	if txb.IsActive() {
		t.Fatal("expected inactive after Cleanup")
	}
}

func TestTxBridge_Cleanup_NoopWhenInactive(t *testing.T) {
	db := setupTestDB(t)
	txb := newTxBridge(db)

	txb.Cleanup()
	if txb.IsActive() {
		t.Fatal("expected inactive")
	}
}

func TestTxBridge_Integration_CommitPersists(t *testing.T) {
	db := setupTestDB(t)
	db.Exec("CREATE TABLE accounts (id TEXT PRIMARY KEY, balance INTEGER)")
	db.Exec("INSERT INTO accounts (id, balance) VALUES ('acc-1', 1000)")

	txb := newTxBridge(db)
	txb.Begin()

	txb.EffectiveDB().Exec("UPDATE accounts SET balance = 900 WHERE id = 'acc-1'")
	txb.Commit()

	var balance int
	db.Raw("SELECT balance FROM accounts WHERE id = 'acc-1'").Scan(&balance)
	if balance != 900 {
		t.Errorf("expected 900 after commit, got %d", balance)
	}
}

func TestTxBridge_Integration_RollbackReverts(t *testing.T) {
	db := setupTestDB(t)
	db.Exec("CREATE TABLE accounts (id TEXT PRIMARY KEY, balance INTEGER)")
	db.Exec("INSERT INTO accounts (id, balance) VALUES ('acc-1', 1000)")

	txb := newTxBridge(db)
	txb.Begin()

	txb.EffectiveDB().Exec("UPDATE accounts SET balance = 500 WHERE id = 'acc-1'")
	txb.Rollback()

	var balance int
	db.Raw("SELECT balance FROM accounts WHERE id = 'acc-1'").Scan(&balance)
	if balance != 1000 {
		t.Errorf("expected 1000 after rollback, got %d", balance)
	}
}

func TestTxBridge_TimeoutAutoRollback(t *testing.T) {
	db := setupTestDB(t)
	db.Exec("CREATE TABLE items (id TEXT PRIMARY KEY, name TEXT)")
	db.Exec("INSERT INTO items (id, name) VALUES ('1', 'original')")

	txb := newTxBridge(db)
	if err := txb.BeginWithTimeout(50 * time.Millisecond); err != nil {
		t.Fatalf("BeginWithTimeout failed: %v", err)
	}

	txb.EffectiveDB().Exec("UPDATE items SET name = 'changed' WHERE id = '1'")

	time.Sleep(100 * time.Millisecond)

	if txb.IsActive() {
		t.Fatal("expected inactive after timeout auto-rollback")
	}

	var name string
	db.Raw("SELECT name FROM items WHERE id = '1'").Scan(&name)
	if name != "original" {
		t.Errorf("expected 'original' after auto-rollback, got %q", name)
	}
}

func TestTxBridge_ZeroTimeoutNoAutoRollback(t *testing.T) {
	db := setupTestDB(t)
	txb := newTxBridge(db)

	if err := txb.BeginWithTimeout(0); err != nil {
		t.Fatalf("BeginWithTimeout(0) failed: %v", err)
	}
	if !txb.IsActive() {
		t.Fatal("expected active")
	}

	time.Sleep(50 * time.Millisecond)

	if !txb.IsActive() {
		t.Fatal("expected still active with timeout=0")
	}
	txb.Rollback()
}

func TestTxBridge_DefaultBeginHasTimeout(t *testing.T) {
	db := setupTestDB(t)
	txb := newTxBridge(db)

	if err := txb.Begin(); err != nil {
		t.Fatalf("Begin failed: %v", err)
	}
	txb.mu.Lock()
	hasTimeout := txb.timeout != nil
	txb.mu.Unlock()

	if !hasTimeout {
		t.Fatal("expected timeout timer to be set with default Begin()")
	}
	txb.Rollback()
}

func TestParseTxTimeout(t *testing.T) {
	tests := []struct {
		name     string
		opts     map[string]any
		expected time.Duration
	}{
		{"nil opts", nil, defaultTxBridgeTimeout},
		{"empty opts", map[string]any{}, defaultTxBridgeTimeout},
		{"no timeout key", map[string]any{"foo": "bar"}, defaultTxBridgeTimeout},
		{"zero int", map[string]any{"timeout": 0}, 0},
		{"zero float", map[string]any{"timeout": float64(0)}, 0},
		{"positive int seconds", map[string]any{"timeout": 60}, 60 * time.Second},
		{"positive float seconds", map[string]any{"timeout": float64(120)}, 120 * time.Second},
		{"string duration", map[string]any{"timeout": "5m"}, 5 * time.Minute},
		{"string duration hours", map[string]any{"timeout": "1h"}, time.Hour},
		{"invalid string", map[string]any{"timeout": "invalid"}, defaultTxBridgeTimeout},
		{"negative int", map[string]any{"timeout": -1}, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseTxTimeout(tt.opts)
			if result != tt.expected {
				t.Errorf("parseTxTimeout(%v) = %v, want %v", tt.opts, result, tt.expected)
			}
		})
	}
}

func TestTxBridge_ReusableAfterCommit(t *testing.T) {
	db := setupTestDB(t)
	txb := newTxBridge(db)

	if err := txb.Begin(); err != nil {
		t.Fatalf("first Begin failed: %v", err)
	}
	if err := txb.Commit(); err != nil {
		t.Fatalf("first Commit failed: %v", err)
	}

	if err := txb.Begin(); err != nil {
		t.Fatalf("second Begin failed: %v", err)
	}
	if err := txb.Rollback(); err != nil {
		t.Fatalf("second Rollback failed: %v", err)
	}
}
