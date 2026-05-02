# AGENTS.md — BitCode Engine

## Overview

Go runtime that reads JSON module definitions and produces running applications. Handles models, APIs, processes, views, workflows, security, plugins, and real-time features.

**Architecture:** Presentation (Fiber HTTP) → Domain (DDD) → Infrastructure (GORM/MongoDB) with Runtime layer for execution engines.

## Package Structure

```
engine/
├── cmd/bitcode/           CLI entry point (serve, dev, init, validate, module, user, db, seed, version)
├── internal/
│   ├── compiler/parser/   JSON parsers — one file per definition type (model, api, process, view, workflow, module)
│   ├── domain/            Domain models (DDD: entity, aggregate, events). No DB imports.
│   │   ├── event/         Domain event bus
│   │   ├── model/         Model registry with reserved namespace validation
│   │   ├── security/      Record rule domain model (GORM struct with domain_filter + domain_filter_expr)
│   │   └── setting/       Setting store
│   ├── runtime/           Execution engines
│   │   ├── bridge/        go-json ↔ engine bridge adapter
│   │   ├── embedded/      Embedded JS runtimes (goja, quickjs, yaegi)
│   │   ├── executor/      Process executor (sequential + DAG), step handlers, condition evaluator
│   │   ├── expression/    Computed field hydrator + aggregate functions (uses go-json EvalExpr)
│   │   ├── format/        Auto-format engine (naming series, sequences)
│   │   ├── hook/          Event dispatcher (model hooks, on_change cascade, retry)
│   │   ├── pkgen/         Primary key generator (UUID v4/v7, auto_increment, naming_series, composite, natural_key)
│   │   ├── plugin/        Plugin manager (TypeScript + Python via JSON-RPC)
│   │   ├── sync/          Offline sync engine
│   │   ├── validation/    Validation engine (built-in + custom validators)
│   │   └── workflow/      State machine engine
│   ├── infrastructure/
│   │   ├── cache/         Memory + Redis cache
│   │   ├── i18n/          Translation system
│   │   ├── module/        Module loader, dependency resolver, data seeder, migration engine
│   │   ├── persistence/   Repository layer (GORM + MongoDB), query builder, record rules, OQL parser
│   │   └── storage/       File storage (local + S3)
│   └── presentation/
│       ├── admin/         Built-in admin panel
│       ├── api/           REST API handlers, router, CRUD, Swagger
│       ├── graphql/       GraphQL schema builder + resolver
│       ├── middleware/    Auth, permission, record rule, tenant, rate limit, audit, CORS
│       ├── template/      Go html/template engine with helpers
│       ├── view/          SSR view renderer
│       └── websocket/     Real-time WebSocket hub + CRUD
├── pkg/                   Public packages (security/JWT, email, DDD helpers)
├── modules/               Built-in modules (base)
├── embedded/              Go-embedded modules compiled into binary
└── docs/                  Engine documentation
```

## Expression Engine (Phase 4.5e)

All expression evaluation is unified through `go-json/runtime.EvalExpr()` backed by `expr-lang/expr`:

| Component | Function Used | Purpose |
|-----------|--------------|---------|
| Computed fields | `runtime.EvalExpr()` | Evaluate `computed`/`formula` field expressions |
| Process conditions | `runtime.EvalExprBool()` | `if`/`switch`/`loop` conditions in process steps |
| Hook conditions | `runtime.EvalExprBool()` | Agent/event trigger conditions |
| DAG edge conditions | `runtime.EvalExprBool()` | DAG process edge conditions (shared with process) |
| Record rules (new) | `runtime.ParseExpr()` + AST walker | `domain_filter_expr` → parameterized SQL WHERE |

**Shared singleton:** `runtime.sharedEngine` (ExprLangEngine with compilation cache, thread-safe).

