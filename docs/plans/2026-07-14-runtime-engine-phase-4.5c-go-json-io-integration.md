# Phase 4.5c — go-json I/O, Bitcode Integration, Code Generation (Draft)

**Status**: ✅ Done
**Date**: 28 April 2026
**Design Decisions**: See [brainstorming design doc](./2026-04-28-go-json-brainstorming-design.md) for rationale
**Depends on**: Phase 4.5a (Core Language), Phase 4.5b (Modularity)
**Blocks**: Phase 7 (Module "setting")

---

## §1. I/O Modules (Standalone)

These modules provide side-effect capabilities for standalone go-json usage. Bitcode replaces these with its bridge API.

### 1.1 Module Overview

| Module | Functions | Purpose |
|---|---|---|
| `io.http` | 5 | HTTP client |
| `io.fs` | 7 | File system |
| `io.sql` | 5 | SQL database |
| `io.exec` | 1 | Command execution |

Regex is a **stdlib** module (pure function, no side effects) — see §1.7.

### 1.2 HTTP Module

```json
// Available when io.http is enabled
{"let": "resp", "call": "http.get", "with": {
  "url": "'https://api.example.com/users'",
  "headers": "{'Authorization': 'Bearer ' + token}"
}}

{"let": "resp", "call": "http.post", "with": {
  "url": "'https://api.example.com/users'",
  "body": "{'name': name, 'email': email}",
  "headers": "{'Content-Type': 'application/json'}"
}}
```

**Functions:**

| Function | Input | Returns |
|---|---|---|
| `http.get(url, headers?, timeout?, auth?)` | URL + optional headers/timeout/auth | `{status, body, headers}` |
| `http.post(url, body?, headers?, timeout?, auth?)` | URL + body + optional headers/auth | `{status, body, headers}` |
| `http.put(url, body?, headers?, timeout?, auth?)` | Same as post | `{status, body, headers}` |
| `http.patch(url, body?, headers?, timeout?, auth?)` | Same as post | `{status, body, headers}` |
| `http.delete(url, headers?, timeout?, auth?)` | Same as get | `{status, body, headers}` |

**Auth parameter:**

```json
// Bearer token
{"auth": {"type": "bearer", "token": "my-jwt-token"}}

// Basic auth
{"auth": {"type": "basic", "username": "user", "password": "pass"}}
```

When both `auth` and `headers.Authorization` are provided, `auth` takes precedence and a compile warning is emitted.

**Response shape:**
```json
{
  "status": 200,
  "body": {"any": "parsed JSON or string"},
  "headers": {"Content-Type": "application/json"}
}
```

Non-JSON response bodies are returned as string. Redirects followed up to 10 hops. Response body truncated at `HTTPSecurityConfig.MaxResponseSize` (default 10MB).

### 1.3 File System Module

```json
{"let": "content", "call": "fs.read", "with": {"path": "'data.json'"}}
{"call": "fs.write", "with": {"path": "'output.txt'", "content": "result"}}
{"call": "fs.append", "with": {"path": "'log.txt'", "content": "line"}}
{"let": "exists", "call": "fs.exists", "with": {"path": "'config.json'"}}
```

| Function | Input | Returns |
|---|---|---|
| `fs.read(path, encoding?)` | File path, encoding (default `"utf-8"`) | `string` (file content) |
| `fs.write(path, content, encoding?)` | Path + content | `void` |
| `fs.append(path, content, encoding?)` | Path + content | `void` |
| `fs.exists(path)` | Path | `bool` |
| `fs.list(path)` | Directory path | `[]string` (file names) |
| `fs.mkdir(path)` | Directory path | `void` |
| `fs.remove(path, recursive?)` | Path, recursive (default `false`) | `void` |

**Behavior notes:**
- `fs.read` on directory → error
- `fs.write` with empty content → creates empty file
- `fs.append` to non-existent file → creates file (like `>>` in bash)
- `fs.remove` on non-empty directory without `recursive: true` → error
- Symlinks → resolved to absolute path, re-checked against allowed paths
- Binary files → `fs.read` with `encoding: "base64"` returns base64-encoded string
- All paths validated against `FSSecurityConfig` (allowed/blocked paths)
- File size limits enforced on read and write via `FSSecurityConfig.MaxFileSize`

