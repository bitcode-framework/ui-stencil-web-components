# Implementation Instruction: Phase 4.5k — WASM + Native Plugins

**Target:** `packages/go-json/runtime/`, `packages/go-json/lang/`, `packages/go-json-runtimes/wasm/`, `packages/go-json-runtimes/native/`
**Reference:** `docs/plans/2026-07-15-phase-4.5k-wasm-native-plugins.md`
**Effort:** 2-3 weeks
**Prerequisites:** Phase 4.5j done (ScriptRuntime interface exists, go-json-runtimes package exists)

---

## Context

Phase 4.5j established the `ScriptRuntime` interface and `script:` import type. This phase adds two new runtime implementations:

1. **WASM** — WebAssembly modules via wazero (pure Go, zero CGO)
2. **Native** — Go shared libraries via `plugin` package (Linux/macOS only)

Both implement the same `ScriptRuntime` interface from Phase 4.5j. The go-json core changes are minimal (just adding `wasm:` and `plugin:` to the parser).

---

## Existing Code to Understand

| File | Purpose |
|------|---------|
| `packages/go-json/runtime/script_runtime.go` | ScriptRuntime interface (from Phase 4.5j) |
| `packages/go-json/runtime/runtime.go` | WithScriptRuntime option (from Phase 4.5j) |
| `packages/go-json/lang/parser.go` | Import classification (add wasm:/plugin:) |
| `packages/go-json-runtimes/runtime.go` | VM interface (from Phase 4.5j) |
| `packages/go-json-runtimes/options.go` | Functional options (add WithWasm/WithNative) |
| `packages/go-json-runtimes/runner.go` | ScriptRuntime aggregator (register new runtimes) |

---

## Task 1: go-json Core — Parser Changes

### What to Modify

In `packages/go-json/lang/parser.go`, add two new import path types:

```go
case strings.HasPrefix(path, "wasm:"):
    return "wasm", strings.TrimPrefix(path, "wasm:")
case strings.HasPrefix(path, "plugin:"):
    return "plugin", strings.TrimPrefix(path, "plugin:")
```

### What to Modify in Runtime

In `packages/go-json/runtime/runtime.go`, in the `Execute()` method where imports are resolved, add handling for `"wasm"` and `"plugin"` path types. These follow the exact same pattern as `"script"`:

```go
case "wasm":
    wasmPath := imp.Path
    if !filepath.IsAbs(wasmPath) {
        wasmPath = filepath.Join(programDir, wasmPath)
    }
    rt := r.scriptRuntimes.Resolve(".wasm")
    if rt == nil {
        return nil, fmt.Errorf("no WASM runtime registered (import '%s')", alias)
    }
    input[alias] = r.buildScriptProxy(rt, wasmPath)

case "plugin":
    pluginPath := imp.Path
    if !filepath.IsAbs(pluginPath) {
        pluginPath = filepath.Join(programDir, pluginPath)
    }
    ext := filepath.Ext(pluginPath)
    rt := r.scriptRuntimes.Resolve(ext)
    if rt == nil {
        return nil, fmt.Errorf("no native plugin runtime for '%s' (import '%s')", ext, alias)
    }
    if err := rt.Validate(); err != nil {
        return nil, fmt.Errorf("native plugin runtime not available: %w", err)
    }
    input[alias] = r.buildScriptProxy(rt, pluginPath)
```

### Tests

- `TestWasmImportParsing` — `"img": "wasm:./plugins/imgproc.wasm"` parses correctly
- `TestPluginImportParsing` — `"fast": "plugin:./plugins/fast.so"` parses correctly
- `TestWasmImportPathType` — PathType is "wasm"
- `TestPluginImportPathType` — PathType is "plugin"

### Verification

```bash
cd packages/go-json
go build ./...
go test ./... -v
# All existing tests must pass + new parser tests
```

---

## Task 2: WASM Runtime Implementation

### Setup

```bash
cd packages/go-json-runtimes
go get github.com/tetratelabs/wazero@latest
mkdir -p wasm
```

### File: `wasm/runtime.go`

