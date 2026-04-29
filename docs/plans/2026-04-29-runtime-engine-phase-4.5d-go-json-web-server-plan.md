# Phase 4.5d — go-json Web Server: Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Add built-in web server execution mode to go-json with declarative routing, plugable framework adapters (Fiber default), middleware (including JWT), template rendering, static files, and framework-aware codegen.

**Architecture:** Server is an execution mode (`go-json serve`), not a module. Handlers are run-to-completion (same as `go-json run`). `ServerAdapter` interface abstracts framework differences. Built-in middleware implemented in Go, custom middleware as go-json functions. Codegen generates framework-specific code per target language.

**Tech Stack:** Go 1.24+, github.com/gofiber/fiber/v2 (default), github.com/golang-jwt/jwt/v5, html/template, net/http

**Design Doc:** `2026-04-29-runtime-engine-phase-4.5d-go-json-web-server.md`
**Depends on:** Phase 4.5c fixes complete

---

## Critical Path

```
Batch 1: Server Core
  Task 1 (config parsing) ──► Task 2 (adapter interface) ──► Task 3 (Fiber adapter)
                                                                │
  Task 4 (route parsing) ──► Task 5 (handler bridge) ──────────┘
                                                                │
Batch 2: Request/Response                                       │
  Task 6 (request object) ──► Task 7 (response convention) ────┘
                                                                │
Batch 3: Middleware                                             │
  Task 8 (middleware chain) ──► Task 9 (built-in: logger, recover, cors)
                                │
  Task 10 (JWT middleware) ──► Task 11 (JWT callable functions)
                                │
  Task 12 (custom middleware) ─┘
                                │
Batch 4: Template + Static      │
  Task 13 (template engine) ──► Task 14 (static files)
                                │
Batch 5: CLI + Dev Mode         │
  Task 15 (serve command) ──► Task 16 (dev mode)
                                │
Batch 6: Additional Adapters    │
  Task 17 (net/http adapter) ──► Task 18 (Echo adapter)
  Task 19 (Gin adapter) ──► Task 20 (Chi adapter)
                                │
Batch 7: Codegen                │
  Task 21 (codegen interface) ──► Task 22 (Go+Fiber codegen)
  Task 23 (Go+net/http codegen)
  Task 24 (JS+Express codegen)
  Task 25 (Python+FastAPI codegen)
                                │
Batch 8: Plugable Auth           │
  Task 26-30 (auth strategies)
                                │
Batch 9: OpenAPI/Swagger        │
  Task 31-33 (spec, UI, CLI)
                                │
Batch 10: SQL Query Params      │
  Task 34-35 (translator + integration)
                                │
Batch 11: FS Enhancement        │
  Task 36 (fs.stat/copy/move/glob) ──► Task 37 (path stdlib)
                                │
Batch 12: Tests                 │
  Task 38-45 (all test suites)
                                │
Batch 13: Docs                  │
  Task 46 (AGENTS.md + docs)
```

---

## Batch 1: Server Core

### Task 1: Server Config Parsing

**Files:**
- Create: `packages/go-json/server/config.go`
- Modify: `packages/go-json/lang/parser.go` (parse `server` and `routes` keys)
- Modify: `packages/go-json/lang/ast.go` (add ServerConfig and RouteConfig to Program)

**Step 1:** Add server-related types to `lang/ast.go`:

