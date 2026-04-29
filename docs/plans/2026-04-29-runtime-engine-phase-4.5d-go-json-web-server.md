# Phase 4.5d — go-json Web Server

**Status**: Draft
**Date**: 29 April 2026
**Depends on**: Phase 4.5c (I/O Modules, Extensions, CLI)
**Blocks**: Phase 7 (Module "setting")

---

## §1. Overview

Phase 4.5d adds a built-in web server execution mode to go-json. Programs can declaratively define HTTP routes, middleware chains, and handlers — all in JSON. The server runtime handles HTTP lifecycle (listen, accept, route, respond) while go-json handles per-request logic via the existing run-to-completion execution model.

### 1.1 Design Principles

1. **Handler = run-to-completion** — Each HTTP request triggers one `Execute()` call. No event loops, no async, no callbacks. Same execution model as `go-json run`.
2. **Declarative routing** — Routes defined in JSON, not imperative code. Framework handles matching.
3. **Plugable framework** — Default runtime framework is Fiber (matching bitcode engine). User can choose net/http, Echo, Gin, or Chi via config. Adapter pattern abstracts framework differences. Codegen framework selected independently via `--framework` flag.
4. **Middleware as functions** — Built-in middleware (logger, recover, CORS, JWT) plus custom middleware written as go-json functions.
5. **Template rendering** — Go's `html/template` for server-side HTML. Auto-escape for XSS protection.
6. **Codegen per framework** — `go-json codegen --target go --framework fiber` generates framework-specific code.

### 1.2 Non-Goals (Phase 4.5d)

- WebSocket support (future)
- Server-Sent Events (future)
- REPL mode (future)
- Hot module replacement (future — dev mode only does file-watch restart)
- GraphQL (future)
- gRPC (future)

---

## §2. Program Format

### 2.1 Top-Level Structure

```json
{
  "name": "my-api",
  "go_json": "1",
  "server": {
    "framework": "fiber",
    "port": 3000,
    "host": "0.0.0.0",
    "static": "./public",
    "templates": "./templates",
    "cors": {
      "origins": ["*"],
      "methods": ["GET", "POST", "PUT", "DELETE"],
      "headers": ["Content-Type", "Authorization"],
      "max_age": 86400
    },
    "jwt": {
      "secret_env": "JWT_SECRET",
      "algorithm": "HS256",
      "expiry": "24h",
      "cookie": "token",
      "header": "Authorization",
      "prefix": "Bearer"
    },
    "rate_limit": {
      "requests": 100,
      "window": "1m",
      "by": "ip"
    },
    "graceful_shutdown": "10s",
    "read_timeout": "30s",
    "write_timeout": "30s",
    "max_body_size": "10mb"
  },
  "middleware": ["logger", "recover", "cors"],
  "routes": [...],
  "functions": {...},
  "structs": {...}
}
```

### 2.2 Server Config Fields

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `framework` | string | `"fiber"` | Runtime server framework: `"fiber"`, `"net/http"`, `"echo"`, `"gin"`, `"chi"`. Only affects `go-json serve`, not codegen. |
| `port` | int | `3000` | Listen port |
| `host` | string | `"0.0.0.0"` | Listen host |
| `static` | string | `""` | Static file directory (relative to program file) |
| `templates` | string | `""` | Template directory (relative to program file) |
| `cors` | object | `null` | CORS configuration |
| `jwt` | object | `null` | JWT configuration |
| `rate_limit` | object | `null` | Rate limiting configuration |
| `graceful_shutdown` | string | `"10s"` | Graceful shutdown timeout |
| `read_timeout` | string | `"30s"` | HTTP read timeout |
| `write_timeout` | string | `"30s"` | HTTP write timeout |
| `max_body_size` | string | `"10mb"` | Maximum request body size |

### 2.3 Framework Selection

```json
{"server": {"framework": "fiber"}}
```

Available frameworks:

| Framework | Package | Notes |
|-----------|---------|-------|
| `net/http` | stdlib | Zero dependency, default, Go 1.22+ ServeMux with patterns |
| `fiber` | `github.com/gofiber/fiber/v2` | Fast, Express-like, used by bitcode engine |
| `echo` | `github.com/labstack/echo/v4` | Minimalist, good middleware ecosystem |
| `gin` | `github.com/gin-gonic/gin` | Popular, battle-tested |
| `chi` | `github.com/go-chi/chi/v5` | Lightweight, stdlib-compatible |

Framework is selected at **build time** for CLI (`go-json serve`) or at **embed time** for library users. The `ServerAdapter` interface abstracts all framework differences.

---

## §3. Route Definitions

### 3.1 Basic Routes