```go
package wasm

import (
    "context"
    "encoding/json"
    "fmt"
    "os"
    "path/filepath"
    "sync"
    "time"

    "github.com/tetratelabs/wazero"
    "github.com/tetratelabs/wazero/api"
    "github.com/tetratelabs/wazero/imports/wasi_snapshot_preview1"
)

type WasmRuntime struct {
    engine wazero.Runtime
    cache  map[string]wazero.CompiledModule
    config Config
    mu     sync.RWMutex
}

type Config struct {
    MaxMemoryMB  int           // per-instance memory limit (default: 64)
    MaxExecTime  time.Duration // per-call timeout (default: 30s)
    CompileCache bool          // cache compiled modules (default: true)
    WASIEnabled  bool          // enable WASI (default: false)
}

func DefaultConfig() Config {
    return Config{
        MaxMemoryMB:  64,
        MaxExecTime:  30 * time.Second,
        CompileCache: true,
        WASIEnabled:  false,
    }
}
```

### Key Implementation Details

**Module Compilation (cached):**
```go
func (w *WasmRuntime) getOrCompile(ctx context.Context, path string) (wazero.CompiledModule, error) {
    absPath, _ := filepath.Abs(path)
    
    w.mu.RLock()
    if compiled, ok := w.cache[absPath]; ok {
        w.mu.RUnlock()
        return compiled, nil
    }
    w.mu.RUnlock()
    
    wasmBytes, err := os.ReadFile(path)
    if err != nil {
        return nil, fmt.Errorf("failed to read WASM file: %w", err)
    }
    
    compiled, err := w.engine.CompileModule(ctx, wasmBytes)
    if err != nil {
        return nil, fmt.Errorf("failed to compile WASM module: %w", err)
    }
    
    if w.config.CompileCache {
        w.mu.Lock()
        w.cache[absPath] = compiled
        w.mu.Unlock()
    }
    
    return compiled, nil
}
```

**Host Functions (bridge_call):**
```go
func (w *WasmRuntime) defineHostModule(ctx context.Context, bridge map[string]any) (api.Module, error) {
    builder := w.engine.NewHostModuleBuilder("env")
    
    builder.NewFunctionBuilder().
        WithFunc(func(ctx context.Context, m api.Module, fnNamePtr, fnNameLen, argsPtr, argsLen uint32) uint64 {
            // Read function name
            fnName := readStringFromMemory(m, fnNamePtr, fnNameLen)
            // Read args
            argsJSON := readBytesFromMemory(m, argsPtr, argsLen)
            
            // Resolve bridge function (supports dotted paths: "db.query")
            result, err := invokeBridgeFunction(bridge, fnName, argsJSON)
            
            // Write result back
            return writeResultToMemory(m, result, err)
        }).
        WithParameterNames("fn_ptr", "fn_len", "args_ptr", "args_len").
        Export("bridge_call")
    
    return builder.Instantiate(ctx)
}
```

**Memory Protocol:**
```go
func readStringFromMemory(m api.Module, ptr, length uint32) string {
    bytes, ok := m.Memory().Read(ptr, length)
    if !ok {
        return ""
    }
    return string(bytes)
}

func readBytesFromMemory(m api.Module, ptr, length uint32) []byte {
    bytes, ok := m.Memory().Read(ptr, length)
    if !ok {
        return nil
    }
    result := make([]byte, length)
    copy(result, bytes)
    return result
}

func writeResultToMemory(m api.Module, result any, err error) uint64 {
    var response []byte
    if err != nil {
        response, _ = json.Marshal(map[string]any{"error": err.Error()})
    } else {
        response, _ = json.Marshal(map[string]any{"value": result})
    }
    
    // Call WASM's malloc to allocate space
    malloc := m.ExportedFunction("malloc")
    if malloc == nil {
        return 0
    }
    results, _ := malloc.Call(context.Background(), uint64(len(response)))
    if len(results) == 0 {
        return 0
    }
    ptr := uint32(results[0])
    
    // Write response
    m.Memory().Write(ptr, response)
    
    // Pack ptr and len into u64
    return (uint64(ptr) << 32) | uint64(len(response))
}
```

