package io

import (
	"encoding/json"
	"fmt"
	"strings"
	"sync"
)

// MongoDriver is the interface for MongoDB operations.
// Implement this with a real driver (go.mongodb.org/mongo-driver/v2) for production.
type MongoDriver interface {
	Find(db, collection string, filter map[string]any, opts map[string]any) ([]map[string]any, error)
	FindOne(db, collection string, filter map[string]any) (map[string]any, error)
	Insert(db, collection string, doc map[string]any) (map[string]any, error)
	InsertMany(db, collection string, docs []map[string]any) (map[string]any, error)
	Update(db, collection string, filter, update map[string]any) (map[string]any, error)
	Delete(db, collection string, filter map[string]any) (map[string]any, error)
	Count(db, collection string, filter map[string]any) (int64, error)
	Aggregate(db, collection string, pipeline []any) ([]map[string]any, error)
	Close() error
}

// InMemoryMongoDriver is a simple in-memory implementation for testing and development.
type InMemoryMongoDriver struct {
	mu          sync.Mutex
	collections map[string][]map[string]any // key: "db.collection"
	nextID      int
}

// NewInMemoryMongoDriver creates a new in-memory MongoDB driver.
func NewInMemoryMongoDriver() *InMemoryMongoDriver {
	return &InMemoryMongoDriver{
		collections: make(map[string][]map[string]any),
	}
}

func (d *InMemoryMongoDriver) collectionKey(db, collection string) string {
	return db + "." + collection
}

func (d *InMemoryMongoDriver) matchesFilter(doc, filter map[string]any) bool {
	if filter == nil || len(filter) == 0 {
		return true
	}
	for key, filterVal := range filter {
		docVal, exists := doc[key]
		if !exists {
			return false
		}
		
		// Handle MongoDB operators
		if filterMap, ok := filterVal.(map[string]any); ok {
			for op, opVal := range filterMap {
				switch op {
				case "$gt":
					if !d.compareValues(docVal, opVal, ">") {
						return false
					}
				case "$gte":
					if !d.compareValues(docVal, opVal, ">=") {
						return false
					}
				case "$lt":
					if !d.compareValues(docVal, opVal, "<") {
						return false
					}
				case "$lte":
					if !d.compareValues(docVal, opVal, "<=") {
						return false
					}
				case "$ne":
					if d.valuesEqual(docVal, opVal) {
						return false
					}
				case "$in":
					if arr, ok := opVal.([]any); ok {
						found := false
						for _, v := range arr {
							if d.valuesEqual(docVal, v) {
								found = true
								break
							}
						}
						if !found {
							return false
						}
					}
				}
			}
		} else {
			// Simple equality
			if !d.valuesEqual(docVal, filterVal) {
				return false
			}
		}
	}
	return true
}

func (d *InMemoryMongoDriver) valuesEqual(a, b any) bool {
	aJSON, _ := json.Marshal(a)
	bJSON, _ := json.Marshal(b)
	return string(aJSON) == string(bJSON)
}

func (d *InMemoryMongoDriver) compareValues(a, b any, op string) bool {
	aFloat, aOk := d.toFloat(a)
	bFloat, bOk := d.toFloat(b)
	if !aOk || !bOk {
		return false
	}
	switch op {
	case ">":
		return aFloat > bFloat
	case ">=":
		return aFloat >= bFloat
	case "<":
		return aFloat < bFloat
	case "<=":
		return aFloat <= bFloat
	}
	return false
}

func (d *InMemoryMongoDriver) toFloat(v any) (float64, bool) {
	switch val := v.(type) {
	case float64:
		return val, true
	case int:
		return float64(val), true
	case int64:
		return float64(val), true
	case float32:
		return float64(val), true
	}
	return 0, false
}

func (d *InMemoryMongoDriver) Find(db, collection string, filter map[string]any, opts map[string]any) ([]map[string]any, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	
	key := d.collectionKey(db, collection)
	docs := d.collections[key]
	
	var result []map[string]any
	for _, doc := range docs {
		if d.matchesFilter(doc, filter) {
			result = append(result, d.copyDoc(doc))
		}
	}
	
	return result, nil
}

