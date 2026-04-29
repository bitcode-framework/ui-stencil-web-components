# Phase 4.5c Fixes, Extensions & New Data Modules — Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Fix all critical bugs, gaps, and improvements identified in Phase 4.5c gap analysis, plus add MongoDB and Redis modules, multi-driver SQL support, SQL injection protection, and connection pooling with proper lifecycle management.

**Architecture:** Three-phase approach: (1) Fix critical bugs in existing code, (2) Refactor SQL module with pooling/multi-driver/security, (3) Add MongoDB and Redis as new I/O modules. Each fix is isolated and testable. Import resolver gets `io:`/`ext:` support for two-layer security. Runtime gets `Close()` for resource cleanup.

**Tech Stack:** Go 1.24+, database/sql, go.mongodb.org/mongo-driver/v2, github.com/redis/go-redis/v9, github.com/lib/pq (postgres), github.com/go-sql-driver/mysql

**Design Doc:** `2026-07-14-runtime-engine-phase-4.5c-go-json-io-integration.md`
**Gap Analysis Source:** Session discussion 2026-04-29 (25 issues identified)
**Depends on:** Phase 4.5c initial implementation complete

---

## Critical Path

```
Batch 1: Critical Bug Fixes (BUG-1 through BUG-6)
  Task 1 (import resolver io:/ext:) ──► Task 2 (ExprOptions type safety)
  Task 3 (SQL pool + close) ──► Task 4 (Runtime.Close)
  Task 5 (GoJSONRunner fix)
  Task 6 (bridge nested namespaces)
  Task 7 (cmdCodegen wiring)

Batch 2: Significant Gap Fixes (GAP-7 through GAP-13)
  Task 8 (test runner import fix)
  Task 9 (migrate safe rename)
  Task 10 (extension alias via import)
  Task 11 (bridge tx removal)
  Task 12 (CLI stdin support)
  Task 13 (codegen missing node types)

Batch 3: SQL Refactor + Multi-Driver + Security
  Task 14 (SQL pooling per-DSN)
  Task 15 (DefaultDSN + driver detection)
  Task 16 (SQL injection protection — BlockedKeywords, MaxQueryLength)
  Task 17 (SQL transaction auto-rollback + per-execution state)

Batch 4: MongoDB Module
  Task 18 (MongoModule + security config)
  Task 19 (mongo.find, findOne, insert, update, delete, count, aggregate)
  Task 20 (MongoDB connection pooling + Close)

Batch 5: Redis Module
  Task 21 (RedisModule + security config)
  Task 22 (redis string/hash/list/set/pub commands)
  Task 23 (Redis auto JSON serialize + KeyPrefix isolation)

Batch 6: Moderate Fixes
  Task 24 (Windows path security)
  Task 25 (HTTP auth fixes — invalid type error, precedence warning)
  Task 26 (bridge silent type coercion → explicit errors)
  Task 27 (exec timeout naming cleanup)

Batch 7: Improvements
  Task 28 (regex LRU cache)
  Task 29 (HTTP client reuse)
  Task 30 (deepEqual fix in test runner)
  Task 31 (bridge convertToAny optimization)
  Task 32 (dead code removal in codegen)

Batch 8: Missing Tests
  Task 33 (io/http_test.go, io/fs_test.go, io/sql_test.go, io/exec_test.go)
  Task 34 (io/mongo_test.go, io/redis_test.go)
  Task 35 (bridge integration tests)
  Task 36 (script detection + backward compat tests)
  Task 37 (CLI tests + test runner tests)
  Task 38 (expanded security, regex, extension, codegen tests)

Batch 9: Documentation
  Task 39 (update AGENTS.md, design doc, SecurityConfig docs)
```

---

## Batch 1: Critical Bug Fixes

### Task 1: Import Resolver — Handle `io:` and `ext:` Path Types

**Files:**
- Modify: `packages/go-json/lang/import.go`
- Modify: `packages/go-json/runtime/runtime.go`
- Test: `packages/go-json/lang/import_test.go`

**Context:** Currently `ResolveImports()` skips all non-relative imports (line 32). This means `io:` and `ext:` imports are parsed but never resolved, breaking the two-layer security model. Programs can call I/O functions without importing them because runtime injects them as top-level env vars unconditionally.

**Step 1:** Modify `lang/import.go` — `ResolveImports()` should collect `io:` and `ext:` imports into the Program's metadata instead of skipping them:

```go
func (ir *ImportResolver) ResolveImports(program *Program, basePath string, importStack []string) error {
    if len(program.Imports) == 0 {
        return nil
    }

    for _, imp := range program.Imports {
        switch imp.PathType {
        case "relative":
            // existing relative resolution logic (unchanged)
            resolvedPath := ir.resolvePath(imp.Path, basePath)
            if ir.isInStack(resolvedPath, importStack) {
                chain := append(importStack, resolvedPath)
                return CompileError("CIRCULAR_IMPORT",
                    fmt.Sprintf("circular import detected: %s", strings.Join(chain, " → ")), -1)
            }
            imported, err := ir.loadFile(resolvedPath, append(importStack, resolvedPath))
            if err != nil {
                return CompileError("IMPORT_ERROR",
                    fmt.Sprintf("error importing '%s' (alias '%s'): %s", imp.Path, imp.Alias, err.Error()), -1)
            }
            ir.mergeExports(program, imported, imp.Alias)

        case "io", "ext":
            // Record for runtime validation — actual binding happens in Runtime.Execute()
            if program.RequestedModules == nil {
                program.RequestedModules = make(map[string]ImportDef)
            }
            program.RequestedModules[imp.Alias] = *imp

        case "stdlib":
            // stdlib is always available, no resolution needed
            continue
        }
    }

    return nil
}
```

