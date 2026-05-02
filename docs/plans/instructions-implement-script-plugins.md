# Implementation Instruction: Phase 4.5j — Script Plugins + Runtime Extraction

**Target:** `packages/go-json/runtime/`, `packages/go-json/lang/`, `packages/go-json-runtimes/` (NEW), `engine/internal/runtime/embedded/`
**Reference:** `docs/plans/2026-07-15-phase-4.5j-script-plugins-runtime-extraction.md`
**Effort:** 3-4 weeks
**Prerequisites:** Phase 4.5i done, Phase 2/3/4/5 done (all runtimes exist in BitCode)

---

## Context

This phase has two major deliverables:
1. **go-json core**: Add `ScriptRuntime` interface + `script:` import type
2. **go-json-runtimes**: Extract all runtime engines from BitCode into standalone package

The work is ordered to minimize risk: core interface first, then new package, then extraction, then BitCode integration.

---

## Existing Code to Understand

Before implementing, read these files:

| File | Purpose |
|------|---------|
| `packages/go-json/runtime/runtime.go` | Runtime struct, options, Execute() — you'll add fields + options here |
| `packages/go-json/runtime/extension.go` | Extension pattern — script: follows same pattern |
| `packages/go-json/lang/parser.go` | Import path classification — add `script:` type |
| `packages/go-json/lang/vm.go` | VM execution — add script call dispatch |
| `engine/internal/runtime/embedded/runtime.go` | Current VM interface (to be decoupled) |
| `engine/internal/runtime/embedded/executor.go` | Shared executor (to be extracted) |
| `engine/internal/runtime/embedded/registry.go` | Engine registry (to be extracted) |
| `engine/internal/runtime/embedded/script_runner.go` | Script runner (to be extracted) |
| `engine/internal/runtime/embedded/bridge_helper.go` | Bridge helpers (STAYS in BitCode) |
| `engine/internal/runtime/embedded/goja/` | goja implementation (to be extracted) |
| `engine/internal/runtime/embedded/qjs/` | quickjs implementation (to be extracted) |
| `engine/internal/runtime/embedded/yaegi/` | yaegi implementation (to be extracted) |
| `engine/pkg/plugin/` | Node.js + Python managers (to be extracted) |

---

## Task 1: ScriptRuntime Interface (go-json core)

### What to Build

Create `packages/go-json/runtime/script_runtime.go`:

```go
package runtime

import "context"

type ScriptRuntime interface {
    Name() string
    Extensions() []string
    CanHandle(extension string) bool
    Execute(ctx context.Context, script string, function string, params map[string]any, bridge map[string]any) (any, error)
    Validate() error
    Close() error
}

type ScriptRuntimeRegistry struct {
    runtimes []ScriptRuntime
}

func NewScriptRuntimeRegistry() *ScriptRuntimeRegistry { ... }
func (r *ScriptRuntimeRegistry) Register(rt ScriptRuntime) { ... }
func (r *ScriptRuntimeRegistry) Resolve(extension string) ScriptRuntime { ... }
func (r *ScriptRuntimeRegistry) ResolveByName(name string) ScriptRuntime { ... }
func (r *ScriptRuntimeRegistry) All() []ScriptRuntime { ... }
func (r *ScriptRuntimeRegistry) Close() error { ... }
```

### What to Modify

In `packages/go-json/runtime/runtime.go`:

1. Add fields to `Runtime` struct:
   ```go
   scriptRuntimes *ScriptRuntimeRegistry
   scriptBridge   map[string]any
   ```

2. Initialize in `NewRuntime()`:
   ```go
   scriptRuntimes: NewScriptRuntimeRegistry(),
   ```

3. Add options:
   ```go
   func WithScriptRuntime(rt ScriptRuntime) Option { ... }
   func WithScriptBridge(bridge map[string]any) Option { ... }
   ```

4. In `Runtime.Close()`, also close script runtimes:
   ```go
   if err := r.scriptRuntimes.Close(); err != nil && firstErr == nil {
       firstErr = err
   }
   ```

### Tests