**Key files:**
- `packages/go-json/runtime/eval.go` — Public API: EvalExpr, EvalExprBool, EvalExprFloat, ParseExpr, ValidateExpr
- `engine/internal/runtime/expression/hydrator.go` — Computed field evaluation
- `engine/internal/runtime/expression/aggregate.go` — sum/count/avg/min/max over child collections
- `engine/internal/runtime/executor/condition.go` — Process/DAG condition evaluation + {{}} normalization
- `engine/internal/runtime/hook/dispatcher.go` — Hook condition evaluation (evaluateHandlerCondition)
- `engine/internal/infrastructure/persistence/record_rule_expr.go` — AST-to-WHERE converter for record rules

**Reserved namespaces** (cannot be model field names for new models):
`ctx`, `input`, `old`, `session`, `self`

## Record Rules

Two formats coexist:
- **Legacy:** `domain_filter` — `[["field","op","{{user.id}}"]]` with string interpolation
- **New:** `domain_filter_expr` — `field == ctx.user_id` with expr-lang AST-to-WHERE conversion

Both produce parameterized SQL. New format supports nested AND/OR, `contains`/`startsWith`/`endsWith` (→ LIKE), `in`/`not in`, and conditional rules (ternary).

Security: `matches` operator rejected (ReDoS risk), tautology detection, field validation, ctx-only access whitelist, MaxNodes=200 limit, empty array/nil → deny-all.

## Polymorphic Relations (Phase 6B)

5 morph types: `morph_to`, `morph_one`, `morph_many`, `morph_to_many`, `morph_by_many`.

| Type | Creates Columns | Description |
|------|----------------|-------------|
| `morph_to` | `{name}_type` + `{name}_id` | Child belongs to any parent type |
| `morph_one` | None (virtual) | Parent has one polymorphic child |
| `morph_many` | None (virtual) | Parent has many polymorphic children |
| `morph_to_many` | Junction table `{morph}s` | Many-to-many polymorphic (parent side) |
| `morph_by_many` | None (uses existing junction) | Many-to-many polymorphic (inverse side) |

Key files:
- `internal/compiler/parser/model.go` — FieldType constants + validation
- `internal/infrastructure/persistence/dynamic_model.go` — Column/index/junction table generation
- `internal/infrastructure/persistence/repository.go` — 5 morph loaders + MorphAttach/Detach/Sync
- `internal/infrastructure/persistence/mongo_migration.go` — MongoDB indexes + junction collections
- `internal/infrastructure/persistence/mongo_repository.go` — MongoDB morph operations
- `internal/runtime/bridge/model.go` — Bridge API (morphAttach/morphDetach/morphSync)
- `internal/domain/model/registry.go` — MorphMap (type aliasing)

Morph Map: `Registry.SetMorphMap(map)`, `Registry.MorphType(modelName)`, `Registry.MorphModel(morphType)`. Default: model name used as-is.

## Engine Enhancements (Phase 6C)

### Array-Backed Models
3 source types: `db` (default), `array`, `process`. Array models load data from inline `rows` or `rows_file` (JSON/CSV/XLSX/XML) into the main DB on startup. Process models execute a named process/script to get rows.

| Mode | `writable` | API | Sync on restart |
|------|:----------:|-----|-----------------|
| Read-only | `false` | GET only, 405 on writes | DELETE all + re-INSERT |
| Writable | `true` | Full CRUD | Seed only if empty |

`sync_source: true` writes DB changes back to the source file. `refresh` config enables interval-based re-sync.

Key files:
- `internal/compiler/parser/model.go` — Source, Writable, DataRows, RowsFile, SyncSource, Refresh, Process, Script fields
- `internal/infrastructure/persistence/array_sync.go` — SyncArrayModel (read-only vs writable sync)
- `internal/infrastructure/persistence/array_parser.go` — LoadRowsFromFile (JSON/CSV/XLSX/XML)
- `internal/infrastructure/persistence/array_writer.go` — WriteBackToFile (sync_source write-back)
- `internal/runtime/refresh/scheduler.go` — Interval-based refresh scheduler

### View Modifiers
`visible_if`, `disabled_if`, `readonly_if`, `css_class`, `help_text` on LayoutRow and ChildTableColumn. Rendered as `data-visible-if`/`data-disabled-if` attributes for client-side evaluation.