### 1.4 SQL Module

```json
// Standalone mode: DSN from program
{"let": "users", "call": "sql.query", "with": {
  "dsn": "'postgres://localhost/mydb'",
  "query": "'SELECT * FROM users WHERE age > ?'",
  "args": "[18]"
}}

// Hosted mode: DSN from host (WithSQLConnection)
{"let": "users", "call": "sql.query", "with": {
  "query": "'SELECT * FROM users WHERE age > ?'",
  "args": "[18]"
}}

{"let": "result", "call": "sql.execute", "with": {
  "query": "'UPDATE users SET active = true WHERE id = ?'",
  "args": "[userId]"
}}
```

| Function | Input | Returns |
|---|---|---|
| `sql.query(query, args?, dsn?)` | SQL + params + optional DSN | `{rows, columns, count}` |
| `sql.execute(query, args?, dsn?)` | SQL + params + optional DSN | `{rows_affected, last_insert_id}` |
| `sql.begin()` | — | `void` (starts transaction) |
| `sql.commit()` | — | `void` (commits transaction) |
| `sql.rollback()` | — | `void` (rolls back transaction) |

**Dual-mode DSN:**
- **Standalone mode:** `dsn` parameter required. Validated against `SQLSecurityConfig.AllowedDrivers`.
- **Hosted mode:** Host provides connection via `WithSQLConnection(db *sql.DB)`. `dsn` parameter ignored. If `dsn` is provided in hosted mode, it is silently ignored (not an error).
- Runtime auto-detects mode based on whether `WithSQLConnection` was called.

**Return shapes:**

```json
// sql.query
{
  "rows": [{"name": "Alice", "age": 30}, {"name": "Bob", "age": 25}],
  "columns": ["name", "age"],
  "count": 2
}

// sql.execute
{
  "rows_affected": 5,
  "last_insert_id": 42
}
```

**Transaction pattern:**
```json
{"call": "sql.begin"},
{"let": "result", "call": "sql.execute", "with": {"query": "'INSERT INTO orders ...', "args": "[...]"}},
{"call": "sql.commit"}
```

If an error occurs between `begin` and `commit`, auto-rollback is triggered. Nested `begin` calls use savepoints. Transaction timeout follows `SQLSecurityConfig.MaxQueryTime`.

**Unified Query Parameters (Phase 4.5d):**

go-json uses `?` as the universal positional placeholder and `:name` for named parameters. The SQL module auto-translates to driver-specific syntax (`$1` for Postgres, `@p1` for SQL Server, etc.) before execution. See Phase 4.5d design doc §16 for full specification.

**Edge cases:**
- `sql.query` with 0 rows → `{"rows": [], "columns": ["name", "age"], "count": 0}` — columns still present
- NULL values → `nil` in go-json
- `last_insert_id` on PostgreSQL → returns 0 (PostgreSQL uses `RETURNING` instead)
- Large result set → truncated at `SQLSecurityConfig.MaxRows` (default 10000)
- Parameterized queries ONLY — no string interpolation (SQL injection prevention)

### 1.5 Exec Module

```json
{"let": "result", "call": "exec.run", "with": {
  "cmd": "'pandoc'",
  "args": "['input.md', '-o', 'output.pdf']",
  "cwd": "'/tmp'",
  "timeout": "30000",
  "env": "{'PATH': '/usr/bin', 'HOME': '/tmp'}"
}}
```

| Function | Input | Returns |
|---|---|---|
| `exec.run(cmd, args?, cwd?, timeout?, env?)` | Command + options | `{exit_code, stdout, stderr}` |

**Parameters:**
- `cmd` — command name, must be in `ExecSecurityConfig.AllowedCommands` whitelist
- `args` — array of arguments (NOT string — no shell expansion)
- `cwd` — working directory (default: system temp)
- `timeout` — milliseconds (default: `ExecSecurityConfig.MaxTimeout`, max 60000)
- `env` — environment variables map. If provided, ONLY these env vars are available (isolated). If omitted, inherits host environment minus `EngineSecrets`.

