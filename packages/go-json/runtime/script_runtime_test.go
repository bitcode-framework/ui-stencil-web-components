package runtime

import (
	"context"
	"fmt"
	"testing"
)

type mockScriptRuntime struct {
	name       string
	extensions []string
	validateFn func() error
	execFn     func(ctx context.Context, script, function string, params, bridge map[string]any) (any, error)
	closed     bool
}

func (m *mockScriptRuntime) Name() string        { return m.name }
func (m *mockScriptRuntime) Extensions() []string { return m.extensions }
func (m *mockScriptRuntime) CanHandle(ext string) bool {
	for _, e := range m.extensions {
		if e == ext {
			return true
		}
	}
	return false
}
func (m *mockScriptRuntime) Execute(ctx context.Context, script, function string, params, bridge map[string]any) (any, error) {
	if m.execFn != nil {
		return m.execFn(ctx, script, function, params, bridge)
	}
	return map[string]any{"script": script, "function": function}, nil
}
func (m *mockScriptRuntime) Validate() error {
	if m.validateFn != nil {
		return m.validateFn()
	}
	return nil
}
func (m *mockScriptRuntime) Close() error {
	m.closed = true
	return nil
}

func newMockPythonRuntime() *mockScriptRuntime {
	return &mockScriptRuntime{name: "python", extensions: []string{".py", ".pyw"}}
}

func newMockNodeRuntime() *mockScriptRuntime {
	return &mockScriptRuntime{name: "node", extensions: []string{".js", ".ts", ".mjs"}}
}

func newMockGoRuntime() *mockScriptRuntime {
	return &mockScriptRuntime{name: "yaegi", extensions: []string{".go"}}
}

func TestScriptRuntimeRegistry_Register(t *testing.T) {
	reg := NewScriptRuntimeRegistry()
	if reg.HasRuntimes() {
		t.Fatal("new registry should have no runtimes")
	}

	reg.Register(newMockPythonRuntime())
	if !reg.HasRuntimes() {
		t.Fatal("registry should have runtimes after Register")
	}
	if len(reg.All()) != 1 {
		t.Fatalf("expected 1 runtime, got %d", len(reg.All()))
	}

	reg.Register(newMockNodeRuntime())
	if len(reg.All()) != 2 {
		t.Fatalf("expected 2 runtimes, got %d", len(reg.All()))
	}
}

func TestScriptRuntimeRegistry_Resolve(t *testing.T) {
	reg := NewScriptRuntimeRegistry()
	reg.Register(newMockPythonRuntime())
	reg.Register(newMockNodeRuntime())
	reg.Register(newMockGoRuntime())

	tests := []struct {
		ext      string
		wantName string
	}{
		{".py", "python"},
		{".pyw", "python"},
		{".js", "node"},
		{".ts", "node"},
		{".mjs", "node"},
		{".go", "yaegi"},
	}

	for _, tt := range tests {
		rt := reg.Resolve(tt.ext)
		if rt == nil {
			t.Errorf("Resolve(%q) returned nil, want %q", tt.ext, tt.wantName)
			continue
		}
		if rt.Name() != tt.wantName {
			t.Errorf("Resolve(%q).Name() = %q, want %q", tt.ext, rt.Name(), tt.wantName)
		}
	}

	if rt := reg.Resolve(".wasm"); rt != nil {
		t.Errorf("Resolve(.wasm) should return nil, got %q", rt.Name())
	}
}

func TestScriptRuntimeRegistry_ResolveByName(t *testing.T) {
	reg := NewScriptRuntimeRegistry()
	reg.Register(newMockPythonRuntime())
	reg.Register(newMockNodeRuntime())

	rt := reg.ResolveByName("python")
	if rt == nil || rt.Name() != "python" {
		t.Error("ResolveByName('python') failed")
	}

	rt = reg.ResolveByName("node")
	if rt == nil || rt.Name() != "node" {
		t.Error("ResolveByName('node') failed")
	}

	if reg.ResolveByName("nonexistent") != nil {
		t.Error("ResolveByName('nonexistent') should return nil")
	}
}

