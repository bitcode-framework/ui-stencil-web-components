package lang

import (
	"context"
	"strings"
	"testing"
	"time"
)

func TestSleep_Literal(t *testing.T) {
	start := time.Now()
	result := compileAndRun(t, `{
		"steps": [
			{"sleep": 50},
			{"return": "'done'"}
		]
	}`, nil)
	elapsed := time.Since(start)
	if result.Value != "done" {
		t.Fatalf("expected 'done', got %v", result.Value)
	}
	if elapsed < 40*time.Millisecond {
		t.Fatalf("sleep too short: %v", elapsed)
	}
}

func TestSleep_Expression(t *testing.T) {
	start := time.Now()
	result := compileAndRun(t, `{
		"steps": [
			{"let": "delay", "value": 50},
			{"sleep": "delay"},
			{"return": "'done'"}
		]
	}`, nil)
	elapsed := time.Since(start)
	if result.Value != "done" {
		t.Fatalf("expected 'done', got %v", result.Value)
	}
	if elapsed < 40*time.Millisecond {
		t.Fatalf("sleep too short: %v", elapsed)
	}
}

func TestSleep_ZeroIsNoop(t *testing.T) {
	result := compileAndRun(t, `{
		"steps": [
			{"sleep": 0},
			{"return": "'ok'"}
		]
	}`, nil)
	if result.Value != "ok" {
		t.Fatalf("expected 'ok', got %v", result.Value)
	}
}

func TestSleep_RespectsTimeout(t *testing.T) {
	program, err := Parse([]byte(`{
		"steps": [
			{"sleep": 10000}
		]
	}`))
	if err != nil {
		t.Fatal(err)
	}
	engine := NewExprLangEngine()
	compiled, err := Compile(program, engine, DefaultLimits())
	if err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	vm := NewVM(compiled, engine, WithContext(ctx))
	_, err = vm.Execute(nil)
	if err == nil {
		t.Fatal("expected timeout error")
	}
	if !strings.Contains(err.Error(), "TIMEOUT") && !strings.Contains(err.Error(), "cancelled") {
		t.Fatalf("expected timeout error, got: %v", err)
	}
}

func TestRetry_SucceedsFirstAttempt(t *testing.T) {
	result := compileAndRun(t, `{
		"steps": [
			{"retry": {
				"steps": [
					{"return": "42"}
				],
				"max": 3,
				"delay": 10
			}}
		]
	}`, nil)
	if !numEq(result.Value, 42) {
		t.Fatalf("expected 42, got %v", result.Value)
	}
}

func TestRetry_SucceedsOnThirdAttempt(t *testing.T) {
	result := compileAndRun(t, `{
		"steps": [
			{"let": "attempts", "value": 0},
			{"retry": {
				"steps": [
					{"set": "attempts", "expr": "attempts + 1"},
					{"if": "attempts < 3", "then": [{"error": "'not ready'"}]},
					{"return": "attempts"}
				],
				"max": 5,
				"delay": 10
			}}
		]
	}`, nil)
	if !numEq(result.Value, 3) {
		t.Fatalf("expected 3, got %v", result.Value)
	}
}

func TestRetry_Exhausted(t *testing.T) {
	program, err := Parse([]byte(`{
		"steps": [
			{"retry": {
				"steps": [
					{"error": "'always fails'"}
				],
				"max": 3,
				"delay": 10
			}}
		]
	}`))
	if err != nil {
		t.Fatal(err)
	}
	engine := NewExprLangEngine()
	compiled, err := Compile(program, engine, DefaultLimits())
	if err != nil {
		t.Fatal(err)
	}
	vm := NewVM(compiled, engine)
	_, err = vm.Execute(nil)
	if err == nil {
		t.Fatal("expected retry exhausted error")
	}
	if !strings.Contains(err.Error(), "retry exhausted") {
		t.Fatalf("expected retry exhausted error, got: %v", err)
	}
}

