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

func TestScriptImportExecution_CallFunction(t *testing.T) {
	dir := t.TempDir()
	programFile := filepath.Join(dir, "main.json")
	os.WriteFile(programFile, []byte(`{
		"import": {"ml": "script:./predict.py"},
		"steps": [
			{"let": "result", "call": "ml.call", "with": ["'predict'", "42"]},
			{"return": "result"}
		]
	}`), 0644)

	mock := &mockScriptRuntime{
		name:       "python",
		extensions: []string{".py"},
		execFn: func(ctx context.Context, script, function string, params, bridge map[string]any) (any, error) {
			if function != "predict" {
				return nil, fmt.Errorf("unexpected function: %s", function)
			}
			args := params["args"].([]any)
			if len(args) != 1 || args[0] != 42 {
				return nil, fmt.Errorf("unexpected args: %v", args)
			}
			return map[string]any{"prediction": 0.95}, nil
		},
	}

	reg := stdlib.DefaultRegistry()
	rt := NewRuntime(
		WithStdlib(reg.All()),
		WithStdlibEnv(reg.EnvVars()),
		WithScriptRuntime(mock),
	)
	defer rt.Close()

	compiled, err := rt.CompileFile(programFile)
	if err != nil {
		t.Fatalf("compile error: %v", err)
	}

	result, err := rt.Execute(compiled, nil)
	if err != nil {
		t.Fatalf("execute error: %v", err)
	}

	resultMap, ok := result.Value.(map[string]any)
	if !ok {
		t.Fatalf("expected map result, got %T: %v", result.Value, result.Value)
	}
	if resultMap["prediction"] != 0.95 {
		t.Errorf("expected prediction=0.95, got %v", resultMap["prediction"])
	}
}

func TestScriptImportExecution_ExecEntireScript(t *testing.T) {
	dir := t.TempDir()
	programFile := filepath.Join(dir, "main.json")
	os.WriteFile(programFile, []byte(`{
		"import": {"util": "script:./helper.js"},
		"steps": [
			{"let": "result", "call": "util.exec", "with": ["'hello'"]},
			{"return": "result"}
		]
	}`), 0644)

	mock := &mockScriptRuntime{
		name:       "goja",
		extensions: []string{".js"},
		execFn: func(ctx context.Context, script, function string, params, bridge map[string]any) (any, error) {
			if function != "" {
				return nil, fmt.Errorf("expected empty function for exec, got: %s", function)
			}
			return "executed", nil
		},
	}

	reg := stdlib.DefaultRegistry()
	rt := NewRuntime(
		WithStdlib(reg.All()),
		WithStdlibEnv(reg.EnvVars()),
		WithScriptRuntime(mock),
	)
	defer rt.Close()

	compiled, err := rt.CompileFile(programFile)
	if err != nil {
		t.Fatalf("compile error: %v", err)
	}

	result, err := rt.Execute(compiled, nil)
	if err != nil {
		t.Fatalf("execute error: %v", err)
	}

	if result.Value != "executed" {
		t.Errorf("expected 'executed', got %v", result.Value)
	}
}

func TestScriptImportExecution_BridgePassed(t *testing.T) {
	dir := t.TempDir()
	programFile := filepath.Join(dir, "main.json")
	os.WriteFile(programFile, []byte(`{
		"import": {"plugin": "script:./plugin.py"},
		"steps": [
			{"let": "result", "call": "plugin.call", "with": ["'useBridge'"]},
			{"return": "result"}
		]
	}`), 0644)

	var receivedBridge map[string]any
	mock := &mockScriptRuntime{
		name:       "python",
		extensions: []string{".py"},
		execFn: func(ctx context.Context, script, function string, params, bridge map[string]any) (any, error) {
			receivedBridge = bridge
			return "ok", nil
		},
	}

	bridge := map[string]any{
		"model": func(name string) any { return name },
		"db":    map[string]any{"query": func(sql string) any { return nil }},
	}

	reg := stdlib.DefaultRegistry()
	rt := NewRuntime(
		WithStdlib(reg.All()),
		WithStdlibEnv(reg.EnvVars()),
		WithScriptRuntime(mock),
		WithScriptBridge(bridge),
	)
	defer rt.Close()

	compiled, err := rt.CompileFile(programFile)
	if err != nil {
		t.Fatalf("compile error: %v", err)
	}

	_, err = rt.Execute(compiled, nil)
	if err != nil {
		t.Fatalf("execute error: %v", err)
	}

	if receivedBridge == nil {
		t.Fatal("bridge was not passed to runtime")
	}
	if _, ok := receivedBridge["model"]; !ok {
		t.Error("expected 'model' in bridge")
	}
	if _, ok := receivedBridge["db"]; !ok {
		t.Error("expected 'db' in bridge")
	}
}

