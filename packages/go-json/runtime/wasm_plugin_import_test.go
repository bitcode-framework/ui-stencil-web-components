package runtime

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/bitcode-framework/go-json/lang"
	"github.com/bitcode-framework/go-json/stdlib"
)

func newMockWasmRuntime() *mockScriptRuntime {
	return &mockScriptRuntime{name: "wasm", extensions: []string{".wasm"}}
}

func newMockNativeRuntime() *mockScriptRuntime {
	return &mockScriptRuntime{name: "native", extensions: []string{".so", ".dylib"}}
}

func compileForTest(t *testing.T, rt *Runtime, program *lang.Program, tmpDir string) *lang.CompiledProgram {
	t.Helper()
	compiled, err := lang.Compile(program, rt.engine, rt.limits.ToResolved())
	if err != nil {
		t.Fatalf("compile error: %v", err)
	}
	compiled.AST.SourcePath = filepath.Join(tmpDir, "program.json")
	return compiled
}

func newTestRuntimeWithScripts(rts ...ScriptRuntime) *Runtime {
	reg := stdlib.DefaultRegistry()
	opts := []Option{
		WithStdlib(reg.All()),
		WithStdlibEnv(reg.EnvVars()),
	}
	for _, rt := range rts {
		opts = append(opts, WithScriptRuntime(rt))
	}
	return NewRuntime(opts...)
}

func TestWasmImportResolution(t *testing.T) {
	wasmRT := newMockWasmRuntime()
	wasmRT.execFn = func(ctx context.Context, script, function string, params, bridge map[string]any) (any, error) {
		return map[string]any{"called": function, "script": script}, nil
	}

	rt := newTestRuntimeWithScripts(wasmRT)
	defer rt.Close()

	tmpDir := t.TempDir()
	os.WriteFile(filepath.Join(tmpDir, "test.wasm"), []byte("fake-wasm"), 0644)

	program, err := lang.Parse([]byte(`{
		"import": {"img": "wasm:./test.wasm"},
		"steps": [
			{"let": "result", "call": "img.call", "with": ["'resize'", "256"]},
			{"return": "result"}
		]
	}`))
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	resolver := lang.NewImportResolver()
	_ = resolver.ResolveImports(program, tmpDir, nil)

	compiled := compileForTest(t, rt, program, tmpDir)

	result, err := rt.Execute(compiled, nil)
	if err != nil {
		t.Fatalf("execute error: %v", err)
	}

	if result == nil || result.Value == nil {
		t.Fatal("expected non-nil result")
	}

	resultMap, ok := result.Value.(map[string]any)
	if !ok {
		t.Fatalf("expected map result, got %T", result.Value)
	}
	if resultMap["called"] != "resize" {
		t.Errorf("expected called=resize, got %v", resultMap["called"])
	}
}

func TestPluginImportResolution(t *testing.T) {
	nativeRT := newMockNativeRuntime()
	nativeRT.execFn = func(ctx context.Context, script, function string, params, bridge map[string]any) (any, error) {
		return map[string]any{"called": function, "script": script}, nil
	}

	rt := newTestRuntimeWithScripts(nativeRT)
	defer rt.Close()

	tmpDir := t.TempDir()
	os.WriteFile(filepath.Join(tmpDir, "fast.so"), []byte("fake-so"), 0644)

	program, err := lang.Parse([]byte(`{
		"import": {"fast": "plugin:./fast.so"},
		"steps": [
			{"let": "result", "call": "fast.call", "with": ["'compute'", "42"]},
			{"return": "result"}
		]
	}`))
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	resolver := lang.NewImportResolver()
	_ = resolver.ResolveImports(program, tmpDir, nil)

	compiled := compileForTest(t, rt, program, tmpDir)

	result, err := rt.Execute(compiled, nil)
	if err != nil {
		t.Fatalf("execute error: %v", err)
	}

	if result == nil || result.Value == nil {
		t.Fatal("expected non-nil result")
	}

	resultMap, ok := result.Value.(map[string]any)
	if !ok {
		t.Fatalf("expected map result, got %T", result.Value)
	}
	if resultMap["called"] != "compute" {
		t.Errorf("expected called=compute, got %v", resultMap["called"])
	}
}