**Bridge Function Invocation:**
```go
func invokeBridgeFunction(bridge map[string]any, fnPath string, argsJSON []byte) (any, error) {
    // Parse args
    var args []any
    if len(argsJSON) > 0 {
        json.Unmarshal(argsJSON, &args)
    }
    
    // Resolve dotted path: "db.query" → bridge["db"].(map[string]any)["query"]
    parts := strings.Split(fnPath, ".")
    var current any = bridge
    for i, part := range parts {
        m, ok := current.(map[string]any)
        if !ok {
            return nil, fmt.Errorf("bridge path '%s' not found", fnPath)
        }
        current = m[part]
        if current == nil {
            return nil, fmt.Errorf("bridge function '%s' not found", fnPath)
        }
        if i < len(parts)-1 {
            continue
        }
        // Last part — should be a function
        // Try various function signatures
        switch fn := current.(type) {
        case func(string, ...any) (any, error):
            if len(args) > 0 {
                str, _ := args[0].(string)
                return fn(str, args[1:]...)
            }
            return fn("")
        case func(...any) (any, error):
            return fn(args...)
        case func(map[string]any) (any, error):
            if len(args) > 0 {
                if m, ok := args[0].(map[string]any); ok {
                    return fn(m)
                }
            }
            return fn(nil)
        default:
            return nil, fmt.Errorf("bridge '%s' is not callable", fnPath)
        }
    }
    return nil, fmt.Errorf("bridge function '%s' not found", fnPath)
}
```

### ScriptRuntime Interface Implementation

```go
func (w *WasmRuntime) Name() string              { return "wasm" }
func (w *WasmRuntime) Extensions() []string       { return []string{".wasm"} }
func (w *WasmRuntime) CanHandle(ext string) bool  { return ext == ".wasm" }
func (w *WasmRuntime) Validate() error            { return nil } // wazero is embedded

func (w *WasmRuntime) Execute(ctx context.Context, script, function string, params, bridge map[string]any) (any, error) {
    // 1. Compile (cached)
    compiled, err := w.getOrCompile(ctx, script)
    if err != nil {
        return nil, err
    }
    
    // 2. Define host functions
    _, err = w.defineHostModule(ctx, bridge)
    if err != nil {
        return nil, err
    }
    
    // 3. Optionally instantiate WASI
    if w.config.WASIEnabled {
        wasi_snapshot_preview1.MustInstantiate(ctx, w.engine)
    }
    
    // 4. Instantiate module
    moduleConfig := wazero.NewModuleConfig().
        WithName("").  // anonymous (allows multiple instances)
        WithStartFunctions() // don't auto-call _start
    
    instance, err := w.engine.InstantiateModule(ctx, compiled, moduleConfig)
    if err != nil {
        return nil, fmt.Errorf("wasm instantiate: %w", err)
    }
    defer instance.Close(ctx)
    
    // 5. Serialize params
    argsJSON, err := json.Marshal(params)
    if err != nil {
        return nil, fmt.Errorf("wasm args marshal: %w", err)
    }
    
    // 6. Allocate + write args to WASM memory
    malloc := instance.ExportedFunction("malloc")
    if malloc == nil {
        return nil, fmt.Errorf("wasm module missing 'malloc' export")
    }
    allocResult, err := malloc.Call(ctx, uint64(len(argsJSON)))
    if err != nil {
        return nil, fmt.Errorf("wasm malloc: %w", err)
    }
    argPtr := uint32(allocResult[0])
    if !instance.Memory().Write(argPtr, argsJSON) {
        return nil, fmt.Errorf("wasm memory write failed")
    }
    
    // 7. Call function with timeout
    fn := instance.ExportedFunction(function)
    if fn == nil {
        return nil, fmt.Errorf("wasm function '%s' not exported", function)
    }
    
    timeoutCtx, cancel := context.WithTimeout(ctx, w.config.MaxExecTime)
    defer cancel()
    
    results, err := fn.Call(timeoutCtx, uint64(argPtr), uint64(len(argsJSON)))
    if err != nil {
        return nil, fmt.Errorf("wasm call '%s': %w", function, err)
    }
    
    // 8. Read result
    if len(results) == 0 {
        return nil, nil
    }
    packed := results[0]
    resultPtr := uint32(packed >> 32)
    resultLen := uint32(packed & 0xFFFFFFFF)
    
    if resultLen == 0 {
        return nil, nil
    }
    
    resultJSON, ok := instance.Memory().Read(resultPtr, resultLen)
    if !ok {
        return nil, fmt.Errorf("wasm result read failed")
    }
    
    // 9. Free WASM memory
    if free := instance.ExportedFunction("free"); free != nil {
        free.Call(ctx, uint64(resultPtr), uint64(resultLen))
    }
    
    // 10. Deserialize
    var result any
    if err := json.Unmarshal(resultJSON, &result); err != nil {
        return nil, fmt.Errorf("wasm result unmarshal: %w", err)
    }
    
    return result, nil
}

func (w *WasmRuntime) Close() error {
    return w.engine.Close(context.Background())
}
```

