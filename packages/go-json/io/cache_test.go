package io

import (
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestCache_SetGet(t *testing.T) {
	m := NewCacheModule(nil)
	defer m.Close()

	_, err := m.set("key1", "hello")
	assert.NoError(t, err)

	val, err := m.get("key1")
	assert.NoError(t, err)
	assert.Equal(t, "hello", val)
}

func TestCache_SetGet_Map(t *testing.T) {
	m := NewCacheModule(nil)
	defer m.Close()

	data := map[string]any{"name": "Alice", "age": 30}
	_, err := m.set("user", data)
	assert.NoError(t, err)

	val, err := m.get("user")
	assert.NoError(t, err)
	assert.Equal(t, data, val)
}

func TestCache_Get_Missing(t *testing.T) {
	m := NewCacheModule(nil)
	defer m.Close()

	val, err := m.get("nonexistent")
	assert.NoError(t, err)
	assert.Nil(t, val)
}

func TestCache_SetGet_NilValue(t *testing.T) {
	m := NewCacheModule(nil)
	defer m.Close()

	_, err := m.set("nilkey", nil)
	assert.NoError(t, err)

	val, err := m.get("nilkey")
	assert.NoError(t, err)
	assert.Nil(t, val)

	has, err := m.has("nilkey")
	assert.NoError(t, err)
	assert.Equal(t, true, has)
}

func TestCache_TTL_Expiry(t *testing.T) {
	m := NewCacheModule(nil)
	defer m.Close()

	_, err := m.set("key1", "hello", 1)
	assert.NoError(t, err)

	val, _ := m.get("key1")
	assert.Equal(t, "hello", val)

	time.Sleep(1100 * time.Millisecond)

	val, _ = m.get("key1")
	assert.Nil(t, val)
}

func TestCache_TTL_NoExpiry(t *testing.T) {
	m := NewCacheModule(nil)
	defer m.Close()

	_, err := m.set("key1", "hello")
	assert.NoError(t, err)

	time.Sleep(100 * time.Millisecond)
	val, _ := m.get("key1")
	assert.Equal(t, "hello", val)
}

func TestCache_TTL_ZeroMeansNoExpiry(t *testing.T) {
	m := NewCacheModule(nil)
	defer m.Close()

	_, err := m.set("key1", "hello", 0)
	assert.NoError(t, err)

	time.Sleep(100 * time.Millisecond)
	val, _ := m.get("key1")
	assert.Equal(t, "hello", val)
}

func TestCache_TTL_NegativeMeansNoExpiry(t *testing.T) {
	m := NewCacheModule(nil)
	defer m.Close()

	_, err := m.set("key1", "hello", -5)
	assert.NoError(t, err)

	time.Sleep(100 * time.Millisecond)
	val, _ := m.get("key1")
	assert.Equal(t, "hello", val)
}

func TestCache_Has(t *testing.T) {
	m := NewCacheModule(nil)
	defer m.Close()

	m.set("exists", 42)

	has, _ := m.has("exists")
	assert.Equal(t, true, has)

	has, _ = m.has("missing")
	assert.Equal(t, false, has)
}

func TestCache_Has_Expired(t *testing.T) {
	m := NewCacheModule(nil)
	defer m.Close()

	m.set("key", "val", 1)
	time.Sleep(1100 * time.Millisecond)

	has, _ := m.has("key")
	assert.Equal(t, false, has)
}

func TestCache_Del(t *testing.T) {
	m := NewCacheModule(nil)
	defer m.Close()

	m.set("key", "val")
	m.del("key")

	val, _ := m.get("key")
	assert.Nil(t, val)
}

func TestCache_Del_NonExistent(t *testing.T) {
	m := NewCacheModule(nil)
	defer m.Close()

	_, err := m.del("nonexistent")
	assert.NoError(t, err)
}

func TestCache_Clear(t *testing.T) {
	m := NewCacheModule(nil)
	defer m.Close()

	m.set("a", 1)
	m.set("b", 2)
	m.set("c", 3)
	m.clear()

	val, _ := m.get("a")
	assert.Nil(t, val)
	val, _ = m.get("b")
	assert.Nil(t, val)
	val, _ = m.get("c")
	assert.Nil(t, val)
}

func TestCache_Overwrite(t *testing.T) {
	m := NewCacheModule(nil)
	defer m.Close()

	m.set("key", "first")
	m.set("key", "second")

	val, _ := m.get("key")
	assert.Equal(t, "second", val)
}

func TestCache_Security_MaxEntries(t *testing.T) {
	m := NewCacheModule(&SecurityConfig{
		Cache: CacheSecurityConfig{MaxEntries: 3},
	})
	defer m.Close()

	m.set("a", 1)
	m.set("b", 2)
	m.set("c", 3)

	_, err := m.set("d", 4)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "max entries limit")
}