```go
type ServerConfig struct {
    Framework       string            `json:"framework"`
    Port            int               `json:"port"`
    Host            string            `json:"host"`
    Static          any               `json:"static"` // string or {dir, prefix}
    Templates       string            `json:"templates"`
    CORS            *CORSConfig       `json:"cors"`
    JWT             *JWTConfig        `json:"jwt"`
    RateLimit       *RateLimitConfig  `json:"rate_limit"`
    GracefulShutdown string           `json:"graceful_shutdown"`
    ReadTimeout     string            `json:"read_timeout"`
    WriteTimeout    string            `json:"write_timeout"`
    MaxBodySize     string            `json:"max_body_size"`
    ErrorTemplates  map[string]string `json:"error_templates"`
}

type CORSConfig struct {
    Origins []string `json:"origins"`
    Methods []string `json:"methods"`
    Headers []string `json:"headers"`
    MaxAge  int      `json:"max_age"`
}

type JWTConfig struct {
    SecretEnv  string            `json:"secret_env"`
    Algorithm  string            `json:"algorithm"`
    Expiry     string            `json:"expiry"`
    Cookie     string            `json:"cookie"`
    Header     string            `json:"header"`
    Prefix     string            `json:"prefix"`
    Claims     map[string]string `json:"claims"`
}

type RateLimitConfig struct {
    Requests int    `json:"requests"`
    Window   string `json:"window"`
    By       string `json:"by"`
}

type RouteConfig struct {
    Method     string        `json:"method"`
    Path       string        `json:"path"`
    Handler    string        `json:"handler"`
    Middleware []string      `json:"middleware"`
    Render     string        `json:"render"`
    Prefix     string        `json:"prefix"`
    Routes     []RouteConfig `json:"routes"`
}
```

Add to `Program`:
```go
type Program struct {
    // ... existing fields ...
    Server     *ServerConfig  `json:"server,omitempty"`
    Routes     []RouteConfig  `json:"routes,omitempty"`
    Middleware []string        `json:"middleware,omitempty"` // global middleware
}
```

**Step 2:** Parse `server`, `routes`, `middleware` in `parser.go`'s `parseProgram()`.

**Step 3:** Create `server/config.go` with defaults and validation:

```go
func DefaultServerConfig() *ServerConfig {
    return &ServerConfig{
        Framework:        "fiber",
        Port:             3000,
        Host:             "0.0.0.0",
        GracefulShutdown: "10s",
        ReadTimeout:      "30s",
        WriteTimeout:     "30s",
        MaxBodySize:      "10mb",
    }
}

func (c *ServerConfig) Validate() error { ... }
```

**Step 4:** Run `go build ./...` and `go test ./...`.

**Step 5:** Commit: `feat(go-json): server config parsing with defaults and validation`

---

### Task 2: Server Adapter Interface

**Files:**
- Create: `packages/go-json/server/adapters/adapter.go`

**Step 1:** Define the `ServerAdapter` interface:

```go
package adapters

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
    User    any
    Store   map[string]any
}

type Response struct {
    Status   int
    Body     any
    Headers  map[string]string
    Redirect string
    Template string
    Data     map[string]any
    Cookies  []CookieConfig
}

type CookieConfig struct {
    Name     string `json:"name"`
    Value    string `json:"value"`
    MaxAge   int    `json:"max_age"`
    Path     string `json:"path"`
    Domain   string `json:"domain"`
    Secure   bool   `json:"secure"`
    HTTPOnly bool   `json:"http_only"`
}

type HandlerFunc func(ctx *RequestContext) *Response
type MiddlewareFunc func(ctx *RequestContext, next func() *Response) *Response

type RouteRegistration struct {
    Method     string
    Path       string
    Handler    HandlerFunc
    Middleware []MiddlewareFunc
}

type ServerAdapter interface {
    RegisterRoute(reg RouteRegistration)
    RegisterGroup(prefix string, middleware []MiddlewareFunc, routes []RouteRegistration)
    RegisterStatic(prefix, dir string)
    SetNotFoundHandler(handler HandlerFunc)
    SetErrorHandler(handler func(err error, ctx *RequestContext) *Response)
    Listen(addr string) error
    Shutdown(ctx context.Context) error
}

type AdapterFactory func() ServerAdapter

var registry = map[string]AdapterFactory{}

func Register(name string, factory AdapterFactory) {
    registry[name] = factory
}

func Get(name string) (ServerAdapter, error) {
    factory, ok := registry[name]
    if !ok {
        return nil, fmt.Errorf("unknown server framework: %s", name)
    }
    return factory(), nil
}
```