func TestScriptRuntimeRegistry_Close(t *testing.T) {
	py := newMockPythonRuntime()
	node := newMockNodeRuntime()

	reg := NewScriptRuntimeRegistry()
	reg.Register(py)
	reg.Register(node)

	if err := reg.Close(); err != nil {
		t.Fatalf("Close() returned error: %v", err)
	}

	if !py.closed {
		t.Error("python runtime should be closed")
	}
	if !node.closed {
		t.Error("node runtime should be closed")
	}
}

func TestScriptRuntimeRegistry_CloseReturnsFirstError(t *testing.T) {
	reg := NewScriptRuntimeRegistry()

	errRT := &errorClosingRuntime{name: "err1", closeErr: fmt.Errorf("close error 1")}
	errRT2 := &errorClosingRuntime{name: "err2", closeErr: fmt.Errorf("close error 2")}
	reg.Register(errRT)
	reg.Register(errRT2)

	err := reg.Close()
	if err == nil {
		t.Fatal("expected error from Close()")
	}
	if err.Error() != "close error 1" {
		t.Errorf("expected first error, got: %v", err)
	}
}

type errorClosingRuntime struct {
	name     string
	closeErr error
}

func (e *errorClosingRuntime) Name() string        { return e.name }
func (e *errorClosingRuntime) Extensions() []string { return nil }
func (e *errorClosingRuntime) CanHandle(string) bool { return false }
func (e *errorClosingRuntime) Execute(context.Context, string, string, map[string]any, map[string]any) (any, error) {
	return nil, nil
}
func (e *errorClosingRuntime) Validate() error { return nil }
func (e *errorClosingRuntime) Close() error    { return e.closeErr }

func TestScriptRuntimeRegistry_ResolveFirstMatchWins(t *testing.T) {
	first := &mockScriptRuntime{name: "first", extensions: []string{".js"}}
	second := &mockScriptRuntime{name: "second", extensions: []string{".js"}}

	reg := NewScriptRuntimeRegistry()
	reg.Register(first)
	reg.Register(second)

	rt := reg.Resolve(".js")
	if rt == nil || rt.Name() != "first" {
		t.Errorf("expected first-registered runtime to win, got %v", rt)
	}
}

func TestWithScriptRuntime(t *testing.T) {
	py := newMockPythonRuntime()
	rt := NewRuntime(WithScriptRuntime(py))
	defer rt.Close()

	if !rt.scriptRuntimes.HasRuntimes() {
		t.Fatal("expected script runtimes to be registered")
	}
	if resolved := rt.scriptRuntimes.Resolve(".py"); resolved == nil {
		t.Fatal("expected .py to resolve")
	}
}

func TestWithScriptBridge(t *testing.T) {
	bridge := map[string]any{
		"model": func(name string) any { return name },
	}
	rt := NewRuntime(WithScriptBridge(bridge))
	defer rt.Close()

	if rt.scriptBridge == nil {
		t.Fatal("expected scriptBridge to be set")
	}
	if _, ok := rt.scriptBridge["model"]; !ok {
		t.Fatal("expected 'model' key in scriptBridge")
	}
}

func TestWithScriptRuntime_MultipleRuntimes(t *testing.T) {
	rt := NewRuntime(
		WithScriptRuntime(newMockPythonRuntime()),
		WithScriptRuntime(newMockNodeRuntime()),
		WithScriptRuntime(newMockGoRuntime()),
	)
	defer rt.Close()

	if len(rt.scriptRuntimes.All()) != 3 {
		t.Fatalf("expected 3 runtimes, got %d", len(rt.scriptRuntimes.All()))
	}
}

func TestRuntime_Close_ClosesScriptRuntimes(t *testing.T) {
	py := newMockPythonRuntime()
	rt := NewRuntime(WithScriptRuntime(py))

	if err := rt.Close(); err != nil {
		t.Fatalf("Close() error: %v", err)
	}
	if !py.closed {
		t.Error("script runtime should be closed after Runtime.Close()")
	}
}
