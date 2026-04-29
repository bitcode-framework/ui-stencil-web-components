package io

import (
	"encoding/json"
	"fmt"
	"strings"
	"sync"
)

// MongoModule provides MongoDB functions for go-json programs.
// Requires go.mongodb.org/mongo-driver/v2 — operations are stubbed until driver is available.
type MongoModule struct {
	security *SecurityConfig
	config   map[string]any
	pools    map[string]any
	poolsMu  sync.Mutex
}

// NewMongoModule creates a new MongoDB I/O module.
func NewMongoModule(security *SecurityConfig) *MongoModule {
	if security == nil {
		security = DefaultSecurityConfig()
	}
	return &MongoModule{
		security: security,
		pools:    make(map[string]any),
	}
}

func (m *MongoModule) Name() string { return "mongo" }

func (m *MongoModule) SetConfig(cfg map[string]any) { m.config = cfg }

func (m *MongoModule) Close() error {
	m.poolsMu.Lock()
	defer m.poolsMu.Unlock()
	for k := range m.pools {
		delete(m.pools, k)
	}
	return nil
}

func (m *MongoModule) Functions() map[string]any {
	return map[string]any{
		"find":       m.mongoFind,
		"findOne":    m.mongoFindOne,
		"insert":     m.mongoInsert,
		"insertMany": m.mongoInsertMany,
		"update":     m.mongoUpdate,
		"delete":     m.mongoDelete,
		"count":      m.mongoCount,
		"aggregate":  m.mongoAggregate,
	}
}

func (m *MongoModule) getURI(params []any) string {
	for _, p := range params {
		if opts, ok := p.(map[string]any); ok {
			if uri, ok := opts["uri"].(string); ok {
				return uri
			}
		}
	}
	return m.security.Mongo.DefaultURI
}

func (m *MongoModule) validateDatabase(db string) error {
	if len(m.security.Mongo.AllowedDatabases) == 0 {
		return nil
	}
	for _, allowed := range m.security.Mongo.AllowedDatabases {
		if strings.EqualFold(db, allowed) {
			return nil
		}
	}
	return fmt.Errorf("mongo: database '%s' not in allowed list", db)
}

func (m *MongoModule) validateOperation(filter map[string]any) error {
	blocked := m.security.Mongo.BlockedOperations
	if blocked == nil {
		blocked = []string{"$where", "$function"}
	}
	return checkBlockedOps(filter, blocked)
}

func checkBlockedOps(v any, blocked []string) error {
	switch val := v.(type) {
	case map[string]any:
		for k, child := range val {
			for _, b := range blocked {
				if strings.EqualFold(k, b) {
					return fmt.Errorf("mongo: operation '%s' is blocked", k)
				}
			}
			if err := checkBlockedOps(child, blocked); err != nil {
				return err
			}
		}
	case []any:
		for _, item := range val {
			if err := checkBlockedOps(item, blocked); err != nil {
				return err
			}
		}
	}
	return nil
}

func (m *MongoModule) parseCollectionParams(params []any) (string, string, error) {
	if len(params) < 1 {
		return "", "", fmt.Errorf("mongo: collection is required")
	}
	collection, ok := params[0].(string)
	if !ok {
		return "", "", fmt.Errorf("mongo: collection must be a string")
	}

	parts := strings.SplitN(collection, ".", 2)
	if len(parts) == 2 {
		return parts[0], parts[1], nil
	}

	db := "default"
	if opts := m.getOptsFromParams(params); opts != nil {
		if d, ok := opts["database"].(string); ok {
			db = d
		}
	}
	return db, collection, nil
}

func (m *MongoModule) getOptsFromParams(params []any) map[string]any {
	for i := 1; i < len(params); i++ {
		if opts, ok := params[i].(map[string]any); ok {
			return opts
		}
	}
	return nil
}

