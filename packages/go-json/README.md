# go-json

A general-purpose programming language written in JSON/JSONC, embeddable in Go applications.

**Write programs in JSON. Run them anywhere Go runs.**

```json
{
  "name": "hello",
  "go_json": "1",
  "steps": [
    {"let": "name", "expr": "input.name ?? 'World'"},
    {"return": "'Hello, ' + name + '!'"}
  ]
}
```

```bash
$ go-json run hello.json --input '{"name": "Alice"}'
"Hello, Alice!"
```

## Why go-json?

go-json fills a gap no existing tool covers: a **general-purpose JSON programming language** that is embeddable in Go applications.

| Tool | What It Does | What It Can't Do |
|------|-------------|-----------------|
| jsonnet | Data templating | No side effects, no I/O, no control flow |
| jq | JSON transformation | Single-purpose, no general programming |
| AWS Step Functions | Cloud workflows | AWS-locked, no local execution |
| Node-RED | IoT/integration | Node.js-dependent, visual-only |
| CEL / expr-lang | Expression evaluation | Expressions only, no statements or functions |

go-json is a **complete programming language** — variables, functions, loops, structs, imports, error handling, parallel execution, I/O, and a built-in web server — all expressed as valid JSON.

### Use Cases

- **No-code/low-code engine** — Build visual programming UIs where the output is JSON
- **Business rules & automation** — Define rules that non-developers can read and modify
- **Embeddable scripting** — Add user-programmable logic to your Go application
- **Workflow orchestration** — Multi-step processes with branching, loops, and parallel execution
- **API servers** — Declarative HTTP routing with middleware, auth, and templates
- **Code generation** — Write once in JSON, generate Go, JavaScript, or Python

## Installation

### As a CLI tool

```bash
go install github.com/bitcode-framework/go-json/cmd/go-json@latest
```

### As a Go library

```bash
go get github.com/bitcode-framework/go-json@latest
```

**Requirements:** Go 1.24+

## Quick Start

### 1. Hello World

Create `hello.json`:

```json
{
  "name": "hello",
  "go_json": "1",
  "steps": [
    {"let": "greeting", "value": "Hello, World!"},
    {"return": "greeting"}
  ]
}
```

```bash
go-json run hello.json
```

### 2. With Input

Create `greet.json`:

```json
{
  "name": "greet",
  "input": {"name": "string", "age": "int"},
  "steps": [
    {"let": "msg", "expr": "'Hello, ' + input.name + '! You are ' + string(input.age) + ' years old.'"},
    {"return": "msg"}
  ]
}
```

```bash
go-json run greet.json --input '{"name": "Alice", "age": 30}'
```

### 3. Functions & Recursion

```json
{
  "name": "math_demo",
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
    {"let": "result", "call": "factorial", "with": {"n": "10"}},
    {"return": "result"}
  ]
}
```

### 4. Web Server

```json
{
  "name": "my_api",
  "server": {"port": 3000},
  "functions": {
    "listUsers": {
      "params": {"request": "map"},
      "steps": [
        {"return": {"value": {"status": 200, "body": [
          {"id": 1, "name": "Alice"},
          {"id": 2, "name": "Bob"}
        ]}}}
      ]
    }
  },
  "routes": [
    {"method": "GET", "path": "/api/users", "handler": "listUsers"}
  ]
}
```

```bash
go-json serve api.json
# Server running at http://localhost:3000
# Swagger UI at http://localhost:3000/docs
```

### 5. I/O Modules

```json
{
  "name": "fetch_and_save",
  "import": {"http": "io:http", "fs": "io:fs"},
  "steps": [
    {"let": "resp", "call": "http.get", "args": ["https://api.example.com/data"]},
    {"call": "fs.write", "args": ["./data.json", "raw content here"]},
    {"call": "fs.write", "with": ["'./result.json'", "toJSON(resp.body)"]},
    {"return": "resp.body"}
  ]
}
```

Three ways to call any function — choose based on your needs:

```json
{"call": "http.get", "args": ["https://api.com"]}
{"call": "http.get", "with": ["url"]}
{"let": "r", "expr": "http.get('https://api.com')"}
```

### 6. JSONC Support (Comments!)

go-json supports JSONC — JSON with comments and trailing commas:

```jsonc
{
  // Calculate discount based on customer tier
  "name": "discount_calculator",
  "go_json": "1",
  "functions": {
    "calculateDiscount": {
      "params": {
        "price": "float",
        "tier": "string",
      }, // trailing comma OK
      "returns": "float",
      "steps": [
        {"_c": "Gold customers get 20% off"},
        {"if": "tier == 'gold'", "then": [
          {"return": "price * 0.8"}
        ]},
        /* Silver customers get 10% off */
        {"if": "tier == 'silver'", "then": [
          {"return": "price * 0.9"}
        ]},
        {"return": "price"}
      ]
    }
  },
  "steps": [
    {"let": "final_price", "call": "calculateDiscount", "with": {
      "price": "input.price",
      "tier": "input.tier"
    }},
    {"return": "final_price"}
  ]
}
```