**Security:**
- `EngineSecrets` (`JWT_SECRET`, `DB_PASSWORD`, `ENCRYPTION_KEY`, `SMTP_PASSWORD`, `STORAGE_S3_SECRET_KEY`, `STORAGE_S3_ACCESS_KEY`) are ALWAYS stripped from inherited environment.
- `DeniedCommands` (`rm`, `rmdir`, `del`, `format`, `shutdown`, `reboot`, `halt`, `poweroff`, `dd`, `mkfs`, `fdisk`) are ALWAYS blocked regardless of whitelist.
- No pipe, no redirect, no shell metacharacters — arguments as array only.
- stdout and stderr truncated at `ExecSecurityConfig.MaxOutputSize` (default 1MB).

**Edge cases:**
- Command not in whitelist → error
- Command not found on system → clear error
- Command hangs → timeout kills process
- Non-zero exit code → not an error, returned in `exit_code`
- Arguments with spaces → handled correctly (array, not string)

### 1.6 I/O Security — Enable/Disable

```go
// Standalone: enable all I/O
rt := gojson.NewRuntime(
    gojson.WithStdlib(stdlib.All()),
    gojson.WithIO(goio.All()),
)

// Selective: only HTTP, no FS/SQL/exec
rt := gojson.NewRuntime(
    gojson.WithStdlib(stdlib.All()),
    gojson.WithIO(goio.HTTP()),
)

// Locked down: no I/O at all
rt := gojson.NewRuntime(
    gojson.WithStdlib(stdlib.All()),
    gojson.WithoutIO(),
)
```

I/O modules must also be **explicitly imported** in the program. They are not magically available just because the host enabled them at runtime.

```json
{
  "import": {
    "http": "io:http",
    "fs": "io:fs",
    "sql": "io:sql",
    "exec": "io:exec"
  }
}
```

Runtime enable/disable and program imports are separate gates:

- Program import decides what the script is allowed to reference.
- Runtime configuration decides what the host actually exposes.
- Imported but disabled I/O → compile error.
- Enabled but not imported I/O → symbol not found.

Host application controls which I/O modules are available. Programs that call disabled I/O → compile error: `"function 'http.get' not available (I/O disabled)"`.

### 1.7 Regex (Stdlib)

Regex is a **stdlib** module, not an I/O module. It has no side effects and requires no security gating. Regex functions are available without import (part of stdlib, like `len`, `upper`, etc.).

```json
{"let": "valid", "expr": "regex.match(email, '^[a-z]+@[a-z]+\\.[a-z]+$')"}
{"let": "numbers", "expr": "regex.findAll(text, '\\d+')"}
{"let": "cleaned", "expr": "regex.replace(text, '\\s+', ' ')"}
```

| Function | Input | Returns |
|---|---|---|
| `regex.match(str, pattern)` | String + regex | `bool` |
| `regex.findAll(str, pattern)` | String + regex | `[]string` |
| `regex.replace(str, pattern, replacement)` | String + regex + replacement | `string` |

**Implementation notes:**
- Existing `matches(str, pattern)` in stdlib is retained as alias for `regex.match`.
- Compiled regexes are cached (map + mutex) for performance.
- ReDoS prevention: max pattern length 1000 chars, max input size 1MB, compile timeout 100ms.

---

## §2. Bitcode Bridge Integration

### 2.1 How Bitcode Uses go-json

Bitcode disables raw I/O and injects its bridge API as an extension:

