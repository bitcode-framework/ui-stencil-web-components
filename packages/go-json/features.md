# go-json Features

go-json is a general-purpose programming language where programs are valid JSON (or JSONC). It is designed as a **no-code engine** — programs can be authored through a visual UI or written directly in `.json`/`.jsonc` files.

> **Current version:** 0.1.0 | **Go:** 1.24+ | **Module:** `github.com/bitcode-framework/go-json`

## Feature Matrix

| Category | Feature | Status |
|----------|---------|--------|
| **Core Language** | Variables (`let`/`set`) with 3 value modes | Done |
| | Gradual type system (inferred → annotated) | Done |
| | Control flow (`if`/`elif`/`else`, `switch`) | Done |
| | Loops (`for`-each, `for`-range, `while`) with `break`/`continue` | Done |
| | Functions with typed params, defaults, returns | Done |
| | Recursion with configurable depth limits | Done |
| | Error handling (`try`/`catch`/`finally`, `error`) | Done |
| | JSONC support (comments, trailing commas) | Done |
| | Semantic comments (`_c`) | Done |
| **Expression Engine** | [expr-lang/expr](https://github.com/expr-lang/expr) integration (~68 built-in functions) | Done |
| | Arithmetic, comparison, logical, ternary operators | Done |
| | Array operations (`filter`, `map`, `reduce`, `find`, `sort`) | Done |
| | Optional chaining (`a?.b`), nil coalescing (`??`) | Done |
| | Pipe operator, string interpolation | Done |
| **Structs** | Struct definitions with typed fields and defaults | Done |
| | Runtime type-checking on construction | Done |
| | Methods with implicit `self` binding | Done |
| | Frozen (immutable) structs | Done |
| | Nested struct construction | Done |
| | Nested property access and mutation (`a.b.c`, `a[0].b`) | Done |
| **Modularity** | Import system (relative, stdlib, I/O, extension) | Done |
| | Circular import detection (direct and indirect) | Done |
| | Import alias collision detection | Done |
| | Re-export / barrel files | Done |
| | Diamond import handling | Done |
| **Parallel Execution** | Parallel branches with isolated scope | Done |
| | 3 join modes (`all`, `any`, `settled`) | Done |
| | 3 error modes (`cancel_all`, `continue`, `collect`) | Done |
| | Compile-time parent write check | Done |
| | Shared step counter across branches (prevents limit bypass) | Done |
| **Standard Library** | 110+ built-in functions across 3 layers | Done |
| | Math, string, array, map, datetime, encoding, crypto, regex, path, JSON | Done |
| | See [Built-in Functions](docs/built-in-functions.md) for complete reference | |
| **I/O Modules** | HTTP (GET/POST/PUT/PATCH/DELETE with auth) | Done |
| | File System (read/write/stat/copy/move/glob) | Done |
| | SQL (query/execute, multi-driver, transactions) | Done |
| | Exec (command execution with whitelisting) | Done |
| | MongoDB (CRUD + aggregation) | Stubbed |
| | Redis (key-value, hash, list, set, pub/sub) | Stubbed |
| | Two-layer security gating (import + runtime) | Done |
| **Web Server** | Declarative HTTP routing with groups | Done |
| | 5 framework adapters (Fiber, net/http, Echo, Gin, Chi) | Done |
| | Built-in middleware (logger, CORS, rate limit, compress, etc.) | Done |
| | Custom middleware as go-json functions | Done |
| | Pluggable auth (JWT, API Key, Basic Auth, Custom) | Done |
| | JWT module (sign, verify, decode, refresh) | Done |
| | Template engine (Go html/template, layouts, partials) | Done |
| | Static file serving | Done |
| | OpenAPI/Swagger auto-generation | Done |
| | Graceful shutdown | Done |
| **Code Generation** | Go, JavaScript, Python targets | Done |
| | Server codegen (Fiber, net/http, Express, FastAPI) | Done |
| | Dependency file generation (go.mod, package.json, etc.) | Done |
| **Generators** | CRUD generator with DB introspection (SQLite, PostgreSQL, MySQL) | Done |
| | Auth scaffold (register, login, refresh, me, change-password) | Done |
| | Project scaffold | Done |
| | Architecture patterns (simple, service-layer, DDD, hexagonal) | Done |
| **CLI** | `run`, `serve`, `check`, `test`, `ast`, `codegen`, `generate`, `openapi`, `migrate` | Done |
| **Safety** | Resource limits (steps, depth, loops, timeout, memory proxies) | Done |
| | Built-in name protection (prevents shadowing critical functions) | Done |
| | I/O security (host blocking, path traversal prevention, DDL protection) | Done |
| **Debugging** | Execution tracing | Done |
| | Debugger interface (step, variable, error, function call hooks) | Done |
| | "Did you mean?" suggestions on typos | Done |
| | Stack traces on errors | Done |
| **Embedding** | `NewRuntime()` / `Execute()` Go API | Done |
| | Compile-once, run-many (concurrent-safe) | Done |
| | Extension API for host applications | Done |
| | Program cache | Done |
| **Not Yet Done** | REPL mode | Planned |
| | Dev mode file watching | Planned |
| | MongoDB/Redis drivers (modules are stubbed) | Planned |
| | Interactive `go-json generate` | Planned |

---

## Core Language

go-json programs are JSON objects with a defined structure:

```json
{
  "name": "program_name",
  "go_json": "1",
  "import": { },
  "structs": { },
  "functions": { },
  "input": { },
  "steps": [ ]
}
```

All top-level keys are optional except `name`.
- A file with `steps` is an **executable program**
- A file without `steps` is a **library** (structs + functions only, importable by other programs)
- A file with `routes` is a **server program** (started with `go-json serve`)

### 16 Step Types

| Step | Purpose | Example |
|------|---------|---------|
| `let` | Declare new variable | `{"let": "x", "value": 42}` |
| `set` | Update existing variable | `{"set": "x", "expr": "x + 1"}` |
| `if`/`elif`/`else` | Conditional branching | `{"if": "x > 0", "then": [...]}` |
| `switch`/`cases` | Multi-way branching | `{"switch": "status", "cases": {...}}` |
| `for`/`in` | Iterate over array | `{"for": "item", "in": "items", "steps": [...]}` |
| `for`/`range` | Iterate over number range | `{"for": "i", "range": [0, 10], "steps": [...]}` |
| `while` | Conditional loop | `{"while": "count < 100", "steps": [...]}` |
| `break` | Exit loop | `{"break": true}` |
| `continue` | Skip to next iteration | `{"continue": true}` |
| `return` | Return value | `{"return": "result"}` |
| `call` | Call function | `{"call": "myFunc", "with": {"x": "42"}}` |
| `try`/`catch`/`finally` | Error handling | `{"try": [...], "catch": {"as": "err", "steps": [...]}}` |
| `error` | Throw error | `{"error": "'something went wrong'"}` |
| `log` | Log message | `{"log": "'Processing item: ' + item.name"}` |
| `parallel` | Parallel execution | `{"parallel": {"a": [...], "b": [...]}, "into": "results"}` |
| `_c` | Semantic comment | `{"_c": "This step does X"}` |

### 3 Value Modes

Every `let`, `set`, and `return` step uses exactly one of three value modes:

| Mode | Key | Behavior | Example |
|------|-----|----------|---------|
| **Literal** | `value` | JSON value stored as-is, no evaluation | `{"let": "x", "value": 42}` |
| **Expression** | `expr` | String evaluated by expr-lang engine | `{"let": "x", "expr": "a + b * 2"}` |
| **Computed Object** | `with` | Each field value is an expression | `{"let": "profile", "with": {"name": "input.name", "adult": "age >= 18"}}` |

Using multiple modes in one step is a compile error.

### Type System

go-json uses **gradual typing** — start untyped for quick scripts, add types for production safety:

| Type | Example | Notes |
|------|---------|-------|
| `string` | `"hello"` | |
| `int` | `42` | 64-bit integer |
| `float` | `3.14` | 64-bit float |
| `bool` | `true` | |
| `[]T` | `[]string`, `[]int` | Typed array |
| `[]any` | `[1, "two", true]` | Mixed array |
| `map` | `{"k": "v"}` | String-keyed map |
| `StructName` | `Person` | User-defined struct |
| `?T` | `?string`, `?Person` | Nullable |
| `any` | anything | Opt-out of type checking |

Types are **inferred** by default and **locked after first assignment**. Assigning a different type to an existing variable is a compile error unless the variable is declared as `any`.

### Expression Engine

Expressions are powered by [expr-lang/expr](https://github.com/expr-lang/expr), a production-grade expression engine with:

- Arithmetic: `+`, `-`, `*`, `/`, `%`, `**`
- Comparison: `==`, `!=`, `>`, `<`, `>=`, `<=`
- Logical: `&&`, `||`, `!`, ternary `a ? b : c`
- String: concatenation with `+`, backtick strings
- Array: `filter(items, .age > 18)`, `map(items, .name)`, `reduce(items, # + #acc, 0)`
- Nil safety: optional chaining `a?.b`, nil coalescing `a ?? default`
- Pipe: `items | filter(.active) | map(.name) | sort()`
- Member access: `person.address.city`, `items[0].name`

**Function namespacing:** Most functions are called flat (`upper()`, `len()`, `contains()`). Domain-specific functions are grouped under namespaces (`crypto.sha256()`, `regex.match()`). I/O modules use import-based namespaces (`http.get()`, `sql.query()`). Namespaces are standard expr-lang map member access — not special syntax. See [Built-in Functions](docs/built-in-functions.md#function-namespacing) for the complete reference and design rationale.

---

## Structs & Methods

Define data structures with typed fields, defaults, and methods:

```json
{
  "structs": {
    "Person": {
      "fields": {
        "name": "string",
        "age": "int",
        "email": "?string",
        "country": {"type": "string", "default": "'ID'"}
      },
      "methods": {
        "greet": {
          "params": {"greeting": "string"},
          "returns": "string",
          "steps": [
            {"return": "greeting + ', ' + self.name + '!'"}
          ]
        },
        "birthday": {
          "steps": [
            {"set": "self.age", "expr": "self.age + 1"}
          ]
        }
      }
    }
  }
}
```

**Key features:**
- Methods have implicit `self` — no need to declare it in params
- Structs are **mutable by default** — methods can modify `self.field`
- Add `"frozen": true` to make a struct immutable (compile error on any `set "self.*"`)
- **Runtime type-checking** on construction — assigning wrong type to a field produces `TYPE_MISMATCH` error
- Nested construction: `{"let": "p", "new": "Person", "with": {"name": "'Alice'", "address": {"new": "Address", "with": {...}}}}`
- Method chaining works naturally: `person.withName('Bob').withAge(30).greet('Hello')`
- No inheritance — use **composition** instead

---

## Import System

```json
{
  "import": {
    "models": "./types/person.json",
    "utils": "./utils/validators.json",
    "http": "io:http",
    "bc": "ext:bitcode"
  }
}
```

| Path Format | Resolves To |
|-------------|-------------|
| `./file.json` | Relative to current file |
| `../dir/file.json` | Relative parent directory |
| `stdlib:name` | Built-in stdlib module |
| `io:name` | I/O module (HTTP, FS, SQL, Exec) |
| `ext:name` | Host-injected extension |

**What gets exported:** Structs and functions. Steps, input, and limits are NOT exported.

**Safety:** Circular imports (direct and indirect) are detected at compile time. Diamond imports (A→B, A→C, B→D, C→D) are handled correctly — D is loaded once and shared. Alias collisions (two imports producing the same namespaced name) are compile errors.

---

## Parallel Execution

Run independent branches concurrently:

```json
{
  "parallel": {
    "users": [
      {"let": "data", "call": "fetchUsers"},
      {"return": "data"}
    ],
    "orders": [
      {"let": "data", "call": "fetchOrders"},
      {"return": "data"}
    ]
  },
  "on_error": "cancel_all",
  "into": "results"
}
```

After execution, `results.users` and `results.orders` contain each branch's return value.

**Join modes** (`"join"`):
| Mode | Behavior |
|------|----------|
| `all` (default) | Wait for all branches |
| `any` | First successful branch wins, cancel remaining |
| `settled` | Wait for all regardless of errors; errors collected as `{"error": true, "message": "..."}` |

**Error modes** (`"on_error"`, applies when join is `all`):
| Mode | Behavior |
|------|----------|
| `cancel_all` (default) | First error cancels all branches |
| `continue` | Other branches continue; failed branch = `nil` |
| `collect` | Other branches continue; failed branch = error object |

**Scope isolation:** Each branch gets a read-only copy of parent scope. Writing to parent variables from a parallel branch is a **compile error**. Step limits are shared across all branches via atomic counter.

---

## I/O Modules

I/O is opt-in at two levels: the program must **import** the module, and the runtime must **enable** it.

```json
{
  "import": {"http": "io:http", "fs": "io:fs"},
  "steps": [
    {"let": "resp", "call": "http.get", "args": ["https://api.example.com/users"]},
    {"let": "data", "call": "fs.read", "args": ["./config.json"]},
    {"call": "fs.write", "with": ["'./output.json'", "toJSON(resp.body)"]},
    {"let": "hash", "expr": "crypto.sha256(data)"}
  ]
}
```

Three calling styles work for all I/O functions:

```json
// call + args — literal values, no escaping
{"call": "fs.write", "args": ["./log.txt", "Don't forget `backtick`"]}

// call + with — expression args, variables evaluated
{"call": "fs.write", "with": ["'./log.txt'", "content"]}

// expr — inline expression
{"let": "_", "expr": "fs.write('./log.txt', content)"}
```

| Module | Functions |
|--------|-----------|
| `io:http` | `get`, `post`, `put`, `patch`, `delete` — with auth (bearer/basic), redirect following, response size limits |
| `io:fs` | `read`, `write`, `append`, `exists`, `list`, `mkdir`, `remove`, `stat`, `copy`, `move`, `glob` — with sandboxing |
| `io:sql` | `query`, `execute`, `begin`, `commit`, `rollback` — multi-driver (SQLite, PostgreSQL, MySQL, SQL Server, Oracle), connection pooling, DDL protection |
| `io:exec` | `run` — command whitelisting, env isolation, output truncation |
| `io:mongo` | `find`, `findOne`, `insert`, `insertMany`, `update`, `delete`, `count`, `aggregate` |
| `io:redis` | `get`, `set`, `del`, `exists`, `expire`, `ttl`, `incr`, `decr`, `hget`, `hset`, `hgetall`, `lpush`, `rpush`, `lrange`, `sadd`, `smembers`, `publish` |

See [I/O Modules](docs/io-modules.md) for complete documentation.

---

## Web Server

go-json can run as a web server with declarative routing:

```json
{
  "name": "my_api",
  "server": {
    "port": 3000,
    "cors": {"origins": ["*"]},
    "auth": {
      "default": "jwt",
      "strategies": {
        "jwt": {"type": "bearer", "secret_env": "JWT_SECRET"}
      }
    }
  },
  "middleware": ["logger", "recover", "cors"],
  "functions": {
    "getUser": {
      "params": {"request": "map"},
      "steps": [
        {"let": "id", "expr": "request.params.id"},
        {"return": {"with": {"status": "200", "body": "{'id': id, 'name': 'Alice'}"}}}
      ]
    }
  },
  "routes": [
    {
      "prefix": "/api",
      "middleware": ["auth"],
      "routes": [
        {"method": "GET", "path": "/users/:id", "handler": "getUser"}
      ]
    }
  ]
}
```

**Features:**
- 5 framework adapters: Fiber (default), net/http, Echo, Gin, Chi
- Built-in middleware: logger, recover, CORS, secure headers, request_id, compress, rate_limit
- Pluggable auth: JWT, API Key, Basic Auth, Custom (go-json function)
- JWT module: `jwt.sign()`, `jwt.verify()`, `jwt.decode()`, `jwt.refresh()`
- Template engine: Go html/template with 20+ helpers, layouts, partials
- Static file serving with path traversal protection
- OpenAPI/Swagger auto-generation at `/docs`
- Graceful shutdown on SIGINT/SIGTERM

See [Web Server](docs/web-server.md) for complete documentation.

---

## Code Generation

Generate native code from go-json programs:

```bash
go-json codegen program.json --target go --output program.go
go-json codegen program.json --target javascript --output program.js
go-json codegen program.json --target python --output program.py
```

For server programs, specify the framework:

```bash
go-json codegen api.json --target go --framework fiber
go-json codegen api.json --target javascript --framework express
go-json codegen api.json --target python --framework fastapi
```

### CRUD Generator

Generate a complete API from a database table:

```bash
# From database introspection
go-json generate crud --table users --dsn "postgres://localhost/mydb"

# From manual field definition
go-json generate crud --table users --fields "name:string,email:string,role:string"

# With auth middleware on write endpoints
go-json generate crud --table users --fields "name:string,email:string" --auth

# Auth scaffold (register, login, refresh, me, change-password)
go-json generate auth --output auth.json

# Full project scaffold
go-json generate project my-api --auth
```

See [Code Generation](docs/code-generation.md) for complete documentation.

---

## Safety & Resource Limits

go-json is safe by default. Every execution is bounded:

| Limit | Default | Hard Max | Purpose |
|-------|---------|----------|---------|
| `MaxSteps` | 10,000 | 100,000 | Total step executions |
| `MaxDepth` | 1,000 | 10,000 | Call/recursion depth |
| `MaxLoopIterations` | 10,000 | 100,000 | Per-loop iterations |
| `MaxNodes` | 1,000 | — | Expression AST complexity |
| `MaxVariables` | 1,000 | — | Variables in scope |
| `MaxVariableSize` | 10 MB | — | Single variable size |
| `MaxOutputSize` | 50 MB | — | Program output size |
| `Timeout` | 30s | — | Wall-clock execution time |

Limits are resolved using the **most restrictive** value across: engine hard limit → project config → module config → program config → step config.

---

## Testing

go-json has a built-in test framework:

```json
{
  "name": "test_math",
  "test": true,
  "import": {"math": "./math.json"},
  "cases": [
    {
      "_c": "Factorial of 5 is 120",
      "call": "math.factorial",
      "with": {"n": "5"},
      "expect": 120
    },
    {
      "_c": "Factorial of 0 is 1",
      "call": "math.factorial",
      "with": {"n": "0"},
      "expect": 1
    }
  ]
}
```

```bash
go-json test tests/
# ✓ test_math: Factorial of 5 is 120 (1ms)
# ✓ test_math: Factorial of 0 is 1 (0ms)
# 2 passed, 0 failed
```

---

## Project Stats

| Metric | Value |
|--------|-------|
| Total Go source files | ~143 |
| Total lines of code | ~18,000 |
| Test count | 723 |
| Test files | 50 |
| Stdlib functions (Layer 2) | 42+ |
| expr-lang built-ins (Layer 1) | ~68 |
| I/O modules | 6 |
| Server framework adapters | 5 |
| Codegen targets | 3 languages × multiple frameworks |
| Architecture patterns | 4 (with template files) |
| CLI commands | 9 |
