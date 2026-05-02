package wasm

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestWasmRuntime_Name(t *testing.T) {
	rt := New(DefaultConfig())
	defer rt.Close()

	if rt.Name() != "wasm" {
		t.Errorf("expected name 'wasm', got %q", rt.Name())
	}
}

func TestWasmRuntime_Extensions(t *testing.T) {
	rt := New(DefaultConfig())
	defer rt.Close()

	exts := rt.Extensions()
	if len(exts) != 1 || exts[0] != ".wasm" {
		t.Errorf("expected [.wasm], got %v", exts)
	}
}

func TestWasmRuntime_CanHandle(t *testing.T) {
	rt := New(DefaultConfig())
	defer rt.Close()

	if !rt.CanHandle(".wasm") {
		t.Error("expected CanHandle(.wasm) = true")
	}
	if rt.CanHandle(".js") {
		t.Error("expected CanHandle(.js) = false")
	}
	if rt.CanHandle(".so") {
		t.Error("expected CanHandle(.so) = false")
	}
}

func TestWasmRuntime_Validate(t *testing.T) {
	rt := New(DefaultConfig())
	defer rt.Close()

	if err := rt.Validate(); err != nil {
		t.Errorf("Validate() should return nil for embedded runtime, got: %v", err)
	}
}

func TestWasmRuntime_Close(t *testing.T) {
	rt := New(DefaultConfig())
	if err := rt.Close(); err != nil {
		t.Errorf("Close() error: %v", err)
	}
}

func TestWasmRuntime_InvalidModule(t *testing.T) {
	rt := New(DefaultConfig())
	defer rt.Close()

	tmpDir := t.TempDir()
	badFile := filepath.Join(tmpDir, "bad.wasm")
	os.WriteFile(badFile, []byte("not a wasm module"), 0644)

	_, err := rt.Execute(context.Background(), badFile, "test", nil, nil)
	if err == nil {
		t.Fatal("expected error for invalid WASM module")
	}
}

func TestWasmRuntime_FileNotFound(t *testing.T) {
	rt := New(DefaultConfig())
	defer rt.Close()

	_, err := rt.Execute(context.Background(), "/nonexistent/file.wasm", "test", nil, nil)
	if err == nil {
		t.Fatal("expected error for missing file")
	}
}

func TestWasmRuntime_FunctionNotFound(t *testing.T) {
	rt := New(DefaultConfig())
	defer rt.Close()

	tmpDir := t.TempDir()
	wasmFile := filepath.Join(tmpDir, "test.wasm")
	os.WriteFile(wasmFile, buildMinimalWasmWithMalloc(t), 0644)

	_, err := rt.Execute(context.Background(), wasmFile, "nonexistent_function", map[string]any{}, nil)
	if err == nil {
		t.Fatal("expected error for missing function")
	}
	if !containsStr(err.Error(), "not exported") {
		t.Errorf("expected 'not exported' error, got: %v", err)
	}
}

func TestWasmRuntime_MissingMalloc(t *testing.T) {
	rt := New(DefaultConfig())
	defer rt.Close()

	tmpDir := t.TempDir()
	wasmFile := filepath.Join(tmpDir, "nomalloc.wasm")
	os.WriteFile(wasmFile, buildWasmWithoutMalloc(t), 0644)

	_, err := rt.Execute(context.Background(), wasmFile, "test", map[string]any{}, nil)
	if err == nil {
		t.Fatal("expected error for missing malloc")
	}
	if !containsStr(err.Error(), "malloc") {
		t.Errorf("expected 'malloc' in error, got: %v", err)
	}
}

func TestWasmRuntime_CompileCache(t *testing.T) {
	rt := New(Config{
		MaxMemoryMB:  64,
		MaxExecTime:  30 * time.Second,
		CompileCache: true,
	})
	defer rt.Close()

	tmpDir := t.TempDir()
	wasmFile := filepath.Join(tmpDir, "test.wasm")
	os.WriteFile(wasmFile, buildMinimalWasmWithMalloc(t), 0644)

	ctx := context.Background()

	_, err := rt.getOrCompile(ctx, wasmFile)
	if err != nil {
		t.Fatalf("first compile error: %v", err)
	}

	rt.mu.RLock()
	cacheSize := len(rt.cache)
	rt.mu.RUnlock()

	if cacheSize != 1 {
		t.Errorf("expected 1 cached module, got %d", cacheSize)
	}

	_, err = rt.getOrCompile(ctx, wasmFile)
	if err != nil {
		t.Fatalf("second compile (cached) error: %v", err)
	}
}

