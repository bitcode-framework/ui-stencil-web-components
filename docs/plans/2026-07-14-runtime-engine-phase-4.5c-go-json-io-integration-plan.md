# Phase 4.5c — go-json I/O + Integration: Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Add I/O modules (HTTP, FS, SQL, Exec), regex stdlib, bitcode bridge integration, process engine migration, CLI runner, test framework, migration tool, and code generation to go-json.

**Architecture:** I/O modules are explicitly imported (`"io:http"`) and enabled at runtime level (two-layer security). Regex is stdlib (no import needed). Bitcode bridge injected as extension (`"ext:bitcode"`). CLI provides run/check/test/ast/codegen/migrate commands. Code generators produce Go/JS/Python from AST.

**Tech Stack:** Go 1.24+, net/http, os, database/sql, os/exec, regexp, encoding/json

**Design Doc:** `2026-07-14-runtime-engine-phase-4.5c-go-json-io-integration.md`
**Decisions:** `2026-04-28-go-json-brainstorming-design.md` (OQ-9, BS-4)
**Depends on:** Phase 4.5b complete

---

## Critical Path

```
Task 1 (I/O interface) ──► Task 2 (security) ──► Task 3-6 (I/O modules)
                                                       │
Task 7 (regex/stdlib) ────────────────────────────────┘
                                                       │
Task 8 (extension API) ──► Task 9 (bitcode adapter) ──► Task 10 (scripts/*.json)
                                                       │
                                          Task 11 (backward compat) ──► Task 12 (data step replacement)
                                                       │
Task 13 (CLI scaffold) ──► Task 14-19 (CLI commands) ─┘
                                                       │
Task 20 (codegen framework) ──► Task 21-23 (generators)
                                                       │
                                                 Task 24-33 (tests)
```

---

## Batch 1: I/O Module Framework

### Task 1: I/O Module Interface

**Files:** Create `io/module.go`
**Effort:** Small
**Depends on:** Phase 4.5b complete

**Steps:**
1. Define `IOModule` interface:
   ```go
   type IOModule interface {
       Name() string                    // "http", "fs", "sql", "exec"
       Functions() map[string]any       // function map for expr-lang registration
       SetConfig(cfg map[string]any)    // runtime configuration
   }
   ```
2. Define `IORegistry` — map of module name → IOModule
3. Implement `RegisterModule(name, module)`, `GetModule(name)`, `AllModules()`
4. Integration with runtime: `WithIO(modules ...IOModule)` option
5. Integration with runtime: `WithoutIO()` option (explicitly disables all I/O)

**Acceptance criteria:** I/O modules registerable and accessible via `"io:name"` import

**Commit:** `feat(go-json): I/O module interface and registry`

---

### Task 2: I/O Security Layer

**Files:** Create `io/security.go`
**Effort:** Medium
**Depends on:** Task 1

**Steps:**
1. Define `SecurityConfig`:
   ```go
   type SecurityConfig struct {
       EnabledModules  []string          // ["http", "fs", "sql", "exec"] — which modules allowed
       HTTP            HTTPSecurityConfig
       FS              FSSecurityConfig
       SQL             SQLSecurityConfig
       Exec            ExecSecurityConfig
   }

   type HTTPSecurityConfig struct {
       AllowedHosts    []string  // ["api.example.com", "*.internal.com"]
       BlockedHosts    []string  // ["localhost", "169.254.*"]
       MaxResponseSize int64     // bytes, default 10MB
       Timeout         int       // seconds, default 30
   }

   type FSSecurityConfig struct {
       AllowedPaths    []string  // ["/data/", "/tmp/go-json/"]
       BlockedPaths    []string  // ["/etc/", "/root/"]
       MaxFileSize     int64     // bytes, default 10MB
       AllowWrite      bool      // default false
   }

   type SQLSecurityConfig struct {
       AllowedDrivers  []string  // ["sqlite3", "postgres"]
       MaxQueryTime    int       // seconds, default 30
       MaxRows         int       // default 10000
   }

   type ExecSecurityConfig struct {
       AllowedCommands []string  // ["pandoc", "wkhtmltopdf"]
       BlockedCommands []string  // ["rm", "shutdown", "dd"] — merged with DeniedCommands
       MaxTimeout      int       // seconds, default 60
       MaxOutputSize   int64     // bytes, default 1MB
   }
   ```
2. Implement `ValidateHTTPRequest(url)`, `ValidateFilePath(path, write bool)`, `ValidateCommand(cmd)`, `ValidateSQLDriver(driver)`
3. Two-layer security:
   - Layer 1: Program must `"import": {"http": "io:http"}` — if not imported, module not available
   - Layer 2: Runtime must enable module — `WithIO(http.New())` — if not enabled, import fails at compile time
4. Hardcoded deny lists (always blocked regardless of config):
   - `EngineSecrets` for env: `JWT_SECRET`, `DB_PASSWORD`, `ENCRYPTION_KEY`, `SMTP_PASSWORD`, `STORAGE_S3_SECRET_KEY`, `STORAGE_S3_ACCESS_KEY`
   - `DeniedCommands` for exec: `rm`, `rmdir`, `del`, `format`, `shutdown`, `reboot`, `halt`, `poweroff`, `dd`, `mkfs`, `fdisk`

