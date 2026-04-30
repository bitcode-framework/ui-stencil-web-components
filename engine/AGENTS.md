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