func (d *InMemoryMongoDriver) FindOne(db, collection string, filter map[string]any) (map[string]any, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	
	key := d.collectionKey(db, collection)
	docs := d.collections[key]
	
	for _, doc := range docs {
		if d.matchesFilter(doc, filter) {
			return d.copyDoc(doc), nil
		}
	}
	
	return nil, fmt.Errorf("document not found")
}

func (d *InMemoryMongoDriver) Insert(db, collection string, doc map[string]any) (map[string]any, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	
	key := d.collectionKey(db, collection)
	
	// Generate _id if not present
	if _, hasID := doc["_id"]; !hasID {
		d.nextID++
		doc["_id"] = d.nextID
	}
	
	newDoc := d.copyDoc(doc)
	d.collections[key] = append(d.collections[key], newDoc)
	
	return map[string]any{"inserted_id": newDoc["_id"]}, nil
}

func (d *InMemoryMongoDriver) InsertMany(db, collection string, docs []map[string]any) (map[string]any, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	
	key := d.collectionKey(db, collection)
	var insertedIDs []any
	
	for _, doc := range docs {
		if _, hasID := doc["_id"]; !hasID {
			d.nextID++
			doc["_id"] = d.nextID
		}
		newDoc := d.copyDoc(doc)
		d.collections[key] = append(d.collections[key], newDoc)
		insertedIDs = append(insertedIDs, newDoc["_id"])
	}
	
	return map[string]any{"inserted_ids": insertedIDs}, nil
}

func (d *InMemoryMongoDriver) Update(db, collection string, filter, update map[string]any) (map[string]any, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	
	key := d.collectionKey(db, collection)
	docs := d.collections[key]
	
	modified := int64(0)
	for i, doc := range docs {
		if d.matchesFilter(doc, filter) {
			// Apply $set updates
			if setOps, ok := update["$set"].(map[string]any); ok {
				for k, v := range setOps {
					doc[k] = v
				}
				docs[i] = doc
				modified++
			}
		}
	}
	
	return map[string]any{"modified_count": modified}, nil
}

func (d *InMemoryMongoDriver) Delete(db, collection string, filter map[string]any) (map[string]any, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	
	key := d.collectionKey(db, collection)
	docs := d.collections[key]
	
	var remaining []map[string]any
	deleted := 0
	for _, doc := range docs {
		if d.matchesFilter(doc, filter) {
			deleted++
		} else {
			remaining = append(remaining, doc)
		}
	}
	
	d.collections[key] = remaining
	return map[string]any{"deleted_count": deleted}, nil
}

func (d *InMemoryMongoDriver) Count(db, collection string, filter map[string]any) (int64, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	
	key := d.collectionKey(db, collection)
	docs := d.collections[key]
	
	count := int64(0)
	for _, doc := range docs {
		if d.matchesFilter(doc, filter) {
			count++
		}
	}
	
	return count, nil
}

func (d *InMemoryMongoDriver) Aggregate(db, collection string, pipeline []any) ([]map[string]any, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	
	key := d.collectionKey(db, collection)
	docs := d.collections[key]
	
	// Basic support: just return all documents
	var result []map[string]any
	for _, doc := range docs {
		result = append(result, d.copyDoc(doc))
	}
	
	return result, nil
}

func (d *InMemoryMongoDriver) Close() error {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.collections = make(map[string][]map[string]any)
	return nil
}

func (d *InMemoryMongoDriver) copyDoc(doc map[string]any) map[string]any {
	result := make(map[string]any)
	for k, v := range doc {
		result[k] = v
	}
	return result
}

// MongoModule provides MongoDB functions for go-json programs.
type MongoModule struct {
	security *SecurityConfig
	config   map[string]any
	driver   MongoDriver
	mu       sync.Mutex
}

// MongoOption is a functional option for MongoModule.
type MongoOption func(*MongoModule)

// WithMongoDriver sets a custom MongoDB driver.
func WithMongoDriver(d MongoDriver) MongoOption {
	return func(m *MongoModule) { m.driver = d }
}

