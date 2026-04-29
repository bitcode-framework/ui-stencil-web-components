package io

import (
	"encoding/json"
	"fmt"
	"strings"
	"sync"
)

// RedisModule provides Redis functions for go-json programs.
// Requires github.com/redis/go-redis/v9 — operations are stubbed until driver is available.
type RedisModule struct {
	security *SecurityConfig
	config   map[string]any
	pools    map[string]any
	poolsMu  sync.Mutex
}

// NewRedisModule creates a new Redis I/O module.
func NewRedisModule(security *SecurityConfig) *RedisModule {
	if security == nil {
		security = DefaultSecurityConfig()
	}
	return &RedisModule{
		security: security,
		pools:    make(map[string]any),
	}
}

func (m *RedisModule) Name() string { return "redis" }

func (m *RedisModule) SetConfig(cfg map[string]any) { m.config = cfg }

func (m *RedisModule) Close() error {
	m.poolsMu.Lock()
	defer m.poolsMu.Unlock()
	for k := range m.pools {
		delete(m.pools, k)
	}
	return nil
}

var defaultBlockedRedisCommands = []string{
	"FLUSHALL", "FLUSHDB", "CONFIG", "DEBUG", "SHUTDOWN", "SLAVEOF", "REPLICAOF",
}

func (m *RedisModule) Functions() map[string]any {
	return map[string]any{
		"get":      m.redisGet,
		"set":      m.redisSet,
		"del":      m.redisDel,
		"exists":   m.redisExists,
		"expire":   m.redisExpire,
		"ttl":      m.redisTTL,
		"incr":     m.redisIncr,
		"decr":     m.redisDecr,
		"hget":     m.redisHGet,
		"hset":     m.redisHSet,
		"hgetall":  m.redisHGetAll,
		"lpush":    m.redisLPush,
		"rpush":    m.redisRPush,
		"lrange":   m.redisLRange,
		"sadd":     m.redisSAdd,
		"smembers": m.redisSMembers,
		"publish":  m.redisPublish,
	}
}

func (m *RedisModule) prefixKey(key string) string {
	if m.security.Redis.KeyPrefix != "" {
		return m.security.Redis.KeyPrefix + key
	}
	return key
}

func (m *RedisModule) validateKey(key string) error {
	maxLen := m.security.Redis.MaxKeyLength
	if maxLen <= 0 {
		maxLen = 1024
	}
	if len(key) > maxLen {
		return fmt.Errorf("redis: key exceeds max length (%d chars, max %d)", len(key), maxLen)
	}
	return nil
}

func (m *RedisModule) validateCommand(cmd string) error {
	blocked := m.security.Redis.BlockedCommands
	if blocked == nil {
		blocked = defaultBlockedRedisCommands
	}
	for _, b := range blocked {
		if strings.EqualFold(cmd, b) {
			return fmt.Errorf("redis: command '%s' is blocked", cmd)
		}
	}
	return nil
}

func (m *RedisModule) autoSerialize(v any) (string, error) {
	switch val := v.(type) {
	case string:
		return val, nil
	default:
		data, err := json.Marshal(val)
		if err != nil {
			return "", fmt.Errorf("redis: cannot serialize value: %s", err.Error())
		}
		return string(data), nil
	}
}

func (m *RedisModule) autoDeserialize(s string) any {
	var result any
	if err := json.Unmarshal([]byte(s), &result); err != nil {
		return s
	}
	return result
}

func stubErr(op string) error {
	return fmt.Errorf("redis.%s: Redis driver not available — add github.com/redis/go-redis/v9 dependency", op)
}

func (m *RedisModule) redisGet(params ...any) (any, error) {
	if len(params) < 1 {
		return nil, fmt.Errorf("redis.get: key is required")
	}
	key, _ := params[0].(string)
	if err := m.validateKey(key); err != nil {
		return nil, err
	}
	return nil, stubErr("get")
}

