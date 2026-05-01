# go-json Language Reference

## Program Structure

A go-json program is a JSON (or JSONC) file with the following top-level shape:

```json
{
  "name": "program_name",
  "go_json": "1",
  "import": {},
  "structs": {},
  "functions": {},
  "input": {},
  "steps": []
}
```

All top-level keys are optional except `name`.

| Shape | Meaning |
|-------|---------|
| Has `steps` | Executable program |
| No `steps` | Library (exports structs and functions only) |
| Has `routes` | Server program |

---

## Step Types

go-json has 16 step types. Every element inside `steps` (or any step-list such as `then`, `else`, loop bodies, etc.) is one of these.

---

### `let` — Declare a New Variable

Creates a new variable in the current scope. Exactly one value mode is required.

**Literal value** — assigned as-is, no evaluation:

```json
{ "let": "x", "value": 42 }
{ "let": "name", "value": "Alice" }
{ "let": "tags", "value": ["a", "b", "c"] }
```

**Expression** — evaluated by the expression engine:

```json
{ "let": "x", "expr": "a + b" }
{ "let": "greeting", "expr": "'Hello, ' + name" }
```

**Computed object** — each field is an expression:

```json
{
  "let": "profile",
  "with": {
    "name": "input.name",
    "adult": "age >= 18"
  }
}
```

**Function call** — assign the return value of a function:

```json
{ "let": "result", "call": "myFunc", "with": { "x": "42" } }
```

**Struct construction** — create a struct instance:

```json
{ "let": "p", "new": "Person", "with": { "name": "'Alice'" } }
```

**Optional type annotation:**

```json
{ "let": "x", "value": 42, "type": "int" }
```

---

### `set` — Update an Existing Variable

Same value modes as `let`. The target can be a dot-path or bracket-notation path into a nested structure.

```json
{ "set": "x", "value": 100 }
{ "set": "x", "expr": "x + 1" }
{ "set": "person.address.city", "expr": "'Jakarta'" }
{ "set": "items[0].name", "expr": "'Updated'" }
```

---

### `if` / `elif` / `else` — Conditional

```json
{
  "if": "score >= 90",
  "then": [
    { "set": "grade", "value": "A" }
  ],
  "elif": [
    {
      "condition": "score >= 80",
      "then": [{ "set": "grade", "value": "B" }]
    },
    {
      "condition": "score >= 70",
      "then": [{ "set": "grade", "value": "C" }]
    }
  ],
  "else": [
    { "set": "grade", "value": "F" }
  ]
}
```

- `elif` and `else` are optional.
- **Block scope:** variables declared inside `then` or `else` are scoped to that block and not visible outside it.

---

### `switch` / `cases` — Multi-way Branching

```json
{
  "switch": "status",
  "cases": {
    "active":   [{ "log": "'User is active'" }],
    "inactive": [{ "log": "'User is inactive'" }],
    "default":  [{ "log": "'Unknown status'" }]
  }
}
```

- **No fallthrough.** Only the matched case executes.
- Cases are matched via string comparison after coercion.
- Use `"default"` for the fallback branch.

---

### `for` / `in` — Iterate an Array

```json
{
  "for": "item",
  "in": "items",
  "index": "i",
  "steps": [
    { "log": "string(i) + ': ' + item.name" }
  ]
}
```

- `"index"` is optional. When provided, it binds the zero-based index.
- Each iteration gets a fresh scope for the loop variable.

---

### `for` / `range` — Iterate a Number Range

```json
{ "for": "i", "range": [0, 10], "steps": [] }
{ "for": "i", "range": [10, 0, -1], "steps": [] }
```

- Range is **`[start, end)`** — start-inclusive, end-exclusive.
- Optional third element is the step value.
- `[0, 10]` produces `0, 1, 2, … 9`.
- `[10, 0, -1]` produces `10, 9, 8, … 1`.

---

### `while` — Conditional Loop

```json
{
  "while": "count < 100",
  "steps": [
    { "set": "count", "expr": "count * 2" }
  ]
}
```