**Step 2:** Add `RequestedModules` field to `Program` in `lang/ast.go`:

```go
type Program struct {
    Name       string
    Imports    []*ImportDef
    Functions  map[string]*FuncDef
    Structs    map[string]*StructDef
    Steps      []Node
    RawJSON    json.RawMessage

    RequestedModules map[string]ImportDef // io: and ext: imports for runtime validation
}
```

**Step 3:** Modify `runtime/runtime.go` — `Execute()` should validate that requested modules are available and create aliases:

```go
func (r *Runtime) Execute(program *lang.CompiledProgram, input map[string]any) (*lang.ExecutionResult, error) {
    // ... existing setup ...

    // Validate and bind io:/ext: imports
    if program.AST != nil && program.AST.RequestedModules != nil {
        for alias, imp := range program.AST.RequestedModules {
            switch imp.PathType {
            case "io":
                moduleName := strings.TrimPrefix(imp.Path, "io:")
                if r.ioDisabled {
                    return nil, fmt.Errorf("I/O module '%s' requested but I/O is disabled", moduleName)
                }
                mod := r.ioRegistry.GetModule(moduleName)
                if mod == nil {
                    return nil, fmt.Errorf("I/O module '%s' not registered (imported as '%s')", moduleName, alias)
                }
                if input == nil {
                    input = make(map[string]any)
                }
                input[alias] = mod.Functions()

            case "ext":
                extName := strings.TrimPrefix(imp.Path, "ext:")
                ext := r.extensions.get(extName)
                if ext == nil {
                    return nil, fmt.Errorf("extension '%s' not registered (imported as '%s')", extName, alias)
                }
                if input == nil {
                    input = make(map[string]any)
                }
                input[alias] = ext.Functions
            }
        }
    }

    // Remove unconditional injection of I/O and extension env vars
    // Only inject stdlib env vars (not io/ext — those are now alias-bound above)
    for k, v := range r.stdlibEnv {
        if input == nil {
            input = make(map[string]any)
        }
        // Skip io module names and extension names — they're now import-gated
        if r.ioRegistry.HasModule(k) || r.extensions.get(k) != nil {
            continue
        }
        input[k] = v
    }

    vm := lang.NewVM(program, r.engine, vmOpts...)
    return vm.Execute(input)
}
```

**Step 4:** Write tests in `lang/import_test.go`:

- Test: `io:http` import recorded in `RequestedModules`
- Test: `ext:bitcode` import recorded in `RequestedModules`
- Test: program with `io:http` import but no HTTP module registered → error
- Test: program with `io:http` import and HTTP module registered → success, alias works
- Test: program without `io:http` import cannot access `http.get` even if module registered

**Step 5:** Run `go test ./... -count=1` — all 171+ tests pass.

**Step 6:** Commit: `fix(go-json): implement two-layer security gating for io:/ext: imports`

---

### Task 2: ExprOptions Type Safety

**Files:**
- Modify: `packages/go-json/io/module.go`
- Test: existing tests should still pass

**Context:** `ExprOptions()` at line 97 does `fn.(func(...any) (any, error))` — hard type assertion that panics if function has wrong signature.

**Step 1:** Replace hard assertion with safe assertion + validation:

```go
func (r *IORegistry) ExprOptions() []expr.Option {
    r.mu.RLock()
    defer r.mu.RUnlock()

    var opts []expr.Option
    for _, mod := range r.modules {
        for fnName, fn := range mod.Functions() {
            exprFn, ok := fn.(func(...any) (any, error))
            if !ok {
                continue // skip non-conforming functions instead of panicking
            }
            fullName := mod.Name() + "." + fnName
            opts = append(opts, expr.Function(fullName, exprFn))
        }
    }
    return opts
}
```

**Step 2:** Run `go build ./...` and `go test ./...`.

**Step 3:** Commit: `fix(go-json): safe type assertion in IORegistry.ExprOptions`

---

### Task 3: SQL Connection Pooling + Close

**Files:**
- Modify: `packages/go-json/io/sql.go`
- Test: `packages/go-json/io/sql_test.go` (new)

**Context:** `getDB()` opens new `*sql.DB` every call in standalone mode, never closes. Need pool-per-DSN cache with proper lifecycle.

**Step 1:** Add pool cache and Close method to SQLModule:

