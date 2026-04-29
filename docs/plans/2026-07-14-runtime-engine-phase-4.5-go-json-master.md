# go-json — Standalone JSON Programming Language (Master Draft)

**Status**: Approved
**Date**: 28 April 2026
**Design Decisions**: See [brainstorming design doc](./2026-04-28-go-json-brainstorming-design.md) for rationale
**Package**: `packages/go-json`
**Module**: `github.com/bitcode-framework/go-json`
**Phases**: 4.5a, 4.5b, 4.5c, 4.5d, 4.5e

---

## 1. Vision

go-json is a standalone, general-purpose programming language written in JSON format, embeddable in Go applications.

**Analogy:**
```
Stencil : Web Components : Bitcode View Engine
go-json : JSON Language  : Bitcode Process Engine
```

Stencil can be used without bitcode — anyone can build web components with Stencil. But bitcode uses Stencil as its core view engine.

go-json can be used without bitcode — anyone can build automation, workflows, business rules with go-json. But bitcode uses go-json as its core process/scripting engine.

### 1.1 What Makes go-json Different

| Existing Tool | Focus | Limitation |
|---|---|---|
| jsonnet | Data templating | No side effects (no I/O) |
| jq | JSON transformation | Single-purpose, no general programming |
| AWS Step Functions | Cloud workflow | AWS-locked, no local execution |
| Node-RED | IoT/integration | Node.js-dependent, visual-only |
| n8n | Automation | SaaS-focused, not embeddable |
| CEL | Policy/rules | Expression only, no control flow |
| expr-lang | Expression evaluation | Expression only, no statements |

**go-json's niche**: General-purpose JSON programming language embeddable in Go applications. No existing tool fills this niche.

### 1.2 Design Principles

1. **JSON-native** — Programs are valid JSON/JSONC. Hybrid format supporting both strict JSON and JSONC (with comments).
2. **Standalone** — Zero dependency on bitcode. Usable by anyone.
3. **Embeddable** — Host applications extend go-json via extension hooks.
4. **Gradually typed** — Untyped for quick scripts, fully typed for production.
5. **Safe by default** — Resource limits, no infinite loops, memory-safe.
6. **Code-generation ready** — Well-defined AST enables transpilation to Go/JS/Python.

---

## 2. Topic Index — Where to Find What

### Phase 4.5a — Core Language + Stdlib
**Document**: [go-json-phase-4.5a-core-language.md](./go-json-phase-4.5a-core-language.md)

| Topic | Section |
|---|---|
| Package structure (`packages/go-json/`) | §1 |
| Expression engine (expr-lang/expr) | §2 |
| Value assignment: `value` / `expr` / `with` | §3 |
| Variable declaration: `let` / `set` | §3 |
| Type system: inference, gradual typing, input schema | §4 |
| Control flow: `if`/`elif`/`else`, `switch` | §5 |
| Loops: `for`/`while`/`range`, `break`/`continue` | §6 |
| Functions: definition, params, returns | §7 |
| Recursion: depth limit, isolated scope, tail-call | §7.4 |
| Error handling: `try`/`catch`/`finally`, `error` | §8 |
| Return values: `return`, computed objects | §9 |
| Resource limits: depth, steps, timeout, memory proxy | §10 |
| Config ordering: engine > project > module > program > step | §10.3 |
| Stdlib Tier 1: math (14), string (20), array (20), type (6) | §11 |
| Context: session, execution metadata | §12 |
| Variable scoping: block scope, isolation | §13 |
| JSONC pre-processor + `_c` semantic comments | §14.5 |
| Debugging & observability | §15.5 |
| Program reuse & concurrency | §15.6 |
| Versioning & backward compatibility | §15.7 |
| Structured logging | §15.8 |

### Phase 4.5b — Modularity (Struct + Import)
**Document**: [go-json-phase-4.5b-modularity.md](./go-json-phase-4.5b-modularity.md)

