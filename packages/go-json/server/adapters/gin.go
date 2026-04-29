//go:build gin

package adapters

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

func init() {
	Register("gin", NewGinAdapter)
}

type GinAdapter struct {
	engine       *gin.Engine
	server       *http.Server
	errorHandler func(err error, ctx *RequestContext) *Response
}

func NewGinAdapter() ServerAdapter {
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	return &GinAdapter{engine: r}
}

func (a *GinAdapter) RegisterRoute(reg RouteRegistration) {
	handler := a.wrapHandler(reg.Handler, reg.Middleware)
	a.engine.Handle(strings.ToUpper(reg.Method), reg.Path, handler)
}

func (a *GinAdapter) RegisterGroup(prefix string, middleware []MiddlewareFunc, routes []RouteRegistration) {
	g := a.engine.Group(prefix)
	for _, route := range routes {
		merged := append(middleware, route.Middleware...)
		handler := a.wrapHandler(route.Handler, merged)
		g.Handle(strings.ToUpper(route.Method), route.Path, handler)
	}
}

func (a *GinAdapter) RegisterStatic(prefix, dir string) {
	a.engine.Static(prefix, dir)
}

func (a *GinAdapter) SetNotFoundHandler(handler HandlerFunc) {
	a.engine.NoRoute(func(c *gin.Context) {
		reqCtx := ginToRequestContext(c)
		resp := handler(reqCtx)
		writeGinResponse(c, resp)
	})
}

func (a *GinAdapter) SetErrorHandler(handler func(err error, ctx *RequestContext) *Response) {
	a.errorHandler = handler
}

func (a *GinAdapter) Listen(addr string) error {
	a.server = &http.Server{Addr: addr, Handler: a.engine}
	return a.server.ListenAndServe()
}

func (a *GinAdapter) Shutdown(ctx context.Context) error {
	if a.server == nil {
		return nil
	}
	return a.server.Shutdown(ctx)
}

func (a *GinAdapter) wrapHandler(handler HandlerFunc, middleware []MiddlewareFunc) gin.HandlerFunc {
	return func(c *gin.Context) {
		reqCtx := ginToRequestContext(c)
		var resp *Response
		if len(middleware) == 0 {
			resp = handler(reqCtx)
		} else {
			resp = executeMiddlewareChain(reqCtx, middleware, handler)
		}
		if resp == nil {
			c.Status(http.StatusNoContent)
			return
		}
		writeGinResponse(c, resp)
	}
}

func ginToRequestContext(c *gin.Context) *RequestContext {
	params := make(map[string]string)
	for _, p := range c.Params {
		params[p.Key] = p.Value
	}
	query := make(map[string]string)
	for k, v := range c.Request.URL.Query() {
		if len(v) > 0 {
			query[k] = v[0]
		}
	}
	headers := make(map[string]string)
	for k, v := range c.Request.Header {
		if len(v) > 0 {
			headers[k] = v[0]
		}
	}
	cookies := make(map[string]string)
	for _, ck := range c.Request.Cookies() {
		cookies[ck.Name] = ck.Value
	}

	var body any
	ct := c.GetHeader("Content-Type")
	if strings.HasPrefix(ct, "application/json") {
		var parsed any
		if err := json.NewDecoder(c.Request.Body).Decode(&parsed); err == nil {
			body = parsed
		}
	}

	return &RequestContext{
		Method:  c.Request.Method,
		Path:    c.FullPath(),
		URL:     c.Request.URL.String(),
		Params:  params,
		Query:   query,
		Headers: headers,
		Body:    body,
		Cookies: cookies,
		IP:      c.ClientIP(),
		Store:   make(map[string]any),
	}
}

func writeGinResponse(c *gin.Context, resp *Response) {
	if resp.Redirect != "" {
		status := resp.Status
		if status == 0 {
			status = http.StatusFound
		}
		c.Redirect(status, resp.Redirect)
		return
	}
	for k, v := range resp.Headers {
		c.Header(k, v)
	}
	for _, ck := range resp.Cookies {
		c.SetCookie(ck.Name, ck.Value, ck.MaxAge, ck.Path, ck.Domain, ck.Secure, ck.HTTPOnly)
	}
	status := resp.Status
	if status == 0 {
		status = http.StatusOK
	}
	if resp.Body != nil {
		c.JSON(status, resp.Body)
	} else {
		c.Status(status)
	}
}
