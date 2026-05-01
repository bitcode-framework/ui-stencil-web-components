package runtime

import (
	"fmt"
	"sync"
	"testing"

	"github.com/bitcode-framework/go-json/stdlib"
)

func newTestRuntime() *Runtime {
	reg := stdlib.DefaultRegistry()
	return NewRuntime(
		WithStdlib(reg.All()),
		WithStdlibEnv(reg.EnvVars()),
	)
}

func TestNewRuntime_Defaults(t *testing.T) {
	rt := NewRuntime()
	if rt == nil {
		t.Fatal("expected non-nil runtime")
	}
	if rt.engine == nil {
		t.Fatal("expected non-nil engine")
	}
}

func TestRuntime_ExecuteJSON_HelloWorld(t *testing.T) {
	rt := newTestRuntime()
	result, err := rt.ExecuteJSON([]byte(`{
		"name": "hello",
		"steps": [{"return": "'Hello, World!'"}]
	}`), nil)
	if err != nil {
		t.Fatalf("execution error: %v", err)
	}
	if result.Value != "Hello, World!" {
		t.Errorf("expected 'Hello, World!', got %v", result.Value)
	}
}

func TestRuntime_ExecuteJSON_WithInput(t *testing.T) {
	rt := newTestRuntime()
	result, err := rt.ExecuteJSON([]byte(`{
		"steps": [{"return": "input.name + ' is ' + string(input.age)"}]
	}`), map[string]any{"name": "Alice", "age": 30})
	if err != nil {
		t.Fatalf("execution error: %v", err)
	}
	if result.Value != "Alice is 30" {
		t.Errorf("expected 'Alice is 30', got %v", result.Value)
	}
}

func TestRuntime_Compile_Cache(t *testing.T) {
	rt := newTestRuntime()
	prog := []byte(`{"steps": [{"return": "42"}]}`)

	p1, err := rt.Compile(prog)
	if err != nil {
		t.Fatalf("first compile error: %v", err)
	}
	p2, err := rt.Compile(prog)
	if err != nil {
		t.Fatalf("second compile error: %v", err)
	}
	if p1 != p2 {
		t.Error("expected same pointer from cache")
	}
}

func TestRuntime_Execute_WithSession(t *testing.T) {
	reg := stdlib.DefaultRegistry()
	rt := NewRuntime(
		WithStdlib(reg.All()),
		WithStdlibEnv(reg.EnvVars()),
		WithSession(&Session{UserID: "user-123", Locale: "en"}),
	)
	result, err := rt.ExecuteJSON([]byte(`{
		"steps": [{"return": "input.session.user_id"}]
	}`), map[string]any{})
	if err != nil {
		t.Fatalf("execution error: %v", err)
	}
	if result.Value != "user-123" {
		t.Errorf("expected 'user-123', got %v", result.Value)
	}
}

func TestRuntime_Execute_WithLimits(t *testing.T) {
	reg := stdlib.DefaultRegistry()
	rt := NewRuntime(
		WithStdlib(reg.All()),
		WithStdlibEnv(reg.EnvVars()),
		WithLimits(Limits{
			MaxDepth: 5, MaxSteps: 10, MaxLoopIterations: 100,
			MaxNodes: 1000, MaxVariables: 100, MaxVariableSize: 1024 * 1024,
			MaxOutputSize: 1024 * 1024, Timeout: 0,
		}),
	)
	_, err := rt.ExecuteJSON([]byte(`{
		"steps": [
			{"let": "i", "value": 0},
			{"while": "i < 100", "steps": [{"set": "i", "expr": "i + 1"}]},
			{"return": "i"}
		]
	}`), nil)
	if err == nil {
		t.Fatal("expected step limit error")
	}
}

func TestRuntime_Execute_Concurrent(t *testing.T) {
	rt := newTestRuntime()
	prog, err := rt.Compile([]byte(`{
		"steps": [{"return": "input.x * 2"}]
	}`))
	if err != nil {
		t.Fatalf("compile error: %v", err)
	}

	var wg sync.WaitGroup
	errors := make(chan error, 20)
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			result, err := rt.Execute(prog, map[string]any{"x": n})
			if err != nil {
				errors <- err
				return
			}
			expected := n * 2
			got, ok := toInt(result.Value)
			if !ok || got != expected {
				errors <- fmt.Errorf("goroutine %d: expected %d, got %v (%T)", n, expected, result.Value, result.Value)
			}
		}(i)
	}
	wg.Wait()
	close(errors)
	for err := range errors {
		t.Error(err)
	}
}