**Step 2:** Commit: `feat(go-json): server adapter interface with registry`

---

### Task 3: Fiber Adapter (Default)

**Files:**
- Create: `packages/go-json/server/adapters/fiber.go`
- Modify: `packages/go-json/go.mod` (add fiber dependency)

**Step 1:** Implement `FiberAdapter` that translates between Fiber's `*fiber.Ctx` and `RequestContext`/`Response`.

**Step 2:** Register as default: `func init() { Register("fiber", NewFiberAdapter) }`

**Step 3:** Run `go build ./...`.

**Step 4:** Commit: `feat(go-json): Fiber server adapter (default framework)`

---

### Task 4: Route Parsing and Flattening

**Files:**
- Create: `packages/go-json/server/router.go`

**Step 1:** Implement route flattening (groups → flat list with full paths and merged middleware):

```go
func FlattenRoutes(routes []RouteConfig, parentPrefix string, parentMiddleware []string) []FlatRoute {
    // Recursively flatten route groups into flat route list
    // Each FlatRoute has: Method, FullPath, Handler, AllMiddleware, Render
}
```

**Step 2:** Validate routes: handler exists, no duplicates, render templates exist.

**Step 3:** Commit: `feat(go-json): route parsing with group flattening and validation`

---

### Task 5: Handler Bridge (Request → Execute → Response)

**Files:**
- Create: `packages/go-json/server/handler.go`

**Step 1:** Implement the bridge that converts HTTP requests to go-json execution:

```go
func (s *Server) buildHandler(route FlatRoute) adapters.HandlerFunc {
    return func(ctx *adapters.RequestContext) *adapters.Response {
        // 1. Build input map with request object
        input := map[string]any{
            "request": buildRequestMap(ctx),
        }

        // 2. Execute handler function
        result, err := s.runtime.Execute(s.compiled, input)

        // 3. Convert return value to Response
        return convertToResponse(result, route, err)
    }
}
```

**Step 2:** Implement `buildRequestMap()` and `convertToResponse()`.

**Step 3:** Commit: `feat(go-json): handler bridge — request to execute to response`

---

## Batch 2: Request/Response

### Task 6: Request Object Builder

**Files:**
- Modify: `packages/go-json/server/handler.go`

**Step 1:** Implement full request object construction:
- Parse body based on Content-Type (JSON, form, multipart, raw)
- Extract path params, query params, headers, cookies, IP
- Initialize empty `store` map
- File upload handling: save multipart files to temp dir, expose as `{_file: true, filename, size, content_type, temp_path}`
- Cleanup temp files after handler completes (deferred)

**Step 2:** Commit: `feat(go-json): request object builder with body parsing and file uploads`

---

### Task 7: Response Convention

**Files:**
- Modify: `packages/go-json/server/handler.go`

**Step 1:** Implement response conversion from handler return value:
- `status` + `body` → JSON response
- `data` → template render (if route has `render`)
- `redirect` → redirect response
- `headers` → custom headers
- `cookies` → Set-Cookie
- nil/no return → 204
- error → 500 with error body

**Step 2:** Commit: `feat(go-json): response convention — JSON, template, redirect, cookies`

---

## Batch 3: Middleware

### Task 8: Middleware Chain Engine

**Files:**
- Create: `packages/go-json/server/middleware.go`

**Step 1:** Implement middleware chain execution:

```go
func (s *Server) buildMiddlewareChain(route FlatRoute) []adapters.MiddlewareFunc {
    var chain []adapters.MiddlewareFunc
    for _, name := range route.AllMiddleware {
        if builtin, ok := s.builtinMiddleware[name]; ok {
            chain = append(chain, builtin)
        } else if fn, ok := s.compiled.Functions[name]; ok {
            chain = append(chain, s.buildCustomMiddleware(name, fn))
        }
    }
    return chain
}
```

