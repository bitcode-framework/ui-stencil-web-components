package server

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/bitcode-framework/go-json/lang"
	"github.com/bitcode-framework/go-json/runtime"
	"github.com/bitcode-framework/go-json/server/adapters"
)

// ServerOption configures a Server.
type ServerOption func(*Server)

// WithDevMode enables development mode (template reload, verbose logging).
func WithDevMode(enabled bool) ServerOption {
	return func(s *Server) { s.devMode = enabled }
}

// WithDocs enables the /docs Swagger UI endpoint.
func WithDocs(enabled bool) ServerOption {
	return func(s *Server) { s.docsEnabled = enabled }
}

// WithPort overrides the server port from config.
func WithPort(port int) ServerOption {
	return func(s *Server) { s.portOverride = port }
}

// WithHost overrides the server host from config.
func WithHost(host string) ServerOption {
	return func(s *Server) { s.hostOverride = host }
}

// Server is the main orchestrator for go-json web server mode.
type Server struct {
	config      *lang.ServerConfig
	runtime     *runtime.Runtime
	compiled    *lang.CompiledProgram
	adapter     adapters.ServerAdapter
	templates   *TemplateEngine
	middleware  *MiddlewareRegistry
	devMode       bool
	docsEnabled   bool
	portOverride  int
	hostOverride  string
	startTime     time.Time
}

// NewServer creates a Server from a program file path.
func NewServer(programPath string, rt *runtime.Runtime, opts ...ServerOption) (*Server, error) {
	compiled, err := rt.CompileFile(programPath)
	if err != nil {
		return nil, fmt.Errorf("compile program: %w", err)
	}

	if compiled.AST.Routes == nil || len(compiled.AST.Routes) == 0 {
		return nil, fmt.Errorf("program has no routes — not a server program")
	}

	cfg := compiled.AST.Server
	if cfg == nil {
		cfg = &lang.ServerConfig{}
	}
	MergeDefaults(cfg)

	if err := Validate(cfg); err != nil {
		return nil, fmt.Errorf("server config: %w", err)
	}

	adapter, err := adapters.Get(cfg.Framework)
	if err != nil {
		return nil, err
	}

	s := &Server{
		config:    cfg,
		runtime:   rt,
		compiled:  compiled,
		adapter:   adapter,
		startTime: time.Now(),
	}

	for _, opt := range opts {
		opt(s)
	}

	s.middleware = NewMiddlewareRegistry(cfg, rt, compiled)

	if cfg.JWT != nil {
		jwtFuncs := JWTFunctions(cfg.JWT)
		s.runtime = runtime.NewRuntime(
			runtime.WithStdlibEnv(map[string]any{"jwt": jwtFuncs}),
		)
		s.compiled, err = s.runtime.CompileFile(programPath)
		if err != nil {
			return nil, fmt.Errorf("recompile with JWT functions: %w", err)
		}
		s.middleware = NewMiddlewareRegistry(cfg, s.runtime, s.compiled)
	}

	if cfg.Templates != "" {
		te, err := NewTemplateEngine(cfg.Templates, s.devMode)
		if err != nil {
			log.Printf("[go-json] warning: template engine init failed: %v", err)
		} else {
			s.templates = te
		}
	}

	if err := s.registerRoutes(); err != nil {
		return nil, err
	}

	return s, nil
}

// Start begins listening and blocks until shutdown.
func (s *Server) Start() error {
	host := s.config.Host
	port := s.config.Port
	if s.hostOverride != "" {
		host = s.hostOverride
	}
	if s.portOverride > 0 {
		port = s.portOverride
	}
	addr := fmt.Sprintf("%s:%d", host, port)

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	errCh := make(chan error, 1)
	go func() {
		log.Printf("[go-json] server starting on %s (framework: %s)", addr, s.config.Framework)
		if err := s.adapter.Listen(addr); err != nil {
			errCh <- err
		}
	}()

	select {
	case err := <-errCh:
		return err
	case sig := <-quit:
		log.Printf("[go-json] received signal %v, shutting down...", sig)
		return s.Shutdown()
	}
}

