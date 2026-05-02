package io

import (
	"encoding/json"
	"fmt"
	"sync"
	"time"
)

type cacheEntry struct {
	value     any
	expiresAt time.Time // zero value = no expiry
}

// CacheModule provides in-memory key-value cache with TTL.
type CacheModule struct {
	mu       sync.RWMutex
	entries  map[string]cacheEntry
	stop     chan struct{}
	closed   bool
	security *CacheSecurityConfig
}

// CacheSecurityConfig controls cache resource limits.
type CacheSecurityConfig struct {
	MaxEntries   int   `json:"max_entries"`    // Max number of entries. Default: 10000. 0 = unlimited.
	MaxValueSize int64 `json:"max_value_size"` // Max size per value in bytes (JSON-serialized). Default: 1048576 (1MB). 0 = unlimited.
	MaxTTL       int   `json:"max_ttl"`        // Max TTL in seconds. Default: 86400 (24h). 0 = unlimited.
}

// NewCacheModule creates a new in-memory cache I/O module.
func NewCacheModule(security *SecurityConfig) *CacheModule {
	sec := &CacheSecurityConfig{
		MaxEntries:   10000,
		MaxValueSize: 1048576,
		MaxTTL:       86400,
	}
	if security != nil {
		cfg := &security.Cache
		if cfg.MaxEntries != 0 {
			sec.MaxEntries = cfg.MaxEntries
		}
		if cfg.MaxValueSize != 0 {
			sec.MaxValueSize = cfg.MaxValueSize
		}
		if cfg.MaxTTL != 0 {
			sec.MaxTTL = cfg.MaxTTL
		}
	}
	m := &CacheModule{
		entries:  make(map[string]cacheEntry),
		stop:     make(chan struct{}),
		security: sec,
	}
	go m.cleanupLoop()
	return m
}

func (m *CacheModule) Name() string { return "cache" }

func (m *CacheModule) Functions() map[string]any {
	return map[string]any{
		"get":   m.get,
		"set":   m.set,
		"del":   m.del,
		"has":   m.has,
		"clear": m.clear,
	}
}

func (m *CacheModule) SetConfig(cfg map[string]any) {}

// Close stops the cleanup goroutine and releases resources.
func (m *CacheModule) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.closed {
		return nil
	}
	m.closed = true
	close(m.stop)
	return nil
}

func (m *CacheModule) get(params ...any) (any, error) {
	if len(params) < 1 {
		return nil, fmt.Errorf("cache.get: key required")
	}
	key, ok := params[0].(string)
	if !ok {
		return nil, fmt.Errorf("cache.get: key must be a string")
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	entry, exists := m.entries[key]
	if !exists {
		return nil, nil
	}
	if !entry.expiresAt.IsZero() && time.Now().After(entry.expiresAt) {
		return nil, nil // expired — lazy eviction on read
	}
	return entry.value, nil
}

func (m *CacheModule) set(params ...any) (any, error) {
	if len(params) < 2 {
		return nil, fmt.Errorf("cache.set: key and value required")
	}
	key, ok := params[0].(string)
	if !ok {
		return nil, fmt.Errorf("cache.set: key must be a string")
	}
	value := params[1]

	if m.security.MaxValueSize > 0 {
		data, err := json.Marshal(value)
		if err == nil && int64(len(data)) > m.security.MaxValueSize {
			return nil, fmt.Errorf("cache.set: value size (%d bytes) exceeds limit (%d bytes)",
				len(data), m.security.MaxValueSize)
		}
	}

	entry := cacheEntry{value: value}
	if len(params) > 2 {
		if ttlRaw, ok := toFloat64Val(params[2]); ok && ttlRaw > 0 {
			ttl := int(ttlRaw)
			if m.security.MaxTTL > 0 && ttl > m.security.MaxTTL {
				ttl = m.security.MaxTTL
			}
			entry.expiresAt = time.Now().Add(time.Duration(ttl) * time.Second)
		}
	}

	m.mu.Lock()
	if m.security.MaxEntries > 0 {
		_, alreadyExists := m.entries[key]
		if !alreadyExists && len(m.entries) >= m.security.MaxEntries {
			m.mu.Unlock()
			return nil, fmt.Errorf("cache.set: max entries limit (%d) reached", m.security.MaxEntries)
		}
	}
	m.entries[key] = entry
	m.mu.Unlock()

	return nil, nil
}

func (m *CacheModule) del(params ...any) (any, error) {
	if len(params) < 1 {
		return nil, fmt.Errorf("cache.del: key required")
	}
	key, ok := params[0].(string)
	if !ok {
		return nil, fmt.Errorf("cache.del: key must be a string")
	}

	m.mu.Lock()
	delete(m.entries, key)
	m.mu.Unlock()

	return nil, nil
}

func (m *CacheModule) has(params ...any) (any, error) {
	if len(params) < 1 {
		return nil, fmt.Errorf("cache.has: key required")
	}
	key, ok := params[0].(string)
	if !ok {
		return nil, fmt.Errorf("cache.has: key must be a string")
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	entry, exists := m.entries[key]
	if !exists {
		return false, nil
	}
	if !entry.expiresAt.IsZero() && time.Now().After(entry.expiresAt) {
		return false, nil
	}
	return true, nil
}

func (m *CacheModule) clear(params ...any) (any, error) {
	m.mu.Lock()
	m.entries = make(map[string]cacheEntry)
	m.mu.Unlock()
	return nil, nil
}

func (m *CacheModule) cleanupLoop() {
	ticker := time.NewTicker(60 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			m.evictExpired()
		case <-m.stop:
			return
		}
	}
}

func (m *CacheModule) evictExpired() {
	m.mu.Lock()
	defer m.mu.Unlock()
	now := time.Now()
	for key, entry := range m.entries {
		if !entry.expiresAt.IsZero() && now.After(entry.expiresAt) {
			delete(m.entries, key)
		}
	}
}
