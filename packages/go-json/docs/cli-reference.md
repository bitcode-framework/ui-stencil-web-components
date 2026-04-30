# CLI Reference

go-json provides a command-line interface for running, validating, testing, and generating code from JSON programs.

```bash
go-json <command> [options]
```

## Commands

| Command | Description |
|---------|-------------|
| `run` | Execute a go-json program |
| `serve` | Start a web server from a server program |
| `check` | Validate a program (compile check, no execution) |
| `test` | Run test files |
| `ast` | Export program AST as JSON |
| `codegen` | Generate Go/JS/Python code from a program |
| `generate` | Scaffold CRUD, auth, or project from templates |
| `openapi` | Generate OpenAPI spec from a server program |
| `migrate` | Migrate deprecated syntax to newer version |
| `version` | Print version |
| `help` | Print help |

---

## `go-json run`

Execute a go-json program.

```bash
go-json run <program.json> [options]
```

### Options

| Flag | Default | Description |
|------|---------|-------------|
| `--input <json>` | — | Inline JSON input |
| `--input-file <path>` | — | Read input from file |
| `--timeout <duration>` | `30s` | Execution timeout |
| `--max-depth <n>` | — | Override call depth limit |
| `--io <modules>` | — | Enable I/O modules (`http,fs,sql,exec` or `all`) |
| `--trace` | `false` | Print execution trace |

### Examples

```bash
# Basic execution
go-json run hello.json

# With inline input
go-json run greet.json --input '{"name": "Alice", "age": 30}'

# With input from file
go-json run process.json --input-file data.json

# With I/O modules enabled
go-json run fetch.json --io http
go-json run script.json --io all

# With execution trace
go-json run program.json --trace

# With custom timeout
go-json run long_task.json --timeout 120s

# From stdin (pipe)
echo '{"name": "Bob"}' | go-json run greet.json
```

### Input Sources

Input is read from exactly one source (mutually exclusive):

1. `--input` — inline JSON string
2. `--input-file` — path to JSON file
3. **stdin** — if piped and neither flag is provided

### Output

- Program return value is printed to stdout as JSON
- Errors are printed to stderr
- Exit code: `0` on success, `1` on error

---

## `go-json serve`

Start a web server from a server program (a program with `routes`).

```bash
go-json serve <program.json> [options]
```

### Options

| Flag | Default | Description |
|------|---------|-------------|
| `--port <n>` | `3000` | Listen port |
| `--host <addr>` | `0.0.0.0` | Listen host |
| `--dev` | `false` | Dev mode (template reload, verbose logging) |
| `--docs` | `false` | Enable Swagger UI at `/docs` |
| `--io <modules>` | — | Enable I/O modules |

### Examples

```bash
# Start server
go-json serve api.json

# Custom port
go-json serve api.json --port 8080

# Dev mode with Swagger UI
go-json serve api.json --dev --docs

# With I/O modules
go-json serve api.json --io http,fs,sql
```

### Built-in Endpoints

| Endpoint | Description |
|----------|-------------|
| `/health` | Health check (bypasses all middleware) |
| `/docs` | Swagger UI (when `--docs` is enabled) |

### Graceful Shutdown

The server handles `SIGINT` and `SIGTERM` for graceful shutdown. Configurable timeout via `server.graceful_shutdown` (default: `10s`).

---

## `go-json check`

Validate a program without executing it. Performs full compilation including import resolution, struct validation, and structural checks.

```bash
go-json check <program.json> [options]
```

### Options

| Flag | Default | Description |
|------|---------|-------------|
| `--verbose` | `false` | Show detailed validation info |

### Examples

```bash
# Basic validation
go-json check program.json

# Verbose output
go-json check program.json --verbose
```

### Exit Codes

- `0` — program is valid
- `1` — compilation errors found

---

## `go-json test`

Run test files. Test files are go-json programs with `"test": true` and a `"cases"` array.

```bash
go-json test <path> [options]
```

### Options

| Flag | Default | Description |
|------|---------|-------------|
| `--filter <pattern>` | — | Run only cases matching pattern |
| `--verbose` | `false` | Show input/output per case |

### Examples

```bash
# Run all tests in directory
go-json test tests/

# Run specific test file
go-json test tests/test_math.json

# Filter by case name
go-json test tests/ --filter "factorial"

# Verbose output
go-json test tests/ --verbose
```

### Test File Format

