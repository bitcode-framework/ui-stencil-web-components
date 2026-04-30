# Code Generation

go-json programs can be transpiled to native Go, JavaScript, or Python source code. Server programs can additionally generate framework-specific server code with dependency management.

## Program Codegen

Generate native code from any go-json program:

```bash
go-json codegen program.json --target go --output program.go
go-json codegen program.json --target javascript --output program.js
go-json codegen program.json --target python --output program.py
```

### Language Mappings

| go-json | Go | JavaScript | Python |
|---------|-----|-----------|--------|
| `let` | `:=` | `const` / `let` | `=` |
| `set` | `=` | `=` | `=` |
| `if`/`elif`/`else` | `if`/`else if`/`else` | `if`/`else if`/`else` | `if`/`elif`/`else` |
| `switch` | `switch` | `switch` | `if`/`elif`/`else` chain |
| `for`/`in` | `for _, v := range` | `for...of` | `for v in` |
| `for`/`range` | `for i := start; i < end; i++` | `for (let i = start; i < end; i++)` | `for i in range(start, end)` |
| `while` | `for condition` | `while (condition)` | `while condition:` |
| `try`/`catch` | `if err != nil` | `try`/`catch` | `try`/`except` |
| `parallel` | goroutines + `sync.WaitGroup` | `Promise.all` | `asyncio.gather` |
| `struct` | Go struct | `class` | `@dataclass` |
| `function` | `func` | `function` / `const fn =` | `def` with type hints |
| `_c` | `//` comment | `//` comment | `#` comment |

### Example

**go-json source:**

```json
{
  "name": "factorial",
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

**Generated Go:**

```go
package main

func factorial(n int) int {
    if n <= 1 {
        return 1
    }
    sub := factorial(n - 1)
    return n * sub
}

func main() {
    result := factorial(10)
    fmt.Println(result)
}
```

**Generated JavaScript:**

```javascript
function factorial(n) {
    if (n <= 1) {
        return 1;
    }
    const sub = factorial(n - 1);
    return n * sub;
}

const result = factorial(10);
console.log(result);
```

**Generated Python:**

```python
def factorial(n: int) -> int:
    if n <= 1:
        return 1
    sub = factorial(n - 1)
    return n * sub

result = factorial(10)
print(result)
```

### Limitations

| Limitation | Reason |
|-----------|--------|
| Dynamic types may need type assertions | go-json allows `any`, target languages may not |
| Extension calls (`ext:*`) not portable | Host-specific — emitted as `// TODO` comments |
| I/O calls use different libraries per language | HTTP client, FS API differ per language |
| Parallel has different concurrency models | Go: goroutines, JS: Promise.all, Python: asyncio |

Code generation works best for **pure logic** programs. I/O-heavy programs need manual adaptation of the generated code.

---

## Server Codegen

For server programs (programs with `routes`), generate framework-specific server code:

```bash
go-json codegen api.json --target go --framework fiber
go-json codegen api.json --target javascript --framework express
go-json codegen api.json --target python --framework fastapi
```

### Available Frameworks

| Language | Frameworks | Default |
|----------|-----------|---------|
| Go | fiber, net/http, echo, gin, chi | fiber |
| JavaScript | express, hono, fastify, koa | express |
| Python | fastapi, flask, django | fastapi |

### Generated Files

Server codegen produces:

- **Main entry point** — `main.go` / `index.js` / `main.py` (server setup, route registration)
- **Handlers** — `handlers.go` / `routes.js` / `routes.py` (handler functions)
- **Middleware** — `middleware.go` / `middleware.js` / `middleware.py` (custom middleware)
- **Types** — `types.go` (Go only — request/response structs)

### Dependency Management

Codegen automatically detects features used in the program and generates dependency files:

| Language | File | Contents |
|----------|------|----------|
| Go | `go.mod` | Module definition + required packages |
| JavaScript | `package.json` | Dependencies |
| Python | `requirements.txt` | pip packages |
| All | `.env.example` | Required environment variables |

Feature detection scans for: JWT usage, database access, file operations, template rendering, CORS config, rate limiting, etc.

---

## CRUD Generator

Generate a complete CRUD API from a database table or manual field definition.

### From Database Introspection

```bash
go-json generate crud --table users --dsn "postgres://user:pass@localhost/mydb"
```

Supported databases:
- **SQLite** — reads `PRAGMA table_info`, `PRAGMA foreign_key_list`
- **PostgreSQL** — reads `information_schema.columns`, `pg_constraint`
- **MySQL** — reads `information_schema.columns`, `information_schema.key_column_usage`

### From Manual Fields

```bash
go-json generate crud --table users --fields "name:string,email:string,age:int,role:string" --auth
```

### What Gets Generated

A complete go-json server program with:

- **List endpoint** — `GET /api/<table>` with pagination, search, filtering
- **Get endpoint** — `GET /api/<table>/:id`
- **Create endpoint** — `POST /api/<table>` with validation
- **Update endpoint** — `PUT /api/<table>/:id` with validation
- **Delete endpoint** — `DELETE /api/<table>/:id`