**Edge cases:**
- Path traversal: `../../etc/passwd` → resolve to absolute, check prefix against allowed paths → blocked
- Host with port: `localhost:8080` → check hostname part against blocked hosts
- Wildcard matching: `*.internal.com` matches `api.internal.com`
- SQL injection in connection string → standalone mode validates DSN against AllowedDrivers; hosted mode DSN from host config only
- IPv4/IPv6 localhost variants: `127.0.0.1`, `::1`, `0.0.0.0` → all treated as `localhost`
- Cloud metadata endpoint: `169.254.169.254` → blocked by default in BlockedHosts

**Acceptance criteria:** Security checks prevent unauthorized access; two-layer gating works correctly

**Commit:** `feat(go-json): I/O security layer with two-layer gating`

---

## Batch 2: I/O Modules

### Task 3: HTTP Module

**Files:** Create `io/http.go`
**Effort:** Large
**Depends on:** Task 2

**Steps:**
1. Implement `http.get(url, headers?, timeout?, auth?)`, `http.post(url, body?, headers?, timeout?, auth?)`, `http.put(...)`, `http.patch(...)`, `http.delete(...)`
2. Auth parameter: `{"type": "bearer", "token": "..."}` or `{"type": "basic", "username": "...", "password": "..."}`
   - When both `auth` and `headers.Authorization` provided → `auth` takes precedence, emit compile warning
3. Response: `{"status": 200, "headers": {...}, "body": any}` — auto-parse JSON body
4. Use Go `net/http` with `context.Context` for cancellation/timeout
5. Security: validate URL against allowed/blocked hosts before request

**Edge cases:**
- Non-JSON response body → return as string
- Redirect following → follow up to 10 redirects
- HTTPS certificate errors → configurable via security config (strict by default)
- Request timeout → return error with timeout info
- Very large response → enforce `HTTPSecurityConfig.MaxResponseSize`
- Connection refused → clear error message with URL
- Invalid auth type → error: `"unsupported auth type 'xxx', expected 'bearer' or 'basic'"`
- Multipart form data → not supported in Phase 4.5c, return clear error
- Response streaming → not supported, return full body

**Acceptance criteria:** HTTP CRUD operations work with security checks and auth support

**Commit:** `feat(go-json): HTTP I/O module with security and auth`

---

### Task 4: FS Module

**Files:** Create `io/fs.go`
**Effort:** Medium
**Depends on:** Task 2

**Steps:**
1. Implement `fs.read(path, encoding?)`, `fs.write(path, content, encoding?)`, `fs.append(path, content, encoding?)`, `fs.exists(path)`, `fs.list(dir)`, `fs.mkdir(path)`, `fs.remove(path, recursive?)`
2. All paths validated against security config (allowed/blocked paths)
3. Path traversal prevention: resolve to absolute, check prefix
4. File size limits enforced on read and write
5. Encoding: UTF-8 default, optional encoding parameter. `encoding: "base64"` for binary files.

**Edge cases:**
- `fs.read` on directory → error: `"cannot read directory, use fs.list"`
- `fs.write` when `AllowWrite: false` → error: `"write operations disabled by security config"`
- `fs.write` with empty content → creates empty file (not error)
- `fs.append` to non-existent file → creates file (like `>>` in bash)
- `fs.remove` on non-empty directory without `recursive: true` → error: `"directory not empty, use recursive: true"`
- Symlink following → resolve and re-check against allowed paths
- Concurrent file access → no locking (OS handles)
- File not found → clear error with path

**Acceptance criteria:** File operations work within sandbox with encoding support

**Commit:** `feat(go-json): FS I/O module with sandboxing and encoding`

---

### Task 5: SQL Module

**Files:** Create `io/sql.go`
**Effort:** Large
**Depends on:** Task 2

**Steps:**
1. Implement dual-mode DSN:
   - **Standalone mode:** `sql.query(query, args?, dsn?)` — DSN from program, validated against `SQLSecurityConfig.AllowedDrivers`
   - **Hosted mode:** `sql.query(query, args?)` — connection from host via `WithSQLConnection(db *sql.DB)`. If `dsn` provided in hosted mode, silently ignored.
   - Runtime auto-detects mode based on whether `WithSQLConnection` was called.
2. Implement `sql.query(query, args?, dsn?)` → `{"rows": [...], "columns": [...], "count": N}`
3. Implement `sql.execute(query, args?, dsn?)` → `{"rows_affected": N, "last_insert_id": N}`
4. Implement `sql.begin()`, `sql.commit()`, `sql.rollback()` — scope-based transactions
   - Auto-rollback on error between begin and commit
   - Nested `begin` calls use savepoints
   - Transaction timeout follows `SQLSecurityConfig.MaxQueryTime`
5. Parameterized queries ONLY — no string interpolation
6. Result mapping: SQL types → go-json types (string, int, float, bool, nil)
7. Query timeout via context