```go
import (
    gojson "github.com/bitcode-framework/go-json/lang"
    "github.com/bitcode-framework/go-json/stdlib"
    "github.com/bitcode-framework/bitcode/internal/runtime/bridge"
)

func createGoJSONRuntime(bc *bridge.Context, limits gojson.Limits) *gojson.Runtime {
    return gojson.NewRuntime(
        gojson.WithStdlib(stdlib.All()),
        gojson.WithoutIO(),
        gojson.WithExtension("bitcode", buildBitcodeExtension(bc)),
        gojson.WithLimits(limits),
        gojson.WithSession(gojson.Session{
            UserID:   bc.Session().UserID,
            Username: bc.Session().Username,
            Email:    bc.Session().Email,
            Locale:   bc.Session().Locale,
            TenantID: bc.Session().TenantID,
            Groups:   bc.Session().Groups,
            Context:  bc.Session().Context,
        }),
    )
}

func buildBitcodeExtension(bc *bridge.Context) gojson.Extension {
    return gojson.Extension{
        Functions: map[string]any{
            // Model CRUD
            "model":     func(name string) any { return bc.Model(name) },

            // Model bulk operations (via model().*)
            // model().Search, Get, Create, Write, Delete, Count, Sum, Upsert
            // model().CreateMany, WriteMany, DeleteMany, UpsertMany
            // model().AddRelation, RemoveRelation, SetRelation, LoadRelation
            // model().Sudo() → SudoModelHandle

            // Database
            "db.query":   func(sql string, args ...any) ([]map[string]any, error) { return bc.DB().Query(sql, args...) },
            "db.execute": func(sql string, args ...any) (*bridge.ExecDBResult, error) { return bc.DB().Execute(sql, args...) },

            // HTTP
            "http.get":    func(url string, opts ...map[string]any) (any, error) { ... },
            "http.post":   func(url string, opts ...map[string]any) (any, error) { ... },
            "http.put":    func(url string, opts ...map[string]any) (any, error) { ... },
            "http.patch":  func(url string, opts ...map[string]any) (any, error) { ... },
            "http.delete": func(url string, opts ...map[string]any) (any, error) { ... },

            // Cache
            "cache.get": func(key string) (any, error) { return bc.Cache().Get(key) },
            "cache.set": func(key string, val any, opts ...map[string]any) error { ... },
            "cache.del": func(key string) error { return bc.Cache().Del(key) },

            // File system
            "fs.read":   func(path string) (string, error) { return bc.FS().Read(path) },
            "fs.write":  func(path string, content string) error { return bc.FS().Write(path, content) },
            "fs.exists": func(path string) (bool, error) { return bc.FS().Exists(path) },
            "fs.list":   func(path string) ([]string, error) { return bc.FS().List(path) },
            "fs.mkdir":  func(path string) error { return bc.FS().Mkdir(path) },
            "fs.remove": func(path string) error { return bc.FS().Remove(path) },

            // Environment & config
            "env":    func(key string) (string, error) { return bc.Env(key) },
            "config": func(key string) any { return bc.Config(key) },

            // Session
            "session": func() bridge.Session { return bc.Session() },

            // Logging
            "log": func(level, msg string, data ...map[string]any) { bc.Log(level, msg, data...) },

            // Events & process
            "emit": func(event string, data map[string]any) error { return bc.Emit(event, data) },
            "call": func(process string, input map[string]any) (any, error) { return bc.Call(process, input) },

            // Command execution
            "exec": func(cmd string, args []string, opts ...map[string]any) (any, error) { ... },

            // Email
            "email.send": func(opts map[string]any) error { ... },

            // Notifications
            "notify.send":      func(opts map[string]any) error { ... },
            "notify.broadcast": func(channel string, data map[string]any) error { return bc.Notify().Broadcast(channel, data) },

            // Storage
            "storage.upload":   func(opts map[string]any) (any, error) { ... },
            "storage.url":      func(id string) (string, error) { return bc.Storage().URL(id) },
            "storage.download": func(id string) ([]byte, error) { return bc.Storage().Download(id) },
            "storage.delete":   func(id string) error { return bc.Storage().Delete(id) },

            // i18n
            "t": func(key string) string { return bc.T(key) },

            // Security
            "security.permissions": func(model string) (any, error) { return bc.Security().Permissions(model) },
            "security.hasGroup":    func(group string) (bool, error) { return bc.Security().HasGroup(group) },
            "security.groups":      func() ([]string, error) { return bc.Security().Groups() },

            // Audit
            "audit.log": func(opts map[string]any) error { ... },

            // Crypto
            "crypto.encrypt": func(plaintext string) (string, error) { return bc.Crypto().Encrypt(plaintext) },
            "crypto.decrypt": func(ciphertext string) (string, error) { return bc.Crypto().Decrypt(ciphertext) },
            "crypto.hash":    func(value string) (string, error) { return bc.Crypto().Hash(value) },
            "crypto.verify":  func(value, hash string) (bool, error) { return bc.Crypto().Verify(value, hash) },

            // Execution log
            "execution.current": func() any { return bc.Execution().Current() },
            "execution.search":  func(opts map[string]any) ([]map[string]any, error) { ... },
            "execution.get":     func(id string) (map[string]any, error) { ... },
            "execution.retry":   func(id string) (map[string]any, error) { return bc.Execution().Retry(id) },
            "execution.cancel":  func(id string) error { return bc.Execution().Cancel(id) },

            // Transaction
            "tx": func(fn func() error) error { return bc.Tx(func(tx *bridge.Context) error { return fn() }) },
        },
    }
}
```

