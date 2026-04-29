# Web Server

go-json can run as a declarative web server. Any program with a `routes` key is a server program, started with `go-json serve`.

```json
{
  "name": "my_api",
  "server": { "port": 3000 },
  "functions": {
    "hello": {
      "params": { "request": "map" },
      "steps": [
        { "return": { "value": { "status": 200, "body": { "message": "Hello, World!" } } } }
      ]
    }
  },
  "routes": [
    { "method": "GET", "path": "/hello", "handler": "hello" }
  ]
}
```

```bash
go-json serve api.json
# Server running at http://localhost:3000
```

---

## Server Configuration

The `server` block configures the HTTP server. All fields are optional — sensible defaults apply.

```json
{
  "name": "my_api",
  "server": {
    "framework": "fiber",
    "port": 3000,
    "host": "0.0.0.0",
    "static": "./public",
    "templates": "./templates",
    "cors": {
      "origins": ["*"],
      "methods": ["GET", "POST", "PUT", "DELETE"],
      "headers": ["Authorization", "Content-Type"],
      "max_age": 86400
    },
    "auth": {
      "default": "jwt",
      "strategies": {
        "jwt": {
          "type": "bearer",
          "secret_env": "JWT_SECRET",
          "algorithm": "HS256",
          "expiry": "24h"
        },
        "apikey": {
          "type": "api_key",
          "header": "X-API-Key",
          "keys_env": "API_KEYS"
        },
        "basic": {
          "type": "basic",
          "users_env": "BASIC_AUTH_USERS",
          "realm": "My App"
        },
        "custom": {
          "type": "custom",
          "handler": "validateToken"
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
  }
}
```

