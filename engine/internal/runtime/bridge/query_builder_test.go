package bridge

import (
	"fmt"
	"testing"
)

type testModelHandle struct {
	searchCalled bool
	searchOpts   SearchOptions
	searchResult []map[string]any
	searchErr    error
	getCalled    bool
	getID        string
	getResult    map[string]any
	getErr       error
	writeCalled  bool
	writeID      string
	writeData    map[string]any
	writeErr     error
	deleteCalled bool
	deleteID     string
	deleteErr    error
	countResult  int64
	countErr     error
	sumResult    float64
	sumErr       error
}

func (m *testModelHandle) Search(opts SearchOptions) ([]map[string]any, error) {
	m.searchCalled = true
	m.searchOpts = opts
	return m.searchResult, m.searchErr
}
func (m *testModelHandle) Get(id string, opts ...GetOptions) (map[string]any, error) {
	m.getCalled = true
	m.getID = id
	return m.getResult, m.getErr
}
func (m *testModelHandle) Create(data map[string]any) (map[string]any, error) {
	data["id"] = "new-1"
	return data, nil
}
func (m *testModelHandle) Write(id string, data map[string]any) error {
	m.writeCalled = true
	m.writeID = id
	m.writeData = data
	return m.writeErr
}
func (m *testModelHandle) Delete(id string) error {
	m.deleteCalled = true
	m.deleteID = id
	return m.deleteErr
}
func (m *testModelHandle) Count(opts SearchOptions) (int64, error) {
	m.searchOpts = opts
	return m.countResult, m.countErr
}
func (m *testModelHandle) Sum(field string, opts SearchOptions) (float64, error) {
	m.searchOpts = opts
	return m.sumResult, m.sumErr
}
func (m *testModelHandle) Upsert(data map[string]any, uniqueFields []string) (map[string]any, error) {
	return data, nil
}
func (m *testModelHandle) CreateMany(records []map[string]any) ([]map[string]any, error) {
	return records, nil
}
func (m *testModelHandle) WriteMany(ids []string, data map[string]any) (*BulkResult, error) {
	return &BulkResult{Affected: int64(len(ids))}, nil
}
func (m *testModelHandle) DeleteMany(ids []string) (*BulkResult, error) {
	return &BulkResult{Affected: int64(len(ids))}, nil
}
func (m *testModelHandle) UpsertMany(records []map[string]any, uniqueFields []string) ([]map[string]any, error) {
	return records, nil
}
func (m *testModelHandle) AddRelation(id, field string, relatedIDs []string) error    { return nil }
func (m *testModelHandle) RemoveRelation(id, field string, relatedIDs []string) error { return nil }
func (m *testModelHandle) SetRelation(id, field string, relatedIDs []string) error    { return nil }
func (m *testModelHandle) LoadRelation(id, field string) ([]map[string]any, error)    { return nil, nil }
func (m *testModelHandle) MorphAttach(id, relation string, relatedIDs []string) error { return nil }
func (m *testModelHandle) MorphDetach(id, relation string, relatedIDs []string) error { return nil }
func (m *testModelHandle) MorphSync(id, relation string, relatedIDs []string) error   { return nil }
func (m *testModelHandle) Sudo() SudoModelHandle                                     { return nil }

func TestQueryBuilder_Where_TwoArgs(t *testing.T) {
	mock := &testModelHandle{searchResult: []map[string]any{{"id": "1", "name": "Alice"}}}
	proxy := buildFluentModelProxy(mock)

	whereProxy := proxy["where"].(func(...any) any)("active", true)
	result, err := whereProxy.(map[string]any)["get"].(func() (any, error))()

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !mock.searchCalled {
		t.Fatal("expected Search to be called")
	}
	if len(mock.searchOpts.Domain) != 1 {
		t.Fatalf("expected 1 condition, got %d", len(mock.searchOpts.Domain))
	}
	cond := mock.searchOpts.Domain[0]
	if cond[0] != "active" || cond[1] != "=" || cond[2] != true {
		t.Errorf("expected [active = true], got %v", cond)
	}
	if result == nil {
		t.Error("expected non-nil result")
	}
}

func TestQueryBuilder_Where_ThreeArgs(t *testing.T) {
	mock := &testModelHandle{searchResult: []map[string]any{}}
	proxy := buildFluentModelProxy(mock)

	whereProxy := proxy["where"].(func(...any) any)("age", ">", 18)
	_, err := whereProxy.(map[string]any)["get"].(func() (any, error))()

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(mock.searchOpts.Domain) != 1 {
		t.Fatalf("expected 1 condition, got %d", len(mock.searchOpts.Domain))
	}
	cond := mock.searchOpts.Domain[0]
	if cond[0] != "age" || cond[1] != ">" || cond[2] != 18 {
		t.Errorf("expected [age > 18], got %v", cond)
	}
}

