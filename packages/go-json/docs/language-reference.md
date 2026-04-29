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

#### `call` vs `expr` — When to Use Which

This is a critical distinction:

**`call`** invokes functions defined in the program's `"functions"` block (or struct methods). It uses **named arguments** via `with`:

```json
{"call": "calculateDiscount", "with": {"price": "100.0", "tier": "'gold'"}}
{"let": "result", "call": "factorial", "with": {"n": "10"}}
{"call": "person.birthday"}
```

**`expr`** evaluates an expression via the expr-lang engine. It can call **anything** available in the expression environment — go-json functions (positional args), stdlib functions, I/O module functions, and extension functions:

```json
{"let": "result", "expr": "factorial(10)"}
{"let": "hash", "expr": "crypto.sha256(password)"}
{"let": "resp", "expr": "http.get('https://api.example.com/users')"}
{"let": "rows", "expr": "sql.query('SELECT * FROM users', [])"}
```

**The rule:**

| What you're calling | Use | Why |
|---------------------|-----|-----|
| Your own function (defined in `functions`) | `call` or `expr` | Both work. `call` gives named args; `expr` gives positional |
| Struct method | `call` or `expr` | Both work. `call`: `{"call": "p.greet"}`. `expr`: `p.greet()` |
| Stdlib function (`upper`, `clamp`, `crypto.*`) | `expr` only | These are expression-environment functions, not program functions |
| I/O module function (`http.*`, `fs.*`, `sql.*`) | `expr` only | These are injected into the expression environment via import |
| Extension function (`ext:*`) | `expr` only | Same — injected into expression environment |

**Common mistake** — this does NOT work:

```json
{"call": "http.get", "with": {"url": "'https://api.example.com'"}}
```

`call` looks in `program.Functions` for a function named `http.get` — it won't find it because `http` is an I/O module, not a program-defined function. Use `expr` instead:

```json
{"let": "resp", "expr": "http.get('https://api.example.com')"}
```

**For side-effect-only calls** (where you don't need the return value), use `call` directly:

```json
{"call": "fs.write", "args": ["./log.txt", "content"]}
{"call": "redis.set", "args": ["user:123", "value"]}
```

Or with expression args:

```json
{"call": "fs.write", "with": ["'./log.txt'", "content"]}
```

**Literal values with `args`** — no expression evaluation, no quote wrapping:

```json
{"call": "fs.write", "args": ["./log.txt", "Don't forget `backtick` and \"quotes\""]}
{"call": "redis.set", "args": ["user:123", {"name": "Alice", "age": 30}, 3600]}
```

`args` passes JSON values as-is. Strings are strings, numbers are numbers. No `'...'` wrapping needed. Use `args` when you have literal data; use `with` when you need computed values.

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

**Error modes:**

| Mode | Behavior |
|------|----------|
| `cancel_all` | (Default) Cancel all branches on first error |
| `continue` | Let remaining branches finish; propagate error after |
| `collect` | Collect all errors; never cancel |

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
upper("hello")                    // → "HELLO"
len(items)                        // → 5
contains("hello world", "world")  // → true
clamp(value, 0, 100)              // → bounded value
filter(users, .active)            // → active users only
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
| General-purpose, everyone uses it | `len()`, `upper()`, `contains()` | — |
| Domain-specific, grouped by concern | — | `crypto.*`, `regex.*` |
| Collision risk (name too generic alone) | — | `crypto.sha256` (not just `sha256`) |
| Side effects / I/O | — | `http.*`, `fs.*`, `sql.*` |

For example, `contains("abc", "b")` is flat because it's a universal utility with no collision risk. But `sha256("hello")` is namespaced as `crypto.sha256("hello")` because "sha256" alone is too specific and could collide with user variables.

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

- **Circular imports** are detected at compile time and rejected.
- **Diamond imports** are handled correctly — each module is loaded exactly once.

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