| Topic | Section |
|---|---|
| Struct definition: fields, defaults, nested structs | §1 |
| Struct methods: `self`, mutation vs immutable | §2 |
| Frozen structs (opt-in immutability) | §2.3 |
| Struct construction: `new` + `with` | §3 |
| Nested property access + mutation: `set "a.b.c"` | §4 |
| Import system: relative, stdlib, extension | §5 |
| I/O module explicit import (`io:http`, `io:fs`) | §5.2 |
| Import resolution rules | §5.3 |
| Export rules: structs + functions exportable, steps not | §5.4 |
| Circular import detection | §5.5 |
| Re-export / barrel files | §5.6 |
| Stdlib Tier 2: map (8), datetime (10), encoding (6), crypto (4), format (2) | §6 |
| Parallel execution: `parallel`/`join`, isolated branches | §7 |
| Nullable types: `?string`, `?Person` | §8 |

### Phase 4.5c — I/O + Integration + Code Generation
**Document**: [go-json-phase-4.5c-io-integration.md](./go-json-phase-4.5c-io-integration.md)

| Topic | Section |
|---|---|
| I/O modules: HTTP, FS, SQL, exec | §1 |
| I/O security: enable/disable per module | §1.2 |
| Bitcode bridge integration via extension hooks | §2 |
| Extension API: `WithExtension()`, `WithoutIO()` | §2.2 |
| How bitcode replaces raw I/O with bridge | §2.3 |
| `scripts/*.json` support in bitcode process engine | §3 |
| Migration path: current process engine → go-json | §3.2 |
| AST export for code generation | §4 |
| Code generation targets: Go, JavaScript, Python | §4.2 |
| Standalone CLI runner: `go-json run program.json` | §5 |
| REPL / playground | §5.2 |
| Testing framework (`go-json test`) | §5.3 |
| Migration tool (`go-json migrate`) | §5.4 |

### Phase 4.5d — Web Server
**Document**: [go-json-phase-4.5d-web-server.md](../2026-04-29-runtime-engine-phase-4.5d-go-json-web-server.md)

| Topic | Section |
|---|---|
| Declarative HTTP routing | §3 |
| Plugable auth (JWT/API Key/Basic/Custom) | §6 |
| Middleware system | §6 |
| Template engine (Go html/template) | §7 |
| Static files | §8 |
| Server adapter interface | §9 |
| JWT module | §10 |
| OpenAPI/Swagger auto-generation | §15 |
| CRUD generator with DB introspection | §24 |
| Architecture patterns (simple/service-layer/DDD/hexagonal) | §25 |

### Phase 4.5e — Unified Expression Engine
**Document**: [go-json-phase-4.5e-unified-expression-engine.md](./2026-07-14-runtime-engine-phase-4.5e-unified-expression-engine.md)

| Topic | Section |
|---|---|
| Problem: 3 hand-rolled evaluators + bugs | §1 |
| Public `EvalExpr()` API for engine consumers | §2 |
| Computed field evaluator replacement | §3 |
| Process condition evaluator replacement (fixes `" > "` bug) | §4 |
| Hook/agent condition evaluator replacement | §5 |
| Record rules: expr-lang with `ctx.*` namespace | §6 |
| AST-to-WHERE conversion pipeline | §6.4–6.5 |
| Record rule security validations | §6.6 |
| Reserved namespaces (`ctx`, `input`, `old`, `session`) | §7 |
| Files changed (deleted, rewritten, added) | §8 |
| Edge cases | §9 |

---

## 3. Open Questions

All open questions have been resolved. See [brainstorming design doc](../plans/2026-04-28-go-json-brainstorming-design.md) for full analysis and rationale.

### OQ-1: Return Computed Objects — RESOLVED
**Phase**: 4.5a §9
**Decision**: Use overloaded `return` supporting expression strings, `with`, and `value` forms.
**Details**: See brainstorming design doc §3

### OQ-2: Struct Mutability — RESOLVED
**Phase**: 4.5b §2
**Decision**: Structs are mutable by default with optional `frozen: true` for opt-in immutability.
**Details**: See brainstorming design doc §3

### OQ-3: Function Call Duality — RESOLVED
**Phase**: 4.5a §7
**Decision**: Support both expression-level and step-level calls, chosen by purity or complexity needs.
**Details**: See brainstorming design doc §3