func TestQueryBuilder_Where_NilValue(t *testing.T) {
	mock := &testModelHandle{searchResult: []map[string]any{}}
	proxy := buildFluentModelProxy(mock)

	whereProxy := proxy["where"].(func(...any) any)("deleted_at", nil)
	_, err := whereProxy.(map[string]any)["get"].(func() (any, error))()

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	cond := mock.searchOpts.Domain[0]
	if cond[0] != "deleted_at" || cond[1] != "=" || cond[2] != nil {
		t.Errorf("expected [deleted_at = nil], got %v", cond)
	}
}

func TestQueryBuilder_Chaining(t *testing.T) {
	mock := &testModelHandle{searchResult: []map[string]any{}}
	proxy := buildFluentModelProxy(mock)

	chain := proxy["where"].(func(...any) any)("active", true)
	chain = chain.(map[string]any)["where"].(func(...any) any)("tier", "=", "vip")
	chain = chain.(map[string]any)["limit"].(func(any) any)(5)
	chain = chain.(map[string]any)["orderBy"].(func(...any) any)("name")
	_, err := chain.(map[string]any)["get"].(func() (any, error))()

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(mock.searchOpts.Domain) != 2 {
		t.Fatalf("expected 2 conditions, got %d", len(mock.searchOpts.Domain))
	}
	if mock.searchOpts.Limit != 5 {
		t.Errorf("expected limit=5, got %d", mock.searchOpts.Limit)
	}
	if mock.searchOpts.Order != "name" {
		t.Errorf("expected order='name', got %q", mock.searchOpts.Order)
	}
}

func TestQueryBuilder_OrderBy_WithDirection(t *testing.T) {
	mock := &testModelHandle{searchResult: []map[string]any{}}
	proxy := buildFluentModelProxy(mock)

	chain := proxy["orderBy"].(func(...any) any)("name", "desc")
	_, err := chain.(map[string]any)["get"].(func() (any, error))()

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if mock.searchOpts.Order != "name desc" {
		t.Errorf("expected order='name desc', got %q", mock.searchOpts.Order)
	}
}

func TestQueryBuilder_Offset(t *testing.T) {
	mock := &testModelHandle{searchResult: []map[string]any{}}
	proxy := buildFluentModelProxy(mock)

	chain := proxy["where"].(func(...any) any)("active", true)
	chain = chain.(map[string]any)["offset"].(func(any) any)(20)
	chain = chain.(map[string]any)["limit"].(func(any) any)(10)
	_, err := chain.(map[string]any)["get"].(func() (any, error))()

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if mock.searchOpts.Offset != 20 {
		t.Errorf("expected offset=20, got %d", mock.searchOpts.Offset)
	}
	if mock.searchOpts.Limit != 10 {
		t.Errorf("expected limit=10, got %d", mock.searchOpts.Limit)
	}
}

func TestQueryBuilder_Get_EmptyResult(t *testing.T) {
	mock := &testModelHandle{searchResult: nil}
	proxy := buildFluentModelProxy(mock)

	chain := proxy["where"].(func(...any) any)("active", true)
	result, err := chain.(map[string]any)["get"].(func() (any, error))()

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	arr, ok := result.([]any)
	if !ok {
		t.Fatalf("expected []any for empty result, got %T", result)
	}
	if len(arr) != 0 {
		t.Errorf("expected empty array, got %d items", len(arr))
	}
}

func TestQueryBuilder_First(t *testing.T) {
	mock := &testModelHandle{searchResult: []map[string]any{{"id": "1", "name": "Alice"}}}
	proxy := buildFluentModelProxy(mock)

	chain := proxy["where"].(func(...any) any)("email", "alice@test.com")
	result, err := chain.(map[string]any)["first"].(func() (any, error))()

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if mock.searchOpts.Limit != 1 {
		t.Errorf("expected limit=1 for first(), got %d", mock.searchOpts.Limit)
	}
}

func TestQueryBuilder_First_NoResult(t *testing.T) {
	mock := &testModelHandle{searchResult: []map[string]any{}}
	proxy := buildFluentModelProxy(mock)

	chain := proxy["where"].(func(...any) any)("email", "nobody@test.com")
	result, err := chain.(map[string]any)["first"].(func() (any, error))()

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != nil {
		t.Errorf("expected nil for no match, got %v", result)
	}
}