```go
type SQLModule struct {
    security *SecurityConfig
    config   map[string]any
    hostedDB *sql.DB

    pools   map[string]*sql.DB
    poolsMu sync.Mutex

    mu sync.Mutex
    tx *sql.Tx
    sp int
}

func NewSQLModule(security *SecurityConfig) *SQLModule {
    if security == nil {
        security = DefaultSecurityConfig()
    }
    return &SQLModule{
        security: security,
        pools:    make(map[string]*sql.DB),
    }
}

func (m *SQLModule) Close() error {
    m.poolsMu.Lock()
    defer m.poolsMu.Unlock()

    // Auto-rollback any active transaction
    m.mu.Lock()
    if m.tx != nil {
        m.tx.Rollback()
        m.tx = nil
        m.sp = 0
    }
    m.mu.Unlock()

    var firstErr error
    for dsn, db := range m.pools {
        if err := db.Close(); err != nil && firstErr == nil {
            firstErr = err
        }
        delete(m.pools, dsn)
    }
    return firstErr
}

func (m *SQLModule) getDB(dsn string) (*sql.DB, error) {
    if m.hostedDB != nil {
        return m.hostedDB, nil
    }

    if dsn == "" {
        dsn = m.security.SQL.DefaultDSN
    }
    if dsn == "" {
        return nil, fmt.Errorf("sql: DSN is required — set DefaultDSN in config or pass dsn parameter")
    }

    m.poolsMu.Lock()
    defer m.poolsMu.Unlock()

    if db, ok := m.pools[dsn]; ok {
        return db, nil
    }

    maxPools := m.security.SQL.MaxPools
    if maxPools <= 0 {
        maxPools = 5
    }
    if len(m.pools) >= maxPools {
        return nil, fmt.Errorf("sql: max pool limit reached (%d)", maxPools)
    }

    driver := detectDriverFromDSN(dsn)
    if err := m.security.ValidateSQLDriver(driver); err != nil {
        return nil, err
    }

    db, err := sql.Open(driver, dsn)
    if err != nil {
        return nil, fmt.Errorf("sql: cannot open database: %s", err.Error())
    }

    maxPoolSize := m.security.SQL.MaxPoolSize
    if maxPoolSize <= 0 {
        maxPoolSize = 10
    }
    db.SetMaxOpenConns(maxPoolSize)
    db.SetMaxIdleConns(maxPoolSize / 2)
    db.SetConnMaxIdleTime(5 * time.Minute)
    db.SetConnMaxLifetime(30 * time.Minute)

    m.pools[dsn] = db
    return db, nil
}
```

**Step 2:** Add new fields to `SQLSecurityConfig`:

```go
type SQLSecurityConfig struct {
    AllowedDrivers  []string `json:"allowed_drivers"`
    MaxQueryTime    int      `json:"max_query_time"`
    MaxRows         int      `json:"max_rows"`
    DefaultDSN      string   `json:"default_dsn"`
    MaxPoolSize     int      `json:"max_pool_size"`
    MaxPools        int      `json:"max_pools"`
    BlockedKeywords []string `json:"blocked_keywords"`
    MaxQueryLength  int      `json:"max_query_length"`
}
```

**Step 3:** Write tests — pool reuse, pool limit, Close idempotent, auto-rollback on Close.

**Step 4:** Run `go test ./...`.

**Step 5:** Commit: `fix(go-json): SQL connection pooling per-DSN with lifecycle management`

---

### Task 4: Runtime.Close() for Resource Cleanup

**Files:**
- Modify: `packages/go-json/runtime/runtime.go`
- Modify: `packages/go-json/io/module.go` (add Closer interface)
- Modify: `packages/go-json/cmd/go-json/main.go` (add defer rt.Close())
- Test: `packages/go-json/runtime/runtime_test.go`

**Context:** Runtime needs a `Close()` method that propagates to all I/O modules that implement `io.Closer`.

**Step 1:** Add `Closeable` check to IOModule and `Close()` to Runtime:

```go
// In runtime.go
func (r *Runtime) Close() error {
    var firstErr error
    for _, mod := range r.ioRegistry.AllModules() {
        if closer, ok := mod.(interface{ Close() error }); ok {
            if err := closer.Close(); err != nil && firstErr == nil {
                firstErr = err
            }
        }
    }
    return firstErr
}
```

**Step 2:** Add `defer rt.Close()` in CLI `cmdRun()` after creating runtime.

**Step 3:** Write test — runtime with SQL module, Close() called, verify pool closed.

**Step 4:** Commit: `feat(go-json): Runtime.Close() for I/O module resource cleanup`

---

### Task 5: Fix GoJSONRunner — Use CompileFile Instead of Compile

**Files:**
- Modify: `engine/internal/runtime/executor/steps/gojson_runner.go`

**Context:** Runner parses and resolves imports, then calls `rt.Compile(data)` which re-parses raw bytes and loses resolved imports.

**Step 1:** Replace the compile path:

```go
func (r *GoJSONRunner) Run(ctx context.Context, script string, params map[string]any) (any, error) {
    scriptPath := script
    if !filepath.IsAbs(scriptPath) && r.ScriptDir != "" {
        scriptPath = filepath.Join(r.ScriptDir, script)
    }

    // ... runtime setup (unchanged) ...

    compiled, err := rt.CompileFile(scriptPath)
    if err != nil {
        return nil, fmt.Errorf("go-json: compile error in %s: %w", script, err)
    }

    // Remove manual Parse + ResolveImports — CompileFile handles both

    input := make(map[string]any)
    if params != nil {
        for k, v := range params {
            input[k] = v
        }
    }

    result, err := rt.Execute(compiled, input)
    if err != nil {
        return nil, fmt.Errorf("go-json: execution error in %s: %w", script, err)
    }

    return result.Value, nil
}
```

**Step 2:** Run `go build ./...` in engine directory.

**Step 3:** Commit: `fix(go-json): GoJSONRunner uses CompileFile for proper import resolution`

---

### Task 6: Bridge Adapter — Nested Namespaces

**Files:**
- Modify: `engine/internal/runtime/bridge/gojson_adapter.go`