**Step 2:** Commit: `feat(go-json): middleware chain engine with built-in and custom support`

---

### Task 9: Built-in Middleware (Logger, Recover, CORS)

**Files:**
- Modify: `packages/go-json/server/middleware.go`

**Step 1:** Implement:
- `logger` — log method, path, status, duration
- `recover` — catch panics, return 500
- `cors` — set CORS headers from config
- `secure` — security headers (HSTS, X-Frame-Options, etc.)
- `request_id` — add X-Request-ID
- `compress` — gzip response
- `rate_limit` — per-IP/per-user rate limiting from config, return 429

**Step 2:** Commit: `feat(go-json): built-in middleware — logger, recover, cors, secure, request_id, compress`

---

### Task 10: JWT Middleware

**Files:**
- Create: `packages/go-json/server/jwt.go`
- Modify: `packages/go-json/go.mod` (add golang-jwt/jwt/v5)

**Step 1:** Implement JWT validation middleware:
- Extract token from header or cookie
- Validate signature, expiry, claims
- Inject decoded payload into `request.user`
- Return 401 if invalid

**Step 2:** Commit: `feat(go-json): JWT validation middleware with header and cookie support`

---

### Task 11: JWT Callable Functions

**Files:**
- Modify: `packages/go-json/server/jwt.go`

**Step 1:** Implement JWT functions callable from handler steps:
- `jwt.sign(payload, expiry)` → token string
- `jwt.verify(token)` → payload or error
- `jwt.decode(token)` → payload without validation
- `jwt.refresh(token, expiry)` → new token

**Step 2:** Register as stdlib functions when server mode is active.

**Step 3:** Commit: `feat(go-json): JWT sign, verify, decode, refresh callable functions`

---

### Task 12: Custom Middleware (go-json Functions)

**Files:**
- Modify: `packages/go-json/server/middleware.go`

**Step 1:** Implement custom middleware execution:
- Execute go-json function with `request` in scope
- If function returns response (has status/body/redirect) → short-circuit
- If function returns nothing → pass through, merge `request.store` changes

**Step 2:** Commit: `feat(go-json): custom middleware as go-json functions with short-circuit`

---

## Batch 4: Template + Static

### Task 13: Template Engine

**Files:**
- Create: `packages/go-json/server/template.go`

**Step 1:** Implement template engine wrapper:
- Load templates from directory (recursive)
- Register built-in template functions (json, formatDate, upper, lower, truncate, default, safeHTML, urlEncode, add, sub, mul, seq)
- Render template with data map
- Cache templates in production, reload in dev mode

**Step 2:** Commit: `feat(go-json): template engine with layouts, partials, and built-in functions`

---

### Task 14: Static File Serving

**Files:**
- Create: `packages/go-json/server/static.go`

**Step 1:** Implement static file serving:
- Serve files from configured directory
- Support custom prefix
- Block directory traversal, hidden files
- Set appropriate Content-Type headers

**Step 2:** Commit: `feat(go-json): static file serving with security`

---

## Batch 5: CLI + Dev Mode

### Task 15: `go-json serve` Command

**Files:**
- Modify: `packages/go-json/cmd/go-json/main.go`
- Create: `packages/go-json/server/server.go` (main orchestrator)

**Step 1:** Add `serve` command to CLI:
- Parse program file
- Detect server mode (check for `"routes"` key)
- Validate server config and routes
- Setup runtime with I/O modules
- Register built-in `/health` endpoint (bypasses middleware)
- Build adapter, register routes, start server
- Handle graceful shutdown on SIGTERM/SIGINT

**Step 2:** Create `Server` orchestrator that ties everything together:

```go
type Server struct {
    config   *lang.ServerConfig
    runtime  *runtime.Runtime
    compiled *lang.CompiledProgram
    adapter  adapters.ServerAdapter
    templates *TemplateEngine
    // ...
}

func NewServer(programPath string, opts ...ServerOption) (*Server, error) { ... }
func (s *Server) Start() error { ... }
func (s *Server) Shutdown(ctx context.Context) error { ... }
```

