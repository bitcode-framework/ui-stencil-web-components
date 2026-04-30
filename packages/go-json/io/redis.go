package io

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
)

// RedisModule provides Redis functions for go-json programs.
type RedisModule struct {
	security *SecurityConfig
	config   map[string]any
	client   *redis.Client
	mu       sync.Mutex
	closed   bool
}

// NewRedisModule creates a new Redis I/O module.
// Connection is lazy — established on first operation using security.Redis.DefaultURI.
func NewRedisModule(security *SecurityConfig) *RedisModule {
	if security == nil {
		security = DefaultSecurityConfig()
	}
	return &RedisModule{
		security: security,
	}
}

func (m *RedisModule) Name() string { return "redis" }

func (m *RedisModule) SetConfig(cfg map[string]any) { m.config = cfg }

func (m *RedisModule) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.closed = true
	if m.client != nil {
		err := m.client.Close()
		m.client = nil
		return err
	}
	return nil
}

func (m *RedisModule) getClient() (*redis.Client, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.closed {
		return nil, fmt.Errorf("redis: module is closed")
	}
	if m.client != nil {
		return m.client, nil
	}

	uri := m.security.Redis.DefaultURI
	if uri == "" {
		uri = "redis://localhost:6379"
	}

	opts, err := redis.ParseURL(uri)
	if err != nil {
		return nil, fmt.Errorf("redis: invalid URI: %s", err.Error())
	}

	m.client = redis.NewClient(opts)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := m.client.Ping(ctx).Err(); err != nil {
		m.client.Close()
		m.client = nil
		return nil, fmt.Errorf("redis: connection failed: %s", err.Error())
	}

	return m.client, nil
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

func (m *RedisModule) redisGet(params ...any) (any, error) {
	if len(params) < 1 {
		return nil, fmt.Errorf("redis.get: key is required")
	}
	key, _ := params[0].(string)
	if err := m.validateKey(key); err != nil {
		return nil, err
	}

	client, err := m.getClient()
	if err != nil {
		return nil, err
	}

	key = m.prefixKey(key)
	val, err := client.Get(context.Background(), key).Result()
	if err == redis.Nil {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("redis.get: %s", err.Error())
	}

	return m.autoDeserialize(val), nil
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

	client, cErr := m.getClient()
	if cErr != nil {
		return nil, cErr
	}

	var ttl time.Duration
	if len(params) > 2 {
		if ttlVal, ok := toFloat64Val(params[2]); ok && ttlVal > 0 {
			ttl = time.Duration(int(ttlVal)) * time.Second
		}
	}

	key = m.prefixKey(key)
	if err := client.Set(context.Background(), key, val, ttl).Err(); err != nil {
		return nil, fmt.Errorf("redis.set: %s", err.Error())
	}
	return "OK", nil
}

func (m *RedisModule) redisDel(params ...any) (any, error) {
	if len(params) < 1 {
		return nil, fmt.Errorf("redis.del: key is required")
	}
	key, _ := params[0].(string)
	if err := m.validateKey(key); err != nil {
		return nil, err
	}

	client, err := m.getClient()
	if err != nil {
		return nil, err
	}

	key = m.prefixKey(key)
	n, err := client.Del(context.Background(), key).Result()
	if err != nil {
		return nil, fmt.Errorf("redis.del: %s", err.Error())
	}
	return n, nil
}

func (m *RedisModule) redisExists(params ...any) (any, error) {
	if len(params) < 1 {
		return nil, fmt.Errorf("redis.exists: key is required")
	}
	key, _ := params[0].(string)
	if err := m.validateKey(key); err != nil {
		return nil, err
	}

	client, err := m.getClient()
	if err != nil {
		return nil, err
	}

	key = m.prefixKey(key)
	n, err := client.Exists(context.Background(), key).Result()
	if err != nil {
		return nil, fmt.Errorf("redis.exists: %s", err.Error())
	}
	return n > 0, nil
}

func (m *RedisModule) redisExpire(params ...any) (any, error) {
	if len(params) < 2 {
		return nil, fmt.Errorf("redis.expire: key and seconds are required")
	}
	key, _ := params[0].(string)
	if err := m.validateKey(key); err != nil {
		return nil, err
	}

	client, err := m.getClient()
	if err != nil {
		return nil, err
	}

	seconds, _ := toFloat64Val(params[1])
	key = m.prefixKey(key)
	ok, err := client.Expire(context.Background(), key, time.Duration(int(seconds))*time.Second).Result()
	if err != nil {
		return nil, fmt.Errorf("redis.expire: %s", err.Error())
	}
	return ok, nil
}

func (m *RedisModule) redisTTL(params ...any) (any, error) {
	if len(params) < 1 {
		return nil, fmt.Errorf("redis.ttl: key is required")
	}
	key, _ := params[0].(string)
	if err := m.validateKey(key); err != nil {
		return nil, err
	}

	client, err := m.getClient()
	if err != nil {
		return nil, err
	}

	key = m.prefixKey(key)
	d, err := client.TTL(context.Background(), key).Result()
	if err != nil {
		return nil, fmt.Errorf("redis.ttl: %s", err.Error())
	}
	return int(d.Seconds()), nil
}

func (m *RedisModule) redisIncr(params ...any) (any, error) {
	if len(params) < 1 {
		return nil, fmt.Errorf("redis.incr: key is required")
	}
	key, _ := params[0].(string)
	if err := m.validateKey(key); err != nil {
		return nil, err
	}

	client, err := m.getClient()
	if err != nil {
		return nil, err
	}

	key = m.prefixKey(key)
	n, err := client.Incr(context.Background(), key).Result()
	if err != nil {
		return nil, fmt.Errorf("redis.incr: %s", err.Error())
	}
	return n, nil
}

func (m *RedisModule) redisDecr(params ...any) (any, error) {
	if len(params) < 1 {
		return nil, fmt.Errorf("redis.decr: key is required")
	}
	key, _ := params[0].(string)
	if err := m.validateKey(key); err != nil {
		return nil, err
	}

	client, err := m.getClient()
	if err != nil {
		return nil, err
	}

	key = m.prefixKey(key)
	n, err := client.Decr(context.Background(), key).Result()
	if err != nil {
		return nil, fmt.Errorf("redis.decr: %s", err.Error())
	}
	return n, nil
}

func (m *RedisModule) redisHGet(params ...any) (any, error) {
	if len(params) < 2 {
		return nil, fmt.Errorf("redis.hget: key and field are required")
	}
	key, _ := params[0].(string)
	if err := m.validateKey(key); err != nil {
		return nil, err
	}

	client, err := m.getClient()
	if err != nil {
		return nil, err
	}

	field, _ := params[1].(string)
	key = m.prefixKey(key)
	val, err := client.HGet(context.Background(), key, field).Result()
	if err == redis.Nil {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("redis.hget: %s", err.Error())
	}
	return m.autoDeserialize(val), nil
}

func (m *RedisModule) redisHSet(params ...any) (any, error) {
	if len(params) < 3 {
		return nil, fmt.Errorf("redis.hset: key, field, and value are required")
	}
	key, _ := params[0].(string)
	if err := m.validateKey(key); err != nil {
		return nil, err
	}

	client, cErr := m.getClient()
	if cErr != nil {
		return nil, cErr
	}

	field, _ := params[1].(string)
	val, err := m.autoSerialize(params[2])
	if err != nil {
		return nil, err
	}

	key = m.prefixKey(key)
	if err := client.HSet(context.Background(), key, field, val).Err(); err != nil {
		return nil, fmt.Errorf("redis.hset: %s", err.Error())
	}
	return "OK", nil
}

func (m *RedisModule) redisHGetAll(params ...any) (any, error) {
	if len(params) < 1 {
		return nil, fmt.Errorf("redis.hgetall: key is required")
	}
	key, _ := params[0].(string)
	if err := m.validateKey(key); err != nil {
		return nil, err
	}

	client, err := m.getClient()
	if err != nil {
		return nil, err
	}

	key = m.prefixKey(key)
	hash, err := client.HGetAll(context.Background(), key).Result()
	if err != nil {
		return nil, fmt.Errorf("redis.hgetall: %s", err.Error())
	}

	result := make(map[string]any, len(hash))
	for k, v := range hash {
		result[k] = m.autoDeserialize(v)
	}
	return result, nil
}

func (m *RedisModule) redisLPush(params ...any) (any, error) {
	if len(params) < 2 {
		return nil, fmt.Errorf("redis.lpush: key and value are required")
	}
	key, _ := params[0].(string)
	if err := m.validateKey(key); err != nil {
		return nil, err
	}

	client, cErr := m.getClient()
	if cErr != nil {
		return nil, cErr
	}

	val, err := m.autoSerialize(params[1])
	if err != nil {
		return nil, err
	}

	key = m.prefixKey(key)
	n, err := client.LPush(context.Background(), key, val).Result()
	if err != nil {
		return nil, fmt.Errorf("redis.lpush: %s", err.Error())
	}
	return n, nil
}

func (m *RedisModule) redisRPush(params ...any) (any, error) {
	if len(params) < 2 {
		return nil, fmt.Errorf("redis.rpush: key and value are required")
	}
	key, _ := params[0].(string)
	if err := m.validateKey(key); err != nil {
		return nil, err
	}

	client, cErr := m.getClient()
	if cErr != nil {
		return nil, cErr
	}

	val, err := m.autoSerialize(params[1])
	if err != nil {
		return nil, err
	}

	key = m.prefixKey(key)
	n, err := client.RPush(context.Background(), key, val).Result()
	if err != nil {
		return nil, fmt.Errorf("redis.rpush: %s", err.Error())
	}
	return n, nil
}

func (m *RedisModule) redisLRange(params ...any) (any, error) {
	if len(params) < 3 {
		return nil, fmt.Errorf("redis.lrange: key, start, and stop are required")
	}
	key, _ := params[0].(string)
	if err := m.validateKey(key); err != nil {
		return nil, err
	}

	client, err := m.getClient()
	if err != nil {
		return nil, err
	}

	start, _ := toFloat64Val(params[1])
	stop, _ := toFloat64Val(params[2])
	key = m.prefixKey(key)
	vals, err := client.LRange(context.Background(), key, int64(start), int64(stop)).Result()
	if err != nil {
		return nil, fmt.Errorf("redis.lrange: %s", err.Error())
	}

	result := make([]any, len(vals))
	for i, v := range vals {
		result[i] = m.autoDeserialize(v)
	}
	return result, nil
}

func (m *RedisModule) redisSAdd(params ...any) (any, error) {
	if len(params) < 2 {
		return nil, fmt.Errorf("redis.sadd: key and member are required")
	}
	key, _ := params[0].(string)
	if err := m.validateKey(key); err != nil {
		return nil, err
	}

	client, cErr := m.getClient()
	if cErr != nil {
		return nil, cErr
	}

	val, err := m.autoSerialize(params[1])
	if err != nil {
		return nil, err
	}

	key = m.prefixKey(key)
	n, err := client.SAdd(context.Background(), key, val).Result()
	if err != nil {
		return nil, fmt.Errorf("redis.sadd: %s", err.Error())
	}
	return n, nil
}

func (m *RedisModule) redisSMembers(params ...any) (any, error) {
	if len(params) < 1 {
		return nil, fmt.Errorf("redis.smembers: key is required")
	}
	key, _ := params[0].(string)
	if err := m.validateKey(key); err != nil {
		return nil, err
	}

	client, err := m.getClient()
	if err != nil {
		return nil, err
	}

	key = m.prefixKey(key)
	vals, err := client.SMembers(context.Background(), key).Result()
	if err != nil {
		return nil, fmt.Errorf("redis.smembers: %s", err.Error())
	}

	result := make([]any, len(vals))
	for i, v := range vals {
		result[i] = m.autoDeserialize(v)
	}
	return result, nil
}

func (m *RedisModule) redisPublish(params ...any) (any, error) {
	if len(params) < 2 {
		return nil, fmt.Errorf("redis.publish: channel and message are required")
	}
	channel, _ := params[0].(string)
	msg, _ := params[1].(string)

	client, err := m.getClient()
	if err != nil {
		return nil, err
	}

	if err := client.Publish(context.Background(), channel, msg).Err(); err != nil {
		return nil, fmt.Errorf("redis.publish: %s", err.Error())
	}
	return "OK", nil
}

// SetClient allows injecting a pre-configured redis client (for testing with miniredis).
func (m *RedisModule) SetClient(client *redis.Client) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.client = client
}
