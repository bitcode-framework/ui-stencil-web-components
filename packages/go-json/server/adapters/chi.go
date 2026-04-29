//go:build chi

package adapters

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
)

func init() {
	Register("chi", NewChiAdapter)
}

type ChiAdapter struct {
	router       *chi.Mux
	server       *http.Server
	errorHandler func(err error, ctx *RequestContext) *Response
}

func NewChiAdapter() ServerAdapter {
	return &ChiAdapter{router: chi.NewRouter()}
}

func (a *ChiAdapter) RegisterRoute(reg RouteRegistration) {
	handler := a.wrapHandler(reg.Handler, reg.Middleware)
	a.router.Method(strings.ToUpper(reg.Method), reg.Path, handler)
}

func (a *ChiAdapter) RegisterGroup(prefix string, middleware []MiddlewareFunc, routes []RouteRegistration) {
	a.router.Route(prefix, func(r chi.Router) {
		for _, route := range routes {
			merged := append(middleware, route.Middleware...)
			handler := a.wrapHandler(route.Handler, merged)
			r.Method(strings.ToUpper(route.Method), route.Path, handler)
		}
	})
}

func (a *ChiAdapter) RegisterStatic(prefix, dir string) {
	fs := http.FileServer(http.Dir(dir))
	a.router.Handle(prefix+"/*", http.StripPrefix(prefix, fs))
}

func (a *ChiAdapter) SetNotFoundHandler(handler HandlerFunc) {
	a.router.NotFound(func(w http.ResponseWriter, r *http.Request) {
		reqCtx := chiToRequestContext(r)
		resp := handler(reqCtx)
		writeHTTPResponse(w, resp)
	})
}

func (a *ChiAdapter) SetErrorHandler(handler func(err error, ctx *RequestContext) *Response) {
	a.errorHandler = handler
}

func (a *ChiAdapter) Listen(addr string) error {
	a.server = &http.Server{Addr: addr, Handler: a.router}
	return a.server.ListenAndServe()
}

func (a *ChiAdapter) Shutdown(ctx context.Context) error {
	if a.server == nil {
		return nil
	}
	return a.server.Shutdown(ctx)
}

func (a *ChiAdapter) wrapHandler(handler HandlerFunc, middleware []MiddlewareFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		reqCtx := chiToRequestContext(r)
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

func chiToRequestContext(r *http.Request) *RequestContext {
	params := make(map[string]string)
	rctx := chi.RouteContext(r.Context())
	if rctx != nil {
		for i, key := range rctx.URLParams.Keys {
			if i < len(rctx.URLParams.Values) {
				params[key] = rctx.URLParams.Values[i]
			}
		}
	}

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

	var body any
	ct := r.Header.Get("Content-Type")
	if strings.HasPrefix(ct, "application/json") {
		var parsed any
		if err := json.NewDecoder(r.Body).Decode(&parsed); err == nil {
			body = parsed
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