### OQ-4: Error Types — RESOLVED
**Phase**: 4.5a §8
**Decision**: Support both string and structured throws, auto-normalized into a structured catch object.
**Details**: See brainstorming design doc §3

### OQ-5: Parallel Error Handling — RESOLVED
**Phase**: 4.5b §7
**Decision**: Make parallel error handling configurable with `cancel_all` default plus `continue` and `collect` modes.
**Details**: See brainstorming design doc §3

### OQ-6: Namespace / Module Namespace for Struct Disambiguation — RESOLVED
**Phase**: 4.5b §5
**Decision**: Import aliases are the namespace; no explicit namespace declaration is required inside program files.
**Details**: See brainstorming design doc §3

### OQ-7: Stdlib Function Call in Expressions — Chained / Namespaced — RESOLVED
**Phase**: 4.5a §11, 4.5b §6
**Decision**: Keep stdlib flat by default, use expr-lang pipe for chaining, and reserve namespaces for grouped modules like `crypto.*` and `regex.*`.
**Details**: See brainstorming design doc §3

### OQ-8: Undefined/Untyped Variables — Dynamic Type or Compile Error? — RESOLVED
**Phase**: 4.5a §4
**Decision**: Default to strict-after-first-assignment, with explicit `any` for dynamic values and `?T` for nullable values.
**Details**: See brainstorming design doc §3

### OQ-9: Eval / Inline Code Execution in Other Languages — RESOLVED
**Phase**: 4.5c
**Decision**: Keep eval out of the core language and expose it only as an opt-in host extension.
**Details**: See brainstorming design doc §3

---

## 4. Architecture Overview

```
packages/go-json/
├── go.mod                        # github.com/bitcode-framework/go-json
├── lang/                         # Core language (Phase 4.5a)
│   ├── ast.go                    # AST node types (with _c metadata)
│   ├── preprocess.go             # JSONC → JSON (strip comments, trailing commas)
│   ├── parser.go                 # JSON → AST
│   ├── compiler.go               # AST → validated program (type inference, cycle detection)
│   ├── vm.go                     # Tree-walk interpreter (with debug hooks)
│   ├── scope.go                  # Variable scoping (block scope, isolation)
│   ├── types.go                  # Type system (gradual)
│   ├── errors.go                 # Error types with position info + enrichment
│   ├── expr_engine.go            # ExprEngine interface + ExprLangEngine impl
│   ├── program.go                # Immutable compiled Program (concurrent-safe)
│   └── debugger.go               # Debugger interface + execution trace
├── stdlib/                       # Layer 2 only — go-json additions (~28 functions)
│   ├── math.go                   # ~7 functions
│   ├── strings.go                # ~5 functions
│   ├── arrays.go                 # ~3 functions
│   ├── maps.go                   # ~4 functions
│   ├── crypto.go                 # ~4 functions
│   ├── regex.go                  # ~3 functions
│   └── fmt.go                    # ~2 functions
├── io/                           # I/O extensions (Phase 4.5c)
│   ├── http.go
│   ├── fs.go
│   ├── sql.go
│   └── exec.go
├── runtime/                      # Runtime configuration
│   ├── runtime.go                # Runtime struct with program cache
│   ├── limits.go                 # Resource limits
│   ├── context.go                # Execution context
│   ├── logger.go                 # Logger interface + default impl
│   └── hooks.go                  # Extension hooks
├── cmd/                          # CLI (Phase 4.5c)
│   └── go-json/
│       └── main.go               # `go-json run/check/test/ast/codegen/migrate`
├── codegen/                      # Code generation (Phase 4.5c)
│   ├── golang.go                 # AST → Go code
│   ├── javascript.go             # AST → JavaScript
│   └── python.go                 # AST → Python
└── testdata/                     # Test programs
    ├── hello.json
    ├── hello.jsonc
    ├── factorial.json
    └── ...
```

### 4.1 How Bitcode Consumes go-json

