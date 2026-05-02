# Native Plugin Guide

## Overview

Native plugins are shared libraries (`.so` / `.dylib`) loaded via Go's `plugin` package. They run **in-process** with zero serialization overhead — function calls are direct Go function invocations.

**When to use native plugins:**

- Maximum performance (no serialization, no IPC, no sandbox overhead)
- Plugins that need full Go standard library access
- Trusted first-party code that you build and control
- Linux/macOS server deployments

**When NOT to use native plugins:**

- Windows deployments (not supported)
- Untrusted or third-party code (no sandbox — use WASM instead)
- Cross-platform distribution (use WASM instead)
- Plugins that might crash (crashes take down the host process)

## Platform Support

| Platform | Status | Extension | Notes |
|----------|--------|-----------|-------|
| Linux (amd64, arm64) | ✅ Supported | `.so` | Full support |
| macOS (amd64, arm64) | ✅ Supported | `.dylib`, `.so` | Full support |
| Windows | ❌ Not supported | — | Go's `plugin` package doesn't support Windows |

> **Windows users:** Use WASM plugins instead. The runtime returns a clear error: `"native plugins not supported on Windows (use wasm: instead)"`

## Quick Start

### 1. Write a plugin

```go
// plugins/math_plugin.go
package main

// Manifest declares which functions this plugin exports.
// Required — the runtime calls this first to discover available functions.
func Manifest() []string {
    return []string{"Add", "Multiply"}
}

// Add adds two numbers from the args map.
func Add(args map[string]any) (any, error) {
    a, _ := args["a"].(float64)
    b, _ := args["b"].(float64)
    return map[string]any{"result": a + b}, nil
}

// Multiply multiplies two numbers.
func Multiply(args map[string]any) (any, error) {
    a, _ := args["a"].(float64)
    b, _ := args["b"].(float64)
    return map[string]any{"result": a * b}, nil
}

func main() {} // Required but unused
```

### 2. Build the plugin

```bash
go build -buildmode=plugin -o plugins/math_plugin.so plugins/math_plugin.go
```

> **Important:** The plugin must be built with the **same Go version** and **same module dependencies** as the host application.

### 3. Call from go-json

```jsonc
{
  "steps": [
    {
      "type": "script",
      "script": "plugins/math_plugin.so",
      "function": "Add",
      "args": { "a": 10, "b": 20 }
    }
  ]
}
```

### 4. Configure the runtime (Go)

```go
import "github.com/bitcode-framework/go-json-runtimes"

runtimes.WithNative(runtimes.NativeRuntimeConfig{
    Enabled:     true,
    AllowedDirs: []string{"./plugins"},
})
```

## Plugin Protocol

### Required: `Manifest()`

Every native plugin must export a `Manifest` function:

```go
func Manifest() []string
```

Returns a list of function names that the plugin exports. The runtime uses this to:
1. Discover available functions at load time
2. Look up each declared symbol via `plugin.Lookup()`
3. Validate that all declared functions have the correct signature

### Required: Function Signature

Every function listed in `Manifest()` must have this exact signature:

```go
func FunctionName(args map[string]any) (any, error)
```

| Parameter | Type | Description |
|-----------|------|-------------|
| `args` | `map[string]any` | JSON-deserialized arguments from the caller |
| Return 1 | `any` | Result value (will be JSON-serialized for the caller) |
| Return 2 | `error` | Error (nil on success) |

### Load Sequence

```
1. plugin.Open(path)           → load shared library
2. Lookup("Manifest")          → get manifest function
3. Manifest()                  → get list of function names
4. Lookup(name) for each name  → resolve function symbols
5. Validate signatures         → ensure func(map[string]any) (any, error)
6. Cache loaded plugin         → subsequent calls skip steps 1-5
```

## Writing Plugins

### Full Example

```go
// plugins/text_plugin.go
package main

import (
    "fmt"
    "strings"
)

func Manifest() []string {
    return []string{"ToUpper", "Repeat", "WordCount"}
}

func ToUpper(args map[string]any) (any, error) {
    text, ok := args["text"].(string)
    if !ok {
        return nil, fmt.Errorf("'text' argument required (string)")
    }
    return map[string]any{"result": strings.ToUpper(text)}, nil
}

func Repeat(args map[string]any) (any, error) {
    text, _ := args["text"].(string)
    count := int(args["count"].(float64)) // JSON numbers are float64
    if count < 0 || count > 1000 {
        return nil, fmt.Errorf("count must be 0-1000")
    }
    return map[string]any{"result": strings.Repeat(text, count)}, nil
}

func WordCount(args map[string]any) (any, error) {
    text, _ := args["text"].(string)
    words := strings.Fields(text)
    return map[string]any{"count": len(words)}, nil
}

func main() {}
```

### Build Command

```bash
# Linux
go build -buildmode=plugin -o text_plugin.so text_plugin.go

# macOS
go build -buildmode=plugin -o text_plugin.dylib text_plugin.go
```

