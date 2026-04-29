# go-json Language Design — Brainstorming Results

**Date**: 2026-04-28
**Status**: Approved (brainstorming complete)
**Scope**: All open questions, blind spots, and fundamental design decisions for go-json
**Motto**: Simplicity, Flexible, Powerful, Complete

---

## Table of Contents

1. [Context & Perspective](#1-context--perspective)
2. [Fundamental Decisions](#2-fundamental-decisions)
   - F-1: File Format (Hybrid JSON/JSONC)
   - F-2: Comment System (`_c` + `//`)
   - F-3: expr-lang Integration Strategy
   - F-4: Stdlib Redesign (3-Layer Architecture)
3. [Open Question Verdicts (OQ-1 to OQ-9)](#3-open-question-verdicts)
4. [Blind Spot Solutions (BS-1 to BS-7)](#4-blind-spot-solutions)
5. [Hard Problem Solutions (HP-1 to HP-4)](#5-hard-problem-solutions)
6. [Updated Architecture Overview](#6-updated-architecture-overview)
7. [Implementation Priority](#7-implementation-priority)
8. [Decision Log](#8-decision-log)

---

## 1. Context & Perspective

### 1.1 Core Insight

The primary users of go-json will NOT write JSON by hand. The primary interfaces are:

```
┌─────────────────────────────────┐
│  Visual Editor (like n8n)       │  ← Primary interface
│  AI Assistant (generate JSON)   │  ← Primary interface
│  Code Editor (write JSON)       │  ← Secondary/advanced
└─────────────────────────────────┘
              │
              ▼
┌─────────────────────────────────┐
│  go-json Engine                 │  ← THIS must be POWERFUL
│  (parser → compiler → VM)       │
└─────────────────────────────────┘
```

This means:
- **Engine correctness and power** is the top priority
- **JSON writability** is secondary — editors and AI generate JSON
- **JSON structure must be machine-friendly** — easy to generate, unambiguous to parse
- **Edge cases must be handled correctly** — lowcode users cannot debug subtle bugs

### 1.2 Design Motto Applied

| Principle | What it means for go-json |
|---|---|
| **Simplicity** | One way to do things. Consistent patterns. Predictable behavior |
| **Flexible** | Works for simple scripts AND complex business logic |
| **Powerful** | Approaches real programming language capability |
| **Complete** | Handles edge cases. No "sorry, you can't do that" moments |

---

## 2. Fundamental Decisions

### F-1: File Format — Hybrid JSON/JSONC

**Decision**: go-json supports both strict JSON and JSONC (JSON with Comments).

**File extensions**: Both `.json` and `.jsonc` are valid go-json program files.

**Parser behavior**: Always runs pre-processor that strips `//`, `/* */` comments and trailing commas. If no comments exist, this is a no-op (zero cost).

```
Input file (.json or .jsonc)
       │
       ▼
┌──────────────────────┐
│  Pre-processor        │  ← Strip // comments, /* */ blocks, trailing commas
│  (~30 lines Go)       │     Output: valid strict JSON string
└──────────────────────┘
       │
       ▼
┌──────────────────────┐
│  encoding/json        │  ← Standard Go JSON parser, battle-tested
│  json.Unmarshal()     │
└──────────────────────┘
       │
       ▼
┌──────────────────────┐
│  go-json parser       │  ← JSON → AST, type check, compile
│  (lang/parser.go)     │     _c fields → AST metadata
└──────────────────────┘
```

**Rules**:

| Scenario | Behavior |
|---|---|
| `program.json` (strict JSON) | Works — pre-processor is no-op |
| `program.json` (with comments) | Works — pre-processor strips comments |
| `program.jsonc` | Works — identical to above |
| AI/editor generates strict JSON | Works — most portable |
| Developer writes JSONC | Works — most expressive |
| Import across formats | Works — `"import": {"a": "./lib.json", "b": "./utils.jsonc"}` |

**CLI support**:
```bash
go-json run program.json      # works
go-json run program.jsonc     # works
go-json check program.json    # works
go-json check program.jsonc   # works
```

---

### F-2: Comment System — Dual Mechanism

**Decision**: Two comment mechanisms with different semantics.

#### `//` and `/* */` — Cosmetic Comments

- Stripped by pre-processor before parsing
- NOT stored in AST
- NOT emitted in code generation output
- For: quick notes, TODOs, section separators, temporary disable

```jsonc
// === PHASE 1: DATA LOADING ===
{"let": "lead", "call": "fetchLead", "with": {"id": "input.id"}},

/* TODO: Add caching here
   Waiting for Redis integration */
{"let": "orders", "call": "fetchOrders", "with": {"user_id": "lead.user_id"}}
```

#### `_c` — Semantic Comments

- Parsed and stored in AST as node metadata
- Emitted as comments in code generation output (Go/JS/Python)
- Available via documentation API
- For: documenting business logic, requirements references

**Two forms**:

```jsonc
// Form 1: Inline — attached to a step
{"_c": "Fetch lead with all relations",
 "let": "lead", "call": "fetchLead", "with": {"id": "input.id"}}

// Form 2: Standalone — comment-only step (parser skips, not counted in step counter)
{"_c": "=== PHASE 2: BUSINESS RULES ==="}

// Form 3: Multi-line
{"_c": ["Business rule: Gold tier auto-approval",
        "Requirement: JIRA-1234",
        "Last reviewed: 2025-01-15"],
 "if": "lead.tier == 'gold'", "then": [...]}
```

**Rules**:

| Property | `//` `/* */` | `_c` |
|---|---|---|
| Nature | Cosmetic — for humans | Semantic — part of program |
| In AST | No — stripped | Yes — stored as metadata |
| In code generation | No — lost | Yes — emitted as comments |
| In runtime | No | No — not executed |
| Analogy | `//` in Go source | Go doc comments / JSDoc |

**Both can coexist** in the same file:

```jsonc
{
  "name": "process_leads",
  "_c": "Main lead processing pipeline",

  // Input validation schema
  "input": {
    "lead_id": "int",
    "action": "string"  // "approve" | "reject" | "defer"
  },

  "steps": [
    // ── PHASE 1: DATA LOADING ───
    {"_c": "Fetch lead with all relations",
     "let": "lead", "call": "fetchLead", "with": {"id": "input.lead_id"}},

    {"_c": "Gold tier auto-approval (JIRA-1234)",
     "if": "lead.tier == 'gold'", "then": [
      {"return": "{'status': 'auto_approved'}"}
    ]}
  ]
}
```

---

### F-3: expr-lang/expr Integration Strategy

**Decision**: Use expr-lang/expr as expression engine, accessed through an abstraction layer.

#### 3.1 Why expr-lang

- Compile-time type checking
- Bytecode VM (fast, concurrent-safe)
- Rich built-in functions (~60 functions)
- Pipe operator, optional chaining, nil coalescing
- AST patching, operator overloading, custom functions
- Expr Editor (CodeMirror-based) available for visual editor integration
- Pure Go, no CGO, actively maintained

#### 3.2 Abstraction Layer

```go
// expr_engine.go
type ExprEngine interface {
    Compile(expr string, env map[string]any) (CompiledExpr, error)
    Run(compiled CompiledExpr, env map[string]any) (any, error)
    Eval(expr string, env map[string]any) (any, error)
    Validate(expr string, env map[string]any) error
    ReturnType(expr string, env map[string]any) (string, error)
}

type ExprLangEngine struct {
    cache map[string]*vm.Program  // compiled expression cache
    mu    sync.RWMutex
}
```

**Why abstraction**:
- Testability — mock expression engine in unit tests
- Future-proofing — swap engine without rewriting VM
- Editor support — `ReturnType()` and `Validate()` needed by visual editor
- Error enrichment — wrap expr-lang errors with human-friendly messages

#### 3.3 expr-lang Configuration for go-json

```go
func (e *ExprLangEngine) compileExpr(expression string, env map[string]any) (*vm.Program, error) {
    return expr.Compile(expression,
        expr.Env(env),
        expr.WithContext("ctx"),           // timeout propagation
        expr.MaxNodes(1000),               // limit expression complexity
        expr.Timezone(e.timezone),         // consistent date handling
        // Register go-json stdlib functions
        e.stdlibFunctions()...,
    )
}
```

**Key expr-lang features to leverage**:

| Feature | Usage in go-json |
|---|---|
| `expr.WithContext("ctx")` | Propagate timeout/cancellation to all function calls |
| `expr.MaxNodes(n)` | Limit expression complexity (DoS prevention) |
| `expr.Function(name, fn, types...)` | Register go-json stdlib functions with type signatures |
| `expr.Operator(op, fn...)` | Future: operator overloading for `decimal` type |
| `expr.Patch(visitor)` | Security transforms, auto-context injection |
| `expr.ConstExpr(fn)` | Optimize constant stdlib calls at compile time |
| `expr.DisableBuiltin(name)` | Security sandboxing per-program |
| `expr.AsBool()` / `AsInt()` | Type-check expression return types |
| `expr.WarnOnAny()` | Warn on ambiguous `any` return types |

#### 3.4 expr-lang Features Already Available (DO NOT Reimplement)

These features exist natively in expr-lang and should be used directly:

| Feature | expr-lang syntax | Available in go-json expressions |
|---|---|---|
| Variables in expression | `let x = 42; x * 2` | Yes — for intermediate values within one expression |
| Pipe operator | `name \| lower() \| trim()` | Yes — for chaining |
| Multiline if | `if x > 0 { "pos" } else { "neg" }` | Yes — for simple conditional values |
| Range | `1..10` | Yes — for generating sequences |
| Optional chaining | `user?.address?.city` | Yes — for safe nested access |
| Nil coalescing | `value ?? "default"` | Yes — for defaults |
| Predicates | `filter(items, .price > 100)` | Yes — for array operations |
| `$env` | `'key' in $env` | Yes — for checking variable existence |
| Map literals | `{"name": x, "age": y}` | Yes — for constructing objects in expressions |
| Backtick strings | `` `multi\nline` `` | Yes — for raw strings |

---

### F-4: Stdlib Redesign — 3-Layer Architecture

**Decision**: Do NOT duplicate expr-lang built-in functions. Organize stdlib into 3 layers.

#### Layer 1: expr-lang Built-ins (DO NOT TOUCH)

These are already available in every expression. go-json does not reimplement them.

**String** (14 functions):
`trim`, `trimPrefix`, `trimSuffix`, `upper`, `lower`, `split`, `splitAfter`, `replace`, `repeat`, `indexOf`, `lastIndexOf`, `hasPrefix`, `hasSuffix`

**Number** (6 functions):
`abs`, `ceil`, `floor`, `round`, `max`, `min`

**Array** (22 functions):
`all`, `any`, `one`, `none`, `map`, `filter`, `find`, `findIndex`, `findLast`, `findLastIndex`, `groupBy`, `count`, `concat`, `flatten`, `uniq`, `join`, `reduce`, `sum`, `mean`, `median`, `first`, `last`, `take`, `reverse`, `sort`, `sortBy`

**Map** (2 functions):
`keys`, `values`

**Type Conversion** (10 functions):
`type`, `int`, `float`, `string`, `toJSON`, `fromJSON`, `toBase64`, `fromBase64`, `toPairs`, `fromPairs`

**Date/Time** (4 functions):
`now`, `date`, `duration`, `timezone`

**Misc** (2 functions):
`len`, `get`

**Bitwise** (8 functions):
`bitand`, `bitor`, `bitxor`, `bitnand`, `bitnot`, `bitshl`, `bitshr`, `bitushr`

**Total Layer 1: ~68 functions — zero implementation effort.**

#### Layer 2: go-json Stdlib Additions (IMPLEMENT THESE)

Only functions NOT available in expr-lang:

**Math** (~7 functions):
`clamp(x, min, max)`, `sign(x)`, `randomInt(min, max)`, `randomFloat(min, max)`, `pow(base, exp)`, `sqrt(x)`, `mod(a, b)`

**String** (~5 functions):
`padLeft(str, len, char)`, `padRight(str, len, char)`, `substring(str, start, end)`, `sprintf(format, args...)`, `contains(str, substr)` (if not covered by expr-lang operator)

**Array** (~3 functions):
`chunk(array, size)`, `zip(array1, array2)`, `range(start, end, step)` (if expr-lang `..` insufficient)

**Map** (~4 functions):
`has(map, key)`, `merge(map1, map2)`, `pick(map, keys)`, `omit(map, keys)`

**Crypto** (~4 functions):
`crypto.sha256(str)`, `crypto.md5(str)`, `crypto.uuid()`, `crypto.hmac(str, key, algo)`

**Regex** (~3 functions):
`regex.match(str, pattern)`, `regex.findAll(str, pattern)`, `regex.replace(str, pattern, replacement)`

**Format** (~2 functions):
`sprintf(format, args...)`, `numberFormat(n, decimals, decSep, thousandSep)`

**Total Layer 2: ~28 functions — actual implementation work.**

#### Layer 3: I/O Modules (Phase 4.5c)

`http.get/post/put/patch/delete`, `fs.read/write/exists/list/mkdir/remove`, `sql.query/execute`, `exec.run`, `regex.match/findAll/replace`

#### Naming Convention

**Rule**: Follow expr-lang naming for all existing functions. Do NOT create aliases with different names.

| expr-lang name | Draft name (OLD) | Decision |
|---|---|---|
| `hasPrefix` | `startsWith` | Use `hasPrefix` |
| `hasSuffix` | `endsWith` | Use `hasSuffix` |
| `uniq` | `unique` | Use `uniq` |
| `mean` | `avg` | Use `mean` (add `avg` as alias if needed) |
| `toPairs` | `entries` | Use `toPairs` |
| `fromPairs` | `fromEntries` | Use `fromPairs` |

---

## 3. Open Question Verdicts

### OQ-1: Return Computed Objects — RESOLVED

**Verdict**: Overloaded `return` — supports `expr` string (including object literals), `with`, and `value`. Consistent with `let`/`set` pattern.

```jsonc
// Form 1: Expression (most common)
{"return": "result"}
{"return": "score + 1"}

// Form 2: Object literal in expression (expr-lang native)
{"return": "{'status': 'ok', 'count': len(items)}"}

// Form 3: Computed object via with (for complex objects)
{"return": {"with": {
  "status": "'eligible'",
  "person": "candidate",
  "count": "len(items)"
}}}

// Form 4: Literal value (for static objects)
{"return": {"value": {"status": "ok", "code": 200}}}
```

**Rules**:
- `{"return": "string"}` → string is expression, evaluated
- `{"return": {"with": {...}}}` → computed object, each value is expression
- `{"return": {"value": ...}}` → literal, not evaluated
- Visual editor generates the most appropriate form

---

### OQ-2: Struct Mutability — RESOLVED

**Verdict**: Mutable by default. Optional `frozen: true` for opt-in immutability per-struct.

```jsonc
// Mutable (default) — simple, intuitive
{"let": "person", "new": "Person", "with": {"name": "'Alice'", "age": "30"}},
{"set": "person.age", "expr": "person.age + 1"}
// person.age is now 31

// Frozen (opt-in) — for config/value objects
{
  "structs": {
    "Config": {
      "frozen": true,
      "fields": {"db_host": "string", "db_port": "int"}
    }
  }
}
// {"set": "config.db_host", ...} → COMPILE ERROR: cannot mutate frozen struct
```

**Rules**:
- Mutable by default — `set "struct.field"` works
- `frozen: true` → compile error on any mutation attempt
- Functions have isolated scope — structs are copied when passed to functions
- Code generation to functional languages: transpiler auto-transforms to copy-on-write

---

### OQ-3: Function Call Duality — RESOLVED

**Verdict**: Both expression-level and step-level calls are valid. Functions callable at both levels.

| Use | When |
|---|---|
| Expression-level `func(args)` | Pure functions, simple args, inside `expr`/`if`/`while` conditions |
| Step-level `call` + `with` | Complex computed input, side-effect calls, calls needing error handling |

```jsonc
// Expression-level — concise
{"let": "discount", "expr": "calculateDiscount(item.price, item.qty, 'gold')"}
{"if": "isValid(email) && isUnique(email)", "then": [...]}
{"let": "result", "expr": "input.name | trim() | upper()"}

// Step-level — explicit
{"let": "result", "call": "processOrder", "with": {
  "items": "filter(cart.items, .quantity > 0)",
  "total": "sum(cart.items, .price * .quantity)"
}}
```

**Param mapping**: Parameter order in function definition = positional order in expression calls.

```jsonc
// Definition
"createUser": {
  "params": {"name": "string", "age": "int"},  // name=pos0, age=pos1
  ...
}

// Expression call: positional
{"let": "u", "expr": "createUser('Alice', 30)"}

// Step call: named
{"let": "u", "call": "createUser", "with": {"name": "'Alice'", "age": "30"}}
```

---

### OQ-4: Error Types — RESOLVED

**Verdict**: Both string and structured. Auto-normalize to structured. Catch always receives normalized error object.

```jsonc
// Throw — string shorthand
{"error": "'Invalid email format'"}

// Throw — structured
{"error": {
  "code": "'VALIDATION_ERROR'",
  "message": "'Invalid email format'",
  "details": "[{'field': 'email', 'error': 'invalid format'}]"
}}

// Catch — always receives normalized object
{"try": [...],
 "catch": {
   "as": "err",
   "steps": [
     {"if": "err.code == 'VALIDATION_ERROR'", "then": [...]},
     {"log": "'Error: ' + err.message"}
   ]
 }}
```

**Normalization rules**:

| Throw form | Internal representation |
|---|---|
| `{"error": "'msg'"}` | `{code: "ERROR", message: "msg", details: nil, step: N, stack: [...]}` |
| `{"error": {"code": "'X'", "message": "'msg'"}}` | `{code: "X", message: "msg", details: nil, step: N, stack: [...]}` |
| `{"error": {"code": "'X'", "message": "'msg'", "details": "data"}}` | `{code: "X", message: "msg", details: data, step: N, stack: [...]}` |

**Error object fields**:
- `err.code` — string error code (default: `"ERROR"`)
- `err.message` — human-readable message
- `err.details` — any type (object, array, string, nil)
- `err.step` — step number where error occurred
- `err.stack` — call stack trace array

---

### OQ-5: Parallel Error Handling — RESOLVED

**Verdict**: Configurable with 3 modes. Default: `cancel_all`. Implementation uses Go context cancellation.

```jsonc
{
  "parallel": {
    "fetch_user": [{"let": "user", "call": "fetchUser", "with": {"id": "input.id"}}],
    "fetch_orders": [{"let": "orders", "call": "fetchOrders", "with": {"user_id": "input.id"}}]
  },
  "on_error": "cancel_all",
  "into": "results"
}
```

**Three modes**:

| Mode | Branch success | Branch error | Overall |
|---|---|---|---|
| `"cancel_all"` (default) | Result value | Cancel others, propagate error | Error thrown to caller |
| `"continue"` | Result value | `nil` (error logged) | Success, partial results |
| `"collect"` | Result value | Error object as value | Success, user checks per-branch |

**Implementation**: Each parallel branch gets a Go `context.Context` derived from parent. On `cancel_all`, context is cancelled, aborting in-flight I/O operations via expr-lang `WithContext`.

---

### OQ-6: Namespace / Module Namespace — RESOLVED

**Verdict**: Import alias IS the namespace. No explicit namespace declaration. Barrel files for deep hierarchies.

```jsonc
{
  "import": {
    "crm": "./modules/crm/index.json",
    "hrm": "./modules/hrm/index.json",
    "bc": "ext:bitcode",
    "http": "io:http",
    "fs": "io:fs"
  },
  "steps": [
    {"let": "c", "new": "crm.Contact", "with": {...}},
    {"let": "h", "new": "hrm.Contact", "with": {...}},
    {"let": "resp", "call": "http.get", "with": {"url": "'https://api.example.com'"}}
  ]
}
```

**Rules**:
- Import alias = namespace prefix for all exported symbols
- No explicit `"namespace"` declaration inside files
- File identity determined by import alias, not by file content
- I/O modules imported explicitly: `"io:http"`, `"io:fs"`, `"io:sql"`, `"io:exec"`
- Extensions imported explicitly: `"ext:bitcode"`, `"ext:eval"`
- Barrel files (`index.json`) for re-exporting from deep hierarchies

---

### OQ-7: Stdlib Function Calls — RESOLVED

**Verdict**: Flat by default (follow expr-lang). Pipe operator for chaining. Namespace only for module-level grouping.

```jsonc
// Flat (expr-lang native) — most functions
{"let": "result", "expr": "upper(trim(input.name))"}
{"let": "total", "expr": "sum(items, .price)"}
{"let": "id", "expr": "randomInt(1000, 9999)"}

// Pipe (expr-lang native) — for chaining
{"let": "result", "expr": "input.name | trim() | upper() | split(' ')"}

// Namespaced — only for module-level grouping
{"let": "hash", "expr": "crypto.sha256(input.password)"}
{"let": "id", "expr": "crypto.uuid()"}
{"let": "valid", "expr": "regex.match(email, '^[a-z]+@[a-z]+\\.[a-z]+$')"}
```

**Rules**:
- expr-lang built-ins → flat, no namespace
- go-json additions that are unique → flat: `clamp()`, `randomInt()`, `sign()`
- go-json additions that could conflict → namespaced: `crypto.*`, `regex.*`
- Do NOT namespace `string.*` or `math.*` — keep flat

---

### OQ-8: Undefined/Untyped Variables — RESOLVED

**Verdict**: Strict after first assignment (default). `any` for explicit dynamic. `?T` for nullable.

```jsonc
// Default: strict after first assignment
{"let": "x", "value": 42},
{"set": "x", "value": "hello"}
// ← COMPILE ERROR: cannot assign string to int variable 'x'

// Explicit dynamic
{"let": "x", "type": "any", "value": 42},
{"set": "x", "value": "hello"}
// ← OK, x is any

// Nullable
{"let": "name", "type": "?string", "value": "Alice"},
{"set": "name", "value": null}
// ← OK, ?string accepts null

// Non-nullable
{"let": "name", "value": "Alice"},
{"set": "name", "value": null}
// ← COMPILE ERROR: cannot assign nil to non-nullable string
```

**Type inference rules**:
- `{"let": "x", "value": 42}` → `x` inferred as `int`
- `{"let": "x", "value": "hello"}` → `x` inferred as `string`
- `{"let": "x", "expr": "fromJSON(payload)"}` → `x` inferred as `any` (fromJSON returns any)
- `{"let": "x", "expr": "if cond { 42 } else { 'hello' }"}` → `x` inferred as `any` + compiler WARNING: "expression returns mixed types"
- `{"let": "x", "type": "int", "value": 42}` → `x` explicitly `int`

---

### OQ-9: Eval / Inline Code Execution — RESOLVED

**Verdict**: Extension only, NOT built-in. Zero dependency on script runtimes.

```go
// Host (bitcode) injects eval capability
gojson.WithExtension("eval", Extension{
    Functions: map[string]any{
        "js":     func(code string, ctx map[string]any) (any, error) { ... },
        "go":     func(code string, ctx map[string]any) (any, error) { ... },
        "python": func(code string, ctx map[string]any) (any, error) { ... },
    },
})
```

```jsonc
// Usage (only if host provides eval extension)
{
  "import": {"eval": "ext:eval"},
  "steps": [
    {"let": "result", "call": "eval.js", "with": {
      "code": "'return items.map(x => x.name).join(\", \")'",
      "ctx": "{'items': items}"
    }}
  ]
}
```

**Rationale**:
- Keeps go-json standalone (zero dependency on yaegi/goja/python)
- Prevents code injection attacks at language level
- Preserves code generation capability (eval blocks can't be transpiled)
- Preserves static analysis (compiler can't type-check eval content)
- Host applications opt-in to eval risk

---

## 4. Blind Spot Solutions

### BS-1: Stdlib Overlap with expr-lang

**Problem**: Draft designed ~90 stdlib functions. ~50 already built-in in expr-lang.

**Solution**: 3-layer stdlib architecture (see F-4 above). Reduces implementation from ~90 functions to ~28 functions. Follow expr-lang naming conventions.

---

### BS-2: Debugging & Observability

**Problem**: No mechanism for debugging go-json programs. Critical for lowcode platform.

**Solution**: Design VM with debug hooks from the start.

#### Execution Trace

```jsonc
// Trace output (enabled via runtime option)
{
  "trace": [
    {"step": 0, "type": "let", "var": "x", "value": 42, "duration_us": 5},
    {"step": 1, "type": "let", "var": "y", "expr": "x + 1", "value": 43, "duration_us": 3},
    {"step": 2, "type": "if", "condition": "y > 40", "result": true, "duration_us": 2},
    {"step": 3, "type": "return", "value": {"grade": "A"}, "duration_us": 1}
  ],
  "total_steps": 4,
  "total_duration_us": 11,
  "variables_final": {"x": 42, "y": 43}
}
```

#### Debugger Interface

```go
type Debugger interface {
    OnStep(step StepInfo) DebugAction  // Continue, StepOver, StepInto, Pause
    OnVariable(name string, old, new any)
    OnError(err error)
    OnFunctionCall(name string, args map[string]any)
    OnFunctionReturn(name string, result any)
}

type DebugAction int
const (
    Continue DebugAction = iota
    StepOver
    StepInto
    Pause
)

// Runtime option
rt := gojson.NewRuntime(
    gojson.WithDebugger(myDebugger),
    gojson.WithTrace(true),
)
```

**Implementation note**: VM must be designed with debug hooks from Phase 4.5a. Adding debugger support after VM is built requires major refactor.

---

### BS-3: Versioning & Backward Compatibility

**Problem**: No mechanism for handling language evolution.

**Solution**: Version field + deprecation warnings + migration tool.

#### Program Version Declaration

```jsonc
{
  "name": "my_program",
  "go_json": "1",
  "steps": [...]
}
```

`"go_json": "1"` declares the language version this program was written for. Engine applies compatibility shims for older versions.

#### Deprecation Mechanism

```go
// In stdlib registration
stdlib.Register("unique", uniqueFunc, stdlib.Deprecated("Use uniq() instead. Removed in go-json v3."))
```

Compiler emits WARNING (not error) for deprecated functions. Visual editor shows deprecation notice.

#### Migration Tool

```bash
go-json migrate program.json --from v1 --to v2
# Auto-transforms deprecated syntax to new syntax
```

---

### BS-4: Testing Framework

**Problem**: No way for users to test their go-json programs.

**Solution**: Built-in test runner with test file format.

```jsonc
// tests/test_discount.json
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
    },
    {
      "_c": "Unknown tier gets 5% discount",
      "call": "calc.calculateDiscount",
      "with": {"price": "200.0", "quantity": "2", "tier": "'bronze'"},
      "expect": 20.0
    }
  ]
}
```

```bash
go-json test tests/
# ✓ test_discount: Gold tier gets 15% discount (2ms)
# ✗ test_discount: Unknown tier gets 5% discount
#   Expected: 20.0
#   Got: 10.0
# 1 passed, 1 failed
```

**Implementation**: Phase 4.5c. But test file format must be designed now for consistency.

---

### BS-5: Log Step Enhancement

**Problem**: `{"log": "'message'"}` too simple for production use.

**Solution**: Support levels, structured data, and host logger callback.

```jsonc
// Simple (backward compatible)
{"log": "'Processing item: ' + string(item.id)"}

// With level
{"log": "'Slow query detected'", "level": "warn"}

// Structured
{"log": {
  "message": "'Order processed'",
  "level": "info",
  "data": {
    "order_id": "order.id",
    "total": "order.total"
  }
}}
```

**Host logger**:

```go
rt := gojson.NewRuntime(
    gojson.WithLogger(func(level, msg string, data map[string]any) {
        slog.Log(level, msg, data)
    }),
)
```

**Default levels**: `debug`, `info`, `warn`, `error`. Default level if omitted: `info`.

---

### BS-6: Nested `with` Semantics

**Problem**: Undefined behavior for nested objects inside `with`.

**Solution**: Recursive evaluation — all string values at all nesting levels are expressions.

```jsonc
{"let": "order", "with": {
  "id": "generateId()",
  "customer": {
    "name": "input.customer_name",
    "email": "input.customer_email"
  },
  "metadata": {
    "created_at": "now()",
    "created_by": "session.user_id"
  }
}}
// customer.name = evaluated expression input.customer_name
// metadata.created_at = evaluated expression now()
```

**Rules**:
- Every string value in `with` at any nesting level = expression
- Non-string values (numbers, booleans, null) = literal
- Arrays in `with` = each element follows same rules
- For literal strings, wrap in single quotes: `"mode": "'production'"`
- Consistent with top-level `with` behavior

---

### BS-7: Concurrency Safety / Program Reuse

**Problem**: Can compiled programs be reused across concurrent executions?

**Solution**: Compile once, run many. Immutable Program + new scope per execution.

```go
// Compile once at startup
program, err := gojson.Compile(programJSON)

// Run concurrently — each run gets its own scope
go func() { result, err := rt.Execute(program, input1) }()
go func() { result, err := rt.Execute(program, input2) }()
go func() { result, err := rt.Execute(program, input3) }()
```

**Rules**:
- `Program` struct is immutable after compilation
- Each `Execute` call creates new scope (no shared state)
- expr-lang compiled programs are already concurrent-safe
- Runtime maintains program cache (compile once, cache by hash, reuse)

```go
type Runtime struct {
    cache map[string]*Program  // program cache by content hash
    mu    sync.RWMutex
}
```

---

## 5. Hard Problem Solutions

### HP-1: Parallel + Shared State Race Condition

**Problem**: Parallel branches writing to same parent variable = race condition.

**Solution**: Parallel branches CANNOT write to parent scope. Compile error enforced.

```jsonc
// This is a COMPILE ERROR:
{"let": "counter", "value": 0},
{"parallel": {
  "a": [{"set": "counter", "expr": "counter + 1"}],
  "b": [{"set": "counter", "expr": "counter + 1"}]
}}
// Error: parallel branch "a" cannot mutate parent variable "counter".
// Use "into" to collect branch results instead.
```

**Rules**:
- Parallel branches CAN read parent scope (read-only) ✅
- Parallel branches CANNOT write parent scope (compile error) ❌
- Parallel branches output via `"into"` only ✅
- Each branch has its own isolated scope for local variables ✅

---

### HP-2: Circular Import Detection

**Problem**: `a.json` imports `b.json`, `b.json` imports `a.json` = deadlock.

**Solution**: Standard cycle detection in import graph. Compile error with clear message.

```
Compile error: circular import detected.
  a.json → b.json → a.json
Remove one of these imports to break the cycle.
```

**Implementation**: Build directed dependency graph during compilation. Detect cycles via DFS. Report full cycle path in error message.

---

### HP-3: Recursive Function Stack Overflow

**Problem**: Infinite recursion exhausts stack.

**Solution**: `MaxDepth` limit with clear error message including call stack.

```
Runtime error: maximum call depth (100) exceeded.
Call stack: main → ping → pong → ping → pong → ... (100 levels)
Hint: Check for infinite recursion between "ping" and "pong".
```

**Implementation**: Each function call increments depth counter. When `MaxDepth` exceeded, runtime error with full call stack trace. Mutual recursion detected by same mechanism.

---

### HP-4: `value` vs `expr` Ambiguity

**Problem**: Is `{"let": "x", "value": "hello"}` a literal string or variable reference?

**Solution**: Clear, unambiguous rules. `value` = always literal. `expr` = always expression.

| Step | `x` equals | Why |
|---|---|---|
| `{"let": "x", "value": 42}` | `42` (int) | JSON number literal |
| `{"let": "x", "value": "hello"}` | `"hello"` (string) | JSON string literal |
| `{"let": "x", "value": true}` | `true` (bool) | JSON boolean literal |
| `{"let": "x", "value": null}` | `nil` | JSON null literal |
| `{"let": "x", "value": [1, 2]}` | `[1, 2]` (array) | JSON array literal |
| `{"let": "x", "value": {"a": 1}}` | `{"a": 1}` (map) | JSON object literal |
| `{"let": "x", "expr": "hello"}` | value of variable `hello` | Expression evaluated |
| `{"let": "x", "expr": "'hello'"}` | `"hello"` (string) | String literal in expression |
| `{"let": "x", "expr": "1 + 2"}` | `3` (int) | Expression evaluated |

**Rule**: Only ONE of `value`/`expr`/`with` allowed per step. Multiple = compile error.

**For visual editor**: Editor knows the mode and generates correct JSON. User never sees this ambiguity.

---

## 6. Updated Architecture Overview

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
│   ├── types.go                  # Type system (gradual, strict-after-assignment)
│   ├── errors.go                 # Error types with position info + enrichment
│   ├── expr_engine.go            # ExprEngine interface + ExprLangEngine impl
│   ├── program.go                # Immutable compiled Program (concurrent-safe)
│   └── debugger.go               # Debugger interface + execution trace
├── stdlib/                       # Layer 2 only — go-json additions
│   ├── math.go                   # ~7 functions (clamp, sign, randomInt, etc.)
│   ├── strings.go                # ~5 functions (padLeft, padRight, substring, etc.)
│   ├── arrays.go                 # ~3 functions (chunk, zip, range)
│   ├── maps.go                   # ~4 functions (has, merge, pick, omit)
│   ├── crypto.go                 # ~4 functions (sha256, md5, uuid, hmac)
│   ├── regex.go                  # ~3 functions (match, findAll, replace)
│   └── fmt.go                    # ~2 functions (sprintf, numberFormat)
├── io/                           # I/O extensions (Phase 4.5c)
│   ├── http.go
│   ├── fs.go
│   ├── sql.go
│   └── exec.go
├── runtime/                      # Runtime configuration
│   ├── runtime.go                # Runtime struct with program cache
│   ├── limits.go                 # Resource limits (MaxDepth, MaxSteps, Timeout, MaxNodes)
│   ├── context.go                # Execution context (session, metadata)
│   ├── logger.go                 # Logger interface + default impl
│   └── hooks.go                  # Extension hooks (WithExtension, WithIO, etc.)
├── cmd/                          # CLI (Phase 4.5c)
│   └── go-json/
│       └── main.go               # go-json run/check/test/ast/codegen/migrate
├── codegen/                      # Code generation (Phase 4.5c)
│   ├── golang.go                 # AST → Go code (with _c as comments)
│   ├── javascript.go             # AST → JavaScript
│   └── python.go                 # AST → Python
└── testdata/                     # Test programs
    ├── hello.json
    ├── hello.jsonc
    ├── factorial.json
    └── ...
```

### Key Architectural Changes from Original Draft

| Component | Original Draft | Updated |
|---|---|---|
| `preprocess.go` | Not present | NEW — JSONC → JSON strip |
| `expr_engine.go` | Direct expr-lang calls | NEW — Abstraction layer with caching |
| `program.go` | Not explicit | NEW — Immutable, concurrent-safe Program |
| `debugger.go` | Not present | NEW — Debug hooks + execution trace |
| `stdlib/` | ~90 functions | ~28 functions (rest from expr-lang) |
| `runtime/logger.go` | Not present | NEW — Structured logging |
| `codegen/` | Not present | NEW — Code generation with _c → comments |

---

## 7. Implementation Priority

### Must Be Right From Day 1 (Phase 4.5a)

These cannot be added later without major refactor:

| Item | Why |
|---|---|
| VM debug hooks | VM architecture must support step callbacks from the start |
| Immutable Program struct | Concurrency model depends on this |
| Scope isolation model | Parallel safety depends on this |
| Type inference architecture | Gradual typing needs inference engine from the start |
| ExprEngine abstraction | All expression evaluation goes through this |
| JSONC pre-processor | Parser pipeline must include this step |
| `_c` metadata in AST | AST node structure must include metadata field |

### Can Be Added Later

| Item | Phase |
|---|---|
| Testing framework | 4.5c |
| Code generation | 4.5c |
| Migration tool | 4.5c |
| Expr Editor integration | Post-4.5c |
| `decimal` type | Future |
| REPL | Future |

---

## 8. Decision Log

| # | Decision | Date | Rationale |
|---|---|---|---|
| D-01 | Hybrid JSON/JSONC format | 2026-04-28 | Best of both worlds. Strict JSON for machines, JSONC for humans |
| D-02 | `.json` and `.jsonc` both supported | 2026-04-28 | Parser doesn't care — always strips comments |
| D-03 | `_c` for semantic comments | 2026-04-28 | Stored in AST, emitted in codegen. `//` for cosmetic |
| D-04 | expr-lang/expr as expression engine | 2026-04-28 | Proven, fast, rich built-ins, type-safe, concurrent-safe |
| D-05 | ExprEngine abstraction layer | 2026-04-28 | Testability, future-proofing, error enrichment |
| D-06 | 3-layer stdlib (don't duplicate expr-lang) | 2026-04-28 | Reduces scope from ~90 to ~28 functions |
| D-07 | Follow expr-lang naming conventions | 2026-04-28 | Consistency. `hasPrefix` not `startsWith`, `uniq` not `unique` |
| D-08 | OQ-1: Overloaded return (expr + with + value) | 2026-04-28 | Consistent with let/set pattern |
| D-09 | OQ-2: Mutable by default + optional frozen | 2026-04-28 | Simple for lowcode. frozen for safety when needed |
| D-10 | OQ-3: Both call levels valid | 2026-04-28 | Expression for pure, step for complex |
| D-11 | OQ-4: Both error types, auto-normalize | 2026-04-28 | Simple to throw, rich to catch |
| D-12 | OQ-5: 3 parallel error modes | 2026-04-28 | cancel_all (default), continue, collect |
| D-13 | OQ-6: Import alias = namespace | 2026-04-28 | Simple, no explicit namespace declaration needed |
| D-14 | OQ-7: Flat by default, namespace for modules | 2026-04-28 | Follow expr-lang. crypto.*, regex.* only |
| D-15 | OQ-8: Strict after first assignment | 2026-04-28 | Safe for lowcode. any for explicit dynamic |
| D-16 | OQ-9: Eval as extension only | 2026-04-28 | Zero dependency on script runtimes |
| D-17 | VM debug hooks from day 1 | 2026-04-28 | Cannot retrofit. Must be in initial architecture |
| D-18 | Immutable Program, compile-once-run-many | 2026-04-28 | Production concurrency requirement |
| D-19 | Parallel branches read-only parent scope | 2026-04-28 | Eliminates race conditions by design |
| D-20 | Recursive with evaluation | 2026-04-28 | Consistent — all string values in with = expression |
| D-21 | go_json version field in programs | 2026-04-28 | Backward compatibility mechanism |
| D-22 | Built-in test runner | 2026-04-28 | First-class testing for lowcode programs |
| D-23 | Structured logging with levels | 2026-04-28 | Production-grade observability |

---

## 9. Deep Discussion: Struct Methods

### 9.1 Method vs Function — Convention

| Use Method | Use Function |
|---|---|
| Operations **intrinsic** to struct (validate, format, compute derived value) | Operations involving **multiple structs** or **external data** |
| Needs access to `self` fields | Doesn't need `self` |
| Mutation (`set "self.*"`) | Pure transformation |
| Called as `person.greet()` | Called as `greetPerson(person)` |

### 9.2 Method Call Syntax

Both expression-level and step-level are valid:

```jsonc
// Expression level
{"let": "info", "expr": "person.fullInfo()"}
{"let": "msg", "expr": "person.greet('Hello')"}
{"if": "person.isAdult()", "then": [...]}

// Step level (preferred for mutation)
{"call": "person.birthday"}
{"let": "msg", "call": "person.greet", "with": {"greeting": "'Hello'"}}
```

### 9.3 Method Chaining

Supported naturally — if method returns a struct, next method can be called on it:

```jsonc
{"let": "result", "expr": "person.withName('Bob').withAge(30).fullInfo()"}
```

No special handling needed — expr-lang handles this.

### 9.4 `self` Scope Rules

- `self` available in all steps within a method
- `self.field` — access field
- `self.method()` — call another method on same instance
- `self` CANNOT be reassigned: `{"set": "self", ...}` → compile error
- `self` CAN be passed to functions: function receives a **copy** (isolated scope)

```jsonc
"methods": {
  "process": {
    "steps": [
      // Allowed — function gets copy of self
      {"let": "result", "call": "externalValidate", "with": {"person": "self"}}
    ]
  }
}
```

### 9.5 Static Methods / Factory Methods

**NOT in Phase 4.5b.** Use regular functions as factories:

```jsonc
"functions": {
  "personFromInput": {
    "params": {"input": "map"},
    "returns": "Person",
    "steps": [
      {"return": {"new": "Person", "with": {
        "name": "input.name ?? 'Unknown'",
        "age": "int(input.age ?? 0)"
      }}}
    ]
  }
}
```

### 9.6 Methods on Frozen Structs

- Read-only methods: ✅ allowed
- Methods returning new instance: ✅ allowed (doesn't mutate self)
- Mutation methods (`set "self.*"`): ❌ **COMPILE ERROR**

```
Compile error: cannot mutate field 'port' on frozen struct 'Config'.
  Method "setPort" uses {"set": "self.port", ...} but Config is frozen.
  Use a method that returns a new Config instead.
```

### 9.7 Method Overloading

**NOT supported.** JSON keys must be unique. Use optional params instead:

```jsonc
"greet": {
  "params": {
    "name": "string",
    "title": {"type": "?string", "default": "nil"}
  },
  "returns": "string",
  "steps": [
    {"let": "prefix", "expr": "title != nil ? title + ' ' : ''"},
    {"return": "'Hello ' + prefix + name"}
  ]
}
```

### 9.8 Interfaces / Protocols

**NOT in Phase 4.5b.** Duck typing for now. Reserve `"interfaces"` keyword for future.

### 9.9 Struct Methods Decision Log

| # | Decision | Verdict |
|---|---|---|
| D-24 | Method vs Function | Convention-based. Method for intrinsic, function for cross-struct |
| D-25 | Method call syntax | Both expression and step level valid |
| D-26 | Method chaining | Supported naturally via expr-lang |
| D-27 | `self` scope | Available in all method steps. Passable to functions (copy) |
| D-28 | Static methods | NOT in Phase 4.5b. Use regular functions |
| D-29 | Methods on frozen struct | Read-only OK. Mutation → compile error |
| D-30 | Method overloading | NOT supported. Use optional params |
| D-31 | Interfaces | NOT in Phase 4.5b. Reserve keyword |

---

## 10. Deep Discussion: Error Message Quality

### 10.1 Error Message Standard Format

Every go-json error follows this structure:

```
[ERROR_CODE] Error Title
  at step N in function "functionName" (program.json)

  What happened:
    Human-readable description of the problem.

  How to fix:
    Actionable suggestion with code example.

  Context:
    Step: {the JSON step that caused the error}
    Variables: relevant variable values
```

### 10.2 Error Categories

| Category | Code Prefix | When |
|---|---|---|
| Compile errors | `COMPILE_*` | Before execution — syntax, type, structure |
| Runtime errors | `RUNTIME_*` | During execution — nil access, division by zero |
| Limit errors | `LIMIT_*` | Resource limits exceeded |
| I/O errors | `IO_*` | HTTP, FS, SQL failures |

### 10.3 Key Error Features

**"Did you mean" suggestions**: Levenshtein distance matching for variable/function names.

```
[COMPILE_UNDEFINED_VAR] Undefined Variable
  at step 5 in "main" (process_leads.json)

  What happened:
    Variable "user_name" is not defined.

  Did you mean:
    - "username" (defined at step 2)
    - "user_id" (from input)
```

**Fix suggestions**: Actionable code examples.

```
[COMPILE_TYPE_MISMATCH] Type Mismatch
  at step 3 in "main" (process_leads.json)

  What happened:
    Variable "age" is type int, but you're assigning a string value.

  How to fix:
    Use int() to convert: {"set": "age", "expr": "int(input.age_text)"}
    Or declare "age" as type any: {"let": "age", "type": "any", ...}
```

**Variable context**: Show relevant variable values in runtime errors.

```
[RUNTIME_NIL_ACCESS] Nil Property Access
  at step 7 in function "processLead" (process_leads.json)

  What happened:
    Tried to access "address.city" but "address" is nil.

  How to fix:
    Use optional chaining: person?.address?.city
    Or check for nil first: {"if": "person.address != nil", ...}

  Context:
    Step: {"let": "city", "expr": "person.address.city"}
    Variable "person.address" = nil
```

### 10.4 Error Implementation Architecture

```go
type GoJSONError struct {
    Code       string            // "COMPILE_TYPE_MISMATCH"
    Title      string            // "Type Mismatch"
    Category   ErrorCategory     // Compile, Runtime, Limit, IO
    Message    string            // "Variable 'age' is type int..."
    Fix        string            // "Use int() to convert..."
    Suggestion []string          // "Did you mean: username, user_id"
    Step       int               // Step number
    Function   string            // Function name
    Program    string            // Program file name
    Context    map[string]any    // Variables, step JSON, etc.
    Stack      []StackFrame      // Call stack
}

// Three output formats:
func (e *GoJSONError) Error() string    // Full formatted for CLI/logs
func (e *GoJSONError) JSON() map[string]any  // Structured for visual editor
func (e *GoJSONError) Short() string    // One-line for log aggregation
```

### 10.5 Error Enrichment Layer

expr-lang errors are wrapped with human-friendly messages:

```go
func enrichError(err error, ctx EnrichContext) *GoJSONError {
    // 1. Detect error type from expr-lang error message
    // 2. Generate human-friendly message
    // 3. Generate fix suggestion
    // 4. Find similar variable/function names (Levenshtein)
    // 5. Include relevant context (variables, step JSON)
    // 6. Return enriched error
}
```

### 10.6 Visual Editor Integration

Error `JSON()` output enables:
- Highlight the step that errored
- Show inline error message
- Offer quick-fix buttons based on `suggestions`
- Navigate to error location

### 10.7 i18n

**NOT in Phase 4.5a.** English only. Error codes are i18n-ready — each error has unique code that can be mapped to translated messages in future.

### 10.8 Error Message Quality Decision Log

| # | Decision | Verdict |
|---|---|---|
| D-32 | Standard error format | Code + Title + What happened + How to fix + Context |
| D-33 | Error categories | COMPILE_*, RUNTIME_*, LIMIT_*, IO_* |
| D-34 | "Did you mean" suggestions | Yes — Levenshtein distance matching |
| D-35 | Fix suggestions | Yes — actionable code examples |
| D-36 | Variable context in errors | Yes — show relevant variable values |
| D-37 | Structured error output | JSON() method for visual editor |
| D-38 | Error enrichment layer | Wrap expr-lang errors with human-friendly messages |
| D-39 | i18n | NOT in Phase 4.5a. Error codes are i18n-ready |