func TestRetry_ExponentialBackoff(t *testing.T) {
	start := time.Now()
	program, err := Parse([]byte(`{
		"steps": [
			{"let": "attempts", "value": 0},
			{"retry": {
				"steps": [
					{"set": "attempts", "expr": "attempts + 1"},
					{"if": "attempts < 3", "then": [{"error": "'retry'"}]},
					{"return": "attempts"}
				],
				"max": 3,
				"delay": 20,
				"backoff": "exponential"
			}}
		]
	}`))
	if err != nil {
		t.Fatal(err)
	}
	engine := NewExprLangEngine()
	compiled, err := Compile(program, engine, DefaultLimits())
	if err != nil {
		t.Fatal(err)
	}
	vm := NewVM(compiled, engine)
	result, err := vm.Execute(nil)
	if err != nil {
		t.Fatal(err)
	}
	elapsed := time.Since(start)
	if !numEq(result.Value, 3) {
		t.Fatalf("expected 3, got %v", result.Value)
	}
	if elapsed < 50*time.Millisecond {
		t.Fatalf("exponential backoff too fast: %v (expected ~60ms: 20ms + 40ms)", elapsed)
	}
}

func TestAssert_TrueCondition(t *testing.T) {
	result := compileAndRun(t, `{
		"steps": [
			{"let": "x", "value": 10},
			{"assert": "x > 0"},
			{"return": "'passed'"}
		]
	}`, nil)
	if result.Value != "passed" {
		t.Fatalf("expected 'passed', got %v", result.Value)
	}
}

func TestAssert_FalseCondition(t *testing.T) {
	program, err := Parse([]byte(`{
		"steps": [
			{"let": "x", "value": -1},
			{"assert": "x > 0"}
		]
	}`))
	if err != nil {
		t.Fatal(err)
	}
	engine := NewExprLangEngine()
	compiled, err := Compile(program, engine, DefaultLimits())
	if err != nil {
		t.Fatal(err)
	}
	vm := NewVM(compiled, engine)
	_, err = vm.Execute(nil)
	if err == nil {
		t.Fatal("expected assertion error")
	}
	if !strings.Contains(err.Error(), "assertion failed") {
		t.Fatalf("expected assertion failed error, got: %v", err)
	}
}

func TestAssert_CustomMessage(t *testing.T) {
	program, err := Parse([]byte(`{
		"steps": [
			{"let": "items", "value": []},
			{"assert": "len(items) > 0", "message": "'Items cannot be empty'"}
		]
	}`))
	if err != nil {
		t.Fatal(err)
	}
	engine := NewExprLangEngine()
	compiled, err := Compile(program, engine, DefaultLimits())
	if err != nil {
		t.Fatal(err)
	}
	vm := NewVM(compiled, engine)
	_, err = vm.Execute(nil)
	if err == nil {
		t.Fatal("expected assertion error")
	}
	if !strings.Contains(err.Error(), "Items cannot be empty") {
		t.Fatalf("expected custom message, got: %v", err)
	}
}

func TestConstants_AccessInExpr(t *testing.T) {
	result := compileAndRun(t, `{
		"constants": {
			"MAX_RETRIES": 3,
			"TAX_RATE": 0.11
		},
		"steps": [
			{"let": "price", "value": 100},
			{"let": "total", "expr": "price * (1 + TAX_RATE)"},
			{"return": "total"}
		]
	}`, nil)
	f, ok := result.Value.(float64)
	if !ok {
		t.Fatalf("expected float64, got %T", result.Value)
	}
	if f < 110.9 || f > 111.1 {
		t.Fatalf("expected ~111, got %v", f)
	}
}