### Tests

Create test WASM modules. The simplest approach is to pre-compile small Rust/TinyGo programs and commit the .wasm files to testdata/:

```
packages/go-json-runtimes/testdata/wasm/
├── hello.wasm       ← greet(name) → "Hello, {name}!"
├── math.wasm        ← add(a, b) → a + b
├── bridge.wasm      ← calls bridge_call("log", ...)
└── build/
    ├── hello/src/lib.rs
    ├── math/src/lib.rs
    └── Makefile
```

For CI without Rust, commit pre-built .wasm files.

**Alternative for testing without pre-built WASM**: Use wazero's WAT (WebAssembly Text) format to create simple test modules inline:

```go
func TestWasmSimpleFunction(t *testing.T) {
    // WAT for a simple add function
    wat := `(module
        (memory (export "memory") 1)
        (func (export "malloc") (param i32) (result i32)
            i32.const 0)
        (func (export "free") (param i32 i32))
        (func (export "add") (param i32 i32) (result i64)
            ;; simplified — real impl reads JSON from memory
            i64.const 0)
    )`
    // Use wazero's WAT compilation for testing
}
```

### Verification

```bash
cd packages/go-json-runtimes
go test ./wasm/ -v
```

---

## Task 3: Native Plugin Runtime Implementation

### File: `native/runtime.go`

```go
package native

import (
    "context"
    "fmt"
    "path/filepath"
    "plugin"
    "runtime"
    "sync"
)

type NativeRuntime struct {
    plugins map[string]*loadedPlugin
    config  Config
    mu      sync.RWMutex
}

type Config struct {
    AllowedDirs []string // directories from which plugins can be loaded
}

type loadedPlugin struct {
    p         *plugin.Plugin
    functions map[string]func(map[string]any) (any, error)
    manifest  []string
}
```

### Platform Guard

```go
func NewNativeRuntime(config Config) (*NativeRuntime, error) {
    if runtime.GOOS == "windows" {
        return nil, fmt.Errorf("native plugins not supported on Windows (use wasm: instead)")
    }
    return &NativeRuntime{
        plugins: make(map[string]*loadedPlugin),
        config:  config,
    }, nil
}

func (n *NativeRuntime) Validate() error {
    if runtime.GOOS == "windows" {
        return fmt.Errorf("native plugins not supported on Windows")
    }
    return nil
}
```

### Plugin Loading

```go
func (n *NativeRuntime) getOrLoad(path string) (*loadedPlugin, error) {
    absPath, _ := filepath.Abs(path)
    
    n.mu.RLock()
    if loaded, ok := n.plugins[absPath]; ok {
        n.mu.RUnlock()
        return loaded, nil
    }
    n.mu.RUnlock()
    
    // Validate path
    if err := n.validatePath(absPath); err != nil {
        return nil, err
    }
    
    // Open plugin
    p, err := plugin.Open(absPath)
    if err != nil {
        return nil, fmt.Errorf("failed to open plugin '%s': %w", filepath.Base(path), err)
    }
    
    // Get manifest
    manifestSym, err := p.Lookup("Manifest")
    if err != nil {
        return nil, fmt.Errorf("plugin '%s' missing Manifest() export", filepath.Base(path))
    }
    manifestFn, ok := manifestSym.(func() []string)
    if !ok {
        return nil, fmt.Errorf("plugin '%s' Manifest has wrong signature (expected func() []string)", filepath.Base(path))
    }
    manifest := manifestFn()
    
    // Load functions
    functions := make(map[string]func(map[string]any) (any, error))
    for _, name := range manifest {
        sym, err := p.Lookup(name)
        if err != nil {
            return nil, fmt.Errorf("plugin '%s' declares '%s' in manifest but symbol not found", filepath.Base(path), name)
        }
        fn, ok := sym.(func(map[string]any) (any, error))
        if !ok {
            return nil, fmt.Errorf("plugin '%s' function '%s' has wrong signature (expected func(map[string]any) (any, error))", filepath.Base(path), name)
        }
        functions[name] = fn
    }
    
    loaded := &loadedPlugin{p: p, functions: functions, manifest: manifest}
    
    n.mu.Lock()
    n.plugins[absPath] = loaded
    n.mu.Unlock()
    
    return loaded, nil
}
```