func TestWasmImportNoRuntimeRegistered(t *testing.T) {
	rt := newTestRuntimeWithScripts()
	defer rt.Close()

	tmpDir := t.TempDir()
	program, err := lang.Parse([]byte(`{
		"import": {"img": "wasm:./test.wasm"},
		"steps": [{"return": "1"}]
	}`))
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	resolver := lang.NewImportResolver()
	_ = resolver.ResolveImports(program, tmpDir, nil)

	compiled := compileForTest(t, rt, program, tmpDir)

	_, err = rt.Execute(compiled, nil)
	if err == nil {
		t.Fatal("expected error for missing WASM runtime")
	}
	if !contains(err.Error(), "no WASM runtime registered") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestPluginImportNoRuntimeRegistered(t *testing.T) {
	rt := newTestRuntimeWithScripts()
	defer rt.Close()

	tmpDir := t.TempDir()
	program, err := lang.Parse([]byte(`{
		"import": {"fast": "plugin:./fast.so"},
		"steps": [{"return": "1"}]
	}`))
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	resolver := lang.NewImportResolver()
	_ = resolver.ResolveImports(program, tmpDir, nil)

	compiled := compileForTest(t, rt, program, tmpDir)

	_, err = rt.Execute(compiled, nil)
	if err == nil {
		t.Fatal("expected error for missing native plugin runtime")
	}
	if !contains(err.Error(), "no native plugin runtime") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestWasmImportAbsolutePathRejected(t *testing.T) {
	wasmRT := newMockWasmRuntime()
	rt := newTestRuntimeWithScripts(wasmRT)
	defer rt.Close()

	tmpDir := t.TempDir()
	absWasmPath := filepath.Join(tmpDir, "absolute", "test.wasm")
	program, err := lang.Parse([]byte(`{
		"import": {"img": "wasm:` + filepath.ToSlash(absWasmPath) + `"},
		"steps": [{"return": "1"}]
	}`))
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	resolver := lang.NewImportResolver()
	_ = resolver.ResolveImports(program, tmpDir, nil)

	compiled := compileForTest(t, rt, program, tmpDir)

	_, err = rt.Execute(compiled, nil)
	if err == nil {
		t.Fatal("expected error for absolute wasm path")
	}
	if !contains(err.Error(), "must be relative") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestPluginImportAbsolutePathRejected(t *testing.T) {
	nativeRT := newMockNativeRuntime()
	rt := newTestRuntimeWithScripts(nativeRT)
	defer rt.Close()

	tmpDir := t.TempDir()
	absPluginPath := filepath.Join(tmpDir, "absolute", "fast.so")
	program, err := lang.Parse([]byte(`{
		"import": {"fast": "plugin:` + filepath.ToSlash(absPluginPath) + `"},
		"steps": [{"return": "1"}]
	}`))
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	resolver := lang.NewImportResolver()
	_ = resolver.ResolveImports(program, tmpDir, nil)

	compiled := compileForTest(t, rt, program, tmpDir)

	_, err = rt.Execute(compiled, nil)
	if err == nil {
		t.Fatal("expected error for absolute plugin path")
	}
	if !contains(err.Error(), "must be relative") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestPluginImportValidateError(t *testing.T) {
	nativeRT := newMockNativeRuntime()
	nativeRT.validateFn = func() error {
		return fmt.Errorf("native plugins not supported on Windows")
	}

	rt := newTestRuntimeWithScripts(nativeRT)
	defer rt.Close()

	tmpDir := t.TempDir()
	os.WriteFile(filepath.Join(tmpDir, "fast.so"), []byte("fake"), 0644)

	program, err := lang.Parse([]byte(`{
		"import": {"fast": "plugin:./fast.so"},
		"steps": [{"return": "1"}]
	}`))
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	resolver := lang.NewImportResolver()
	_ = resolver.ResolveImports(program, tmpDir, nil)

	compiled := compileForTest(t, rt, program, tmpDir)

	_, err = rt.Execute(compiled, nil)
	if err == nil {
		t.Fatal("expected error for validate failure")
	}
	if !contains(err.Error(), "not available") {
		t.Errorf("unexpected error: %v", err)
	}
}
