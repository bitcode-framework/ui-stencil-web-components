package quickjs

import (
	"testing"

	runtimes "github.com/bitcode-framework/go-json-runtimes"
)

func TestQuickJSRuntime_Name(t *testing.T) {
	rt := New()
	if rt.Name() != "quickjs" {
		t.Errorf("expected 'quickjs', got %q", rt.Name())
	}
}

func TestQuickJSRuntime_NewVM(t *testing.T) {
	rt := New()
	vm, err := rt.NewVM(runtimes.VMOptions{})
	if err != nil {
		t.Fatalf("NewVM error: %v", err)
	}
	defer vm.Close()
}

func TestQuickJSVM_Execute_SimpleReturn(t *testing.T) {
	rt := New()
	vm, _ := rt.NewVM(runtimes.VMOptions{})
	defer vm.Close()

	vm.InjectBridge(map[string]any{})
	vm.InjectParams(map[string]any{})

	result, err := vm.Execute(`1 + 2`, "test.js")
	if err != nil {
		t.Fatalf("Execute error: %v", err)
	}
	if result != float64(3) && result != int64(3) {
		t.Errorf("expected 3, got %v (%T)", result, result)
	}
}

func TestQuickJSVM_Execute_WithParams(t *testing.T) {
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

func TestQuickJSVM_Execute_SyntaxError(t *testing.T) {
	rt := New()
	vm, _ := rt.NewVM(runtimes.VMOptions{})
	defer vm.Close()

	vm.InjectBridge(map[string]any{})
	_, err := vm.Execute(`function(`, "bad.js")
	if err == nil {
		t.Fatal("expected syntax error")
	}
}

func TestQuickJSVM_NilResult(t *testing.T) {
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
