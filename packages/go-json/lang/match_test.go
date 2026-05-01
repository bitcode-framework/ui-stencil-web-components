package lang

import (
	"strings"
	"testing"
)

func TestMatch_LiteralNumber(t *testing.T) {
	result := compileAndRun(t, `{
		"steps": [
			{"let": "resp", "value": {"status": 200, "body": "hello"}},
			{"match": "resp.status", "cases": [
				{"pattern": 200, "then": [{"return": "'ok'"}]},
				{"pattern": 404, "then": [{"return": "'not found'"}]}
			]}
		]
	}`, nil)
	if result.Value != "ok" {
		t.Fatalf("expected 'ok', got %v", result.Value)
	}
}

func TestMatch_Wildcard(t *testing.T) {
	result := compileAndRun(t, `{
		"steps": [
			{"let": "x", "value": 42},
			{"match": "x", "cases": [
				{"pattern": "_", "then": [{"return": "'matched anything'"}]}
			]}
		]
	}`, nil)
	if result.Value != "matched anything" {
		t.Fatalf("expected 'matched anything', got %v", result.Value)
	}
}

func TestMatch_VariableBinding(t *testing.T) {
	result := compileAndRun(t, `{
		"steps": [
			{"let": "resp", "value": {"status": 200, "body": {"name": "Alice"}}},
			{"match": "resp", "cases": [
				{"pattern": {"status": 200, "body": "$user"}, "then": [{"return": "user.name"}]}
			]}
		]
	}`, nil)
	if result.Value != "Alice" {
		t.Fatalf("expected 'Alice', got %v", result.Value)
	}
}

func TestMatch_NestedPattern(t *testing.T) {
	result := compileAndRun(t, `{
		"steps": [
			{"let": "data", "value": {"response": {"data": {"id": "abc-123", "role": "admin"}}}},
			{"match": "data", "cases": [
				{"pattern": {"response": {"data": {"role": "admin", "id": "$id"}}}, "then": [{"return": "id"}]}
			]}
		]
	}`, nil)
	if result.Value != "abc-123" {
		t.Fatalf("expected 'abc-123', got %v", result.Value)
	}
}

func TestMatch_Guard(t *testing.T) {
	result := compileAndRun(t, `{
		"steps": [
			{"let": "resp", "value": {"status": 503}},
			{"match": "resp", "cases": [
				{"pattern": {"status": "$code"}, "when": "code >= 500", "then": [{"return": "'server error'"}]},
				{"pattern": {"status": "$code"}, "when": "code >= 400", "then": [{"return": "'client error'"}]},
				{"pattern": "_", "then": [{"return": "'other'"}]}
			]}
		]
	}`, nil)
	if result.Value != "server error" {
		t.Fatalf("expected 'server error', got %v", result.Value)
	}
}

func TestMatch_FirstMatchWins(t *testing.T) {
	result := compileAndRun(t, `{
		"steps": [
			{"let": "x", "value": 42},
			{"match": "x", "cases": [
				{"pattern": "$n", "then": [{"return": "'first: ' + string(n)"}]},
				{"pattern": "_", "then": [{"return": "'second'"}]}
			]}
		]
	}`, nil)
	if result.Value != "first: 42" {
		t.Fatalf("expected 'first: 42', got %v", result.Value)
	}
}

func TestMatch_NoMatch_ReturnsNil(t *testing.T) {
	result := compileAndRun(t, `{
		"steps": [
			{"let": "x", "value": 99},
			{"match": "x", "cases": [
				{"pattern": 1, "then": [{"return": "'one'"}]},
				{"pattern": 2, "then": [{"return": "'two'"}]}
			]},
			{"return": "'no match'"}
		]
	}`, nil)
	if result.Value != "no match" {
		t.Fatalf("expected 'no match', got %v", result.Value)
	}
}

func TestMatch_ArrayPattern(t *testing.T) {
	result := compileAndRun(t, `{
		"steps": [
			{"let": "point", "value": [10, 20, 30]},
			{"match": "point", "cases": [
				{"pattern": ["$x", "$y", "$z"], "then": [{"return": "x + y + z"}]}
			]}
		]
	}`, nil)
	if !numEq(result.Value, 60) {
		t.Fatalf("expected 60, got %v", result.Value)
	}
}

func TestMatch_NilPattern(t *testing.T) {
	result := compileAndRun(t, `{
		"steps": [
			{"let": "x", "value": null},
			{"match": "x", "cases": [
				{"pattern": null, "then": [{"return": "'was nil'"}]},
				{"pattern": "_", "then": [{"return": "'was something'"}]}
			]}
		]
	}`, nil)
	if result.Value != "was nil" {
		t.Fatalf("expected 'was nil', got %v", result.Value)
	}
}