### Go Version Requirement

The plugin **must** be compiled with:
- The **same Go version** as the host binary
- The **same module path** and dependency versions for any shared packages
- `CGO_ENABLED=1` (plugins require cgo)

```bash
# Check host Go version
go version

# Build plugin with matching version
go build -buildmode=plugin -o plugin.so ./plugin/
```

> **Tip:** Build plugins in the same CI pipeline as the host to guarantee version alignment.

## Configuration

```go
type NativeRuntimeConfig struct {
    Enabled     bool     // Enable native plugin runtime
    AllowedDirs []string // Restrict plugin loading to these directories
}
```

### BitCode YAML

```yaml
runtimes:
  native:
    enabled: true
    allowed_dirs:
      - ./plugins
      - /opt/bitcode/plugins
```

| Field | Default | Description |
|-------|---------|-------------|
| `Enabled` | `true` | Enable/disable native plugin loading |
| `AllowedDirs` | `[]` (unrestricted) | If set, plugins can only be loaded from these directories. Empty = allow all paths. |

### Path Validation

When `AllowedDirs` is configured, the runtime validates that the plugin's absolute path starts with one of the allowed directory prefixes. This prevents path traversal attacks:

```go
// Allowed: ./plugins/math.so (within AllowedDirs)
// Blocked: ../../etc/malicious.so (outside AllowedDirs)
```

## Security

> **⚠️ WARNING: Native plugins have NO sandbox.**

### Risks

| Risk | Impact | Mitigation |
|------|--------|------------|
| **In-process execution** | Plugin code runs in the same address space as the host | Only load trusted plugins |
| **No memory isolation** | A plugin can read/write any host memory | Code review all plugins |
| **Crash propagation** | A panic/segfault in a plugin crashes the entire host process | Use WASM for untrusted code |
| **No resource limits** | Plugins can consume unlimited CPU/memory | Enforce timeouts via `context.Context` |
| **Full system access** | Plugins can access filesystem, network, env vars | Restrict with `AllowedDirs`, OS-level controls |

### Recommendations

1. **Only load plugins you built yourself** — never load untrusted `.so` files
2. **Use `AllowedDirs`** — restrict where plugins can be loaded from
3. **Pin Go versions** — mismatched versions cause silent corruption or panics
4. **Prefer WASM** for third-party or user-supplied plugins
5. **Use context timeouts** — the runtime respects `context.Context` cancellation

### Context Cancellation

The runtime wraps plugin execution in a goroutine and respects context cancellation:

```go
ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
defer cancel()

result, err := runtime.Execute(ctx, "plugin.so", "SlowFunc", args, bridge)
// Returns ctx.Err() if timeout exceeded
```

> **Note:** Context cancellation does NOT kill the goroutine — it only unblocks the caller. A misbehaving plugin can still leak goroutines.

## Troubleshooting

### "native plugins not supported on Windows"

Go's `plugin` package only works on Linux and macOS. On Windows, use WASM plugins instead:

```yaml
runtimes:
  wasm:
    enabled: true
  native:
    enabled: false  # or omit entirely
```

### "plugin 'X' missing Manifest() export"

Your plugin doesn't export a `Manifest` function. Add:

```go
func Manifest() []string {
    return []string{"YourFunction"}
}
```

### "Manifest has wrong signature"

The `Manifest` function must return `[]string`, not `[]any` or anything else:

```go
// ✅ Correct
func Manifest() []string { return []string{"Foo"} }

// ❌ Wrong
func Manifest() interface{} { return []string{"Foo"} }
```

### "function 'X' has wrong signature"

Every exported function must match exactly:

```go
func Name(map[string]any) (any, error)
```

Common mistakes:
```go
// ❌ Wrong: no error return
func Bad(args map[string]any) any { ... }

// ❌ Wrong: typed map
func Bad(args map[string]string) (any, error) { ... }

// ❌ Wrong: extra parameters
func Bad(ctx context.Context, args map[string]any) (any, error) { ... }
```

### "plugin was built with a different version of package X"

The plugin and host were compiled with different Go versions or dependency versions. Fix:

```bash
# Ensure same Go version
go version  # must match between plugin and host builds

# Rebuild plugin
go build -buildmode=plugin -o plugin.so ./plugin/
```

### "plugin path 'X' not in allowed directories"

The plugin file is outside the configured `AllowedDirs`. Either:
- Move the plugin to an allowed directory
- Add the plugin's directory to `AllowedDirs`
- Remove `AllowedDirs` restriction (not recommended for production)

### "plugin declares 'X' in manifest but symbol not found"

`Manifest()` returns a function name that doesn't exist as an exported symbol. Ensure:
- The function name is capitalized (exported in Go)
- The function is in `package main`
- The function name in `Manifest()` matches exactly (case-sensitive)
