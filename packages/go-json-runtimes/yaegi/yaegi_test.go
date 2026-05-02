package yaegi

import (
	"testing"
	"time"

	runtimes "github.com/bitcode-framework/go-json-runtimes"
)

func TestYaegiRuntime_Name(t *testing.T) {
	rt := New()
	if rt.Name() != "yaegi" {
		t.Errorf("expected 'yaegi', got %q", rt.Name())
	}
}

func TestYaegiRuntime_NewVM(t *testing.T) {
	rt := New()
	vm, err := rt.NewVM(runtimes.VMOptions{Timeout: 5 * time.Second})
	if err != nil {
		t.Fatalf("NewVM error: %v", err)
	}
	defer vm.Close()
}

func TestYaegiVM_Execute_SimpleReturn(t *testing.T) {
	rt := New()
	vm, _ := rt.NewVM(runtimes.VMOptions{Timeout: 5 * time.Second})
	defer vm.Close()

	vm.InjectBridge(map[string]any{})
	vm.InjectParams(map[string]any{})

	code := `package main

func Execute() (any, error) {
	return 42, nil
}
`
	result, err := vm.Execute(code, "test.go")
	if err != nil {
		t.Fatalf("Execute error: %v", err)
	}
	if result != 42 {
		t.Errorf("expected 42, got %v (%T)", result, result)
	}
}

func TestYaegiVM_Execute_WithParams(t *testing.T) {
	rt := New()
	vm, _ := rt.NewVM(runtimes.VMOptions{Timeout: 5 * time.Second})
	defer vm.Close()

	vm.InjectBridge(map[string]any{})
	vm.InjectParams(map[string]any{"name": "World"})

	code := `package main

import "params"

func Execute(p map[string]any) (any, error) {
	return "Hello, " + p["name"].(string) + "!", nil
}
`
	result, err := vm.Execute(code, "test.go")
	if err != nil {
		t.Fatalf("Execute error: %v", err)
	}
	if result != "Hello, World!" {
		t.Errorf("expected 'Hello, World!', got %v", result)
	}
}

func TestYaegiVM_Execute_WithBridge(t *testing.T) {
	rt := New()
	vm, _ := rt.NewVM(runtimes.VMOptions{Timeout: 5 * time.Second})
	defer vm.Close()

	bridge := map[string]any{
		"greeting": "Hi from bridge",
	}
	vm.InjectBridge(bridge)
	vm.InjectParams(map[string]any{})

	code := `package main

import "bitcode"

func Execute() (any, error) {
	b := bitcode.Bridge
	return b["greeting"], nil
}
`
	result, err := vm.Execute(code, "test.go")
	if err != nil {
		t.Fatalf("Execute error: %v", err)
	}
	if result != "Hi from bridge" {
		t.Errorf("expected 'Hi from bridge', got %v", result)
	}
}

func TestYaegiVM_Execute_Timeout(t *testing.T) {
	rt := New()
	vm, _ := rt.NewVM(runtimes.VMOptions{Timeout: 100 * time.Millisecond})
	defer vm.Close()

	vm.InjectBridge(map[string]any{})
	vm.InjectParams(map[string]any{})

	code := `package main

import "context"
import "time"

func Execute(ctx context.Context, p map[string]any) (any, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-time.After(5 * time.Second):
		return "should not reach", nil
	}
}
`
	_, err := vm.Execute(code, "timeout.go")
	if err == nil {
		t.Fatal("expected timeout error")
	}
}