Write in `packages/go-json/runtime/script_runtime_test.go`:
- `TestScriptRuntimeRegistry_Register`
- `TestScriptRuntimeRegistry_Resolve`
- `TestScriptRuntimeRegistry_ResolveByName`
- `TestScriptRuntimeRegistry_Close`
- `TestWithScriptRuntime`
- `TestWithScriptBridge`

Use a mock ScriptRuntime for testing.

### Verification

```bash
cd packages/go-json
go build ./...
go vet ./...
go test ./runtime/ -v
```

---

## Task 2: Parser — `script:` Import Type

### What to Modify

In `packages/go-json/lang/parser.go`, find the import path classification logic and add:

```go
case strings.HasPrefix(path, "script:"):
    return "script", strings.TrimPrefix(path, "script:")
```

The exact location depends on how `classifyImportPath` or equivalent is structured. Search for where `"io:"` and `"ext:"` are handled.

### AST Storage

Script imports should be stored in `AST.RequestedModules` with `PathType: "script"`. The path stored should be the resolved path (without `script:` prefix).

### Path Validation

Add path traversal prevention:
```go
func validateScriptImportPath(basePath, scriptPath string) error {
    if filepath.IsAbs(scriptPath) {
        return fmt.Errorf("script: import path must be relative, got: %s", scriptPath)
    }
    abs := filepath.Join(basePath, scriptPath)
    abs = filepath.Clean(abs)
    if !strings.HasPrefix(abs, filepath.Clean(basePath)) {
        return fmt.Errorf("script: import path escapes base directory: %s", scriptPath)
    }
    return nil
}
```

### Tests

Write in `packages/go-json/lang/` (or appropriate test file):
- `TestScriptImportParsing` — `"ml": "script:./plugins/predict.py"` parses correctly
- `TestScriptImportPathType` — PathType is "script"
- `TestScriptImportPathResolution` — path is relative to program dir
- `TestScriptImportAbsolutePathRejected` — absolute paths rejected
- `TestScriptImportPathTraversal` — `"../../../etc/passwd"` rejected

### Verification

```bash
cd packages/go-json
go test ./lang/ -run TestScript -v
```

---

## Task 3: VM — Script Call Dispatch

### Step-Level Calls

When the VM encounters `{"call": "ml.predict", "with": {...}}`:

1. Split on first `.` → namespace = "ml", function = "predict"
2. Check if "ml" is a script import (check `RequestedModules`)
3. If yes → resolve ScriptRuntime, call `Execute(ctx, scriptPath, "predict", params, bridge)`
4. If no → fall through to existing function resolution

### Expression-Level Calls

When `Runtime.Execute()` processes script imports, inject a proxy into the input scope:

```go
case "script":
    scriptPath := imp.Path
    if !filepath.IsAbs(scriptPath) {
        scriptPath = filepath.Join(programDir, scriptPath)
    }
    ext := filepath.Ext(scriptPath)
    rt := r.scriptRuntimes.Resolve(ext)
    if rt == nil {
        return nil, fmt.Errorf("no script runtime for extension '%s' (import '%s')", ext, alias)
    }
    if err := rt.Validate(); err != nil {
        return nil, fmt.Errorf("script runtime '%s' not available: %w", rt.Name(), err)
    }
    // Inject proxy that allows expression-level calls
    input[alias] = r.buildScriptProxy(rt, scriptPath)
```

The proxy should be a `map[string]any` where accessing any key returns a callable function. Since we can't enumerate functions upfront, use a special approach:

**Option A (Simplest — recommended for Phase 4.5j):**
Inject a single `call` function:
```go
input[alias] = map[string]any{
    "call": func(function string, args ...any) (any, error) {
        params := map[string]any{"args": args}
        return rt.Execute(r.ctx, scriptPath, function, params, r.scriptBridge)
    },
}
```
Usage: `ml.call("predict", features)` — works immediately with expr-lang.

**Option B (Better UX — if expr-lang supports it):**
Use a custom type with method dispatch. Test if expr-lang can call methods on a struct:
```go
type scriptProxy struct { ... }
func (s *scriptProxy) Predict(args ...any) (any, error) { ... }
```
This requires knowing function names upfront (manifest).

**Option C (Best UX — requires VM modification):**
Intercept `ml.predict(...)` in the VM before expr-lang evaluation. Parse the expression, detect script namespace access, replace with direct call.