### 2.2 Extension API

```go
type Extension struct {
    Name      string
    Functions map[string]any    // function name → Go function
    Structs   map[string]any    // struct name → struct definition (reserved for future)
    Constants map[string]any    // constant name → value (reserved for future)
}

// Runtime option
func WithExtension(name string, ext Extension) Option
```

Extensions are accessed in programs via `ext:name`:

```json
{
  "import": {
    "bc": "ext:bitcode"
  },
  "steps": [
    {"let": "leads", "call": "bc.model('lead').search", "with": {
      "domain": "[['status', '=', 'new']]",
      "limit": 100
    }},
    {"call": "bc.log", "with": {
      "level": "'info'",
      "msg": "'Found ' + string(len(leads)) + ' leads'"
    }}
  ]
}
```

### 2.3 How Bitcode Replaces Raw I/O with Bridge

| Standalone I/O | Bitcode Bridge | Difference |
|---|---|---|
| `http.get(url)` | `bc.http.get(url)` | Bridge uses tls-client, has rate limiting |
| `fs.read(path)` | `bc.fs.read(path)` | Bridge enforces fs_allow/fs_deny |
| `sql.query(dsn, sql)` | `bc.db.query(sql)` | Bridge uses connection pool, tenant-scoped |
| `exec.run(cmd)` | `bc.exec(cmd)` | Bridge enforces exec_allow whitelist |

Programs written for bitcode use `ext:bitcode`. Programs written for standalone use raw I/O. The language is the same — only the I/O layer differs.

### 2.4 Bitcode Model Access Pattern

The `bc.model()` pattern needs special handling because it returns a proxy object with methods:

```json
// In bitcode context
{"let": "leads", "expr": "bc.model('lead').search({'domain': [['status', '=', 'new']]})"}
{"let": "lead", "expr": "bc.model('lead').get(leadId)"}
{"call": "bc.model('lead').write", "with": {"id": "leadId", "data": "{'status': 'processed'}"}}
```

This works because `bc.model('lead')` returns a Go object with methods, and expr-lang can call methods on Go objects.

**Available model methods:**

| Method | Returns | Notes |
|---|---|---|
| `.search(opts)` | `[]map` | Search with domain/limit/offset |
| `.get(id)` | `map` | Get single record |
| `.create(data)` | `map` | Create record, returns created |
| `.write(id, data)` | `void` | Update record |
| `.delete(id)` | `void` | Soft delete |
| `.count(opts)` | `int` | Count matching records |
| `.sum(field, opts)` | `float` | Sum field values |
| `.upsert(data, unique)` | `map` | Create or update |
| `.createMany(records)` | `[]map` | Bulk create |
| `.writeMany(ids, data)` | `{affected}` | Bulk update |
| `.deleteMany(ids)` | `{affected}` | Bulk delete |
| `.upsertMany(records, unique)` | `[]map` | Bulk upsert |
| `.addRelation(id, field, ids)` | `void` | Add M2M relation |
| `.removeRelation(id, field, ids)` | `void` | Remove M2M relation |
| `.setRelation(id, field, ids)` | `void` | Replace M2M relation |
| `.loadRelation(id, field)` | `[]map` | Load related records |
| `.sudo()` | `SudoModelHandle` | Bypass permissions |