**Context:** Bridge functions use flat keys like `"db.query"` but scripts expect `bc.db.query(...)`. Need nested maps.

**Step 1:** Restructure `BuildGoJSONExtension` to use nested maps:

```go
func BuildGoJSONExtension(bc *Context) gojsonrt.Extension {
    return gojsonrt.Extension{
        Name: "bitcode",
        Functions: map[string]any{
            "model":   func(name string) any { return buildModelProxy(bc, name, false) },
            "db":      buildDBNamespace(bc),
            "http":    buildHTTPNamespace(bc),
            "cache":   buildCacheNamespace(bc),
            "fs":      buildFSNamespace(bc),
            "env":     func(key string) (any, error) { return bc.Env(key) },
            "config":  func(key string) any { return bc.Config(key) },
            "session": buildSessionFunc(bc),
            "log":     buildLogFunc(bc),
            "emit":    buildEmitFunc(bc),
            "call":    buildCallFunc(bc),
            "exec":    buildExecFunc(bc),
            "email":   map[string]any{"send": func(opts map[string]any) (any, error) { return nil, bc.Email().Send(mapToEmailOptions(opts)) }},
            "notify":  map[string]any{"send": func(opts map[string]any) (any, error) { return nil, bc.Notify().Send(mapToNotifyOptions(opts)) }, "broadcast": buildNotifyBroadcast(bc)},
            "storage": buildStorageNamespace(bc),
            "t":       func(key string) any { return bc.T(key) },
            "security": buildSecurityNamespace(bc),
            "audit":   map[string]any{"log": func(opts map[string]any) (any, error) { return nil, bc.Audit().Log(mapToAuditOptions(opts)) }},
            "crypto":  buildCryptoNamespace(bc),
            "execution": buildExecutionNamespace(bc),
        },
    }
}

func buildDBNamespace(bc *Context) map[string]any {
    return map[string]any{
        "query":   buildDBQuery(bc),
        "execute": buildDBExecute(bc),
    }
}

func buildHTTPNamespace(bc *Context) map[string]any {
    return map[string]any{
        "get":    buildHTTPFunc(bc, "GET"),
        "post":   buildHTTPFunc(bc, "POST"),
        "put":    buildHTTPFunc(bc, "PUT"),
        "patch":  buildHTTPFunc(bc, "PATCH"),
        "delete": buildHTTPFunc(bc, "DELETE"),
    }
}

// ... similar for cache, fs, storage, security, crypto, execution
```

**Step 2:** Run `go build ./...` in engine directory.

**Step 3:** Commit: `fix(go-json): bridge adapter uses nested namespaces for dotted access`

---

### Task 7: Wire cmdCodegen to Generators

**Files:**
- Modify: `packages/go-json/cmd/go-json/main.go`

**Context:** `cmdCodegen` prints "not yet implemented" but `codegen/` package exists with working generators.

**Step 1:** Replace stub with actual implementation:

```go
func cmdCodegen(args []string) {
    fs := flag.NewFlagSet("codegen", flag.ExitOnError)
    target := fs.String("target", "", "Target language: go, js, python (required)")
    output := fs.String("output", "", "Write to file (default: stdout)")
    pkg := fs.String("package", "main", "Go package name (only for --target go)")
    fs.Parse(args)

    if fs.NArg() < 1 || *target == "" {
        fmt.Fprintln(os.Stderr, "Usage: go-json codegen <program.json> --target go|js|python [--output file]")
        os.Exit(1)
    }

    programPath := fs.Arg(0)

    reg := stdlib.DefaultRegistry()
    rt := runtime.NewRuntime(
        runtime.WithStdlib(reg.All()),
        runtime.WithStdlibEnv(reg.EnvVars()),
    )
    defer rt.Close()

    compiled, err := rt.CompileFile(programPath)
    if err != nil {
        fmt.Fprintf(os.Stderr, "Error: %s\n", err.Error())
        os.Exit(1)
    }

    var gen cg.CodeGenerator
    switch *target {
    case "go":
        gen = &cg.GoGenerator{PackageName: *pkg}
    case "js", "javascript":
        gen = &cg.JSGenerator{}
    case "python", "py":
        gen = &cg.PythonGenerator{}
    default:
        fmt.Fprintf(os.Stderr, "Error: unsupported target '%s' (use go, js, or python)\n", *target)
        os.Exit(1)
    }

    code, err := gen.Generate(compiled)
    if err != nil {
        fmt.Fprintf(os.Stderr, "Error: %s\n", err.Error())
        os.Exit(1)
    }

    if *output != "" {
        if err := os.WriteFile(*output, []byte(code), 0644); err != nil {
            fmt.Fprintf(os.Stderr, "Error: %s\n", err.Error())
            os.Exit(1)
        }
        fmt.Printf("Generated %s code written to %s\n", gen.Language(), *output)
    } else {
        fmt.Print(code)
    }
}
```

Add import: `cg "github.com/bitcode-framework/go-json/codegen"`

**Step 2:** Run `go build ./...`.

**Step 3:** Commit: `feat(go-json): wire cmdCodegen to Go/JS/Python code generators`

---

## Batch 2: Significant Gap Fixes

### Task 8: Test Runner Import Resolution Fix

**Files:**
- Modify: `packages/go-json/cmd/go-json/test_runner.go`

**Context:** `buildTestWrapper` generates a wrapper JSON with relative import path, but `ExecuteJSON` compiles without base path so imports can't resolve.