**Step 3:** Commit: `feat(go-json): go-json serve command with graceful shutdown`

---

### Task 16: Dev Mode

**Files:**
- Modify: `packages/go-json/server/server.go`

**Step 1:** Implement dev mode:
- File watching with `fsnotify` (optional dependency)
- Template reload on every request
- Pretty error responses with stack trace
- Verbose request/response logging

**Step 2:** Commit: `feat(go-json): dev mode with file watching and pretty errors`

---

## Batch 6: Additional Framework Adapters

### Task 17: net/http Adapter

**Files:**
- Create: `packages/go-json/server/adapters/nethttp.go`

**Step 1:** Implement `NetHTTPAdapter` using Go 1.22+ `http.ServeMux` with method patterns.

**Step 2:** Commit: `feat(go-json): net/http server adapter`

---

### Task 18: Echo Adapter

**Files:**
- Create: `packages/go-json/server/adapters/echo.go`

**Step 1:** Implement `EchoAdapter`. Build-tagged: `//go:build echo`.

**Step 2:** Commit: `feat(go-json): Echo server adapter`

---

### Task 19: Gin Adapter

**Files:**
- Create: `packages/go-json/server/adapters/gin.go`

**Step 1:** Implement `GinAdapter`. Build-tagged: `//go:build gin`.

**Step 2:** Commit: `feat(go-json): Gin server adapter`

---

### Task 20: Chi Adapter

**Files:**
- Create: `packages/go-json/server/adapters/chi.go`

**Step 1:** Implement `ChiAdapter`. Build-tagged: `//go:build chi`.

**Step 2:** Commit: `feat(go-json): Chi server adapter`

---

## Batch 7: Server Codegen

### Task 21: Server Codegen Interface + Registry

**Files:**
- Create: `packages/go-json/codegen/server.go`

**Step 1:** Define server codegen interface:

```go
type ServerCodegenAdapter interface {
    GenerateServer(program *lang.CompiledProgram) (map[string]string, error) // filename → content
    Framework() string
    Language() string
}

var serverCodegenRegistry = map[string]map[string]ServerCodegenAdapter{} // language → framework → adapter

func RegisterServerCodegen(lang, framework string, adapter ServerCodegenAdapter) { ... }
func GetServerCodegen(lang, framework string) (ServerCodegenAdapter, error) { ... }
func DefaultFramework(lang string) string { ... }
func ResolveFramework(lang, explicit, serverConfig string) string { ... } // selection logic from §12.2
```

**Step 2:** Commit: `feat(go-json): server codegen interface with language×framework registry`

---

### Task 22: Go + Fiber Codegen (Default)

**Files:**
- Create: `packages/go-json/codegen/server_go_fiber.go`

**Step 1:** Generate Fiber server code: main.go, handlers.go, middleware.go, types.go.

**Step 2:** Commit: `feat(go-json): Go+Fiber server codegen`

---

### Task 23: Go + net/http Codegen

**Files:**
- Create: `packages/go-json/codegen/server_go_nethttp.go`

**Step 1:** Generate net/http server code using Go 1.22+ ServeMux patterns.

**Step 2:** Commit: `feat(go-json): Go+net/http server codegen`

---

### Task 24: JavaScript + Express Codegen (Default)

**Files:**
- Create: `packages/go-json/codegen/server_js_express.go`

**Step 1:** Generate Express.js server code: index.js, routes.js, middleware.js.

**Step 2:** Commit: `feat(go-json): JS+Express server codegen`

---

### Task 25: Python + FastAPI Codegen (Default)

**Files:**
- Create: `packages/go-json/codegen/server_py_fastapi.go`

**Step 1:** Generate FastAPI server code: main.py, routes.py, middleware.py.

**Step 2:** Commit: `feat(go-json): Python+FastAPI server codegen`