### Metadata API
10 endpoints under `/api/v1/_meta/`: models, models/:name, models/:name/fields, views, views/:name, modules, modules/:name, processes, processes/:name, field-types. Plus `POST /api/v1/_meta/models/:name/refresh` for manual refresh.

Key file: `internal/presentation/api/meta_handler.go`

### Embedded View filter_by
TabDefinition supports `filter_by` (string or map) to scope embedded views by parent record.

### Process Data Source
DataSourceDefinition supports `process` field (mutually exclusive with `model`) to execute a process for view data.

### Eager Loading Fixes
- `WithClause.Conditions` applied to ALL relation types (many2one, one2many, many2many, all 5 morph types)
- `WithClause.Select`, `OrderBy`, `Limit` applied consistently via `applyWithClauseToQuery` helper
- `WithClause.Nested` triggers recursive loading (max depth 3, configurable)
- `?with=` query param on Read endpoint (`GET /:id`) — comma-separated or JSON array

## Bridge API Ergonomics (Phase 4.5h)

Three additive enhancements to the `ext:bitcode` bridge:

### Fluent Model API
Chainable query builder alongside existing `search({...})` API:
- `bc.model('user').where('active', true).limit(10).get()`
- `bc.model('user').find(id).update(data)` / `.delete()`
- `bc.model('user').where(...).orderBy('name').first()`
- `bc.model('user').where(...).count()` / `.sum('field')`

Key files: `bridge/query_builder.go`

### Unified Imperative Transaction API
All runtimes now use imperative `begin/commit/rollback` (callback style removed from goja/yaegi):

| Runtime | API |
|---------|-----|
| go-json | `bc.tx.begin()` / `bc.tx.begin({timeout: "5m"})` / `bc.tx.begin({timeout: 0})` |
| goja | `bitcode.tx.begin()` / `bitcode.tx.begin({timeout: 60})` |
| yaegi | `bitcode.Tx().Begin()` / `bitcode.Tx().BeginWithTimeout(5*time.Minute)` |
| Node.js | `await bitcode.tx.begin()` / `await bitcode.tx.begin({timeout: 300})` |
| Python | `bitcode.tx.begin()` / `bitcode.tx.begin({timeout: 300})` |

Timeout behavior (consistent across all runtimes):
- Default: 30s — tx auto-rolled-back if not committed/rolled-back within 30 seconds
- Custom: pass `timeout` in seconds (int/float), or duration string ("5m", "1h")
- Infinite: `timeout: 0` — no auto-rollback, relies on program timeout + VM Close/Cleanup

Safety: configurable auto-rollback timeout, auto-rollback on VM close, mutex-protected state.

Key files: `bridge/tx_bridge.go`, `embedded/goja/proxy.go`, `embedded/yaegi/symbols.go`, `plugin/tx_store.go`

### Email Template Shorthand
`bc.email.template('welcome', email, data)` — shorthand for `bc.email.send({template: 'welcome', to: email, data: data})`

## Primary Keys

6 strategies: `uuid` (v4/v7), `auto_increment`, `composite`, `natural_key`, `naming_series`, `manual`.
Default: UUID v4. All PKs and FKs are string type (TEXT/UUID/CHAR(36) depending on driver).

## Testing

```bash
cd engine
go build ./...           # Must pass
go vet ./...             # Must pass
go test ./... -v         # Requires DB setup for some packages
go test ./internal/runtime/expression/ -v    # Expression tests
go test ./internal/runtime/executor/ -v      # Process executor tests
go test ./internal/runtime/hook/ -v          # Hook dispatcher tests
```

## Conventions

- Follow root `AGENTS.md` conventions
- All exported types and functions need Go doc comments
- No unnecessary comments
- Tests required for new functionality
- `go build ./...` and `go vet ./...` must pass
- UUID for all IDs (never auto-increment integers for business entities)
- Fail-closed for security (error → deny access, not allow)
