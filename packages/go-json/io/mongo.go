package io

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

// MongoModule provides MongoDB functions for go-json programs.
type MongoModule struct {
	security *SecurityConfig
	config   map[string]any
	client   *mongo.Client
	mu       sync.Mutex
	closed   bool
}

// NewMongoModule creates a new MongoDB I/O module.
// Connection is lazy — established on first operation using security.Mongo.DefaultURI.
func NewMongoModule(security *SecurityConfig) *MongoModule {
	if security == nil {
		security = DefaultSecurityConfig()
	}
	return &MongoModule{
		security: security,
	}
}

func (m *MongoModule) Name() string { return "mongo" }

func (m *MongoModule) SetConfig(cfg map[string]any) { m.config = cfg }

func (m *MongoModule) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.closed = true
	if m.client != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		err := m.client.Disconnect(ctx)
		m.client = nil
		return err
	}
	return nil
}

func (m *MongoModule) getClient() (*mongo.Client, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.closed {
		return nil, fmt.Errorf("mongo: module is closed")
	}
	if m.client != nil {
		return m.client, nil
	}

	uri := m.security.Mongo.DefaultURI
	if uri == "" {
		uri = "mongodb://localhost:27017"
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	client, err := mongo.Connect(options.Client().ApplyURI(uri))
	if err != nil {
		return nil, fmt.Errorf("mongo: connection failed: %s", err.Error())
	}

	if err := client.Ping(ctx, nil); err != nil {
		client.Disconnect(ctx)
		return nil, fmt.Errorf("mongo: ping failed: %s", err.Error())
	}

	m.client = client
	return m.client, nil
}

func (m *MongoModule) getCollection(params []any) (*mongo.Collection, string, error) {
	if len(params) < 1 {
		return nil, "", fmt.Errorf("mongo: collection is required")
	}
	collection, ok := params[0].(string)
	if !ok {
		return nil, "", fmt.Errorf("mongo: collection must be a string")
	}

	db := "default"
	coll := collection
	parts := strings.SplitN(collection, ".", 2)
	if len(parts) == 2 {
		db = parts[0]
		coll = parts[1]
	}

	if err := m.validateDatabase(db); err != nil {
		return nil, "", err
	}

	client, err := m.getClient()
	if err != nil {
		return nil, "", err
	}

	return client.Database(db).Collection(coll), coll, nil
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

func toBsonDoc(m map[string]any) bson.D {
	if m == nil {
		return bson.D{}
	}
	doc := bson.D{}
	for k, v := range m {
		doc = append(doc, bson.E{Key: k, Value: v})
	}
	return doc
}

func (m *MongoModule) mongoFind(params ...any) (any, error) {
	coll, _, err := m.getCollection(params)
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

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	findOpts := options.Find()
	if len(params) > 2 {
		if opts, ok := params[2].(map[string]any); ok {
			if limit, ok := opts["limit"]; ok {
				if lf, ok := toFloat64Val(limit); ok {
					findOpts.SetLimit(int64(lf))
				}
			}
			if skip, ok := opts["skip"]; ok {
				if sf, ok := toFloat64Val(skip); ok {
					findOpts.SetSkip(int64(sf))
				}
			}
		}
	}

	maxResults := m.security.Mongo.MaxResults
	if maxResults > 0 {
		findOpts.SetLimit(int64(maxResults))
	}

	cursor, err := coll.Find(ctx, toBsonDoc(filter), findOpts)
	if err != nil {
		return nil, fmt.Errorf("mongo.find: %s", err.Error())
	}
	defer cursor.Close(ctx)

	var results []map[string]any
	if err := cursor.All(ctx, &results); err != nil {
		return nil, fmt.Errorf("mongo.find: %s", err.Error())
	}
	if results == nil {
		results = []map[string]any{}
	}
	return results, nil
}

func (m *MongoModule) mongoFindOne(params ...any) (any, error) {
	coll, _, err := m.getCollection(params)
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

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	var result map[string]any
	err = coll.FindOne(ctx, toBsonDoc(filter)).Decode(&result)
	if err == mongo.ErrNoDocuments {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("mongo.findOne: %s", err.Error())
	}
	return result, nil
}

func (m *MongoModule) mongoInsert(params ...any) (any, error) {
	coll, _, err := m.getCollection(params)
	if err != nil {
		return nil, err
	}

	var doc map[string]any
	if len(params) > 1 {
		doc, _ = params[1].(map[string]any)
	}
	if doc == nil {
		return nil, fmt.Errorf("mongo.insert: document is required")
	}

	if m.security.Mongo.MaxDocumentSize > 0 {
		data, _ := json.Marshal(doc)
		if int64(len(data)) > m.security.Mongo.MaxDocumentSize {
			return nil, fmt.Errorf("mongo.insert: document exceeds max size (%d bytes, max %d)", len(data), m.security.Mongo.MaxDocumentSize)
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	result, err := coll.InsertOne(ctx, doc)
	if err != nil {
		return nil, fmt.Errorf("mongo.insert: %s", err.Error())
	}
	return map[string]any{"inserted_id": result.InsertedID}, nil
}

func (m *MongoModule) mongoInsertMany(params ...any) (any, error) {
	coll, _, err := m.getCollection(params)
	if err != nil {
		return nil, err
	}

	var docs []any
	if len(params) > 1 {
		docs, _ = params[1].([]any)
	}
	if len(docs) == 0 {
		return nil, fmt.Errorf("mongo.insertMany: documents array is required")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	result, err := coll.InsertMany(ctx, docs)
	if err != nil {
		return nil, fmt.Errorf("mongo.insertMany: %s", err.Error())
	}
	return map[string]any{"inserted_ids": result.InsertedIDs}, nil
}

func (m *MongoModule) mongoUpdate(params ...any) (any, error) {
	coll, _, err := m.getCollection(params)
	if err != nil {
		return nil, err
	}

	var filter, update map[string]any
	if len(params) > 1 {
		filter, _ = params[1].(map[string]any)
	}
	if len(params) > 2 {
		update, _ = params[2].(map[string]any)
	}
	if update == nil {
		return nil, fmt.Errorf("mongo.update: update document is required")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	result, err := coll.UpdateMany(ctx, toBsonDoc(filter), toBsonDoc(update))
	if err != nil {
		return nil, fmt.Errorf("mongo.update: %s", err.Error())
	}
	return map[string]any{"modified_count": result.ModifiedCount}, nil
}

func (m *MongoModule) mongoDelete(params ...any) (any, error) {
	coll, _, err := m.getCollection(params)
	if err != nil {
		return nil, err
	}

	var filter map[string]any
	if len(params) > 1 {
		filter, _ = params[1].(map[string]any)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	result, err := coll.DeleteMany(ctx, toBsonDoc(filter))
	if err != nil {
		return nil, fmt.Errorf("mongo.delete: %s", err.Error())
	}
	return map[string]any{"deleted_count": result.DeletedCount}, nil
}

func (m *MongoModule) mongoCount(params ...any) (any, error) {
	coll, _, err := m.getCollection(params)
	if err != nil {
		return nil, err
	}

	var filter map[string]any
	if len(params) > 1 {
		filter, _ = params[1].(map[string]any)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	count, err := coll.CountDocuments(ctx, toBsonDoc(filter))
	if err != nil {
		return nil, fmt.Errorf("mongo.count: %s", err.Error())
	}
	return count, nil
}

func (m *MongoModule) mongoAggregate(params ...any) (any, error) {
	coll, _, err := m.getCollection(params)
	if err != nil {
		return nil, err
	}

	var pipeline []any
	if len(params) > 1 {
		pipeline, _ = params[1].([]any)
	}
	if pipeline == nil {
		return nil, fmt.Errorf("mongo.aggregate: pipeline is required")
	}

	if err := m.validateOperation(map[string]any{"pipeline": pipeline}); err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	cursor, err := coll.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, fmt.Errorf("mongo.aggregate: %s", err.Error())
	}
	defer cursor.Close(ctx)

	var results []map[string]any
	if err := cursor.All(ctx, &results); err != nil {
		return nil, fmt.Errorf("mongo.aggregate: %s", err.Error())
	}
	if results == nil {
		results = []map[string]any{}
	}
	return results, nil
}

// SetClient allows injecting a pre-configured mongo client (for testing).
func (m *MongoModule) SetClient(client *mongo.Client) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.client = client
}