func TestWasmRuntime_NoCacheMode(t *testing.T) {
	rt := New(Config{
		MaxMemoryMB:  64,
		MaxExecTime:  30 * time.Second,
		CompileCache: false,
	})
	defer rt.Close()

	tmpDir := t.TempDir()
	wasmFile := filepath.Join(tmpDir, "test.wasm")
	os.WriteFile(wasmFile, buildMinimalWasmWithMalloc(t), 0644)

	ctx := context.Background()

	_, err := rt.getOrCompile(ctx, wasmFile)
	if err != nil {
		t.Fatalf("compile error: %v", err)
	}

	rt.mu.RLock()
	cacheSize := len(rt.cache)
	rt.mu.RUnlock()

	if cacheSize != 0 {
		t.Errorf("expected 0 cached modules with CompileCache=false, got %d", cacheSize)
	}
}

func TestWasmRuntime_ContextCancellation(t *testing.T) {
	// Verify that a pre-cancelled context returns an error immediately.
	rt := New(DefaultConfig())
	defer rt.Close()

	tmpDir := t.TempDir()
	wasmFile := filepath.Join(tmpDir, "test.wasm")
	os.WriteFile(wasmFile, buildMinimalWasmWithMalloc(t), 0644)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := rt.Execute(ctx, wasmFile, "malloc", map[string]any{}, nil)
	if err == nil {
		t.Fatal("expected error for cancelled context")
	}
}

func TestWasmRuntime_MemoryProtocol(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping memory protocol test in short mode")
	}

	rt := New(DefaultConfig())
	defer rt.Close()

	tmpDir := t.TempDir()
	wasmFile := filepath.Join(tmpDir, "echo.wasm")
	wasmBytes := buildEchoWasm(t)
	if wasmBytes == nil {
		t.Skip("echo WASM module not available for this test")
	}
	os.WriteFile(wasmFile, wasmBytes, 0644)

	params := map[string]any{"args": []any{"hello"}}
	result, err := rt.Execute(context.Background(), wasmFile, "echo", params, nil)
	if err != nil {
		t.Fatalf("execute error: %v", err)
	}

	if result == nil {
		t.Fatal("expected non-nil result")
	}
}

func TestPackUnpackPtrLen(t *testing.T) {
	tests := []struct {
		ptr uint32
		len uint32
	}{
		{0, 0},
		{1024, 256},
		{2048, 1},
		{0xFFFF, 0xFFFF},
		{0xFFFFFFFF, 0xFFFFFFFF},
	}

	for _, tt := range tests {
		packed := packPtrLen(tt.ptr, tt.len)
		gotPtr, gotLen := unpackPtrLen(packed)
		if gotPtr != tt.ptr || gotLen != tt.len {
			t.Errorf("pack/unpack(%d, %d): got (%d, %d)", tt.ptr, tt.len, gotPtr, gotLen)
		}
	}
}

func TestInvokeBridgeFunction_NilBridge(t *testing.T) {
	_, err := invokeBridgeFunction(nil, "test", nil)
	if err == nil {
		t.Fatal("expected error for nil bridge")
	}
}

func TestInvokeBridgeFunction_NotFound(t *testing.T) {
	bridge := map[string]any{"foo": "not a function"}
	_, err := invokeBridgeFunction(bridge, "foo", nil)
	if err == nil {
		t.Fatal("expected error for non-callable bridge value")
	}
}

func TestInvokeBridgeFunction_DottedPath(t *testing.T) {
	called := false
	bridge := map[string]any{
		"db": map[string]any{
			"query": func(args ...any) (any, error) {
				called = true
				return "result", nil
			},
		},
	}

	argsJSON, _ := json.Marshal([]any{"SELECT 1"})
	result, err := invokeBridgeFunction(bridge, "db.query", argsJSON)
	if err != nil {
		t.Fatalf("bridge call error: %v", err)
	}
	if !called {
		t.Error("bridge function was not called")
	}
	if result != "result" {
		t.Errorf("expected 'result', got %v", result)
	}
}

func TestInvokeBridgeFunction_MapArgs(t *testing.T) {
	bridge := map[string]any{
		"process": func(args map[string]any) (any, error) {
			return args["key"], nil
		},
	}

	argsJSON, _ := json.Marshal([]any{map[string]any{"key": "value"}})
	result, err := invokeBridgeFunction(bridge, "process", argsJSON)
	if err != nil {
		t.Fatalf("bridge call error: %v", err)
	}
	if result != "value" {
		t.Errorf("expected 'value', got %v", result)
	}
}

func containsStr(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