**Step 1:** Change `executeTestCase` to use `compileWithBasePath` approach — write wrapper to temp file in same directory as test file, then use `CompileFile`:

```go
func executeTestCase(rt *runtime.Runtime, dir string, tf testFile, tc testCase) (any, error) {
    parts := strings.SplitN(tc.Call, ".", 2)
    if len(parts) != 2 {
        return nil, fmt.Errorf("invalid call format: %s (expected 'module.function')", tc.Call)
    }

    alias := parts[0]
    funcName := parts[1]

    importPath, ok := tf.Import[alias]
    if !ok {
        return nil, fmt.Errorf("import alias '%s' not found", alias)
    }

    wrapperJSON := buildTestWrapper(importPath, funcName, tc.With)

    tmpFile := filepath.Join(dir, fmt.Sprintf("_test_wrapper_%d.json", time.Now().UnixNano()))
    if err := os.WriteFile(tmpFile, []byte(wrapperJSON), 0644); err != nil {
        return nil, fmt.Errorf("cannot write test wrapper: %s", err.Error())
    }
    defer os.Remove(tmpFile)

    compiled, err := rt.CompileFile(tmpFile)
    if err != nil {
        return nil, err
    }

    result, err := rt.Execute(compiled, nil)
    if err != nil {
        return nil, err
    }

    return result.Value, nil
}
```

**Step 2:** Remove unused `fn` variable lookup (lines 229-239 in current code).

**Step 3:** Run `go build ./...`.

**Step 4:** Commit: `fix(go-json): test runner resolves imports via temp file in test directory`

---

### Task 9: Safe Migrate — Parse JSON, Only Rename Keys

**Files:**
- Modify: `packages/go-json/cmd/go-json/main.go`

**Context:** Current `migrateProgram` does blind `strings.ReplaceAll` which corrupts string literals.

**Step 1:** Replace with JSON-aware migration:

```go
func migrateProgram(source, from, to string) (string, []string) {
    var raw any
    if err := json.Unmarshal([]byte(source), &raw); err != nil {
        return source, nil
    }

    renames := map[string]string{
        "unique":     "uniq",
        "startsWith": "hasPrefix",
        "endsWith":   "hasSuffix",
    }

    var changes []string
    migrated := migrateValue(raw, renames, &changes)

    out, err := json.MarshalIndent(migrated, "", "  ")
    if err != nil {
        return source, nil
    }

    return string(out), changes
}

func migrateValue(v any, renames map[string]string, changes *[]string) any {
    switch val := v.(type) {
    case map[string]any:
        result := make(map[string]any, len(val))
        for k, v := range val {
            newKey := k
            if renamed, ok := renames[k]; ok {
                newKey = renamed
                *changes = append(*changes, fmt.Sprintf("renamed key '%s' → '%s'", k, renamed))
            }
            result[newKey] = migrateValue(v, renames, changes)
        }
        return result
    case []any:
        result := make([]any, len(val))
        for i, item := range val {
            result[i] = migrateValue(item, renames, changes)
        }
        return result
    default:
        return v
    }
}
```

**Step 2:** Commit: `fix(go-json): migrate uses JSON-aware key renaming, preserves string values`

---

### Task 10: Extension Alias via Import

**Context:** Handled by Task 1 — `ext:` imports now create aliases. No separate task needed.

---

### Task 11: Remove Bridge `tx` Function

**Files:**
- Modify: `engine/internal/runtime/bridge/gojson_adapter.go`

**Context:** `tx` expects Go callback `func() error` which go-json programs cannot produce.

**Step 1:** Remove `"tx"` from the Functions map. Add comment in design doc that transaction support via bridge requires future work (go-json needs closure/callback support first).

**Step 2:** Commit: `fix(go-json): remove unusable tx bridge function`

---

### Task 12: CLI stdin Support

**Files:**
- Modify: `packages/go-json/cmd/go-json/main.go`

**Step 1:** Add stdin detection in `cmdRun` after flag parsing:

```go
if *inputJSON == "" && *inputFile == "" {
    stat, _ := os.Stdin.Stat()
    if (stat.Mode() & os.ModeCharDevice) == 0 {
        data, err := io.ReadAll(os.Stdin)
        if err != nil {
            fmt.Fprintf(os.Stderr, "Error reading stdin: %s\n", err.Error())
            os.Exit(1)
        }
        if len(data) > 0 {
            if err := json.Unmarshal(data, &input); err != nil {
                fmt.Fprintf(os.Stderr, "Error: invalid JSON from stdin: %s\n", err.Error())
                os.Exit(1)
            }
        }
    }
}
```

**Step 2:** Commit: `feat(go-json): CLI reads JSON input from stdin when piped`

---

### Task 13: Codegen Missing Node Types

**Files:**
- Modify: `packages/go-json/codegen/golang.go`
- Modify: `packages/go-json/codegen/javascript.go`
- Modify: `packages/go-json/codegen/python.go`
- Modify: `packages/go-json/codegen/codegen.go` (remove dead code)

**Step 1:** Add handlers for `NewNode` (struct construction), `ImportNode` (as comment), `ExpressionNode` (standalone expression). Remove unused `stepsToNodes`, `indentSpaces`, and `NodeType` constants.

**Step 2:** Commit: `feat(go-json): codegen handles new/import/expression node types`

---

## Batch 3: SQL Refactor + Multi-Driver + Security