| Field | Default | Description |
|-------|---------|-------------|
| `framework` | `"fiber"` | HTTP framework adapter. See [Supported Frameworks](#supported-frameworks). |
| `port` | `3000` | Listen port. Overridden by `--port` CLI flag. |
| `host` | `"0.0.0.0"` | Bind address. Overridden by `--host` CLI flag. |
| `static` | — | Static file directory. String or object with `dir` + `prefix`. |
| `templates` | — | Template directory for server-side rendering. |
| `cors` | — | CORS configuration. |
| `auth` | — | Authentication strategies. See [Auth System](#auth-system). |
| `rate_limit` | — | Global rate limiting. |
| `graceful_shutdown` | `"10s"` | Timeout for in-flight requests on SIGINT/SIGTERM. |
| `read_timeout` | `"30s"` | Maximum duration for reading the request. |
| `write_timeout` | `"30s"` | Maximum duration for writing the response. |
| `max_body_size` | `"10mb"` | Maximum request body size. |

### Supported Frameworks

go-json uses a `ServerAdapter` interface that abstracts the underlying HTTP framework. Five adapters are available:

| Framework | Value | Notes |
|-----------|-------|-------|
| [Fiber](https://gofiber.io/) | `"fiber"` | Default. High performance, Express-inspired. |
| net/http | `"net/http"` | Go standard library. Zero dependencies. |
| [Echo](https://echo.labstack.com/) | `"echo"` | Build-tagged. |
| [Gin](https://gin-gonic.com/) | `"gin"` | Build-tagged. |
| [Chi](https://go-chi.io/) | `"chi"` | Build-tagged. |

---

## Routes

Routes map HTTP methods and URL paths to go-json handler functions.

```json
{
  "routes": [
    { "method": "GET", "path": "/api/users", "handler": "listUsers" },
    { "method": "GET", "path": "/api/users/:id", "handler": "getUser" },
    { "method": "POST", "path": "/api/users", "handler": "createUser", "middleware": ["auth"] },
    { "method": "PUT", "path": "/api/users/:id", "handler": "updateUser", "middleware": ["auth"] },
    { "method": "DELETE", "path": "/api/users/:id", "handler": "deleteUser", "middleware": ["auth"] }
  ]
}
```

### Route Fields

| Field | Required | Description |
|-------|----------|-------------|
| `method` | Yes | HTTP method: `GET`, `POST`, `PUT`, `PATCH`, `DELETE`, `OPTIONS`, `HEAD`. |
| `path` | Yes | URL path. Supports `:id` params and `*` wildcard. |
| `handler` | Yes | Name of the go-json function to execute. |
| `middleware` | No | Array of middleware names applied to this route. |
| `render` | No | Template path for server-side rendering. |
| `api` | No | OpenAPI annotation object for richer docs. |

### Path Parameters

Use `:name` syntax for dynamic segments:

```json
{ "method": "GET", "path": "/api/users/:id", "handler": "getUser" }
{ "method": "GET", "path": "/api/posts/:postId/comments/:commentId", "handler": "getComment" }
```

Parameters are available in `request.params`:

```json
{ "let": "userId", "expr": "request.params.id" }
```

### Wildcard Routes

Use `*` to match any remaining path:

```json
{ "method": "GET", "path": "/files/*", "handler": "serveFile" }
```

### Route Groups

Groups apply a shared prefix and middleware to nested routes. Middleware merges in order: **global → group → route**.

```json
{
  "routes": [
    {
      "prefix": "/api/admin",
      "middleware": ["auth", "requireAdmin"],
      "routes": [
        { "method": "GET", "path": "/stats", "handler": "getStats" },
        { "method": "GET", "path": "/users", "handler": "listAllUsers" },
        { "method": "DELETE", "path": "/users/:id", "handler": "deleteUser" }
      ]
    }
  ]
}
```

The above produces:

| Method | Full Path | Middleware |
|--------|-----------|------------|
| GET | `/api/admin/stats` | auth → requireAdmin → getStats |
| GET | `/api/admin/users` | auth → requireAdmin → listAllUsers |
| DELETE | `/api/admin/users/:id` | auth → requireAdmin → deleteUser |

Groups can nest arbitrarily:

```json
{
  "routes": [
    {
      "prefix": "/api",
      "middleware": ["logger"],
      "routes": [
        { "method": "GET", "path": "/health", "handler": "healthCheck" },
        {
          "prefix": "/v1",
          "middleware": ["auth"],
          "routes": [
            { "method": "GET", "path": "/users", "handler": "listUsers" }
          ]
        }
      ]
    }
  ]
}
```

---

## Request Object

Every handler function receives a `request` parameter containing the full HTTP request context.

```json
"getUser": {
  "params": { "request": "map" },
  "steps": [
    { "let": "id", "expr": "request.params.id" },
    { "let": "search", "expr": "request.query.q" },
    { "let": "authHeader", "expr": "request.headers['Authorization']" },
    { "let": "email", "expr": "request.body.email" }
  ]
}
```

### Request Fields

| Field | Type | Description |
|-------|------|-------------|
| `method` | string | HTTP method (`GET`, `POST`, etc.). |
| `path` | string | Request path. |
| `url` | string | Full request URL. |
| `params` | map | Path parameters (`:id` → `params.id`). |
| `query` | map | Query string parameters (`?q=foo` → `query.q`). |
| `headers` | map | Request headers. |
| `body` | any | Parsed request body (see below). |
| `cookies` | map | Request cookies. |
| `ip` | string | Client IP address. |
| `user` | map | Set by auth middleware. Contains decoded user info. |
| `store` | map | Mutable map for middleware data passing. |

### Body Parsing

The request body is automatically parsed based on `Content-Type`:

| Content-Type | Parsed As |
|-------------|-----------|
| `application/json` | map or array |
| `application/x-www-form-urlencoded` | `map[string]string` |
| `multipart/form-data` | map with `_file` objects |
| anything else | raw string |

#### File Uploads

Multipart file fields become objects with `_file: true`:

```json
{
  "_file": true,
  "filename": "photo.jpg",
  "size": 204800,
  "content_type": "image/jpeg",
  "temp_path": "/tmp/go-json-upload-123456"
}
```

Access in a handler:

```json
{
  "let": "file", "expr": "request.body.avatar",
  "_c": "file.filename, file.size, file.content_type, file.temp_path"
}
```

Temp files are cleaned up automatically after the request completes.

---

## Response Convention

Handlers return a map describing the HTTP response. The server interprets the map fields to build the actual response.

### JSON Response

```json
{ "return": { "value": { "status": 200, "body": { "id": 1, "name": "Alice" } } } }
```

### Template Rendering

```json
{ "return": { "value": { "data": { "title": "Home", "items": [] }, "render": "pages/home.html" } } }
```

### Redirect

```json
{ "return": { "value": { "redirect": "/login" } } }
{ "return": { "value": { "redirect": "/login", "status": 301 } } }
```

Default redirect status is `302 Found`.

### Custom Headers

```json
{
  "return": { "value": {
    "status": 200,
    "body": "...",
    "headers": { "X-Custom": "value", "X-Request-Id": "abc-123" }
  }}
}
```

### Set Cookies

```json
{
  "return": { "value": {
    "status": 200,
    "body": "...",
    "cookies": [
      { "name": "token", "value": "abc123", "max_age": 3600, "http_only": true },
      { "name": "theme", "value": "dark", "max_age": 31536000 }
    ]
  }}
}
```

### Error Response

```json
{ "return": { "value": { "error": "Something went wrong" } } }
{ "return": { "value": { "status": 404, "body": { "error": "User not found" } } } }
```

`"error"` without `"status"` defaults to `500 Internal Server Error`.

### No Content

If a handler returns `nil` or has no return statement, the server responds with `204 No Content`.

### Response Field Summary

| Field | Type | Description |
|-------|------|-------------|
| `status` | int | HTTP status code. |
| `body` | any | Response body. Maps/arrays are JSON-encoded. |
| `headers` | map | Additional response headers. |
| `cookies` | array | Cookies to set. Each: `name`, `value`, `max_age`, `http_only`, `secure`, `path`, `domain`, `same_site`. |
| `redirect` | string | URL to redirect to. |
| `render` | string | Template path for server-side rendering. |
| `data` | map | Template data (used with `render`). |
| `error` | string | Error message. Sets status to 500 if no `status` given. |

---

## Middleware

Middleware runs before (and optionally after) route handlers. Middleware is specified as an array of names at the global, group, or route level.

### Execution Order

```
Global middleware → Group middleware → Route middleware → Handler
```

If any middleware returns a response, the chain stops (short-circuit). The handler is not called.

### Built-in Middleware

| Name | Description |
|------|-------------|
| `logger` | Logs method, path, status, and duration for every request. |
| `recover` | Catches panics and returns 500 instead of crashing. |
| `cors` | Applies CORS headers from `server.cors` config. |
| `auth` | Authenticates using the default strategy from `server.auth.default`. |
| `auth:<strategy>` | Authenticates using a specific named strategy (e.g., `auth:apikey`). |
| `rate_limit` | Enforces rate limiting from `server.rate_limit` config. |
| `request_id` | Adds a unique `X-Request-Id` header to every response. |
| `compress` | Gzip/deflate response compression. |
| `secure` | Sets security headers (X-Frame-Options, X-Content-Type-Options, etc.). |

`"jwt"` is an alias for `"auth:jwt"`.

### Custom Middleware

Any go-json function can be used as middleware. The function receives `request` in scope and can:

1. **Modify `request.store`** — pass data to downstream middleware and the handler.
2. **Return a response** — short-circuit the chain (e.g., return 403).
3. **Return nothing** — pass through to the next middleware or handler.

```json
"requireAdmin": {
  "params": { "request": "map" },
  "steps": [
    {
      "if": "request.user == nil || request.user.role != 'admin'",
      "then": [
        { "return": { "value": { "status": 403, "body": { "error": "Admin access required" } } } }
      ]
    }
  ]
}
```

```json
"requestTimer": {
  "params": { "request": "map" },
  "steps": [
    { "set": "request.store.started_at", "expr": "now()" }
  ]
}
```

### Applying Middleware

**Global** — applies to all routes:

```json
{
  "server": { "middleware": ["logger", "recover", "cors"] },
  "routes": [...]
}
```

**Group-level** — applies to all routes in the group:

```json
{
  "prefix": "/api",
  "middleware": ["auth"],
  "routes": [...]
}
```

**Route-level** — applies to a single route:

```json
{ "method": "POST", "path": "/api/users", "handler": "createUser", "middleware": ["auth", "requireAdmin"] }
```

Middleware merges additively. A route inside an `auth` group with its own `requireAdmin` middleware runs: `auth → requireAdmin → handler`.

---

## Auth System

go-json supports four authentication strategy types, configured in `server.auth.strategies`. The `server.auth.default` key sets which strategy `"auth"` middleware uses.

### Bearer / JWT

Extracts a token from the `Authorization: Bearer <token>` header or a cookie. Validates the signature and expiry. On success, injects the decoded payload into `request.user`.

```json
{
  "auth": {
    "default": "jwt",
    "strategies": {
      "jwt": {
        "type": "bearer",
        "secret_env": "JWT_SECRET",
        "algorithm": "HS256",
        "expiry": "24h"
      }
    }
  }
}
```

The secret is read from the environment variable named in `secret_env`. Never put secrets directly in JSON.

### API Key

Extracts a key from a header (default `X-API-Key`) or query parameter. Validates against a comma-separated list in an environment variable.

```json
{
  "strategies": {
    "apikey": {
      "type": "api_key",
      "header": "X-API-Key",
      "keys_env": "API_KEYS"
    }
  }
}
```

Environment variable format: `key1:name1,key2:name2`

```bash
API_KEYS="sk-abc123:alice,sk-def456:bob"
```

On success, `request.user` is set to `{ "name": "alice", "key": "sk-abc123" }`.

### Basic Auth

Extracts credentials from the `Authorization: Basic <base64>` header. Validates against a comma-separated list in an environment variable.

```json
{
  "strategies": {
    "basic": {
      "type": "basic",
      "users_env": "BASIC_AUTH_USERS",
      "realm": "My App"
    }
  }
}
```

Environment variable format: `user1:pass1,user2:pass2`

```bash
BASIC_AUTH_USERS="admin:secret123,readonly:viewer"
```

### Custom Auth

Executes a go-json function for authentication. The function receives the request and must return either a user object (success) or a response with a status code (failure).

```json
{
  "strategies": {
    "custom": {
      "type": "custom",
      "handler": "validateToken"
    }
  }
}
```

```json
"validateToken": {
  "params": { "request": "map" },
  "steps": [
    { "let": "token", "expr": "request.headers['X-Custom-Token']" },
    {
      "if": "token == nil",
      "then": [
        { "return": { "value": { "status": 401, "body": { "error": "Token required" } } } }
      ]
    },
    { "let": "user", "call": "lookupToken", "with": { "token": "token" } },
    {
      "if": "user == nil",
      "then": [
        { "return": { "value": { "status": 401, "body": { "error": "Invalid token" } } } }
      ]
    },
    { "return": { "value": { "id": "user.id", "email": "user.email" } } }
  ]
}
```

If the function returns a map without a `status` field, it's treated as the user object and injected into `request.user`. If it returns a map with a `status` field, it's treated as an HTTP response (authentication failed).

### Route-Level Auth

Use `"auth"` to apply the default strategy, or `"auth:<name>"` for a specific one:

```json
{ "method": "GET", "path": "/api/data", "handler": "getData", "middleware": ["auth"] },
{ "method": "GET", "path": "/api/data", "handler": "getData", "middleware": ["auth:apikey"] },
{ "method": "GET", "path": "/api/data", "handler": "getData", "middleware": ["auth:basic"] }
```

---

## JWT Module

Beyond the `auth:jwt` middleware, go-json provides callable JWT functions for use in handler logic.

### jwt.sign(payload, expiry)

Creates a signed JWT token.

```json
{ "let": "token", "expr": "jwt.sign({'user_id': user.id, 'email': user.email}, '24h')" }
```

### jwt.verify(token)

Validates a token's signature and expiry. Returns the decoded payload or an error.

```json
{ "let": "decoded", "expr": "jwt.verify(token)" }
```

### jwt.decode(token)

Decodes a token **without** validating the signature. Useful for debugging.

```json
{ "let": "claims", "expr": "jwt.decode(token)" }
```

### jwt.refresh(token, expiry)

Creates a new token with the same payload but an extended expiry.

```json
{ "let": "newToken", "expr": "jwt.refresh(oldToken, '24h')" }
```

### Login Flow Example

A complete login endpoint with JWT token generation and cookie setting:

```json
"login": {
  "params": { "request": "map" },
  "steps": [
    { "let": "user", "call": "findUserByEmail", "with": { "email": "request.body.email" } },
    {
      "if": "user == nil",
      "then": [
        { "return": { "value": { "status": 401, "body": { "error": "Invalid credentials" } } } }
      ]
    },
    {
      "let": "validPassword",
      "call": "verifyPassword",
      "with": { "hash": "user.password_hash", "plain": "request.body.password" }
    },
    {
      "if": "!validPassword",
      "then": [
        { "return": { "value": { "status": 401, "body": { "error": "Invalid credentials" } } } }
      ]
    },
    { "let": "token", "expr": "jwt.sign({'user_id': user.id, 'email': user.email, 'role': user.role}, '24h')" },
    {
      "return": { "with": {
        "status": "200",
        "body": "{'token': token, 'user': {'id': user.id, 'name': user.name, 'email': user.email}}",
        "cookies": "[{'name': 'token', 'value': token, 'max_age': 86400, 'http_only': true}]"
      }}
    }
  ]
}
```

### Token Refresh Endpoint

```json
"refreshToken": {
  "params": { "request": "map" },
  "steps": [
    { "let": "oldToken", "expr": "request.headers['Authorization']" },
    {
      "if": "oldToken == nil",
      "then": [
        { "return": { "value": { "status": 401, "body": { "error": "No token provided" } } } }
      ]
    },
    { "set": "oldToken", "expr": "replace(oldToken, 'Bearer ', '')" },
    { "let": "newToken", "expr": "jwt.refresh(oldToken, '24h')" },
    {
      "return": { "value": {
        "status": 200,
        "body": { "token": "newToken" },
        "cookies": [{ "name": "token", "value": "newToken", "max_age": 86400, "http_only": true }]
      }}
    }
  ]
}
```

---

## Template Engine

go-json uses Go's `html/template` for server-side rendering with automatic XSS protection via context-aware escaping.

### Directory Structure

```
templates/
├── layouts/
│   └── base.html
├── partials/
│   ├── header.html
│   └── footer.html
└── pages/
    ├── home.html
    └── users.html
```

### Rendering a Template

Return `render` + `data` from a handler:

```json
"homePage": {
  "params": { "request": "map" },
  "steps": [
    { "let": "users", "call": "getAllUsers" },
    {
      "return": { "value": {
        "data": { "title": "Home", "users": "users" },
        "render": "pages/home.html"
      }}
    }
  ]
}
```

Or use the `render` field on the route itself:

```json
{ "method": "GET", "path": "/", "handler": "homePage", "render": "pages/home.html" }
```

### Built-in Template Functions

20+ functions are available in templates:

| Function | Description | Example |
|----------|-------------|---------|
| `json` | Marshal to JSON string | `{{ json .data }}` |
| `formatDate` | Format a date | `{{ formatDate .date "YYYY-MM-DD" }}` |
| `upper` | Uppercase | `{{ upper .name }}` |
| `lower` | Lowercase | `{{ lower .name }}` |
| `truncate` | Truncate with ellipsis | `{{ truncate .text 100 }}` |
| `default` | Default value if empty | `{{ default .name "Anonymous" }}` |
| `safeHTML` | Mark HTML as safe (skip escaping) | `{{ safeHTML .content }}` |
| `urlEncode` | URL-encode a string | `{{ urlEncode .query }}` |
| `add` | Addition | `{{ add .a .b }}` |
| `sub` | Subtraction | `{{ sub .a .b }}` |
| `mul` | Multiplication | `{{ mul .price .qty }}` |
| `div` | Division | `{{ div .total .count }}` |
| `mod` | Modulo | `{{ mod .index 2 }}` |
| `seq` | Generate integer sequence | `{{ range seq 1 10 }}...{{ end }}` |

### Layouts and Partials

**Layout** (`templates/layouts/base.html`):

```html
<!DOCTYPE html>
<html>
<head><title>{{ .title }}</title></head>
<body>
  {{ template "header" . }}
  {{ template "content" . }}
  {{ template "footer" . }}
</body>
</html>
```

**Partial** (`templates/partials/header.html`):

```html
{{ define "header" }}
<nav>
  <a href="/">Home</a>
  <a href="/about">About</a>
</nav>
{{ end }}
```

### Caching

- **Production mode**: Templates are parsed once and cached.
- **Dev mode** (`--dev`): Templates are re-parsed on every request for instant feedback.

---

## Static Files

Serve static assets (CSS, JS, images) from a directory.

### Simple Configuration

```json
{ "server": { "static": "./public" } }
```

Files in `./public/` are served at the root path. `./public/style.css` → `GET /style.css`.

### With Prefix

```json
{
  "server": {
    "static": {
      "dir": "./assets",
      "prefix": "/static"
    }
  }
}
```

Files in `./assets/` are served under `/static/`. `./assets/style.css` → `GET /static/style.css`.

### Security

- **Path traversal blocked** — requests containing `..` are rejected.
- **Hidden files not served** — files starting with `.` are not accessible.

---

## OpenAPI / Swagger

go-json auto-generates an OpenAPI 3.0 specification from your routes. Zero configuration required.

### Enable Swagger UI

```bash
go-json serve api.json --docs
```

Swagger UI is available at `/docs`.

### Export OpenAPI Spec

```bash
go-json openapi api.json --output openapi.json
```

### Route Annotations

Add an `api` field to routes for richer documentation:

```json
{
  "method": "POST",
  "path": "/api/users",
  "handler": "createUser",
  "middleware": ["auth"],
  "api": {
    "summary": "Create a new user",
    "description": "Creates a user account and returns the created user object.",
    "tags": ["Users"],
    "body": {
      "name": { "type": "string", "required": true },
      "email": { "type": "string", "required": true },
      "role": { "type": "string", "enum": ["admin", "user"], "default": "user" }
    },
    "query": {
      "notify": { "type": "boolean", "description": "Send welcome email" }
    },
    "responses": {
      "201": { "description": "User created" },
      "400": { "description": "Validation error" },
      "409": { "description": "Email already exists" }
    }
  }
}
```

Auth strategies are automatically mapped to OpenAPI security schemes.

---

## Health Endpoint

Every go-json server exposes a built-in `/health` endpoint. It bypasses all middleware (including auth) and returns:

```json
{
  "status": "ok",
  "name": "my_api",
  "uptime": 3600
}
```

`uptime` is in seconds since server start.

---

## Complete Example

A full REST API with authentication, middleware, and CRUD operations:

```json
{
  "name": "todo_api",
  "go_json": "1",
  "server": {
    "port": 3000,
    "cors": {
      "origins": ["http://localhost:5173"],
      "methods": ["GET", "POST", "PUT", "DELETE"],
      "headers": ["Authorization", "Content-Type"]
    },
    "auth": {
      "default": "jwt",
      "strategies": {
        "jwt": {
          "type": "bearer",
          "secret_env": "JWT_SECRET",
          "algorithm": "HS256",
          "expiry": "24h"
        }
      }
    },
    "middleware": ["logger", "recover", "cors"]
  },
  "functions": {
    "listTodos": {
      "params": { "request": "map" },
      "steps": [
        { "let": "userId", "expr": "request.user.user_id" },
        { "let": "todos", "call": "db.query", "with": {
          "sql": "'SELECT * FROM todos WHERE user_id = ? ORDER BY created_at DESC'",
          "params": "[userId]"
        }},
        { "return": { "value": { "status": 200, "body": "todos" } } }
      ]
    },
    "createTodo": {
      "params": { "request": "map" },
      "steps": [
        {
          "if": "request.body.title == nil || request.body.title == ''",
          "then": [
            { "return": { "value": { "status": 400, "body": { "error": "Title is required" } } } }
          ]
        },
        { "let": "todo", "call": "db.execute", "with": {
          "sql": "'INSERT INTO todos (title, user_id) VALUES (?, ?) RETURNING *'",
          "params": "[request.body.title, request.user.user_id]"
        }},
        { "return": { "value": { "status": 201, "body": "todo" } } }
      ]
    },
    "deleteTodo": {
      "params": { "request": "map" },
      "steps": [
        { "call": "db.execute", "with": {
          "sql": "'DELETE FROM todos WHERE id = ? AND user_id = ?'",
          "params": "[request.params.id, request.user.user_id]"
        }},
        { "_c": "No return → 204 No Content" }
      ]
    }
  },
  "routes": [
    { "method": "GET", "path": "/api/todos", "handler": "listTodos", "middleware": ["auth"] },
    { "method": "POST", "path": "/api/todos", "handler": "createTodo", "middleware": ["auth"] },
    { "method": "DELETE", "path": "/api/todos/:id", "handler": "deleteTodo", "middleware": ["auth"] }
  ]
}
```

```bash
JWT_SECRET=my-secret-key go-json serve todo.json --io sql --docs
```

---

## CLI Reference

```bash
# Start server
go-json serve api.json

# Custom port and host
go-json serve api.json --port 8080 --host 127.0.0.1

# Dev mode (template reload, verbose logging)
go-json serve api.json --dev

# Enable Swagger UI at /docs
go-json serve api.json --docs

# Enable I/O modules (required for db, filesystem, http calls)
go-json serve api.json --io http,fs,sql

# Export OpenAPI spec
go-json openapi api.json --output openapi.json

# Combine flags
go-json serve api.json --port 8080 --dev --docs --io http,fs,sql
```

| Flag | Default | Description |
|------|---------|-------------|
| `--port` | `3000` | Override `server.port`. |
| `--host` | `0.0.0.0` | Override `server.host`. |
| `--dev` | `false` | Dev mode: template reload, verbose logging. |
| `--docs` | `false` | Enable Swagger UI at `/docs`. |
| `--io` | — | Comma-separated I/O modules to enable: `http`, `fs`, `sql`, `exec`, `mongodb`, `redis`. |