Protected by `MaxLoopIterations` (default 10,000). The loop terminates if the iteration limit is reached.

---

### `break` / `continue` — Loop Control

```json
{ "break": true }
{ "continue": true }
```

- `break` exits the innermost enclosing loop.
- `continue` skips to the next iteration of the innermost loop.

---

### `return` — Return a Value

```json
{ "return": "result" }
{ "return": { "value": 42 } }
{ "return": { "expr": "a + b" } }
{ "return": { "with": { "name": "input.name", "total": "sum(items)" } } }
{ "return": { "new": "Person", "with": { "name": "'Alice'" } } }
```

- In a function, `return` ends execution and sends the value back to the caller.
- At the top level, `return` sets the program output.

---

### `call` — Call a Function

**Fire-and-forget (no return capture):**

```json
{ "call": "processOrder", "with": { "order_id": "input.id" } }
```

**Capture return value:**

```json
{ "let": "result", "call": "calculate", "with": { "x": "10", "y": "20" } }
```

**Call a method on a struct instance:**

```json
{ "call": "person.birthday" }
```

Arguments in `with` are expressions — each value string is evaluated.

#### Three Ways to Call Functions

go-json provides three calling styles. All three work for **all function types** — program functions, struct methods, I/O modules, and extensions.

##### 1. `call` + `with` (object) — Named Expression Args

Each value is an **expression string** matched to parameter names:

```json
{"call": "calculateDiscount", "with": {"price": "input.price", "tier": "input.tier"}}
{"let": "result", "call": "factorial", "with": {"n": "10"}}
{"call": "person.greet", "with": {"greeting": "'Hello'"}}
```

Best for: go-json defined functions where you want **named, self-documenting** arguments.

Note: Named `with` does NOT work for I/O/extension namespace functions (map ordering is not guaranteed). Use array `with` or `args` instead.

##### 2. `call` + `with` (array) — Positional Expression Args

Each element is an **expression string** passed by position:

```json
{"call": "calculateDiscount", "with": ["input.price", "5", "'gold'"]}
{"let": "resp", "call": "http.get", "with": ["url"]}
{"call": "fs.write", "with": ["'./log.txt'", "content"]}
{"call": "redis.set", "with": ["'user:' + id", "userData", "3600"]}
```

Best for: I/O modules and extensions where args are **computed from variables/expressions**.

Note: Strings are expressions — `"content"` means the variable `content`, not the literal string. Use `'...'` for string literals: `"'./log.txt'"`.

##### 3. `call` + `args` — Literal JSON Values

Each element is a **literal JSON value** — no evaluation, no quote wrapping:

```json
{"call": "calculateDiscount", "args": [100.0, 5, "gold"]}
{"call": "fs.write", "args": ["./log.txt", "Hello, World!"]}
{"call": "redis.set", "args": ["user:123", {"name": "Alice", "age": 30}, 3600]}
{"call": "sql.query", "args": ["SELECT * FROM users WHERE age > ?", [18]]}
```

Best for: **Literal data** — strings with special characters, objects, arrays, numbers. No escaping needed. `"Alice"` is the string `Alice`, not a variable lookup.

##### 4. `expr` — Inline Expression

Call any function directly inside an expression:

```json
{"let": "result", "expr": "calculateDiscount(input.price, 5, 'gold')"}
{"let": "resp", "expr": "http.get('https://api.example.com/users')"}
{"let": "hash", "expr": "crypto.sha256(password)"}
{"let": "names", "expr": "users | filter(.active) | map(.name) | sort()"}
```

Best for: **One-liners**, chaining, and using the result in a larger expression.

##### Side-by-Side Comparison

The same operation written all four ways:

**Calling a go-json function:**

```json
{"let": "d", "call": "calculateDiscount", "with": {"price": "input.price", "tier": "'gold'"}}
{"let": "d", "call": "calculateDiscount", "with": ["input.price", "5", "'gold'"]}
{"let": "d", "call": "calculateDiscount", "args": [100.0, 5, "gold"]}
{"let": "d", "expr": "calculateDiscount(input.price, 5, 'gold')"}
```

