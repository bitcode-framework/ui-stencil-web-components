package adapters

import (
	"context"
	"encoding/json"
	"io"
	"mime/multipart"
	"os"
	"path/filepath"
	"strings"

	"github.com/gofiber/fiber/v2"
)

func init() {
	Register("fiber", NewFiberAdapter)
}

// FiberAdapter implements ServerAdapter using gofiber/fiber/v2.
type FiberAdapter struct {
	app            *fiber.App
	notFoundHandler HandlerFunc
	errorHandler   func(err error, ctx *RequestContext) *Response
}

// NewFiberAdapter creates a new FiberAdapter.
func NewFiberAdapter() ServerAdapter {
	app := fiber.New(fiber.Config{
		DisableStartupMessage: true,
	})
	return &FiberAdapter{app: app}
}

func (a *FiberAdapter) RegisterRoute(reg RouteRegistration) {
	handler := a.wrapHandler(reg.Handler, reg.Middleware)
	a.app.Add(strings.ToUpper(reg.Method), reg.Path, handler)
}

func (a *FiberAdapter) RegisterGroup(prefix string, middleware []MiddlewareFunc, routes []RouteRegistration) {
	group := a.app.Group(prefix)
	for _, route := range routes {
		merged := append(middleware, route.Middleware...)
		handler := a.wrapHandler(route.Handler, merged)
		group.Add(strings.ToUpper(route.Method), route.Path, handler)
	}
}

func (a *FiberAdapter) RegisterStatic(prefix, dir string) {
	a.app.Static(prefix, dir)
}

func (a *FiberAdapter) SetNotFoundHandler(handler HandlerFunc) {
	a.notFoundHandler = handler
	a.app.Use(func(c *fiber.Ctx) error {
		reqCtx := fiberToRequestContext(c)
		resp := handler(reqCtx)
		return sendResponse(c, resp)
	})
}

func (a *FiberAdapter) SetErrorHandler(handler func(err error, ctx *RequestContext) *Response) {
	a.errorHandler = handler
}

func (a *FiberAdapter) Listen(addr string) error {
	return a.app.Listen(addr)
}

func (a *FiberAdapter) Shutdown(ctx context.Context) error {
	return a.app.ShutdownWithContext(ctx)
}

// App returns the underlying fiber.App for advanced configuration.
func (a *FiberAdapter) App() *fiber.App {
	return a.app
}

func (a *FiberAdapter) wrapHandler(handler HandlerFunc, middleware []MiddlewareFunc) fiber.Handler {
	return func(c *fiber.Ctx) error {
		reqCtx := fiberToRequestContext(c)

		var resp *Response
		if len(middleware) == 0 {
			resp = handler(reqCtx)
		} else {
			resp = executeMiddlewareChain(reqCtx, middleware, handler)
		}

		if resp == nil {
			return c.SendStatus(fiber.StatusNoContent)
		}
		return sendResponse(c, resp)
	}
}

func fiberToRequestContext(c *fiber.Ctx) *RequestContext {
	params := make(map[string]string)
	for _, key := range c.Route().Params {
		params[key] = c.Params(key)
	}

	query := make(map[string]string)
	c.Context().QueryArgs().VisitAll(func(key, value []byte) {
		query[string(key)] = string(value)
	})

	headers := make(map[string]string)
	c.Request().Header.VisitAll(func(key, value []byte) {
		headers[string(key)] = string(value)
	})

	cookies := make(map[string]string)
	c.Request().Header.VisitAllCookie(func(key, value []byte) {
		cookies[string(key)] = string(value)
	})

	var body any
	contentType := string(c.Request().Header.ContentType())
	if strings.HasPrefix(contentType, "application/json") {
		var parsed any
		if err := json.Unmarshal(c.Body(), &parsed); err == nil {
			body = parsed
		}
	} else if strings.HasPrefix(contentType, "application/x-www-form-urlencoded") {
		formData := make(map[string]any)
		c.Request().PostArgs().VisitAll(func(key, value []byte) {
			formData[string(key)] = string(value)
		})
		body = formData
	} else if strings.HasPrefix(contentType, "text/") {
		body = string(c.Body())
	} else if strings.HasPrefix(contentType, "multipart/form-data") {
		formData := make(map[string]any)
		if form, err := c.MultipartForm(); err == nil {
			for key, values := range form.Value {
				if len(values) == 1 {
					formData[key] = values[0]
				} else {
					formData[key] = values
				}
			}
			for key, files := range form.File {
				if len(files) == 1 {
					f := files[0]
					tempPath := saveTempFile(f)
					formData[key] = map[string]any{
						"_file":        true,
						"filename":     f.Filename,
						"size":         f.Size,
						"content_type": f.Header.Get("Content-Type"),
						"temp_path":    tempPath,
					}
				} else {
					fileList := make([]map[string]any, len(files))
					for i, f := range files {
						tempPath := saveTempFile(f)
						fileList[i] = map[string]any{
							"_file":        true,
							"filename":     f.Filename,
							"size":         f.Size,
							"content_type": f.Header.Get("Content-Type"),
							"temp_path":    tempPath,
						}
					}
					formData[key] = fileList
				}
			}
		}
		body = formData
	}

	return &RequestContext{
		Method:  c.Method(),
		Path:    c.Path(),
		URL:     c.OriginalURL(),
		Params:  params,
		Query:   query,
		Headers: headers,
		Body:    body,
		Cookies: cookies,
		IP:      c.IP(),
		Store:   make(map[string]any),
	}
}

func sendResponse(c *fiber.Ctx, resp *Response) error {
	if resp.Redirect != "" {
		status := resp.Status
		if status == 0 {
			status = fiber.StatusFound
		}
		return c.Redirect(resp.Redirect, status)
	}

	for key, value := range resp.Headers {
		c.Set(key, value)
	}

	for _, cookie := range resp.Cookies {
		c.Cookie(&fiber.Cookie{
			Name:     cookie.Name,
			Value:    cookie.Value,
			MaxAge:   cookie.MaxAge,
			Path:     cookie.Path,
			Domain:   cookie.Domain,
			Secure:   cookie.Secure,
			HTTPOnly: cookie.HTTPOnly,
		})
	}

	status := resp.Status
	if status == 0 {
		status = fiber.StatusOK
	}
	c.Status(status)

	if resp.Body != nil {
		return c.JSON(resp.Body)
	}

	return nil
}

func executeMiddlewareChain(ctx *RequestContext, middleware []MiddlewareFunc, handler HandlerFunc) *Response {
	index := 0
	var next func() *Response
	next = func() *Response {
		if index >= len(middleware) {
			return handler(ctx)
		}
		mw := middleware[index]
		index++
		return mw(ctx, next)
	}
	return next()
}

func saveTempFile(fh *multipart.FileHeader) string {
	src, err := fh.Open()
	if err != nil {
		return ""
	}
	defer src.Close()

	tmpDir := os.TempDir()
	tmpFile, err := os.CreateTemp(tmpDir, "gojson-upload-*"+filepath.Ext(fh.Filename))
	if err != nil {
		return ""
	}
	defer tmpFile.Close()

	if _, err := io.Copy(tmpFile, src); err != nil {
		os.Remove(tmpFile.Name())
		return ""
	}

	return tmpFile.Name()
}
