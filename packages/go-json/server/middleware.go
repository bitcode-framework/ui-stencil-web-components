package server

import (
	"compress/gzip"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/bitcode-framework/go-json/lang"
	"github.com/bitcode-framework/go-json/runtime"
	"github.com/bitcode-framework/go-json/server/adapters"
	"github.com/google/uuid"
)

// MiddlewareRegistry holds built-in and custom middleware resolvers.
type MiddlewareRegistry struct {
	builtin map[string]adapters.MiddlewareFunc
	config  *lang.ServerConfig
	rt      *runtime.Runtime
	compiled *lang.CompiledProgram
}

// NewMiddlewareRegistry creates a registry with built-in middleware.
func NewMiddlewareRegistry(cfg *lang.ServerConfig, rt *runtime.Runtime, compiled *lang.CompiledProgram) *MiddlewareRegistry {
	mr := &MiddlewareRegistry{
		builtin:  make(map[string]adapters.MiddlewareFunc),
		config:   cfg,
		rt:       rt,
		compiled: compiled,
	}

	mr.builtin["logger"] = loggerMiddleware()
	mr.builtin["recover"] = recoverMiddleware()
	mr.builtin["request_id"] = requestIDMiddleware()
	mr.builtin["compress"] = compressMiddleware()

	if cfg.CORS != nil {
		mr.builtin["cors"] = corsMiddleware(cfg.CORS)
	}

	mr.builtin["secure"] = secureMiddleware()

	if cfg.RateLimit != nil {
		mr.builtin["rate_limit"] = rateLimitMiddleware(cfg.RateLimit)
	}

	if cfg.JWT != nil {
		mr.builtin["jwt"] = JWTMiddleware(cfg.JWT)
	}

	return mr
}

// Resolve returns the middleware function for a given name.
// It checks built-in first, then falls back to custom go-json functions.
func (mr *MiddlewareRegistry) Resolve(name string) (adapters.MiddlewareFunc, error) {
	if mw, ok := mr.builtin[name]; ok {
		return mw, nil
	}

	if mr.compiled != nil {
		if _, ok := mr.compiled.Functions[name]; ok {
			return mr.buildCustomMiddleware(name), nil
		}
	}

	return nil, fmt.Errorf("middleware %q not found", name)
}

// BuildChain resolves a list of middleware names into a chain of MiddlewareFuncs.
func (mr *MiddlewareRegistry) BuildChain(names []string) ([]adapters.MiddlewareFunc, error) {
	chain := make([]adapters.MiddlewareFunc, 0, len(names))
	for _, name := range names {
		mw, err := mr.Resolve(name)
		if err != nil {
			return nil, err
		}
		chain = append(chain, mw)
	}
	return chain, nil
}

func (mr *MiddlewareRegistry) buildCustomMiddleware(funcName string) adapters.MiddlewareFunc {
	return func(ctx *adapters.RequestContext, next func() *adapters.Response) *adapters.Response {
		requestMap := BuildRequestMap(ctx)
		args := map[string]any{
			"request": requestMap,
		}

		result, err := mr.rt.ExecuteFunction(mr.compiled, funcName, args)
		if err != nil {
			log.Printf("[go-json] middleware %q error: %v", funcName, err)
			return &adapters.Response{
				Status: 500,
				Body: map[string]any{
					"error": map[string]any{
						"code":    "MIDDLEWARE_ERROR",
						"message": err.Error(),
					},
				},
			}
		}

		if result != nil {
			if resultMap, ok := result.(map[string]any); ok {
				if _, hasStatus := resultMap["status"]; hasStatus {
					return ConvertToResponse(result, FlatRoute{})
				}
				if _, hasRedirect := resultMap["redirect"]; hasRedirect {
					return ConvertToResponse(result, FlatRoute{})
				}
				if _, hasBody := resultMap["body"]; hasBody {
					return ConvertToResponse(result, FlatRoute{})
				}

				if store, ok := resultMap["store"].(map[string]any); ok {
					for k, v := range store {
						ctx.Store[k] = v
					}
				}
			}
		}

		return next()
	}
}

func loggerMiddleware() adapters.MiddlewareFunc {
	return func(ctx *adapters.RequestContext, next func() *adapters.Response) *adapters.Response {
		start := time.Now()
		resp := next()
		duration := time.Since(start)
		status := 200
		if resp != nil {
			status = resp.Status
		}
		log.Printf("[%s] %s %s → %d (%s)", ctx.Method, ctx.Path, ctx.IP, status, duration)
		return resp
	}
}