### Build Tags

Native plugin tests should be build-tagged for Linux/macOS:

```go
//go:build linux || darwin

package native
```

### Tests

Create a test plugin:

```go
// testdata/native/hello_plugin.go
package main

func Greet(args map[string]any) (any, error) {
    name, _ := args["args"].([]any)
    if len(name) > 0 {
        return "Hello, " + fmt.Sprint(name[0]) + "!", nil
    }
    return "Hello, World!", nil
}

func Manifest() []string {
    return []string{"Greet"}
}
```

Build in Makefile:
```makefile
# testdata/native/Makefile
hello.so: hello_plugin.go
	go build -buildmode=plugin -o hello.so hello_plugin.go
```

### Verification

```bash
cd packages/go-json-runtimes
# Linux/macOS only:
cd testdata/native && make && cd ../..
go test ./native/ -v
```

---

## Task 4: Configuration Options

### Add to `packages/go-json-runtimes/options.go`:

```go
func WithWasm(opts ...wasm.Config) RuntimeOption {
    return func(rs *runtimeSet) {
        cfg := wasm.DefaultConfig()
        if len(opts) > 0 {
            cfg = opts[0]
        }
        rs.wasmConfig = &cfg
    }
}

func WithNative(opts ...native.Config) RuntimeOption {
    return func(rs *runtimeSet) {
        cfg := native.Config{}
        if len(opts) > 0 {
            cfg = opts[0]
        }
        rs.nativeConfig = &cfg
    }
}
```

### Register in RuntimeSet Builder

When building the runtime set, if WASM is configured, create and register the WasmRuntime. Same for native.

---

## Task 5: Codegen Support

### WASM Codegen (Go target)

In `packages/go-json/codegen/`, add WASM import handling:

```go
// When generating Go code for a program with wasm: imports:
func (g *GoGenerator) generateWasmImport(alias, path string) string {
    return fmt.Sprintf(`
// WASM plugin: %s
var %sModule wazero.CompiledModule

func init() {
    wasmBytes, err := os.ReadFile(%q)
    if err != nil {
        panic("wasm plugin not found: " + err.Error())
    }
    %sModule, err = wasmRuntime.CompileModule(context.Background(), wasmBytes)
    if err != nil {
        panic("wasm compile error: " + err.Error())
    }
}
`, alias, alias, path, alias)
}
```

### WASM Codegen (JavaScript target)

```javascript
// Generated for wasm: import
const ${alias}Bytes = await fetch('${path}').then(r => r.arrayBuffer());
const { instance: ${alias}Instance } = await WebAssembly.instantiate(${alias}Bytes, {
    env: { bridge_call: (fnPtr, fnLen, argsPtr, argsLen) => { /* ... */ } }
});
```

### Native Codegen (Go target only)

```go
// Generated for plugin: import (Go target only)
var ${alias}Plugin *plugin.Plugin

func init() {
    var err error
    ${alias}Plugin, err = plugin.Open(${path})
    if err != nil {
        panic("native plugin not available: " + err.Error())
    }
}
```

### Native Codegen (JS/Python targets)

Generate a stub with clear error:
```go
func (g *JSGenerator) generateNativeImport(alias, path string) string {
    return fmt.Sprintf(`
// Native plugin '%s' cannot be used in JavaScript target.
// Use wasm: import instead for cross-platform compatibility.
const %s = { call: () => { throw new Error("Native plugins not supported in JS target"); } };
`, alias, alias)
}
```