---

## §3. Bitcode Process Engine Migration

### 3.1 Current Process Engine → go-json

Current process engine (executor/) will be replaced by go-json. Migration path:

| Current Step Type | go-json Equivalent | Complexity |
|---|---|---|
| `validate` | `if` + `error` steps | Low — direct mapping |
| `query` | `bc.model(name).search(opts)` | Medium — domain conversion needed |
| `create` | `bc.model(name).create(data)` | Low |
| `update` | `bc.model(name).write(id, data)` | Low |
| `delete` | `bc.model(name).delete(id)` | Low |
| `if` | `if`/`elif`/`else` (real expression evaluator) | Low — syntax change only |
| `switch` | `switch`/`cases` (same) | Low |
| `loop` | `for`/`in` (with break/continue) | Medium — break/continue mapping |
| `emit` | `bc.emit(event, data)` | Low |
| `call` | `call` with isolated scope (improved) | Medium — scope isolation differs |
| `script` | `call` to external script (or inline go-json function) | Low |
| `http` | `bc.http.get/post/...` (through bridge, not raw) | Medium — response shape may differ |
| `assign` | `let`/`set` (improved with expressions) | Low |
| `log` | `bc.log(level, msg)` | Low |
| `upsert` | `bc.model(name).upsert(data, unique)` | Low |
| `count` | `bc.model(name).count(opts)` | Low |
| `sum` | `bc.model(name).sum(field, opts)` | Low |

### 3.2 JSON Script Support in Bitcode

Bitcode's script step handler detects `.json` files and routes to go-json:

```json
// In bitcode process definition
{
  "type": "script",
  "script": "scripts/process_data.json",
  "runtime": "go-json"
}
```

Or auto-detected by extension:
```go
func detectRuntimeFromExtension(script string) string {
    switch {
    case strings.HasSuffix(script, ".js"):   return "javascript"
    case strings.HasSuffix(script, ".go"):   return "go"
    case strings.HasSuffix(script, ".ts"):   return "node"
    case strings.HasSuffix(script, ".py"):   return "python"
    case strings.HasSuffix(script, ".json"): return "go-json"  // NEW
    }
    return ""
}
```

### 3.3 Backward Compatibility

Current process JSON format (with `type: "query"`, `type: "create"`, etc.) continues to work. go-json is an ADDITIONAL runtime, not a replacement of the existing format.

- Processes without `"runtime": "go-json"` → old executor (default, no change)
- Processes with `"runtime": "go-json"` → go-json VM with bitcode bridge
- Old format enters maintenance mode — new features only added to go-json format
- Migration tool (`go-json migrate`) can convert old process format to go-json format

---

## §4. Code Generation Foundation

### 4.1 AST Export

go-json programs are parsed into a well-defined AST. This AST can be exported for code generation:

```go
program, err := gojson.Parse(jsonBytes)
ast := program.AST()

// Export as JSON (for external tools)
astJSON, _ := json.Marshal(ast)

// Export as Go code
goCode := codegen.ToGo(ast)

// Export as JavaScript
jsCode := codegen.ToJS(ast)
```

### 4.2 Code Generation Targets

| Target | Feasibility | Notes |
|---|---|---|
| Go | High | Direct mapping — structs, functions, control flow all map 1:1 |
| JavaScript | High | Most constructs map directly. Structs → classes. |
| Python | High | Most constructs map directly. Structs → dataclasses. |
| SQL | Medium | Only data-heavy programs (queries, transforms) |
| BPMN XML | Medium | Only workflow-style programs (steps, conditions, parallel) |

### 4.3 AST Node Types