**Calling an I/O module function:**

```json
{"let": "resp", "call": "http.get", "with": ["url"]}
{"let": "resp", "call": "http.get", "args": ["https://api.example.com/users"]}
{"let": "resp", "expr": "http.get('https://api.example.com/users')"}
```

**Fire-and-forget (no return value needed):**

```json
{"call": "fs.write", "with": ["'./log.txt'", "content"]}
{"call": "fs.write", "args": ["./log.txt", "Hello, World!"]}
{"let": "_", "expr": "fs.write('./log.txt', content)"}
```

**Calling a struct method:**

```json
{"call": "person.birthday"}
{"let": "name", "call": "person.greet", "with": {"greeting": "'Hello'"}}
{"let": "name", "call": "person.greet", "with": ["'Hello'"]}
{"let": "name", "call": "person.greet", "args": ["Hello"]}
{"let": "name", "expr": "person.greet('Hello')"}
```

**Multi-level namespace (extensions):**

```json
{"let": "rows", "call": "bc.db.query", "with": ["'SELECT * FROM users'"]}
{"let": "rows", "call": "bc.db.query", "args": ["SELECT * FROM users"]}
{"let": "rows", "expr": "bc.db.query('SELECT * FROM users')"}
```

##### When to Use Which

| Situation | Recommended | Why |
|-----------|-------------|-----|
| Literal strings with quotes/backticks/markdown | `args` | Zero escaping — `"Don't forget"` just works |
| Passing objects or arrays as data | `args` | `{"name": "Alice"}` is literal, not expression |
| Computed values from variables | `with` (array) | `"input.price * 0.9"` is evaluated |
| Named args for readability | `with` (object) | `{"price": "...", "tier": "..."}` is self-documenting |
| One-liner with chaining | `expr` | `users \| filter(.active) \| map(.name)` |
| Fire-and-forget side effect | `call` + `args` or `with` | No throwaway `let "_"` needed |
| Complex expression with multiple calls | `expr` | `upper(name) + ' (' + string(age) + ')'` |

##### `with` vs `args` — The Key Difference

```json
{"call": "fn", "with": ["name"]}
```
`"name"` is an **expression** — looks up the variable `name` and passes its value.

```json
{"call": "fn", "args": ["name"]}
```
`"name"` is a **literal string** — passes the string `"name"` as-is.

```json
{"call": "fn", "with": ["'Alice'"]}
```
`"'Alice'"` is an **expression** containing a string literal — passes the string `"Alice"`.

```json
{"call": "fn", "args": ["Alice"]}
```
`"Alice"` is a **literal string** — passes the string `"Alice"`.

Both produce the same result, but `args` is cleaner when you have literal data.

`with` and `args` are **mutually exclusive** — using both in the same step is a compile error.

---

### `try` / `catch` / `finally` — Error Handling

```json
{
  "try": [
    { "let": "data", "call": "riskyOperation" }
  ],
  "catch": {
    "as": "err",
    "steps": [
      { "log": "'Error: ' + err.message" },
      { "log": "'Code: ' + err.code" }
    ]
  },
  "finally": [
    { "call": "cleanup" }
  ]
}
```

- `catch` and `finally` are both optional (but at least one should be present).
- The error object bound by `"as"` has the shape:

| Field | Type | Description |
|-------|------|-------------|
| `message` | string | Human-readable error message |
| `code` | string | Error code |
| `details` | any | Additional context |
| `step` | string | Step that threw |
| `stack` | array | Call stack trace |

All errors are auto-normalized into this shape.

---

### `error` — Throw an Error

**Simple (expression evaluated as the message):**

```json
{ "error": "'something went wrong'" }
```

**Structured:**

```json
{
  "error": {
    "code": "'VALIDATION'",
    "message": "'Invalid email'",
    "details": "input.email"
  }
}
```