---

## Batch 8: Plugable Auth System

### Task 26: Auth Strategy Interface + Registry

**Files:**
- Create: `packages/go-json/server/auth.go`
- Modify: `packages/go-json/server/config.go` (add AuthConfig, StrategyConfig)

**Step 1:** Define auth strategy interface:

```go
type AuthStrategy interface {
    Name() string
    Type() string
    Authenticate(ctx *adapters.RequestContext) (any, error) // returns user object or error
}

type AuthRegistry struct {
    strategies map[string]AuthStrategy
    defaultStrategy string
}
```

**Step 2:** Parse `server.auth` config with `default` and `strategies` map.

**Step 3:** Implement `auth` and `auth:<name>` middleware resolution — `auth` uses default, `auth:apikey` uses specific. Keep `jwt` as alias for `auth:jwt`.

**Step 4:** Commit: `feat(go-json): plugable auth strategy interface and registry`

---

### Task 27: Bearer (JWT) Auth Strategy

**Files:**
- Modify: `packages/go-json/server/auth.go`
- Modify: `packages/go-json/server/jwt.go`

**Step 1:** Implement `BearerStrategy` that wraps existing JWT logic. Extract token from header or cookie, validate, return decoded payload.

**Step 2:** Commit: `feat(go-json): bearer/JWT auth strategy`

---

### Task 28: API Key Auth Strategy

**Files:**
- Modify: `packages/go-json/server/auth.go`

**Step 1:** Implement `APIKeyStrategy`:
- Extract key from header (`X-API-Key`) or query param (`api_key`)
- Validate against keys from env var (format: `key1:name1,key2:name2`)
- Return `{"key": "...", "name": "..."}` as user object

**Step 2:** Commit: `feat(go-json): API key auth strategy`

---

### Task 29: Basic Auth Strategy

**Files:**
- Modify: `packages/go-json/server/auth.go`

**Step 1:** Implement `BasicStrategy`:
- Extract credentials from `Authorization: Basic <base64>`
- Validate against users from env var (format: `user1:pass1,user2:pass2`)
- Return `{"username": "..."}` as user object
- Return 401 with `WWW-Authenticate: Basic realm="..."` on failure

**Step 2:** Commit: `feat(go-json): basic auth strategy`

---

### Task 30: Custom Auth Strategy (go-json Function)

**Files:**
- Modify: `packages/go-json/server/auth.go`

**Step 1:** Implement `CustomStrategy`:
- Execute named go-json function with `request` in scope
- Function returns user object → auth success
- Function returns response with status → auth failure (short-circuit)

**Step 2:** Commit: `feat(go-json): custom auth strategy via go-json functions`

---

## Batch 9: OpenAPI / Swagger

### Task 31: OpenAPI Spec Generator

**Files:**
- Create: `packages/go-json/server/openapi.go`

**Step 1:** Implement OpenAPI 3.0 spec generation from program:
- Extract paths from routes (translate `:id` → `{id}`)
- Extract security schemes from `server.auth.strategies`
- Extract tags from route group prefixes
- Parse `api` annotations on routes for body/query/response schemas
- Routes without `api` annotation get minimal spec (method, path, security only)

**Step 2:** Commit: `feat(go-json): OpenAPI 3.0 spec auto-generation from routes`

---

### Task 32: Swagger UI Endpoint

**Files:**
- Modify: `packages/go-json/server/openapi.go`
- Modify: `packages/go-json/server/server.go`

**Step 1:** When `--docs` flag enabled:
- Register `GET /docs` → embedded Swagger UI HTML (use swagger-ui-dist CDN or embedded)
- Register `GET /docs/openapi.json` → generated spec

**Step 2:** Commit: `feat(go-json): Swagger UI endpoint at /docs`

---

### Task 33: `go-json openapi` CLI Command

**Files:**
- Modify: `packages/go-json/cmd/go-json/main.go`