// Shutdown gracefully stops the server.
func (s *Server) Shutdown() error {
	timeout, err := ParseDurationConfig(s.config.GracefulShutdown)
	if err != nil {
		timeout = 10 * time.Second
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	if err := s.adapter.Shutdown(ctx); err != nil {
		return fmt.Errorf("shutdown: %w", err)
	}

	if err := s.runtime.Close(); err != nil {
		log.Printf("[go-json] runtime close error: %v", err)
	}

	log.Printf("[go-json] server stopped")
	return nil
}

func (s *Server) registerRoutes() error {
	flatRoutes := FlattenRoutes(s.compiled.AST.Routes, "", s.compiled.AST.Middleware)

	errs := ValidateRoutes(flatRoutes, s.compiled.Functions)
	if len(errs) > 0 {
		for _, e := range errs {
			log.Printf("[go-json] route error: %v", e)
		}
		return fmt.Errorf("found %d route validation error(s)", len(errs))
	}

	s.registerHealthEndpoint()

	staticCfg := ResolveStaticConfig(s.config)
	if staticCfg != nil {
		if err := ValidateStaticDir(staticCfg.Dir); err != nil {
			log.Printf("[go-json] warning: %v", err)
		} else {
			s.adapter.RegisterStatic(staticCfg.Prefix, staticCfg.Dir)
			log.Printf("[go-json] static files: %s → %s", staticCfg.Prefix, staticCfg.Dir)
		}
	}

	for _, route := range flatRoutes {
		handler := BuildHandler(s.runtime, s.compiled, route)

		var mwChain []adapters.MiddlewareFunc
		if len(route.Middleware) > 0 {
			chain, err := s.middleware.BuildChain(route.Middleware)
			if err != nil {
				return fmt.Errorf("route %s %s: %w", route.Method, route.FullPath, err)
			}
			mwChain = chain
		}

		if s.templates != nil && route.Render != "" {
			handler = s.wrapTemplateHandler(handler, route)
		}

		s.adapter.RegisterRoute(adapters.RouteRegistration{
			Method:     route.Method,
			Path:       route.FullPath,
			Handler:    handler,
			Middleware: mwChain,
		})

		log.Printf("[go-json] %s %s → %s", route.Method, route.FullPath, route.Handler)
	}

	return nil
}

func (s *Server) registerHealthEndpoint() {
	s.adapter.RegisterRoute(adapters.RouteRegistration{
		Method: "GET",
		Path:   "/health",
		Handler: func(ctx *adapters.RequestContext) *adapters.Response {
			return &adapters.Response{
				Status: 200,
				Body: map[string]any{
					"status": "ok",
					"name":   s.compiled.Name,
					"uptime": time.Since(s.startTime).String(),
				},
			}
		},
	})
}

func (s *Server) wrapTemplateHandler(handler adapters.HandlerFunc, route FlatRoute) adapters.HandlerFunc {
	return func(ctx *adapters.RequestContext) *adapters.Response {
		resp := handler(ctx)
		if resp == nil || resp.Template == "" || s.templates == nil {
			return resp
		}

		html, err := s.templates.Render(resp.Template, resp.Data)
		if err != nil {
			log.Printf("[go-json] template render error: %v", err)
			return &adapters.Response{
				Status: 500,
				Body: map[string]any{
					"error": map[string]any{
						"code":    "TEMPLATE_ERROR",
						"message": err.Error(),
					},
				},
			}
		}

		return &adapters.Response{
			Status: resp.Status,
			Headers: map[string]string{
				"Content-Type": "text/html; charset=utf-8",
			},
			Body:    json.RawMessage(`"` + escapeJSONString(html) + `"`),
			Cookies: resp.Cookies,
		}
	}
}

func escapeJSONString(s string) string {
	b, _ := json.Marshal(s)
	if len(b) >= 2 {
		return string(b[1 : len(b)-1])
	}
	return s
}

// IsServerProgram checks if a compiled program has routes (is a server program).
func IsServerProgram(prog *lang.CompiledProgram) bool {
	return prog.AST != nil && len(prog.AST.Routes) > 0
}