---

### `log` — Log a Message

**Simple:**

```json
{ "log": "'Processing: ' + item.name" }
```

**Structured:**

```json
{
  "log": {
    "message": "'Order processed'",
    "level": "'info'",
    "data": {
      "order_id": "id",
      "total": "total"
    }
  }
}
```

Log levels: `debug`, `info`, `warn`, `error`.

---

### `parallel` — Parallel Execution

```json
{
  "parallel": {
    "users":  [{ "let": "u", "call": "fetchUsers" }],
    "orders": [{ "let": "o", "call": "fetchOrders" }]
  },
  "on_error": "cancel_all",
  "into": "results"
}
```

- Each named branch runs concurrently.
- Results are collected into the variable named by `"into"` as a map keyed by branch name.
- Each branch gets an **isolated scope**: it can read parent variables but cannot write to them.
- Writing to a parent variable from a parallel branch is a **compile error**.

**Join modes** (`"join"`):

| Mode | Behavior |
|------|----------|
| `all` | (Default) Wait for all branches to complete |
| `any` | First successful branch wins; cancel remaining |
| `settled` | Wait for all branches regardless of errors; errors collected as `{"error": true, "message": "..."}` |

**Error modes** (`"on_error"`, applies when `join` is `all`):

| Mode | Behavior |
|------|----------|
| `cancel_all` | (Default) Cancel all branches on first error, propagate error |
| `continue` | Let remaining branches finish; failed branch = `nil` in results |
| `collect` | Let remaining branches finish; failed branch = error object in results |

When `join` is `settled`, the `on_error` mode is ignored — all branches always run to completion and errors are always collected as objects.

---

### `_c` — Semantic Comment

```json
{ "_c": "This step validates the input" }
```

Skipped during execution. Preserved in the AST for tooling and documentation.

---

## Type System

go-json uses **gradual typing**. Types are inferred by default and locked after first assignment.

### Built-in Types

| Type | Description |
|------|-------------|
| `string` | Text |
| `int` | Integer |
| `float` | Floating-point number |
| `bool` | `true` or `false` |
| `[]T` | Typed array (e.g. `[]string`, `[]int`) |
| `[]any` | Array of mixed types |
| `map` | Key-value map |
| `StructName` | Instance of a defined struct |
| `?T` | Nullable variant of type `T` (e.g. `?string`) |
| `any` | Any type |

### Nullability

- Non-nullable types assigned `nil` produce a **compile error**.
- Use `?T` to declare a variable that may be `nil`.

```json
{ "let": "email", "value": null, "type": "?string" }
```

---

## Expressions & Function Namespacing

