# AGENTS.md — go-json

## Overview

Standalone JSON/JSONC programming language engine. Embeddable in Go applications. Part of the BitCode platform but independently usable.

**Pipeline:** JSONC pre-process → JSON parse → import resolution → AST → compile (struct registration, structural validation, limit resolution) → immutable Program → VM execution with debug hooks.

**Expression evaluation** delegated to [expr-lang/expr](https://github.com/expr-lang/expr) via `ExprEngine` abstraction layer. The VM never calls expr-lang directly.

**Phase 4.5a design:** `docs/plans/2026-07-14-runtime-engine-phase-4.5a-go-json-core-language.md`
**Phase 4.5b design:** `docs/plans/2026-07-14-runtime-engine-phase-4.5b-go-json-modularity.md`
**Phase 4.5b plan:** `docs/plans/2026-07-14-runtime-engine-phase-4.5b-go-json-modularity-plan.md`
**Decisions:** `docs/plans/2026-04-28-go-json-brainstorming-design.md`

## Package Structure

```
packages/go-json/
├── lang/           Core language engine (AST, parser, compiler, VM, scope, types, errors, expr engine, debugger, import resolver)
├── stdlib/         Layer 2 stdlib (42+ functions + crypto namespace + regex + path + JSON). Layer 1 = expr-lang built-ins (~68 functions)
├── runtime/        Runtime API: NewRuntime(), Execute(), ExecuteFunction(), CompileFile(), Close(), program cache, limits, logger, session, extensions. Also: EvalExpr(), EvalExprBool(), EvalExprFloat(), ParseExpr(), ValidateExpr() — lightweight expression evaluation API for engine consumers (shared singleton ExprLangEngine with stdlib + compilation cache).
├── io/             I/O modules: HTTP, FS, SQL, Exec, MongoDB, Redis with security layer + unified SQL param translation (Phase 4.5c-d)
├── codegen/        Code generation: Go/JS/Python generators + server codegen (Fiber/net-http/Express/FastAPI) + dependency management (Phase 4.5c-d)
├── server/         Web server execution mode: adapters (Fiber/net-http/Echo/Gin/Chi), middleware, JWT, auth, templates, static, OpenAPI (Phase 4.5d)
├── server/adapters/ Framework adapter implementations (ServerAdapter interface)
├── generate/       CRUD generator, auth scaffold, project scaffold, pattern templates, DB introspection (Phase 4.5d)
├── cmd/go-json/    CLI: run, serve, check, test, ast, codegen, generate, openapi, migrate commands
└── testdata/       Test fixture programs (.json, .jsonc)
```

## Key Architecture Decisions

1. **ExprEngine abstraction** — VM never calls expr-lang directly. All expression work goes through `ExprEngine` interface for testability and swappability.
2. **Compile-once, run-many** — `CompiledProgram` is immutable after compilation. Each execution gets fresh VM + scope. Multiple goroutines can run the same program concurrently.
3. **Structural validation at compile time, expression validation at runtime** — expr-lang's compile-time type checking requires a fully-typed environment which we don't have with gradual typing. Runtime catches expression errors.
4. **Scope isolation for functions and methods** — `IsolatedChild()` creates scope WITHOUT parent link. Functions and methods cannot access caller variables. `self` is injected into method scope. Block scope (`NewChild()`) for if/for/while allows reading and mutating outer variables.
5. **Sentinel types for control flow** — `returnValue`, `breakSignal`, `continueSignal` are unexported struct types that propagate through `executeSteps()` return values.
6. **Resource limits at every step** — step count, call depth, loop iterations, timeout checked before each step execution. MaxVariables, MaxVariableSize checked after every `Declare()`. MaxOutputSize checked on program return.
7. **JSON param ordering** — Go maps don't preserve insertion order. `extractOrderedKeys()` uses `json.Decoder` tokenization to recover function param order from raw JSON.
8. **Built-in name protection** — `let` blocks variable names that shadow critical built-in functions (len, abs, min, max, etc.). Curated list excludes common-word functions (count, filter, sort) that are also natural variable names.
9. **Implicit scope variables** — `session.*` (user_id, locale, tenant_id, groups) and `execution.*` (id, program, started_at, depth, step_count) injected automatically into every execution.
10. **Trace enrichment** — `TraceEntry` captures Var/Value for let/set, Condition/Result for if/while/switch, per step type via `enrichTraceEntry()`.

## Step Types (16)

`let`, `set`, `if`/`elif`/`else`, `switch`, `for` (each + range), `while`, `break`, `continue`, `return`, `call`, `try`/`catch`/`finally`, `error`, `log`, `parallel`, `_c` (comment)

## Struct System (Phase 4.5b)

- Structs defined in `structs` block with fields, methods, optional `frozen: true`
- Construction via `{"let": "x", "new": "StructName", "with": {...}}`
- Nested construction: `"field": {"new": "Other", "with": {...}}`
- Runtime type-checking on construction: field values validated against declared types (TYPE_MISMATCH error)
- Methods with implicit `self` binding, callable at expression and step level
- Frozen structs: compile-time rejection of `set "self.*"` in methods
- `self` reassignment blocked in all node types (if/for/while/try/switch/parallel)
- Forward references resolved via two-pass compilation
- Circular non-nullable struct references detected at compile time

## Import System (Phase 4.5b)

- Import key: `"import"` (preferred) or `"imports"` (compat)
- Path types: relative (`./`), stdlib (`stdlib:`), extension (`ext:`), I/O (`io:`)
- Imported items namespaced via alias: `alias.StructName`, `alias.functionName`
- Circular import detection via import stack
- Import alias collision detection (compile error on duplicate namespaced names)
- Barrel file re-export via `{"alias": "imported.Type"}` in structs block
- Diamond imports handled correctly (loaded once, cached)
- Wired via `Runtime.CompileFile(path)` — import resolution between parse and compile

## Parallel Execution (Phase 4.5b)

- `{"parallel": {"branch1": [...], "branch2": [...]}, "into": "results"}`
- Each branch gets own VM + scope (read parent, cannot write parent)
- Compile-time check: `set` targeting parent variable in parallel branch = error
- Join modes: `all` (default, wait for all), `any` (first success wins), `settled` (wait for all regardless of errors, errors collected as error objects)
- Error modes: `cancel_all` (default), `continue`, `collect`
- Goroutine leak prevention: drain channel after cancel
- Shared step counter across branches via `sync/atomic` — prevents step limit bypass via many parallel branches
- Struct instances are NOT thread-safe — do not share mutable struct instances across parallel branches

## Stdlib Layers

| Layer | Contents | Ownership |
|-------|----------|-----------|
| Layer 1 | expr-lang built-ins (~68 functions: abs, ceil, floor, round, min, max, len, upper, lower, trim, split, filter, map, reduce, find, sort, int, float, string, type, etc.) | expr-lang — DO NOT reimplement |
| Layer 2 | go-json additions (42+ functions + crypto namespace). Phase 4.5a: clamp, sign, randomInt, randomFloat, pow, sqrt, mod, padLeft, padRight, substring, format, matches, append, prepend, slice, chunk, zip, bool, isNil. Phase 4.5b: has, get, merge, pick, omit, formatDate (universal format), addDuration, diffDates, urlEncode, urlDecode, sprintf, crypto.sha256, crypto.md5, crypto.uuid, crypto.hmac. Phase 4.5d: toJSON, fromJSON, basename, dirname, extname, joinpath, cleanpath, isabs, stemname, pathsep | `stdlib/` package |
| Layer 3 | I/O modules (HTTP, FS, SQL, Exec, MongoDB, Redis) with two-layer security gating. Regex stdlib (match, findAll, replace with LRU caching). FS enhancements: stat, copy, move, glob. SQL unified query parameter translation (?/:name across drivers). | `io/` and `stdlib/regex.go` |

## Conventions

- Follow root `AGENTS.md` conventions (no unnecessary comments, tests required)
- All exported types and functions need Go doc comments
- `go build ./...` and `go vet ./...` must pass
- Tests: `go test ./... -v`

## Testing

```bash
cd packages/go-json
go test ./... -v          # All tests (579)
go test ./lang/ -v        # Language engine tests
go test ./lang/ -run TestStruct -v       # Struct tests
go test ./lang/ -run TestMethod -v       # Method tests
go test ./lang/ -run TestParallel -v     # Parallel tests
go test ./lang/ -run TestImport -v       # Import tests
go test ./lang/ -run TestIntegration -v  # Integration tests
go test ./stdlib/ -v                     # Stdlib tests (including regex, path, JSON)
go test ./io/ -v                         # I/O + SQL param translation tests
go test ./runtime/ -v                    # Runtime + extension tests
go test ./codegen/ -v                    # Code generation + server codegen tests
go test ./server/ -v                     # Server config, routing, handler, auth, JWT, middleware, OpenAPI, template, static tests
go test ./generate/ -v                   # CRUD generator, auth scaffold, project scaffold, introspection, pattern tests
```

## What's Done (Phase 4.5b)

- Struct definitions with fields, defaults, frozen, methods
- Struct construction (`new` + `with`), nested construction, return with new
- Method system with `self` binding, mutation, frozen compile-time check
- Import system with relative file resolution, circular detection, barrel files
- Parallel execution with 3 error modes, scope isolation, compile-time parent write check
- Stdlib Layer 2 extensions: maps, datetime, encoding, crypto (namespaced), format
- Nullable type support (`?T`), optional chaining via expr-lang

## What's Done (Phase 4.5c)

- I/O module framework: `IOModule` interface, `IORegistry`, `WithIO()`/`WithoutIO()` runtime options
- I/O security layer: `SecurityConfig`, two-layer gating (import + runtime enable), hardcoded deny lists, path traversal prevention, cloud metadata blocking
- HTTP module: GET/POST/PUT/PATCH/DELETE with auth (bearer/basic), redirect following, response size limits, client reuse
- FS module: read/write/append/exists/list/mkdir/remove with sandboxing, encoding (UTF-8/base64), symlink resolution
- SQL module: query/execute with connection pooling per-DSN, DefaultDSN, multi-driver detection (postgres/mysql/sqlserver/oracle/sqlite), transactions with savepoints, DDL protection (blocked keywords + max query length), auto-rollback on Close/Cleanup
- Exec module: command whitelisting, DeniedCommands (always blocked), env isolation, EngineSecrets stripping, output truncation
- MongoDB module: find/findOne/insert/insertMany/update/delete/count/aggregate with NoSQL injection protection ($where/$function blocking), security config (AllowedDatabases, MaxDocumentSize, MaxResults, MaxPools)
- Redis module: get/set/del/exists/expire/ttl/incr/decr/hget/hset/hgetall/lpush/rpush/lrange/sadd/smembers/publish with auto JSON serialize/deserialize, KeyPrefix tenant isolation, blocked commands (FLUSHALL, CONFIG, etc.)
- Regex stdlib: match/findAll/replace with LRU cache (max 1000 patterns), ReDoS prevention
- Extension API: `Extension` struct with Functions/Structs/Constants, `WithExtension()` runtime option, `ext:name` import support
- Import resolver: `io:` and `ext:` imports recorded in `RequestedModules`, validated at runtime (two-layer security enforced)
- Runtime.Close(): propagates to all I/O modules implementing Close() error
- Bitcode bridge adapter: nested namespace maps for dotted access (bc.db.query), explicit type errors instead of silent coercion, optimized convertToAny
- GoJSONRunner: uses CompileFile for proper import resolution
- scripts/*.json support: `detectRuntimeFromExtension` detects `.json` → `go-json`, `GoJSONRunner` implements `EmbeddedRunner`
- Backward compatibility: `ProcessDefinition.Runtime` field, `IsGoJSON()` helper, old format unchanged
- Process engine data step replacement: `GoJSONDataHandler` adapts old step types to bridge calls
- CLI: run (--input/--input-file/--timeout/--io/--trace/stdin), check (--verbose), test (--filter/--verbose), ast (--output), codegen (--target/--output/--package), migrate (--from/--to/--dry-run, JSON-aware key renaming)
- Code generation: `CodeGenerator` interface, Go/JavaScript/Python generators handling all step types including new/import/expression/switch
- Windows path security: case-insensitive matching, directory boundary checks, `filepath.Rel`-based AllowedPaths validation
- Security: AllowedHosts explicitly override BlockedHosts (explicitly allowed hosts skip blocked check)
- Exec timeout: context deadline check prioritized over ExitError for reliable timeout detection
- MongoDB module: real driver (`go.mongodb.org/mongo-driver/v2`), lazy connection, full CRUD + aggregation
- Redis module: real driver (`github.com/redis/go-redis/v9`), lazy connection, 16 commands, auto JSON serialize
- CLI: `go-json ast --format` flag (forward-compatible, json only for now)
- 723 tests total across 10 packages (lang: 177, io: 132, stdlib: 103, server: 108, runtime: 79, generate: 48, cmd: 45, codegen: 36)

## What's Done (Phase 4.5d)

- **Server execution mode** (`go-json serve`): declarative routing, plugable framework adapters, middleware, template rendering, static files
- **ServerAdapter interface**: framework-agnostic abstraction with `RequestContext`/`Response` types, adapter registry
- **Framework adapters**: Fiber (default), net/http (stdlib), Echo (build-tagged), Gin (build-tagged), Chi (build-tagged)
- **Route system**: declarative routes with groups, prefix nesting, middleware merging, route flattening, validation
- **Handler bridge**: `BuildHandler` converts HTTP→go-json function→HTTP. `ExecuteFunction` added to Runtime and VM for direct function invocation
- **Request object**: full body parsing (JSON, form, multipart, text), file upload with temp file handling + cleanup, path params, query, headers, cookies, IP, store
- **Response convention**: status+body (JSON), data+render (template), redirect, cookies, headers, error field, 204 for nil
- **Middleware chain engine**: built-in (logger, recover, CORS, secure headers, request_id, compress, rate_limit) + custom go-json functions with short-circuit
- **JWT module**: middleware (header + cookie extraction, validation) + callable functions (sign, verify, decode, refresh)
- **Plugable auth system**: `AuthStrategy` interface with Bearer/JWT, API Key, Basic Auth, Custom (go-json function) strategies. `auth` and `auth:name` middleware resolution
- **Template engine**: Go html/template with 20+ built-in functions (json, formatDate, upper, lower, truncate, default, safeHTML, urlEncode, add/sub/mul/div/mod, seq, etc.), layouts, partials, dev-mode reload
- **Static file serving**: configurable directory + prefix, path traversal protection, hidden file blocking
- **OpenAPI/Swagger**: auto-generated OpenAPI 3.0 spec from routes, `api` annotation support (body/query/responses), security scheme mapping, Swagger UI at `/docs`, `go-json openapi` CLI command
- **Server codegen**: `ServerCodegenAdapter` interface with language×framework registry. Implementations: Go+Fiber, Go+net/http, JS+Express, Python+FastAPI
- **Codegen dependency management**: feature detection, go.mod/package.json/requirements.txt/.env.example generation
- **SQL unified query parameters**: `TranslateQuery` converts `?` and `:name` placeholders to driver-specific syntax (postgres $1, sqlserver @p1, oracle :1, sqlite/mysql ?). Integrated into SQL module
- **FS enhancements**: fs.stat, fs.copy, fs.move, fs.glob + enhanced fs.list
- **Path stdlib**: basename, dirname, extname, joinpath, cleanpath, isabs, stemname, pathsep
- **Stdlib additions**: toJSON/fromJSON, formatDate universal format (YYYY-MM-DD → Go layout translation)
- **CRUD generator**: database introspection (SQLite, PostgreSQL, MySQL), type mapping (DB→go-json→OpenAPI), manual fields mode, CRUD route+handler generation with validation
- **Auth scaffold generator**: register, login, refresh, me, change-password endpoints with JWT
- **Project scaffold generator**: full project structure (api.json, templates/, public/, migrations/, .env.example, README.md)
- **Pattern template engine**: built-in patterns (simple, service-layer, DDD, hexagonal), custom template support, per-model file generation
- **CLI commands**: `go-json serve` (--dev, --port, --host, --docs, --io), `go-json generate` (crud/auth/project with --table/--fields/--dsn/--auth/--output), `go-json openapi` (--output)
- **Server mode detection**: `IsServerProgram()` checks for routes key
- **Health endpoint**: built-in `/health` with status, name, uptime (bypasses middleware)
- **Graceful shutdown**: SIGINT/SIGTERM handling with configurable timeout
- 723 tests total across 9 packages (includes Phase 4.5a–d cumulative)
- Pattern templates: 4 built-in patterns (simple, service-layer, ddd, hexagonal) with actual template files

## What's NOT Done

- Expression-level compile-time type validation (deferred to runtime)
- MongoDB/Redis require running servers for actual I/O (tests use miniredis for Redis, security-only tests for MongoDB)
- REPL mode (future)
- Dev mode file watching (fsnotify — optional dependency, not yet added)
- Interactive mode for `go-json generate` (--interactive flag)
