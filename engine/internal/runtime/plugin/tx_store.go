package plugin

import (
	"fmt"
	"sync"
	"time"

	"github.com/bitcode-framework/bitcode/internal/runtime/bridge"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type txEntry struct {
	txID      string
	gormTx    *gorm.DB
	bridgeCtx *bridge.Context
	doneCh    chan error
	createdAt time.Time
}

type txStore struct {
	mu      sync.Mutex
	entries map[string]*txEntry
}

func newTxStore() *txStore {
	return &txStore{
		entries: make(map[string]*txEntry),
	}
}

const defaultPluginTxTimeout = 30 * time.Second

func (s *txStore) Begin(parentCtx *bridge.Context, timeout ...time.Duration) (string, *bridge.Context, error) {
	db := parentCtx.GormDB()
	if db == nil {
		return "", nil, fmt.Errorf("database not available for transactions")
	}

	txID := uuid.New().String()
	doneCh := make(chan error, 1)

	gormTx := db.Begin()
	if gormTx.Error != nil {
		return "", nil, gormTx.Error
	}

	txCtx := parentCtx.CloneWithGormTx(gormTx)

	entry := &txEntry{
		txID:      txID,
		gormTx:    gormTx,
		bridgeCtx: txCtx,
		doneCh:    doneCh,
		createdAt: time.Now(),
	}

	s.mu.Lock()
	s.entries[txID] = entry
	s.mu.Unlock()

	txDuration := defaultPluginTxTimeout
	if len(timeout) > 0 {
		txDuration = timeout[0]
	}
	if txDuration > 0 {
		go s.watchTimeout(txID, txDuration)
	}

	return txID, txCtx, nil
}

func (s *txStore) GetContext(txID string) *bridge.Context {
	s.mu.Lock()
	defer s.mu.Unlock()
	if entry, ok := s.entries[txID]; ok {
		return entry.bridgeCtx
	}
	return nil
}

func (s *txStore) Commit(txID string) error {
	s.mu.Lock()
	entry, ok := s.entries[txID]
	if !ok {
		s.mu.Unlock()
		return fmt.Errorf("transaction %s not found", txID)
	}
	delete(s.entries, txID)
	s.mu.Unlock()

	return entry.gormTx.Commit().Error
}

func (s *txStore) Rollback(txID string) error {
	s.mu.Lock()
	entry, ok := s.entries[txID]
	if !ok {
		s.mu.Unlock()
		return nil
	}
	delete(s.entries, txID)
	s.mu.Unlock()

	return entry.gormTx.Rollback().Error
}

func (s *txStore) watchTimeout(txID string, timeout time.Duration) {
	time.Sleep(timeout)

	s.mu.Lock()
	entry, ok := s.entries[txID]
	if !ok {
		s.mu.Unlock()
		return
	}
	delete(s.entries, txID)
	s.mu.Unlock()

	entry.gormTx.Rollback()
}

func (s *txStore) CleanupAll() {
	s.mu.Lock()
	entries := make(map[string]*txEntry, len(s.entries))
	for k, v := range s.entries {
		entries[k] = v
	}
	s.entries = make(map[string]*txEntry)
	s.mu.Unlock()

	for _, entry := range entries {
		entry.gormTx.Rollback()
	}
}