**Edge cases:**
- SQL injection via params → impossible (parameterized queries)
- NULL values → nil in go-json
- Large result set → enforce `SQLSecurityConfig.MaxRows`
- Connection closed → clear error
- `sql.query` with 0 rows → `{"rows": [], "columns": ["name", "age"], "count": 0}` — columns still present
- `last_insert_id` on PostgreSQL → returns 0 (PostgreSQL uses `RETURNING`)
- `sql.commit` without `sql.begin` → error: `"no active transaction"`
- `sql.begin` when transaction already active → create savepoint
- Standalone mode with invalid DSN → clear error with driver name
- Multiple DSN in one program (standalone) → each query can target different DB

**Acceptance criteria:** SQL queries work with parameterized inputs, dual-mode DSN, and transactions

**Commit:** `feat(go-json): SQL I/O module with dual-mode DSN and transactions`

---

### Task 6: Exec Module

**Files:** Create `io/exec.go`
**Effort:** Medium
**Depends on:** Task 2

**Steps:**
1. Implement `exec.run(cmd, args?, cwd?, timeout?, env?)` → `{"exit_code": 0, "stdout": "...", "stderr": "..."}`
2. Command must be in whitelist — `ExecSecurityConfig.AllowedCommands`
3. Command must NOT be in `DeniedCommands` (hardcoded: rm, shutdown, dd, etc.) — checked BEFORE whitelist
4. Arguments as array, NOT string — no shell expansion
5. No pipe, no redirect, no shell metacharacters
6. Timeout per command via context
7. Capture stdout and stderr separately, truncate at `ExecSecurityConfig.MaxOutputSize` (default 1MB)
8. Environment variables:
   - If `env` provided → ONLY those env vars available (isolated execution)
   - If `env` omitted → inherit host environment MINUS `EngineSecrets`
   - `EngineSecrets` always stripped: `JWT_SECRET`, `DB_PASSWORD`, `ENCRYPTION_KEY`, `SMTP_PASSWORD`, `STORAGE_S3_SECRET_KEY`, `STORAGE_S3_ACCESS_KEY`

**Edge cases:**
- Command not in whitelist → error: `"command 'xxx' not in allowed list"`
- Command in DeniedCommands → error: `"command 'xxx' is permanently blocked"`
- Command not found on system → clear error
- Command hangs → timeout kills process
- Very large stdout → truncate at MaxOutputSize, add `"truncated": true` to result
- Non-zero exit code → not an error, returned in `exit_code`
- Arguments with spaces → handled correctly (array, not string)
- Empty `env` map → command runs with zero env vars (fully isolated)

**Acceptance criteria:** Whitelisted commands execute safely with env isolation

**Commit:** `feat(go-json): Exec I/O module with command whitelisting and env isolation`

---

## Batch 3: Regex Module (Stdlib)

### Task 7: Regex Module

**Files:** Create `stdlib/regex.go`, modify `stdlib/strings.go` (move `matches`)
**Effort:** Small
**Depends on:** Phase 4.5b stdlib