```go
type NodeType string

const (
    NodeProgram    NodeType = "program"
    NodeLet        NodeType = "let"
    NodeSet        NodeType = "set"
    NodeIf         NodeType = "if"
    NodeSwitch     NodeType = "switch"
    NodeForIn      NodeType = "for_in"
    NodeForRange   NodeType = "for_range"
    NodeWhile      NodeType = "while"
    NodeBreak      NodeType = "break"
    NodeContinue   NodeType = "continue"
    NodeReturn     NodeType = "return"
    NodeCall       NodeType = "call"
    NodeTry        NodeType = "try"
    NodeError      NodeType = "error"
    NodeLog        NodeType = "log"
    NodeNew        NodeType = "new"
    NodeParallel   NodeType = "parallel"
    NodeFunction   NodeType = "function"
    NodeStruct     NodeType = "struct"
    NodeImport     NodeType = "import"
    NodeExpression NodeType = "expression"
)
```

### 4.4 Code Generation Example

**go-json source:**
```json
{
  "name": "factorial",
  "functions": {
    "factorial": {
      "params": {"n": "int"},
      "returns": "int",
      "steps": [
        {"if": "n <= 1", "then": [{"return": "1"}]},
        {"let": "sub", "call": "factorial", "with": {"n": "n - 1"}},
        {"return": "n * sub"}
      ]
    }
  },
  "steps": [
    {"let": "result", "call": "factorial", "with": {"n": "10"}},
    {"return": "result"}
  ]
}
```

**Generated Go:**
```go
package main

func factorial(n int) int {
    if n <= 1 {
        return 1
    }
    sub := factorial(n - 1)
    return n * sub
}

func main() {
    result := factorial(10)
    fmt.Println(result)
}
```

**Generated JavaScript:**
```javascript
function factorial(n) {
    if (n <= 1) return 1;
    const sub = factorial(n - 1);
    return n * sub;
}

const result = factorial(10);
console.log(result);
```

**Generated Python:**
```python
def factorial(n: int) -> int:
    if n <= 1:
        return 1
    sub = factorial(n - 1)
    return n * sub

result = factorial(10)
print(result)
```

### 4.5 Code Generation Limitations

| Limitation | Why |
|---|---|
| Dynamic types → generated code may need type assertions | go-json allows `any`, target languages may not |
| Extension calls (`ext:bitcode`) → not portable | Host-specific, cannot generate standalone code |
| Parallel → different concurrency models per language | Go: goroutines, JS: Promise.all, Python: asyncio |
| I/O calls → different libraries per language | HTTP client, FS API differ per language |

Code generation works best for **pure logic** (functions, control flow, data transformation). I/O-heavy programs need manual adaptation.

---

## §5. Standalone CLI

### 5.1 CLI Runner

```bash
# Run a program
go-json run program.json --input '{"name": "Alice"}'

# Run with input from file
go-json run program.json --input-file input.json

# Run with limits
go-json run program.json --timeout 60s --max-depth 500

# Run with I/O modules enabled
go-json run program.json --io http,fs

# Validate (compile check, no execution)
go-json check program.json

# Run tests
go-json test tests/

# Export AST
go-json ast program.json --output ast.json

# Generate code
go-json codegen program.json --target go --output program.go
go-json codegen program.json --target js --output program.js
go-json codegen program.json --target python --output program.py

# Migrate deprecated syntax
go-json migrate program.json --from v1 --to v2
```

### 5.2 REPL (Future)

Interactive mode for experimentation:

```bash
go-json repl

> let x = 42
> x + 1
43
> let items = [1, 2, 3, 4, 5]
> filter(items, # > 3)
[4, 5]
> sum(items)
15
```

This is a nice-to-have, not a must for Phase 4.5c.

### 5.3 Testing Framework

Phase 4.5c includes a built-in test runner for go-json programs. Test files use normal go-json structure plus `"test": true` and a `"cases"` array.

**Test file format:**

```json
{
  "name": "test_discount",
  "test": true,
  "import": {
    "calc": "../functions/discount.json"
  },
  "cases": [
    {
      "_c": "Gold tier gets 15% discount",
      "call": "calc.calculateDiscount",
      "with": {
        "price": "100.0",
        "quantity": "5",
        "tier": "'gold'"
      },
      "expect": 75.0
    },
    {
      "_c": "Unknown tier gets 5% discount",
      "call": "calc.calculateDiscount",
      "with": {
        "price": "200.0",
        "quantity": "2",
        "tier": "'bronze'"
      },
      "expect": 20.0
    }
  ]
}
```