## Embedding in Go

```go
package main

import (
    "fmt"
    gojson "github.com/bitcode-framework/go-json/runtime"
    "github.com/bitcode-framework/go-json/stdlib"
)

func main() {
    // Create runtime
    reg := stdlib.DefaultRegistry()
    rt := gojson.NewRuntime(
        gojson.WithStdlib(reg.All()),
        gojson.WithStdlibEnv(reg.EnvVars()),
    )

    // Compile program
    program, err := rt.CompileFile("program.json")
    if err != nil {
        panic(err)
    }

    // Execute with input
    result, err := rt.Execute(program, map[string]any{
        "name": "Alice",
        "age":  30,
    })
    if err != nil {
        panic(err)
    }

    fmt.Println(result.Value)
    fmt.Printf("Completed in %d steps, %v\n", result.Steps, result.Duration)
}
```

### With I/O Modules

```go
rt := gojson.NewRuntime(
    gojson.WithStdlib(reg.All()),
    gojson.WithStdlibEnv(reg.EnvVars()),
    gojson.WithIO(goio.HTTP(), goio.FS(), goio.SQL()),
)
```

### With Custom Extensions

```go
rt := gojson.NewRuntime(
    gojson.WithStdlib(reg.All()),
    gojson.WithStdlibEnv(reg.EnvVars()),
    gojson.WithoutIO(),
    gojson.WithExtension("myapp", runtime.Extension{
        Functions: map[string]any{
            "getUser": func(id string) (map[string]any, error) {
                // your Go code here
                return map[string]any{"name": "Alice"}, nil
            },
        },
    }),
)
```

## Documentation

| Document | Description |
|----------|-------------|
| [Features](features.md) | Complete feature overview with capability matrix |
| **Language** | |
| [Language Reference](docs/language-reference.md) | Step types, syntax, type system, scoping rules |
| [Built-in Functions](docs/built-in-functions.md) | All 110+ functions (expr-lang + stdlib) |
| **I/O & Server** | |
| [I/O Modules](docs/io-modules.md) | HTTP, FS, SQL, Exec, MongoDB, Redis |
| [Web Server](docs/web-server.md) | Routing, middleware, auth, templates, OpenAPI |
| **Integration** | |
| [Embedding Guide](docs/embedding-guide.md) | Go API, extensions, resource limits |
| [CLI Reference](docs/cli-reference.md) | All commands and flags |
| [Code Generation](docs/code-generation.md) | Go/JS/Python codegen, CRUD generator, patterns |

## Architecture

```
                    ┌─────────────────────────────────────────┐
                    │              Your Program               │
                    │           (.json / .jsonc)               │
                    └──────────────┬──────────────────────────┘
                                   │
                    ┌──────────────▼──────────────────────────┐
                    │         JSONC Pre-processor              │
                    │   Strip comments, trailing commas        │
                    └──────────────┬──────────────────────────┘
                                   │
                    ┌──────────────▼──────────────────────────┐
                    │            Parser                        │
                    │      JSON → AST nodes                    │
                    └──────────────┬──────────────────────────┘
                                   │
                    ┌──────────────▼──────────────────────────┐
                    │        Import Resolver                   │
                    │   Resolve ./relative, stdlib:, io:, ext: │
                    └──────────────┬──────────────────────────┘
                                   │
                    ┌──────────────▼──────────────────────────┐
                    │           Compiler                       │
                    │  Struct registration, validation,        │
                    │  limit resolution → CompiledProgram      │
                    └──────────────┬──────────────────────────┘
                                   │
                    ┌──────────────▼──────────────────────────┐
                    │         Virtual Machine                  │
                    │  Tree-walk interpreter with debug hooks  │
                    │  Expressions via expr-lang/expr          │
                    └─────────────────────────────────────────┘
```

**Key design decisions:**

- **Compile-once, run-many** — Programs compile to an immutable `CompiledProgram`. Multiple goroutines can execute the same program concurrently.
- **Expression engine abstraction** — The VM never calls expr-lang directly. All expressions go through the `ExprEngine` interface.
- **Safe by default** — Resource limits (step count, call depth, loop iterations, timeout, memory proxies) enforced at every step.
- **JSON-native** — Programs are valid JSON (or JSONC). No custom syntax to learn beyond JSON.

## License

MIT
