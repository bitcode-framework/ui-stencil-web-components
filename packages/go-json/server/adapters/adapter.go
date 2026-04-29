package adapters

import (
	"context"
	"fmt"
	"sync"
)

// RequestContext holds all HTTP request data in a framework-agnostic form.
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

// Response holds the handler's response in a framework-agnostic form.
type Response struct {
	Status   int
	Body     any
	Headers  map[string]string
	Redirect string
	Template string
	Data     map[string]any
	Cookies  []CookieConfig
}

// CookieConfig describes a Set-Cookie directive.
type CookieConfig struct {
	Name     string `json:"name"`
	Value    string `json:"value"`
	MaxAge   int    `json:"max_age"`
	Path     string `json:"path"`
	Domain   string `json:"domain"`
	Secure   bool   `json:"secure"`
	HTTPOnly bool   `json:"http_only"`
}

// HandlerFunc is the signature for request handlers.
type HandlerFunc func(ctx *RequestContext) *Response

// MiddlewareFunc is the signature for middleware.
// Call next() to pass control to the next handler; return its result or your own.
type MiddlewareFunc func(ctx *RequestContext, next func() *Response) *Response

// RouteRegistration describes a single route to register with an adapter.
type RouteRegistration struct {
	Method     string
	Path       string
	Handler    HandlerFunc
	Middleware []MiddlewareFunc
}

// ServerAdapter abstracts the underlying HTTP framework.
type ServerAdapter interface {
	RegisterRoute(reg RouteRegistration)
	RegisterGroup(prefix string, middleware []MiddlewareFunc, routes []RouteRegistration)
	RegisterStatic(prefix, dir string)
	SetNotFoundHandler(handler HandlerFunc)
	SetErrorHandler(handler func(err error, ctx *RequestContext) *Response)
	Listen(addr string) error
	Shutdown(ctx context.Context) error
}

// AdapterFactory creates a new ServerAdapter instance.
type AdapterFactory func() ServerAdapter

var (
	mu       sync.RWMutex
	registry = map[string]AdapterFactory{}
)

// Register adds a named adapter factory to the global registry.
func Register(name string, factory AdapterFactory) {
	mu.Lock()
	defer mu.Unlock()
	registry[name] = factory
}

// Get returns a new adapter instance by name.
func Get(name string) (ServerAdapter, error) {
	mu.RLock()
	defer mu.RUnlock()
	factory, ok := registry[name]
	if !ok {
		available := make([]string, 0, len(registry))
		for k := range registry {
			available = append(available, k)
		}
		return nil, fmt.Errorf("unknown server framework: %q (available: %v)", name, available)
	}
	return factory(), nil
}

// Available returns the names of all registered adapters.
func Available() []string {
	mu.RLock()
	defer mu.RUnlock()
	names := make([]string, 0, len(registry))
	for k := range registry {
		names = append(names, k)
	}
	return names
}