**Recommendation:** Start with Option A. It works immediately. Add Option C as enhancement later.

### Tests

- `TestScriptCallStep` — `{"call": "ml.predict", "with": {...}}` dispatches correctly
- `TestScriptCallExpression` — `ml.call("predict", data)` works in expression
- `TestScriptRuntimeNotRegistered` — clear error when no runtime for .py
- `TestScriptRuntimeNotAvailable` — clear error when Validate() fails
- `TestScriptCallWithBridge` — bridge map is passed to runtime

### Verification

```bash
cd packages/go-json
go test ./... -v
# All 969+ existing tests must still pass
```

---

## Task 4: Create `packages/go-json-runtimes/` Package

### Directory Structure

```bash
mkdir -p packages/go-json-runtimes
mkdir -p packages/go-json-runtimes/pool
mkdir -p packages/go-json-runtimes/goja
mkdir -p packages/go-json-runtimes/quickjs
mkdir -p packages/go-json-runtimes/yaegi
mkdir -p packages/go-json-runtimes/node
mkdir -p packages/go-json-runtimes/python
mkdir -p packages/go-json-runtimes/testdata/scripts
```

### go.mod

```bash
cd packages/go-json-runtimes
go mod init github.com/bitcode-framework/go-json-runtimes
```

Add dependencies from BitCode's current go.mod (goja, yaegi, etc.).

### Core Files to Write

1. **runtime.go** — VM, EmbeddedRuntime, ExternalRuntime interfaces (decoupled from bridge.Context)
2. **executor.go** — Copy from `engine/internal/runtime/embedded/executor.go`, change `*bridge.Context` → `map[string]any`
3. **registry.go** — Copy from `engine/internal/runtime/embedded/registry.go` (no changes needed)
4. **loader.go** — Copy from `engine/internal/runtime/embedded/script_loader.go` (no changes)
5. **helpers.go** — Copy ONLY generic helpers from `bridge_helper.go` (ToInt, ToStringSlice, toAnySliceSlice). Do NOT copy Parse*Opts functions (they depend on bridge types).
6. **config.go** — Configuration types (GojaConfig, NodeConfig, PythonConfig, etc.)
7. **options.go** — Functional options (WithGoja, WithNode, etc.)
8. **runner.go** — ScriptRuntime implementation that aggregates all engines

### Key Interface Change

The critical change is `InjectBridge`:

```go
// BEFORE (in BitCode):
type VM interface {
    InjectBridge(bc *bridge.Context) error  // ← typed, coupled
    // ...
}

// AFTER (in go-json-runtimes):
type VM interface {
    InjectBridge(bridge map[string]any) error  // ← generic, decoupled
    // ...
}
```

### Verification

```bash
cd packages/go-json-runtimes
go build ./...
go vet ./...
```

---

## Task 5: Extract Embedded Runtimes (goja, quickjs, yaegi)

### For Each Runtime (goja, quickjs, yaegi):

1. Copy the entire directory from `engine/internal/runtime/embedded/{runtime}/` to `packages/go-json-runtimes/{runtime}/`
2. Change package declaration
3. Change `InjectBridge(*bridge.Context)` → `InjectBridge(map[string]any)`
4. Remove imports of `github.com/bitcode-framework/bitcode/internal/runtime/bridge`
5. Update bridge injection code:

**goja example (before):**
```go
func (v *GojaVM) InjectBridge(bc *bridge.Context) error {
    v.rt.Set("bitcode", v.buildBitcodeObject(bc))
    return nil
}
```

**goja example (after):**
```go
func (v *GojaVM) InjectBridge(bridge map[string]any) error {
    v.rt.Set("bitcode", v.convertBridgeToGoja(bridge))
    return nil
}

func (v *GojaVM) convertBridgeToGoja(bridge map[string]any) any {
    // Convert map[string]any to goja-compatible object
    // Functions in the map are already Go functions — goja can call them directly
    return bridge
}
```

### CRITICAL: goja's `buildBitcodeObject`

The current goja implementation has `buildBitcodeObject(bc *bridge.Context)` which manually constructs the bridge object from `*bridge.Context`. After extraction, this is no longer needed because the bridge is ALREADY a `map[string]any` built by BitCode's `BuildGoJSONExtension()`.

