package runtimes

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	goruntime "github.com/bitcode-framework/go-json/runtime"
)

// RuntimeSet aggregates all configured runtimes and implements go-json's ScriptRuntime interface.
type RuntimeSet struct {
	embedded []EmbeddedRuntime
	external []ExternalRuntime
	registry *EngineRegistry
}

// New creates a configured runtime set from the given options.
func New(opts ...RuntimeOption) (*RuntimeSet, error) {
	rs := &runtimeSet{}
	for _, opt := range opts {
		opt(rs)
	}
	return rs.build()
}

func (rs *runtimeSet) build() (*RuntimeSet, error) {
	set := &RuntimeSet{
		registry: NewEngineRegistry(),
	}
	return set, nil
}

// AsScriptRuntimes returns ScriptRuntime adapters for registering with go-json core.
func (s *RuntimeSet) AsScriptRuntimes() []goruntime.ScriptRuntime {
	var rts []goruntime.ScriptRuntime
	for _, emb := range s.embedded {
		rts = append(rts, &embeddedAdapter{runtime: emb, set: s})
	}
	for _, ext := range s.external {
		rts = append(rts, &externalAdapter{runtime: ext})
	}
	return rts
}

// Close releases all resources.
func (s *RuntimeSet) Close() error {
	var firstErr error
	for _, ext := range s.external {
		if err := ext.Close(); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	return firstErr
}

type embeddedAdapter struct {
	runtime EmbeddedRuntime
	set     *RuntimeSet
}

func (a *embeddedAdapter) Name() string { return a.runtime.Name() }

func (a *embeddedAdapter) Extensions() []string {
	switch a.runtime.Name() {
	case "goja":
		return []string{".js"}
	case "quickjs":
		return []string{".js"}
	case "yaegi":
		return []string{".go"}
	default:
		return nil
	}
}

func (a *embeddedAdapter) CanHandle(ext string) bool {
	for _, e := range a.Extensions() {
		if e == ext {
			return true
		}
	}
	return false
}

func (a *embeddedAdapter) Execute(ctx context.Context, script string, function string, params map[string]any, bridge map[string]any) (any, error) {
	return ExecuteEmbedded(ctx, a.runtime, script, function, params, bridge, 0)
}

func (a *embeddedAdapter) Validate() error { return nil }
func (a *embeddedAdapter) Close() error    { return nil }

type externalAdapter struct {
	runtime ExternalRuntime
}

func (a *externalAdapter) Name() string { return a.runtime.Name() }

func (a *externalAdapter) Extensions() []string {
	switch a.runtime.Name() {
	case "node":
		return []string{".ts", ".mjs"}
	case "python":
		return []string{".py", ".pyw"}
	default:
		return nil
	}
}

func (a *externalAdapter) CanHandle(ext string) bool {
	for _, e := range a.Extensions() {
		if e == ext {
			return true
		}
	}
	return false
}

func (a *externalAdapter) Execute(ctx context.Context, script string, function string, params map[string]any, bridge map[string]any) (any, error) {
	return a.runtime.Execute(ctx, script, function, params, bridge)
}

func (a *externalAdapter) Validate() error { return a.runtime.Validate() }
func (a *externalAdapter) Close() error    { return a.runtime.Close() }

// EngineRegistry maps engine names to EmbeddedRuntime implementations.
type EngineRegistry struct {
	engines map[string]EmbeddedRuntime
}

func NewEngineRegistry() *EngineRegistry {
	return &EngineRegistry{engines: make(map[string]EmbeddedRuntime)}
}

func (r *EngineRegistry) Register(name string, rt EmbeddedRuntime) {
	r.engines[name] = rt
}

func (r *EngineRegistry) Get(name string) EmbeddedRuntime {
	return r.engines[name]
}

func (r *EngineRegistry) Resolve(runtimeField string, defaultEngine string) EmbeddedRuntime {
	engine := ParseEngine(runtimeField)
	if engine == "" {
		engine = defaultEngine
	}
	if engine == "" {
		engine = "goja"
	}
	return r.engines[engine]
}

// ParseEngine extracts the engine name from a runtime field value.
// Examples: "javascript" → "", "javascript:goja" → "goja", "go:yaegi" → "yaegi"
func ParseEngine(runtimeField string) string {
	if !strings.Contains(runtimeField, ":") {
		return ""
	}
	parts := strings.SplitN(runtimeField, ":", 2)
	return parts[1]
}

// ExecuteEmbedded runs a script using an embedded runtime with timeout and panic recovery.
func ExecuteEmbedded(ctx context.Context, rt EmbeddedRuntime, scriptPath string, function string, params map[string]any, bridge map[string]any, timeout_unused int) (any, error) {
	code, err := LoadScript(scriptPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load script '%s': %w", filepath.Base(scriptPath), err)
	}

	vm, err := rt.NewVM(VMOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to create VM: %w", err)
	}
	defer vm.Close()

	if err := vm.InjectBridge(bridge); err != nil {
		return nil, fmt.Errorf("failed to inject bridge: %w", err)
	}

	execParams := params
	if function != "" {
		if execParams == nil {
			execParams = make(map[string]any)
		}
		execParams["__function"] = function
	}

	if err := vm.InjectParams(execParams); err != nil {
		return nil, fmt.Errorf("failed to inject params: %w", err)
	}

	type execResult struct {
		value any
		err   error
	}

	done := make(chan execResult, 1)
	go func() {
		defer func() {
			if r := recover(); r != nil {
				done <- execResult{nil, fmt.Errorf("panic in script '%s': %v", filepath.Base(scriptPath), r)}
			}
		}()
		result, execErr := vm.Execute(code, scriptPath)
		done <- execResult{result, execErr}
	}()

	select {
	case <-ctx.Done():
		vm.Interrupt("context cancelled")
		return nil, ctx.Err()
	case res := <-done:
		return res.value, res.err
	}
}