### Task 14: SQL Pooling Per-DSN

**Context:** Already handled in Task 3.

---

### Task 15: DefaultDSN + Multi-Driver Detection

**Files:**
- Modify: `packages/go-json/io/sql.go`
- Modify: `packages/go-json/io/security.go`

**Step 1:** Expand `detectDriverFromDSN`:

```go
func detectDriverFromDSN(dsn string) string {
    switch {
    case strings.HasPrefix(dsn, "postgres://"), strings.HasPrefix(dsn, "postgresql://"):
        return "postgres"
    case strings.HasPrefix(dsn, "mysql://"):
        return "mysql"
    case strings.HasPrefix(dsn, "sqlserver://"):
        return "sqlserver"
    case strings.HasPrefix(dsn, "oracle://"):
        return "oracle"
    case strings.HasPrefix(dsn, "sqlite3://"), strings.HasPrefix(dsn, "sqlite://"), strings.HasPrefix(dsn, "file:"):
        return "sqlite"
    default:
        return "sqlite"
    }
}
```

**Step 2:** Add `DefaultDSN` to `SQLSecurityConfig` (already in Task 3 struct).

**Step 3:** Commit: `feat(go-json): multi-driver SQL detection and DefaultDSN config`

---

### Task 16: SQL Injection Protection

**Files:**
- Modify: `packages/go-json/io/sql.go`
- Test: `packages/go-json/io/sql_test.go`

**Step 1:** Add query validation before execution:

```go
var defaultBlockedKeywords = []string{"DROP", "TRUNCATE", "ALTER", "GRANT", "REVOKE", "CREATE INDEX", "CREATE TABLE"}

func (m *SQLModule) validateQuery(query string) error {
    maxLen := m.security.SQL.MaxQueryLength
    if maxLen <= 0 {
        maxLen = 10000
    }
    if len(query) > maxLen {
        return fmt.Errorf("sql: query exceeds max length (%d chars, max %d)", len(query), maxLen)
    }

    blocked := m.security.SQL.BlockedKeywords
    if blocked == nil {
        blocked = defaultBlockedKeywords
    }

    upper := strings.ToUpper(strings.TrimSpace(query))
    for _, kw := range blocked {
        if strings.Contains(upper, strings.ToUpper(kw)) {
            return fmt.Errorf("sql: query contains blocked keyword '%s'", kw)
        }
    }

    return nil
}
```

**Step 2:** Call `validateQuery` at start of `sqlQuery` and `sqlExecute`.

**Step 3:** Write tests — blocked keywords, max length, parameterized queries pass.

**Step 4:** Commit: `feat(go-json): SQL DDL protection with blocked keywords and query length limit`

---

### Task 17: SQL Transaction Auto-Rollback

**Files:**
- Modify: `packages/go-json/io/sql.go`

**Step 1:** Add `Cleanup()` method (called by `Close()` and can be called after execution):

```go
func (m *SQLModule) Cleanup() {
    m.mu.Lock()
    defer m.mu.Unlock()
    if m.tx != nil {
        m.tx.Rollback()
        m.tx = nil
        m.sp = 0
    }
}
```

**Step 2:** `Close()` already calls this (from Task 3).

**Step 3:** Commit: `feat(go-json): SQL auto-rollback on cleanup for uncommitted transactions`

---

## Batch 4: MongoDB Module

### Task 18: MongoModule + Security Config

**Files:**
- Create: `packages/go-json/io/mongo.go`
- Modify: `packages/go-json/io/security.go` (add MongoSecurityConfig)
- Modify: `packages/go-json/io/all.go` (add Mongo to All())
- Modify: `packages/go-json/go.mod` (add mongo driver dependency)

**Step 1:** Add `MongoSecurityConfig` to security.go:

```go
type MongoSecurityConfig struct {
    DefaultURI        string   `json:"default_uri"`
    AllowedDatabases  []string `json:"allowed_databases"`
    MaxDocumentSize   int64    `json:"max_document_size"`
    MaxResults        int      `json:"max_results"`
    MaxPools          int      `json:"max_pools"`
    BlockedOperations []string `json:"blocked_operations"`
}
```

Add `Mongo MongoSecurityConfig` field to `SecurityConfig`.

**Step 2:** Create `io/mongo.go` with `MongoModule` struct, `NewMongoModule()`, interface implementation, connection pooling per URI, `Close()`.

**Step 3:** Add `go.mongodb.org/mongo-driver/v2` to go.mod.

**Step 4:** Commit: `feat(go-json): MongoDB module scaffold with security config`

---

### Task 19: MongoDB CRUD + Aggregate Operations

**Files:**
- Modify: `packages/go-json/io/mongo.go`

**Step 1:** Implement functions:
- `mongo.find(collection, filter, opts)` → `[]map[string]any`
- `mongo.findOne(collection, filter)` → `map[string]any`
- `mongo.insert(collection, document)` → `{inserted_id}`
- `mongo.insertMany(collection, documents)` → `{inserted_ids}`
- `mongo.update(collection, filter, update)` → `{modified_count}`
- `mongo.delete(collection, filter)` → `{deleted_count}`
- `mongo.count(collection, filter)` → `int`
- `mongo.aggregate(collection, pipeline)` → `[]map[string]any`

**Step 2:** Block `$where` and `$function` operations by default (NoSQL injection prevention).

**Step 3:** Commit: `feat(go-json): MongoDB CRUD and aggregation with injection protection`