func TestMatch_SubsetMapMatch(t *testing.T) {
	result := compileAndRun(t, `{
		"steps": [
			{"let": "resp", "value": {"status": 200, "body": "hello", "headers": {"x": "y"}}},
			{"match": "resp", "cases": [
				{"pattern": {"status": 200}, "then": [{"return": "'matched with extra keys'"}]}
			]}
		]
	}`, nil)
	if result.Value != "matched with extra keys" {
		t.Fatalf("expected 'matched with extra keys', got %v", result.Value)
	}
}

func TestMatch_BindingInGuard(t *testing.T) {
	result := compileAndRun(t, `{
		"steps": [
			{"let": "order", "value": {"total": 1500, "customer": "vip"}},
			{"match": "order", "cases": [
				{"pattern": {"total": "$t", "customer": "$c"}, "when": "t > 1000 && c == 'vip'", "then": [{"return": "'big vip order'"}]},
				{"pattern": "_", "then": [{"return": "'normal'"}]}
			]}
		]
	}`, nil)
	if result.Value != "big vip order" {
		t.Fatalf("expected 'big vip order', got %v", result.Value)
	}
}

func TestMatch_StringLiteralMatch(t *testing.T) {
	result := compileAndRun(t, `{
		"steps": [
			{"let": "event", "value": {"type": "order.created", "data": {"id": "ord-1"}}},
			{"match": "event", "cases": [
				{"pattern": {"type": "order.created", "data": "$d"}, "then": [{"return": "d.id"}]},
				{"pattern": {"type": "order.cancelled"}, "then": [{"return": "'cancelled'"}]}
			]}
		]
	}`, nil)
	if result.Value != "ord-1" {
		t.Fatalf("expected 'ord-1', got %v", result.Value)
	}
}

func TestMatch_WildcardBind(t *testing.T) {
	result := compileAndRun(t, `{
		"steps": [
			{"let": "data", "value": {"a": 1, "b": 2, "c": 3}},
			{"match": "data", "cases": [
				{"pattern": {"a": "$_", "b": "$val", "c": "$_"}, "then": [{"return": "val"}]}
			]}
		]
	}`, nil)
	if !numEq(result.Value, 2) {
		t.Fatalf("expected 2, got %v", result.Value)
	}
}

func TestMatch_ReturnPropagates(t *testing.T) {
	result := compileAndRun(t, `{
		"functions": {
			"classify": {
				"params": {"code": "int"},
				"steps": [
					{"match": "code", "cases": [
						{"pattern": 200, "then": [{"return": "'success'"}]},
						{"pattern": "$c", "when": "c >= 400", "then": [{"return": "'error'"}]},
						{"pattern": "_", "then": [{"return": "'unknown'"}]}
					]}
				]
			}
		},
		"steps": [
			{"let": "r", "call": "classify", "with": {"code": "503"}},
			{"return": "r"}
		]
	}`, nil)
	if result.Value != "error" {
		t.Fatalf("expected 'error', got %v", result.Value)
	}
}

func TestMatch_InvalidSubject(t *testing.T) {
	_, err := Parse([]byte(`{
		"steps": [
			{"match": 123, "cases": [{"pattern": "_", "then": [{"return": "1"}]}]}
		]
	}`))
	if err == nil {
		t.Fatal("expected parse error for non-string match subject")
	}
	if !strings.Contains(err.Error(), "match subject must") {
		t.Fatalf("expected match subject error, got: %v", err)
	}
}

func TestMatch_NoCases(t *testing.T) {
	_, err := Parse([]byte(`{
		"steps": [
			{"match": "x", "cases": []}
		]
	}`))
	if err == nil {
		t.Fatal("expected parse error for empty cases")
	}
}

func TestMatch_BreakInsideLoop(t *testing.T) {
	result := compileAndRun(t, `{
		"steps": [
			{"let": "items", "value": [1, 2, 3, 4, 5]},
			{"let": "found", "value": 0},
			{"for": "item", "in": "items", "steps": [
				{"match": "item", "cases": [
					{"pattern": 3, "then": [
						{"set": "found", "expr": "item"},
						{"break": true}
					]},
					{"pattern": "_", "then": []}
				]}
			]},
			{"return": "found"}
		]
	}`, nil)
	if !numEq(result.Value, 3) {
		t.Fatalf("expected 3, got %v", result.Value)
	}
}
