# go-json Enhancement Roadmap

**Date**: 15 July 2026
**Status**: Draft (Analysis Complete, Pending Review)
**Scope**: Stdlib expansion, language enhancements, bridge ergonomics, architectural positioning

> This document captures the complete analysis of go-json gaps, proposed enhancements, and architectural decisions regarding go-json's role within the BitCode ecosystem.

---

## Table of Contents

1. [Context & Positioning](#1-context--positioning)
2. [Current State Assessment](#2-current-state-assessment)
3. [Architectural Decisions](#3-architectural-decisions)
4. [Gap Analysis: Stdlib](#4-gap-analysis-stdlib)
5. [Gap Analysis: Language Features](#5-gap-analysis-language-features)
6. [Gap Analysis: I/O & Ecosystem](#6-gap-analysis-io--ecosystem)
7. [Gap Analysis: Testing](#7-gap-analysis-testing)
8. [Lambda Design: `fn(x) => expr`](#8-lambda-design-fnx--expr)
9. [Bridge API Ergonomics](#9-bridge-api-ergonomics)
10. [`env()` Function Design](#10-env-function-design)
11. [DAG as Language Feature](#11-dag-as-language-feature)
12. [Phase Plan](#12-phase-plan)
13. [Appendix: Comparison Tables](#13-appendix-comparison-tables)

---

## 1. Context & Positioning

### 1.1 go-json's Role in BitCode

go-json is one of 5+ script runtimes in BitCode's multi-runtime architecture (see `runtime-engine-redesign-master.md`). Its unique position:

```
Embedded runtimes (zero dependency):
├── go-json     → JSON-native, sandboxed, codegen-able, no-code UI compatible
├── goja        → embedded JS (ES6+), lightest, fastest Go interop
├── quickjs     → embedded JS (ES2023), async/await, modules
└── yaegi       → embedded Go, goroutines, full Go stdlib

External runtimes (ecosystem access):
├── node        → full npm ecosystem (puppeteer, sharp, etc.)
└── python      → full pip ecosystem (pandas, sklearn, etc.)
```

### 1.2 go-json's Unique Advantages Over Other Runtimes

| Advantage | Detail |
|-----------|--------|
| **JSON-native** | Programs are valid JSON — no-code UI can generate go-json, cannot generate JS/Python |
| **Codegen** | go-json programs can be compiled to Go, JavaScript, or Python native code |
| **Sandboxed** | Built-in resource limits (MaxSteps, MaxDepth, Timeout, MaxMemory) — other runtimes lack this |
| **Standalone** | go-json can run as independent web server without BitCode |
| **Embeddable** | Zero external dependencies — any Go app can embed go-json without spawning processes |
| **Deterministic** | No npm version hell or pip conflicts |

### 1.3 Recommended Rule of Thumb

> Start with `go-json` for new processes (JSON-native, codegen-able, sandboxed). If you need JS syntax → `javascript` (goja). If you need npm packages → `node`. If you need pip packages → `python`. If you need goroutines → `go` (yaegi).

### 1.4 Architectural Boundaries

**WHAT stays in BitCode (declarative, configuration):**
- Model definitions (`model.json`) — data structure, not logic
- Workflow definitions (`workflow.json`) — state machine rules, not logic
- View definitions (`view.json`) — UI layout, not logic
- API definitions (`api.json`) — route declarations, not logic
- Agent definitions (`agent.json`) — event/cron triggers, not logic
- Module definitions (`module.json`) — package metadata, not logic
- Process engine orchestration — mixed-runtime routing, DAG scheduling, override resolution, pool management
- Config resolution (`.toml` > `.yaml` > `.env` > OS env) — framework concern

**HOW goes in go-json (imperative, logic):**
- Business logic within workflow transitions (`"process": "confirm_order"`)
- Model lifecycle hooks (before_create, after_update, etc.)
- Custom validation beyond simple eq/neq/required
- Complex computations (pricing, scoring, aggregation)
- Data transformation and enrichment
- Integration orchestration (call APIs, send emails, emit events)

**Bridge connects them:**
- `bc.model(name).*` — CRUD operations on BitCode models
- `bc.email.*`, `bc.notify.*` — communication
- `bc.cache.*`, `bc.storage.*` — infrastructure
- `bc.emit()`, `bc.call()` — event bus, sub-process invocation
- `bc.script(path, data)` — delegate to other runtimes (node/python/goja/yaegi)
- `bc.env()`, `bc.config()` — configuration access
- `bc.session()`, `bc.security.*` — auth context
- `bc.tx.*` — database transactions
- `bc.t()` — i18n translations

### 1.5 Script Runtime Delegation

go-json does NOT embed other runtimes. BitCode handles multi-runtime dispatch:

```jsonc
// go-json delegates to other runtimes via bridge:
{"let": "pdf", "expr": "bc.script('scripts/generate-pdf.ts', data)"}
{"let": "prediction", "expr": "bc.script('scripts/ml-predict.py', features)"}

// BitCode resolves: extension → runtime → pool → timeout → execute
// go-json does not know (or care) which runtime handles the script
```

Convention-based runtime detection (BitCode handles):
- `.json` → go-json
- `.js` → goja (embedded JS)
- `.ts` → node (external, npm)
- `.py` → python (external, pip)
- `.go` → yaegi (embedded Go)

Override via module config or explicit `runtime` field — all invisible to go-json.

---

## 2. Current State Assessment

### 2.1 What go-json Already Has (v0.1.0)

| Category | Count | Details |
|----------|-------|---------|
| Step types | 16 | let, set, if/elif/else, switch, for/in, for/range, while, break, continue, return, call, try/catch/finally, error, log, parallel, _c |
| Built-in functions | 110+ | 68 expr-lang + 42+ stdlib |
| I/O modules | 6 | HTTP, FS, SQL, Exec, MongoDB, Redis |
| Type system | Gradual | string, int, float, bool, []T, map, ?T, any, structs |
| Structs | Full | Fields, defaults, methods, frozen, nested construction |
| Import system | Full | Relative, stdlib, io:, ext:, circular detection, diamond handling |
| Parallel | Full | 3 join modes (all/any/settled), 3 error modes, scope isolation |
| Web server | Full | 5 framework adapters, middleware, auth, JWT, templates, OpenAPI |
| Code generation | Full | Go, JavaScript, Python targets + server codegen |
| Testing | Basic | cases with call/with/expect |
| Safety | Full | MaxSteps, MaxDepth, MaxLoopIterations, Timeout, MaxVariables, MaxVariableSize, MaxOutputSize |
| Debugging | Full | Execution tracing, debugger hooks, stack traces, "did you mean?" |
| Embedding | Full | NewRuntime/Execute Go API, extensions, program cache |
| CLI | 9 commands | run, serve, check, test, ast, codegen, generate, openapi, migrate |
| Tests | 723 | Across 10 packages |

### 2.2 Implicit Variables (Already Available)

| Variable | Available | Example |
|----------|-----------|---------|
| `input.*` | Yes | `input.name`, `input.id` |
| `session.user_id` | Yes | Injected by host via `WithSession()` |
| `session.locale` | Yes | |
| `session.tenant_id` | Yes | |
| `session.groups` | Yes | |
| `execution.id` | Yes | |
| `execution.program` | Yes | |
| `execution.started_at` | Yes | |
| `execution.depth` | Yes | |
| `execution.step_count` | Yes | |

---

## 3. Architectural Decisions

### 3.1 Process Engine Stays in BitCode

The BitCode process engine (17 step types, `{{}}` interpolation, DAG, mixed-runtime dispatch) remains the **orchestration layer**. go-json is one of the runtimes it orchestrates.

**Rationale:**
- Process engine = orchestrator (routes steps to runtimes). go-json = executor (runs logic).
- Mixed-runtime pipelines (node → python → goja → go-json in one process) require a runtime-agnostic orchestrator.
- DAG scheduling (fan-out, fan-in, conditional edges) is infrastructure concern.
- Override resolution (app > module > script > step) is framework convention.
- Pool routing (worker vs background) is infrastructure concern.

**Analogy:** Process engine = Kubernetes. go-json = Container. Kubernetes doesn't run code — it routes, schedules, and manages containers.

### 3.2 Workflow Engine Stays in BitCode

Workflow definitions (states, transitions, permissions) are **declarative configuration**, not programming.

**Rationale:**
- "State X can transition to state Y if user has permission Z" = config, not logic
- go-json handles the **process** triggered by a transition, not the transition rules themselves
- Reimplementing state machine validation in go-json would be verbose and redundant

### 3.3 go-json = Recommended Default for New Processes

For new business logic, go-json should be the recommended runtime because:
- JSON-native (no-code UI compatible)
- Codegen-able (Go/JS/Python output)
- Sandboxed (resource limits built-in)
- Zero external dependency
- More powerful than process engine's built-in steps (functions, recursion, try/catch, parallel, structs)

### 3.4 Script Runners Stay in BitCode

Node.js, Python, goja, yaegi runtimes are **framework concerns**. go-json exposes them via `bc.script()` bridge function.

**Rationale:**
- Embedding Node.js/Python in go-json would explode dependencies
- go-json must remain embeddable (zero external deps)
- `bc.script(path, data)` is sufficient — go-json doesn't need to know which runtime handles it

### 3.5 Config Resolution Stays in BitCode

`.toml` > `.yaml` > `.env` > OS env resolution is framework concern. go-json provides pluggable `env()` function.

**Rationale:**
- Adding Viper (+ TOML/YAML/dotenv parsers) to go-json would bloat dependencies
- go-json standalone only needs `os.Getenv()` (zero dependency)
- BitCode injects Viper resolver via `WithEnvResolver(viper.GetString)`
- Other apps embedding go-json can inject their own resolver

---

## 4. Gap Analysis: Stdlib

### 4.1 String Functions — Currently ~20, Need ~40

**Already have:** `upper`, `lower`, `trim`, `trimPrefix`, `trimSuffix`, `split`, `splitAfter`, `join`, `replace`, `repeat`, `indexOf`, `lastIndexOf`, `hasPrefix`, `hasSuffix`, `padLeft`, `padRight`, `substring`, `format`, `strContains`, `strStartsWith`, `strEndsWith`, `strMatches`

**Missing:**

| Function | Signature | Description | Priority |
|----------|-----------|-------------|----------|
| `capitalize(s)` | `string → string` | "hello world" → "Hello world" | High |
| `title(s)` | `string → string` | "hello world" → "Hello World" | High |
| `camelCase(s)` | `string → string` | "hello_world" → "helloWorld" | High |
| `snakeCase(s)` | `string → string` | "helloWorld" → "hello_world" | High |
| `kebabCase(s)` | `string → string` | "helloWorld" → "hello-world" | High |
| `pascalCase(s)` | `string → string` | "hello_world" → "HelloWorld" | Medium |
| `truncate(s, max, suffix?)` | `string, int, string? → string` | Truncate with "..." | High |
| `slugify(s)` | `string → string` | "Hello World!" → "hello-world" | High |
| `reverse(s)` | `string → string` | Reverse string characters | Medium |
| `count(s, sub)` | `string, string → int` | Count substring occurrences | Medium |
| `replaceFirst(s, old, new)` | `string, string, string → string` | Replace first occurrence only | Medium |
| `lines(s)` | `string → []string` | Split by newlines | Medium |
| `words(s)` | `string → []string` | Split by word boundaries | Medium |
| `isDigit(s)` | `string → bool` | All characters are digits | Medium |
| `isAlpha(s)` | `string → bool` | All characters are letters | Medium |
| `isAlphaNum(s)` | `string → bool` | All characters are letters or digits | Medium |
| `isEmpty(s)` | `string → bool` | String is "" | Medium |
| `isBlank(s)` | `string → bool` | String is empty or whitespace only | Medium |
| `escapeHTML(s)` | `string → string` | HTML entity encoding | Medium |
| `unescapeHTML(s)` | `string → string` | HTML entity decoding | Medium |

### 4.2 Math Functions — Currently ~15, Need ~28

**Already have:** `abs`, `ceil`, `floor`, `round`, `min`, `max`, `sum`, `mean`, `median`, `clamp`, `sign`, `pow`, `sqrt`, `mod`, `randomInt`, `randomFloat`

**Missing:**

| Function | Signature | Description | Priority |
|----------|-----------|-------------|----------|
| `sin(x)` | `float → float` | Sine | Medium |
| `cos(x)` | `float → float` | Cosine | Medium |
| `tan(x)` | `float → float` | Tangent | Medium |
| `asin(x)` | `float → float` | Arc sine | Low |
| `acos(x)` | `float → float` | Arc cosine | Low |
| `atan(x)` | `float → float` | Arc tangent | Low |
| `atan2(y, x)` | `float, float → float` | Two-argument arc tangent | Low |
| `log(x)` | `float → float` | Natural logarithm | Medium |
| `log2(x)` | `float → float` | Base-2 logarithm | Low |
| `log10(x)` | `float → float` | Base-10 logarithm | Medium |
| `exp(x)` | `float → float` | e^x | Medium |
| `trunc(x)` | `float → int` | Truncate toward zero | Medium |
| `toFixed(x, decimals)` | `float, int → string` | Format to N decimal places | High |
| `random()` | `→ float` | Random float [0, 1) | Medium |
| `isNaN(x)` | `any → bool` | Check if NaN | Medium |
| `isInfinite(x)` | `any → bool` | Check if infinite | Low |
| `PI` | constant `3.14159...` | Pi constant | Medium |
| `E` | constant `2.71828...` | Euler's number | Low |

### 4.3 Date/Time Functions — Currently ~7, Need ~19

**Already have:** `now()`, `date()`, `duration()`, `timezone()`, `formatDate()`, `addDuration()`, `diffDates()`

**Missing:**

| Function | Signature | Description | Priority |
|----------|-----------|-------------|----------|
| `toUnix(dt)` | `datetime → int` | Unix timestamp (seconds) | High |
| `fromUnix(ts)` | `int → datetime` | Unix timestamp to datetime | High |
| `toISO(dt)` | `datetime → string` | ISO 8601 string | High |
| `startOfDay(dt)` | `datetime → datetime` | 00:00:00 of the day | High |
| `endOfDay(dt)` | `datetime → datetime` | 23:59:59 of the day | High |
| `startOfMonth(dt)` | `datetime → datetime` | First day of month | Medium |
| `endOfMonth(dt)` | `datetime → datetime` | Last day of month | Medium |
| `startOfYear(dt)` | `datetime → datetime` | Jan 1 of year | Low |
| `endOfYear(dt)` | `datetime → datetime` | Dec 31 of year | Low |
| `isWeekend(dt)` | `datetime → bool` | Saturday or Sunday | Medium |
| `isBefore(a, b)` | `datetime, datetime → bool` | a < b | High |
| `isAfter(a, b)` | `datetime, datetime → bool` | a > b | High |
| `daysInMonth(dt)` | `datetime → int` | 28/29/30/31 | Medium |
| `isLeapYear(dt)` | `datetime → bool` | Leap year check | Low |
| `age(birthDate)` | `datetime → int` | Years since date | Medium |

### 4.4 Array Functions — Currently ~30, Need ~44

**Already have (expr-lang + stdlib):** `len`, `first`, `last`, `get`, `take`, `filter`, `map`, `reduce`, `find`, `findIndex`, `findLast`, `findLastIndex`, `count`, `sum`, `all`, `any`, `none`, `one`, `sort`, `sortBy`, `groupBy`, `reverse`, `flatten`, `uniq`, `concat`, `append`, `prepend`, `slice`, `chunk`, `zip`

**Missing:**

| Function | Signature | Description | Priority |
|----------|-----------|-------------|----------|
| `compact(arr)` | `[]any → []any` | Remove nil/falsy values | High |
| `difference(a, b)` | `[]any, []any → []any` | Elements in a not in b | Medium |
| `intersection(a, b)` | `[]any, []any → []any` | Elements in both | Medium |
| `union(a, b)` | `[]any, []any → []any` | Unique elements from both | Medium |
| `includes(arr, item)` | `[]any, any → bool` | Check if array contains item | High |
| `indexOf(arr, item)` | `[]any, any → int` | Find index of item (-1 if not found) | High |
| `fill(n, value)` | `int, any → []any` | Create array of N identical values | Medium |
| `flatMap(arr, fn)` | `[]any, predicate → []any` | Map + flatten | Medium |
| `partition(arr, pred)` | `[]any, predicate → [[]any, []any]` | Split into [matches, non-matches] | Medium |
| `sample(arr, n?)` | `[]any, int? → any` | Random element(s) | Low |
| `shuffle(arr)` | `[]any → []any` | Random order | Low |
| `keyBy(arr, key)` | `[]map, string → map` | Array of objects → map keyed by field | High |
| `drop(arr, n)` | `[]any, int → []any` | Remove first N elements | Medium |
| `dropRight(arr, n)` | `[]any, int → []any` | Remove last N elements | Low |
| `takeRight(arr, n)` | `[]any, int → []any` | Last N elements | Low |

### 4.5 Map/Object Functions — Currently ~10, Need ~18

**Already have:** `has`, `get`, `getIn`, `merge`, `pick`, `omit`, `keys`, `values`, `toPairs`, `fromPairs`

**Missing:**

| Function | Signature | Description | Priority |
|----------|-----------|-------------|----------|
| `deepMerge(a, b)` | `map, map → map` | Recursive merge | High |
| `deepClone(obj)` | `any → any` | Deep copy | High |
| `deepEqual(a, b)` | `any, any → bool` | Deep equality check | High |
| `setIn(obj, path, value)` | `map, string, any → map` | Set nested value by dot-path | Medium |
| `deleteIn(obj, path)` | `map, string → map` | Delete nested key by dot-path | Medium |
| `mapKeys(obj, fn)` | `map, predicate → map` | Transform keys | Medium |
| `mapValues(obj, fn)` | `map, predicate → map` | Transform values | Medium |
| `defaults(obj, defaults)` | `map, map → map` | Fill missing keys from defaults | Medium |

### 4.6 Validation Namespace — Currently 0, Need ~10

**Proposed: `validate.*` namespace**

| Function | Signature | Description | Priority |
|----------|-----------|-------------|----------|
| `validate.isEmail(s)` | `string → bool` | RFC 5322 email check | High |
| `validate.isURL(s)` | `string → bool` | Valid URL check | High |
| `validate.isIP(s)` | `string → bool` | IPv4 or IPv6 | Medium |
| `validate.isUUID(s)` | `string → bool` | UUID v4 format | High |
| `validate.isJSON(s)` | `string → bool` | Valid JSON string | Medium |
| `validate.isNumeric(s)` | `string → bool` | Numeric string | Medium |
| `validate.isAlpha(s)` | `string → bool` | Alpha-only string | Low |
| `validate.isBase64(s)` | `string → bool` | Valid base64 | Low |
| `validate.isHexColor(s)` | `string → bool` | #RGB or #RRGGBB | Low |
| `validate.isCreditCard(s)` | `string → bool` | Luhn algorithm | Low |

### 4.7 Number Formatting — Currently 0, Need ~4

| Function | Signature | Description | Priority |
|----------|-----------|-------------|----------|
| `formatNumber(n, decimals?, sep?, decSep?)` | `float, ... → string` | 1234567.89 → "1,234,567.89" | High |
| `formatBytes(n)` | `int → string` | 1048576 → "1 MB" | Medium |
| `formatPercent(n, decimals?)` | `float, int? → string` | 0.156 → "15.6%" | Medium |
| `ordinal(n)` | `int → string` | 1 → "1st", 2 → "2nd" | Low |

### 4.8 `env()` Function — Currently 0, Need 1

| Function | Signature | Description | Priority |
|----------|-----------|-------------|----------|
| `env(key, default?)` | `string, string? → string` | Read environment variable with optional default | **Critical** |

**Design:** Pluggable `EnvResolver` — default `os.Getenv()`, BitCode injects `viper.GetString()`. Security via `WithEnvAccess()` allow/deny patterns. See [Section 10](#10-env-function-design).

---

## 5. Gap Analysis: Language Features

### 5.1 Lambda / First-Class Functions

**Status:** Not available. expr-lang predicates (`.age > 18`, `# * 2`) work inline but cannot be stored in variables or passed as arguments.

**Proposed syntax:**

```jsonc
// Single expression
{"let": "double", "expr": "fn(x) => x * 2"}

// Multi-statement (expr-lang let chains)
{"let": "transform", "expr": "fn(x) => let y = x * 2; let z = y + 10; z * z"}

// Usage
{"let": "result", "expr": "double(5)"}                    // 10
{"let": "doubled", "expr": "map([1,2,3], double)"}        // [2,4,6]
{"let": "processed", "expr": "map(names, fn(s) => upper(trim(s)))"}
```

**Codegen mapping:**

| go-json | Go | JavaScript | Python |
|---------|-----|-----------|--------|
| `fn(x) => x * 2` | `func(x any) any { return x.(int) * 2 }` | `(x) => x * 2` | `lambda x: x * 2` |
| `fn(x) => let y = x*2; y+10` | `func(x any) any { y := x.(int)*2; return y+10 }` | `(x) => { let y = x*2; return y+10; }` | `def _fn(x): y = x*2; return y+10` |

**Implementation:** go-json detects `fn(` prefix in expression string, parses params + body, creates Go `func(...any)(any, error)` with scope capture (snapshot at definition time). ~500-700 lines.

**Scope capture:** Capture at definition (snapshot), not late binding. Consistent with JS arrow functions and Python lambda.

**Higher-order integration:** May need custom `mapFn`/`filterFn` wrappers since expr-lang `map`/`filter` expect predicate syntax, not arbitrary callables. Alternative: register lambda as Go func in scope, test if expr-lang `map(arr, lambdaVar)` works natively.

See [Section 8](#8-lambda-design-fnx--expr) for full design.

### 5.2 Dynamic Function Dispatch (`call_ref`)

**Status:** `call` only resolves function names from `program.Functions` (compile-time). Cannot call a function whose name is stored in a variable.

**Proposed:**

```jsonc
{"let": "fnName", "value": "double"},
{"let": "result", "call": "fnName", "with": {"x": "5"}}  // resolves "fnName" → "double" → call
```

**Implementation:** ~15-20 lines in `callFunctionUnified` — fallback: if name not in `program.Functions`, check if it's a variable containing a string (function name) or a callable `func(...any)(any, error)`.

### 5.3 `sleep` Step

**Status:** Not available. No way to pause execution.

**Proposed:**

```jsonc
{"sleep": 1000}                    // milliseconds (literal)
{"sleep": "delay * 1000"}          // expression
```

**Implementation:** ~10 lines. Must interact with context cancellation (respect Timeout).

### 5.4 `retry` Step

**Status:** Not available. Must manually implement retry loops with try/catch.

**Proposed:**

```jsonc
{
  "retry": {
    "steps": [
      {"let": "resp", "call": "http.get", "args": ["https://api.example.com/data"]}
    ],
    "max": 3,
    "delay": 1000,
    "backoff": "exponential"
  }
}
```

**Implementation:** Wrap try/catch in loop with configurable delay and backoff. ~50-80 lines.

### 5.5 Constants Block

**Status:** Not available. No way to define immutable values.

**Proposed:**

```jsonc
{
  "constants": {
    "MAX_RETRIES": 3,
    "STATUS_ACTIVE": "active",
    "STATUS_INACTIVE": "inactive",
    "TAX_RATE": 0.11
  }
}
```

Constants are injected into scope as immutable variables. `set` on a constant = compile error.

### 5.6 Enum System

**Status:** Not available.

**Proposed:**

```jsonc
{
  "enums": {
    "Status": ["draft", "confirmed", "done", "cancelled"],
    "Priority": {"LOW": 1, "MEDIUM": 2, "HIGH": 3}
  }
}
```

Access: `Status.draft`, `Priority.HIGH`. Type checking: `{"let": "s", "value": "draft", "type": "Status"}`.

### 5.7 `assert` Step

**Status:** Not available.

**Proposed:**

```jsonc
{"assert": "len(items) > 0", "message": "'Items cannot be empty'"}
{"assert": "total >= 0", "message": "'Total must be non-negative'"}
```

Throws `ASSERTION_FAILED` error if condition is false. Useful for debugging and testing.

### 5.8 DAG Step Type

**Status:** Not available. `parallel` covers flat fan-out/fan-in but not arbitrary graph dependencies.

**Proposed:**

```jsonc
{
  "dag": {
    "nodes": {
      "validate":      {"steps": [...]},
      "check_stock":   {"steps": [...]},
      "check_payment": {"steps": [...]},
      "reserve":       {"steps": [...]},
      "notify":        {"steps": [...]}
    },
    "edges": [
      {"from": "validate", "to": "check_stock"},
      {"from": "validate", "to": "check_payment"},
      {"from": "check_stock", "to": "reserve"},
      {"from": "check_payment", "to": "reserve"},
      {"from": "reserve", "to": "notify", "if": "reserve.success"}
    ]
  },
  "into": "results"
}
```

**Note:** `parallel` + imperative code already covers most DAG patterns. This is syntactic sugar for readability, not a capability unlock. Medium priority.

**Implementation:** ~400-600 lines (parser + compiler + VM execution + topological sort + cycle detection + conditional edges + tests).

### 5.9 Features NOT Planned

| Feature | Reason |
|---------|--------|
| Full closures (mutable capture) | Scope lifetime management too complex. Go codegen nightmare. |
| Inheritance | go-json chose composition. Adding inheritance = 3x type system complexity. |
| Generator/yield | Requires coroutines. Go has no native yield. Codegen to Python possible but Go/JS difficult. |
| Full OOP (private fields, polymorphism) | Overkill for JSON-based language. Complexity explosion. |
| Package manager | Scope too large. Import system is sufficient. |
| Interface/Protocol | Deferred. May revisit after lambda + enum are stable. |

---

## 6. Gap Analysis: I/O & Ecosystem

### 6.1 Missing I/O Modules

| Module | Importance | Notes |
|--------|-----------|-------|
| `io:email` / `io:smtp` | High | Almost every app sends email. SMTP client wrapper. |
| `io:cache` | High | In-memory cache with TTL. Useful without Redis. |
| `io:queue` | Medium | Abstract over RabbitMQ/SQS/etc. Complex. |
| `io:websocket` | Medium | Real-time communication. |
| `io:graphql` | Low | GraphQL client. |
| `io:ftp` / `io:sftp` | Low | Legacy integration. |

### 6.2 Note on BitCode Bridge

When go-json runs inside BitCode, many I/O capabilities are already available via bridge:
- `bc.email.*` — email sending
- `bc.cache.*` — cache (memory/Redis)
- `bc.notify.*` — WebSocket notifications
- `bc.storage.*` — file storage (S3/local)

The `io:email` and `io:cache` modules are for **standalone go-json** use (without BitCode).

---

## 7. Gap Analysis: Testing

### 7.1 Current Testing

```jsonc
{
  "test": true,
  "cases": [
    {"call": "math.factorial", "with": {"n": "5"}, "expect": 120}
  ]
}
```

### 7.2 Missing Testing Features

| Feature | Description | Priority |
|---------|-------------|----------|
| `expect_error` | Test that function throws specific error | High |
| `expect_type` | Test return type | Medium |
| `expect_contains` | Partial matching (array contains, string contains) | Medium |
| `expect_match` | Regex matching on result | Medium |
| `before` / `after` | Setup/teardown per test file | High |
| `beforeEach` / `afterEach` | Setup/teardown per test case | Medium |
| `skip` / `only` | Selective test execution | Medium |
| `timeout` per case | Per-test timeout | Medium |
| `describe` / `it` grouping | Nested test organization | Low |
| Table-driven tests | Parameterized test cases | Medium |
| Mock/stub | Mock I/O calls | Low (complex) |
| Benchmark | Performance measurement | Low |
| Coverage reporting | Code coverage | Low |

---

## 8. Lambda Design: `fn(x) => expr`

### 8.1 Syntax

```
fn(x) => x * 2                           // single param, single expression
fn(a, b) => a + b                         // multi param
fn(x) => let y = x * 2; y + 10           // multi-statement (expr-lang let chains)
fn(x) => let a = x + 1; let b = a * 2; b - 3  // complex
fn(x) => x > 0 ? x : -x                  // ternary
```

### 8.2 Implementation Strategy

1. Detect `fn(` prefix in expression string (~20 lines)
2. Parse params (comma-separated identifiers) and body (everything after `=>`) (~50 lines)
3. Create Go `func(...any)(any, error)` wrapper with scope capture (~80 lines)
4. Register in scope as callable (~30 lines)
5. Codegen mapping for Go/JS/Python (~100 lines per target)
6. Tests (~200 lines)

Total: ~500-700 lines.

### 8.3 Scope Capture

Capture at definition time (snapshot of current scope). Not late binding.

```jsonc
{"let": "multiplier", "value": 3},
{"let": "multiply", "expr": "fn(x) => x * multiplier"},
// multiply captures multiplier=3 at this point
{"set": "multiplier", "value": 10},
{"let": "result", "expr": "multiply(5)"}
// result = 15 (not 50) — captured value, not current
```

### 8.4 Codegen

| go-json | Go | JavaScript | Python (single-expr) | Python (multi) |
|---------|-----|-----------|---------------------|----------------|
| `fn(x) => x * 2` | `func(x any) any { return x.(int) * 2 }` | `(x) => x * 2` | `lambda x: x * 2` | `lambda x: x * 2` |
| `fn(x) => let y = x*2; y+10` | `func(x any) any { y := x.(int)*2; return y+10 }` | `(x) => { let y = x*2; return y+10; }` | N/A | `def _fn(x):\n  y = x*2\n  return y+10` |

Python caveat: multi-statement lambdas must be generated as `def` functions.

### 8.5 Higher-Order Function Integration

expr-lang `map`/`filter` use `Predicate: true` — they expect inline predicate syntax. Need to verify if passing a `func(...any)(any, error)` variable works. If not, register custom `mapFn`/`filterFn` that accept callables.

---

## 9. Bridge API Ergonomics

### 9.1 Current (Verbose)

```jsonc
{"let": "users", "expr": "bc.model('user').search({'domain': [['active','=',true]], 'limit': 10})"}
{"let": "_", "expr": "bc.email.send({'to': user.email, 'subject': 'Welcome', 'template': 'welcome', 'data': user})"}
{"let": "pdf", "expr": "bc.script.run('scripts/generate-pdf.js', {'template': 'invoice', 'data': order})"}
```

### 9.2 Proposed (Ergonomic)

```jsonc
// Fluent model API
{"let": "users", "expr": "bc.model('user').where('active', true).limit(10).get()"}
{"let": "user", "expr": "bc.model('user').find(id)"}
{"let": "count", "expr": "bc.model('user').where('age', '>', 18).count()"}
{"let": "_", "expr": "bc.model('order').find(id).update({'status': 'shipped'})"}

// Simplified email
{"let": "_", "expr": "bc.email.template('welcome', user.email, user)"}

// Clean script delegation
{"let": "pdf", "expr": "bc.script('scripts/generate-invoice.ts', order)"}

// Transactions
{"let": "_", "expr": "bc.tx.begin()"}
{"let": "_", "expr": "bc.model('account').find(from_id).update({'balance': from.balance - amount})"}
{"let": "_", "expr": "bc.model('account').find(to_id).update({'balance': to.balance + amount})"}
{"let": "_", "expr": "bc.tx.commit()"}
// On error: bc.tx.rollback() in catch block

// Module-level function exports (modules expose functions, not raw scripts)
{"let": "pdf", "expr": "pdf.generate('invoice', order)"}
{"let": "prediction", "expr": "ml.predict(features)"}
```

### 9.3 Implementation

Bridge API improvements are in **BitCode** (`gojson_adapter.go`), not in go-json. go-json doesn't need changes — only the bridge functions exposed via `ext:bitcode` need redesign.

---

## 10. `env()` Function Design

### 10.1 API

```jsonc
{"let": "host", "expr": "env('DB_HOST')"}                // returns "" if not set
{"let": "port", "expr": "env('PORT', '3000')"}            // with default
{"let": "debug", "expr": "env('DEBUG', 'false') == 'true'"}
```

### 10.2 Convention

Always use UPPERCASE keys — works everywhere (standalone `os.Getenv`, BitCode Viper, Docker, K8s, CI/CD).

### 10.3 Implementation

```go
// stdlib/env.go
func RegisterEnv(r *Registry, resolver EnvResolver) {
    r.Register(expr.Function("env",
        func(params ...any) (any, error) {
            key := params[0].(string)
            if err := checkEnvAccess(key); err != nil {
                return nil, err
            }
            val, err := resolver(key)
            if err != nil { return nil, err }
            if val == "" && len(params) > 1 {
                return params[1], nil
            }
            return val, nil
        },
    ))
}
```

### 10.4 Pluggable Resolver

```go
// Default (standalone): os.Getenv
rt := runtime.NewRuntime()

// BitCode: Viper (resolves .toml > .yaml > .env > OS)
rt := runtime.NewRuntime(
    runtime.WithEnvResolver(viper.GetString),
)

// Custom app: any resolver
rt := runtime.NewRuntime(
    runtime.WithEnvResolver(myConfigLoader.Get),
)
```

### 10.5 Security

```go
rt := runtime.NewRuntime(
    runtime.WithEnvAccess(&runtime.EnvAccessConfig{
        Allow: []string{"PORT", "APP_*", "FEATURE_*"},
        Deny:  []string{"*_SECRET", "*_PASSWORD", "*_KEY"},
    }),
)
```

Program-level declaration (optional):

```jsonc
{
  "name": "my_api",
  "env": ["PORT", "DB_HOST", "APP_*"],
  "steps": [
    {"let": "port", "expr": "env('PORT', '3000')"}
  ]
}
```

---

## 11. DAG as Language Feature

### 11.1 Current: `parallel` Already Covers Most DAG Patterns

```jsonc
// Fan-out + fan-in — already works:
{
  "parallel": {
    "stock":   [{"let": "s", "expr": "http.get('https://api/stock/' + id)"}],
    "payment": [{"let": "p", "expr": "http.get('https://api/payment/' + id)"}]
  },
  "join": "all",
  "into": "checks"
}
// After: checks.stock and checks.payment available
```

### 11.2 Proposed: `dag` Step Type (Syntactic Sugar)

```jsonc
{
  "dag": {
    "nodes": {
      "validate":      {"steps": [...]},
      "check_stock":   {"steps": [...]},
      "check_payment": {"steps": [...]},
      "reserve":       {"steps": [...]},
      "notify":        {"steps": [...]}
    },
    "edges": [
      {"from": "validate", "to": "check_stock"},
      {"from": "validate", "to": "check_payment"},
      {"from": "check_stock", "to": "reserve"},
      {"from": "check_payment", "to": "reserve"},
      {"from": "reserve", "to": "notify", "if": "reserve.success"}
    ]
  },
  "into": "results"
}
```

### 11.3 Assessment

- `parallel` + imperative code already achieves DAG patterns
- `dag` is syntactic sugar for readability, not a capability unlock
- Useful for complex pipelines with many dependencies
- If implemented, BitCode process engine could delegate DAG execution to go-json instead of implementing its own
- **Priority: Medium** — after stdlib + lambda + bridge are solid

---

## 12. Phase Plan

### Phase 4.5f: Stdlib Expansion (2-3 weeks)

| Task | Functions | Effort | Priority |
|------|-----------|--------|----------|
| String functions | capitalize, title, camelCase, snakeCase, kebabCase, truncate, slugify, reverse, count, replaceFirst, lines, words, isDigit, isAlpha, isAlphaNum, isEmpty, isBlank, escapeHTML, unescapeHTML, pascalCase | 3-4 days | High |
| Math functions | sin, cos, tan, log, log10, exp, trunc, toFixed, random, isNaN, isInfinite, PI, E, asin, acos, atan, atan2, log2 | 2 days | High |
| Date functions | toUnix, fromUnix, toISO, startOfDay, endOfDay, startOfMonth, endOfMonth, isWeekend, isBefore, isAfter, daysInMonth, isLeapYear, age, startOfYear, endOfYear | 2-3 days | High |
| Array functions | compact, includes, indexOf, keyBy, difference, intersection, union, fill, flatMap, partition, drop, dropRight, takeRight, sample, shuffle | 3-4 days | High |
| Map functions | deepMerge, deepClone, deepEqual, setIn, deleteIn, mapKeys, mapValues, defaults | 2 days | High |
| validate.* namespace | isEmail, isURL, isUUID, isIP, isJSON, isNumeric, isAlpha, isBase64, isHexColor, isCreditCard | 2 days | High |
| Number formatting | formatNumber, toFixed (if not in math), formatBytes, formatPercent, ordinal | 1 day | Medium |
| `env()` function | env(key, default?) with pluggable resolver + security config | 1 day | **Critical** |
| Tests + docs | All new functions | 2-3 days | High |

### Phase 4.5g: Lambda + Language Core (3-4 weeks)

| Task | Effort | Priority |
|------|--------|----------|
| `fn(x) => expr` (single-expression lambda) | 1 week | High |
| `fn(x) => let..; expr` (multi-statement lambda) | 3-4 days | High |
| `call_ref` (dynamic function dispatch from variable) | 2 days | High |
| Higher-order function helpers (mapFn, filterFn, applyEach) if needed | 3 days | Medium |
| `sleep` step | 1 day | High |
| `retry` step | 2 days | High |
| `assert` step | 1 day | Medium |
| `constants` block | 2 days | Medium |
| Enum system | 3-4 days | Medium |
| Enhanced testing (expect_error, before/after, skip/only, table-driven) | 3-4 days | Medium |
| Codegen updates for lambda (Go/JS/Python targets) | 1 week | High |
| Tests + docs | 3-4 days | High |

### Phase 4.5h: Bridge Ergonomics (1-2 weeks, BitCode side)

| Task | Effort | Priority |
|------|--------|----------|
| Fluent model API (`bc.model(x).where().limit().get()`) | 3-4 days | High |
| `bc.tx.begin/commit/rollback` (transactions) | 2-3 days | High |
| `bc.email.template()` shorthand | 1 day | Medium |
| `bc.script()` clean delegation | 1-2 days | Medium |
| Module-level function exports (`pdf.generate()` instead of `bc.script(path)`) | 2-3 days | Medium |
| Tests + docs | 2 days | High |

### Phase 4.5i: Advanced Features (4-6 weeks, lower priority)

| Task | Effort | Priority |
|------|--------|----------|
| DAG step type | 1-2 weeks | Medium |
| `io:email` module (standalone SMTP) | 3 days | Medium |
| `io:cache` module (standalone in-memory with TTL) | 3 days | Medium |
| Interface/protocol system | 1-2 weeks | Low |
| Generator/iterator pattern | 1-2 weeks | Low |

### Dependency Graph

```
Phase 4.5f (Stdlib)
  │
  ├──► Phase 4.5g (Lambda + Core)
  │    │
  │    └──► Phase 4.5i (Advanced — DAG, io:email, io:cache)
  │
  └──► Phase 4.5h (Bridge Ergonomics — BitCode side, can run in parallel with 4.5g)
```

### Timeline Summary

```
Phase 4.5f (Stdlib)              2-3 weeks    ← START HERE
Phase 4.5g (Lambda + Core)       3-4 weeks    ← biggest impact
Phase 4.5h (Bridge Ergonomics)   1-2 weeks    ← parallel with 4.5g
Phase 4.5i (Advanced)            4-6 weeks    ← lower priority

Total: ~10-15 weeks
```

---

## 13. Appendix: Comparison Tables

### 13.1 go-json vs Other Runtimes (After Enhancements)

| Capability | go-json (enhanced) | goja | node | python |
|------------|-------------------|------|------|--------|
| JSON-native | **Yes** | No | No | No |
| Codegen | **Go/JS/Python** | No | No | No |
| Sandboxed | **Built-in** | No | No | No |
| Standalone server | **Yes** | No | Yes | Yes |
| Zero dependency | **Yes** | Yes | No | No |
| Functions + recursion | **Yes** | Yes | Yes | Yes |
| Lambda | **Yes** (after 4.5g) | Yes | Yes | Yes |
| Try/catch | **Yes** | Yes | Yes | Yes |
| Parallel | **Yes** | No | async | asyncio |
| Structs | **Yes** | Objects | Objects | Classes |
| Typed | **Gradual** | No | No | Optional |
| npm packages | No | No | **Yes** | No |
| pip packages | No | No | No | **Yes** |
| Goroutines | No | No | No | No |

### 13.2 Stdlib Completeness (After Phase 4.5f)

| Category | Python | JavaScript | Lua | go-json (after) |
|----------|--------|-----------|-----|-----------------|
| String | 44+ | 30+ | 15+ | ~40 |
| Math | 50+ | 40+ | 30+ | ~28 |
| Date | 30+ | 40+ | minimal | ~19 |
| Array | 30+ | 30+ | 15+ | ~44 |
| Map | 15+ | 15+ | 10+ | ~18 |
| Regex | 10+ | 10+ | 10+ | 3 |
| Crypto | 50+ | 30+ | minimal | 5 |
| Validation | 20+ (lib) | 20+ (lib) | minimal | ~10 |

### 13.3 Process Engine vs go-json Script

| Use Case | Process Engine | go-json Script |
|----------|---------------|----------------|
| Simple CRUD (3-5 steps) | **Better** — more concise | Verbose |
| Complex logic (functions, recursion) | Cannot do | **Better** |
| Mixed-runtime pipeline | **Required** — orchestrator | Cannot orchestrate |
| DAG with fan-out/fan-in | Built-in | `parallel` covers most cases |
| Error handling (try/catch) | Cannot do | **Better** |
| Typed variables + structs | Cannot do | **Better** |
| i18n in error messages | Built-in `{{t()}}` | Via `bc.t()` |
| Codegen output | Cannot do | **Better** |
| No-code UI generation | Cannot do | **Better** (JSON-native) |