func (m *RedisModule) redisSet(params ...any) (any, error) {
	if len(params) < 2 {
		return nil, fmt.Errorf("redis.set: key and value are required")
	}
	key, _ := params[0].(string)
	if err := m.validateKey(key); err != nil {
		return nil, err
	}

	val, err := m.autoSerialize(params[1])
	if err != nil {
		return nil, err
	}

	if m.security.Redis.MaxValueSize > 0 && int64(len(val)) > m.security.Redis.MaxValueSize {
		return nil, fmt.Errorf("redis.set: value exceeds max size (%d bytes, max %d)", len(val), m.security.Redis.MaxValueSize)
	}

	return nil, stubErr("set")
}

func (m *RedisModule) redisDel(params ...any) (any, error) {
	if len(params) < 1 {
		return nil, fmt.Errorf("redis.del: key is required")
	}
	return nil, stubErr("del")
}

func (m *RedisModule) redisExists(params ...any) (any, error) {
	if len(params) < 1 {
		return nil, fmt.Errorf("redis.exists: key is required")
	}
	return nil, stubErr("exists")
}

func (m *RedisModule) redisExpire(params ...any) (any, error) {
	if len(params) < 2 {
		return nil, fmt.Errorf("redis.expire: key and seconds are required")
	}
	return nil, stubErr("expire")
}

func (m *RedisModule) redisTTL(params ...any) (any, error) {
	if len(params) < 1 {
		return nil, fmt.Errorf("redis.ttl: key is required")
	}
	return nil, stubErr("ttl")
}

func (m *RedisModule) redisIncr(params ...any) (any, error) {
	if len(params) < 1 {
		return nil, fmt.Errorf("redis.incr: key is required")
	}
	return nil, stubErr("incr")
}

func (m *RedisModule) redisDecr(params ...any) (any, error) {
	if len(params) < 1 {
		return nil, fmt.Errorf("redis.decr: key is required")
	}
	return nil, stubErr("decr")
}

func (m *RedisModule) redisHGet(params ...any) (any, error) {
	if len(params) < 2 {
		return nil, fmt.Errorf("redis.hget: key and field are required")
	}
	return nil, stubErr("hget")
}

func (m *RedisModule) redisHSet(params ...any) (any, error) {
	if len(params) < 3 {
		return nil, fmt.Errorf("redis.hset: key, field, and value are required")
	}
	return nil, stubErr("hset")
}

func (m *RedisModule) redisHGetAll(params ...any) (any, error) {
	if len(params) < 1 {
		return nil, fmt.Errorf("redis.hgetall: key is required")
	}
	return nil, stubErr("hgetall")
}

func (m *RedisModule) redisLPush(params ...any) (any, error) {
	if len(params) < 2 {
		return nil, fmt.Errorf("redis.lpush: key and value are required")
	}
	return nil, stubErr("lpush")
}

func (m *RedisModule) redisRPush(params ...any) (any, error) {
	if len(params) < 2 {
		return nil, fmt.Errorf("redis.rpush: key and value are required")
	}
	return nil, stubErr("rpush")
}

func (m *RedisModule) redisLRange(params ...any) (any, error) {
	if len(params) < 3 {
		return nil, fmt.Errorf("redis.lrange: key, start, and stop are required")
	}
	return nil, stubErr("lrange")
}

func (m *RedisModule) redisSAdd(params ...any) (any, error) {
	if len(params) < 2 {
		return nil, fmt.Errorf("redis.sadd: key and member are required")
	}
	return nil, stubErr("sadd")
}

func (m *RedisModule) redisSMembers(params ...any) (any, error) {
	if len(params) < 1 {
		return nil, fmt.Errorf("redis.smembers: key is required")
	}
	return nil, stubErr("smembers")
}

func (m *RedisModule) redisPublish(params ...any) (any, error) {
	if len(params) < 2 {
		return nil, fmt.Errorf("redis.publish: channel and message are required")
	}
	return nil, stubErr("publish")
}