func recoverMiddleware() adapters.MiddlewareFunc {
	return func(ctx *adapters.RequestContext, next func() *adapters.Response) (resp *adapters.Response) {
		defer func() {
			if r := recover(); r != nil {
				log.Printf("[go-json] panic in %s %s: %v", ctx.Method, ctx.Path, r)
				resp = &adapters.Response{
					Status: 500,
					Body: map[string]any{
						"error": map[string]any{
							"code":    "INTERNAL_ERROR",
							"message": "Internal Server Error",
						},
					},
				}
			}
		}()
		return next()
	}
}

func corsMiddleware(cfg *lang.CORSConfig) adapters.MiddlewareFunc {
	origins := strings.Join(cfg.Origins, ", ")
	methods := "GET, POST, PUT, PATCH, DELETE, OPTIONS"
	if len(cfg.Methods) > 0 {
		methods = strings.Join(cfg.Methods, ", ")
	}
	headers := "Content-Type, Authorization"
	if len(cfg.Headers) > 0 {
		headers = strings.Join(cfg.Headers, ", ")
	}
	maxAge := "86400"
	if cfg.MaxAge > 0 {
		maxAge = fmt.Sprintf("%d", cfg.MaxAge)
	}

	return func(ctx *adapters.RequestContext, next func() *adapters.Response) *adapters.Response {
		if ctx.Method == "OPTIONS" {
			return &adapters.Response{
				Status: 204,
				Headers: map[string]string{
					"Access-Control-Allow-Origin":  origins,
					"Access-Control-Allow-Methods": methods,
					"Access-Control-Allow-Headers": headers,
					"Access-Control-Max-Age":       maxAge,
				},
			}
		}

		resp := next()
		if resp == nil {
			resp = &adapters.Response{Status: 204}
		}
		if resp.Headers == nil {
			resp.Headers = make(map[string]string)
		}
		resp.Headers["Access-Control-Allow-Origin"] = origins
		return resp
	}
}

func secureMiddleware() adapters.MiddlewareFunc {
	return func(ctx *adapters.RequestContext, next func() *adapters.Response) *adapters.Response {
		resp := next()
		if resp == nil {
			resp = &adapters.Response{Status: 204}
		}
		if resp.Headers == nil {
			resp.Headers = make(map[string]string)
		}
		resp.Headers["X-Content-Type-Options"] = "nosniff"
		resp.Headers["X-Frame-Options"] = "SAMEORIGIN"
		resp.Headers["X-XSS-Protection"] = "1; mode=block"
		resp.Headers["Referrer-Policy"] = "strict-origin-when-cross-origin"
		return resp
	}
}

func requestIDMiddleware() adapters.MiddlewareFunc {
	return func(ctx *adapters.RequestContext, next func() *adapters.Response) *adapters.Response {
		reqID := ctx.Headers["X-Request-ID"]
		if reqID == "" {
			reqID = uuid.New().String()
		}
		ctx.Store["request_id"] = reqID

		resp := next()
		if resp == nil {
			resp = &adapters.Response{Status: 204}
		}
		if resp.Headers == nil {
			resp.Headers = make(map[string]string)
		}
		resp.Headers["X-Request-ID"] = reqID
		return resp
	}
}

func compressMiddleware() adapters.MiddlewareFunc {
	_ = gzip.DefaultCompression
	return func(ctx *adapters.RequestContext, next func() *adapters.Response) *adapters.Response {
		return next()
	}
}

func rateLimitMiddleware(cfg *lang.RateLimitConfig) adapters.MiddlewareFunc {
	window, _ := ParseDurationConfig(cfg.Window)
	if window == 0 {
		window = time.Minute
	}

	type entry struct {
		count   int
		resetAt time.Time
	}

	var mu sync.Mutex
	store := make(map[string]*entry)

	return func(ctx *adapters.RequestContext, next func() *adapters.Response) *adapters.Response {
		key := ctx.IP
		if cfg.By == "user" {
			if user, ok := ctx.User.(map[string]any); ok {
				if id, ok := user["sub"].(string); ok {
					key = id
				}
			}
		}

		mu.Lock()
		e, ok := store[key]
		now := time.Now()
		if !ok || now.After(e.resetAt) {
			e = &entry{count: 0, resetAt: now.Add(window)}
			store[key] = e
		}
		e.count++
		count := e.count
		mu.Unlock()

		if count > cfg.Requests {
			return &adapters.Response{
				Status: 429,
				Body: map[string]any{
					"error": map[string]any{
						"code":    "RATE_LIMIT_EXCEEDED",
						"message": "Too many requests",
					},
				},
				Headers: map[string]string{
					"Retry-After": fmt.Sprintf("%d", int(window.Seconds())),
				},
			}
		}

		return next()
	}
}
