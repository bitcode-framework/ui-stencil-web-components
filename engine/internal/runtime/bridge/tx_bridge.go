package bridge

import (
	"fmt"
	"sync"
	"time"

	"gorm.io/gorm"
)

const defaultTxBridgeTimeout = 30 * time.Second

type txBridge struct {
	mu      sync.Mutex
	db      *gorm.DB
	gormTx  *gorm.DB
	active  bool
	timeout *time.Timer
}

func newTxBridge(db *gorm.DB) *txBridge {
	return &txBridge{db: db}
}

func (t *txBridge) EffectiveDB() *gorm.DB {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.active && t.gormTx != nil {
		return t.gormTx
	}
	return t.db
}

func (t *txBridge) Begin() error {
	return t.BeginWithTimeout(defaultTxBridgeTimeout)
}

// BeginWithTimeout starts a tx with explicit timeout. 0 = no timeout.
func (t *txBridge) BeginWithTimeout(timeout time.Duration) error {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.active {
		return fmt.Errorf("transaction already active — commit or rollback first")
	}
	gormTx := t.db.Begin()
	if gormTx.Error != nil {
		return fmt.Errorf("failed to begin transaction: %w", gormTx.Error)
	}
	t.gormTx = gormTx
	t.active = true
	if timeout > 0 {
		t.timeout = time.AfterFunc(timeout, t.autoRollback)
	}
	return nil
}

func (t *txBridge) Commit() error {
	t.mu.Lock()
	defer t.mu.Unlock()
	if !t.active {
		return fmt.Errorf("no active transaction to commit")
	}
	if t.timeout != nil {
		t.timeout.Stop()
		t.timeout = nil
	}
	err := t.gormTx.Commit().Error
	t.gormTx = nil
	t.active = false
	if err != nil {
		return fmt.Errorf("commit failed: %w", err)
	}
	return nil
}

func (t *txBridge) Rollback() error {
	t.mu.Lock()
	defer t.mu.Unlock()
	if !t.active {
		return fmt.Errorf("no active transaction to rollback")
	}
	if t.timeout != nil {
		t.timeout.Stop()
		t.timeout = nil
	}
	err := t.gormTx.Rollback().Error
	t.gormTx = nil
	t.active = false
	if err != nil {
		return fmt.Errorf("rollback failed: %w", err)
	}
	return nil
}

func (t *txBridge) IsActive() bool {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.active
}

// Cleanup auto-rollbacks any dangling transaction.
// Called by Context.Cleanup() when the program execution ends.
func (t *txBridge) Cleanup() {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.active && t.gormTx != nil {
		if t.timeout != nil {
			t.timeout.Stop()
			t.timeout = nil
		}
		t.gormTx.Rollback()
		t.gormTx = nil
		t.active = false
	}
}

func (t *txBridge) autoRollback() {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.active && t.gormTx != nil {
		t.gormTx.Rollback()
		t.gormTx = nil
		t.active = false
		t.timeout = nil
	}
}

// parseTxTimeout extracts timeout from optional params map.
// Returns defaultTxBridgeTimeout if not specified.
// Accepts: int/float64 (seconds), string (duration like "5m"), 0 (infinite).
func parseTxTimeout(opts map[string]any) time.Duration {
	if opts == nil {
		return defaultTxBridgeTimeout
	}
	raw, ok := opts["timeout"]
	if !ok {
		return defaultTxBridgeTimeout
	}
	switch v := raw.(type) {
	case float64:
		if v <= 0 {
			return 0
		}
		return time.Duration(v) * time.Second
	case int:
		if v <= 0 {
			return 0
		}
		return time.Duration(v) * time.Second
	case string:
		d, err := time.ParseDuration(v)
		if err != nil || d < 0 {
			return defaultTxBridgeTimeout
		}
		return d
	default:
		return defaultTxBridgeTimeout
	}
}
