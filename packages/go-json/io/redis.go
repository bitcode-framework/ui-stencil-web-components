package io

import (
	"encoding/json"
	"fmt"
	"strings"
	"sync"
)

// RedisDriver is the interface for Redis operations.
// Implement this with github.com/redis/go-redis/v9 for production.
type RedisDriver interface {
	Get(key string) (string, error)
	Set(key, value string, ttl int) error
	Del(keys ...string) (int64, error)
	Exists(keys ...string) (int64, error)
	Expire(key string, seconds int) (bool, error)
	TTL(key string) (int, error)
	Incr(key string) (int64, error)
	Decr(key string) (int64, error)
	HGet(key, field string) (string, error)
	HSet(key string, values map[string]string) error
	HGetAll(key string) (map[string]string, error)
	LPush(key string, values ...string) (int64, error)
	RPush(key string, values ...string) (int64, error)
	LRange(key string, start, stop int64) ([]string, error)
	SAdd(key string, members ...string) (int64, error)
	SMembers(key string) ([]string, error)
	Publish(channel, message string) error
	Close() error
}

// InMemoryRedisDriver is a simple in-memory implementation for testing and development.
type InMemoryRedisDriver struct {
	mu      sync.Mutex
	strings map[string]string
	hashes  map[string]map[string]string
	lists   map[string][]string
	sets    map[string]map[string]bool
	ttls    map[string]int
}

// NewInMemoryRedisDriver creates a new in-memory Redis driver.
func NewInMemoryRedisDriver() *InMemoryRedisDriver {
	return &InMemoryRedisDriver{
		strings: make(map[string]string),
		hashes:  make(map[string]map[string]string),
		lists:   make(map[string][]string),
		sets:    make(map[string]map[string]bool),
		ttls:    make(map[string]int),
	}
}

func (d *InMemoryRedisDriver) Get(key string) (string, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	
	val, exists := d.strings[key]
	if !exists {
		return "", nil
	}
	return val, nil
}

func (d *InMemoryRedisDriver) Set(key, value string, ttl int) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	
	d.strings[key] = value
	if ttl > 0 {
		d.ttls[key] = ttl
	} else {
		d.ttls[key] = -1
	}
	return nil
}

func (d *InMemoryRedisDriver) Del(keys ...string) (int64, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	
	deleted := int64(0)
	for _, key := range keys {
		if _, exists := d.strings[key]; exists {
			delete(d.strings, key)
			delete(d.ttls, key)
			deleted++
		}
		if _, exists := d.hashes[key]; exists {
			delete(d.hashes, key)
			delete(d.ttls, key)
			deleted++
		}
		if _, exists := d.lists[key]; exists {
			delete(d.lists, key)
			delete(d.ttls, key)
			deleted++
		}
		if _, exists := d.sets[key]; exists {
			delete(d.sets, key)
			delete(d.ttls, key)
			deleted++
		}
	}
	return deleted, nil
}

func (d *InMemoryRedisDriver) Exists(keys ...string) (int64, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	
	count := int64(0)
	for _, key := range keys {
		if _, exists := d.strings[key]; exists {
			count++
		} else if _, exists := d.hashes[key]; exists {
			count++
		} else if _, exists := d.lists[key]; exists {
			count++
		} else if _, exists := d.sets[key]; exists {
			count++
		}
	}
	return count, nil
}

func (d *InMemoryRedisDriver) Expire(key string, seconds int) (bool, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	
	if _, exists := d.strings[key]; exists {
		d.ttls[key] = seconds
		return true, nil
	}
	return false, nil
}

func (d *InMemoryRedisDriver) TTL(key string) (int, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	
	ttl, exists := d.ttls[key]
	if !exists {
		return -2, nil
	}
	return ttl, nil
}

func (d *InMemoryRedisDriver) Incr(key string) (int64, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	
	val := int64(0)
	if str, exists := d.strings[key]; exists {
		fmt.Sscanf(str, "%d", &val)
	}
	val++
	d.strings[key] = fmt.Sprintf("%d", val)
	return val, nil
}