---

### Task 20: MongoDB Connection Pooling + Close

**Context:** Handled within Task 18 — pool per URI, Close() closes all clients.

---

## Batch 5: Redis Module

### Task 21: RedisModule + Security Config

**Files:**
- Create: `packages/go-json/io/redis.go`
- Modify: `packages/go-json/io/security.go` (add RedisSecurityConfig)
- Modify: `packages/go-json/io/all.go`
- Modify: `packages/go-json/go.mod` (add redis dependency)

**Step 1:** Add `RedisSecurityConfig`:

```go
type RedisSecurityConfig struct {
    DefaultURI      string   `json:"default_uri"`
    AllowedCommands []string `json:"allowed_commands"`
    BlockedCommands []string `json:"blocked_commands"`
    MaxKeyLength    int      `json:"max_key_length"`
    MaxValueSize    int64    `json:"max_value_size"`
    KeyPrefix       string   `json:"key_prefix"`
}
```

Default `BlockedCommands`: `["FLUSHALL", "FLUSHDB", "CONFIG", "DEBUG", "SHUTDOWN", "SLAVEOF", "REPLICAOF"]`

**Step 2:** Create `io/redis.go` with `RedisModule`, connection pooling, `Close()`.

**Step 3:** Commit: `feat(go-json): Redis module scaffold with security config`

---

### Task 22: Redis Commands

**Files:**
- Modify: `packages/go-json/io/redis.go`

**Step 1:** Implement: `get`, `set`, `del`, `exists`, `expire`, `ttl`, `incr`, `decr`, `hget`, `hset`, `hgetall`, `lpush`, `rpush`, `lrange`, `sadd`, `smembers`, `publish`.

**Step 2:** Commit: `feat(go-json): Redis string, hash, list, set, and pub/sub commands`

---

### Task 23: Redis Auto JSON + KeyPrefix

**Files:**
- Modify: `packages/go-json/io/redis.go`

**Step 1:** `set` auto-serializes non-string values to JSON. `get` auto-deserializes JSON back to map/array. All key operations prepend `KeyPrefix` if configured.

**Step 2:** Commit: `feat(go-json): Redis auto JSON serialize and KeyPrefix tenant isolation`

---

## Batch 6: Moderate Fixes

### Task 24: Windows Path Security

**Files:**
- Modify: `packages/go-json/io/security.go`

**Step 1:** Add Windows default blocked paths. Use case-insensitive comparison. Add directory boundary check (trailing slash).

```go
var defaultBlockedPathsWindows = []string{
    "C:\\Windows\\",
    "C:\\Program Files\\",
    "C:\\Program Files (x86)\\",
}

func init() {
    if runtime.GOOS == "windows" {
        defaultBlockedPaths = append(defaultBlockedPaths, defaultBlockedPathsWindows...)
    }
}
```

Fix prefix matching to use `strings.EqualFold` and ensure directory boundary:

```go
func isPathBlocked(absPath string, blockedPaths []string) bool {
    absPath = strings.ToLower(filepath.ToSlash(absPath))
    for _, blocked := range blockedPaths {
        blocked = strings.ToLower(filepath.ToSlash(blocked))
        if !strings.HasSuffix(blocked, "/") {
            blocked += "/"
        }
        if strings.HasPrefix(absPath+"/", blocked) {
            return true
        }
    }
    return false
}
```

**Step 2:** Commit: `fix(go-json): Windows-aware path security with case-insensitive matching`

---

### Task 25: HTTP Auth Fixes

**Files:**
- Modify: `packages/go-json/io/http.go`

**Step 1:** Return error for invalid auth type. Log warning when both auth and Authorization header present.

**Step 2:** Commit: `fix(go-json): HTTP auth error on invalid type, warn on header conflict`

---

### Task 26: Bridge Silent Type Coercion → Explicit Errors

**Files:**
- Modify: `engine/internal/runtime/bridge/gojson_adapter.go`

**Step 1:** Replace `val, _ := params[0].(string)` patterns with explicit type checks that return errors.

**Step 2:** Commit: `fix(go-json): bridge adapter returns explicit type errors instead of silent coercion`

---

### Task 27: Exec Timeout Naming Cleanup

**Files:**
- Modify: `packages/go-json/io/exec.go`

**Step 1:** Rename `timeoutMs` to `timeoutSec`. Add clarifying variable names.

**Step 2:** Commit: `refactor(go-json): exec timeout variable naming clarity`

---

## Batch 7: Improvements

### Task 28: Regex LRU Cache

**Files:**
- Modify: `packages/go-json/stdlib/regex.go`

**Step 1:** Replace unbounded map with LRU cache (max 1000 entries). Simple eviction: when cache full, clear oldest half.

**Step 2:** Commit: `perf(go-json): regex cache with LRU eviction (max 1000 patterns)`

---

### Task 29: HTTP Client Reuse

**Files:**
- Modify: `packages/go-json/io/http.go`

**Step 1:** Create `http.Client` once in `NewHTTPModule`, reuse across requests. Configure transport with connection pooling.

**Step 2:** Commit: `perf(go-json): reuse HTTP client across requests for connection pooling`

---

### Task 30: deepEqual Fix in Test Runner

**Files:**
- Modify: `packages/go-json/cmd/go-json/test_runner.go`

**Step 1:** Replace JSON marshal comparison with `reflect.DeepEqual` + numeric tolerance.