func TestRuntime_Execute_NilInput(t *testing.T) {
	rt := newTestRuntime()
	result, err := rt.ExecuteJSON([]byte(`{
		"steps": [{"return": "'ok'"}]
	}`), nil)
	if err != nil {
		t.Fatalf("execution error: %v", err)
	}
	if result.Value != "ok" {
		t.Errorf("expected 'ok', got %v", result.Value)
	}
}

func TestRuntime_WithoutIO(t *testing.T) {
	rt := NewRuntime(WithoutIO())
	if !rt.IODisabled() {
		t.Error("expected IO to be disabled")
	}
}

func TestRuntime_Resolve_MostRestrictive(t *testing.T) {
	engine := Limits{MaxDepth: 1000, MaxSteps: 10000, MaxLoopIterations: 10000, MaxNodes: 1000, MaxVariables: 1000, MaxVariableSize: 10 * 1024 * 1024, MaxOutputSize: 50 * 1024 * 1024, Timeout: 30_000_000_000}
	program := Limits{MaxDepth: 100, MaxSteps: 50000, MaxLoopIterations: 5000, MaxNodes: 500, MaxVariables: 500, MaxVariableSize: 5 * 1024 * 1024, MaxOutputSize: 25 * 1024 * 1024, Timeout: 60_000_000_000}
	result := Resolve(engine, program)
	if result.MaxDepth != 100 {
		t.Errorf("MaxDepth: expected 100, got %d", result.MaxDepth)
	}
	if result.MaxSteps != 10000 {
		t.Errorf("MaxSteps: expected 10000, got %d", result.MaxSteps)
	}
	if result.MaxLoopIterations != 5000 {
		t.Errorf("MaxLoopIterations: expected 5000, got %d", result.MaxLoopIterations)
	}
	if result.Timeout != 30_000_000_000 {
		t.Errorf("Timeout: expected 30s, got %v", result.Timeout)
	}
}

func toInt(v any) (int, bool) {
	switch val := v.(type) {
	case int:
		return val, true
	case int64:
		return int(val), true
	case float64:
		return int(val), true
	default:
		return 0, false
	}
}

func TestRuntime_WithEnvResolver(t *testing.T) {
	customResolver := func(key string) string {
		if key == "MY_CUSTOM_VAR" {
			return "custom_value"
		}
		return ""
	}

	reg := stdlib.DefaultRegistry()
	rt := NewRuntime(
		WithStdlib(reg.All()),
		WithStdlibEnv(reg.EnvVars()),
		WithEnvHandle(reg.EnvHandle()),
		WithEnvResolver(customResolver),
	)

	result, err := rt.ExecuteJSON([]byte(`{
		"steps": [{"return": "env('MY_CUSTOM_VAR')"}]
	}`), nil)
	if err != nil {
		t.Fatalf("execution error: %v", err)
	}
	if result.Value != "custom_value" {
		t.Errorf("expected 'custom_value', got %v", result.Value)
	}
}

func TestRuntime_WithEnvAccess_Deny(t *testing.T) {
	reg := stdlib.DefaultRegistry()
	rt := NewRuntime(
		WithStdlib(reg.All()),
		WithStdlibEnv(reg.EnvVars()),
		WithEnvHandle(reg.EnvHandle()),
		WithEnvAccess(&stdlib.EnvAccessConfig{
			Deny: []string{"*_SECRET", "*_PASSWORD"},
		}),
	)

	_, err := rt.ExecuteJSON([]byte(`{
		"steps": [{"return": "env('JWT_SECRET')"}]
	}`), nil)
	if err == nil {
		t.Fatal("expected error for denied env key, got nil")
	}
}

func TestRuntime_WithEnvAccess_Allow(t *testing.T) {
	customResolver := func(key string) string {
		if key == "APP_NAME" {
			return "myapp"
		}
		return ""
	}

	reg := stdlib.DefaultRegistry()
	rt := NewRuntime(
		WithStdlib(reg.All()),
		WithStdlibEnv(reg.EnvVars()),
		WithEnvHandle(reg.EnvHandle()),
		WithEnvResolver(customResolver),
		WithEnvAccess(&stdlib.EnvAccessConfig{
			Allow: []string{"APP_*"},
		}),
	)

	result, err := rt.ExecuteJSON([]byte(`{
		"steps": [{"return": "env('APP_NAME')"}]
	}`), nil)
	if err != nil {
		t.Fatalf("execution error: %v", err)
	}
	if result.Value != "myapp" {
		t.Errorf("expected 'myapp', got %v", result.Value)
	}

	_, err = rt.ExecuteJSON([]byte(`{
		"steps": [{"return": "env('DB_HOST')"}]
	}`), nil)
	if err == nil {
		t.Fatal("expected error for non-allowed env key, got nil")
	}
}