func TestScriptImportExecution_RuntimeNotRegistered(t *testing.T) {
	dir := t.TempDir()
	programFile := filepath.Join(dir, "main.json")
	os.WriteFile(programFile, []byte(`{
		"import": {"ml": "script:./predict.py"},
		"steps": [{"return": "1"}]
	}`), 0644)

	reg := stdlib.DefaultRegistry()
	rt := NewRuntime(
		WithStdlib(reg.All()),
		WithStdlibEnv(reg.EnvVars()),
	)
	defer rt.Close()

	compiled, err := rt.CompileFile(programFile)
	if err != nil {
		t.Fatalf("compile error: %v", err)
	}

	_, err = rt.Execute(compiled, nil)
	if err == nil {
		t.Fatal("expected error for unregistered runtime")
	}
	if !contains(err.Error(), "no script runtime registered for extension '.py'") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestScriptImportExecution_RuntimeNotAvailable(t *testing.T) {
	dir := t.TempDir()
	programFile := filepath.Join(dir, "main.json")
	os.WriteFile(programFile, []byte(`{
		"import": {"ml": "script:./predict.py"},
		"steps": [{"return": "1"}]
	}`), 0644)

	mock := &mockScriptRuntime{
		name:       "python",
		extensions: []string{".py"},
		validateFn: func() error {
			return fmt.Errorf("Python 3.10+ required, found 3.8")
		},
	}

	reg := stdlib.DefaultRegistry()
	rt := NewRuntime(
		WithStdlib(reg.All()),
		WithStdlibEnv(reg.EnvVars()),
		WithScriptRuntime(mock),
	)
	defer rt.Close()

	compiled, err := rt.CompileFile(programFile)
	if err != nil {
		t.Fatalf("compile error: %v", err)
	}

	_, err = rt.Execute(compiled, nil)
	if err == nil {
		t.Fatal("expected error for unavailable runtime")
	}
	if !contains(err.Error(), "script runtime 'python' not available") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestScriptImportExecution_AbsolutePathRejected(t *testing.T) {
	dir := t.TempDir()
	programFile := filepath.Join(dir, "main.json")

	absPath := "/etc/evil.py"
	if filepath.VolumeName(dir) != "" {
		absPath = "C:/evil.py"
	}

	programJSON := fmt.Sprintf(`{
		"import": {"ml": "script:%s"},
		"steps": [{"return": "1"}]
	}`, absPath)
	os.WriteFile(programFile, []byte(programJSON), 0644)

	mock := newMockPythonRuntime()
	reg := stdlib.DefaultRegistry()
	rt := NewRuntime(
		WithStdlib(reg.All()),
		WithStdlibEnv(reg.EnvVars()),
		WithScriptRuntime(mock),
	)
	defer rt.Close()

	compiled, err := rt.CompileFile(programFile)
	if err != nil {
		t.Fatalf("compile error: %v", err)
	}

	_, err = rt.Execute(compiled, nil)
	if err == nil {
		t.Fatal("expected error for absolute path")
	}
	if !contains(err.Error(), "script: import path must be relative") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestScriptImportExecution_PathTraversalRejected(t *testing.T) {
	dir := t.TempDir()
	programFile := filepath.Join(dir, "main.json")
	os.WriteFile(programFile, []byte(`{
		"import": {"ml": "script:../../../etc/passwd.py"},
		"steps": [{"return": "1"}]
	}`), 0644)

	mock := newMockPythonRuntime()
	reg := stdlib.DefaultRegistry()
	rt := NewRuntime(
		WithStdlib(reg.All()),
		WithStdlibEnv(reg.EnvVars()),
		WithScriptRuntime(mock),
	)
	defer rt.Close()

	compiled, err := rt.CompileFile(programFile)
	if err != nil {
		t.Fatalf("compile error: %v", err)
	}

	_, err = rt.Execute(compiled, nil)
	if err == nil {
		t.Fatal("expected error for path traversal")
	}
	if !contains(err.Error(), "script: import path escapes base directory") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestScriptImportExecution_WithoutSourcePath(t *testing.T) {
	mock := &mockScriptRuntime{
		name:       "goja",
		extensions: []string{".js"},
		execFn: func(ctx context.Context, script, function string, params, bridge map[string]any) (any, error) {
			return "ok", nil
		},
	}

	reg := stdlib.DefaultRegistry()
	rt := NewRuntime(
		WithStdlib(reg.All()),
		WithStdlibEnv(reg.EnvVars()),
		WithScriptRuntime(mock),
	)
	defer rt.Close()

	program, err := lang.Parse([]byte(`{
		"import": {"util": "script:./helper.js"},
		"steps": [
			{"let": "r", "call": "util.call", "with": ["'test'"]},
			{"return": "r"}
		]
	}`))
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	resolver := lang.NewImportResolver()
	_ = resolver.ResolveImports(program, "", nil)

	compiled, err := lang.Compile(program, rt.engine, rt.limits.ToResolved())
	if err != nil {
		t.Fatalf("compile error: %v", err)
	}

	result, err := rt.Execute(compiled, nil)
	if err != nil {
		t.Fatalf("execute error: %v", err)
	}
	if result.Value != "ok" {
		t.Errorf("expected 'ok', got %v", result.Value)
	}
}

func TestScriptImportExecution_ScriptPathResolution(t *testing.T) {
	dir := t.TempDir()
	subdir := filepath.Join(dir, "plugins")
	os.MkdirAll(subdir, 0755)
	programFile := filepath.Join(dir, "main.json")
	os.WriteFile(programFile, []byte(`{
		"import": {"ml": "script:./plugins/predict.py"},
		"steps": [
			{"let": "r", "call": "ml.call", "with": ["'test'"]},
			{"return": "r"}
		]
	}`), 0644)

	var receivedScript string
	mock := &mockScriptRuntime{
		name:       "python",
		extensions: []string{".py"},
		execFn: func(ctx context.Context, script, function string, params, bridge map[string]any) (any, error) {
			receivedScript = script
			return "ok", nil
		},
	}

	reg := stdlib.DefaultRegistry()
	rt := NewRuntime(
		WithStdlib(reg.All()),
		WithStdlibEnv(reg.EnvVars()),
		WithScriptRuntime(mock),
	)
	defer rt.Close()

	compiled, err := rt.CompileFile(programFile)
	if err != nil {
		t.Fatalf("compile error: %v", err)
	}

	_, err = rt.Execute(compiled, nil)
	if err != nil {
		t.Fatalf("execute error: %v", err)
	}

	expectedPath := filepath.Join(dir, "plugins", "predict.py")
	if receivedScript != expectedPath {
		t.Errorf("expected script path %q, got %q", expectedPath, receivedScript)
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsSubstring(s, substr))
}

func containsSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
