package io

import (
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
)

func newTestRedis(t *testing.T) (*RedisModule, *miniredis.Miniredis) {
	t.Helper()
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("miniredis: %v", err)
	}
	t.Cleanup(mr.Close)

	sec := DefaultSecurityConfig()
	m := NewRedisModule(sec)
	m.SetClient(redis.NewClient(&redis.Options{Addr: mr.Addr()}))
	return m, mr
}

func TestRedisModule_SetGet(t *testing.T) {
	m, _ := newTestRedis(t)
	defer m.Close()

	_, err := m.redisSet("key1", "hello")
	if err != nil {
		t.Fatalf("set: %v", err)
	}

	val, err := m.redisGet("key1")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if val != "hello" {
		t.Errorf("expected 'hello', got %v", val)
	}
}

func TestRedisModule_SetGetJSON(t *testing.T) {
	m, _ := newTestRedis(t)
	defer m.Close()

	data := map[string]any{"name": "Alice", "age": float64(30)}
	_, err := m.redisSet("user:1", data)
	if err != nil {
		t.Fatalf("set: %v", err)
	}

	val, err := m.redisGet("user:1")
	if err != nil {
		t.Fatalf("get: %v", err)
	}

	obj, ok := val.(map[string]any)
	if !ok {
		t.Fatalf("expected map, got %T", val)
	}
	if obj["name"] != "Alice" {
		t.Errorf("expected name=Alice, got %v", obj["name"])
	}
}

func TestRedisModule_Del(t *testing.T) {
	m, _ := newTestRedis(t)
	defer m.Close()

	m.redisSet("delme", "value")
	result, err := m.redisDel("delme")
	if err != nil {
		t.Fatalf("del: %v", err)
	}
	if result != int64(1) {
		t.Errorf("expected 1 deleted, got %v", result)
	}

	val, _ := m.redisGet("delme")
	if val != nil {
		t.Errorf("expected nil after delete, got %v", val)
	}
}

func TestRedisModule_Exists(t *testing.T) {
	m, _ := newTestRedis(t)
	defer m.Close()

	m.redisSet("exists_key", "yes")

	exists, err := m.redisExists("exists_key")
	if err != nil {
		t.Fatalf("exists: %v", err)
	}
	if exists != true {
		t.Errorf("expected true, got %v", exists)
	}

	exists, err = m.redisExists("no_such_key")
	if err != nil {
		t.Fatalf("exists: %v", err)
	}
	if exists != false {
		t.Errorf("expected false, got %v", exists)
	}
}

func TestRedisModule_IncrDecr(t *testing.T) {
	m, _ := newTestRedis(t)
	defer m.Close()

	val, err := m.redisIncr("counter")
	if err != nil {
		t.Fatalf("incr: %v", err)
	}
	if val != int64(1) {
		t.Errorf("expected 1, got %v", val)
	}

	val, err = m.redisIncr("counter")
	if err != nil {
		t.Fatalf("incr: %v", err)
	}
	if val != int64(2) {
		t.Errorf("expected 2, got %v", val)
	}

	val, err = m.redisDecr("counter")
	if err != nil {
		t.Fatalf("decr: %v", err)
	}
	if val != int64(1) {
		t.Errorf("expected 1, got %v", val)
	}
}

func TestRedisModule_ExpireTTL(t *testing.T) {
	m, mr := newTestRedis(t)
	defer m.Close()

	m.redisSet("ttl_key", "value")
	_, err := m.redisExpire("ttl_key", 60)
	if err != nil {
		t.Fatalf("expire: %v", err)
	}

	ttl, err := m.redisTTL("ttl_key")
	if err != nil {
		t.Fatalf("ttl: %v", err)
	}

	ttlInt, ok := ttl.(int)
	if !ok {
		t.Fatalf("expected int, got %T", ttl)
	}
	if ttlInt <= 0 || ttlInt > 60 {
		t.Errorf("expected TTL 1-60, got %d", ttlInt)
	}
	_ = mr
}

func TestRedisModule_HashOps(t *testing.T) {
	m, _ := newTestRedis(t)
	defer m.Close()

	_, err := m.redisHSet("user:1", "name", "Alice")
	if err != nil {
		t.Fatalf("hset: %v", err)
	}
	_, err = m.redisHSet("user:1", "age", "30")
	if err != nil {
		t.Fatalf("hset: %v", err)
	}

	val, err := m.redisHGet("user:1", "name")
	if err != nil {
		t.Fatalf("hget: %v", err)
	}
	if val != "Alice" {
		t.Errorf("expected Alice, got %v", val)
	}

	all, err := m.redisHGetAll("user:1")
	if err != nil {
		t.Fatalf("hgetall: %v", err)
	}
	hashMap, ok := all.(map[string]any)
	if !ok {
		t.Fatalf("expected map, got %T", all)
	}
	if hashMap["name"] != "Alice" {
		t.Errorf("expected name=Alice, got %v", hashMap["name"])
	}
}

func TestRedisModule_ListOps(t *testing.T) {
	m, _ := newTestRedis(t)
	defer m.Close()

	m.redisRPush("mylist", "a")
	m.redisRPush("mylist", "b")
	m.redisLPush("mylist", "z")

	result, err := m.redisLRange("mylist", 0, -1)
	if err != nil {
		t.Fatalf("lrange: %v", err)
	}
	items, ok := result.([]any)
	if !ok {
		t.Fatalf("expected []any, got %T", result)
	}
	if len(items) != 3 {
		t.Errorf("expected 3 items, got %d", len(items))
	}
}

func TestRedisModule_SetOps(t *testing.T) {
	m, _ := newTestRedis(t)
	defer m.Close()

	m.redisSAdd("myset", "a")
	m.redisSAdd("myset", "b")
	m.redisSAdd("myset", "a")

	result, err := m.redisSMembers("myset")
	if err != nil {
		t.Fatalf("smembers: %v", err)
	}
	members, ok := result.([]any)
	if !ok {
		t.Fatalf("expected []any, got %T", result)
	}
	if len(members) != 2 {
		t.Errorf("expected 2 members, got %d", len(members))
	}
}

func TestRedisModule_Publish(t *testing.T) {
	m, _ := newTestRedis(t)
	defer m.Close()

	_, err := m.redisPublish("channel", "hello")
	if err != nil {
		t.Fatalf("publish: %v", err)
	}
}

func TestRedisModule_KeyMaxLength(t *testing.T) {
	sec := DefaultSecurityConfig()
	sec.Redis.MaxKeyLength = 5
	m := NewRedisModule(sec)

	err := m.validateKey("toolong")
	if err == nil {
		t.Error("expected max length error")
	}
}

func TestRedisModule_MaxValueSize(t *testing.T) {
	sec := DefaultSecurityConfig()
	sec.Redis.MaxValueSize = 5
	m, _ := newTestRedis(t)
	m.security = sec
	defer m.Close()

	_, err := m.redisSet("key", "this is way too long")
	if err == nil {
		t.Error("expected max value size error")
	}
}

func TestRedisModule_SetWithTTL(t *testing.T) {
	m, _ := newTestRedis(t)
	defer m.Close()

	_, err := m.redisSet("ttlkey", "value", 30)
	if err != nil {
		t.Fatalf("set with ttl: %v", err)
	}

	val, _ := m.redisGet("ttlkey")
	if val != "value" {
		t.Errorf("expected 'value', got %v", val)
	}
}
