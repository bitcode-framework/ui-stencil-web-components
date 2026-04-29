package lang

import (
	"strings"
	"testing"
)

func TestCall_WithArray_GoJsonFunction(t *testing.T) {
	result := compileAndRun(t, `{
		"functions": {
			"add": {
				"params": {"a": "int", "b": "int"},
				"returns": "int",
				"steps": [{"return": "a + b"}]
			}
		},
		"steps": [
			{"let": "result", "call": "add", "with": ["3", "4"]},
			{"return": "result"}
		]
	}`, nil)

	if !numEq(result.Value, 7) {
		t.Errorf("expected 7, got %v (%T)", result.Value, result.Value)
	}
}

func TestCall_WithArray_Method(t *testing.T) {
	result := compileAndRun(t, `{
		"structs": {
			"Calc": {
				"fields": {"value": "int"},
				"methods": {
					"add": {
						"params": {"n": "int"},
						"returns": "int",
						"steps": [{"return": "self.value + n"}]
					}
				}
			}
		},
		"steps": [
			{"let": "c", "new": "Calc", "with": {"value": "10"}},
			{"let": "result", "call": "c.add", "with": {"n": "5"}},
			{"return": "result"}
		]
	}`, nil)

	if !numEq(result.Value, 15) {
		t.Errorf("expected 15, got %v (%T)", result.Value, result.Value)
	}
}

func TestCall_Args_Literal(t *testing.T) {
	result := compileAndRun(t, `{
		"functions": {
			"describe": {
				"params": {"name": "string", "age": "int", "active": "bool"},
				"returns": "string",
				"steps": [
					{"return": "name + ' is ' + string(age) + ' active=' + string(active)"}
				]
			}
		},
		"steps": [
			{"let": "result", "call": "describe", "args": ["Alice", 30, true]},
			{"return": "result"}
		]
	}`, nil)

	if result.Value != "Alice is 30 active=true" {
		t.Errorf("expected 'Alice is 30 active=true', got %v", result.Value)
	}
}

func TestCall_Args_StringNoEscaping(t *testing.T) {
	result := compileAndRun(t, `{
		"functions": {
			"echo": {
				"params": {"msg": "string"},
				"returns": "string",
				"steps": [{"return": "msg"}]
			}
		},
		"steps": [
			{"let": "result", "call": "echo", "args": ["Don't forget the \"quotes\" and backticks"]},
			{"return": "result"}
		]
	}`, nil)

	expected := `Don't forget the "quotes" and backticks`
	if result.Value != expected {
		t.Errorf("expected %q, got %q", expected, result.Value)
	}
}

func runWithNamespace(t *testing.T, programJSON string, ns map[string]any) (*ExecutionResult, error) {
	t.Helper()
	program, err := Parse([]byte(programJSON))
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	engine := NewExprLangEngine()
	compiled, err := Compile(program, engine, DefaultLimits())
	if err != nil {
		t.Fatalf("compile error: %v", err)
	}
	vm := NewVM(compiled, engine)
	return vm.Execute(ns)
}

func TestCall_NamespaceFallback(t *testing.T) {
	program, err := Parse([]byte(`{
		"steps": [
			{"let": "result", "call": "ns.greet", "with": ["'World'"]},
			{"return": "result"}
		]
	}`))
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	engine := NewExprLangEngine()
	compiled, err := Compile(program, engine, DefaultLimits())
	if err != nil {
		t.Fatalf("compile error: %v", err)
	}
	vm := NewVM(compiled, engine)
	result, err := vm.Execute(map[string]any{
		"ns": map[string]any{
			"greet": func(args ...any) (any, error) {
				if len(args) < 1 {
					return "Hello!", nil
				}
				return "Hello, " + args[0].(string) + "!", nil
			},
		},
	})
	if err != nil {
		t.Fatalf("execution error: %v", err)
	}

	if result.Value != "Hello, World!" {
		t.Errorf("expected 'Hello, World!', got %v", result.Value)
	}
}

func TestCall_NamespaceFunction_Args(t *testing.T) {
	result, err := runWithNamespace(t, `{
		"steps": [
			{"let": "result", "call": "ns.add", "args": [10, 20]},
			{"return": "result"}
		]
	}`, map[string]any{
		"ns": map[string]any{
			"add": func(args ...any) (any, error) {
				a, _ := args[0].(float64)
				b, _ := args[1].(float64)
				return a + b, nil
			},
		},
	})
	if err != nil {
		t.Fatalf("execution error: %v", err)
	}

	if !numEq(result.Value, 30) {
		t.Errorf("expected 30, got %v", result.Value)
	}
}

