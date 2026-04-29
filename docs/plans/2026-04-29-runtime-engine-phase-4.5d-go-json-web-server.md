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
    "auth": {
      "default": "jwt",
      "strategies": {
        "jwt": {
          "type": "bearer",
          "secret_env": "JWT_SECRET",
          "algorithm": "HS256",
          "expiry": "24h",
          "cookie": "token",
          "header": "Authorization",
          "prefix": "Bearer"
        }
      }
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
| `multipart/form-data` | `map[string]any` (see §4.3 File Uploads) |
| other / missing | raw `string` |

### 4.3 File Uploads

For `multipart/form-data` requests, `request.body` contains both field values and file metadata:

```json
{
  "request": {
    "body": {
      "name": "Alice",
      "avatar": {
        "_file": true,
        "filename": "photo.jpg",
        "size": 245760,
        "content_type": "image/jpeg",
        "temp_path": "/tmp/go-json-upload-abc123.jpg"
      }
    }
  }
}
```

File handling:
- Files saved to temp directory automatically before handler executes
- `temp_path` is the path to the temp file (readable via `fs.read` if FS module enabled)
- `max_body_size` in server config limits total upload size
- Temp files cleaned up after handler completes
- Multiple files: each file field gets its own `_file` object
- File arrays (multiple files same field): `"photos": [{"_file": true, ...}, {"_file": true, ...}]`

Handler example:
```json
{
  "steps": [
    {"let": "file", "expr": "request.body.avatar"},
    {"if": "file._file != true", "then": [
      {"return": {"status": 400, "body": {"error": "avatar file required"}}}
    ]},
    {"if": "file.size > 5242880", "then": [
      {"return": {"status": 400, "body": {"error": "file too large (max 5MB)"}}}
    ]},
    {"let": "content", "call": "fs.read", "with": {"path": "file.temp_path", "encoding": "'base64'"}},
    {"let": "result", "call": "storage.upload", "with": {"filename": "file.filename", "content": "content"}},
    {"return": {"status": 200, "body": {"url": "result.url"}}}
  ]
}
```

### 4.4 Request Validation

go-json does not include a built-in JSON schema validator. Validation is done in handler steps using conditional logic:

```json
{
  "steps": [
    {"let": "body", "expr": "request.body"},
    {"if": "body.email == nil || !matches(body.email, '^[^@]+@[^@]+$')", "then": [
      {"return": {"status": 400, "body": {"error": "valid email required"}}}
    ]},
    {"if": "body.name == nil || len(body.name) < 2", "then": [
      {"return": {"status": 400, "body": {"error": "name must be at least 2 characters"}}}
    ]}
  ]
}
```

For reusable validation, define a validation function:

```json
{
  "functions": {
    "validateUser": {
      "params": {"body": "map"},
      "steps": [
        {"if": "body.email == nil", "then": [
          {"return": {"status": 400, "body": {"error": "email required"}}}
        ]},
        {"if": "body.name == nil", "then": [
          {"return": {"status": 400, "body": {"error": "name required"}}}
        ]}
      ]
    },
    "createUser": {
      "steps": [
        {"let": "body", "expr": "request.body"},
        {"let": "err", "call": "validateUser", "with": {"body": "body"}},
        {"if": "err != nil", "then": [{"return": "err"}]}
      ]
    }
  }
}
```

A dedicated `validate` middleware or JSON schema support may be added in a future phase.

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

→ Returns 404 because `NOT_FOUND` maps to 404 (see §18.4 for full mapping).

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
| `auth` | Authenticate using default strategy, inject `request.user` | `server.auth` |
| `auth:<strategy>` | Authenticate using specific strategy | `server.auth.strategies.<name>` |
| `rate_limit` | Rate limiting per IP/user | `server.rate_limit` |
| `request_id` | Add `X-Request-ID` header | None |
| `compress` | Gzip/Brotli response compression | None |
| `secure` | Security headers (HSTS, X-Frame-Options, etc.) | None |

Note: `"jwt"` is kept as alias for `"auth:jwt"` for backward compatibility.

### 6.2 Plugable Auth System

Auth is configured via `server.auth` with named strategies. Each strategy has a `type` that determines the authentication mechanism.

```json
{
  "server": {
    "auth": {
      "default": "jwt",
      "strategies": {
        "jwt": {
          "type": "bearer",
          "secret_env": "JWT_SECRET",
          "algorithm": "HS256",
          "expiry": "24h",
          "cookie": "token",
          "header": "Authorization",
          "prefix": "Bearer",
          "claims": {"issuer": "my-app"}
        },
        "apikey": {
          "type": "api_key",
          "header": "X-API-Key",
          "query_param": "api_key",
          "keys_env": "API_KEYS"
        },
        "basic": {
          "type": "basic",
          "users_env": "BASIC_AUTH_USERS",
          "realm": "My App"
        }
      }
    }
  }
}
```

### 6.3 Auth Strategy Types

| Type | Description | Token Source | `request.user` |
|------|-------------|-------------|-----------------|
| `bearer` | JWT Bearer token | `Authorization: Bearer <token>` or cookie | Decoded JWT payload |
| `api_key` | API key validation | Header or query param | `{"key": "the-key", "name": "key-name"}` |
| `basic` | HTTP Basic Auth | `Authorization: Basic <base64>` | `{"username": "user"}` |
| `oauth2` | OAuth2 token introspection (future) | `Authorization: Bearer <token>` | Provider-specific payload |
| `custom` | Custom go-json function | Defined by function | Whatever function returns |

### 6.4 Auth Strategy Detail: Bearer (JWT)

```json
{
  "jwt": {
    "type": "bearer",
    "secret_env": "JWT_SECRET",
    "algorithm": "HS256",
    "expiry": "24h",
    "cookie": "token",
    "header": "Authorization",
    "prefix": "Bearer",
    "claims": {"issuer": "my-app", "audience": "api"}
  }
}
```

Flow:
1. Extract token from `Authorization: Bearer <token>` header OR cookie (configurable)
2. Validate signature using secret from env var
3. Check expiry and claims (issuer, audience)
4. If valid: decode payload → inject into `request.user`
5. If invalid/missing: return 401

After auth, handler can access:
```json
{"let": "user_id", "expr": "request.user.sub"},
{"let": "email", "expr": "request.user.email"},
{"let": "roles", "expr": "request.user.roles"}
```

### 6.5 Auth Strategy Detail: API Key

```json
{
  "apikey": {
    "type": "api_key",
    "header": "X-API-Key",
    "query_param": "api_key",
    "keys_env": "API_KEYS"
  }
}
```

`API_KEYS` env var format: `key1:name1,key2:name2` (comma-separated key:name pairs).

Flow:
1. Extract key from header `X-API-Key` or query param `?api_key=...`
2. Validate against keys from env var
3. If valid: inject `{"key": "...", "name": "..."}` into `request.user`
4. If invalid/missing: return 401

### 6.6 Auth Strategy Detail: Basic

```json
{
  "basic": {
    "type": "basic",
    "users_env": "BASIC_AUTH_USERS",
    "realm": "My App"
  }
}
```

`BASIC_AUTH_USERS` env var format: `user1:pass1,user2:pass2`.

Flow:
1. Extract credentials from `Authorization: Basic <base64>`
2. Validate against users from env var
3. If valid: inject `{"username": "..."}` into `request.user`
4. If invalid/missing: return 401 with `WWW-Authenticate: Basic realm="My App"`

### 6.7 Auth Strategy Detail: Custom

For auth mechanisms not covered by built-in types, use a go-json function:

```json
{
  "server": {
    "auth": {
      "strategies": {
        "custom_token": {
          "type": "custom",
          "handler": "validateCustomToken"
        }
      }
    }
  },
  "functions": {
    "validateCustomToken": {
      "steps": [
        {"let": "token", "expr": "request.headers['X-Custom-Token']"},
        {"if": "token == nil", "then": [
          {"return": {"status": 401, "body": {"error": "missing token"}}}
        ]},
        {"let": "user", "call": "sql.query", "with": {
          "query": "SELECT * FROM tokens WHERE token = ? AND expires_at > NOW()",
          "args": ["token"]
        }},
        {"if": "len(user.rows) == 0", "then": [
          {"return": {"status": 401, "body": {"error": "invalid token"}}}
        ]},
        {"return": "user.rows[0]"}
      ]
    }
  }
}
```

Custom auth function:
- Receives `request` in scope
- Returns user object → injected into `request.user`
- Returns response with `status` → short-circuit (auth failed)

### 6.8 Route-Level Auth

```json
{
  "routes": [
    {"method": "GET",  "path": "/api/public",   "handler": "publicData"},
    {"method": "GET",  "path": "/api/profile",  "handler": "getProfile",  "middleware": ["auth"]},
    {"method": "POST", "path": "/api/admin",    "handler": "adminAction", "middleware": ["auth", "requireAdmin"]},
    {"method": "POST", "path": "/webhook",      "handler": "webhook",     "middleware": ["auth:apikey"]},
    {"method": "GET",  "path": "/internal",     "handler": "internal",    "middleware": ["auth:basic"]}
  ]
}
```

- `"auth"` → uses `server.auth.default` strategy (e.g., `"jwt"`)
- `"auth:apikey"` → uses specific `"apikey"` strategy
- `"auth:basic"` → uses specific `"basic"` strategy
- `"jwt"` → alias for `"auth:jwt"` (backward compatible)

### 6.9 JWT Token Generation

For login endpoints, handlers can generate tokens:

```json
{
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

## §15. OpenAPI / Swagger

### 15.1 Auto-Generated OpenAPI Spec

`go-json serve` can auto-generate an OpenAPI 3.0 spec from route definitions:

```bash
# Serve with Swagger UI at /docs
go-json serve api.json --docs

# Export OpenAPI spec to file
go-json openapi api.json --output openapi.json
```

When `--docs` is enabled, two endpoints are added:
- `GET /docs` — Swagger UI (embedded)
- `GET /docs/openapi.json` — raw OpenAPI spec

### 15.2 What's Auto-Generated

From the program JSON, the following OpenAPI fields are inferred automatically:

| OpenAPI Field | Source |
|---|---|
| `info.title` | `name` from program |
| `info.version` | `"1.0.0"` default or `server.version` |
| `paths` | `routes[].method` + `routes[].path` |
| `paths.*.parameters` (path) | `:id` in path → `{id}` path parameter |
| `paths.*.security` | Route has `auth`/`jwt` middleware → bearer/apiKey scheme |
| `tags` | Route group prefix → tag name |
| `components.securitySchemes` | From `server.auth.strategies` |

### 15.3 What's NOT Auto-Generated (Needs `api` Annotation)

Request body schemas, response schemas, query parameters, and descriptions cannot be inferred from go-json code (dynamically typed). Use optional `api` annotation on routes:

```json
{
  "routes": [
    {
      "method": "POST",
      "path": "/api/users",
      "handler": "createUser",
      "middleware": ["auth"],
      "api": {
        "summary": "Create a new user",
        "description": "Creates a user account and returns the created user object",
        "tags": ["Users"],
        "body": {
          "required": true,
          "content": {
            "name": {"type": "string", "required": true, "description": "User's full name"},
            "email": {"type": "string", "required": true, "format": "email"},
            "role": {"type": "string", "enum": ["user", "admin"], "default": "user"}
          }
        },
        "query": {
          "notify": {"type": "boolean", "description": "Send welcome email", "default": true}
        },
        "responses": {
          "201": {
            "description": "User created",
            "content": {
              "id": {"type": "integer"},
              "name": {"type": "string"},
              "email": {"type": "string"}
            }
          },
          "400": {"description": "Validation error"},
          "401": {"description": "Unauthorized"}
        }
      }
    }
  ]
}
```

### 15.4 `api` Annotation Fields

| Field | Type | Description |
|---|---|---|
| `summary` | string | Short description (shown in Swagger UI route list) |
| `description` | string | Detailed description (shown when expanded) |
| `tags` | []string | Grouping tags |
| `deprecated` | bool | Mark route as deprecated |
| `body` | object | Request body schema |
| `body.required` | bool | Whether body is required |
| `body.content` | map | Field definitions: `{name: {type, required, format, enum, default, description}}` |
| `query` | map | Query parameter definitions (same field format) |
| `responses` | map | Response definitions keyed by status code |
| `responses.*.description` | string | Response description |
| `responses.*.content` | map | Response body field definitions |

### 15.5 Security Schemes in OpenAPI

Auth strategies from `server.auth.strategies` are auto-mapped to OpenAPI security schemes:

| Auth Type | OpenAPI Security Scheme |
|---|---|
| `bearer` (JWT) | `bearerAuth` — `type: http, scheme: bearer, bearerFormat: JWT` |
| `api_key` | `apiKeyAuth` — `type: apiKey, in: header, name: X-API-Key` |
| `basic` | `basicAuth` — `type: http, scheme: basic` |

### 15.6 Minimal vs Full Annotation

Routes without `api` annotation still appear in the spec — just with minimal info (method, path, security). This means:

- **Zero-config:** `go-json serve api.json --docs` works immediately with basic spec
- **Progressive enhancement:** Add `api` annotations to routes that need detailed docs
- **No all-or-nothing:** Some routes documented, some not — that's fine

---

## §16. Unified SQL Query Parameters

### 16.1 The Problem

Different SQL drivers use different placeholder syntax:

| Driver | Positional | Named |
|--------|-----------|-------|
| SQLite | `?` | not supported |
| MySQL | `?` | not supported |
| PostgreSQL | `$1`, `$2` | not supported natively |
| SQL Server | `@p1`, `@p2` | `@name` |
| Oracle | `:1`, `:2` | `:name` |

This means switching databases requires rewriting all queries — bad for portability.

### 16.2 Solution: Universal Placeholders

go-json uses `?` as universal positional placeholder and `:name` for named parameters. The SQL module auto-translates to driver-specific syntax before execution.

**Positional (user always writes `?`):**

```json
{"call": "sql.query", "with": {
  "query": "SELECT * FROM users WHERE id = ? AND status = ?",
  "args": [42, "active"]
}}
```

**Named (user always writes `:name`):**

```json
{"call": "sql.query", "with": {
  "query": "SELECT * FROM users WHERE id = :id AND status = :status",
  "args": {"id": 42, "status": "active"}
}}
```

### 16.3 Translation Table

| Driver | `?` becomes | `:name` becomes | Args format |
|--------|------------|----------------|-------------|
| SQLite | `?` (no change) | `?` + ordered args | `[]any` |
| MySQL | `?` (no change) | `?` + ordered args | `[]any` |
| PostgreSQL | `$1`, `$2`, ... | `$1`, `$2`, ... + ordered args | `[]any` |
| SQL Server | `@p1`, `@p2`, ... | `@name` | named |
| Oracle | `:1`, `:2`, ... | `:name` | named |

User never needs to know which driver is active. Switching from SQLite to PostgreSQL requires zero query changes.

### 16.4 Edge Cases

| Case | Behavior |
|---|---|
| `??` in query | Literal `?` (escape). For PostgreSQL JSON operator `data ? 'key'` |
| `?` inside single quotes | NOT treated as placeholder. `'what?'` stays as-is |
| Mixed `?` and `:name` in same query | Error: "cannot mix positional and named parameters" |
| `:name` not found in args map | Error: "named parameter ':name' not found in args" |
| More `?` than args | Error: "query has N placeholders but only M args provided" |
| Fewer `?` than args | Error: "query has N placeholders but M args provided" |

---

## §17. Server Mode Detection

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

### 15.2 Built-in Health Check

Every server automatically registers `GET /health` that returns:

```json
{"status": "ok", "name": "my-api", "uptime": 3600}
```

This endpoint:
- Bypasses all middleware (no auth, no rate limit)
- Always returns 200 if server is running
- Useful for load balancer health checks, Kubernetes probes
- Can be disabled: `"server": {"health": false}`
- Can be customized: `"server": {"health": "/custom-health"}` changes the path

### 15.3 Validation at Startup

Before starting the server, validate:
1. All `handler` references point to existing functions
2. All `render` template files exist on disk
3. All `middleware` references are either built-in names or existing functions
4. No duplicate route paths for same method
5. JWT config has `secret_env` if any route uses `jwt` middleware
6. Static directory exists if configured

Validation errors are reported at startup, not at request time.

---

## §18. Error Handling

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

## §19. Middleware Data Passing

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

## §20. Full Example: Blog API + SSR

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

## §21. FS Module Enhancement + File Upload Integration

### 21.1 Missing FS Operations

The Phase 4.5c FS module provides basic CRUD (`read`, `write`, `append`, `exists`, `list`, `mkdir`, `remove`). Web server use cases (file uploads, exports, backups) require additional operations:

| Function | Input | Returns | Description |
|---|---|---|---|
| `fs.stat(path)` | path | `{name, size, is_dir, is_file, ext, modified, permissions}` | File/directory metadata |
| `fs.copy(src, dst)` | source, destination | `void` | Copy file or directory |
| `fs.move(src, dst)` | source, destination | `void` | Move/rename file or directory |
| `fs.glob(pattern)` | glob pattern | `[]string` | Find files matching pattern |
| `fs.list(path, detail?)` | path, optional detail flag | `[]string` or `[]FileInfo` | Enhanced: return metadata when `detail: true` |

### 21.2 Path Utility Functions (Stdlib)

These are pure string operations (no disk I/O), so they belong in stdlib, not the FS module:

| Function | Input | Returns | Example |
|---|---|---|---|
| `basename(path)` | path string | filename | `basename("/a/b/c.txt")` → `"c.txt"` |
| `dirname(path)` | path string | directory | `dirname("/a/b/c.txt")` → `"/a/b"` |
| `extname(path)` | path string | extension | `extname("photo.jpg")` → `".jpg"` |
| `joinpath(parts...)` | path segments | joined path | `joinpath("/a", "b", "c.txt")` → `"/a/b/c.txt"` |

### 21.3 `fs.stat` Return Shape

```json
{"let": "info", "call": "fs.stat", "with": {"path": "'data.json'"}}
// info = {
//   "name": "data.json",
//   "size": 1024,
//   "is_dir": false,
//   "is_file": true,
//   "ext": ".json",
//   "modified": "2026-04-29T10:30:00Z",
//   "permissions": "0644"
// }
```

### 21.4 `fs.list` Enhanced Mode

```json
// Simple mode (backward compatible)
{"let": "names", "call": "fs.list", "with": {"path": "'./data'"}}
// names = ["a.txt", "b.txt", "subdir"]

// Detailed mode
{"let": "entries", "call": "fs.list", "with": {"path": "'./data'", "detail": true}}
// entries = [
//   {"name": "a.txt", "size": 512, "is_dir": false, "ext": ".txt", "modified": "2026-04-29T10:00:00Z"},
//   {"name": "subdir", "size": 0, "is_dir": true, "modified": "2026-04-28T09:00:00Z"}
// ]
```

### 21.5 File Upload → FS Integration Flow

File uploads (§4.3) produce temp files. FS operations are the primary way handlers process uploaded files:

```
HTTP multipart request
  → Server saves to temp dir → {_file: true, temp_path: "/tmp/xxx"}
    → Handler uses FS operations:
        fs.stat(file.temp_path)           ← check size, validate
        extname(file.filename)            ← check extension
        fs.copy(file.temp_path, dest)     ← save to permanent location
        fs.move(file.temp_path, dest)     ← or move (more efficient)
    → Server auto-deletes temp file after handler completes
```

**Complete file upload example:**

```json
{
  "functions": {
    "uploadDocument": {
      "steps": [
        {"let": "file", "expr": "request.body.document"},
        {"if": "file._file != true", "then": [
          {"return": {"status": 400, "body": {"error": "document file required"}}}
        ]},

        {"_c": "Validate extension"},
        {"let": "ext", "expr": "extname(file.filename)"},
        {"if": "ext != '.pdf' && ext != '.docx' && ext != '.xlsx'", "then": [
          {"return": {"status": 400, "body": {"error": "only pdf/docx/xlsx allowed"}}}
        ]},

        {"_c": "Validate size (10MB max)"},
        {"if": "file.size > 10485760", "then": [
          {"return": {"status": 400, "body": {"error": "file too large (max 10MB)"}}}
        ]},

        {"_c": "Generate unique filename and save"},
        {"let": "unique_name", "expr": "crypto.uuid() + ext"},
        {"let": "dest", "expr": "joinpath('./uploads/documents', unique_name)"},
        {"call": "fs.mkdir", "with": {"path": "'./uploads/documents'"}},
        {"call": "fs.copy", "with": {"src": "file.temp_path", "dst": "dest"}},

        {"_c": "Save record to database"},
        {"let": "result", "call": "sql.execute", "with": {
          "query": "INSERT INTO documents (filename, original_name, size, path, user_id) VALUES (?, ?, ?, ?, ?)",
          "args": ["unique_name", "file.filename", "file.size", "dest", "request.user.sub"]
        }},

        {"return": {"status": 201, "body": {"id": "result.last_insert_id", "filename": "unique_name"}}}
      ]
    }
  }
}
```

### 21.6 Codegen Translation

All FS operations and path utilities have direct equivalents in target languages:

| go-json | Go | JavaScript (Node.js) | Python |
|---|---|---|---|
| `fs.stat(path)` | `os.Stat(path)` | `fs.statSync(path)` | `Path(path).stat()` |
| `fs.copy(src, dst)` | `io.Copy(dst, src)` | `fs.copyFileSync(src, dst)` | `shutil.copy2(src, dst)` |
| `fs.move(src, dst)` | `os.Rename(src, dst)` | `fs.renameSync(src, dst)` | `Path(src).rename(dst)` |
| `fs.glob(pattern)` | `filepath.Glob(pattern)` | `glob.sync(pattern)` | `list(Path('.').glob(pattern))` |
| `basename(path)` | `filepath.Base(path)` | `path.basename(path)` | `Path(path).name` |
| `dirname(path)` | `filepath.Dir(path)` | `path.dirname(path)` | `str(Path(path).parent)` |
| `extname(path)` | `filepath.Ext(path)` | `path.extname(path)` | `Path(path).suffix` |
| `joinpath(a, b)` | `filepath.Join(a, b)` | `path.join(a, b)` | `str(Path(a) / b)` |

---

## §22. Stdlib Additions

### 22.1 `toJSON` / `fromJSON`

Serialize and deserialize JSON strings — needed for web server response building, data export, and API integrations.

```json
{"let": "json_str", "expr": "toJSON(data)"}
{"let": "parsed", "expr": "fromJSON(json_str)"}
```

| Function | Input | Returns | Description |
|---|---|---|---|
| `toJSON(value)` | any | string | Serialize value to JSON string |
| `fromJSON(str)` | string | any | Parse JSON string to value |

Codegen translation:

| go-json | Go | JavaScript | Python |
|---|---|---|---|
| `toJSON(x)` | `json.Marshal(x)` | `JSON.stringify(x)` | `json.dumps(x)` |
| `fromJSON(s)` | `json.Unmarshal(s, &v)` | `JSON.parse(s)` | `json.loads(s)` |

### 22.2 `formatDate` Universal Format

`formatDate` currently uses Go's reference time layout (`"2006-01-02"`). This is Go-specific and not translatable to other languages.

**Solution:** Support both Go layout AND universal format strings. Auto-detect based on content:

```json
// Go format (existing — backward compatible)
{"let": "d", "expr": "formatDate(now(), '2006-01-02 15:04:05')"}

// Universal format (new)
{"let": "d", "expr": "formatDate(now(), 'YYYY-MM-DD HH:mm:ss')"}
```

Detection: if layout contains `2006` → Go format. If contains `YYYY` → universal format.

Universal format tokens:

| Token | Meaning | Example |
|---|---|---|
| `YYYY` | 4-digit year | `2026` |
| `YY` | 2-digit year | `26` |
| `MM` | Month (zero-padded) | `04` |
| `DD` | Day (zero-padded) | `29` |
| `HH` | Hour 24h (zero-padded) | `14` |
| `hh` | Hour 12h (zero-padded) | `02` |
| `mm` | Minute (zero-padded) | `05` |
| `ss` | Second (zero-padded) | `09` |
| `A` | AM/PM | `PM` |
| `Z` | Timezone offset | `+07:00` |

Codegen translation:

| go-json Universal | Go Layout | JS (dayjs) | Python (strftime) |
|---|---|---|---|
| `YYYY` | `2006` | `YYYY` | `%Y` |
| `MM` | `01` | `MM` | `%m` |
| `DD` | `02` | `DD` | `%d` |
| `HH` | `15` | `HH` | `%H` |
| `mm` | `04` | `mm` | `%M` |
| `ss` | `05` | `ss` | `%S` |

---

## §23. Codegen Dependency Management

### 23.1 The Problem

Generated code requires dependencies (HTTP framework, JWT library, database driver, etc.). Without dependency files, generated code cannot run.

### 23.2 Solution: Generate Complete Runnable Projects

Codegen generates dependency files alongside source code:

**Go:**
```
generated/
├── go.mod
├── main.go
├── handlers.go
├── middleware.go
└── .env.example
```

**JavaScript:**
```
generated/
├── package.json
├── index.js
├── routes.js
├── middleware.js
└── .env.example
```

**Python:**
```
generated/
├── requirements.txt
├── main.py
├── routes.py
├── middleware.py
└── .env.example
```

### 23.3 Smart Dependency Detection

Codegen scans the program to detect which features are used, then includes only needed dependencies:

| Feature Detected | Go Dep | JS Dep | Python Dep |
|---|---|---|---|
| HTTP server (fiber) | `github.com/gofiber/fiber/v2` | — | — |
| HTTP server (express) | — | `express` | — |
| HTTP server (fastapi) | — | — | `fastapi`, `uvicorn` |
| JWT auth | `github.com/golang-jwt/jwt/v5` | `jsonwebtoken` | `PyJWT` |
| SQL (postgres) | `github.com/lib/pq` | `pg` | `psycopg2` |
| SQL (mysql) | `github.com/go-sql-driver/mysql` | `mysql2` | `mysql-connector-python` |
| SQL (sqlite) | `modernc.org/sqlite` | `better-sqlite3` | (stdlib) |
| MongoDB | `go.mongodb.org/mongo-driver/v2` | `mongodb` | `pymongo` |
| Redis | `github.com/redis/go-redis/v9` | `ioredis` | `redis` |
| CORS | (framework built-in) | `cors` | (framework built-in) |
| Template | (stdlib) | `ejs` | `jinja2` |
| File upload | (framework built-in) | `multer` | `python-multipart` |
| Env loading | `github.com/joho/godotenv` | `dotenv` | `python-dotenv` |

### 23.4 `.env.example` Generation

Auto-generated from program config:

```env
# Server
PORT=3000

# Database
DATABASE_URL=postgres://user:password@localhost:5432/dbname

# Auth
JWT_SECRET=change-me-to-a-random-secret

# Redis
REDIS_URL=redis://localhost:6379
```

Sources: `server.port`, `server.auth.*.secret_env`, SQL `DefaultDSN`, Redis `DefaultURI`.

---

## §24. CLI Code Generator (`go-json generate`)

### 24.1 Overview

`go-json generate` scaffolds go-json programs and codegen output from templates. Three generators:

| Command | Description |
|---|---|
| `go-json generate crud` | Generate CRUD API for a model |
| `go-json generate auth` | Generate authentication endpoints |
| `go-json generate project` | Generate full project scaffold |

### 24.2 `go-json generate crud`

**From manual field definition:**

```bash
go-json generate crud users --fields "name:string,email:string,role:string,age:int" --auth
```

**From database introspection:**

```bash
go-json generate crud --from-db --dsn "postgres://localhost/mydb" --table users --auth
go-json generate crud --from-db --dsn-env DATABASE_URL --table users,orders,products
go-json generate crud --from-db --dsn "postgres://..." --all
go-json generate crud --from-db --dsn "postgres://..." --interactive
```

**Options:**

| Flag | Default | Description |
|---|---|---|
| `--fields` | — | Manual field definition: `"name:type,name:type"` |
| `--from-db` | false | Introspect database for fields |
| `--dsn` | — | Database connection string |
| `--dsn-env` | — | Env var name containing DSN |
| `--table` | — | Table name(s), comma-separated |
| `--all` | false | All tables in database |
| `--interactive` | false | Interactive table picker |
| `--auth` | false | Add auth middleware to routes |
| `--pattern` | `simple` | Architecture pattern (see §25) |
| `--output` | `./` | Output directory |
| `--dry-run` | false | Preview without writing |

### 24.3 Database Introspection

When `--from-db` is used, the generator connects to the database and reads metadata:

| Metadata | Source | Used For |
|---|---|---|
| Column names | `information_schema.columns` | Field list |
| Column types | `data_type` | Type mapping, OpenAPI schema |
| Nullable | `is_nullable` | Required vs optional |
| Primary key | `table_constraints` | Route params, GET/PUT/DELETE by PK |
| Default values | `column_default` | Skip in create body, use `??` in handler |
| Foreign keys | `referential_constraints` | Relationship endpoints, FK validation |
| Unique constraints | `table_constraints` | Duplicate detection |
| Max length | `character_maximum_length` | Validation rules |
| Enum values | `pg_enum` / `SHOW COLUMNS` | OpenAPI enum, validation |
| Table/column comments | `pg_description` / `table_comment` | OpenAPI descriptions |

**Type mapping:**

| DB Type (Postgres) | DB Type (MySQL) | go-json Type | OpenAPI Type |
|---|---|---|---|
| `integer`, `bigint` | `int`, `bigint` | `int` | `integer` |
| `real`, `double precision` | `float`, `double` | `float` | `number` |
| `boolean` | `tinyint(1)` | `bool` | `boolean` |
| `varchar(n)`, `text` | `varchar(n)`, `text` | `string` | `string` |
| `timestamp`, `timestamptz` | `datetime` | `string` | `string` (format: date-time) |
| `date` | `date` | `string` | `string` (format: date) |
| `json`, `jsonb` | `json` | `any` | `object` |
| `uuid` | `char(36)` | `string` | `string` (format: uuid) |
| `numeric`, `decimal` | `decimal` | `float` | `number` |
| `enum` | `enum` | `string` | `string` (enum: [...]) |

**Smart field detection:**

| DB Metadata | Generated Behavior |
|---|---|
| `NOT NULL` without default | → Required field, validation in handler |
| `NOT NULL` with default | → Optional field, `body.field ?? default` |
| `SERIAL` / auto-increment | → Skip from create/update body |
| `DEFAULT NOW()` | → Skip from create body (auto-generated) |
| `VARCHAR(255)` | → Validation: `len(body.field) > 255` |
| `REFERENCES other(id)` | → FK validation: check referenced record exists |
| `UNIQUE` | → Duplicate check before insert |
| `ENUM('a','b','c')` | → Validation: `contains(['a','b','c'], body.field)` |

**Edge cases:**

| Case | Behavior |
|---|---|
| Table without primary key | Warning, generate list-only endpoint |
| Composite primary key | Route: `/api/items/:key1/:key2` |
| JSON/JSONB columns | Type `any`, no schema validation |
| Computed/generated columns | Skip from create/update (read-only) |
| Views | Read-only endpoints (GET only) |
| Junction table (2 FKs only) | Detect as many-to-many, generate relationship endpoints |

**Relationship detection:**

Foreign keys generate additional endpoints:
- `GET /api/users/:id/orders` — list related records
- `POST /api/users/:id/orders` — create with auto-set FK
- Junction tables → `POST /api/users/:id/roles` (add), `DELETE /api/users/:id/roles/:role_id` (remove)

### 24.4 `go-json generate auth`

```bash
go-json generate auth --strategy jwt --output auth.json
```

Generates auth endpoints:
- `POST /auth/register` — create user with hashed password
- `POST /auth/login` — validate credentials, return JWT
- `POST /auth/refresh` — refresh token
- `GET /auth/me` — get current user profile
- `POST /auth/change-password` — change password
- SQL migration for `users` table

### 24.5 `go-json generate project`

```bash
go-json generate project my-api --db postgres --auth jwt --framework fiber
```

Generates full project scaffold:

```
my-api/
├── api.json              ← main program (server config, routes, imports)
├── functions/
│   ├── auth.json         ← auth handlers
│   └── health.json       ← health check
├── templates/            ← empty, ready for SSR
├── public/               ← empty, ready for static files
├── migrations/
│   └── 001_initial.sql   ← users table
├── tests/
│   └── auth_test.json    ← test cases
├── .env.example
└── README.md
```

### 24.6 `--interactive` Mode

```
$ go-json generate crud --from-db --dsn "postgres://..." --interactive

Connecting to postgres://localhost/mydb...
Found 8 tables:

  [x] users (12 columns, 3 FKs)
  [x] orders (8 columns, 2 FKs)
  [ ] migrations (3 columns)
  [x] products (9 columns, 1 FK)
  [x] categories (4 columns)

Selected: users, orders, products, categories
Generate with auth? [Y/n]: Y
Architecture pattern? [simple]: service-layer
Output directory [./api/]: ./api/

Generating...
  ✓ api/users.json (5 routes, 5 handlers)
  ✓ api/orders.json (5 routes + 2 relationship routes)
  ✓ api/products.json (5 routes)
  ✓ api/categories.json (5 routes)
  ✓ api/api.json (main server config)
  ✓ api/.env.example

Done! Run: go-json serve api/api.json --dev
```

---

## §25. Architecture Patterns + Custom Templates

### 25.1 Two Levels of Patterns

**Level 1: go-json program structure** (how JSON files are organized):

| Pattern | Structure | Best For |
|---|---|---|
| `simple` | Single JSON file, all functions inline | Prototypes, small APIs |
| `service-layer` | Handlers → services → queries in separate files | Medium apps |
| `modular` | Per-module directories with handlers/service/queries | Large apps |

**Level 2: Codegen output structure** (how generated Go/JS/Python code is organized):

| Pattern | Structure | Best For |
|---|---|---|
| `simple` | Flat files: main.go, handlers.go | Small services |
| `service-layer` | handlers/ + services/ + repositories/ | Medium apps |
| `ddd` | cmd/ + internal/domain/ + application/ + infrastructure/ | Enterprise, complex domain |
| `hexagonal` | cmd/ + internal/core/ (ports+services) + adapters/ (inbound+outbound) | High testability needs |

### 25.2 Using Patterns

```bash
# go-json program pattern (affects JSON structure)
go-json generate crud --from-db --dsn "..." --table users --pattern service-layer

# Codegen output pattern (affects generated code structure)
go-json codegen api.json --target go --framework fiber --pattern ddd
```

### 25.3 Pattern: Simple (Default)

**go-json:**
```
api.json          ← everything in one file
```

**Codegen Go output:**
```
generated/
├── main.go
├── handlers.go
├── go.mod
└── .env.example
```

### 25.4 Pattern: Service Layer

**go-json:**
```
api.json
functions/
├── handlers/users.json     ← parse request, call service, format response
├── services/users.json     ← business logic, validation, orchestration
└── queries/users.json      ← pure database calls
```

**Codegen Go output:**
```
generated/
├── main.go
├── handlers/users.go
├── services/users.go
├── repositories/users.go
├── models/users.go
├── go.mod
└── .env.example
```

### 25.5 Pattern: DDD (Codegen Only)

**Codegen Go output:**
```
generated/
├── cmd/api/main.go
├── internal/
│   ├── domain/user/
│   │   ├── entity.go
│   │   ├── repository.go      (interface)
│   │   └── service.go
│   ├── application/user/
│   │   ├── create_user.go     (use case)
│   │   ├── list_users.go
│   │   └── dto.go
│   └── infrastructure/
│       ├── persistence/postgres/user_repo.go
│       └── http/handlers/user_handler.go
├── go.mod
└── .env.example
```

### 25.6 Pattern: Hexagonal (Codegen Only)

**Codegen Go output:**
```
generated/
├── cmd/api/main.go
├── internal/
│   ├── core/
│   │   ├── domain/user.go
│   │   ├── ports/
│   │   │   ├── inbound.go     (use case interfaces)
│   │   │   └── outbound.go    (repository interfaces)
│   │   └── services/user_service.go
│   └── adapters/
│       ├── inbound/http/handler.go
│       └── outbound/postgres/user_repo.go
├── go.mod
└── .env.example
```

### 25.7 Custom Templates

Users can export, modify, and reuse templates:

```bash
# Export built-in pattern as template
go-json generate --export-pattern ddd --output ./my-templates/ddd/

# Use custom template
go-json generate crud --from-db --dsn "..." --table users --pattern ./my-templates/ddd/
```

**Template structure:**

```
my-templates/ddd/
├── template.json                                    ← metadata
├── cmd/api/main.go.tmpl
├── internal/domain/{{.Model}}/entity.go.tmpl
├── internal/domain/{{.Model}}/repository.go.tmpl
├── internal/domain/{{.Model}}/service.go.tmpl
├── internal/application/{{.Model}}/create.go.tmpl
├── internal/application/{{.Model}}/dto.go.tmpl
├── internal/infrastructure/persistence/{{.Driver}}/{{.Model}}_repo.go.tmpl
├── internal/infrastructure/http/handlers/{{.Model}}_handler.go.tmpl
├── go.mod.tmpl
└── .env.example.tmpl
```

**`template.json`:**

```json
{
  "name": "ddd",
  "description": "Domain-Driven Design pattern for Go",
  "language": "go",
  "files": [
    {"template": "cmd/api/main.go.tmpl", "output": "cmd/api/main.go", "once": true},
    {"template": "internal/domain/{{.model}}/entity.go.tmpl", "per_model": true},
    {"template": "internal/domain/{{.model}}/repository.go.tmpl", "per_model": true}
  ]
}
```

**Template variables available:**

| Variable | Type | Description |
|---|---|---|
| `{{.Model}}` | string | PascalCase: `User`, `OrderItem` |
| `{{.model}}` | string | camelCase: `user`, `orderItem` |
| `{{.table}}` | string | snake_case: `users`, `order_items` |
| `{{.Fields}}` | []Field | All columns with metadata |
| `{{.PrimaryKey}}` | Field | Primary key column |
| `{{.ForeignKeys}}` | []FK | Foreign key relationships |
| `{{.RequiredFields}}` | []Field | NOT NULL without default |
| `{{.OptionalFields}}` | []Field | Nullable or has default |
| `{{.AutoFields}}` | []Field | Auto-generated (id, timestamps) |
| `{{.FilterableFields}}` | []Field | Candidates for query filters |
| `{{.UniqueFields}}` | []Field | Unique constraint columns |
| `{{.Driver}}` | string | `postgres`, `mysql`, `sqlite` |
| `{{.Framework}}` | string | `fiber`, `echo`, `express`, `fastapi` |
| `{{.HasAuth}}` | bool | Auth enabled |
| `{{.AuthStrategy}}` | string | `jwt`, `apikey`, `basic` |
| `{{.Port}}` | int | Server port |

**Community templates (future):**

```bash
go-json template install github.com/user/go-json-clean-arch
go-json template list
go-json generate crud --pattern clean-arch
```

---

## §26. Implementation Scope

### Package Structure

```
packages/go-json/
├── server/                  NEW — web server runtime
│   ├── server.go            Server orchestrator
│   ├── router.go            Route parsing and registration
│   ├── handler.go           Request→Execute→Response bridge
│   ├── middleware.go         Built-in middleware implementations
│   ├── auth.go              Plugable auth strategies
│   ├── jwt.go               JWT sign/verify/decode/refresh
│   ├── template.go          Template engine wrapper
│   ├── static.go            Static file serving
│   ├── config.go            Server config parsing
│   ├── openapi.go           OpenAPI spec generator + Swagger UI
│   └── adapters/            Framework adapters
│       ├── adapter.go       ServerAdapter interface
│       ├── fiber.go         Fiber adapter (default)
│       ├── nethttp.go       net/http adapter
│       ├── echo.go          Echo adapter
│       ├── gin.go           Gin adapter
│       └── chi.go           Chi adapter
├── generate/                NEW — code generators / scaffolding
│   ├── generate.go          CLI generate command dispatcher
│   ├── crud.go              CRUD generator (manual + from-db)
│   ├── auth.go              Auth scaffold generator
│   ├── project.go           Project scaffold generator
│   ├── introspect.go        Database introspection (information_schema)
│   ├── introspect_pg.go     PostgreSQL-specific introspection
│   ├── introspect_mysql.go  MySQL-specific introspection
│   ├── introspect_sqlite.go SQLite-specific introspection (PRAGMA)
│   ├── typemap.go           DB type → go-json type → OpenAPI type mapping
│   └── templates/           Built-in pattern templates
│       ├── simple/          Simple pattern (default)
│       ├── service-layer/   Service layer pattern
│       ├── ddd/             DDD pattern (codegen only)
│       └── hexagonal/       Hexagonal pattern (codegen only)
├── cmd/go-json/
│   └── main.go              Add "serve", "generate", "openapi" commands
├── stdlib/
│   ├── path.go              NEW — basename, dirname, extname, joinpath
│   └── json.go              NEW — toJSON, fromJSON
├── io/
│   ├── sql_params.go        NEW — unified query parameter translation
│   └── ...                  (existing + fs.stat/copy/move/glob)
├── codegen/
│   ├── server.go            Server codegen interface + registry
│   ├── deps.go              NEW — dependency detection + go.mod/package.json generation
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
