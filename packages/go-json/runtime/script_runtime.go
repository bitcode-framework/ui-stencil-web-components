package runtime

import "context"

// ScriptRuntime defines the interface for external script execution engines.
// Implementations live in go-json-runtimes package (or any custom implementation).
// go-json core defines this interface but never imports implementations.
type ScriptRuntime interface {
	// Name returns the runtime identifier (e.g., "python", "node", "goja").
	Name() string

	// Extensions returns file extensions this runtime can handle (e.g., [".py", ".pyw"]).
	Extensions() []string

	// CanHandle reports whether this runtime can execute scripts with the given extension.
	CanHandle(extension string) bool

	// Execute runs a script file with the given parameters and bridge.
	// - ctx: execution context (for cancellation/timeout)
	// - script: path to the script file (absolute or relative to working dir)
	// - function: function name to call within the script (empty = execute entire script)
	// - params: input parameters passed to the script as JSON-compatible map
	// - bridge: host functions exposed to the script (map[string]any, not typed)
	// Returns the script's return value (JSON-compatible) or error.
	Execute(ctx context.Context, script string, function string, params map[string]any, bridge map[string]any) (any, error)

	// Validate checks if the runtime is available (e.g., Python installed, correct version).
	// Returns nil if ready, descriptive error if not.
	Validate() error

	// Close releases all resources held by this runtime (process pools, VMs, etc.).
	Close() error
}

// ScriptRuntimeRegistry manages multiple ScriptRuntime implementations.
// It provides resolution by file extension or by name.
type ScriptRuntimeRegistry struct {
	runtimes []ScriptRuntime
}

// NewScriptRuntimeRegistry creates an empty registry.
func NewScriptRuntimeRegistry() *ScriptRuntimeRegistry {
	return &ScriptRuntimeRegistry{}
}

// Register adds a runtime to the registry.
// Runtimes are resolved in registration order — first match wins.
func (r *ScriptRuntimeRegistry) Register(rt ScriptRuntime) {
	r.runtimes = append(r.runtimes, rt)
}

// Resolve finds a runtime that can handle the given file extension.
// Returns nil if no runtime can handle it.
func (r *ScriptRuntimeRegistry) Resolve(extension string) ScriptRuntime {
	for _, rt := range r.runtimes {
		if rt.CanHandle(extension) {
			return rt
		}
	}
	return nil
}

// ResolveByName finds a runtime by name.
// Returns nil if no runtime with that name is registered.
func (r *ScriptRuntimeRegistry) ResolveByName(name string) ScriptRuntime {
	for _, rt := range r.runtimes {
		if rt.Name() == name {
			return rt
		}
	}
	return nil
}

// All returns all registered runtimes.
func (r *ScriptRuntimeRegistry) All() []ScriptRuntime {
	return r.runtimes
}

// Close closes all registered runtimes, releasing their resources.
// Returns the first error encountered (but still attempts to close all).
func (r *ScriptRuntimeRegistry) Close() error {
	var firstErr error
	for _, rt := range r.runtimes {
		if err := rt.Close(); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	return firstErr
}

// HasRuntimes reports whether any runtimes are registered.
func (r *ScriptRuntimeRegistry) HasRuntimes() bool {
	return len(r.runtimes) > 0
}