All `expr` values and `with` field values are evaluated by the [expr-lang/expr](https://github.com/expr-lang/expr) expression engine. Expressions support arithmetic, comparison, logical operators, ternary, optional chaining (`?.`), nil coalescing (`??`), pipe operator (`|`), and 110+ built-in functions.

### Flat vs Namespaced Functions

go-json functions come in two calling styles:

**Flat functions** — called directly by name. Used for general-purpose utilities:

```
upper("hello")                          // → "HELLO"
len(items)                              // → 5
"hello world" contains "world"          // → true (operator style)
strContains("hello world", "world")     // → true (function style)
clamp(value, 0, 100)                    // → bounded value
filter(users, .active)                  // → active users only
```

**Namespaced functions** — called with a dot-prefix. Used for domain-specific groups:

```
crypto.sha256("hello")            // → hash string
crypto.uuid()                     // → UUID v4
regex.match("abc123", "\\d+")     // → true
regex.replace("hello", "[aeiou]", "*")  // → "h*ll*"
```

**I/O module functions** — namespaced via import alias:

```
http.get("https://api.example.com/data")
fs.read("./config.json")
sql.query("SELECT * FROM users WHERE id = ?", [42])
```

### Why Some Functions Are Namespaced

| Criteria | Flat | Namespaced |
|----------|------|-----------|
| General-purpose, everyone uses it | `len()`, `upper()`, `strContains()` | — |
| Domain-specific, grouped by concern | — | `crypto.*`, `regex.*` |
| Collision risk (name too generic alone) | — | `crypto.sha256` (not just `sha256`) |
| Side effects / I/O | — | `http.*`, `fs.*`, `sql.*` |

For example, `strContains("abc", "b")` is flat because it's a universal utility with no collision risk. Note: `contains` is an expr-lang operator keyword, so the function alias uses the `str` prefix. But `sha256("hello")` is namespaced as `crypto.sha256("hello")` because "sha256" alone is too specific and could collide with user variables.

### How Namespaces Work

Namespaces are not special syntax — they use standard expr-lang member access on maps. A namespace is a `map[string]any` where each value is a function. When you write `crypto.sha256("hello")`, expr-lang resolves it as: variable lookup (`crypto`) → member access (`.sha256`) → function call (`("hello")`).

This means you can create your own namespaces via extensions. See [Built-in Functions](built-in-functions.md#function-namespacing) for the complete namespace reference and [Embedding Guide](embedding-guide.md) for creating custom namespaces.

---

## Functions

Functions are defined under the top-level `"functions"` key.

```json
{
  "functions": {
    "calculateDiscount": {
      "params": {
        "price": "float",
        "quantity": "int",
        "tier": { "type": "string", "default": "'standard'" }
      },
      "returns": "float",
      "steps": [
        { "let": "base", "expr": "price * quantity" },
        {
          "if": "tier == 'premium'",
          "then": [{ "return": { "expr": "base * 0.8" } }]
        },
        { "return": { "expr": "base * 0.95" } }
      ]
    }
  }
}
```

### Parameters

- Each parameter is a name mapped to a type string, or an object with `"type"` and optional `"default"`.
- Default values are expressions.

### Calling Functions

**From a step (with named arguments):**

```json
{ "let": "x", "call": "calculateDiscount", "with": { "price": "100.0", "quantity": "5" } }
```

**Inside an expression (positional arguments):**

```json
{ "let": "x", "expr": "calculateDiscount(100.0, 5, 'premium')" }
```

### Scope Isolation

Functions have **fully isolated scope**. They cannot access the caller's variables. All input must be passed explicitly via `with` (or as positional arguments in expressions).

### Recursion

Recursion is supported. Depth is limited by `MaxDepth` (default 1,000).

---

## Structs

Structs are defined under the top-level `"structs"` key.

```json
{
  "structs": {
    "Person": {
      "fields": {
        "name": "string",
        "age": "int",
        "email": "?string",
        "country": { "type": "string", "default": "'ID'" }
      },
      "methods": {
        "greet": {
          "returns": "string",
          "steps": [
            { "return": "'Hello, ' + self.name" }
          ]
        }
      }
    }
  }
}
```

### Fields

- Each field is a name mapped to a type string, or an object with `"type"` and optional `"default"`.
- Defaults are expressions evaluated at construction time.

### Construction

```json
{ "let": "p", "new": "Person", "with": { "name": "'Alice'", "age": "30" } }
```

Nested construction is supported:

```json
{ "let": "p", "new": "Person", "with": {
  "name": "'Alice'",
  "address": { "new": "Address", "with": { "city": "'Jakarta'" } }
}}
```

Field values are **type-checked at runtime** — assigning a string to an `int` field produces a `TYPE_MISMATCH` error. Nullable fields (`?T`) accept `nil`; non-nullable fields reject it.

### Methods

- Methods are defined inside `"methods"` with the same shape as functions.
- Methods have an implicit `self` variable referring to the struct instance.
- Call a method: `{ "call": "p.greet" }`

### Mutability

- Struct instances are **mutable by default**.
- Set `"frozen": true` on the struct definition to make instances immutable after construction.

### No Inheritance

go-json does not support struct inheritance. Use **composition** — embed one struct as a field of another.

---

## Import System

```json
{
  "import": {
    "models": "./types.json",
    "http": "io:http",
    "bc": "ext:bitcode"
  }
}
```

Each key is a local alias; the value is the import path.

### Path Types

| Prefix | Meaning | Example |
|--------|---------|---------|
| `./` or `../` | Relative file path | `"./types.json"` |
| `stdlib:` | Standard library module | `"stdlib:math"` |
| `io:` | I/O module | `"io:http"` |
| `ext:` | Extension module | `"ext:bitcode"` |

### What Gets Exported

- **Exported:** structs and functions.
- **Not exported:** `steps`, `input`, `limits`.

### Import Resolution

- **Circular imports** are detected at compile time and rejected (direct and indirect cycles).
- **Diamond imports** are handled correctly — each module is loaded exactly once.
- **Alias collisions** are detected — importing two modules that produce the same namespaced name (e.g. two files both exporting `Item` under the same alias) is a compile error.

---

## Variable Scoping

| Rule | Behavior |
|------|----------|
| Block scope | Variables declared inside `if`/`else`/loop bodies are scoped to that block |
| Outer variable read | Inner blocks **can** read variables from enclosing scopes |
| Outer variable mutation | Inner blocks **can** mutate outer variables via `set` |
| Function scope isolation | Functions **cannot** access the caller's scope — input only via `with` |
| Loop variables | Each iteration gets a fresh scope for the loop variable |

---

## Implicit Variables

These variables are available in every program without declaration.

### `input`

The program's input data, provided at invocation. Always a map.

```json
{ "let": "name", "expr": "input.name" }
```

### `session`

Session context for the current execution.

| Field | Type | Description |
|-------|------|-------------|
| `user_id` | string | Current user identifier |
| `locale` | string | Locale code (e.g. `"en-US"`) |
| `tenant_id` | string | Multi-tenant identifier |
| `groups` | []string | User's group memberships |

### `execution`

Metadata about the current execution.

| Field | Type | Description |
|-------|------|-------------|
| `id` | string | Unique execution identifier |
| `program` | string | Program name |
| `started_at` | string | ISO 8601 timestamp |
| `depth` | int | Current call depth |
| `step_count` | int | Steps executed so far |

---

## Resource Limits

| Limit | Default | Hard Max | Description |
|-------|---------|----------|-------------|
| `MaxSteps` | 10,000 | 100,000 | Total steps executed |
| `MaxDepth` | 1,000 | 10,000 | Call stack depth |
| `MaxLoopIterations` | 10,000 | 100,000 | Iterations per loop |
| `MaxNodes` | 1,000 | — | AST nodes in a single program |
| `MaxVariables` | 1,000 | — | Variables in scope |
| `MaxVariableSize` | 10 MB | — | Size of a single variable |
| `MaxOutputSize` | 50 MB | — | Total program output size |
| `Timeout` | 30s | — | Wall-clock execution time |

### Limit Resolution

When limits are configured at multiple levels, the **most restrictive** value wins:

```
engine → project → module → program → step
```

---

## JSONC Support

go-json accepts both `.json` and `.jsonc` files. The preprocessor strips:

- `//` single-line comments
- `/* */` block comments
- Trailing commas

The `_c` step key provides **semantic comments** that are preserved in the AST (unlike stripped JSONC comments).

```jsonc
{
  // This is a JSONC comment (stripped during preprocessing)
  "name": "example",
  "steps": [
    { "_c": "This is a semantic comment (preserved in AST)" },
    { "let": "x", "value": 42 }
  ]
}
```

---

## Lambda Expressions

Lambda expressions create anonymous functions using the syntax `fn(params) => body`.

### Syntax

```jsonc
{"let": "double", "expr": "fn(x) => x * 2"}
{"let": "add", "expr": "fn(a, b) => a + b"}
{"let": "greet", "expr": "fn() => 'Hello!'"}
{"let": "classify", "expr": "fn(age) => age >= 18 ? 'adult' : 'minor'"}
```

### Calling Lambdas

```jsonc
{"let": "result", "expr": "double(5)"}           // 10
{"let": "result", "expr": "add(3, 7)"}           // 10
```

### Inline Lambdas with Higher-Order Functions

```jsonc
{"let": "evens", "expr": "filterFn([1,2,3,4], fn(x) => x % 2 == 0)"}
{"let": "squares", "expr": "mapFn([1,2,3], fn(x) => x * x)"}
{"let": "total", "expr": "reduceFn([1,2,3,4,5], fn(acc, x) => acc + x, 0)"}
```

### Scope Capture (Snapshot)

Lambdas capture variables at **definition time**. Subsequent mutations to captured variables are NOT visible to the lambda:

```jsonc
{"let": "factor", "value": 3},
{"let": "multiply", "expr": "fn(x) => x * factor"},
{"set": "factor", "value": 10},
{"let": "result", "expr": "multiply(5)"}
// result = 15 (captured factor=3, NOT current 10)
```

### Known Limitations

| Limitation | Reason | Workaround |
|-----------|--------|------------|
| No self-recursion | Snapshot capture — lambda not in own env at definition time | Use named `functions` block for recursion |
| No outer scope mutation | Lambdas are pure — snapshot env is read-only | Use `reduceFn` to accumulate, or step-level `set` |
| Runtime-only type checking | Gradual typing — expressions validated at runtime | Use `assert` before lambda calls for validation |

---

## Constants

Declare immutable values accessible throughout the program:

```jsonc
{
  "constants": {
    "MAX_RETRIES": 3,
    "TAX_RATE": 0.11,
    "STATUS_ACTIVE": "active"
  },
  "steps": [
    {"let": "total", "expr": "price * (1 + TAX_RATE)"}
  ]
}
```

Attempting to `set` a constant produces a **compile-time error**:

```jsonc
{"set": "MAX_RETRIES", "value": 5}  // ❌ CONST_REASSIGN error
```

---

## Enums

Define named value sets:

```jsonc
{
  "enums": {
    "Status": ["draft", "confirmed", "done", "cancelled"],
    "Priority": {"LOW": 1, "MEDIUM": 2, "HIGH": 3, "CRITICAL": 4}
  }
}
```

Access via dot notation:

```jsonc
{"let": "s", "expr": "Status.draft"}        // "draft"
{"let": "p", "expr": "Priority.HIGH"}       // 3
{"if": "order.status == Status.confirmed", "then": [...]}
```

Array enums map each value to itself (`"draft" → "draft"`). Map enums use the declared key-value pairs.

Attempting to `set` an enum produces a **compile-time error**.

---

## Sleep

Pause execution for a specified duration (milliseconds):

```jsonc
{"sleep": 1000}                    // literal: 1 second
{"sleep": "delay * 1000"}          // expression
```

Constraints:
- Maximum: 300,000ms (5 minutes)
- Zero or negative: no-op
- Respects program timeout (context cancellation)

---

## Retry

Retry a block of steps with configurable backoff:

```jsonc
{
  "retry": {
    "steps": [
      {"let": "resp", "call": "http.get", "args": ["https://api.example.com/data"]},
      {"assert": "resp.status == 200"}
    ],
    "max": 3,
    "delay": 1000,
    "backoff": "exponential"
  }
}
```

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `steps` | array | required | Steps to retry on error |
| `max` | int | 3 | Maximum attempts |
| `delay` | int | 1000 | Base delay in ms |
| `backoff` | string | "fixed" | `"fixed"`, `"linear"`, `"exponential"` |

Backoff calculation:
- **fixed**: `delay` ms every time
- **linear**: `delay × attempt` ms
- **exponential**: `delay × 2^(attempt-1)` ms

---

## Assert

Validate a condition at runtime:

```jsonc
{"assert": "len(items) > 0", "message": "'Items cannot be empty'"}
{"assert": "total >= 0"}
```

If the condition is false, throws `ASSERTION_FAILED` error with the condition text or custom message.