```json
{
  "routes": [
    {"method": "GET",    "path": "/api/users",      "handler": "listUsers"},
    {"method": "POST",   "path": "/api/users",      "handler": "createUser"},
    {"method": "GET",    "path": "/api/users/:id",   "handler": "getUser"},
    {"method": "PUT",    "path": "/api/users/:id",   "handler": "updateUser"},
    {"method": "DELETE", "path": "/api/users/:id",   "handler": "deleteUser"}
  ]
}
```

### 3.2 Route with Middleware

```json
{
  "routes": [
    {"method": "GET",  "path": "/api/public",  "handler": "publicData"},
    {"method": "GET",  "path": "/api/profile", "handler": "getProfile", "middleware": ["jwt"]},
    {"method": "POST", "path": "/api/admin",   "handler": "adminAction", "middleware": ["jwt", "requireAdmin"]}
  ]
}
```

### 3.3 Route with Template Rendering

```json
{
  "routes": [
    {"method": "GET", "path": "/",           "handler": "homePage",    "render": "pages/home.html"},
    {"method": "GET", "path": "/users",      "handler": "usersPage",   "render": "pages/users.html"},
    {"method": "GET", "path": "/users/:id",  "handler": "userDetail",  "render": "pages/user-detail.html"}
  ]
}
```

### 3.4 Route Groups

```json
{
  "routes": [
    {
      "prefix": "/api/v1",
      "middleware": ["jwt"],
      "routes": [
        {"method": "GET",  "path": "/users",     "handler": "listUsers"},
        {"method": "POST", "path": "/users",     "handler": "createUser"},
        {"method": "GET",  "path": "/users/:id", "handler": "getUser"}
      ]
    },
    {
      "prefix": "/api/public",
      "routes": [
        {"method": "GET", "path": "/health", "handler": "healthCheck"},
        {"method": "GET", "path": "/version", "handler": "getVersion"}
      ]
    }
  ]
}
```

### 3.5 Route Definition Fields

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `method` | string | yes (leaf) | HTTP method: GET, POST, PUT, PATCH, DELETE, OPTIONS, HEAD |
| `path` | string | yes | URL path with params (`:id`, `*wildcard`) |
| `handler` | string | yes (leaf) | Function name to handle request |
| `middleware` | []string | no | Route-specific middleware chain |
| `render` | string | no | Template file path (relative to `templates` dir) |
| `prefix` | string | group only | URL prefix for route group |
| `routes` | []route | group only | Nested routes in group |

---

## §4. Request Object

Every handler receives a `request` object in scope:

```json
{
  "request": {
    "method": "POST",
    "path": "/api/users/123",
    "url": "http://localhost:3000/api/users/123?include=orders",
    "params": {"id": "123"},
    "query": {"include": "orders"},
    "headers": {
      "Content-Type": "application/json",
      "Authorization": "Bearer eyJ...",
      "User-Agent": "Mozilla/5.0..."
    },
    "body": {"name": "Alice", "email": "alice@example.com"},
    "cookies": {"session": "abc123"},
    "ip": "192.168.1.100",
    "user": null
  }
}
```

### 4.1 Request Fields

| Field | Type | Description |
|-------|------|-------------|
| `method` | string | HTTP method (uppercase) |
| `path` | string | Request path (without query string) |
| `url` | string | Full request URL |
| `params` | map | Path parameters (`:id` → `params.id`) |
| `query` | map | Query string parameters |
| `headers` | map | Request headers (case-insensitive keys normalized to canonical form) |
| `body` | any | Parsed request body (JSON → map, form → map, raw → string) |
| `cookies` | map | Request cookies |
| `ip` | string | Client IP address |
| `user` | any | Set by JWT middleware (decoded token payload) |

### 4.2 Body Parsing

Body is auto-parsed based on Content-Type:

| Content-Type | Parsed As |
|---|---|
| `application/json` | `map[string]any` or `[]any` |
| `application/x-www-form-urlencoded` | `map[string]string` |
| `multipart/form-data` | `map[string]any` (files as metadata) |
| other / missing | raw `string` |

---

## §5. Response Convention

Handler return value determines the HTTP response:

### 5.1 JSON API Response

```json
{"return": {"status": 200, "body": {"users": "data", "total": "count"}}}
```

→ Sets `Content-Type: application/json`, marshals `body` to JSON.

### 5.2 Template Render Response

```json
{"return": {"data": {"users": "users_list", "title": "'User List'"}}}
```

→ Route must have `"render"` field. Template rendered with `data` map. Sets `Content-Type: text/html`.

### 5.3 Redirect Response

```json
{"return": {"redirect": "/login"}}
{"return": {"redirect": "/dashboard", "status": 301}}
```

→ Default status 302 (temporary redirect).

### 5.4 Custom Headers Response