func TestConstants_ReassignBlocked(t *testing.T) {
	program, err := Parse([]byte(`{
		"constants": {"MAX": 10},
		"steps": [
			{"set": "MAX", "value": 20}
		]
	}`))
	if err != nil {
		t.Fatal(err)
	}
	engine := NewExprLangEngine()
	_, err = Compile(program, engine, DefaultLimits())
	if err == nil {
		t.Fatal("expected compile error for constant reassignment")
	}
	if !strings.Contains(err.Error(), "cannot reassign") {
		t.Fatalf("expected cannot reassign error, got: %v", err)
	}
}

func TestEnums_ArrayEnum(t *testing.T) {
	result := compileAndRun(t, `{
		"enums": {
			"Status": ["draft", "confirmed", "done"]
		},
		"steps": [
			{"let": "s", "expr": "Status.draft"},
			{"return": "s"}
		]
	}`, nil)
	if result.Value != "draft" {
		t.Fatalf("expected 'draft', got %v", result.Value)
	}
}

func TestEnums_MapEnum(t *testing.T) {
	result := compileAndRun(t, `{
		"enums": {
			"Priority": {"LOW": 1, "MEDIUM": 2, "HIGH": 3}
		},
		"steps": [
			{"let": "p", "expr": "Priority.HIGH"},
			{"return": "p"}
		]
	}`, nil)
	if !numEq(result.Value, 3) {
		t.Fatalf("expected 3, got %v", result.Value)
	}
}

func TestEnums_InCondition(t *testing.T) {
	result := compileAndRun(t, `{
		"enums": {
			"Status": ["draft", "confirmed", "done"]
		},
		"steps": [
			{"let": "order", "value": {"status": "confirmed"}},
			{"if": "order.status == Status.confirmed", "then": [
				{"return": "'match'"}
			]},
			{"return": "'no match'"}
		]
	}`, nil)
	if result.Value != "match" {
		t.Fatalf("expected 'match', got %v", result.Value)
	}
}

func TestEnums_ReassignBlocked(t *testing.T) {
	program, err := Parse([]byte(`{
		"enums": {"Status": ["a", "b"]},
		"steps": [
			{"set": "Status", "value": "hacked"}
		]
	}`))
	if err != nil {
		t.Fatal(err)
	}
	engine := NewExprLangEngine()
	_, err = Compile(program, engine, DefaultLimits())
	if err == nil {
		t.Fatal("expected compile error for enum reassignment")
	}
	if !strings.Contains(err.Error(), "cannot reassign") {
		t.Fatalf("expected cannot reassign error, got: %v", err)
	}
}

func TestCallRef_VariableContainingFunctionName(t *testing.T) {
	result := compileAndRun(t, `{
		"functions": {
			"double": {
				"params": {"x": "int"},
				"steps": [{"return": "x * 2"}]
			}
		},
		"steps": [
			{"let": "fnName", "value": "double"},
			{"let": "result", "call": "fnName", "with": ["5"]},
			{"return": "result"}
		]
	}`, nil)
	if !numEq(result.Value, 10) {
		t.Fatalf("expected 10, got %v", result.Value)
	}
}

func TestCallRef_VariableContainingLambda(t *testing.T) {
	result := compileAndRun(t, `{
		"steps": [
			{"let": "transform", "expr": "fn(x) => x * 3"},
			{"let": "result", "call": "transform", "with": ["7"]},
			{"return": "result"}
		]
	}`, nil)
	if !numEq(result.Value, 21) {
		t.Fatalf("expected 21, got %v", result.Value)
	}
}

func TestCallRef_StrategyPattern(t *testing.T) {
	result := compileAndRun(t, `{
		"functions": {
			"add": {
				"params": {"a": "int", "b": "int"},
				"steps": [{"return": "a + b"}]
			},
			"multiply": {
				"params": {"a": "int", "b": "int"},
				"steps": [{"return": "a * b"}]
			}
		},
		"steps": [
			{"let": "op", "value": "multiply"},
			{"let": "result", "call": "op", "with": ["3", "4"]},
			{"return": "result"}
		]
	}`, nil)
	if !numEq(result.Value, 12) {
		t.Fatalf("expected 12, got %v", result.Value)
	}
}
