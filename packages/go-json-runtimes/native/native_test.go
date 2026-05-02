package native

import (
	"context"
	"path/filepath"
	"runtime"
	"testing"
)

func TestNativeRuntime_Name(t *testing.T) {
	rt := &Runtime{plugins: make(map[string]*loadedPlugin)}
	if rt.Name() != "native" {
		t.Errorf("expected name 'native', got %q", rt.Name())
	}
}

func TestNativeRuntime_Extensions(t *testing.T) {
	rt := &Runtime{plugins: make(map[string]*loadedPlugin)}
	exts := rt.Extensions()
	if len(exts) == 0 {
		t.Fatal("expected at least one extension")
	}

	switch runtime.GOOS {
	case "linux":
		if exts[0] != ".so" {
			t.Errorf("expected .so on Linux, got %v", exts)
		}
	case "darwin":
		found := false
		for _, e := range exts {
			if e == ".dylib" || e == ".so" {
				found = true
			}
		}
		if !found {
			t.Errorf("expected .dylib or .so on macOS, got %v", exts)
		}
	}
}

func TestNativeRuntime_CanHandle(t *testing.T) {
	rt := &Runtime{plugins: make(map[string]*loadedPlugin)}

	if !rt.CanHandle(".so") {
		t.Error("expected CanHandle(.so) = true")
	}
	if rt.CanHandle(".wasm") {
		t.Error("expected CanHandle(.wasm) = false")
	}
	if rt.CanHandle(".js") {
		t.Error("expected CanHandle(.js) = false")
	}
}

func TestNativeRuntime_Validate(t *testing.T) {
	rt := &Runtime{plugins: make(map[string]*loadedPlugin)}
	err := rt.Validate()

	if runtime.GOOS == "windows" {
		if err == nil {
			t.Error("expected error on Windows")
		}
	} else {
		if err != nil {
			t.Errorf("expected nil on %s, got: %v", runtime.GOOS, err)
		}
	}
}

func TestNativeRuntime_Close(t *testing.T) {
	rt := &Runtime{plugins: make(map[string]*loadedPlugin)}
	if err := rt.Close(); err != nil {
		t.Errorf("Close() error: %v", err)
	}
}

func TestNativeRuntime_New_Windows(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("Windows-only test")
	}
	_, err := New(Config{})
	if err == nil {
		t.Fatal("expected error on Windows")
	}
}

func TestNativeRuntime_PathValidation(t *testing.T) {
	tmpDir := t.TempDir()
	allowedDir := filepath.Join(tmpDir, "allowed", "plugins")

	rt := &Runtime{
		plugins: make(map[string]*loadedPlugin),
		config: Config{
			AllowedDirs: []string{allowedDir},
		},
	}

	if err := rt.validatePath(filepath.Join(allowedDir, "test.so")); err != nil {
		t.Errorf("expected path in allowed dir to pass, got: %v", err)
	}

	if err := rt.validatePath(filepath.Join(tmpDir, "not-allowed", "test.so")); err == nil {
		t.Error("expected error for path outside allowed dirs")
	}
}

func TestNativeRuntime_PathValidation_NoRestriction(t *testing.T) {
	rt := &Runtime{
		plugins: make(map[string]*loadedPlugin),
		config:  Config{AllowedDirs: nil},
	}

	if err := rt.validatePath("/any/path/test.so"); err != nil {
		t.Errorf("expected no restriction when AllowedDirs is empty, got: %v", err)
	}
}

func TestNativeRuntime_PathValidation_MultipleAllowedDirs(t *testing.T) {
	tmpDir := t.TempDir()
	dir1 := filepath.Join(tmpDir, "plugins1")
	dir2 := filepath.Join(tmpDir, "plugins2")

	rt := &Runtime{
		plugins: make(map[string]*loadedPlugin),
		config: Config{
			AllowedDirs: []string{dir1, dir2},
		},
	}

	if err := rt.validatePath(filepath.Join(dir1, "test.so")); err != nil {
		t.Errorf("expected dir1 to be allowed, got: %v", err)
	}
	if err := rt.validatePath(filepath.Join(dir2, "test.so")); err != nil {
		t.Errorf("expected dir2 to be allowed, got: %v", err)
	}
	if err := rt.validatePath(filepath.Join(tmpDir, "other", "test.so")); err == nil {
		t.Error("expected error for path outside allowed dirs")
	}
}