```json
{"return": {"status": 200, "body": "'csv data'", "headers": {"Content-Type": "text/csv", "Content-Disposition": "attachment; filename=export.csv"}}}
```

### 5.5 Error Response

```json
{"error": "'User not found'"}
```

→ Returns 500 with `{"error": {"message": "User not found"}}`.

Structured error with explicit status:

```json
{"return": {"status": 404, "body": {"error": "user not found"}}}
```

Or use the `error` step with code (status inferred from code pattern, see §16.4):

```json
{"error": {"code": "'NOT_FOUND'", "message": "'User not found'"}}
```

→ Returns 404 because `NOT_FOUND` maps to 404 (see §16.4 for full mapping).

### 5.6 No Return (204 No Content)

If handler has no `return` step or returns `nil`:
→ Returns 204 No Content.

### 5.7 Response Field Summary

| Return Field | Type | Description |
|---|---|---|
| `status` | int | HTTP status code (default 200) |
| `body` | any | Response body (auto JSON-serialized if map/array) |
| `headers` | map | Custom response headers |
| `redirect` | string | Redirect URL |
| `data` | map | Template data (requires route `render` field) |
| `cookies` | []object | Set-Cookie entries: `[{"name": "x", "value": "y", "max_age": 3600}]` |

---

## §6. Middleware

### 6.1 Built-in Middleware

| Name | Description | Config |
|------|-------------|--------|
| `logger` | Log every request (method, path, status, duration) | None |
| `recover` | Catch panics, return 500 with error | None |
| `cors` | CORS headers from `server.cors` config | `server.cors` |
| `jwt` | Validate JWT token, inject `request.user` | `server.jwt` |
| `rate_limit` | Rate limiting per IP/user | `server.rate_limit` |
| `request_id` | Add `X-Request-ID` header | None |
| `compress` | Gzip/Brotli response compression | None |
| `secure` | Security headers (HSTS, X-Frame-Options, etc.) | None |

### 6.2 JWT Middleware Detail

When `jwt` middleware is applied to a route:

1. Extract token from `Authorization: Bearer <token>` header OR cookie (configurable)
2. Validate signature using secret from env var (`server.jwt.secret_env`)
3. Check expiry
4. If valid: decode payload → inject into `request.user`
5. If invalid/missing: return 401 `{"error": "unauthorized"}`

```json
{
  "server": {
    "jwt": {
      "secret_env": "JWT_SECRET",
      "algorithm": "HS256",
      "expiry": "24h",
      "cookie": "token",
      "header": "Authorization",
      "prefix": "Bearer",
      "claims": {
        "issuer": "my-app",
        "audience": "api"
      }
    }
  }
}
```

After JWT validation, handler can access:

```json
{"let": "user_id", "expr": "request.user.sub"},
{"let": "email", "expr": "request.user.email"},
{"let": "roles", "expr": "request.user.roles"}
```

### 6.3 JWT Token Generation

For login endpoints, handlers can generate tokens:

```json
{
  "functions": {
    "login": {
      "steps": [
        {"let": "body", "expr": "request.body"},
        {"let": "user", "call": "sql.query", "with": {
          "query": "SELECT * FROM users WHERE email = ? AND password_hash = ?",
          "args": ["body.email", "crypto.hash(body.password)"]
        }},
        {"if": "len(user.rows) == 0", "then": [
          {"return": {"status": 401, "body": {"error": "invalid credentials"}}}
        ]},
        {"let": "token", "call": "jwt.sign", "with": {
          "payload": {"sub": "user.rows[0].id", "email": "user.rows[0].email", "role": "user.rows[0].role"},
          "expiry": "'24h'"
        }},
        {"return": {"status": 200, "body": {"token": "token"}, "cookies": [{"name": "token", "value": "token", "http_only": true, "max_age": 86400}]}}
      ]
    }
  }
}
```

### 6.4 Custom Middleware (go-json Functions)

Custom middleware is a go-json function that receives `request` and can:
- Modify `request` (add fields like `request.user`)
- Return early (short-circuit with error response)
- Do nothing (pass through)

```json
{
  "functions": {
    "requireAdmin": {
      "steps": [
        {"if": "request.user == nil", "then": [
          {"return": {"status": 401, "body": {"error": "unauthorized"}}}
        ]},
        {"if": "request.user.role != 'admin'", "then": [
          {"return": {"status": 403, "body": {"error": "forbidden — admin required"}}}
        ]}
      ]
    },
    "addRequestTiming": {
      "steps": [
        {"set": "request.store.start_time", "expr": "now()"}
      ]
    }
  }
}
```

### 6.5 Middleware Execution Order

```
Global middleware (from top-level "middleware" array)
  → Group middleware (from route group "middleware")
    → Route middleware (from route "middleware")
      → Handler function
```