### Smart Field Detection

When using `--from-db`, the generator reads database metadata to produce intelligent output:

| DB Metadata | Generated Behavior |
|-------------|-------------------|
| `NOT NULL` without default | Required field + validation |
| `SERIAL` / auto-increment | Excluded from create/update body |
| `DEFAULT NOW()` | Excluded from create body |
| `VARCHAR(255)` | Length validation |
| `REFERENCES other(id)` | Foreign key validation |
| `UNIQUE` | Duplicate check before insert |
| `ENUM('a','b','c')` | Enum validation |

### Type Mapping

| DB Type | go-json Type | OpenAPI Type |
|---------|-------------|-------------|
| `integer`, `bigint` | `int` | `integer` |
| `real`, `double` | `float` | `number` |
| `boolean` | `bool` | `boolean` |
| `varchar`, `text` | `string` | `string` |
| `timestamp` | `string` | `string` (format: date-time) |
| `json`, `jsonb` | `any` | `object` |
| `uuid` | `string` | `string` (format: uuid) |
| `enum` | `string` | `string` (enum: [...]) |

### Relationship Detection

Foreign keys are detected and generate relationship endpoints:

```
GET /api/users/:id/orders    # One-to-many (users → orders)
```

Junction tables (tables with exactly 2 foreign keys) generate many-to-many endpoints.

---

## Auth Scaffold

Generate authentication endpoints:

```bash
go-json generate auth --output ./auth/
```

Generates:

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/api/auth/register` | POST | User registration |
| `/api/auth/login` | POST | Login (returns JWT) |
| `/api/auth/refresh` | POST | Refresh token |
| `/api/auth/me` | GET | Current user profile |
| `/api/auth/change-password` | POST | Change password |

Includes JWT configuration, password hashing, and users table migration SQL.

---

## Project Scaffold

Generate a complete project structure:

```bash
go-json generate project --name my-api --output ./my-api/
```

Generated structure:

```
my-api/
├── api.json              # Main server program
├── functions/            # Handler functions (organized by domain)
├── templates/            # HTML templates
│   ├── layouts/          # Layout templates
│   ├── partials/         # Reusable partials
│   └── pages/            # Page templates
├── public/               # Static files (CSS, JS, images)
├── migrations/           # SQL migration files
├── tests/                # Test files
├── .env.example          # Environment variable template
└── README.md             # Project documentation
```

---

## Architecture Patterns

The generator supports 4 architecture patterns for organizing generated code:

### Simple (default)

Single-file or flat structure. Best for small APIs.

```
api.json                  # Everything in one file
```

### Service Layer

Separates handlers from business logic. Best for medium APIs.

```
handlers/                 # HTTP handlers
services/                 # Business logic
queries/                  # Database queries
```

### DDD (Domain-Driven Design)

Organized by domain. Best for complex applications.

```
cmd/                      # Entry point
internal/
  domain/                 # Domain models and interfaces
  application/            # Use cases / application services
  infrastructure/         # Database, external services
```

### Hexagonal

Ports and adapters architecture. Best for highly testable applications.

```
cmd/                      # Entry point
internal/
  core/
    ports/                # Interfaces (inbound + outbound)
    services/             # Business logic
  adapters/
    inbound/              # HTTP handlers, CLI
    outbound/             # Database, external APIs
```

### Using Patterns

```bash
# Use built-in pattern
go-json generate crud --table users --dsn "..." --pattern service-layer

# Export pattern for customization
go-json generate --export-pattern ddd --output ./my-templates/ddd/

# Use custom pattern
go-json generate crud --table users --fields "name:string" --pattern ./my-templates/ddd/
```

### Custom Templates

Pattern templates use Go `text/template` syntax with these variables:

| Variable | Scope | Description |
|----------|-------|-------------|
| `{{.ProjectName}}` | All | Project name |
| `{{.Models}}` | `once` files | List of all table models |
| `{{.Model}}` | `per_model` files | Current table model |
| `{{.Model.Name}}` | `per_model` files | Table name |
| `{{.Model.Columns}}` | `per_model` files | Column definitions |
| `{{.Model.PrimaryKey}}` | `per_model` files | Primary key columns |
| `{{.Model.ForeignKeys}}` | `per_model` files | Foreign key relationships |

Available template functions: `lower`, `upper`, `title`, `capitalize`, `singular`, `plural`, `snake`, `camel`, `pascal`.

Each pattern has a `template.json` metadata file defining `once` files (generated once per project) and `per_model` files (generated per table/model).

### Built-in Pattern Templates

All 4 patterns include ready-to-use template files:

| Pattern | Template Files | Description |
|---------|---------------|-------------|
| `simple` | `api.json.tmpl` | Single go-json file with all routes |
| `service-layer` | `api.json.tmpl`, `service.json.tmpl` | Routes + per-model service files |
| `ddd` | `main.go.tmpl`, `entity.go.tmpl`, `repository.go.tmpl` | Go DDD structure |
| `hexagonal` | `main.go.tmpl`, `port.go.tmpl`, `service.go.tmpl` | Go hexagonal structure |