func TestNativeRuntime_Execute_ContextCancellation(t *testing.T) {
	tmpDir := t.TempDir()
	pluginPath := filepath.Join(tmpDir, "slow.so")
	absPath, _ := filepath.Abs(pluginPath)

	rt := &Runtime{
		plugins: make(map[string]*loadedPlugin),
		config:  Config{},
	}

	rt.mu.Lock()
	rt.plugins[absPath] = &loadedPlugin{
		manifest:  []string{"SlowFunc"},
		functions: map[string]func(map[string]any) (any, error){
			"SlowFunc": func(args map[string]any) (any, error) {
				select {}
			},
		},
	}
	rt.mu.Unlock()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := rt.Execute(ctx, pluginPath, "SlowFunc", nil, nil)
	if err == nil {
		t.Fatal("expected error for cancelled context")
	}
}

func TestNativeRuntime_Execute_FunctionNotFound(t *testing.T) {
	tmpDir := t.TempDir()
	pluginPath := filepath.Join(tmpDir, "plugin.so")
	absPath, _ := filepath.Abs(pluginPath)

	rt := &Runtime{
		plugins: make(map[string]*loadedPlugin),
		config:  Config{},
	}

	rt.mu.Lock()
	rt.plugins[absPath] = &loadedPlugin{
		manifest:  []string{"Compute"},
		functions: map[string]func(map[string]any) (any, error){
			"Compute": func(args map[string]any) (any, error) {
				return 42, nil
			},
		},
	}
	rt.mu.Unlock()

	_, err := rt.Execute(context.Background(), pluginPath, "NonExistent", nil, nil)
	if err == nil {
		t.Fatal("expected error for missing function")
	}
}

func TestNativeRuntime_Execute_Success(t *testing.T) {
	tmpDir := t.TempDir()
	pluginPath := filepath.Join(tmpDir, "plugin.so")
	absPath, _ := filepath.Abs(pluginPath)

	rt := &Runtime{
		plugins: make(map[string]*loadedPlugin),
		config:  Config{},
	}

	rt.mu.Lock()
	rt.plugins[absPath] = &loadedPlugin{
		manifest:  []string{"Compute"},
		functions: map[string]func(map[string]any) (any, error){
			"Compute": func(args map[string]any) (any, error) {
				return map[string]any{"result": 42}, nil
			},
		},
	}
	rt.mu.Unlock()

	result, err := rt.Execute(context.Background(), pluginPath, "Compute", map[string]any{"x": 1}, nil)
	if err != nil {
		t.Fatalf("execute error: %v", err)
	}

	m, ok := result.(map[string]any)
	if !ok {
		t.Fatalf("expected map result, got %T", result)
	}
	if m["result"] != 42 {
		t.Errorf("expected result=42, got %v", m["result"])
	}
}

func TestNativeRuntime_PluginCache(t *testing.T) {
	tmpDir := t.TempDir()
	pluginPath := filepath.Join(tmpDir, "cached.so")
	absPath, _ := filepath.Abs(pluginPath)

	rt := &Runtime{
		plugins: make(map[string]*loadedPlugin),
		config:  Config{},
	}

	rt.mu.Lock()
	rt.plugins[absPath] = &loadedPlugin{
		manifest:  []string{"Test"},
		functions: map[string]func(map[string]any) (any, error){
			"Test": func(args map[string]any) (any, error) { return "cached", nil },
		},
	}
	rt.mu.Unlock()

	loaded1, err := rt.getOrLoad(pluginPath)
	if err != nil {
		t.Fatalf("first load error: %v", err)
	}

	loaded2, err := rt.getOrLoad(pluginPath)
	if err != nil {
		t.Fatalf("second load error: %v", err)
	}

	if loaded1 != loaded2 {
		t.Error("expected same cached plugin instance")
	}
}