**Step 1:** Add `openapi` command:
```bash
go-json openapi api.json --output openapi.json
go-json openapi api.json  # stdout
```

**Step 2:** Commit: `feat(go-json): go-json openapi CLI command`

---

## Batch 10: SQL Query Parameter Translation

### Task 34: Query Parameter Translator

**Files:**
- Create: `packages/go-json/io/sql_params.go`
- Test: `packages/go-json/io/sql_params_test.go`

**Step 1:** Implement `translateQuery(query, driver string, args any) (string, []any, error)`:
- Detect mode: positional (`[]any` args) or named (`map[string]any` args)
- Parse query, track string literals (skip `?` inside single quotes)
- Handle `??` escape → literal `?`
- Reject mixed `?` and `:name` in same query
- Translate `?` to driver-specific: `$1` (postgres), `@p1` (sqlserver), `:1` (oracle), `?` (sqlite/mysql)
- Translate `:name` to driver-specific: extract names in order, build ordered args for drivers that need positional

**Step 2:** Write comprehensive tests:
- Positional for each driver
- Named for each driver
- `??` escape
- `?` inside string literals
- Mixed positional+named → error
- Arg count mismatch → error
- Named param not found → error

**Step 3:** Commit: `feat(go-json): unified SQL query parameter translation across drivers`

---

### Task 35: Integrate Translator into SQL Module

**Files:**
- Modify: `packages/go-json/io/sql.go`

**Step 1:** Call `translateQuery()` at the start of `sqlQuery()` and `sqlExecute()` before passing to `database/sql`.

**Step 2:** Commit: `feat(go-json): SQL module uses unified parameter translation`

---

## Batch 11: FS Enhancement + Path Utilities

### Task 36: `fs.stat`, `fs.copy`, `fs.move`, `fs.glob`

**Files:**
- Modify: `packages/go-json/io/fs.go`
- Test: `packages/go-json/io/fs_test.go`

**Step 1:** Add new functions to FSModule:
- `fs.stat(path)` → `{name, size, is_dir, is_file, ext, modified, permissions}`
- `fs.copy(src, dst)` → copy file, validate both paths
- `fs.move(src, dst)` → rename/move, validate both paths
- `fs.glob(pattern)` → `filepath.Glob`, validate results against security config

**Step 2:** Enhance `fs.list` — add optional `detail` parameter. When `true`, return `[]FileInfo` instead of `[]string`. Backward compatible (default `false`).

**Step 3:** Write tests for all new functions + enhanced list.

**Step 4:** Commit: `feat(go-json): fs.stat, fs.copy, fs.move, fs.glob and enhanced fs.list`

---

### Task 37: Path Utility Stdlib Functions

**Files:**
- Create: `packages/go-json/stdlib/path.go`
- Modify: `packages/go-json/stdlib/registry.go` (register path functions)
- Test: `packages/go-json/stdlib/path_test.go`

**Step 1:** Implement stdlib path functions (pure string operations, no I/O):
- `basename(path)` → `filepath.Base`
- `dirname(path)` → `filepath.Dir`
- `extname(path)` → `filepath.Ext`
- `joinpath(parts...)` → `filepath.Join`

**Step 2:** Register in `DefaultRegistry()`.

**Step 3:** Write tests — edge cases: empty path, no extension, trailing slash, Windows paths.

**Step 4:** Commit: `feat(go-json): path utility stdlib — basename, dirname, extname, joinpath`

---

## Batch 12: Tests

### Task 38: Server Config Tests

**Files:**
- Create: `packages/go-json/server/config_test.go`

Tests: default config, validation, parsing from JSON, invalid config errors, auth config parsing.

**Commit:** `test(go-json): server config parsing and validation tests`

---

### Task 39: Route Parsing Tests

**Files:**
- Create: `packages/go-json/server/router_test.go`

Tests: basic routes, groups, nested groups, middleware merging, duplicate detection, handler validation, api annotation parsing.