If any middleware returns a response (has `return` with `status`/`body`/`redirect`), the chain stops and that response is sent.

---

## §7. Template Engine

### 7.1 Go html/template

Templates use Go's `html/template` syntax:

```html
<!-- templates/pages/users.html -->
{{template "layout" .}}

{{define "content"}}
<h1>{{.title}}</h1>
<table>
  <tr><th>Name</th><th>Email</th></tr>
  {{range .users}}
  <tr>
    <td><a href="/users/{{.id}}">{{.name}}</a></td>
    <td>{{.email}}</td>
  </tr>
  {{end}}
</table>
{{end}}
```

```html
<!-- templates/layouts/layout.html -->
<!DOCTYPE html>
<html>
<head><title>{{.title}} — My App</title></head>
<body>
  <nav>...</nav>
  <main>{{template "content" .}}</main>
  <footer>...</footer>
</body>
</html>
```

### 7.2 Template Features

- **Layouts** — `{{template "layout" .}}` for shared chrome
- **Partials** — `{{template "partial_name" .}}` for reusable components
- **Auto-escape** — HTML content auto-escaped (XSS protection)
- **Custom functions** — `{{formatDate .created_at "2006-01-02"}}`, `{{json .data}}`
- **Conditionals** — `{{if .user}}...{{else}}...{{end}}`
- **Loops** — `{{range .items}}...{{end}}`

### 7.3 Template Directory Structure

```
templates/
├── layouts/
│   └── layout.html
├── partials/
│   ├── header.html
│   └── footer.html
└── pages/
    ├── home.html
    ├── users.html
    └── user-detail.html
```

### 7.4 Template Functions (Built-in)

| Function | Description | Example |
|----------|-------------|---------|
| `json` | Marshal to JSON string | `{{json .data}}` |
| `formatDate` | Format time | `{{formatDate .date "2006-01-02"}}` |
| `upper` | Uppercase | `{{upper .name}}` |
| `lower` | Lowercase | `{{lower .email}}` |
| `truncate` | Truncate string | `{{truncate .desc 100}}` |
| `default` | Default value | `{{default .title "Untitled"}}` |
| `safeHTML` | Mark as safe (no escape) | `{{safeHTML .rendered_markdown}}` |
| `urlEncode` | URL encode | `{{urlEncode .query}}` |
| `add` | Integer addition | `{{add .page 1}}` |
| `sub` | Integer subtraction | `{{sub .total .used}}` |
| `mul` | Integer multiplication | `{{mul .price .qty}}` |
| `seq` | Generate sequence | `{{range seq 1 .totalPages}}` |

---

## §8. Static Files

### 8.1 Configuration

```json
{"server": {"static": "./public"}}
```

Files in `./public/` served at root path:
- `./public/css/style.css` → `GET /css/style.css`
- `./public/js/app.js` → `GET /js/app.js`
- `./public/images/logo.png` → `GET /images/logo.png`

### 8.2 Custom Static Prefix

```json
{"server": {"static": {"dir": "./assets", "prefix": "/static"}}}
```

→ `./assets/style.css` → `GET /static/style.css`

### 8.3 Static File Security

- Directory traversal blocked (no `../`)
- Hidden files (`.env`, `.git`) not served
- Optional: file extension whitelist

---

## §9. Server Adapter Interface

### 9.1 Interface Definition

```go
type HandlerFunc func(ctx *RequestContext) *Response

type MiddlewareFunc func(ctx *RequestContext, next func()) *Response

type ServerAdapter interface {
    RegisterRoute(method, path string, handler HandlerFunc, middleware ...MiddlewareFunc)
    RegisterGroup(prefix string, middleware []MiddlewareFunc, routes []RouteConfig)
    RegisterStatic(prefix, dir string)
    SetErrorHandler(handler func(err error, ctx *RequestContext) *Response)
    Listen(addr string) error
    Shutdown(ctx context.Context) error
}
```

### 9.2 RequestContext

```go
type RequestContext struct {
    Method  string
    Path    string
    URL     string
    Params  map[string]string
    Query   map[string]string
    Headers map[string]string
    Body    any
    Cookies map[string]string
    IP      string
    User    any                // set by JWT middleware
    Store   map[string]any    // per-request storage for middleware
}
```

### 9.3 Response

```go
type Response struct {
    Status   int
    Body     any
    Headers  map[string]string
    Redirect string
    Template string            // template path to render
    Data     map[string]any    // template data
    Cookies  []CookieConfig
}
```

### 9.4 Adapter Implementations

Each framework adapter translates between the framework's native types and `RequestContext`/`Response`:

```go
// adapters/nethttp/adapter.go
type NetHTTPAdapter struct { ... }

// adapters/fiber/adapter.go
type FiberAdapter struct { ... }

// adapters/echo/adapter.go
type EchoAdapter struct { ... }

// adapters/gin/adapter.go
type GinAdapter struct { ... }

// adapters/chi/adapter.go
type ChiAdapter struct { ... }
```