func TestCall_MultiLevelNamespace(t *testing.T) {
	result, err := runWithNamespace(t, `{
		"steps": [
			{"let": "result", "call": "bc.db.query", "args": ["SELECT 1"]},
			{"return": "result"}
		]
	}`, map[string]any{
		"bc": map[string]any{
			"db": map[string]any{
				"query": func(args ...any) (any, error) {
					return map[string]any{"sql": args[0], "rows": []any{}}, nil
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("execution error: %v", err)
	}

	m, ok := result.Value.(map[string]any)
	if !ok {
		t.Fatalf("expected map, got %T", result.Value)
	}
	if m["sql"] != "SELECT 1" {
		t.Errorf("expected sql='SELECT 1', got %v", m["sql"])
	}
}

func TestCall_NamespaceNotFound(t *testing.T) {
	_, err := runWithNamespace(t, `{
		"steps": [
			{"call": "ns.nonexistent", "with": []}
		]
	}`, map[string]any{
		"ns": map[string]any{
			"greet": func(args ...any) (any, error) { return nil, nil },
		},
	})
	if err == nil {
		t.Fatal("expected error for nonexistent namespace function")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("expected 'not found' error, got: %v", err)
	}
}

func TestCall_WithArgsConflict(t *testing.T) {
	_, err := Parse([]byte(`{
		"steps": [
			{"call": "fn", "with": ["a"], "args": [1]}
		]
	}`))
	if err == nil {
		t.Fatal("expected compile error for with + args conflict")
	}
	if !strings.Contains(err.Error(), "cannot use both") {
		t.Errorf("expected 'cannot use both' error, got: %v", err)
	}
}

func TestCall_NamedWithNamespace_Error(t *testing.T) {
	_, err := runWithNamespace(t, `{
		"steps": [
			{"call": "ns.greet", "with": {"name": "'World'"}}
		]
	}`, map[string]any{
		"ns": map[string]any{
			"greet": func(args ...any) (any, error) { return nil, nil },
		},
	})
	if err == nil {
		t.Fatal("expected error for named with on namespace function")
	}
	if !strings.Contains(err.Error(), "cannot use named") {
		t.Errorf("expected 'cannot use named' error, got: %v", err)
	}
}

func TestCall_FireAndForget_Namespace(t *testing.T) {
	called := false
	program, err := Parse([]byte(`{
		"steps": [
			{"call": "ns.doSomething", "args": ["hello"]},
			{"return": "'done'"}
		]
	}`))
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	engine := NewExprLangEngine()
	compiled, err := Compile(program, engine, DefaultLimits())
	if err != nil {
		t.Fatalf("compile error: %v", err)
	}

	vm := NewVM(compiled, engine)
	result, err := vm.Execute(map[string]any{
		"ns": map[string]any{
			"doSomething": func(args ...any) (any, error) {
				called = true
				return nil, nil
			},
		},
	})
	if err != nil {
		t.Fatalf("execution error: %v", err)
	}

	if !called {
		t.Error("expected namespace function to be called")
	}
	if result.Value != "done" {
		t.Errorf("expected 'done', got %v", result.Value)
	}
}

func TestCall_Args_Empty(t *testing.T) {
	result := compileAndRun(t, `{
		"functions": {
			"hello": {
				"params": {},
				"returns": "string",
				"steps": [{"return": "'Hello!'"}]
			}
		},
		"steps": [
			{"let": "result", "call": "hello", "args": []},
			{"return": "result"}
		]
	}`, nil)

	if result.Value != "Hello!" {
		t.Errorf("expected 'Hello!', got %v", result.Value)
	}
}

func TestCall_BackwardCompat_NamedWith(t *testing.T) {
	result := compileAndRun(t, `{
		"functions": {
			"add": {
				"params": {"a": "int", "b": "int"},
				"returns": "int",
				"steps": [{"return": "a + b"}]
			}
		},
		"steps": [
			{"let": "result", "call": "add", "with": {"a": "3", "b": "4"}},
			{"return": "result"}
		]
	}`, nil)

	if !numEq(result.Value, 7) {
		t.Errorf("expected 7, got %v (%T)", result.Value, result.Value)
	}
}

func TestCall_NativePanicRecovery(t *testing.T) {
	_, err := runWithNamespace(t, `{
		"steps": [
			{"call": "ns.panicker", "args": []}
		]
	}`, map[string]any{
		"ns": map[string]any{
			"panicker": func(args ...any) (any, error) {
				panic("intentional panic")
			},
		},
	})
	if err == nil {
		t.Fatal("expected error from panic recovery")
	}
	if !strings.Contains(err.Error(), "panic") {
		t.Errorf("expected panic error, got: %v", err)
	}
}

func TestCall_WithArray_NonStringElement_Error(t *testing.T) {
	_, err := Parse([]byte(`{
		"steps": [
			{"call": "fn", "with": [{"key": "value"}]}
		]
	}`))
	if err == nil {
		t.Fatal("expected compile error for non-string element in with array")
	}
	if !strings.Contains(err.Error(), "must be a string expression") {
		t.Errorf("expected 'must be a string expression' error, got: %v", err)
	}
}