func (d *InMemoryRedisDriver) Decr(key string) (int64, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	
	val := int64(0)
	if str, exists := d.strings[key]; exists {
		fmt.Sscanf(str, "%d", &val)
	}
	val--
	d.strings[key] = fmt.Sprintf("%d", val)
	return val, nil
}

func (d *InMemoryRedisDriver) HGet(key, field string) (string, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	
	hash, exists := d.hashes[key]
	if !exists {
		return "", fmt.Errorf("key not found")
	}
	val, exists := hash[field]
	if !exists {
		return "", fmt.Errorf("field not found")
	}
	return val, nil
}

func (d *InMemoryRedisDriver) HSet(key string, values map[string]string) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	
	if d.hashes[key] == nil {
		d.hashes[key] = make(map[string]string)
	}
	for field, val := range values {
		d.hashes[key][field] = val
	}
	return nil
}

func (d *InMemoryRedisDriver) HGetAll(key string) (map[string]string, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	
	hash, exists := d.hashes[key]
	if !exists {
		return make(map[string]string), nil
	}
	
	result := make(map[string]string)
	for k, v := range hash {
		result[k] = v
	}
	return result, nil
}

func (d *InMemoryRedisDriver) LPush(key string, values ...string) (int64, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	
	list := d.lists[key]
	list = append(values, list...)
	d.lists[key] = list
	return int64(len(list)), nil
}

func (d *InMemoryRedisDriver) RPush(key string, values ...string) (int64, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	
	list := d.lists[key]
	list = append(list, values...)
	d.lists[key] = list
	return int64(len(list)), nil
}

func (d *InMemoryRedisDriver) LRange(key string, start, stop int64) ([]string, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	
	list, exists := d.lists[key]
	if !exists {
		return []string{}, nil
	}
	
	length := int64(len(list))
	if start < 0 {
		start = length + start
	}
	if stop < 0 {
		stop = length + stop
	}
	if start < 0 {
		start = 0
	}
	if stop >= length {
		stop = length - 1
	}
	if start > stop {
		return []string{}, nil
	}
	
	return list[start : stop+1], nil
}

func (d *InMemoryRedisDriver) SAdd(key string, members ...string) (int64, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	
	if d.sets[key] == nil {
		d.sets[key] = make(map[string]bool)
	}
	added := int64(0)
	for _, member := range members {
		if !d.sets[key][member] {
			d.sets[key][member] = true
			added++
		}
	}
	return added, nil
}

func (d *InMemoryRedisDriver) SMembers(key string) ([]string, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	
	set, exists := d.sets[key]
	if !exists {
		return []string{}, nil
	}
	
	var members []string
	for member := range set {
		members = append(members, member)
	}
	return members, nil
}

func (d *InMemoryRedisDriver) Publish(channel, message string) error {
	return nil
}

func (d *InMemoryRedisDriver) Close() error {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.strings = make(map[string]string)
	d.hashes = make(map[string]map[string]string)
	d.lists = make(map[string][]string)
	d.sets = make(map[string]map[string]bool)
	d.ttls = make(map[string]int)
	return nil
}

// RedisModule provides Redis functions for go-json programs.
type RedisModule struct {
	security *SecurityConfig
	config   map[string]any
	driver   RedisDriver
	mu       sync.Mutex
}

// RedisOption is a functional option for RedisModule.
type RedisOption func(*RedisModule)

// WithRedisDriver sets a custom Redis driver.
func WithRedisDriver(d RedisDriver) RedisOption {
	return func(m *RedisModule) { m.driver = d }
}

// NewRedisModule creates a new Redis I/O module.
func NewRedisModule(security *SecurityConfig, opts ...RedisOption) *RedisModule {
	if security == nil {
		security = DefaultSecurityConfig()
	}
	m := &RedisModule{
		security: security,
	}
	for _, opt := range opts {
		opt(m)
	}
	if m.driver == nil {
		m.driver = NewInMemoryRedisDriver()
	}
	return m
}

func (m *RedisModule) Name() string { return "redis" }

func (m *RedisModule) SetConfig(cfg map[string]any) { m.config = cfg }