---

## §10. JWT Module

JWT is both a middleware AND a callable module for token generation:

### 10.1 As Middleware

Applied to routes — validates incoming tokens, injects `request.user`.

### 10.2 As Callable Functions

Available in handler steps:

```
jwt.sign(payload, expiry)     → generate token string
jwt.verify(token)             → decode and validate, return payload or error
jwt.decode(token)             → decode WITHOUT validation (for debugging)
jwt.refresh(token, expiry)    → generate new token with extended expiry
```

### 10.3 Example: Full Auth Flow

```json
{
  "name": "auth-api",
  "server": {
    "port": 3000,
    "jwt": {"secret_env": "JWT_SECRET", "algorithm": "HS256", "expiry": "24h"}
  },
  "middleware": ["logger", "recover", "cors"],
  "routes": [
    {"method": "POST", "path": "/auth/login",    "handler": "login"},
    {"method": "POST", "path": "/auth/register", "handler": "register"},
    {"method": "POST", "path": "/auth/refresh",  "handler": "refreshToken", "middleware": ["jwt"]},
    {"method": "GET",  "path": "/api/me",        "handler": "getProfile",   "middleware": ["jwt"]}
  ],
  "functions": {
    "login": {
      "steps": [
        {"let": "body", "expr": "request.body"},
        {"let": "users", "call": "sql.query", "with": {
          "query": "SELECT id, email, role, password_hash FROM users WHERE email = ?",
          "args": ["body.email"]
        }},
        {"if": "len(users.rows) == 0", "then": [
          {"return": {"status": 401, "body": {"error": "invalid credentials"}}}
        ]},
        {"let": "user", "expr": "users.rows[0]"},
        {"let": "valid", "call": "crypto.verify", "with": {"value": "body.password", "hash": "user.password_hash"}},
        {"if": "!valid", "then": [
          {"return": {"status": 401, "body": {"error": "invalid credentials"}}}
        ]},
        {"let": "token", "call": "jwt.sign", "with": {
          "payload": {"sub": "user.id", "email": "user.email", "role": "user.role"}
        }},
        {"return": {"status": 200, "body": {"token": "token", "user": {"id": "user.id", "email": "user.email"}}}}
      ]
    },
    "getProfile": {
      "steps": [
        {"let": "user", "call": "sql.query", "with": {
          "query": "SELECT id, name, email, role FROM users WHERE id = ?",
          "args": ["request.user.sub"]
        }},
        {"if": "len(user.rows) == 0", "then": [
          {"return": {"status": 404, "body": {"error": "user not found"}}}
        ]},
        {"return": {"status": 200, "body": "user.rows[0]"}}
      ]
    }
  }
}
```

---

## §11. Dev Mode

```bash
go-json serve api.json --dev
```

Dev mode enables:
- **File watching** — restart server on `.json` file changes
- **Pretty errors** — stack trace + source context in error responses
- **Template reload** — templates re-parsed on every request (no cache)
- **No compression** — easier debugging
- **Verbose logging** — log request/response bodies

Production mode (default, no `--dev`):
- Template caching
- Compressed responses
- Minimal error messages (no stack traces)
- Graceful shutdown on SIGTERM/SIGINT

---

## §12. Codegen for Server

### 12.1 Two Dimensions: Target Language × Framework

Codegen target language and framework are **independent choices**:

```bash
# Go + Fiber (default for Go)
go-json codegen api.json --target go

# Go + Gin (override)
go-json codegen api.json --target go --framework gin

# JavaScript + Express (default for JS)
go-json codegen api.json --target js

# JavaScript + Hono (override)
go-json codegen api.json --target js --framework hono

# Python + FastAPI (default for Python)
go-json codegen api.json --target python
```

### 12.2 Framework Selection Logic

`--framework` flag selects the codegen framework. If omitted:

1. If `server.framework` in JSON is compatible with target language → use it
2. Otherwise → use default for that language

**Defaults per language:**

| Language | Default Framework |
|----------|-------------------|
| Go | `fiber` |
| JavaScript | `express` |
| Python | `fastapi` |

**Compatibility matrix:**

| `server.framework` | `--target go` | `--target js` | `--target python` |
|---------------------|---------------|----------------|-------------------|
| `fiber` | fiber (match) | express (default) | fastapi (default) |
| `echo` | echo (match) | express (default) | fastapi (default) |
| `gin` | gin (match) | express (default) | fastapi (default) |
| `chi` | chi (match) | express (default) | fastapi (default) |
| `net/http` | net/http (match) | express (default) | fastapi (default) |
| not set | fiber (default) | express (default) | fastapi (default) |

