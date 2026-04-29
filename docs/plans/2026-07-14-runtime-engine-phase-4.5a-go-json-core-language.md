# Phase 4.5a — go-json Core Language + Stdlib (Draft)

**Status**: Approved
**Date**: 28 April 2026
**Design Decisions**: See [brainstorming design doc](./2026-04-28-go-json-brainstorming-design.md) for rationale
**Depends on**: None (foundational)
**Blocked by**: Phase 4.5b, 4.5c, Phase 7

---

## §1. Package Structure

```
packages/go-json/
├── go.mod                    # module github.com/bitcode-framework/go-json
├── lang/
│   ├── ast.go                # AST node types (all step types)
│   ├── preprocess.go         # JSONC → JSON strip (comments, trailing commas)
│   ├── parser.go             # JSON → AST (validate structure, resolve types)
│   ├── compiler.go           # AST → validated Program (type check, limit check)
│   ├── vm.go                 # Tree-walk interpreter
│   ├── scope.go              # Variable scoping (block scope per if/loop/function)
│   ├── types.go              # Type system (gradual: untyped → schema → full)
│   ├── errors.go             # Error types with step position info
│   ├── expr_engine.go        # ExprEngine interface + ExprLangEngine implementation with caching
│   ├── debugger.go           # Debugger interface + execution trace
│   └── program.go            # Program struct (compiled, ready to run)
├── stdlib/
│   ├── registry.go           # Function registry + All() helper
│   ├── math.go               # 7 math functions
│   ├── strings.go            # 5 string functions
│   ├── arrays.go             # 5 array functions
│   └── types.go              # 2 type conversion functions
├── runtime/
│   ├── limits.go             # Resource limits struct + defaults
│   ├── context.go            # Execution context (session, metadata)
│   ├── logger.go             # Logger interface + structured logging
│   └── runtime.go            # NewRuntime(), Execute(), options
└── testdata/
    ├── hello.json
    ├── hello.jsonc
    ├── variables.json
    ├── variables.jsonc
    ├── control_flow.json
    ├── loops.json
    ├── functions.json
    ├── recursion.json
    ├── error_handling.json
    ├── stdlib_test.json
    └── stdlib_test.jsonc
```

---

## §2. Expression Engine — expr-lang/expr

