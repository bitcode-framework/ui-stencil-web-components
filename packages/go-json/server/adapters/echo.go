//go:build echo

package adapters

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/labstack/echo/v4"
)

func init() {
	Register("echo", NewEchoAdapter)
}

type EchoAdapter struct {
	e            *echo.Echo
	errorHandler func(err error, ctx *RequestContext) *Response
}

func NewEchoAdapter() ServerAdapter {
	e := echo.New()
	e.HideBanner = true
	return &EchoAdapter{e: e}
}

func (a *EchoAdapter) RegisterRoute(reg RouteRegistration) {
	handler := a.wrapHandler(reg.Handler, reg.Middleware)
	a.e.Add(strings.ToUpper(reg.Method), reg.Path, handler)
}

func (a *EchoAdapter) RegisterGroup(prefix string, middleware []MiddlewareFunc, routes []RouteRegistration) {
	g := a.e.Group(prefix)
	for _, route := range routes {
		merged := append(middleware, route.Middleware...)
		handler := a.wrapHandler(route.Handler, merged)
		g.Add(strings.ToUpper(route.Method), route.Path, handler)
	}
}

func (a *EchoAdapter) RegisterStatic(prefix, dir string) {
	a.e.Static(prefix, dir)
}

func (a *EchoAdapter) SetNotFoundHandler(handler HandlerFunc) {
	a.e.HTTPErrorHandler = func(err error, c echo.Context) {
		if he, ok := err.(*echo.HTTPError); ok && he.Code == http.StatusNotFound {
			reqCtx := echoToRequestContext(c)
			resp := handler(reqCtx)
			writeEchoResponse(c, resp)
			return
		}
		c.JSON(http.StatusInternalServerError, map[string]any{"error": err.Error()})
	}
}

func (a *EchoAdapter) SetErrorHandler(handler func(err error, ctx *RequestContext) *Response) {
	a.errorHandler = handler
}

func (a *EchoAdapter) Listen(addr string) error {
	return a.e.Start(addr)
}

func (a *EchoAdapter) Shutdown(ctx context.Context) error {
	return a.e.Shutdown(ctx)
}

func (a *EchoAdapter) wrapHandler(handler HandlerFunc, middleware []MiddlewareFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		reqCtx := echoToRequestContext(c)
		var resp *Response
		if len(middleware) == 0 {
			resp = handler(reqCtx)
		} else {
			resp = executeMiddlewareChain(reqCtx, middleware, handler)
		}
		if resp == nil {
			return c.NoContent(http.StatusNoContent)
		}
		return writeEchoResponse(c, resp)
	}
}

func echoToRequestContext(c echo.Context) *RequestContext {
	params := make(map[string]string)
	for _, name := range c.ParamNames() {
		params[name] = c.Param(name)
	}
	query := make(map[string]string)
	for k, v := range c.QueryParams() {
		if len(v) > 0 {
			query[k] = v[0]
		}
	}
	headers := make(map[string]string)
	for k, v := range c.Request().Header {
		if len(v) > 0 {
			headers[k] = v[0]
		}
	}
	cookies := make(map[string]string)
	for _, ck := range c.Cookies() {
		cookies[ck.Name] = ck.Value
	}

	var body any
	ct := c.Request().Header.Get("Content-Type")
	if strings.HasPrefix(ct, "application/json") {
		var parsed any
		if err := json.NewDecoder(c.Request().Body).Decode(&parsed); err == nil {
			body = parsed
		}
	}

	return &RequestContext{
		Method:  c.Request().Method,
		Path:    c.Path(),
		URL:     c.Request().URL.String(),
		Params:  params,
		Query:   query,
		Headers: headers,
		Body:    body,
		Cookies: cookies,
		IP:      c.RealIP(),
		Store:   make(map[string]any),
	}
}

func writeEchoResponse(c echo.Context, resp *Response) error {
	if resp.Redirect != "" {
		status := resp.Status
		if status == 0 {
			status = http.StatusFound
		}
		return c.Redirect(status, resp.Redirect)
	}
	for k, v := range resp.Headers {
		c.Response().Header().Set(k, v)
	}
	for _, ck := range resp.Cookies {
		c.SetCookie(&http.Cookie{
			Name: ck.Name, Value: ck.Value, MaxAge: ck.MaxAge,
			Path: ck.Path, Domain: ck.Domain, Secure: ck.Secure, HttpOnly: ck.HTTPOnly,
		})
	}
	status := resp.Status
	if status == 0 {
		status = http.StatusOK
	}
	if resp.Body != nil {
		return c.JSON(status, resp.Body)
	}
	return c.NoContent(status)
}