func (m *RedisModule) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.driver != nil {
		return m.driver.Close()
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

func (m *RedisModule) redisGet(params ...any) (any, error) {
	if len(params) < 1 {
		return nil, fmt.Errorf("redis.get: key is required")
	}
	key, _ := params[0].(string)
	if err := m.validateKey(key); err != nil {
		return nil, err
	}
	
	key = m.prefixKey(key)
	val, err := m.driver.Get(key)
	if err != nil {
		return nil, err
	}
	if val == "" {
		return nil, nil
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

	ttl := 0
	if len(params) > 2 {
		if ttlFloat, ok := params[2].(float64); ok {
			ttl = int(ttlFloat)
		} else if ttlInt, ok := params[2].(int); ok {
			ttl = ttlInt
		}
	}

	key = m.prefixKey(key)
	return "OK", m.driver.Set(key, val, ttl)
}

func (m *RedisModule) redisDel(params ...any) (any, error) {
	if len(params) < 1 {
		return nil, fmt.Errorf("redis.del: key is required")
	}
	
	var keys []string
	for _, p := range params {
		if key, ok := p.(string); ok {
			if err := m.validateKey(key); err != nil {
				return nil, err
			}
			keys = append(keys, m.prefixKey(key))
		}
	}
	
	return m.driver.Del(keys...)
}

func (m *RedisModule) redisExists(params ...any) (any, error) {
	if len(params) < 1 {
		return nil, fmt.Errorf("redis.exists: key is required")
	}
	
	var keys []string
	for _, p := range params {
		if key, ok := p.(string); ok {
			keys = append(keys, m.prefixKey(key))
		}
	}
	
	return m.driver.Exists(keys...)
}

func (m *RedisModule) redisExpire(params ...any) (any, error) {
	if len(params) < 2 {
		return nil, fmt.Errorf("redis.expire: key and seconds are required")
	}
	key, _ := params[0].(string)
	if err := m.validateKey(key); err != nil {
		return nil, err
	}
	
	seconds := 0
	if secFloat, ok := params[1].(float64); ok {
		seconds = int(secFloat)
	} else if secInt, ok := params[1].(int); ok {
		seconds = secInt
	}
	
	key = m.prefixKey(key)
	return m.driver.Expire(key, seconds)
}

func (m *RedisModule) redisTTL(params ...any) (any, error) {
	if len(params) < 1 {
		return nil, fmt.Errorf("redis.ttl: key is required")
	}
	key, _ := params[0].(string)
	if err := m.validateKey(key); err != nil {
		return nil, err
	}
	
	key = m.prefixKey(key)
	return m.driver.TTL(key)
}

func (m *RedisModule) redisIncr(params ...any) (any, error) {
	if len(params) < 1 {
		return nil, fmt.Errorf("redis.incr: key is required")
	}
	key, _ := params[0].(string)
	if err := m.validateKey(key); err != nil {
		return nil, err
	}
	
	key = m.prefixKey(key)
	return m.driver.Incr(key)
}

func (m *RedisModule) redisDecr(params ...any) (any, error) {
	if len(params) < 1 {
		return nil, fmt.Errorf("redis.decr: key is required")
	}
	key, _ := params[0].(string)
	if err := m.validateKey(key); err != nil {
		return nil, err
	}
	
	key = m.prefixKey(key)
	return m.driver.Decr(key)
}

func (m *RedisModule) redisHGet(params ...any) (any, error) {
	if len(params) < 2 {
		return nil, fmt.Errorf("redis.hget: key and field are required")
	}
	key, _ := params[0].(string)
	field, _ := params[1].(string)
	if err := m.validateKey(key); err != nil {
		return nil, err
	}
	
	key = m.prefixKey(key)
	val, err := m.driver.HGet(key, field)
	if err != nil {
		return nil, err
	}
	return m.autoDeserialize(val), nil
}

func (m *RedisModule) redisHSet(params ...any) (any, error) {
	if len(params) < 2 {
		return nil, fmt.Errorf("redis.hset: key and field-value pairs are required")
	}
	key, _ := params[0].(string)
	if err := m.validateKey(key); err != nil {
		return nil, err
	}
	
	values := make(map[string]string)
	if len(params) == 3 {
		field, _ := params[1].(string)
		val, err := m.autoSerialize(params[2])
		if err != nil {
			return nil, err
		}
		values[field] = val
	} else if len(params) == 2 {
		if valMap, ok := params[1].(map[string]any); ok {
			for field, v := range valMap {
				val, err := m.autoSerialize(v)
				if err != nil {
					return nil, err
				}
				values[field] = val
			}
		}
	}
	
	key = m.prefixKey(key)
	return "OK", m.driver.HSet(key, values)
}

func (m *RedisModule) redisHGetAll(params ...any) (any, error) {
	if len(params) < 1 {
		return nil, fmt.Errorf("redis.hgetall: key is required")
	}
	key, _ := params[0].(string)
	if err := m.validateKey(key); err != nil {
		return nil, err
	}
	
	key = m.prefixKey(key)
	hash, err := m.driver.HGetAll(key)
	if err != nil {
		return nil, err
	}
	
	result := make(map[string]any)
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
	
	var values []string
	for i := 1; i < len(params); i++ {
		val, err := m.autoSerialize(params[i])
		if err != nil {
			return nil, err
		}
		values = append(values, val)
	}
	
	key = m.prefixKey(key)
	return m.driver.LPush(key, values...)
}

func (m *RedisModule) redisRPush(params ...any) (any, error) {
	if len(params) < 2 {
		return nil, fmt.Errorf("redis.rpush: key and value are required")
	}
	key, _ := params[0].(string)
	if err := m.validateKey(key); err != nil {
		return nil, err
	}
	
	var values []string
	for i := 1; i < len(params); i++ {
		val, err := m.autoSerialize(params[i])
		if err != nil {
			return nil, err
		}
		values = append(values, val)
	}
	
	key = m.prefixKey(key)
	return m.driver.RPush(key, values...)
}

func (m *RedisModule) redisLRange(params ...any) (any, error) {
	if len(params) < 3 {
		return nil, fmt.Errorf("redis.lrange: key, start, and stop are required")
	}
	key, _ := params[0].(string)
	if err := m.validateKey(key); err != nil {
		return nil, err
	}
	
	start := int64(0)
	stop := int64(-1)
	if startFloat, ok := params[1].(float64); ok {
		start = int64(startFloat)
	} else if startInt, ok := params[1].(int); ok {
		start = int64(startInt)
	}
	if stopFloat, ok := params[2].(float64); ok {
		stop = int64(stopFloat)
	} else if stopInt, ok := params[2].(int); ok {
		stop = int64(stopInt)
	}
	
	key = m.prefixKey(key)
	values, err := m.driver.LRange(key, start, stop)
	if err != nil {
		return nil, err
	}
	
	var result []any
	for _, v := range values {
		result = append(result, m.autoDeserialize(v))
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
	
	var members []string
	for i := 1; i < len(params); i++ {
		val, err := m.autoSerialize(params[i])
		if err != nil {
			return nil, err
		}
		members = append(members, val)
	}
	
	key = m.prefixKey(key)
	return m.driver.SAdd(key, members...)
}

func (m *RedisModule) redisSMembers(params ...any) (any, error) {
	if len(params) < 1 {
		return nil, fmt.Errorf("redis.smembers: key is required")
	}
	key, _ := params[0].(string)
	if err := m.validateKey(key); err != nil {
		return nil, err
	}
	
	key = m.prefixKey(key)
	members, err := m.driver.SMembers(key)
	if err != nil {
		return nil, err
	}
	
	var result []any
	for _, member := range members {
		result = append(result, m.autoDeserialize(member))
	}
	return result, nil
}

func (m *RedisModule) redisPublish(params ...any) (any, error) {
	if len(params) < 2 {
		return nil, fmt.Errorf("redis.publish: channel and message are required")
	}
	channel, _ := params[0].(string)
	
	message, err := m.autoSerialize(params[1])
	if err != nil {
		return nil, err
	}
	
	return "OK", m.driver.Publish(channel, message)
}