**Commit:** `test(go-json): route parsing, flattening, and validation tests`

---

### Task 40: Handler Bridge Tests

**Files:**
- Create: `packages/go-json/server/handler_test.go`

Tests: request object construction, file upload handling, response conversion (JSON, template, redirect, 204), error handling, timeout.

**Commit:** `test(go-json): handler bridge request/response tests`

---

### Task 41: Middleware + Auth Tests

**Files:**
- Create: `packages/go-json/server/middleware_test.go`
- Create: `packages/go-json/server/auth_test.go`

Tests: chain execution order, short-circuit, built-in middleware, custom middleware, JWT strategy, API key strategy, basic auth strategy, custom auth strategy, `auth` vs `auth:name` resolution.

**Commit:** `test(go-json): middleware chain, auth strategies, and JWT tests`

---

### Task 42: Template + Static Tests

**Files:**
- Create: `packages/go-json/server/template_test.go`
- Create: `packages/go-json/server/static_test.go`

Tests: template rendering, layouts, partials, custom functions, static file serving, directory traversal blocking.

**Commit:** `test(go-json): template engine and static file serving tests`

---

### Task 43: OpenAPI Tests

**Files:**
- Create: `packages/go-json/server/openapi_test.go`

Tests: spec generation from routes, security scheme mapping, api annotation parsing, tag extraction, minimal spec without annotations.

**Commit:** `test(go-json): OpenAPI spec generation tests`

---

### Task 44: SQL Parameter Translation Tests

Already covered in Task 34 (`packages/go-json/io/sql_params_test.go`).

---

### Task 45: Server Codegen Tests

**Files:**
- Create: `packages/go-json/codegen/server_test.go`

Tests: Go+Fiber output, Go+net/http output, JS+Express output, Python+FastAPI output, framework selection logic, default framework per language.

**Commit:** `test(go-json): server codegen tests for all language×framework combinations`

---

## Batch 13: Documentation

### Task 46: Update Documentation

**Files:**
- Modify: `packages/go-json/AGENTS.md`

**Step 1:** Update with:
- New `server/` package in structure
- `go-json serve` and `go-json openapi` commands
- Server mode detection
- Framework adapter pattern
- Plugable auth system (Bearer, API Key, Basic, Custom)
- OpenAPI/Swagger support
- SQL unified query parameters
- Codegen framework selection

**Step 2:** Commit and push: `docs(go-json): update AGENTS.md for Phase 4.5d web server`

---

## Summary

| Batch | Tasks | Scope |
|-------|-------|-------|
| 1: Server Core | 1-5 | Config, adapter interface, Fiber adapter, routing, handler bridge |
| 2: Request/Response | 6-7 | Request object (with file uploads), response convention |
| 3: Middleware | 8-12 | Chain engine, built-in (logger/recover/cors/secure/rate_limit), JWT, custom |
| 4: Template + Static | 13-14 | html/template engine, static file serving |
| 5: CLI + Dev Mode | 15-16 | `go-json serve` command, file watching, pretty errors |
| 6: Framework Adapters | 17-20 | net/http, Echo, Gin, Chi adapters |
| 7: Server Codegen | 21-25 | Interface, Go+Fiber, Go+net/http, JS+Express, Python+FastAPI |
| 8: Plugable Auth | 26-30 | Auth interface, Bearer/JWT, API key, Basic, Custom strategies |
| 9: OpenAPI/Swagger | 31-33 | Spec generator, Swagger UI, CLI command |
| 10: SQL Query Params | 34-35 | Unified `?`/`:name` translation across drivers |
| 11: FS Enhancement | 36-37 | fs.stat/copy/move/glob, path stdlib (basename/dirname/extname/joinpath) |
| 12: Tests | 38-45 | Config, routing, handler, auth, middleware, template, OpenAPI, codegen tests |
| 13: Docs | 46 | AGENTS.md update |
| **Total** | **46 tasks** | |