func TestQueryBuilder_Count(t *testing.T) {
	mock := &testModelHandle{countResult: 42}
	proxy := buildFluentModelProxy(mock)

	chain := proxy["where"].(func(...any) any)("active", true)
	result, err := chain.(map[string]any)["count"].(func() (any, error))()

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != int64(42) {
		t.Errorf("expected 42, got %v", result)
	}
}

func TestQueryBuilder_Sum(t *testing.T) {
	mock := &testModelHandle{sumResult: 1500.50}
	proxy := buildFluentModelProxy(mock)

	chain := proxy["where"].(func(...any) any)("status", "paid")
	result, err := chain.(map[string]any)["sum"].(func(...any) (any, error))("amount")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != 1500.50 {
		t.Errorf("expected 1500.50, got %v", result)
	}
}

func TestQueryBuilder_Sum_NoField(t *testing.T) {
	mock := &testModelHandle{}
	proxy := buildFluentModelProxy(mock)

	chain := proxy["where"].(func(...any) any)("active", true)
	_, err := chain.(map[string]any)["sum"].(func(...any) (any, error))()

	if err == nil {
		t.Fatal("expected error for sum without field")
	}
}

func TestQueryBuilder_Find_Get(t *testing.T) {
	mock := &testModelHandle{getResult: map[string]any{"id": "123", "name": "Bob"}}
	proxy := buildFluentModelProxy(mock)

	recordProxy := proxy["find"].(func(any) any)("123")
	result, err := recordProxy.(map[string]any)["get"].(func() (any, error))()

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !mock.getCalled {
		t.Fatal("expected Get to be called")
	}
	if mock.getID != "123" {
		t.Errorf("expected id='123', got %q", mock.getID)
	}
	if result == nil {
		t.Error("expected non-nil result")
	}
}

func TestQueryBuilder_Find_NumericID(t *testing.T) {
	mock := &testModelHandle{getResult: map[string]any{"id": "456"}}
	proxy := buildFluentModelProxy(mock)

	recordProxy := proxy["find"].(func(any) any)(456)
	_, err := recordProxy.(map[string]any)["get"].(func() (any, error))()

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if mock.getID != "456" {
		t.Errorf("expected id='456' (converted from int), got %q", mock.getID)
	}
}

func TestQueryBuilder_Find_Update(t *testing.T) {
	mock := &testModelHandle{}
	proxy := buildFluentModelProxy(mock)

	recordProxy := proxy["find"].(func(any) any)("ord-123")
	_, err := recordProxy.(map[string]any)["update"].(func(map[string]any) (any, error))(map[string]any{"status": "shipped"})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !mock.writeCalled {
		t.Fatal("expected Write to be called")
	}
	if mock.writeID != "ord-123" {
		t.Errorf("expected id='ord-123', got %q", mock.writeID)
	}
	if mock.writeData["status"] != "shipped" {
		t.Errorf("expected status='shipped', got %v", mock.writeData["status"])
	}
}

func TestQueryBuilder_Find_Delete(t *testing.T) {
	mock := &testModelHandle{}
	proxy := buildFluentModelProxy(mock)

	recordProxy := proxy["find"].(func(any) any)("ord-123")
	_, err := recordProxy.(map[string]any)["delete"].(func() (any, error))()

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !mock.deleteCalled {
		t.Fatal("expected Delete to be called")
	}
	if mock.deleteID != "ord-123" {
		t.Errorf("expected id='ord-123', got %q", mock.deleteID)
	}
}

func TestQueryBuilder_SearchError(t *testing.T) {
	mock := &testModelHandle{searchErr: fmt.Errorf("db connection lost")}
	proxy := buildFluentModelProxy(mock)

	chain := proxy["where"].(func(...any) any)("active", true)
	_, err := chain.(map[string]any)["get"].(func() (any, error))()

	if err == nil {
		t.Fatal("expected error to propagate")
	}
	if err.Error() != "db connection lost" {
		t.Errorf("expected 'db connection lost', got %q", err.Error())
	}
}

func TestToIntSafe(t *testing.T) {
	tests := []struct {
		input    any
		expected int
	}{
		{5, 5},
		{float64(10), 10},
		{int64(20), 20},
		{int32(30), 30},
		{float32(40), 40},
		{"invalid", 0},
		{nil, 0},
	}
	for _, tt := range tests {
		result := toIntSafe(tt.input)
		if result != tt.expected {
			t.Errorf("toIntSafe(%v) = %d, want %d", tt.input, result, tt.expected)
		}
	}
}
