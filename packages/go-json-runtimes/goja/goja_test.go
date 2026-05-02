package goja

import (
	"testing"

	runtimes "github.com/bitcode-framework/go-json-runtimes"
)

func TestGojaRuntime_Name(t *testing.T) {
	rt := New()
	if rt.Name() != "goja" {
		t.Errorf("expected 'goja', got %q", rt.Name())
	}
}

func TestGojaRuntime_NewVM(t *testing.T) {
	rt := New()
	vm, err := rt.NewVM(runtimes.VMOptions{})
	if err != nil {
		t.Fatalf("NewVM error: %v", err)
	}
	defer vm.Close()
}

func TestGojaVM_Execute_SimpleReturn(t *testing.T) {
	rt := New()
	vm, _ := rt.NewVM(runtimes.VMOptions{})
	defer vm.Close()

	vm.InjectBridge(map[string]any{})
	vm.InjectParams(map[string]any{})

	result, err := vm.Execute(`1 + 2`, "test.js")
	if err != nil {
		t.Fatalf("Execute error: %v", err)
	}
	if result != int64(3) {
		t.Errorf("expected 3, got %v (%T)", result, result)
	}
}

func TestGojaVM_Execute_WithParams(t *testing.T) {
	rt := New()
	vm, _ := rt.NewVM(runtimes.VMOptions{})
	defer vm.Close()

	vm.InjectBridge(map[string]any{})
	vm.InjectParams(map[string]any{"name": "World"})

	result, err := vm.Execute(`"Hello, " + params.name + "!"`, "test.js")
	if err != nil {
		t.Fatalf("Execute error: %v", err)
	}
	if result != "Hello, World!" {
		t.Errorf("expected 'Hello, World!', got %v", result)
	}
}

func TestGojaVM_Execute_WithBridge(t *testing.T) {
	rt := New()
	vm, _ := rt.NewVM(runtimes.VMOptions{})
	defer vm.Close()

	bridge := map[string]any{
		"greet": func(name string) string { return "Hi, " + name },
	}
	vm.InjectBridge(bridge)
	vm.InjectParams(map[string]any{})

	result, err := vm.Execute(`bitcode.greet("Alice")`, "test.js")
	if err != nil {
		t.Fatalf("Execute error: %v", err)
	}
	if result != "Hi, Alice" {
		t.Errorf("expected 'Hi, Alice', got %v", result)
	}
}

func TestGojaVM_Execute_FunctionExport(t *testing.T) {
	rt := New()
	vm, _ := rt.NewVM(runtimes.VMOptions{})
	defer vm.Close()

	vm.InjectBridge(map[string]any{})
	vm.InjectParams(map[string]any{"x": 10})

	result, err := vm.Execute(`(function(bitcode, params) { return params.x * 2; })`, "test.js")
	if err != nil {
		t.Fatalf("Execute error: %v", err)
	}
	if result != int64(20) {
		t.Errorf("expected 20, got %v (%T)", result, result)
	}
}

func TestGojaVM_Execute_ObjectWithExecute(t *testing.T) {
	rt := New()
	vm, _ := rt.NewVM(runtimes.VMOptions{})
	defer vm.Close()

	vm.InjectBridge(map[string]any{})
	vm.InjectParams(map[string]any{"val": 5})

	result, err := vm.Execute(`({execute: function(bitcode, params) { return params.val + 1; }})`, "test.js")
	if err != nil {
		t.Fatalf("Execute error: %v", err)
	}
	if result != int64(6) {
		t.Errorf("expected 6, got %v (%T)", result, result)
	}
}

func TestGojaVM_Execute_SyntaxError(t *testing.T) {
	rt := New()
	vm, _ := rt.NewVM(runtimes.VMOptions{})
	defer vm.Close()

	vm.InjectBridge(map[string]any{})
	_, err := vm.Execute(`function(`, "bad.js")
	if err == nil {
		t.Fatal("expected syntax error")
	}
}

func TestGojaVM_Execute_RuntimeError(t *testing.T) {
	rt := New()
	vm, _ := rt.NewVM(runtimes.VMOptions{})
	defer vm.Close()

	vm.InjectBridge(map[string]any{})
	_, err := vm.Execute(`undefinedVar.property`, "error.js")
	if err == nil {
		t.Fatal("expected runtime error")
	}
}

func TestGojaVM_Interrupt(t *testing.T) {
	rt := New()
	vm, _ := rt.NewVM(runtimes.VMOptions{})
	defer vm.Close()

	vm.InjectBridge(map[string]any{})
	vm.Interrupt("test interrupt")
	_, err := vm.Execute(`1 + 1`, "test.js")
	if err == nil {
		t.Fatal("expected interrupt error")
	}
}

func TestGojaVM_Console(t *testing.T) {
	rt := New()
	vm, _ := rt.NewVM(runtimes.VMOptions{})
	defer vm.Close()

	var logged []string
	bridge := map[string]any{
		"log": func(args ...any) (any, error) {
			if len(args) >= 2 {
				logged = append(logged, args[1].(string))
			}
			return nil, nil
		},
	}
	vm.InjectBridge(bridge)

	_, err := vm.Execute(`console.log("hello from js")`, "test.js")
	if err != nil {
		t.Fatalf("Execute error: %v", err)
	}
	if len(logged) == 0 {
		t.Error("expected console.log to be captured")
	}
}

func TestGojaVM_NilResult(t *testing.T) {
	rt := New()
	vm, _ := rt.NewVM(runtimes.VMOptions{})
	defer vm.Close()

	vm.InjectBridge(map[string]any{})
	result, err := vm.Execute(`undefined`, "test.js")
	if err != nil {
		t.Fatalf("Execute error: %v", err)
	}
	if result != nil {
		t.Errorf("expected nil, got %v", result)
	}
}

func TestGojaVM_NestedBridge(t *testing.T) {
	rt := New()
	vm, _ := rt.NewVM(runtimes.VMOptions{})
	defer vm.Close()

	bridge := map[string]any{
		"db": map[string]any{
			"query": func(sql string) (any, error) {
				return []map[string]any{{"id": 1, "name": "test"}}, nil
			},
		},
	}
	vm.InjectBridge(bridge)
	vm.InjectParams(map[string]any{})

	result, err := vm.Execute(`bitcode.db.query("SELECT 1")`, "test.js")
	if err != nil {
		t.Fatalf("Execute error: %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil result")
	}
}