```json
{
  "name": "test_math",
  "test": true,
  "import": {"math": "../functions/math.json"},
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

### Output

```
✓ test_math: Factorial of 5 is 120 (1ms)
✓ test_math: Factorial of 0 is 1 (0ms)
2 passed, 0 failed
```

### Comparison Rules

| Type | Comparison |
|------|-----------|
| Object | Deep equality |
| Array | Deep equality with order |
| Float | Tolerance of 1e-9 (so `0.1 + 0.2` vs `0.3` passes) |
| Nil | Result must be nil |

---

## `go-json ast`

Export the program's Abstract Syntax Tree as JSON.

```bash
go-json ast <program.json> [options]
```

### Options

| Flag | Default | Description |
|------|---------|-------------|
| `--output <path>` | stdout | Write AST to file |
| `--format <fmt>` | `json` | Output format (currently only `json` supported) |

### Examples

```bash
# Print AST to stdout
go-json ast program.json

# Save to file
go-json ast program.json --output ast.json
```

---

## `go-json codegen`

Generate native source code from a go-json program.

```bash
go-json codegen <program.json> [options]
```

### Options

| Flag | Default | Description |
|------|---------|-------------|
| `--target <lang>` | `go` | Target language: `go`, `javascript`, `python` |
| `--output <path>` | stdout | Output file path |
| `--package <name>` | `main` | Package name (Go only) |
| `--framework <name>` | varies | Server framework (for server programs) |

### Default Frameworks per Language

| Language | Default Framework |
|----------|-------------------|
| Go | fiber |
| JavaScript | express |
| Python | fastapi |

### Examples

```bash
# Generate Go code
go-json codegen program.json --target go --output program.go

# Generate JavaScript
go-json codegen program.json --target javascript --output program.js

# Generate Python
go-json codegen program.json --target python --output program.py

# Server codegen with specific framework
go-json codegen api.json --target go --framework fiber --output server.go
go-json codegen api.json --target javascript --framework express --output server.js
```

---

## `go-json generate`

Scaffold projects, CRUD APIs, and auth endpoints.

```bash
go-json generate <type> [options]
```

### Subcommands

#### `go-json generate crud`

Generate CRUD API for a database table.

| Flag | Description |
|------|-------------|
| `--from-db` | Use database introspection |
| `--dsn <dsn>` | Database connection string |
| `--table <name>` | Table name to introspect |
| `--fields <spec>` | Manual field definition (`name:string,age:int`) |
| `--auth` | Include auth middleware on write endpoints |
| `--output <dir>` | Output directory |

```bash
# From database
go-json generate crud --from-db --dsn "postgres://localhost/mydb" --table users

# Manual fields
go-json generate crud --fields "name:string,email:string,role:string" --auth
```

#### `go-json generate auth`

Generate authentication endpoints (register, login, refresh, me, change-password).

```bash
go-json generate auth --output ./auth/
```

#### `go-json generate project`

Generate a full project scaffold.

```bash
go-json generate project --name my-api --output ./my-api/
```

Generated structure:
```
my-api/
├── api.json          # Main server program
├── functions/        # Handler functions
├── templates/        # HTML templates
├── public/           # Static files
├── migrations/       # SQL migrations
├── tests/            # Test files
├── .env.example      # Environment variables
└── README.md         # Project documentation
```

---

## `go-json openapi`

Generate OpenAPI 3.0 specification from a server program.

```bash
go-json openapi <program.json> [options]
```

### Options

| Flag | Default | Description |
|------|---------|-------------|
| `--output <path>` | stdout | Output file path |

### Examples

```bash
# Print to stdout
go-json openapi api.json

# Save to file
go-json openapi api.json --output openapi.json
```

---

## `go-json migrate`

Migrate deprecated syntax to a newer version.

```bash
go-json migrate <program.json> [options]
```

### Options

| Flag | Default | Description |
|------|---------|-------------|
| `--from <version>` | auto-detect | Source version |
| `--to <version>` | latest | Target version |
| `--dry-run` | `false` | Show changes without applying |
| `--output <path>` | in-place | Write to different file |

### Examples

```bash
# Auto-detect and migrate
go-json migrate program.json

# Dry run (preview changes)
go-json migrate program.json --dry-run

# Write to new file
go-json migrate program.json --output migrated.json
```

The migration tool uses **JSON-aware key renaming** (not blind string replace) and preserves JSONC comments.

---

## Global Flags

| Flag | Description |
|------|-------------|
| `--version`, `-v` | Print version |
| `--help`, `-h` | Print help |

## Exit Codes

| Code | Meaning |
|------|---------|
| `0` | Success |
| `1` | Error (compilation, runtime, or validation failure) |