---

## Task 6: BitCode Integration

### Register WASM Runtime in Engine Startup

In BitCode's engine initialization (where runtimes are configured):

```go
import (
    runtimes "github.com/bitcode-framework/go-json-runtimes"
    "github.com/bitcode-framework/go-json-runtimes/wasm"
)

// In engine startup:
runtimeOpts := []runtimes.RuntimeOption{
    runtimes.WithGoja(),
    runtimes.WithQuickJS(),
    runtimes.WithYaegi(yaegiConfig),
    runtimes.WithNode(nodeConfig),
    runtimes.WithPython(pythonConfig),
}

// Add WASM if enabled in config
if cfg.Runtime.Wasm.Enabled {
    runtimeOpts = append(runtimeOpts, runtimes.WithWasm(wasm.Config{
        MaxMemoryMB: cfg.Runtime.Wasm.MaxMemoryMB,
        MaxExecTime: cfg.Runtime.Wasm.MaxExecTime,
    }))
}

// Add native if enabled in config (disabled by default)
if cfg.Runtime.Native.Enabled {
    runtimeOpts = append(runtimeOpts, runtimes.WithNative(native.Config{
        AllowedDirs: cfg.Runtime.Native.AllowedDirs,
    }))
}
```

### Add Config Parsing

In BitCode's config parsing (Viper), add:

```yaml
runtime:
  wasm:
    enabled: true
    max_memory_mb: 64
    max_exec_time: "30s"
  native:
    enabled: false
    allowed_dirs: ["modules/*/plugins/", "plugins/"]
```

---

## Order of Operations

1. **Task 1** — Parser changes (minimal, additive)
2. **Task 2** — WASM runtime implementation (bulk of work)
3. **Task 3** — Native plugin runtime (simpler, platform-specific)
4. **Task 4** — Configuration options
5. **Task 5** — Codegen support
6. **Task 6** — BitCode integration

---

## Common Pitfalls

### 1. wazero Module Naming

wazero requires unique module names per instantiation. Use empty string `""` for anonymous modules, or generate unique names:
```go
moduleConfig := wazero.NewModuleConfig().WithName("")
```

### 2. WASM Memory is Per-Instance

Each WASM instance has its own linear memory. Don't share memory references across instances.

### 3. Packed Return Value

The WASM function returns `i64` with ptr in high 32 bits and len in low 32 bits:
```go
packed := (uint64(ptr) << 32) | uint64(len)
// Unpack:
ptr := uint32(packed >> 32)
len := uint32(packed & 0xFFFFFFFF)
```

### 4. Native Plugins Can't Be Unloaded

Go's `plugin` package doesn't support unloading. Once loaded, a plugin stays in memory for the process lifetime. This is fine for production but means tests that load plugins can't clean up.

### 5. Native Plugin Build Mode

Plugins must be built with `-buildmode=plugin` and the SAME Go version as the host. Version mismatch = load failure. Document this clearly.

### 6. WASM malloc/free Convention

The WASM module MUST export `malloc` and `free`. Without these, the host cannot write data to WASM memory. If a module doesn't export them, return a clear error:
```
"wasm module missing required 'malloc' export — see docs/wasm-plugin-protocol.md"
```

### 7. Don't Block on WASM Execution

wazero supports context cancellation. Always use `context.WithTimeout`:
```go
timeoutCtx, cancel := context.WithTimeout(ctx, w.config.MaxExecTime)
defer cancel()
results, err := fn.Call(timeoutCtx, ...)
```

---

## Final Verification Checklist

```bash
# 1. go-json core (parser changes)
cd packages/go-json
go build ./...
go vet ./...
go test ./... -v
# All existing tests pass + new wasm/plugin parser tests

# 2. go-json-runtimes (WASM + native)
cd packages/go-json-runtimes
go build ./...
go vet ./...
go test ./wasm/ -v
go test ./native/ -v  # Linux/macOS only

# 3. BitCode engine
cd engine
go build ./...
go test ./... -v
# All existing tests pass

# 4. Codegen
cd packages/go-json
go test ./codegen/ -v
# WASM/native codegen tests pass
```
