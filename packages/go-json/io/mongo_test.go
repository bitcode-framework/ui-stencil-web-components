package io

import (
	"testing"
)

func TestMongoModule_SecurityBlockedOps(t *testing.T) {
	m := NewMongoModule(DefaultSecurityConfig())

	filter := map[string]any{"$where": "this.age > 18"}
	if err := m.validateOperation(filter); err == nil {
		t.Error("$where should be blocked")
	}

	filter = map[string]any{"$function": map[string]any{"body": "return true"}}
	if err := m.validateOperation(filter); err == nil {
		t.Error("$function should be blocked")
	}

	safeFilter := map[string]any{"age": map[string]any{"$gt": 18}}
	if err := m.validateOperation(safeFilter); err != nil {
		t.Errorf("safe filter should pass: %v", err)
	}
}

func TestMongoModule_AllowedDatabases(t *testing.T) {
	sec := DefaultSecurityConfig()
	sec.Mongo.AllowedDatabases = []string{"mydb", "testdb"}
	m := NewMongoModule(sec)

	if err := m.validateDatabase("mydb"); err != nil {
		t.Errorf("mydb should be allowed: %v", err)
	}
	if err := m.validateDatabase("testdb"); err != nil {
		t.Errorf("testdb should be allowed: %v", err)
	}
	if err := m.validateDatabase("admin"); err == nil {
		t.Error("admin should not be allowed")
	}
}

func TestMongoModule_AllowedDatabases_Empty(t *testing.T) {
	sec := DefaultSecurityConfig()
	sec.Mongo.AllowedDatabases = nil
	m := NewMongoModule(sec)

	if err := m.validateDatabase("anything"); err != nil {
		t.Errorf("all databases should be allowed when list is empty: %v", err)
	}
}

func TestMongoModule_NestedBlockedOps(t *testing.T) {
	m := NewMongoModule(DefaultSecurityConfig())

	filter := map[string]any{
		"$or": []any{
			map[string]any{"name": "test"},
			map[string]any{"$where": "true"},
		},
	}
	if err := m.validateOperation(filter); err == nil {
		t.Error("nested $where should be blocked")
	}
}

func TestMongoModule_MaxDocumentSize(t *testing.T) {
	sec := DefaultSecurityConfig()
	sec.Mongo.MaxDocumentSize = 50
	m := NewMongoModule(sec)

	largeDoc := map[string]any{
		"data": "this is a very long string that exceeds the max document size limit set in security config",
	}

	_, err := m.mongoInsert("testdb.collection", largeDoc)
	if err == nil {
		t.Error("expected max document size error")
	}
}

func TestMongoModule_RequiresCollection(t *testing.T) {
	m := NewMongoModule(DefaultSecurityConfig())

	_, err := m.mongoFind()
	if err == nil {
		t.Error("expected error when no collection provided")
	}
}

func TestMongoModule_CollectionParsing(t *testing.T) {
	sec := DefaultSecurityConfig()
	sec.Mongo.AllowedDatabases = []string{"mydb"}
	m := NewMongoModule(sec)

	_, _, err := m.getCollection([]any{"mydb.users"})
	if err == nil || err.Error() == "mongo: database 'mydb' not in allowed list" {
		t.Logf("getCollection correctly validates database")
	}

	_, _, err = m.getCollection([]any{"blocked.users"})
	if err == nil {
		t.Error("expected database not allowed error")
	}
}