The extracted goja just needs to inject the map directly:
```go
func (v *GojaVM) InjectBridge(bridge map[string]any) error {
    // goja can handle map[string]any directly — it converts to JS object
    v.rt.Set("bitcode", bridge)
    return nil
}
```

### yaegi Special Case: Custom Bridges

yaegi has a bridges loader that loads `.go` files from a directory. This must be preserved:

```go
// packages/go-json-runtimes/yaegi/bridges_loader.go
func (y *YaegiVM) LoadBridges(dir string) error {
    // Same logic as before — load .go files from dir, interpret with yaegi
    // ...
}
```

The `BridgesDir` config is passed via `YaegiConfig`.

### Tests

Copy tests from BitCode, update imports. All tests must pass with the new `map[string]any` interface.

### Verification

```bash
cd packages/go-json-runtimes
go test ./goja/ -v
go test ./quickjs/ -v
go test ./yaegi/ -v
```

---

## Task 6: Extract External Runtimes (Node.js, Python)

### Node.js

Extract from `engine/pkg/plugin/` (the Node.js-specific parts):
- Process management (spawn, kill, recycle)
- JSON-RPC protocol (bidirectional communication)
- Bridge proxy (host function calls from Node.js)
- esbuild TypeScript transpilation
- Version validation

Key change: The Node.js runtime currently receives `*bridge.Context` and builds its own bridge proxy. After extraction, it receives `map[string]any` and proxies function calls from that map.

### Python

Same pattern as Node.js:
- Process management
- JSON-RPC protocol
- Bridge proxy
- Version validation
- venv management

### Pool Integration

Both Node.js and Python use the pool manager. Extract pool code to `packages/go-json-runtimes/pool/` and have both runtimes use it.

### Verification

```bash
cd packages/go-json-runtimes
go test ./node/ -v      # requires Node.js installed
go test ./python/ -v    # requires Python installed
go test ./pool/ -v
```

---

## Task 7: BitCode Integration (Thin Wrapper)

### What to Build

Replace BitCode's `engine/internal/runtime/embedded/` with a thin wrapper that:
1. Imports `go-json-runtimes`
2. Adapts `*bridge.Context` → `map[string]any` using existing `BuildGoJSONExtension()`
3. Delegates all VM operations to go-json-runtimes

### VMAdapter Pattern

```go
// engine/internal/runtime/embedded/adapter.go
package embedded

import (
    runtimes "github.com/bitcode-framework/go-json-runtimes"
    "github.com/bitcode-framework/bitcode/internal/runtime/bridge"
)

type VMAdapter struct {
    vm runtimes.VM
}

func NewVMAdapter(vm runtimes.VM) *VMAdapter {
    return &VMAdapter{vm: vm}
}

func (a *VMAdapter) InjectBridge(bc *bridge.Context) error {
    bridgeMap := bridge.BuildGoJSONExtension(bc).Functions
    return a.vm.InjectBridge(bridgeMap)
}

func (a *VMAdapter) InjectParams(params map[string]any) error {
    return a.vm.InjectParams(params)
}

func (a *VMAdapter) Execute(code string, filename string) (any, error) {
    return a.vm.Execute(code, filename)
}

func (a *VMAdapter) Interrupt(reason string) {
    a.vm.Interrupt(reason)
}

func (a *VMAdapter) Close() {
    a.vm.Close()
}
```

### Keep These Files in BitCode

- `bridge_helper.go` — depends on bridge types (SearchOptions, HTTPOptions, etc.)
- `gojson_adapter.go` — builds bridge map from *bridge.Context
- `gojson_adapter_test.go` — tests bridge building
- All files in `engine/internal/runtime/bridge/` — unchanged

### Update go.mod

```bash
cd engine
go get github.com/bitcode-framework/go-json-runtimes@latest
```

### Verification (CRITICAL)

```bash
cd engine
go build ./...
go vet ./...
go test ./... -v
# ALL existing tests must pass — this is a refactoring, not a rewrite
```

