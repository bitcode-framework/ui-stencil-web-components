package adapters

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"strings"
	"sync"
)

func init() {
	Register("nethttp", NewNetHTTPAdapter)
}

// NetHTTPAdapter implements ServerAdapter using Go's net/http stdlib.
type NetHTTPAdapter struct {
	mux             *http.ServeMux
	server          *http.Server
	notFoundHandler HandlerFunc
	errorHandler    func(err error, ctx *RequestContext) *Response
	mu              sync.Mutex
}

// NewNetHTTPAdapter creates a new net/http adapter.
func NewNetHTTPAdapter() ServerAdapter {
	return &NetHTTPAdapter{
		mux: http.NewServeMux(),
	}
}

func (a *NetHTTPAdapter) RegisterRoute(reg RouteRegistration) {
	pattern := fmt.Sprintf("%s %s", strings.ToUpper(reg.Method), reg.Path)
	a.mux.HandleFunc(pattern, a.wrapHandler(reg.Handler, reg.Middleware))
}

func (a *NetHTTPAdapter) RegisterGroup(prefix string, middleware []MiddlewareFunc, routes []RouteRegistration) {
	for _, route := range routes {
		fullPath := strings.TrimRight(prefix, "/") + route.Path
		merged := append(middleware, route.Middleware...)
		pattern := fmt.Sprintf("%s %s", strings.ToUpper(route.Method), fullPath)
		a.mux.HandleFunc(pattern, a.wrapHandler(route.Handler, merged))
	}
}

func (a *NetHTTPAdapter) RegisterStatic(prefix, dir string) {
	fs := http.FileServer(http.Dir(dir))
	a.mux.Handle(prefix+"/", http.StripPrefix(prefix, fs))
}

func (a *NetHTTPAdapter) SetNotFoundHandler(handler HandlerFunc) {
	a.notFoundHandler = handler
}

func (a *NetHTTPAdapter) SetErrorHandler(handler func(err error, ctx *RequestContext) *Response) {
	a.errorHandler = handler
}

func (a *NetHTTPAdapter) Listen(addr string) error {
	a.server = &http.Server{
		Addr:    addr,
		Handler: a.mux,
	}
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}
	return a.server.Serve(ln)
}

func (a *NetHTTPAdapter) Shutdown(ctx context.Context) error {
	if a.server == nil {
		return nil
	}
	return a.server.Shutdown(ctx)
}

func (a *NetHTTPAdapter) wrapHandler(handler HandlerFunc, middleware []MiddlewareFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		reqCtx := netHTTPToRequestContext(r)

		var resp *Response
		if len(middleware) == 0 {
			resp = handler(reqCtx)
		} else {
			resp = executeMiddlewareChain(reqCtx, middleware, handler)
		}

		if resp == nil {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		writeHTTPResponse(w, resp)
	}
}

func netHTTPToRequestContext(r *http.Request) *RequestContext {
	query := make(map[string]string)
	for k, v := range r.URL.Query() {
		if len(v) > 0 {
			query[k] = v[0]
		}
	}

	headers := make(map[string]string)
	for k, v := range r.Header {
		if len(v) > 0 {
			headers[k] = v[0]
		}
	}

	cookies := make(map[string]string)
	for _, c := range r.Cookies() {
		cookies[c.Name] = c.Value
	}

	// net/http path params require Go 1.22+ {name} patterns; extracted at registration time
	params := make(map[string]string)

	var body any
	contentType := r.Header.Get("Content-Type")
	if strings.HasPrefix(contentType, "application/json") {
		var parsed any
		if err := json.NewDecoder(r.Body).Decode(&parsed); err == nil {
			body = parsed
		}
	} else if strings.HasPrefix(contentType, "application/x-www-form-urlencoded") {
		if err := r.ParseForm(); err == nil {
			formData := make(map[string]any)
			for k, v := range r.PostForm {
				if len(v) == 1 {
					formData[k] = v[0]
				} else {
					formData[k] = v
				}
			}
			body = formData
		}
	}

	ip := r.RemoteAddr
	if fwd := r.Header.Get("X-Forwarded-For"); fwd != "" {
		ip = strings.Split(fwd, ",")[0]
	}

	return &RequestContext{
		Method:  r.Method,
		Path:    r.URL.Path,
		URL:     r.URL.String(),
		Params:  params,
		Query:   query,
		Headers: headers,
		Body:    body,
		Cookies: cookies,
		IP:      ip,
		Store:   make(map[string]any),
	}
}

func writeHTTPResponse(w http.ResponseWriter, resp *Response) {
	if resp.Redirect != "" {
		status := resp.Status
		if status == 0 {
			status = http.StatusFound
		}
		w.Header().Set("Location", resp.Redirect)
		w.WriteHeader(status)
		return
	}

	for key, value := range resp.Headers {
		w.Header().Set(key, value)
	}

	for _, cookie := range resp.Cookies {
		http.SetCookie(w, &http.Cookie{
			Name:     cookie.Name,
			Value:    cookie.Value,
			MaxAge:   cookie.MaxAge,
			Path:     cookie.Path,
			Domain:   cookie.Domain,
			Secure:   cookie.Secure,
			HttpOnly: cookie.HTTPOnly,
		})
	}

	status := resp.Status
	if status == 0 {
		status = http.StatusOK
	}

	if resp.Body != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		json.NewEncoder(w).Encode(resp.Body)
	} else {
		w.WriteHeader(status)
	}
}