// NewMongoModule creates a new MongoDB I/O module.
func NewMongoModule(security *SecurityConfig, opts ...MongoOption) *MongoModule {
	if security == nil {
		security = DefaultSecurityConfig()
	}
	m := &MongoModule{
		security: security,
	}
	for _, opt := range opts {
		opt(m)
	}
	if m.driver == nil {
		m.driver = NewInMemoryMongoDriver()
	}
	return m
}

func (m *MongoModule) Name() string { return "mongo" }

func (m *MongoModule) SetConfig(cfg map[string]any) { m.config = cfg }

func (m *MongoModule) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.driver != nil {
		return m.driver.Close()
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
	db, coll, err := m.parseCollectionParams(params)
	if err != nil {
		return nil, err
	}

	if err := m.validateDatabase(db); err != nil {
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

	opts := m.getOptsFromParams(params)
	docs, err := m.driver.Find(db, coll, filter, opts)
	if err != nil {
		return nil, err
	}
	
	result := make([]any, len(docs))
	for i, doc := range docs {
		result[i] = doc
	}
	return result, nil
}

func (m *MongoModule) mongoFindOne(params ...any) (any, error) {
	db, coll, err := m.parseCollectionParams(params)
	if err != nil {
		return nil, err
	}

	if err := m.validateDatabase(db); err != nil {
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

	return m.driver.FindOne(db, coll, filter)
}

func (m *MongoModule) mongoInsert(params ...any) (any, error) {
	db, coll, err := m.parseCollectionParams(params)
	if err != nil {
		return nil, err
	}

	if err := m.validateDatabase(db); err != nil {
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

	return m.driver.Insert(db, coll, doc)
}

func (m *MongoModule) mongoInsertMany(params ...any) (any, error) {
	db, coll, err := m.parseCollectionParams(params)
	if err != nil {
		return nil, err
	}

	if err := m.validateDatabase(db); err != nil {
		return nil, err
	}

	var docs []map[string]any
	if len(params) > 1 {
		if docsAny, ok := params[1].([]any); ok {
			for _, d := range docsAny {
				if docMap, ok := d.(map[string]any); ok {
					docs = append(docs, docMap)
				}
			}
		}
	}

	return m.driver.InsertMany(db, coll, docs)
}

func (m *MongoModule) mongoUpdate(params ...any) (any, error) {
	db, coll, err := m.parseCollectionParams(params)
	if err != nil {
		return nil, err
	}

	if err := m.validateDatabase(db); err != nil {
		return nil, err
	}

	var filter, update map[string]any
	if len(params) > 1 {
		filter, _ = params[1].(map[string]any)
	}
	if len(params) > 2 {
		update, _ = params[2].(map[string]any)
	}

	if filter != nil {
		if err := m.validateOperation(filter); err != nil {
			return nil, err
		}
	}

	return m.driver.Update(db, coll, filter, update)
}

func (m *MongoModule) mongoDelete(params ...any) (any, error) {
	db, coll, err := m.parseCollectionParams(params)
	if err != nil {
		return nil, err
	}

	if err := m.validateDatabase(db); err != nil {
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

	return m.driver.Delete(db, coll, filter)
}

func (m *MongoModule) mongoCount(params ...any) (any, error) {
	db, coll, err := m.parseCollectionParams(params)
	if err != nil {
		return nil, err
	}

	if err := m.validateDatabase(db); err != nil {
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

	return m.driver.Count(db, coll, filter)
}

func (m *MongoModule) mongoAggregate(params ...any) (any, error) {
	db, coll, err := m.parseCollectionParams(params)
	if err != nil {
		return nil, err
	}

	if err := m.validateDatabase(db); err != nil {
		return nil, err
	}

	var pipeline []any
	if len(params) > 1 {
		pipeline, _ = params[1].([]any)
	}

	if pipeline != nil {
		if err := m.validateOperation(map[string]any{"pipeline": pipeline}); err != nil {
			return nil, err
		}
	}

	docs, err := m.driver.Aggregate(db, coll, pipeline)
	if err != nil {
		return nil, err
	}
	
	result := make([]any, len(docs))
	for i, doc := range docs {
		result[i] = doc
	}
	return result, nil
}