**Key principle:** `server.framework` is for runtime (`go-json serve`). Codegen framework is for code generation (`go-json codegen`). They are independent. No extra JSON property needed — codegen framework is a CLI concern.

### 12.3 Available Frameworks per Language

| Language | Frameworks | Notes |
|----------|------------|-------|
| Go | `fiber`, `echo`, `gin`, `chi`, `net/http` | Fiber default, matches bitcode engine |
| JavaScript | `express`, `hono`, `fastify`, `koa` | Express default, most ecosystem support |
| Python | `fastapi`, `flask`, `django` | FastAPI default, modern async |

### 12.4 Extensibility

Adding a new framework codegen adapter:

1. Create adapter file: `codegen/server_<lang>_<framework>.go`
2. Implement `ServerCodegenAdapter` interface
3. Register in framework registry with language + name

No JSON schema changes needed. No existing code changes needed.

### 12.5 Generated Output

`go-json codegen api.json --target go --framework fiber --output ./generated/`

Generates:
- `main.go` — server setup, route registration, listen
- `handlers.go` — handler functions (translated from go-json steps)
- `middleware.go` — custom middleware functions
- `types.go` — request/response structs

### 12.6 Framework-Specific Output Examples

| Target | Framework | Generated Code Style |
|--------|-----------|---------------------|
| Go | fiber | `app.Get("/path", handler)` |
| Go | net/http | `mux.HandleFunc("GET /path", handler)` |
| Go | echo | `e.GET("/path", handler)` |
| Go | gin | `r.GET("/path", handler)` |
| Go | chi | `r.Get("/path", handler)` |
| JavaScript | express | `app.get('/path', handler)` |
| JavaScript | hono | `app.get('/path', handler)` |
| JavaScript | fastify | `fastify.get('/path', handler)` |
| Python | fastapi | `@app.get('/path')` |
| Python | flask | `@app.route('/path')` |

### 12.7 Codegen Limitations

- **Middleware chains** — built-in middleware generated as framework-specific setup; custom middleware translated to functions
- **Template rendering** — template loading code generated, template files NOT converted (`.html` files stay as-is)
- **Database connections** — connection setup generated with placeholder DSN
- **JWT** — framework-specific JWT middleware setup generated
- **Complex expressions** — some go-json expressions may not translate cleanly; emitted as comments with original expression
- **I/O module calls** — translated to framework-native equivalents where possible (e.g., `sql.query` → `db.Query`)

---

## §13. CLI Command

### 13.1 `go-json serve`

```bash
# Basic serve
go-json serve api.json

# With options
go-json serve api.json --port 8080 --host 127.0.0.1

# Dev mode
go-json serve api.json --dev

# With I/O modules
go-json serve api.json --io sql,redis

# With framework override (if multiple compiled in)
go-json serve api.json --framework fiber
```

### 13.2 CLI Flags

| Flag | Default | Description |
|------|---------|-------------|
| `--port` | from config or 3000 | Override listen port |
| `--host` | from config or 0.0.0.0 | Override listen host |
| `--dev` | false | Enable dev mode |
| `--io` | from config | Enable I/O modules |
| `--framework` | from config or fiber | Override runtime framework |
| `--env-file` | `.env` | Load environment from file |

---

## §14. Security Considerations

### 14.1 Request Validation

- `max_body_size` enforced before body parsing
- Path parameters sanitized (no null bytes, no path traversal)
- Query parameters limited in count and size
- Headers limited in count and total size

### 14.2 Response Security

- `X-Content-Type-Options: nosniff` (always)
- `X-Frame-Options: DENY` (with `secure` middleware)
- `Strict-Transport-Security` (with `secure` middleware, HTTPS only)
- `Content-Security-Policy` (configurable)
- Template auto-escape prevents XSS

### 14.3 JWT Security

- Secret from environment variable (never in JSON config)
- Algorithm explicitly configured (no `alg: none` attack)
- Token expiry enforced
- Refresh token rotation (optional)

### 14.4 Rate Limiting

- Per-IP by default
- Per-user (JWT sub) when authenticated
- Configurable window and limit
- Returns 429 Too Many Requests

---

## §15. Server Mode Detection

### 15.1 How `go-json serve` Detects Server Programs

A program is a server program if it has a `"routes"` top-level key. The `"server"` key is optional (defaults apply).

```json
// Minimal server program — routes required, server config optional
{
  "routes": [
    {"method": "GET", "path": "/", "handler": "index"}
  ],
  "functions": {
    "index": {
      "steps": [{"return": {"status": 200, "body": {"message": "hello"}}}]
    }
  }
}
```

If `go-json serve` is called on a program without `"routes"`:
→ Error: `"program has no routes — not a server program"`