Pay special attention to:
- Node.js integration tests (78 tests)
- Python integration tests (11 tests)
- yaegi tests (18 tests)
- goja/quickjs tests
- Process execution tests
- Bridge tests

---

## Task 8: Documentation

### Files to Create

1. `packages/go-json-runtimes/README.md` — package overview, installation, usage examples
2. `packages/go-json-runtimes/AGENTS.md` — package structure, conventions, testing

### Files to Update

1. `packages/go-json/AGENTS.md` — add ScriptRuntime interface, script: import type
2. `packages/go-json/features.md` — add script plugin support
3. `packages/go-json/docs/language-reference.md` — add script: import documentation
4. `packages/go-json/docs/embedding-guide.md` — add WithScriptRuntime() option
5. `engine/AGENTS.md` — note that engine uses go-json-runtimes
6. `engine/docs/architecture.md` — update runtime layer diagram
7. `engine/docs/codebase.md` — update file map (remove embedded/, add import)
8. `engine/docs/features/plugins.md` — reference go-json-runtimes

---

## Order of Operations (Risk-Minimized)

1. **Task 1** — ScriptRuntime interface (additive, no breaking changes)
2. **Task 2** — Parser changes (additive, no breaking changes)
3. **Task 3** — VM changes (additive, existing tests must pass)
4. **Task 4** — Create go-json-runtimes package (new code, no impact on existing)
5. **Task 5** — Extract embedded runtimes (copy + modify, test in isolation)
6. **Task 6** — Extract external runtimes (copy + modify, test in isolation)
7. **Task 7** — BitCode integration (the risky part — do last, verify extensively)
8. **Task 8** — Documentation (after everything works)

---

## Common Pitfalls

### 1. Don't Remove bridge_helper.go from BitCode

`bridge_helper.go` has `ParseSearchOpts`, `ParseHTTPOpts`, etc. that depend on `bridge.SearchOptions`, `bridge.HTTPOptions`. These are BitCode-specific types. Only copy the GENERIC helpers (ToInt, ToStringSlice) to go-json-runtimes.

### 2. Don't Break the goja `tx` Callback

goja's `proxy.go` has callback-based `tx` that uses `goja.Callable`. This MUST be preserved in the extracted goja runtime. The callback-based tx works because goja CAN pass Go callbacks. Only go-json's bridge uses imperative tx (Phase 4.5h).

### 3. Don't Forget Console Interception

goja has console.log interception that routes to the bridge logger. This must be preserved in the extracted code.

### 4. Pool Manager is Optional

The pool manager must work as optional. Standalone users (without BitCode) should be able to use runtimes without pool:
```go
// No pool — spawn per call:
rt, _ := runtimes.New(runtimes.WithPython())

// With pool:
rt, _ := runtimes.New(runtimes.WithPython(runtimes.PythonConfig{Pool: &poolConfig}))
```

### 5. go-json Core Must Remain Zero-Dependency

After all changes, verify:
```bash
cd packages/go-json
go list -m all | wc -l
# Should be same as before (only expr-lang/expr as external dep)
```

The `ScriptRuntime` interface is defined in go-json core but NEVER imported from go-json-runtimes. The dependency flows one way: `go-json-runtimes → go-json` (not the reverse).

### 6. Test with Mock ScriptRuntime

For go-json core tests, use a mock ScriptRuntime:
```go
type mockScriptRuntime struct {
    name       string
    extensions []string
    execFn     func(ctx context.Context, script, function string, params, bridge map[string]any) (any, error)
}
```

This avoids go-json core tests depending on actual Python/Node.js installations.

---

## Final Verification Checklist

```bash
# 1. go-json core
cd packages/go-json
go build ./...
go vet ./...
go test ./... -v
# Expected: 969+ tests pass (all existing + new script tests)

# 2. go-json-runtimes
cd packages/go-json-runtimes
go build ./...
go vet ./...
go test ./... -v
# Expected: ~170+ tests pass (extracted from BitCode)

# 3. BitCode engine
cd engine
go build ./...
go vet ./...
go test ./... -v
# Expected: ALL existing tests pass (this is a refactoring)

# 4. Zero-dependency check
cd packages/go-json
grep -c "require" go.mod
# Should NOT have go-json-runtimes as dependency
```