**Step 2:** Commit: `fix(go-json): test runner deepEqual uses reflect.DeepEqual with numeric tolerance`

---

### Task 31: Bridge convertToAny Optimization

**Files:**
- Modify: `engine/internal/runtime/bridge/gojson_adapter.go`

**Step 1:** Replace JSON roundtrip with direct map construction for known types. Keep JSON fallback for unknown types.

**Step 2:** Commit: `perf(go-json): bridge adapter direct map conversion instead of JSON roundtrip`

---

### Task 32: Dead Code Removal in Codegen

**Files:**
- Modify: `packages/go-json/codegen/codegen.go`

**Step 1:** Remove `indentSpaces()`, `stepsToNodes()`, and unused `NodeType` constants.

**Step 2:** Commit: `refactor(go-json): remove dead code from codegen package`

---

## Batch 8: Missing Tests

### Task 33: I/O Module Tests (HTTP, FS, SQL, Exec)

**Files:**
- Create: `packages/go-json/io/http_test.go` — mock HTTP server, test all methods, auth, security blocking
- Create: `packages/go-json/io/fs_test.go` — temp dir, read/write/append/exists/list/mkdir/remove, security blocking
- Create: `packages/go-json/io/sql_test.go` — SQLite in-memory, query/execute, transactions, pool reuse, DDL blocking
- Create: `packages/go-json/io/exec_test.go` — echo command, timeout, env stripping, denied commands

**Step 1:** Write comprehensive tests for each module.

**Step 2:** Commit: `test(go-json): HTTP, FS, SQL, Exec module tests`

---

### Task 34: MongoDB and Redis Tests

**Files:**
- Create: `packages/go-json/io/mongo_test.go` — mock or integration tests
- Create: `packages/go-json/io/redis_test.go` — miniredis for unit tests

**Step 1:** Write tests. Use `github.com/alicebob/miniredis/v2` for Redis unit tests without real server.

**Step 2:** Commit: `test(go-json): MongoDB and Redis module tests`

---

### Task 35: Bridge Integration Tests

**Files:**
- Create: `engine/internal/runtime/bridge/gojson_adapter_test.go`

**Step 1:** Test nested namespace access, model proxy, type coercion errors.

**Step 2:** Commit: `test(go-json): bridge adapter integration tests`

---

### Task 36: Script Detection + Backward Compat Tests

**Files:**
- Modify: `engine/internal/runtime/executor/steps/steps_test.go`

**Step 1:** Test `.json` → `go-json` runtime detection, `ProcessDefinition.IsGoJSON()`, old format compatibility.

**Step 2:** Commit: `test(go-json): script detection and backward compatibility tests`

---

### Task 37: CLI + Test Runner Tests

**Files:**
- Create: `packages/go-json/cmd/go-json/cli_test.go`
- Create: `packages/go-json/cmd/go-json/test_runner_test.go`

**Step 1:** Test CLI commands via `exec.Command`. Test runner with fixture test files.

**Step 2:** Commit: `test(go-json): CLI and test runner tests`

---

### Task 38: Expanded Existing Test Coverage

**Files:**
- Modify: `packages/go-json/io/security_test.go` — add Windows paths, import-without-enable, enable-without-import
- Modify: `packages/go-json/stdlib/regex_test.go` — add `matches()` alias test
- Modify: `packages/go-json/runtime/extension_test.go` — add error propagation, panic catching, stdlib collision
- Modify: `packages/go-json/codegen/codegen_test.go` — add all step types, comment preservation

**Step 1:** Expand each test file with missing cases.

**Step 2:** Commit: `test(go-json): expanded security, regex, extension, and codegen test coverage`

---

## Batch 9: Documentation

### Task 39: Update All Documentation

**Files:**
- Modify: `packages/go-json/AGENTS.md`
- Modify: `docs/plans/2026-07-14-runtime-engine-phase-4.5c-go-json-io-integration.md` (mark completed items)

**Step 1:** Update AGENTS.md with:
- New package structure (drivers/, mongo, redis)
- Updated security model description (two-layer gating now enforced)
- New test count
- MongoDB and Redis in stdlib layers table
- Runtime.Close() documented
- SQL pooling and DefaultDSN documented

**Step 2:** Commit: `docs(go-json): update AGENTS.md and design doc for Phase 4.5c fixes`

**Step 3:** Push all commits.

---

## Summary

| Batch | Tasks | Scope |
|-------|-------|-------|
| 1: Critical Bugs | 1-7 | Import resolver, type safety, SQL pool, Runtime.Close, runner fix, bridge namespaces, codegen wiring |
| 2: Significant Gaps | 8-13 | Test runner, migrate, extension alias, bridge tx, stdin, codegen nodes |
| 3: SQL Refactor | 14-17 | Pooling, multi-driver, DDL protection, auto-rollback |
| 4: MongoDB | 18-20 | Module, CRUD+aggregate, pooling |
| 5: Redis | 21-23 | Module, commands, auto JSON + KeyPrefix |
| 6: Moderate Fixes | 24-27 | Windows security, HTTP auth, bridge types, exec naming |
| 7: Improvements | 28-32 | Regex LRU, HTTP reuse, deepEqual, convertToAny, dead code |
| 8: Tests | 33-38 | All missing test files + expanded coverage |
| 9: Docs | 39 | AGENTS.md + design doc update |
| **Total** | **39 tasks** | |
