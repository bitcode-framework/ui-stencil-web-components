package native

import (
	"context"
	"fmt"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
)

// Runtime implements the ScriptRuntime interface for native Go plugins (.so/.dylib).
// Only supported on Linux and macOS. Windows is not supported by Go's plugin package.
type Runtime struct {
	plugins map[string]*loadedPlugin
	config  Config
	mu      sync.RWMutex
}

type loadedPlugin struct {
	functions map[string]func(map[string]any) (any, error)
	manifest  []string
}

// New creates a new native plugin runtime.
// Returns an error on Windows since Go's plugin package doesn't support it.
func New(config Config) (*Runtime, error) {
	if runtime.GOOS == "windows" {
		return nil, fmt.Errorf("native plugins not supported on Windows (use wasm: instead)")
	}
	return &Runtime{
		plugins: make(map[string]*loadedPlugin),
		config:  config,
	}, nil
}

func (n *Runtime) Name() string { return "native" }

func (n *Runtime) Extensions() []string {
	switch runtime.GOOS {
	case "linux":
		return []string{".so"}
	case "darwin":
		return []string{".dylib", ".so"}
	default:
		return []string{".so"}
	}
}

func (n *Runtime) CanHandle(ext string) bool {
	for _, e := range n.Extensions() {
		if e == ext {
			return true
		}
	}
	return false
}

func (n *Runtime) Execute(ctx context.Context, script, function string, params, bridge map[string]any) (any, error) {
	loaded, err := n.getOrLoad(script)
	if err != nil {
		return nil, err
	}

	fn, ok := loaded.functions[function]
	if !ok {
		return nil, fmt.Errorf("native plugin function '%s' not found in %s (available: %s)",
			function, filepath.Base(script), strings.Join(loaded.manifest, ", "))
	}

	type result struct {
		value any
		err   error
	}
	done := make(chan result, 1)
	go func() {
		v, e := fn(params)
		done <- result{v, e}
	}()

	select {
	case r := <-done:
		return r.value, r.err
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

func (n *Runtime) Validate() error {
	if runtime.GOOS == "windows" {
		return fmt.Errorf("native plugins not supported on Windows (use wasm: instead)")
	}
	return nil
}

func (n *Runtime) Close() error { return nil }

func (n *Runtime) getOrLoad(path string) (*loadedPlugin, error) {
	absPath, err := filepath.Abs(path)
	if err != nil {
		absPath = path
	}

	n.mu.RLock()
	if loaded, ok := n.plugins[absPath]; ok {
		n.mu.RUnlock()
		return loaded, nil
	}
	n.mu.RUnlock()

	if err := n.validatePath(absPath); err != nil {
		return nil, err
	}

	loaded, err := n.loadPlugin(absPath)
	if err != nil {
		return nil, err
	}

	n.mu.Lock()
	n.plugins[absPath] = loaded
	n.mu.Unlock()

	return loaded, nil
}

func (n *Runtime) validatePath(absPath string) error {
	if len(n.config.AllowedDirs) == 0 {
		return nil
	}
	for _, dir := range n.config.AllowedDirs {
		allowedAbs, err := filepath.Abs(dir)
		if err != nil {
			continue
		}
		if strings.HasPrefix(absPath, allowedAbs+string(filepath.Separator)) || absPath == allowedAbs {
			return nil
		}
	}
	return fmt.Errorf("plugin path '%s' not in allowed directories", filepath.Base(absPath))
}