```go
// In bitcode engine
import (
    gojson "github.com/bitcode-framework/go-json/lang"
    "github.com/bitcode-framework/go-json/stdlib"
)

rt := gojson.NewRuntime(
    gojson.WithStdlib(stdlib.All()),
    gojson.WithoutIO(),                              // disable raw I/O
    gojson.WithExtension("bitcode", bitcodebridge),   // inject bridge
    gojson.WithLimits(gojson.Limits{...}),
)

result, err := rt.Execute(programJSON, input)
```

### 4.2 How Others Use go-json Standalone

```go
import (
    gojson "github.com/bitcode-framework/go-json/lang"
    "github.com/bitcode-framework/go-json/stdlib"
    goio "github.com/bitcode-framework/go-json/io"
)

rt := gojson.NewRuntime(
    gojson.WithStdlib(stdlib.All()),
    gojson.WithIO(goio.All()),    // enable HTTP, FS, SQL, exec
)

result, err := rt.Execute(programJSON, input)
```

---

## 5. Phase Dependencies

```
Phase 4.5a (Core Language)
  │
  ├──► Phase 4.5b (Modularity) ──► Phase 4.5c (I/O + Integration)
  │                                       │
  └───────────────────────────────────────►├──► Phase 4.5d (Web Server)
                                           │         │
                                           ├──► Phase 4.5e (Unified Expression Engine)
                                           │         │
                                           ▼         ▼
                                    Phase 7 (Module "setting")
```

Phase 4.5a MUST complete before 4.5b.
Phase 4.5b MUST complete before 4.5c.
Phase 4.5c MUST complete before 4.5d and 4.5e (4.5d and 4.5e can run in parallel).
Phase 4.5d and 4.5e MUST complete before Phase 7.

---

## 6. Language Quick Reference

### 6.1 Program Structure

```json
{
  "name": "program_name",
  "go_json": "1",
  "import": { ... },
  "structs": { ... },
  "functions": { ... },
  "input": { ... },
  "steps": [ ... ]
}
```

All top-level keys are optional except `name`.
- File with `steps` = executable program
- File without `steps` = library (only structs + functions, importable)

### 6.2 Step Types

| Step | Phase | Purpose |
|---|---|---|
| `let` | 4.5a | Declare new variable |
| `set` | 4.5a | Update existing variable |
| `if`/`elif`/`else` | 4.5a | Conditional branching |
| `switch`/`cases` | 4.5a | Multi-way branching |
| `for`/`in` | 4.5a | Iterate over array |
| `for`/`range` | 4.5a | Iterate over number range |
| `while` | 4.5a | Conditional loop |
| `break` | 4.5a | Exit loop |
| `continue` | 4.5a | Skip to next iteration |
| `return` | 4.5a | Return value from function/program |
| `call` | 4.5a | Call function with input |
| `try`/`catch`/`finally` | 4.5a | Error handling |
| `error` | 4.5a | Throw error |
| `log` | 4.5a | Log message |
| `new` | 4.5b | Construct struct instance |
| `parallel` | 4.5b | Parallel execution |

### 6.3 Value Modes

| Mode | Syntax | Semantics |
|---|---|---|
| Literal | `"value": 42` | JSON value as-is, no evaluation |
| Expression | `"expr": "age + 1"` | Evaluated by expr-lang |
| Computed object | `"with": {"k": "expr"}` | Each field value is expression |

Only one of `value`/`expr`/`with` allowed per step. Multiple = compile error.

Nested `with` values are evaluated recursively at all nesting levels: every nested string is treated as an expression, while non-string values remain literal.

### 6.4 Type Vocabulary

| Type | Example | Note |
|---|---|---|
| `string` | `"hello"` | |
| `int` | `42` | |
| `float` | `3.14` | |
| `bool` | `true` | |
| `[]T` | `[]string`, `[]int` | Typed array |
| `[]any` | `[1, "two", true]` | Mixed array |
| `map` | `{"k": "v"}` | String-keyed map |
| `StructName` | `Person`, `Address` | User-defined struct |
| `?T` | `?string`, `?Person` | Nullable |
| `any` | anything | Opt-out of type checking |
