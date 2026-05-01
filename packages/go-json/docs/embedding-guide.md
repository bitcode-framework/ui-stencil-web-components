# Embedding go-json in Go Applications

This guide covers everything you need to embed the go-json runtime in your Go application — from basic setup to advanced patterns like custom extensions, I/O sandboxing, and the Bitcode bridge.

**Module:** `github.com/bitcode-framework/go-json`
**Go version:** 1.24+

---

## Table of Contents

- [Quick Start](#quick-start)
- [Compilation](#compilation)
  - [CompileFile](#compilefile)
  - [Compile](#compile)
  - [Compile-Once, Run-Many](#compile-once-run-many)
- [Execution](#execution)
  - [Execute](#execute)
  - [ExecuteJSON](#executejson)
  - [ExecuteFunction](#executefunction)
- [Runtime Options](#runtime-options)
- [Resource Limits](#resource-limits)
- [Session Context](#session-context)
- [Extensions](#extensions)
- [I/O Modules](#io-modules)
- [Expression Evaluation](#expression-evaluation)
- [Debugging](#debugging)
- [Logging](#logging)
- [Execution Tracing](#execution-tracing)
- [Error Handling](#error-handling)
- [Bitcode Bridge Pattern](#bitcode-bridge-pattern)

---

## Quick Start

```go
package main

import (
	"fmt"
	"log"

	gojson "github.com/bitcode-framework/go-json/runtime"
	"github.com/bitcode-framework/go-json/stdlib"
)

func main() {
	// 1. Create a registry with the standard library
	reg := stdlib.DefaultRegistry()

	// 2. Build the runtime
	rt := gojson.NewRuntime(
		gojson.WithStdlib(reg.All()),
		gojson.WithStdlibEnv(reg.EnvVars()),
	)

	// 3. Compile a program from a file
	program, err := rt.CompileFile("program.json")
	if err != nil {
		log.Fatal(err)
	}

	// 4. Execute with input
	result, err := rt.Execute(program, map[string]any{
		"name": "Alice",
		"age":  30,
	})
	if err != nil {
		log.Fatal(err)
	}

	// 5. Read the result
	fmt.Println(result.Value)    // program return value
	fmt.Println(result.Steps)    // number of steps executed
	fmt.Println(result.Duration) // wall-clock execution time
}
```

---

## Compilation

go-json separates compilation from execution. You compile a program once, then execute it as many times as you need.

### CompileFile

Compiles a `.json` program from disk. This is the preferred method because it enables `import` resolution — relative imports are resolved against the file's directory.

```go
program, err := rt.CompileFile("path/to/program.json")
```

Programs compiled with `CompileFile` are cached by file path.

### Compile

Compiles a program from raw bytes. Import resolution is **not** available because there is no base directory to resolve against.

```go
jsonBytes := []byte(`{
    "steps": [
        {"let": "greeting", "expr": "'Hello, ' + name"},
        {"return": "greeting"}
    ]
}`)

program, err := rt.Compile(jsonBytes)
```

Programs compiled with `Compile` are cached by content hash.

### Compile-Once, Run-Many

A `CompiledProgram` is immutable after compilation. It is safe to execute the same program from multiple goroutines concurrently — each execution gets its own VM instance and variable scope.

```go
program, err := rt.CompileFile("handler.json")
if err != nil {
	log.Fatal(err)
}

// Safe to call from multiple goroutines
http.HandleFunc("/process", func(w http.ResponseWriter, r *http.Request) {
	result, err := rt.Execute(program, map[string]any{
		"method": r.Method,
		"path":   r.URL.Path,
	})
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	json.NewEncoder(w).Encode(result.Value)
})
```

---

## Execution

### Execute

Runs a compiled program with the given input map. Input keys become top-level variables in the program.

```go
result, err := rt.Execute(program, map[string]any{
	"name": "Alice",
	"age":  30,
})
```

The `result` struct contains:

| Field      | Type            | Description                              |
|------------|-----------------|------------------------------------------|
| `Value`    | `any`           | The program's return value               |
| `Steps`    | `int`           | Total steps executed                     |
| `Duration` | `time.Duration` | Wall-clock execution time                |
| `Trace`    | `[]TraceEntry`  | Execution trace (if tracing is enabled)  |

### ExecuteJSON

A convenience method that compiles and executes in a single call. The compiled program is cached internally, so repeated calls with the same JSON are efficient.

```go
result, err := rt.ExecuteJSON(programJSON, map[string]any{
	"x": 10,
	"y": 20,
})
```

Best for one-off or dynamic programs. For programs you execute repeatedly, prefer explicit `Compile` + `Execute`.

### ExecuteFunction

Calls a specific named function within a compiled program. The input map is passed as the function's arguments.

```go
result, err := rt.ExecuteFunction(program, "calculateDiscount", map[string]any{
	"price": 100.0,
	"tier":  "gold",
})
```

This is used internally by the server handler bridge to invoke route handler functions, but you can use it directly to call any function defined in a go-json program.

---

## Runtime Options

All configuration is done through functional options passed to `NewRuntime`:

```go
rt := gojson.NewRuntime(
	gojson.WithStdlib(reg.All()),
	gojson.WithLimits(limits),
	gojson.WithRuntimeLogger(myLogger),
	// ... more options
)
```

| Option | Signature | Description |
|--------|-----------|-------------|
| `WithStdlib` | `(funcs []expr.Option)` | Register stdlib functions for expressions |
| `WithStdlibEnv` | `(envVars map[string]any)` | Register environment variables for expressions |
| `WithLimits` | `(limits Limits)` | Set resource limits (steps, depth, timeout, etc.) |
| `WithRuntimeLogger` | `(l Logger)` | Set a custom logger |
| `WithRuntimeDebugger` | `(d Debugger)` | Attach a step-level debugger |
| `WithRuntimeTrace` | `(enabled bool)` | Enable execution tracing |
| `WithSession` | `(s *Session)` | Set session context (user, locale, tenant) |
| `WithRuntimeContext` | `(ctx context.Context)` | Set a Go context for cancellation/deadlines |
| `WithIO` | `(modules ...IOModule)` | Enable I/O modules (HTTP, filesystem, etc.) |
| `WithoutIO` | `()` | Explicitly disable all I/O (this is the default) |
| `WithIOSecurity` | `(cfg *SecurityConfig)` | Configure I/O security policies |
| `WithEnvHandle` | `(h *stdlib.EnvHandle)` | Provide env handle from registry (enables WithEnvResolver/WithEnvAccess) |
| `WithEnvResolver` | `(resolver stdlib.EnvResolver)` | Override env() resolver (requires WithEnvHandle) |
| `WithEnvAccess` | `(config *stdlib.EnvAccessConfig)` | Override env() access control (requires WithEnvHandle) |
| `WithExtension` | `(name string, ext Extension)` | Register a host extension |

**Customizing `env()` function:**

```go
// Option A: Override at runtime (recommended for dynamic config)
reg := stdlib.DefaultRegistry()
rt := gojson.NewRuntime(
    gojson.WithStdlib(reg.All()),
    gojson.WithStdlibEnv(reg.EnvVars()),
    gojson.WithEnvHandle(reg.EnvHandle()),
    gojson.WithEnvResolver(viper.GetString),
    gojson.WithEnvAccess(&stdlib.EnvAccessConfig{
        Allow: []string{"APP_*", "PUBLIC_*"},
        Deny:  []string{"*_SECRET", "*_PASSWORD"},
    }),
)

// Option B: Configure at registry creation (simpler, static config)
reg := stdlib.DefaultRegistryWithEnv(viper.GetString, &stdlib.EnvAccessConfig{
    Deny: []string{"*_SECRET"},
})
```

---

## Resource Limits

Resource limits protect your host application from runaway programs.

```go
rt := gojson.NewRuntime(
	gojson.WithLimits(gojson.Limits{
		MaxSteps:          5000,              // max VM steps
		MaxDepth:          100,               // max call/scope depth
		MaxLoopIterations: 1000,              // max iterations per loop
		MaxVariables:      500,               // max variables in scope
		MaxVariableSize:   1024 * 1024,       // 1 MB per variable
		MaxOutputSize:     5 * 1024 * 1024,   // 5 MB total output
		Timeout:           10 * time.Second,  // wall-clock timeout
	}),
)
```

### Defaults

| Limit               | Default    |
|----------------------|------------|
| `MaxSteps`           | 10,000     |
| `MaxDepth`           | 1,000      |
| `MaxLoopIterations`  | 10,000     |
| `MaxVariables`       | 1,000      |
| `MaxVariableSize`    | 10 MB      |
| `MaxOutputSize`      | 50 MB      |
| `Timeout`            | 30 seconds |

### Hard Limits

The runtime enforces hard ceilings that cannot be exceeded regardless of configuration:

| Limit               | Hard Maximum |
|----------------------|--------------|
| `MaxSteps`           | 100,000      |
| `MaxDepth`           | 10,000       |
| `MaxLoopIterations`  | 100,000      |

If you set a value above the hard limit, it is silently clamped.

---

## Session Context

Session context lets you pass per-request identity and locale information into programs.

```go
rt := gojson.NewRuntime(
	gojson.WithSession(&gojson.Session{
		UserID:   "user-123",
		Locale:   "en",
		TenantID: "tenant-456",
		Groups:   []string{"admin", "editor"},
	}),
)
```

Inside a go-json program, session fields are available as:

- `session.user_id`
- `session.locale`
- `session.tenant_id`
- `session.groups`

---

## Extensions

Extensions let your host application inject custom functions, nested namespaces, and constants into the go-json runtime. This is the primary mechanism for giving programs access to your application's domain logic.

### Registering an Extension

```go
rt := gojson.NewRuntime(
	gojson.WithoutIO(),
	gojson.WithExtension("myapp", gojson.Extension{
		Functions: map[string]any{
			"getUser": func(id string) (map[string]any, error) {
				return map[string]any{"id": id, "name": "Alice"}, nil
			},
			"db": map[string]any{
				"query": func(sql string, args ...any) ([]map[string]any, error) {
					// Nested namespace: accessed as myapp.db.query(...)
					return nil, nil
				},
			},
		},
	}),
)
```

### Using Extensions in Programs

Programs import extensions with the `ext:` prefix:

```json
{
  "import": { "app": "ext:myapp" },
  "steps": [
    { "let": "user", "expr": "app.getUser('user-123')" },
    { "let": "rows", "expr": "app.db.query('SELECT * FROM users')" }
  ]
}
```

### Nested Namespaces

When a value in the `Functions` map is itself a `map[string]any`, it creates a nested namespace. This enables dotted access patterns like `app.db.query(...)` — organize your extension API however makes sense for your domain.

---

## I/O Modules

By default, go-json programs have **no I/O access**. You must explicitly opt in.

```go
import goio "github.com/bitcode-framework/go-json/io"
```

### Enable All I/O

```go
rt := gojson.NewRuntime(
	gojson.WithIO(goio.All()),
)
defer rt.Close() // close I/O resources when done
```

### Selective I/O

Only enable the modules you need:

```go
rt := gojson.NewRuntime(
	gojson.WithIO(goio.HTTP(), goio.FS()),
)
defer rt.Close()
```

### I/O Security

Lock down I/O with a security configuration:

```go
rt := gojson.NewRuntime(
	gojson.WithIO(goio.All()),
	gojson.WithIOSecurity(&goio.SecurityConfig{
		FS: goio.FSSecurityConfig{
			AllowedPaths: []string{"/tmp/sandbox"},
			AllowWrite:   true,
			MaxFileSize:  1024 * 1024, // 1 MB
		},
	}),
)
defer rt.Close()
```

> **Important:** Always call `rt.Close()` when you're done with a runtime that has I/O enabled, to release underlying resources.

### MongoDB & Redis

MongoDB and Redis modules use real drivers (`go.mongodb.org/mongo-driver/v2` and `github.com/redis/go-redis/v9`). Connection is lazy — established on first operation using the URI from security config:

```go
sec := goio.DefaultSecurityConfig()
sec.Mongo.DefaultURI = "mongodb://localhost:27017"
sec.Redis.DefaultURI = "redis://localhost:6379"

rt := gojson.NewRuntime(
	gojson.WithIO(goio.Mongo(sec), goio.Redis(sec)),
)
defer rt.Close()
```

---

## Expression Evaluation

For cases where you need to evaluate a standalone expression without compiling a full program, use the `EvalExpr` family of functions. These are lightweight, thread-safe, and use a shared singleton engine with compilation caching.

### General Evaluation

```go
import "github.com/bitcode-framework/go-json/runtime"

result, err := runtime.EvalExpr("price * quantity * (1 - discount/100)", map[string]any{
	"price":    100.0,
	"quantity": 5,
	"discount": 10.0,
})
// result = 450.0
```

### Typed Evaluation

```go
// Boolean result
ok, err := runtime.EvalExprBool("age >= 18 && status == 'active'", map[string]any{
	"age":    25,
	"status": "active",
})

// Float result
total, err := runtime.EvalExprFloat("price * 1.1", map[string]any{
	"price": 100.0,
})
```

These are useful for rule engines, dynamic configuration, computed fields, and anywhere you need user-defined expressions without the overhead of a full program.

---

## Debugging

Implement the `Debugger` interface to get step-level visibility into program execution:

```go
type Debugger interface {
	OnStep(info StepInfo) DebugAction
	OnVariable(name string, value any, scope string)
	OnError(err error, step int)
	OnFunctionCall(name string, args map[string]any)
	OnFunctionReturn(name string, result any)
}
```

### Example

```go
type MyDebugger struct{}

func (d *MyDebugger) OnStep(info gojson.StepInfo) gojson.DebugAction {
	fmt.Printf("Step %d: %s\n", info.Index, info.Type)
	return gojson.Continue // or gojson.StepOver, gojson.StepInto, gojson.Abort
}

func (d *MyDebugger) OnVariable(name string, value any, scope string) {
	fmt.Printf("  %s.%s = %v\n", scope, name, value)
}

func (d *MyDebugger) OnError(err error, step int) {
	fmt.Printf("  ERROR at step %d: %v\n", step, err)
}

func (d *MyDebugger) OnFunctionCall(name string, args map[string]any) {
	fmt.Printf("  CALL %s(%v)\n", name, args)
}

func (d *MyDebugger) OnFunctionReturn(name string, result any) {
	fmt.Printf("  RETURN %s -> %v\n", name, result)
}

// Attach to runtime
rt := gojson.NewRuntime(
	gojson.WithRuntimeDebugger(&MyDebugger{}),
)
```

---

## Logging

Implement the `Logger` interface for custom log output:

```go
type Logger interface {
	Log(level, message string, data map[string]any)
}
```

The default logger prints to stdout with timestamps. Replace it to integrate with your application's logging infrastructure:

```go
type AppLogger struct {
	logger *slog.Logger
}

func (l *AppLogger) Log(level, message string, data map[string]any) {
	l.logger.Info(message, "level", level, "data", data)
}

rt := gojson.NewRuntime(
	gojson.WithRuntimeLogger(&AppLogger{logger: slog.Default()}),
)
```

---

## Execution Tracing

Enable tracing to capture a detailed timeline of every step in a program's execution:

```go
rt := gojson.NewRuntime(
	gojson.WithRuntimeTrace(true),
)

result, err := rt.Execute(program, input)
if err != nil {
	log.Fatal(err)
}

for _, entry := range result.Trace {
	fmt.Printf("Step %d: %s (%dμs)\n", entry.Step, entry.Type, entry.DurationUs)
}
```

### Trace Entry Fields

| Field        | Type     | Description                                    |
|--------------|----------|------------------------------------------------|
| `Step`       | `int`    | Step index in the program                      |
| `Type`       | `string` | Step type (`let`, `set`, `if`, `while`, etc.)  |
| `DurationUs` | `int64`  | Execution time for this step in microseconds   |
| `Var`        | `string` | Variable name (for `let`/`set` steps)          |
| `Value`      | `any`    | Resulting value                                |
| `Condition`  | `any`    | Condition result (for `if`/`while` steps)      |
| `Result`     | `any`    | Step result                                    |

Tracing adds overhead. Enable it for development and diagnostics, not production hot paths.

---

## Error Handling

go-json returns structured errors of type `*lang.GoJSONError`. These provide rich context for diagnosing issues.

```go
import "github.com/bitcode-framework/go-json/lang"

result, err := rt.Execute(program, input)
if err != nil {
	if gjErr, ok := err.(*lang.GoJSONError); ok {
		fmt.Println("Code:       ", gjErr.Code)
		fmt.Println("Category:   ", gjErr.Category)  // "compile", "runtime", or "limit"
		fmt.Println("Message:    ", gjErr.Message)
		fmt.Println("Step:       ", gjErr.Step)
		fmt.Println("Function:   ", gjErr.Function)
		fmt.Println("Stack:      ", gjErr.Stack)
		fmt.Println("Context:    ", gjErr.Context)
		fmt.Println("Suggestions:", gjErr.Suggestions)
		fmt.Println("Fix:        ", gjErr.Fix)
	} else {
		// Non-GoJSON error (e.g., I/O failure)
		fmt.Println("Error:", err)
	}
}
```

### Error Fields

| Field         | Type       | Description                                      |
|---------------|------------|--------------------------------------------------|
| `Code`        | `string`   | Machine-readable error code                      |
| `Category`    | `string`   | `"compile"`, `"runtime"`, or `"limit"`           |
| `Message`     | `string`   | Human-readable error message                     |
| `Step`        | `int`      | Step index where the error occurred               |
| `Function`    | `string`   | Function name (if inside a function call)         |
| `Stack`       | `[]string` | Call stack at the point of failure                |
| `Context`     | `map`      | Additional context about the error                |
| `Suggestions` | `[]string` | Suggested fixes                                  |
| `Fix`         | `string`   | Specific fix recommendation                      |

---

## Bitcode Bridge Pattern

The Bitcode bridge pattern is the recommended approach for production applications. Instead of giving programs raw I/O access, you disable I/O entirely and inject a controlled bridge through extensions. This gives you full control over what programs can do.

```go
rt := gojson.NewRuntime(
	gojson.WithStdlib(reg.All()),
	gojson.WithStdlibEnv(reg.EnvVars()),
	gojson.WithoutIO(),                              // no raw I/O
	gojson.WithExtension("bc", gojson.Extension{     // inject your bridge
		Functions: map[string]any{
			"model": func(name string) any {
				// Return a model instance from your ORM
				return nil
			},
			"db": map[string]any{
				"query":   func(sql string, args ...any) ([]map[string]any, error) { return nil, nil },
				"execute": func(sql string, args ...any) (int64, error) { return 0, nil },
			},
			"http": map[string]any{
				"get":  func(url string) (map[string]any, error) { return nil, nil },
				"post": func(url string, body any) (map[string]any, error) { return nil, nil },
			},
			"cache": map[string]any{
				"get": func(key string) (any, error) { return nil, nil },
				"set": func(key string, value any, ttl int) error { return nil },
				"del": func(key string) error { return nil },
			},
			"log":  func(level, msg string) { /* your logger */ },
			"emit": func(event string, data any) { /* your event bus */ },
		},
	}),
	gojson.WithLimits(limits),
	gojson.WithSession(session),
)
```

Programs use the bridge through a standard import:

```json
{
  "import": { "bc": "ext:bc" },
  "steps": [
    { "let": "users", "expr": "bc.db.query('SELECT * FROM users WHERE active = ?', true)" },
    { "do": "bc.cache.set('active_users', users, 300)" },
    { "do": "bc.log('info', 'Cached ' + string(len(users)) + ' active users')" },
    { "return": "users" }
  ]
}
```

### Why Use the Bridge Pattern?

- **Security** — Programs can only call functions you explicitly provide. No filesystem access, no arbitrary HTTP, no shell execution.
- **Observability** — Every bridge function is your code. Add logging, metrics, and tracing at the boundary.
- **Testability** — Swap the bridge implementation in tests to mock external dependencies.
- **Consistency** — All programs interact with your infrastructure through a single, well-defined API.