go-json uses [expr-lang/expr](https://github.com/expr-lang/expr) as its expression evaluation engine. This is NOT a custom parser — it's a proven, production-grade library.

### 2.1 Why expr-lang

| Requirement | expr-lang |
|---|---|
| Arithmetic: `+`, `-`, `*`, `/`, `%`, `**` | ✅ |
| Comparison: `==`, `!=`, `<`, `>`, `<=`, `>=` | ✅ |
| Logical: `&&`, `\|\|`, `!`, ternary `?:` | ✅ |
| Nil coalescing: `??` | ✅ |
| String: `contains`, `startsWith`, `endsWith`, `matches` | ✅ |
| Array: `filter`, `map`, `reduce`, `find`, `all`, `any` | ✅ |
| Member access: `a.b.c`, `a[0]`, `a?.b` (optional chaining) | ✅ |
| Pipe: `value \| function` | ✅ |
| Type-safe | ✅ Compile-time type checking |
| Memory-safe | ✅ No buffer overflows |
| Terminating | ✅ No infinite loops in expressions |
| Bytecode VM | ✅ Compiled, fast |
| Custom functions | ✅ `expr.Function()` |
| Go-native | ✅ Pure Go, no CGO |

### 2.2 What expr-lang Does NOT Do (go-json fills the gap)

- Statements (let, set, if/else blocks, loops) → go-json VM
- Functions with multiple steps → go-json function system
- Side effects (I/O, DB) → go-json I/O modules
- Struct definitions → go-json type system
- Import/module system → go-json import system

### 2.3 Abstraction Layer

go-json does not call expr-lang directly from the VM. All expression work goes through an abstraction layer:

```go
type ExprEngine interface {
    Compile(expr string, env map[string]any) (CompiledExpr, error)
    Run(compiled CompiledExpr, env map[string]any) (any, error)
    Eval(expr string, env map[string]any) (any, error)
    Validate(expr string, env map[string]any) error
    ReturnType(expr string, env map[string]any) (string, error)
}

type ExprLangEngine struct {
    cache map[string]*vm.Program
    mu    sync.RWMutex
}
```

**Why this exists:**
- Testability — VM tests can mock expression evaluation.
- Future-proofing — expr-lang can be swapped without rewriting runtime flow.
- Editor support — `Validate()` and `ReturnType()` are needed by visual tooling.
- Performance — `ExprLangEngine` caches compiled expressions and reuses them across executions.

### 2.4 expr-lang Configuration

go-json configures expr-lang explicitly rather than accepting library defaults blindly.

| Option | Purpose in go-json |
|---|---|
| `WithContext("ctx")` | Propagates timeout/cancellation into expression evaluation and custom functions |
| `MaxNodes(n)` | Caps expression complexity to prevent pathological expressions / DoS |
| `Timezone(...)` | Ensures consistent date/time behavior across environments |
| `Function(...)` | Registers go-json stdlib functions with explicit type signatures |
| `Operator(...)` | Reserved for future operator customization such as domain types |
| `Patch(...)` | Enables AST rewrites for security, normalization, and future transforms |
| `ConstExpr(...)` | Allows constant-folding / compile-time optimization for safe pure helpers |
| `DisableBuiltin(...)` | Allows sandboxing or per-program builtin restrictions |

go-json uses `WithContext`, `MaxNodes`, `Timezone`, and `Function` from day 1. `Operator`, `Patch`, `ConstExpr`, and `DisableBuiltin` are part of the abstraction so the runtime can evolve without redesign.

### 2.5 expr-lang Features Available in Expressions

These features are already available inside go-json expressions and should NOT be reimplemented in go-json itself:

| Feature | Example |
|---|---|
| `let` variables inside one expression | `let x = price * qty; x - discount` |
| Pipe operator | `name \| trim() \| upper()` |
| Multiline `if` | `if score >= 90 { 'A' } else { 'B' }` |
| Range | `1..10` |
| Optional chaining | `user?.address?.city` |
| Nil coalescing | `nickname ?? name` |
| Predicates | `filter(items, .price > 100)` |
| `$env` | `'user_id' in $env` |
| Map literals | `{'status': 'ok', 'count': len(items)}` |
| Backtick strings | `` `multi\nline raw string` `` |

---

## §3. Variable Declaration — `let` / `set`

### 3.1 `let` — Declare New Variable

```json
{"let": "name", "value": "Alice"}           // literal string
{"let": "age", "value": 30}                  // literal int
{"let": "scores", "value": [90, 85, 92]}     // literal array
{"let": "active", "value": true}             // literal bool
{"let": "config", "value": {"k": "v"}}       // literal object

{"let": "next_age", "expr": "age + 1"}       // expression
{"let": "greeting", "expr": "'Hello ' + name"}// expression with string concat
{"let": "first", "expr": "scores[0]"}        // expression with array access
{"let": "adult", "expr": "age >= 18"}        // expression with comparison

{"let": "profile", "with": {                  // computed object
  "name": "name",                             //   each value = expression
  "age": "age",
  "is_adult": "age >= 18"
}}
```

**Rules:**
- `let` declares a NEW variable. Error if variable already exists in current scope.
- Exactly one of `value`/`expr`/`with` required. Zero or multiple = compile error.
- Type is inferred from the assigned value.

### 3.2 `set` — Update Existing Variable

```json
{"set": "age", "value": 31}                  // literal
{"set": "age", "expr": "age + 1"}            // expression
{"set": "name", "expr": "upper(name)"}       // expression with function
```

**Rules:**
- `set` updates an EXISTING variable. Error if variable does not exist.
- Type must be compatible with original declaration. `let age = 30` then `set age = "hello"` → type error.

### 3.3 Nested Property Mutation

```json
{"set": "person.address.city", "expr": "'Bandung'"}
{"set": "items[0].name", "expr": "'Updated'"}
```

Dot notation and bracket notation supported for nested mutation.

### 3.4 Value Modes — Complete Rules

| Mode | Key | Semantics | Example |
|---|---|---|---|
| Literal | `value` | JSON value stored as-is. No evaluation. | `"value": 42`, `"value": "hello"`, `"value": [1,2]` |
| Expression | `expr` | String evaluated by expr-lang. Result stored. | `"expr": "age + 1"`, `"expr": "len(items)"` |
| Computed object | `with` | Object where each field's value is an expression string. | `"with": {"name": "input.name", "age": "input.age + 1"}` |

**Why three modes?**
- `value` is unambiguous — `"value": "Alice"` is always the string "Alice", never a variable lookup.
- `expr` is unambiguous — `"expr": "Alice"` is always a variable lookup for `Alice`.
- `with` is unambiguous — every field is an expression, no need for per-field prefix.

This eliminates the `"= "` prefix problem entirely.

---

## §4. Type System — Gradual Typing

### 4.1 Level 1: Inferred + Strict After First Assignment (Default)

No type annotations are required, but variables are **not** dynamically typed by default. Type is inferred from the first assignment and becomes strict after that.

```json
{
  "name": "quick_script",
  "steps": [
    {"let": "x", "value": 42},
    {"let": "y", "expr": "x + 1"},
    {"return": "y"}
  ]
}
```

Examples:

```json
[
  {"let": "x", "value": 42},
  {"set": "x", "value": 100}
]
```

This is valid because `x` was inferred as `int` and stays `int`.

```json
[
  {"let": "x", "value": 42},
  {"set": "x", "value": "hello"}
]
```

This is a compile error:

```
cannot assign string to variable 'x' (type int)
```

`any` is explicit opt-in for dynamic behavior:

```json
[
  {"let": "payload", "type": "any", "value": 42},
  {"set": "payload", "value": "hello"},
  {"set": "payload", "value": {"status": "ok"}}
]
```

Nullable types are also explicit:

```json
[
  {"let": "name", "type": "?string", "value": "Alice"},
  {"set": "name", "value": null}
]
```

But non-nullable inferred variables reject `null`:

```json
[
  {"let": "name", "value": "Alice"},
  {"set": "name", "value": null}
]
```

Compile error:

```
cannot assign nil to non-nullable string variable 'name'
```

### 4.2 Level 2: Input Schema

Input types declared. Compiler validates expressions against schema.

```json
{
  "name": "typed_script",
  "input": {
    "name": "string",
    "age": "int",
    "tags": "[]string",
    "metadata": "any"
  },
  "steps": [
    {"let": "greeting", "expr": "'Hello ' + input.name"},
    {"let": "next_year", "expr": "input.age + 1"},
    {"return": "greeting"}
  ]
}
```

Compiler knows `input.name` is `string`, `input.age` is `int`. Expression `input.name + 1` → compile error.

### 4.3 Level 3: Full Typing (with structs, Phase 4.5b)

All variables, function params, and return types declared.

```json
{
  "input": {"data": "Person"},
  "functions": {
    "greet": {
      "params": {"p": "Person"},
      "returns": "string",
      "steps": [{"return": "p.name + ' (' + string(p.age) + ')'"}]
    }
  }
}
```

### 4.4 Type Vocabulary

| Type | JSON representation | Go equivalent |
|---|---|---|
| `string` | `"string"` | `string` |
| `int` | `"int"` | `int64` |
| `float` | `"float"` | `float64` |
| `bool` | `"bool"` | `bool` |
| `[]T` | `"[]string"`, `"[]int"` | `[]T` |
| `[]any` | `"[]any"` | `[]any` |
| `map` | `"map"` | `map[string]any` |
| `any` | `"any"` | `any` |
| `?T` | `"?string"`, `"?int"` | `*T` (pointer, nullable) |
| Struct | `"Person"` | Generated struct |

### 4.5 Type Inference Rules

```
literal 42          → int
literal 3.14        → float
literal "hello"     → string
literal true        → bool
literal [1, 2, 3]   → []int
literal [1, "a"]    → []any
literal {"k": "v"}  → map
expr "age + 1"      → typeof(age)  (if age is int → int)
expr "name + ' hi'" → string
expr "len(items)"   → int
expr "items[0]"     → element type of items
```

---

## §5. Control Flow — `if` / `elif` / `else` / `switch`

### 5.1 If / Elif / Else

```json
{
  "if": "score >= 90",
  "then": [
    {"let": "grade", "value": "A"}
  ],
  "elif": [
    {
      "condition": "score >= 80",
      "then": [{"let": "grade", "value": "B"}]
    },
    {
      "condition": "score >= 70",
      "then": [{"let": "grade", "value": "C"}]
    }
  ],
  "else": [
    {"let": "grade", "value": "F"}
  ]
}
```

**Rules:**
- `if` value is an expression string evaluated to bool.
- `then` is required.
- `elif` is optional, array of `{condition, then}` objects.
- `else` is optional.
- Variables declared inside `then`/`else` are scoped to that block (block scope).

### 5.2 Switch

```json
{
  "switch": "status",
  "cases": {
    "active": [
      {"call": "process_active", "with": {"id": "id"}}
    ],
    "pending": [
      {"call": "process_pending", "with": {"id": "id"}}
    ],
    "default": [
      {"log": "'Unknown status: ' + status"}
    ]
  }
}
```

**Rules:**
- `switch` value is an expression evaluated to a value.
- `cases` keys are matched against the switch value (string comparison after `string()` coercion).
- `default` case is optional, executed if no match.
- No fallthrough (unlike C/Go). Each case is independent.

---

## §6. Loops — `for` / `while` / `range` / `break` / `continue`

### 6.1 For-each Loop

```json
{
  "for": "item",
  "in": "items",
  "index": "i",
  "steps": [
    {"log": "string(i) + ': ' + string(item)"}
  ]
}
```

- `for` — variable name for current element
- `in` — expression evaluating to array
- `index` — optional, variable name for current index (0-based)
- `steps` — loop body
- `item` and `i` are block-scoped to the loop

### 6.2 While Loop

```json
{
  "while": "count < 100",
  "steps": [
    {"set": "count", "expr": "count * 2"}
  ]
}
```

- `while` — expression evaluated to bool before each iteration
- Protected by `max_loop_iterations` limit (default 10000)

### 6.3 Range Loop

```json
{
  "for": "i",
  "range": [0, 10],
  "steps": [
    {"log": "'Iteration: ' + string(i)"}
  ]
}
```

- `range` — `[start, end]` or `[start, end, step]`
- `range: [0, 10]` → i = 0, 1, 2, ..., 9 (exclusive end)
- `range: [0, 10, 2]` → i = 0, 2, 4, 6, 8

### 6.4 Break / Continue

```json
{
  "for": "item", "in": "items",
  "steps": [
    {"if": "item.invalid", "then": [{"continue": true}]},
    {"if": "item.stop", "then": [{"break": true}]},
    {"call": "process_item", "with": {"data": "item"}}
  ]
}
```

- `break` — exit the innermost loop
- `continue` — skip to next iteration of innermost loop
- Using `break`/`continue` outside a loop → compile error

---

## §7. Functions

### 7.1 Function Definition

```json
{
  "functions": {
    "calculateDiscount": {
      "params": {
        "price": "float",
        "quantity": "int",
        "tier": "string"
      },
      "returns": "float",
      "steps": [
        {"let": "rate", "value": 0.0},
        {"if": "tier == 'gold'", "then": [
          {"set": "rate", "value": 0.15}
        ], "elif": [
          {"condition": "tier == 'silver'", "then": [
            {"set": "rate", "value": 0.10}
          ]}
        ], "else": [
          {"set": "rate", "value": 0.05}
        ]},
        {"return": "price * quantity * rate"}
      ]
    }
  }
}
```

### 7.2 Function Call — Step Level

```json
{"let": "discount", "call": "calculateDiscount", "with": {
  "price": "item.price",
  "quantity": "item.qty",
  "tier": "customer.tier"
}}
```

- `call` — function name (string)
- `with` — computed input object (each value = expression)
- `let` + `call` — store return value in variable
- `call` without `let` — execute for side effects, discard return

### 7.3 Function Call — Expression Level

For simple pure functions, callable directly in expressions:

```json
{"let": "total", "expr": "calculateDiscount(100.0, 5, 'gold')"}
{"if": "isValid(email)", "then": [...]}
```

**Resolved rule:**
- Expression-level: pure functions with simple arguments, especially inside `expr`, `if`, `while`, and other computed contexts.
- Step-level: functions with complex computed input, side effects, or explicit error handling / control-flow needs.

**Parameter mapping rule:** parameter order in function definition = positional order in expression calls.

```json
{
  "functions": {
    "createUser": {
      "params": {
        "name": "string",
        "age": "int"
      },
      "returns": "map",
      "steps": [
        {"return": {"with": {"name": "name", "age": "age"}}}
      ]
    }
  }
}
```

`createUser('Alice', 30)` maps to `name='Alice'`, `age=30`.

### 7.4 Recursion

```json
{
  "functions": {
    "factorial": {
      "params": {"n": "int"},
      "returns": "int",
      "steps": [
        {"if": "n <= 1", "then": [{"return": 1}]},
        {"let": "sub", "call": "factorial", "with": {"n": "n - 1"}},
        {"return": "n * sub"}
      ]
    }
  },
  "steps": [
    {"let": "result", "call": "factorial", "with": {"n": 10}},
    {"return": "result"}
  ]
}
```

**Recursion rules:**
- Each function call creates a new scope (isolated from parent).
- Depth counter is GLOBAL per execution (not per function). A→B→A counts as depth 3.
- Default max depth: 1000. Configurable per program. Hard limit: 10000.
- Infinite recursion → depth limit error with clear stack trace.

### 7.5 Function Scope Isolation

```json
{"let": "x", "value": 10},
{"call": "myFunc", "with": {"a": "x"}},
// x is still 10 here — myFunc cannot modify parent's x
```

Functions receive input via `with`. They CANNOT access parent scope variables directly. This makes functions:
- Testable independently
- Safe from side effects
- Predictable

---

## §8. Error Handling — `try` / `catch` / `finally` / `error`

### 8.1 Try / Catch / Finally

```json
{
  "try": [
    {"let": "data", "call": "fetchData", "with": {"url": "api_url"}},
    {"let": "parsed", "expr": "fromJSON(data)"}
  ],
  "catch": {
    "as": "err",
    "steps": [
      {"log": "'Failed: ' + err.message"},
      {"let": "parsed", "value": null}
    ]
  },
  "finally": [
    {"log": "'Fetch attempt completed'"}
  ]
}
```

**Rules:**
- `try` — array of steps. If any step throws, execution jumps to `catch`.
- `catch.as` — variable name for the error object.
- `catch.steps` — error handling steps.
- `finally` — optional, always executed (success or error).
- Error in `catch` → propagates to parent try/catch or program level.
- Error in `finally` → replaces original error.

### 8.2 Throw Error

```json
{"error": "'Invalid input: name is required'"}
```

Structured errors are also supported:

```json
// Simple (string)
{"error": "'something went wrong'"}

// Structured
{"error": {
  "code": "'VALIDATION_ERROR'",
  "message": "'Invalid email format'",
  "details": "validationErrors"
}}
```

**Auto-normalization rules:**

| Throw form | Normalized runtime error |
|---|---|
| `{"error": "'msg'"}` | `{code: "ERROR", message: "msg", details: nil, step: N, stack: [...]}` |
| `{"error": {"code": "'X'", "message": "'msg'"}}` | `{code: "X", message: "msg", details: nil, step: N, stack: [...]}` |
| `{"error": {"code": "'X'", "message": "'msg'", "details": "data"}}` | `{code: "X", message: "msg", details: data, step: N, stack: [...]}` |

`catch.as` always receives the normalized object shape, regardless of how the error was thrown.

### 8.3 Error Object Shape

```json
{
  "message": "string — human-readable error message",
  "code": "string — error code (optional, 'ERROR' if not set)",
  "details": "any — optional structured payload (object, array, string, nil)",
  "step": "int — step index where error occurred",
  "function": "string — function name (if inside function)",
  "stack": ["string — call stack trace"]
}
```

---

## §9. Return Values

### 9.1 Return Expression

```json
{"return": "result"}
{"return": "age + 1"}
{"return": "nil"}
{"return": "true"}
{"return": "'hello'"}
{"return": "calculateTotal(items)"}
```

`return` value is always an expression string.

### 9.2 Return Computed Object

Current design — overloaded `return`:

```json
// Return expression
{"return": "candidate"}

// Return object literal directly in expression
{"return": "{'status': 'ok', 'count': len(items)}"}

// Return computed object — use "with" sub-key
{"return": {"with": {
  "status": "'eligible'",
  "person": "candidate",
  "count": "len(items)"
}}}

// Return literal object
{"return": {"value": {"status": "ok", "code": 200}}}
```

This follows the same `value`/`expr`/`with` pattern as `let`/`set`:
- `{"return": "expr"}` — shorthand for expression (most common), including object literals
- `{"return": {"expr": "..."}}` — explicit expression
- `{"return": {"value": ...}}` — literal value
- `{"return": {"with": {...}}}` — computed object

### 9.3 Return from Function vs Program

- Return inside function → returns to caller, function scope destroyed.
- Return inside top-level steps → ends program, value becomes program output.
- Return inside loop → exits loop AND function/program (like Go).
- Return inside if/else → exits function/program.

---

## §10. Resource Limits

### 10.1 Limit Types

```go
type Limits struct {
    MaxDepth          int           // max recursion/call depth. Default: 1000, hard: 10000
    MaxSteps          int           // max total step executions. Default: 10000, hard: 100000
    MaxLoopIterations int           // max iterations per single loop. Default: 10000, hard: 100000
    MaxNodes          int           // max expr-lang AST nodes per expression. Default: 1000
    MaxVariables      int           // max variables in scope. Default: 1000
    MaxVariableSize   int           // max single variable size in bytes. Default: 10MB
    MaxOutputSize     int           // max program output size in bytes. Default: 50MB
    Timeout           time.Duration // max execution time. Default: 30s
}
```

### 10.2 Why These Limits (Not Raw Memory)

Per-execution memory limiting is not possible in Go (shared heap, no per-goroutine memory tracking). These limits are **observable proxies** for memory:

| Limit | What it prevents |
|---|---|
| `MaxSteps` | Runaway execution (each step allocates memory) |
| `MaxLoopIterations` | Infinite/huge loops |
| `MaxNodes` | Pathologically complex expressions during expr-lang compilation/evaluation |
| `MaxVariables` | Variable accumulation |
| `MaxVariableSize` | Single huge variable (e.g. 1GB query result) |
| `MaxDepth` | Stack overflow from recursion |
| `Timeout` | Wall-clock safety net |

### 10.3 Config Resolution Order

```
Engine hard limit (non-overridable)
  ↓
Project config (bitcode.toml / go-json.toml)
  ↓
Module config (module.json)
  ↓
Program config (in JSON program)
  ↓
Step config (per-step timeout)
```

**Resolution rule: always take the MOST RESTRICTIVE (minimum).**

If project says `max_depth: 1000` and program says `max_depth: 5000`, effective limit is `1000`. A more specific level can be MORE restrictive but NEVER less restrictive than its parent.

### 10.4 Program-Level Limits

```json
{
  "name": "heavy_process",
  "limits": {
    "max_depth": 100,
    "max_steps": 50000,
    "timeout": "120s"
  },
  "steps": [...]
}
```

### 10.5 Timeout Inheritance for Sub-calls

When process A (timeout 60s) calls function B at t=20s:
- B gets remaining timeout: 40s
- B cannot exceed parent's remaining time
- B's own timeout (if set) is capped to min(B.timeout, remaining)

---

## §11. Stdlib — Layer 2 (go-json Additions)

### 11.0 Three-Layer Architecture

go-json stdlib is split into three layers:

| Layer | Contents | Ownership |
|---|---|---|
| Layer 1 | expr-lang built-ins (~68 functions) | Already provided by expr-lang |
| Layer 2 | go-json additions | Implemented in `packages/go-json/stdlib/` |
| Layer 3 | I/O / host modules | Added in later phases / host integrations |

**Important:** Layer 1 functions are already available in expressions and should **not** be reimplemented in go-json. This includes common helpers like `abs`, `ceil`, `floor`, `round`, `min`, `max`, `sum`, `len`, `upper`, `lower`, `trim`, `split`, `join`, `filter`, `map`, `reduce`, `find`, `groupBy`, `int`, `float`, `string`, and `type`.

All Layer 2 functions are pure (no side effects) and available in expressions unless later promoted into a namespaced module.

### 11.1 Math (7 functions)

| Function | Signature | Description |
|---|---|---|
| `clamp(x, min, max)` | `number, number, number → number` | Clamp to range |
| `pow(base, exp)` | `number, number → number` | Power |
| `sqrt(x)` | `number → float` | Square root |
| `mod(a, b)` | `number, number → number` | Modulo |
| `randomInt(min, max)` | `int, int → int` | Random integer in range |
| `randomFloat(min, max)` | `number, number → float` | Random float in range |
| `sign(x)` | `number → int` | -1, 0, or 1 |

### 11.2 String (8 functions)

| Function | Signature | Description |
|---|---|---|
| `padLeft(s, n, char?)` | `string, int, string? → string` | Pad left |
| `padRight(s, n, char?)` | `string, int, string? → string` | Pad right |
| `substring(s, start, end?)` | `string, int, int? → string` | Substring |
| `format(template, args...)` | `string, ...any → string` | String formatting (`%s`, `%d`, etc.) |
| `matches(s, pattern)` | `string, string → bool` | Regex match (function alias for `s matches p` operator) |
| `contains(s, substr)` | `string, string → bool` | Substring check (function alias for `s contains sub` operator) |
| `startsWith(s, prefix)` | `string, string → bool` | Prefix check (function alias for `s startsWith pre` operator) |
| `endsWith(s, suffix)` | `string, string → bool` | Suffix check (function alias for `s endsWith suf` operator) |

expr-lang also provides `hasPrefix()` and `hasSuffix()` as built-in functions. `startsWith`/`endsWith` are aliases that match the operator naming convention.

### 11.3 Array (5 functions)

| Function | Signature | Description |
|---|---|---|
| `append(arr, item)` | `[]any, any → []any` | Append (returns new array) |
| `prepend(arr, item)` | `[]any, any → []any` | Prepend (returns new array) |
| `slice(arr, start, end?)` | `[]any, int, int? → []any` | Slice |
| `chunk(arr, size)` | `[]any, int → [][]any` | Split array into fixed-size chunks |
| `zip(a, b)` | `[]any, []any → []any` | Pair two arrays element-by-element |

Helpers such as `len`, `first`, `last`, `get`, `concat`, `reverse`, `sort`, `sortBy`, `uniq`, `flatten`, `filter`, `map`, `reduce`, `find`, and `groupBy` are already provided by expr-lang. Note: expr-lang `get(obj, key)` is single-level access only; for deep nested traversal use go-json's `getIn(obj, "a.b[0].c")` (see Phase 4.5b §6.1).

### 11.4 Type Conversion (2 functions)

| Function | Signature | Description |
|---|---|---|
| `bool(x)` | `any → bool` | Convert to bool |
| `isNil(x)` | `any → bool` | Check if nil |

---

## §12. Execution Context

### 12.1 Session (Implicit, Always Available)

```json
// Accessible via session.* in expressions
{
  "session": {
    "user_id": "string",
    "locale": "string",
    "tenant_id": "string",
    "groups": "[]string"
  }
}
```

Usage: `{"if": "session.user_id != nil", "then": [...]}`

Session is provided by the host application (bitcode passes user session, standalone apps pass custom session or empty).

### 12.2 Execution Metadata (Implicit)

```json
{
  "execution": {
    "id": "string — unique execution ID",
    "program": "string — program name",
    "started_at": "datetime",
    "depth": "int — current call depth",
    "step_count": "int — steps executed so far"
  }
}
```

Usage: `{"if": "execution.depth > 50", "then": [{"log": "'Deep recursion warning'"}]}`

### 12.3 Session is Immutable

Session cannot be modified during execution. It is read-only context from the host.

---

## §13. Variable Scoping

### 13.1 Block Scope

Variables declared inside `if`/`else`/`loop`/`function` are scoped to that block.

```json
[
  {"let": "x", "value": 10},
  {
    "if": "true",
    "then": [
      {"let": "y", "value": 20},
      {"log": "string(x + y)"}
    ]
  },
  {"log": "string(y)"}
]
```

Last step → **runtime error**: `variable 'y' not defined` (y was scoped to if-then block).

### 13.2 Outer Variable Access

Inner blocks CAN READ outer variables:

```json
[
  {"let": "x", "value": 10},
  {"if": "true", "then": [
    {"let": "y", "expr": "x + 5"}
  ]}
]
```

This works — `x` is accessible from inner block.

### 13.3 Outer Variable Mutation

Inner blocks CAN MUTATE outer variables via `set`:

```json
[
  {"let": "total", "value": 0},
  {"for": "item", "in": "items", "steps": [
    {"set": "total", "expr": "total + item.price"}
  ]},
  {"return": "total"}
]
```

This works — `set` looks up the scope chain to find `total`.

### 13.4 Function Scope Isolation

Functions do NOT have access to caller's scope:

```json
[
  {"let": "x", "value": 10},
  {"let": "result", "call": "myFunc", "with": {"a": "x"}}
]
```

Inside `myFunc`, only `a` (from input) is available. `x` is NOT accessible. This is intentional — functions are isolated units.

### 13.5 Loop Variable Scoping

Each loop iteration gets a fresh scope for `item`/`index`:

```json
{
  "for": "item", "in": "items",
  "steps": [
    {"let": "processed", "expr": "transform(item)"}
  ]
}
```

`processed` is re-declared each iteration (block scope). No pollution between iterations.

---

## §14. Edge Cases

### 14.1 Execution

| Edge Case | Behavior |
|---|---|
| Empty steps array | Program returns nil |
| Step with unknown type | Compile error: "unknown step type 'xyz'" |
| Expression syntax error | Compile error with position info |
| Division by zero | Runtime error: "division by zero at step N" |
| Array index out of bounds | `get()` returns nil. Direct `arr[i]` → runtime error. |
| Nil property access | `a.b` where a is nil → runtime error. Use `a?.b` for safe access. |
| Variable name collision with stdlib | Compile error: "variable 'len' shadows built-in function" |

### 14.2 Limits

| Edge Case | Behavior |
|---|---|
| MaxSteps exceeded | Runtime error: "step limit (10000) exceeded" |
| MaxDepth exceeded | Runtime error: "call depth limit (1000) exceeded at function 'X'" |
| MaxLoopIterations exceeded | Runtime error: "loop iteration limit (10000) exceeded" |
| Timeout exceeded | Runtime error: "execution timeout (30s) exceeded at step N" |
| MaxVariableSize exceeded | Runtime error: "variable 'X' exceeds size limit (10MB)" |

### 14.3 Type Errors

| Edge Case | Behavior |
|---|---|
| `let` variable already exists | Compile/runtime error: "variable 'x' already declared" |
| `set` variable doesn't exist | Runtime error: "variable 'x' not defined" |
| Type mismatch on `set` | Runtime error: "cannot assign string to variable 'age' (type int)" |
| Wrong argument type to function | Compile error (if typed) or runtime error |
| Wrong number of arguments | Compile error: "function 'X' expects 3 arguments, got 2" |

### 14.5 JSONC Support

go-json supports a hybrid JSON/JSONC source format.

**File extensions:** both `.json` and `.jsonc` are accepted.

**Pre-processing pipeline:** input is first normalized by `lang/preprocess.go`, which strips:
- `//` line comments
- `/* ... */` block comments
- trailing commas

The output of the pre-processor is strict JSON, then parsed by the standard Go JSON parser.

**Comment mechanisms:**

| Mechanism | Meaning |
|---|---|
| `//` / `/* */` | Cosmetic comments only — removed before parsing |
| `_c` inline | Semantic comment attached to a step |
| `_c` standalone | Comment-only step; parser skips execution semantics |
| `_c` multi-line array | Semantic multi-line documentation preserved as metadata |

Examples:

```jsonc
// Cosmetic comment
{"_c": "Inline semantic comment", "let": "x", "value": 42}
{"_c": "=== Phase 2 ==="}
{"_c": ["Business rule A", "Requirement: JIRA-123"], "return": "x"}
```

---

## §15. Implementation Tasks

| # | Task | Effort | Priority |
|---|---|---|---|
| 1 | Create `packages/go-json/` with go.mod | Small | Must |
| 2 | Define AST node types (`lang/ast.go`) | Medium | Must |
| 3 | JSONC pre-processor (`lang/preprocess.go`) | Small | Must |
| 4 | JSON parser → AST (`lang/parser.go`) | Large | Must |
| 5 | Expression engine abstraction + expr-lang implementation (`lang/expr_engine.go`) | Large | Must |
| 6 | Expression integration in compiler/VM (`lang/compiler.go`) | Large | Must |
| 7 | Tree-walk VM (`lang/vm.go`) | Large | Must |
| 8 | Variable scoping (`lang/scope.go`) | Medium | Must |
| 9 | Type system — inference + gradual (`lang/types.go`) | Medium | Must |
| 10 | Error types with position info (`lang/errors.go`) | Small | Must |
| 11 | Debugger interface + trace capture (`lang/debugger.go`) | Medium | Must |
| 12 | Resource limits (`runtime/limits.go`) | Medium | Must |
| 13 | Runtime API — NewRuntime, Execute, options (`runtime/runtime.go`) | Medium | Must |
| 14 | Execution context — session, metadata (`runtime/context.go`) | Small | Must |
| 15 | Structured logger interface + runtime wiring (`runtime/logger.go`) | Small | Must |
| 16 | Stdlib: math Layer 2 additions (7 functions) | Medium | Must |
| 17 | Stdlib: strings Layer 2 additions (5 functions) | Medium | Must |
| 18 | Stdlib: arrays Layer 2 additions (5 functions) | Medium | Must |
| 19 | Stdlib: type conversion Layer 2 additions (2 functions) | Small | Must |
| 20 | Tests: variable let/set/scoping | Medium | Must |
| 21 | Tests: control flow (if/elif/else/switch) | Medium | Must |
| 22 | Tests: loops (for/while/range/break/continue) | Medium | Must |
| 23 | Tests: functions + recursion | Medium | Must |
| 24 | Tests: error handling (try/catch/finally) | Medium | Must |
| 25 | Tests: resource limits + expression node limits | Medium | Must |
| 26 | Tests: JSONC preprocess + `.jsonc` fixtures | Medium | Must |
| 27 | Tests: debugger / trace / logger hooks | Medium | Must |
| 28 | Tests: Layer 2 stdlib functions | Medium | Must |
| 29 | Tests: edge cases (type errors, nil, overflow) | Medium | Must |

### 15.5 Debugging & Observability

The runtime should support execution tracing from the first VM implementation.

**Trace format:**

```json
{
  "trace": [
    {"step": 0, "type": "let", "var": "x", "value": 42, "duration_us": 5},
    {"step": 1, "type": "if", "condition": "x > 0", "result": true, "duration_us": 2},
    {"step": 2, "type": "return", "value": "ok", "duration_us": 1}
  ],
  "total_steps": 3,
  "total_duration_us": 8
}
```

`lang/debugger.go` defines the `Debugger` interface and execution-trace support. Runtime options should include:
- `WithDebugger(debugger)` — step/function/error callbacks
- `WithTrace(true)` — capture execution trace for inspection

### 15.6 Program Reuse & Concurrency

go-json follows a **compile-once, run-many** pattern.

- `Program` is immutable after compilation.
- Each execution gets a fresh scope and execution context.
- Multiple goroutines can run the same compiled program concurrently.
- Runtime maintains a program cache to avoid recompiling identical program content.

This architecture aligns with expr-lang's concurrent-safe compiled programs and is required for production reuse.

### 15.7 Versioning

Programs may declare a language version field:

```json
{
  "name": "my_program",
  "go_json": "1",
  "steps": []
}
```

Versioning rules:
- `go_json` declares the language version the program targets.
- Deprecated syntax/functions should emit warnings first, not immediate hard failures.
- A migration tool should convert old syntax to new syntax across versions.

This provides a compatibility path as the language evolves.

### 15.8 Structured Logging

The `log` step should support both backward-compatible string logging and enhanced structured logging.

```json
{"log": "'Processing order'"}

{"log": {
  "message": "'Order processed'",
  "level": "info",
  "data": {
    "order_id": "order.id",
    "total": "order.total"
  }
}}
```

`runtime/logger.go` defines a `Logger` interface for host-provided structured logging. Runtime option:
- `WithLogger(logger)` — inject structured logger implementation

Default levels: `debug`, `info`, `warn`, `error`.