**Steps:**
1. Move existing `matches(str, pattern)` from `stdlib/strings.go` to `stdlib/regex.go`
2. Keep `matches()` as alias for backward compatibility
3. Implement `regex.match(str, pattern)` → bool (same as `matches`)
4. Implement `regex.findAll(str, pattern)` → []string
5. Implement `regex.replace(str, pattern, replacement)` → string
6. Compile regex with Go `regexp` package
7. Cache compiled regexes (map + sync.RWMutex)
8. ReDoS prevention:
   - Max pattern length: 1000 chars
   - Max input size: 1MB
   - Compile timeout: 100ms (Go's regexp is RE2-based, inherently safe from catastrophic backtracking, but limit pattern complexity anyway)

**Edge cases:**
- Invalid regex pattern → clear error with pattern: `"regex: invalid pattern '(': missing closing paren"`
- Empty string → valid input, returns appropriate result
- No matches → `regex.findAll` returns empty array, `regex.replace` returns original string
- `matches()` alias → delegates to `regex.match`, no duplication

**Acceptance criteria:** Regex operations work with caching, safety limits, and backward-compatible `matches()` alias

**Commit:** `feat(go-json): regex stdlib module — match, findAll, replace with caching`

---

## Batch 4: Extension & Bridge Integration

### Task 8: Extension API

**Files:** Modify `runtime/runtime.go`, create `runtime/extension.go`
**Effort:** Medium
**Depends on:** Phase 4.5b runtime

**Steps:**
1. Define `Extension` struct:
   ```go
   type Extension struct {
       Name      string
       Functions map[string]any  // function map for expr-lang
       Structs   map[string]any  // reserved for future struct injection
       Constants map[string]any  // reserved for future constant injection
   }
   ```
2. Implement `WithExtension(name string, ext Extension)` runtime option
3. Extensions imported in programs via `"ext:name"`
4. Extension functions registered in expr-lang environment under namespace
5. `Structs` and `Constants` fields: validate they are nil or empty for now, log warning if populated (future-proofing without implementation)

**Acceptance criteria:** Extensions injectable and usable in programs; `Structs`/`Constants` fields accepted without error

**Commit:** `feat(go-json): extension API with forward-compatible struct`

---

### Task 9: Bitcode Bridge Adapter

**Files:** Create `bridge/gojson_adapter.go` (in bitcode engine codebase: `engine/internal/runtime/bridge/`)
**Effort:** Large
**Depends on:** Task 8

**Steps:**
1. Create adapter that wraps bitcode's existing `bridge.Context` as go-json `Extension`
2. Map ALL 20 bridge namespaces (50+ functions total):

   **Model CRUD (via proxy object):**
   - `model(name)` → returns proxy with: `search`, `get`, `create`, `write`, `delete`, `count`, `sum`, `upsert`
   - Bulk: `createMany`, `writeMany`, `deleteMany`, `upsertMany`
   - Relations: `addRelation`, `removeRelation`, `setRelation`, `loadRelation`
   - Sudo: `sudo()` → returns `SudoModelHandle`

   **Database:**
   - `db.query(sql, args...)` → `[]map[string]any`
   - `db.execute(sql, args...)` → `{rows_affected}`

   **HTTP:**
   - `http.get(url, opts?)`, `http.post(url, opts?)`, `http.put(url, opts?)`, `http.patch(url, opts?)`, `http.delete(url, opts?)`

   **Cache:**
   - `cache.get(key)`, `cache.set(key, value, opts?)`, `cache.del(key)`

   **File system:**
   - `fs.read(path)`, `fs.write(path, content)`, `fs.exists(path)`, `fs.list(path)`, `fs.mkdir(path)`, `fs.remove(path)`

   **Environment & config:**
   - `env(key)`, `config(key)`, `session()` → returns session object

   **Logging & events:**
   - `log(level, msg, data...)`, `emit(event, data)`, `call(process, input)`

   **Command execution:**
   - `exec(cmd, args, opts?)`

   **Email & notifications:**
   - `email.send(opts)`, `notify.send(opts)`, `notify.broadcast(channel, data)`

   **Storage:**
   - `storage.upload(opts)`, `storage.url(id)`, `storage.download(id)`, `storage.delete(id)`

   **i18n:**
   - `t(key)` → translated string

   **Security:**
   - `security.permissions(model)`, `security.hasGroup(group)`, `security.groups()`

   **Audit:**
   - `audit.log(opts)`

   **Crypto:**
   - `crypto.encrypt(plaintext)`, `crypto.decrypt(ciphertext)`, `crypto.hash(value)`, `crypto.verify(value, hash)`

   **Execution log:**
   - `execution.current()`, `execution.search(opts)`, `execution.get(id)`, `execution.retry(id)`, `execution.cancel(id)`

   **Transaction:**
   - `tx(fn)` → runs fn in database transaction

3. Map bitcode types to go-json types:
   - Go structs → `map[string]any`
   - `bridge.Session` → map with all 7 fields: `userId`, `username`, `email`, `tenantId`, `groups`, `locale`, `context`
   - `bridge.ExecResult` → `{"exit_code": N, "stdout": "...", "stderr": "..."}`
   - `bridge.HTTPResponse` → `{"status": N, "headers": {...}, "body": any}`
   - `bridge.BulkResult` → `{"affected": N}`
   - `bridge.Attachment` → `{"id": "...", "url": "...", "filename": "...", "size": N, "contentType": "..."}`
4. Handle async operations (bitcode.call) → synchronous in go-json (blocking)

**Edge cases:**
- Bitcode bridge returns Go structs → convert to `map[string]any` via JSON marshal/unmarshal or reflection
- Bitcode bridge error → wrap as go-json error with original message preserved
- Bitcode session → inject as go-json session (all 7 fields)
- `model().sudo()` → must verify program has sudo permission before allowing
- `execution.retry(id)` where id = current execution → error to prevent infinite loop
- `tx(fn)` with nested `tx` call → use savepoint (delegate to bridge.Context.Tx)

**Acceptance criteria:** go-json programs can use ALL bitcode bridge functions via `"ext:bitcode"`

**Commit:** `feat(go-json): bitcode bridge adapter — full 50+ function mapping`

---

### Task 10: scripts/*.json Support

**Files:** Modify `engine/internal/runtime/executor/steps/script.go`
**Effort:** Medium
**Depends on:** Task 9

**Steps:**
1. Add `.json` case to existing `detectRuntimeFromExtension` function:
   ```go
   case strings.HasSuffix(script, ".json"): return "go-json"
   ```
2. Refactor existing cases from manual slice (`script[len(script)-3:]`) to `strings.HasSuffix` for consistency and safety
3. Implement `GoJSONRunner` that satisfies `EmbeddedRunner` interface:
   - `CanHandle("go-json")` → true
   - `Run(ctx, script, params)`:
     a. Load and compile go-json program (with caching — use `runtime.Runtime.cache`)
     b. Inject bitcode bridge as extension (from Task 9)
     c. Map process step input → go-json input
     d. Execute program
     e. Map go-json return → process step output
4. Register `GoJSONRunner` as `EmbeddedRunner` in script handler

**Edge cases:**
- `.json` file that is NOT a go-json program (e.g., config file) → compile error with clear message
- Circular script calls (A.json calls B.json calls A.json) → caught by existing call depth limit
- Large `.json` program → compile once, cache compiled program

**Acceptance criteria:** go-json programs executable as bitcode process steps via `"runtime": "go-json"` or auto-detected from `.json` extension

**Commit:** `feat(go-json): scripts/*.json support in bitcode process engine`

---

### Task 11: Backward Compatibility Layer

**Files:** Modify `engine/internal/runtime/executor/executor.go`
**Effort:** Medium
**Depends on:** Task 10

**Steps:**
1. Old process format (`type: "query"`, `type: "create"`, etc.) continues to work without any change — this is the DEFAULT behavior
2. New processes can opt-in to go-json runtime via `"runtime": "go-json"` in process definition
3. Without `"runtime"` field → old executor (no change to existing behavior)
4. With `"runtime": "go-json"` → route ALL steps through go-json VM with bitcode bridge
5. Document: old format enters maintenance mode — new features only added to go-json format

**Edge cases:**
- Process with mixed step types (some old, some new) → not supported. Entire process is either old or go-json.
- Process references old step type names in go-json mode → compile error with migration hint
- Existing tests for old executor → must continue to pass unchanged

**Acceptance criteria:** Existing processes work unchanged; new processes can opt-in to go-json

**Commit:** `feat(go-json): backward compatibility layer for process engine migration`

---

### Task 12: Process Engine Data Step Replacement

**Files:** Modify `engine/internal/runtime/executor/steps/data.go`, create `engine/internal/runtime/executor/steps/gojson_data.go`
**Effort:** Large
**Depends on:** Task 11

**Steps:**
1. For processes with `runtime: "go-json"`, implement adapter that converts old data step definitions to go-json bridge calls:
   - `type: "query"` → `bc.model(name).search(opts)` with domain conversion
   - `type: "create"` → `bc.model(name).create(data)`
   - `type: "update"` → `bc.model(name).write(id, data)`
   - `type: "delete"` → `bc.model(name).delete(id)`
   - `type: "upsert"` → `bc.model(name).upsert(data, unique)`
   - `type: "count"` → `bc.model(name).count(opts)`
   - `type: "sum"` → `bc.model(name).sum(field, opts)`
2. Map old executor context variables to go-json scope variables
3. Map go-json return values back to old executor context format
4. Domain conversion: old format `[["field", "op", "value"]]` → go-json expression format

**Edge cases:**
- Old domain format with complex nested AND/OR → must be faithfully converted
- Old step `into` variable → map to go-json `let` variable name
- Old step `on_error` → map to go-json `try/catch`
- Result shape differences → adapter must normalize

**Acceptance criteria:** go-json runtime produces identical results to old executor for all data operations

**Commit:** `feat(go-json): process engine data step replacement via bridge`

---

## Batch 5: CLI

### Task 13: CLI Scaffold

**Files:** Create `cmd/go-json/main.go`
**Effort:** Small
**Depends on:** Phase 4.5b runtime

**Steps:**
1. Use Go stdlib `flag` package (no external dependency)
2. Subcommand dispatch: run, check, test, ast, codegen, migrate
3. Common flags: `--timeout`, `--trace`, `--verbose`
4. Version flag: `--version`
5. Help text with usage examples for each subcommand

**Commit:** `feat(go-json): CLI scaffold with subcommand dispatch`

---

### Task 14: `go-json run`

**Files:** Modify `cmd/go-json/main.go`
**Effort:** Medium
**Depends on:** Task 13

**Steps:**
1. Read program file
2. Input sources (mutually exclusive):
   - `--input '{"key": "value"}'` — inline JSON
   - `--input-file input.json` — read from file
   - stdin — if neither flag provided and stdin is not a terminal
3. Compile and execute
4. Output result as JSON to stdout
5. Exit code 0 on success, 1 on error
6. Flags:
   - `--timeout 60s` — execution timeout
   - `--max-depth 500` — override default call depth limit
   - `--io http,fs,sql,exec` — enable specific I/O modules (default: none)
   - `--io all` — enable all I/O modules
   - `--trace` — print execution trace after result

**Edge cases:**
- Both `--input` and `--input-file` provided → error
- Invalid JSON input → clear error with parse position
- Program file not found → clear error with path
- Timeout reached → exit code 1 with timeout error

**Commit:** `feat(go-json): CLI run command with input and I/O flags`

---

### Task 15: `go-json check`

**Files:** Modify `cmd/go-json/main.go`
**Effort:** Small
**Depends on:** Task 13

**Steps:**
1. Parse and compile program without executing
2. Report all compile errors with line/position info
3. Exit code 0 if valid, 1 if errors
4. `--verbose` flag → show program metadata (name, functions, imports)

**Commit:** `feat(go-json): CLI check command`

---

### Task 16: `go-json test`

**Files:** Create `cmd/go-json/test_runner.go`
**Effort:** Medium
**Depends on:** Task 13

**Steps:**
1. Find test files: files with `"test": true` in specified directory (recursive)
2. Parse test format:
   ```json
   {
     "name": "test_discount",
     "test": true,
     "import": {"calc": "../functions/discount.json"},
     "cases": [
       {
         "_c": "Gold tier gets 15% discount",
         "call": "calc.calculateDiscount",
         "with": {"price": "100.0", "quantity": "5", "tier": "'gold'"},
         "expect": 75.0
       }
     ]
   }
   ```
3. Execute each test case: call function with args, compare result to `expect`
4. Report: ✓/✗ per case, timing, summary (passed/failed/total)
5. Exit code 0 if all pass, 1 if any fail
6. `--verbose` flag → show input/output for each case
7. `--filter` flag → run only cases matching pattern

**Edge cases:**
- `expect` is object → deep equality comparison
- `expect` is array → deep equality with order
- `expect` is nil → result must be nil
- Float comparison → tolerance (1e-9)
- Test case throws error → fail with error message
- No test files found → warning, exit 0
- Test file with 0 cases → warning, skip

**Acceptance criteria:** Test runner finds, executes, and reports test results with float tolerance

**Commit:** `feat(go-json): CLI test command with built-in test runner`

---

### Task 17: `go-json ast`

**Files:** Modify `cmd/go-json/main.go`
**Effort:** Small
**Depends on:** Task 13

**Steps:**
1. Parse program, output AST as formatted JSON
2. Include `_c` metadata in output
3. Flags:
   - `--output ast.json` — write to file (default: stdout)
   - `--format json` — output format (only json for now, yaml reserved for future)

**Commit:** `feat(go-json): CLI ast command`

---

### Task 18: `go-json codegen`

**Files:** Modify `cmd/go-json/main.go`
**Effort:** Small
**Depends on:** Task 13, Task 20 (codegen framework)

**Steps:**
1. Parse program, generate code via CodeGenerator
2. Flags:
   - `--target go|js|python` — required, target language
   - `--output program.go` — write to file (default: stdout)
   - `--package main` — Go package name (only for `--target go`, default: `main`)
3. Exit code 0 on success, 1 on error
4. If program uses extensions (`ext:bitcode`) → warning: generated code will not include extension calls

**Commit:** `feat(go-json): CLI codegen command`

---

### Task 19: `go-json migrate`

**Files:** Create `cmd/go-json/migrate.go`
**Effort:** Medium
**Depends on:** Task 13

**Steps:**
1. Read program file
2. Flags:
   - `--from v1` — source version (optional, auto-detect if omitted)
   - `--to v2` — target version (default: latest)
   - `--output migrated.json` — write to file (default: stdout)
   - `--dry-run` — show changes without applying
3. Apply version-specific transforms:
   - Rename deprecated functions (e.g., `unique` → `uniq`, `startsWith` → `hasPrefix`)
   - Update syntax patterns
4. Preserve JSONC comments during migration
5. Report changes made

**Edge cases:**
- Program already up-to-date → no changes, report "already current"
- Unknown version → error with list of known versions
- Transform changes expression strings → must be careful not to break string content (only transform function names, not string literals)
- JSONC comments → preserved in output

**Acceptance criteria:** Migration transforms deprecated syntax correctly, preserves comments

**Commit:** `feat(go-json): CLI migrate command with version detection`

---

## Batch 6: Code Generation

### Task 20: Code Generation Framework

**Files:** Create `codegen/codegen.go`
**Effort:** Small
**Depends on:** Phase 4.5b

**Steps:**
1. Define `CodeGenerator` interface:
   ```go
   type CodeGenerator interface {
       Generate(program *lang.CompiledProgram) (string, error)
       Language() string
   }
   ```
2. Must handle all 21 AST node types defined in design doc §4.3:
   `program`, `let`, `set`, `if`, `switch`, `for_in`, `for_range`, `while`, `break`, `continue`, `return`, `call`, `try`, `error`, `log`, `new`, `parallel`, `function`, `struct`, `import`, `expression`
3. Handle `_c` metadata → emit as comments in target language
4. Common utilities: indent management, string escaping, expression transformation

**Commit:** `feat(go-json): code generation framework`

---

### Task 21: Go Code Generator

**Files:** Create `codegen/golang.go`
**Effort:** Large
**Depends on:** Task 20

**Steps:**
1. Map go-json types → Go types (string→string, int→int64, float→float64, bool→bool, any→any)
2. Map steps → Go statements:
   - `let` → `var` / `:=`
   - `set` → assignment
   - `if/elif/else` → `if/else if/else`
   - `switch` → `switch`
   - `for_in` → `for _, v := range`
   - `for_range` → `for i := start; i < end; i++`
   - `while` → `for condition`
   - `call` → function call
   - `try/catch` → `if err != nil` pattern
   - `parallel` → goroutines with `sync.WaitGroup`
   - `struct` → Go struct type
   - `function` → Go function
3. Map expressions → Go expressions (most expr-lang syntax is Go-compatible)
4. Emit `_c` as `//` comments
5. Generate `package main` with `func main()`

**Edge cases:**
- expr-lang syntax not valid Go (e.g., `filter(items, .price > 100)`) → transform to Go equivalent (loop + append)
- `with` computed objects → Go struct literals or map literals
- `try/catch` → Go error handling pattern (`if err != nil`)
- `parallel` → Go goroutines with `sync.WaitGroup`
- Extension calls (`ext:bitcode`) → emit as comment: `// TODO: replace with native implementation`

**Acceptance criteria:** Generated Go code compiles and produces same result for pure-logic programs

**Commit:** `feat(go-json): Go code generator`

---

### Task 22: JavaScript Code Generator

**Files:** Create `codegen/javascript.go`
**Effort:** Large
**Depends on:** Task 20

**Steps:**
1. Map go-json types → JS types (string→string, int→number, float→number, bool→boolean, any→any)
2. Key differences from Go generator:
   - `struct` → `class`
   - `parallel` → `Promise.all`
   - `try/catch` → native try/catch (simpler than Go)
   - `for_range` → `for (let i = start; i < end; i++)`
   - `let` → `const` (or `let` if reassigned via `set`)
   - `function` → `function` or arrow function
3. Emit `_c` as `//` comments
4. Generate as ES module or CommonJS (configurable)

**Acceptance criteria:** Generated JavaScript runs in Node.js and produces same result

**Commit:** `feat(go-json): JavaScript code generator`

---

### Task 23: Python Code Generator

**Files:** Create `codegen/python.go`
**Effort:** Large
**Depends on:** Task 20

**Steps:**
1. Map go-json types → Python types with type hints (string→str, int→int, float→float, bool→bool, any→Any)
2. Key differences from Go generator:
   - `struct` → `@dataclass`
   - `parallel` → `asyncio.gather`
   - `try/catch` → `try/except`
   - `for_range` → `for i in range(start, end)`
   - Indentation-based blocks (no braces)
   - `function` → `def` with type annotations
3. Emit `_c` as `#` comments
4. Generate with `if __name__ == "__main__":` guard

**Acceptance criteria:** Generated Python runs and produces same result

**Commit:** `feat(go-json): Python code generator`

---

## Batch 7: Tests

### Task 24: I/O Module Tests

**Files:** Create `io/http_test.go`, `io/fs_test.go`, `io/sql_test.go`, `io/exec_test.go`
**Effort:** Medium
**Depends on:** Tasks 3-6

**Key tests:**
- HTTP: mock server, GET/POST/PUT/PATCH/DELETE, auth (bearer + basic), timeout, redirect, large response
- FS: temp dir, read/write/append/exists/list/mkdir/remove, encoding, binary (base64), empty content
- SQL: SQLite in-memory, query/execute, transactions (begin/commit/rollback, nested savepoint), dual-mode DSN, NULL handling
- Exec: echo command, timeout, env isolation, EngineSecrets stripping, DeniedCommands blocking

**Commit:** `test(go-json): I/O module tests`

---

### Task 25: I/O Security Tests

**Files:** Create `io/security_test.go`
**Effort:** Medium
**Depends on:** Task 2

**Key tests:**
- `fs.read("../../etc/passwd")` → blocked (path traversal)
- `exec.run("rm", ["-rf", "/"])` → blocked (DeniedCommands)
- `http.get("http://169.254.169.254/metadata")` → blocked (cloud metadata)
- `http.get("http://localhost:8080")` → blocked (default BlockedHosts)
- `sql.query("DROP TABLE users")` → allowed only if execute permission granted
- Import `"io:http"` without runtime enabling → compile error
- Enabled but not imported → symbol not found
- Wildcard host matching: `*.internal.com` matches `api.internal.com`
- IPv6 localhost `::1` → treated as localhost → blocked
- Env inheritance strips EngineSecrets

**Commit:** `test(go-json): I/O security tests`

---

### Task 26: Regex Tests

**Files:** Create `stdlib/regex_test.go`
**Effort:** Small
**Depends on:** Task 7

**Key tests:**
- `regex.match` — valid match, no match, invalid pattern
- `regex.findAll` — multiple matches, no matches, empty string
- `regex.replace` — replacement, no match (returns original), empty replacement
- `matches()` alias → same result as `regex.match`
- Regex caching — same pattern compiled once
- ReDoS prevention — pattern length limit, input size limit
- Edge: empty pattern, empty string, special regex chars

**Commit:** `test(go-json): regex stdlib tests`

---

### Task 27: Extension API Tests

**Files:** Create `runtime/extension_test.go`
**Effort:** Medium
**Depends on:** Task 8

**Key tests:**
- Register extension, import via `"ext:name"`, call functions
- Extension with `Structs: nil` and `Constants: nil` → accepted without error
- Multiple extensions → each accessible via own namespace
- Extension function error → propagated as go-json error
- Extension function panic → caught and wrapped as error
- Name collision between extension function and stdlib → error at registration

**Commit:** `test(go-json): extension API tests`

---

### Task 28: Bitcode Bridge Integration Tests

**Files:** Create `engine/internal/runtime/bridge/gojson_adapter_test.go`
**Effort:** Large
**Depends on:** Task 9

**Key tests:**
- Model CRUD: search, get, create, write, delete via bridge
- Model bulk: createMany, writeMany, deleteMany
- Model relations: addRelation, removeRelation, setRelation, loadRelation
- Model sudo: sudo().create bypasses permissions
- DB: query returns rows, execute returns affected
- HTTP: get/post through bridge
- Cache: get/set/del
- FS: read/write/exists/list/mkdir/remove through bridge
- Session: returns all 7 fields
- Crypto: encrypt/decrypt roundtrip, hash/verify roundtrip
- Execution: current() returns execution info, retry/cancel
- Tx: transaction commit and rollback
- Type conversion: Go structs → map[string]any

**Commit:** `test(go-json): bitcode bridge integration tests`

---

### Task 29: Script Detection & Backward Compat Tests

**Files:** Modify `engine/internal/runtime/executor/steps/steps_test.go`
**Effort:** Small
**Depends on:** Tasks 10-11

**Key tests:**
- `detectRuntimeFromExtension("script.json")` → `"go-json"`
- `detectRuntimeFromExtension("script.js")` → `"javascript"` (unchanged)
- `.json` file routed to GoJSONRunner
- Old process format (without `runtime` field) → old executor (unchanged)
- New process with `runtime: "go-json"` → go-json VM
- Old process tests → all still pass

**Commit:** `test(go-json): script detection and backward compatibility tests`

---

### Task 30: AST Export Tests

**Files:** Create `lang/ast_export_test.go`
**Effort:** Medium
**Depends on:** Task 20

**Key tests:**
- Parse → AST → JSON → parse back → compare (roundtrip)
- All 21 node types present in AST output
- `_c` metadata preserved in AST
- Complex program with nested functions, structs, imports → valid AST

**Commit:** `test(go-json): AST export roundtrip tests`

---

### Task 31: Code Generation Tests

**Files:** Create `codegen/golang_test.go`, `codegen/javascript_test.go`, `codegen/python_test.go`
**Effort:** Large
**Depends on:** Tasks 21-23

**Key tests:**
- Factorial program → generates valid Go/JS/Python
- All step types → generates correct construct per language
- `_c` comments → preserved as language-appropriate comments
- Extension calls → emitted as TODO comments
- Generated Go code → `go build` succeeds
- Generated JS code → `node --check` succeeds
- Generated Python code → `python -m py_compile` succeeds

**Commit:** `test(go-json): code generation tests for Go/JS/Python`

---

### Task 32: CLI Tests

**Files:** Create `cmd/go-json/cli_test.go`
**Effort:** Medium
**Depends on:** Tasks 14-19

**Key tests:**
- `go-json run` — with --input, --input-file, --timeout, --max-depth, --io, --trace
- `go-json check` — valid program (exit 0), invalid program (exit 1)
- `go-json test` — all pass (exit 0), some fail (exit 1), no tests found (exit 0 + warning)
- `go-json ast` — output to stdout, output to file (--output)
- `go-json codegen` — all 3 targets, --output flag, --package flag
- `go-json migrate` — transform deprecated syntax, --dry-run, --from/--to, already current
- `go-json --version` — prints version
- Unknown subcommand → error with help text

**Commit:** `test(go-json): CLI command tests`

---

### Task 33: Test Runner Tests

**Files:** Create `cmd/go-json/test_runner_test.go`
**Effort:** Medium
**Depends on:** Task 16

**Key tests:**
- Test file format parsing — `"test": true`, `"cases"` array
- Pass/fail reporting — ✓/✗ per case
- Float tolerance — `expect: 0.1 + 0.2` vs `0.3` → pass (within 1e-9)
- Object deep equality — nested maps
- Array deep equality — order matters
- Nil comparison — `expect: null`
- Error case — test case throws → fail with error message
- Empty cases array → warning, skip
- Import resolution — `"import": {"calc": "../functions/calc.json"}`
- Timing — each case reports duration

**Commit:** `test(go-json): test runner tests with float tolerance`

---

## Summary

| Task | Description | Effort | Batch |
|------|-------------|--------|-------|
| 1 | I/O module interface | Small | 1 |
| 2 | I/O security layer | Medium | 1 |
| 3 | HTTP module | Large | 2 |
| 4 | FS module | Medium | 2 |
| 5 | SQL module | Large | 2 |
| 6 | Exec module | Medium | 2 |
| 7 | Regex module (stdlib) | Small | 3 |
| 8 | Extension API | Medium | 4 |
| 9 | Bitcode bridge adapter | Large | 4 |
| 10 | scripts/*.json support | Medium | 4 |
| 11 | Backward compatibility layer | Medium | 4 |
| 12 | Data step replacement | Large | 4 |
| 13 | CLI scaffold | Small | 5 |
| 14 | CLI run | Medium | 5 |
| 15 | CLI check | Small | 5 |
| 16 | CLI test | Medium | 5 |
| 17 | CLI ast | Small | 5 |
| 18 | CLI codegen | Small | 5 |
| 19 | CLI migrate | Medium | 5 |
| 20 | Codegen framework | Small | 6 |
| 21 | Go code generator | Large | 6 |
| 22 | JS code generator | Large | 6 |
| 23 | Python code generator | Large | 6 |
| 24 | Tests: I/O modules | Medium | 7 |
| 25 | Tests: I/O security | Medium | 7 |
| 26 | Tests: Regex | Small | 7 |
| 27 | Tests: Extension API | Medium | 7 |
| 28 | Tests: Bitcode bridge | Large | 7 |
| 29 | Tests: Script detection + compat | Small | 7 |
| 30 | Tests: AST export | Medium | 7 |
| 31 | Tests: Code generation | Large | 7 |
| 32 | Tests: CLI commands | Medium | 7 |
| 33 | Tests: Test runner | Medium | 7 |

**Total: 33 tasks across 7 batches**
**Critical path: Task 1 → 2 → 3 → 8 → 9 → 10 → 11 → 12 → tests**
**Estimated total effort: ~4-5 weeks for one developer**

**Security-critical tasks:** Task 2, 3, 4, 5, 6 — every I/O operation must be validated against security config.
**Bridge-critical tasks:** Task 9, 10, 11, 12 — bitcode integration must map all 50+ bridge functions and maintain backward compatibility.