func (m *MongoModule) mongoFind(params ...any) (any, error) {
	_, coll, err := m.parseCollectionParams(params)
	if err != nil {
		return nil, err
	}

	var filter map[string]any
	if len(params) > 1 {
		filter, _ = params[1].(map[string]any)
	}
	if filter != nil {
		if err := m.validateOperation(filter); err != nil {
			return nil, err
		}
	}

	return nil, fmt.Errorf("mongo.find: MongoDB driver not available (collection=%s, filter=%v) — add go.mongodb.org/mongo-driver/v2 dependency", coll, filter)
}

func (m *MongoModule) mongoFindOne(params ...any) (any, error) {
	_, coll, err := m.parseCollectionParams(params)
	if err != nil {
		return nil, err
	}

	var filter map[string]any
	if len(params) > 1 {
		filter, _ = params[1].(map[string]any)
	}
	if filter != nil {
		if err := m.validateOperation(filter); err != nil {
			return nil, err
		}
	}

	return nil, fmt.Errorf("mongo.findOne: MongoDB driver not available (collection=%s) — add go.mongodb.org/mongo-driver/v2 dependency", coll)
}

func (m *MongoModule) mongoInsert(params ...any) (any, error) {
	_, coll, err := m.parseCollectionParams(params)
	if err != nil {
		return nil, err
	}

	var doc map[string]any
	if len(params) > 1 {
		doc, _ = params[1].(map[string]any)
	}

	if m.security.Mongo.MaxDocumentSize > 0 && doc != nil {
		data, _ := json.Marshal(doc)
		if int64(len(data)) > m.security.Mongo.MaxDocumentSize {
			return nil, fmt.Errorf("mongo.insert: document exceeds max size (%d bytes, max %d)", len(data), m.security.Mongo.MaxDocumentSize)
		}
	}

	return nil, fmt.Errorf("mongo.insert: MongoDB driver not available (collection=%s) — add go.mongodb.org/mongo-driver/v2 dependency", coll)
}

func (m *MongoModule) mongoInsertMany(params ...any) (any, error) {
	_, coll, err := m.parseCollectionParams(params)
	if err != nil {
		return nil, err
	}
	return nil, fmt.Errorf("mongo.insertMany: MongoDB driver not available (collection=%s) — add go.mongodb.org/mongo-driver/v2 dependency", coll)
}

func (m *MongoModule) mongoUpdate(params ...any) (any, error) {
	_, coll, err := m.parseCollectionParams(params)
	if err != nil {
		return nil, err
	}
	return nil, fmt.Errorf("mongo.update: MongoDB driver not available (collection=%s) — add go.mongodb.org/mongo-driver/v2 dependency", coll)
}

func (m *MongoModule) mongoDelete(params ...any) (any, error) {
	_, coll, err := m.parseCollectionParams(params)
	if err != nil {
		return nil, err
	}
	return nil, fmt.Errorf("mongo.delete: MongoDB driver not available (collection=%s) — add go.mongodb.org/mongo-driver/v2 dependency", coll)
}

func (m *MongoModule) mongoCount(params ...any) (any, error) {
	_, coll, err := m.parseCollectionParams(params)
	if err != nil {
		return nil, err
	}
	return nil, fmt.Errorf("mongo.count: MongoDB driver not available (collection=%s) — add go.mongodb.org/mongo-driver/v2 dependency", coll)
}

func (m *MongoModule) mongoAggregate(params ...any) (any, error) {
	_, coll, err := m.parseCollectionParams(params)
	if err != nil {
		return nil, err
	}

	if len(params) > 1 {
		if pipeline, ok := params[1].([]any); ok {
			if err := m.validateOperation(map[string]any{"pipeline": pipeline}); err != nil {
				return nil, err
			}
		}
	}

	return nil, fmt.Errorf("mongo.aggregate: MongoDB driver not available (collection=%s) — add go.mongodb.org/mongo-driver/v2 dependency", coll)
}