If `go-json run` is called on a program with `"routes"`:
→ Routes are ignored, `"steps"` executed normally (backward compatible)

### 15.2 Validation at Startup

Before starting the server, validate:
1. All `handler` references point to existing functions
2. All `render` template files exist on disk
3. All `middleware` references are either built-in names or existing functions
4. No duplicate route paths for same method
5. JWT config has `secret_env` if any route uses `jwt` middleware
6. Static directory exists if configured

Validation errors are reported at startup, not at request time.

---

## §16. Error Handling

### 16.1 Handler Panics

If a handler function panics during execution:
- `recover` middleware catches it (if enabled)
- Returns 500 with `{"error": "internal server error"}` (production) or full stack trace (dev mode)
- Panic is logged with request context

### 16.2 Handler Returns Invalid Response

If handler returns a value that doesn't match response convention:
- No `status`, `body`, `redirect`, or `data` → 204 No Content
- `render` route but no `data` in return → 500 error: "handler must return data for template rendering"
- `status` out of range (< 100 or > 599) → 500 error

### 16.3 Handler Timeout

Each handler execution respects the runtime timeout (default 30s). If exceeded:
- Execution cancelled via context
- Returns 504 Gateway Timeout
- Logged as timeout error

### 16.4 Error Response Format

All error responses follow consistent format:

```json
{
  "error": {
    "code": "NOT_FOUND",
    "message": "User not found",
    "request_id": "req_abc123"
  }
}
```

Status code mapping for structured errors:

| Error Code Pattern | HTTP Status |
|---|---|
| `NOT_FOUND` | 404 |
| `UNAUTHORIZED` | 401 |
| `FORBIDDEN` | 403 |
| `VALIDATION_*` | 400 |
| `CONFLICT` | 409 |
| `RATE_LIMITED` | 429 |
| anything else | 500 |

### 16.5 Custom Error Pages (HTML)

For template-rendered routes, errors can render error templates:

```json
{
  "server": {
    "error_templates": {
      "404": "errors/404.html",
      "500": "errors/500.html"
    }
  }
}
```

---

## §17. Middleware Data Passing

### 17.1 The Problem

Middleware needs to pass data to handlers (e.g., JWT middleware sets `request.user`). But `request` is injected as input to `Execute()` — middleware and handler run as separate executions.

### 17.2 Solution: request.store

Middleware can set values on `request.store`, which is a mutable map that persists through the middleware chain and into the handler:

```json
{
  "functions": {
    "addTiming": {
      "steps": [
        {"set": "request.store.start_time", "expr": "now()"}
      ]
    }
  }
}
```

Handler can read:
```json
{"let": "start", "expr": "request.store.start_time"}
```

Built-in middleware (JWT) sets `request.user` directly — this is a special case handled by the server runtime, not by go-json execution.

### 17.3 Middleware Execution Model

```
Request arrives
  → Build request object
  → For each middleware (global → group → route):
      → If built-in (Go-implemented): execute directly, modify request object
      → If custom (go-json function): Execute() with request, check for early return
      → If middleware returns response → short-circuit, send response
  → Execute handler function with final request object
  → Convert return value to HTTP response
  → Send response
```

Custom middleware that returns nothing (no `return` step) = pass-through.
Custom middleware that returns with `status`/`body`/`redirect` = short-circuit.

---

## §18. Full Example: Blog API + SSR