func TestCache_Security_MaxEntries_OverwriteAllowed(t *testing.T) {
	m := NewCacheModule(&SecurityConfig{
		Cache: CacheSecurityConfig{MaxEntries: 3},
	})
	defer m.Close()

	m.set("a", 1)
	m.set("b", 2)
	m.set("c", 3)

	_, err := m.set("a", 99)
	assert.NoError(t, err)

	val, _ := m.get("a")
	assert.Equal(t, 99, val)
}

func TestCache_Security_MaxValueSize(t *testing.T) {
	m := NewCacheModule(&SecurityConfig{
		Cache: CacheSecurityConfig{MaxValueSize: 100},
	})
	defer m.Close()

	_, err := m.set("small", "hello")
	assert.NoError(t, err)

	largeData := make(map[string]any)
	for i := 0; i < 50; i++ {
		largeData[fmt.Sprintf("key_%d", i)] = "some long value string here"
	}
	_, err = m.set("large", largeData)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "value size")
}

func TestCache_Security_MaxTTL(t *testing.T) {
	m := NewCacheModule(&SecurityConfig{
		Cache: CacheSecurityConfig{MaxTTL: 60},
	})
	defer m.Close()

	m.set("key", "val", 9999)

	val, _ := m.get("key")
	assert.Equal(t, "val", val)

	m.mu.RLock()
	entry := m.entries["key"]
	m.mu.RUnlock()
	assert.False(t, entry.expiresAt.IsZero())
	assert.True(t, entry.expiresAt.Before(time.Now().Add(61*time.Second)))
}

func TestCache_Concurrent(t *testing.T) {
	m := NewCacheModule(nil)
	defer m.Close()

	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			m.set(fmt.Sprintf("key_%d", i), i)
		}(i)
	}
	wg.Wait()

	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			m.get(fmt.Sprintf("key_%d", i))
		}(i)
	}
	wg.Wait()
}

func TestCache_ErrorHandling(t *testing.T) {
	m := NewCacheModule(nil)
	defer m.Close()

	_, err := m.get()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "key required")

	_, err = m.get(123)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "key must be a string")

	_, err = m.set("key")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "key and value required")

	_, err = m.set(123, "val")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "key must be a string")

	_, err = m.del()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "key required")

	_, err = m.del(123)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "key must be a string")

	_, err = m.has()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "key required")

	_, err = m.has(123)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "key must be a string")
}

func TestCache_Name(t *testing.T) {
	m := NewCacheModule(nil)
	defer m.Close()
	assert.Equal(t, "cache", m.Name())
}

func TestCache_Functions(t *testing.T) {
	m := NewCacheModule(nil)
	defer m.Close()
	fns := m.Functions()
	assert.Contains(t, fns, "get")
	assert.Contains(t, fns, "set")
	assert.Contains(t, fns, "del")
	assert.Contains(t, fns, "has")
	assert.Contains(t, fns, "clear")
}

func TestCache_TTL_Float64(t *testing.T) {
	m := NewCacheModule(nil)
	defer m.Close()

	_, err := m.set("key", "val", float64(1))
	assert.NoError(t, err)

	val, _ := m.get("key")
	assert.Equal(t, "val", val)

	time.Sleep(1100 * time.Millisecond)
	val, _ = m.get("key")
	assert.Nil(t, val)
}

func TestCache_TTL_Int64(t *testing.T) {
	m := NewCacheModule(nil)
	defer m.Close()

	_, err := m.set("key", "val", int64(1))
	assert.NoError(t, err)

	val, _ := m.get("key")
	assert.Equal(t, "val", val)

	time.Sleep(1100 * time.Millisecond)
	val, _ = m.get("key")
	assert.Nil(t, val)
}