**CLI:**

```bash
go-json test tests/
```

**Expected output:**

```bash
# ✓ test_discount: Gold tier gets 15% discount (2ms)
# ✗ test_discount: Unknown tier gets 5% discount
#   Expected: 20.0
#   Got: 10.0
# 1 passed, 1 failed
```

This is a Phase 4.5c feature, but the file format is designed now for consistency across CLI, editor, and future CI integration.

### 5.4 Migration Tool

Language evolution needs an official upgrade path for deprecated syntax.

```bash
go-json migrate program.json --from v1 --to v2
go-json migrate program.json --dry-run    # show changes without applying
go-json migrate program.json --output migrated.json
```

The migration tool auto-transforms deprecated syntax to the target version where a safe rewrite is known. Typical examples include renamed stdlib functions, deprecated aliases, and structural syntax that has a deterministic replacement.

The migrated output should remain valid go-json source, preserve non-conflicting program structure (including JSONC comments), and report any constructs that require manual review.

Without `--from`, the tool auto-detects the source version from program syntax.

---

## §6. Implementation Tasks

| # | Task | Effort | Priority |
|---|---|---|---|
| **I/O Framework** | | | |
| 1 | I/O module interface and registry | Small | Must |
| 2 | I/O security layer (two-layer gating) | Medium | Must |
| **I/O Modules** | | | |
| 3 | HTTP module (get/post/put/patch/delete + auth) | Large | Must |
| 4 | FS module (read/write/append/exists/list/mkdir/remove) | Medium | Must |
| 5 | SQL module (query/execute/begin/commit/rollback, dual-mode DSN) | Large | Must |
| 6 | Exec module (run, env isolation) | Medium | Must |
| **Stdlib** | | | |
| 7 | Regex module (match/findAll/replace, caching, ReDoS prevention) | Small | Must |
| **Bitcode Integration** | | | |
| 8 | Extension API (WithExtension, Extension struct with Structs/Constants) | Medium | Must |
| 9 | Bitcode bridge adapter (bridge.Context → Extension, 50+ functions) | Large | Must |
| 10 | Script handler: detect .json → route to go-json | Small | Must |
| 11 | Backward compatibility layer for old process format | Medium | Must |
| 12 | Replace current process engine data steps with bridge calls | Large | Must |
| **Code Generation** | | | |
| 13 | AST export (JSON serialization) | Medium | Must |
| 14 | Go code generator | Large | Should |
| 15 | JavaScript code generator | Large | Should |
| 16 | Python code generator | Large | Should |
| **CLI** | | | |
| 17 | CLI scaffold (subcommand dispatch) | Small | Must |
| 18 | `go-json run` command (--input, --input-file, --timeout, --max-depth, --io) | Medium | Must |
| 19 | `go-json check` command (validate) | Small | Must |
| 20 | `go-json test` command (test runner) | Medium | Must |
| 21 | `go-json ast` command (export AST, --output) | Small | Should |
| 22 | `go-json codegen` command (--target, --output) | Small | Should |
| 23 | `go-json migrate` command (--from, --to, --dry-run) | Medium | Should |
| **Tests** | | | |
| 24 | Tests: I/O modules (HTTP mock, FS temp, SQL SQLite, Exec echo) | Medium | Must |
| 25 | Tests: I/O security (path traversal, injection, unauthorized) | Medium | Must |
| 26 | Tests: Regex (match, findAll, replace, caching, ReDoS) | Small | Must |
| 27 | Tests: Extension API | Medium | Must |
| 28 | Tests: Bitcode bridge integration | Large | Must |
| 29 | Tests: .json script detection + backward compat | Small | Must |
| 30 | Tests: AST export roundtrip | Medium | Must |
| 31 | Tests: Code generation (Go/JS/Python) | Large | Should |
| 32 | Tests: CLI commands (run/check/test/ast/codegen/migrate) | Medium | Must |
| 33 | Tests: Test runner (format, pass/fail, float tolerance) | Medium | Must |