```json
{
  "name": "blog",
  "server": {
    "port": 3000,
    "framework": "fiber",
    "static": "./public",
    "templates": "./templates",
    "jwt": {"secret_env": "JWT_SECRET", "algorithm": "HS256", "expiry": "7d"},
    "cors": {"origins": ["http://localhost:5173"]}
  },
  "import": {
    "db": "io:sql"
  },
  "middleware": ["logger", "recover", "cors", "secure"],
  "routes": [
    {"method": "GET",  "path": "/",              "handler": "homePage",     "render": "pages/home.html"},
    {"method": "GET",  "path": "/posts/:slug",   "handler": "postPage",    "render": "pages/post.html"},

    {"prefix": "/api", "routes": [
      {"method": "GET",    "path": "/posts",       "handler": "listPosts"},
      {"method": "GET",    "path": "/posts/:slug", "handler": "getPost"},
      {"method": "POST",   "path": "/posts",       "handler": "createPost",  "middleware": ["jwt"]},
      {"method": "PUT",    "path": "/posts/:id",   "handler": "updatePost",  "middleware": ["jwt"]},
      {"method": "DELETE", "path": "/posts/:id",   "handler": "deletePost",  "middleware": ["jwt", "requireAdmin"]}
    ]},

    {"method": "POST", "path": "/auth/login",    "handler": "login"},
    {"method": "POST", "path": "/auth/register", "handler": "register"}
  ],
  "functions": {
    "homePage": {
      "steps": [
        {"let": "posts", "call": "db.query", "with": {
          "query": "SELECT id, title, slug, excerpt, created_at FROM posts ORDER BY created_at DESC LIMIT 10"
        }},
        {"return": {"data": {"posts": "posts.rows", "title": "'Home'"}}}
      ]
    },
    "postPage": {
      "steps": [
        {"let": "post", "call": "db.query", "with": {
          "query": "SELECT * FROM posts WHERE slug = ?",
          "args": ["request.params.slug"]
        }},
        {"if": "len(post.rows) == 0", "then": [
          {"return": {"status": 404, "data": {"title": "'Not Found'", "message": "'Post not found'"}}}
        ]},
        {"return": {"data": {"post": "post.rows[0]", "title": "post.rows[0].title"}}}
      ]
    },
    "listPosts": {
      "steps": [
        {"let": "page", "expr": "int(request.query.page ?? '1')"},
        {"let": "posts", "call": "db.query", "with": {
          "query": "SELECT id, title, slug, excerpt, created_at FROM posts ORDER BY created_at DESC LIMIT 20 OFFSET ?",
          "args": ["(page - 1) * 20"]
        }},
        {"let": "total", "call": "db.query", "with": {"query": "SELECT COUNT(*) as count FROM posts"}},
        {"return": {"status": 200, "body": {"posts": "posts.rows", "total": "total.rows[0].count", "page": "page"}}}
      ]
    },
    "createPost": {
      "steps": [
        {"let": "body", "expr": "request.body"},
        {"if": "body.title == nil || body.content == nil", "then": [
          {"return": {"status": 400, "body": {"error": "title and content required"}}}
        ]},
        {"let": "slug", "expr": "lower(replace(body.title, ' ', '-'))"},
        {"let": "result", "call": "db.execute", "with": {
          "query": "INSERT INTO posts (title, slug, content, author_id) VALUES (?, ?, ?, ?)",
          "args": ["body.title", "slug", "body.content", "request.user.sub"]
        }},
        {"return": {"status": 201, "body": {"id": "result.last_insert_id", "slug": "slug"}}}
      ]
    },
    "requireAdmin": {
      "steps": [
        {"if": "request.user.role != 'admin'", "then": [
          {"return": {"status": 403, "body": {"error": "admin access required"}}}
        ]}
      ]
    }
  }
}
```

---

## §19. Implementation Scope

### Package Structure

```
packages/go-json/
├── server/              NEW — web server runtime
│   ├── server.go        Server orchestrator (parse config, setup routes, start)
│   ├── router.go        Route parsing and registration
│   ├── handler.go       Request→Execute→Response bridge
│   ├── middleware.go    Built-in middleware implementations
│   ├── jwt.go           JWT sign/verify/decode/refresh
│   ├── template.go      Template engine wrapper
│   ├── static.go        Static file serving
│   ├── config.go        Server config parsing
│   └── adapters/        Framework adapters
│       ├── adapter.go   ServerAdapter interface
│       ├── fiber.go     Fiber adapter (default)
│       ├── nethttp.go   net/http adapter
│       ├── echo.go      Echo adapter
│       ├── gin.go       Gin adapter
│       └── chi.go       Chi adapter
├── cmd/go-json/
│   └── main.go          Add "serve" command
├── codegen/
│   ├── server.go            Server codegen interface + registry
│   ├── server_go_fiber.go   Go + Fiber codegen (default)
│   ├── server_go_nethttp.go Go + net/http codegen
│   ├── server_go_echo.go    Go + Echo codegen
│   ├── server_go_gin.go     Go + Gin codegen
│   ├── server_go_chi.go     Go + Chi codegen
│   ├── server_js_express.go JS + Express codegen (default)
│   ├── server_js_hono.go    JS + Hono codegen
│   ├── server_py_fastapi.go Python + FastAPI codegen (default)
│   └── server_py_flask.go   Python + Flask codegen
```

### Dependencies (New)

| Package | Purpose | Required? |
|---------|---------|-----------|
| `github.com/golang-jwt/jwt/v5` | JWT sign/verify | Yes |
| `github.com/gofiber/fiber/v2` | Fiber adapter | Optional (build tag) |
| `github.com/labstack/echo/v4` | Echo adapter | Optional (build tag) |
| `github.com/gin-gonic/gin` | Gin adapter | Optional (build tag) |
| `github.com/go-chi/chi/v5` | Chi adapter | Optional (build tag) |
| `github.com/fsnotify/fsnotify` | Dev mode file watching | Optional (dev only) |

Default build includes Fiber adapter + net/http adapter + JWT. Other frameworks via build tags:

```bash
go build -tags "fiber,echo" ./cmd/go-json/
```
