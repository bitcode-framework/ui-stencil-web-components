package server

import (
	"fmt"
	"log"

	"github.com/bitcode-framework/go-json/lang"
	"github.com/bitcode-framework/go-json/runtime"
	"github.com/bitcode-framework/go-json/server/adapters"
)

// BuildHandler creates an adapters.HandlerFunc that bridges HTTP requests to go-json function execution.
func BuildHandler(rt *runtime.Runtime, compiled *lang.CompiledProgram, route FlatRoute) adapters.HandlerFunc {
	return func(ctx *adapters.RequestContext) *adapters.Response {
		requestMap := BuildRequestMap(ctx)

		args := map[string]any{
			"request": requestMap,
		}

		result, err := rt.ExecuteFunction(compiled, route.Handler, args)
		if err != nil {
			log.Printf("[go-json] handler %q error: %v", route.Handler, err)
			return &adapters.Response{
				Status: 500,
				Body: map[string]any{
					"error": map[string]any{
						"code":    "HANDLER_ERROR",
						"message": err.Error(),
					},
				},
			}
		}

		return ConvertToResponse(result, route)
	}
}

// BuildRequestMap converts a RequestContext into the go-json request object shape.
func BuildRequestMap(ctx *adapters.RequestContext) map[string]any {
	store := ctx.Store
	if store == nil {
		store = map[string]any{}
	}
	return map[string]any{
		"method":  ctx.Method,
		"path":    ctx.Path,
		"url":     ctx.URL,
		"params":  mapOrEmpty(ctx.Params),
		"query":   mapOrEmpty(ctx.Query),
		"headers": mapOrEmpty(ctx.Headers),
		"body":    ctx.Body,
		"cookies": mapOrEmpty(ctx.Cookies),
		"ip":      ctx.IP,
		"user":    ctx.User,
		"store":   store,
	}
}

// ConvertToResponse converts a go-json function return value to an HTTP Response.
func ConvertToResponse(result any, route FlatRoute) *adapters.Response {
	if result == nil {
		return &adapters.Response{Status: 204}
	}

	resultMap, ok := result.(map[string]any)
	if !ok {
		return &adapters.Response{
			Status: 200,
			Body:   result,
		}
	}

	resp := &adapters.Response{}

	if status, ok := extractInt(resultMap, "status"); ok {
		resp.Status = status
	}

	if redirect, ok := resultMap["redirect"].(string); ok {
		resp.Redirect = redirect
		if resp.Status == 0 {
			resp.Status = 302
		}
		return resp
	}

	if data, ok := resultMap["data"].(map[string]any); ok && route.Render != "" {
		resp.Template = route.Render
		resp.Data = data
		if resp.Status == 0 {
			resp.Status = 200
		}
		return resp
	}

	if body, ok := resultMap["body"]; ok {
		resp.Body = body
	}

	if headers, ok := resultMap["headers"].(map[string]any); ok {
		resp.Headers = make(map[string]string)
		for k, v := range headers {
			if s, ok := v.(string); ok {
				resp.Headers[k] = s
			} else {
				resp.Headers[k] = fmt.Sprintf("%v", v)
			}
		}
	}

	if cookies, ok := resultMap["cookies"]; ok {
		resp.Cookies = parseCookies(cookies)
	}

	if resp.Status == 0 {
		if resp.Body != nil {
			resp.Status = 200
		} else {
			resp.Status = 204
		}
	}

	return resp
}

func parseCookies(raw any) []adapters.CookieConfig {
	switch v := raw.(type) {
	case map[string]any:
		return []adapters.CookieConfig{parseSingleCookie(v)}
	case []any:
		cookies := make([]adapters.CookieConfig, 0, len(v))
		for _, item := range v {
			if m, ok := item.(map[string]any); ok {
				cookies = append(cookies, parseSingleCookie(m))
			}
		}
		return cookies
	}
	return nil
}

func parseSingleCookie(m map[string]any) adapters.CookieConfig {
	c := adapters.CookieConfig{}
	if v, ok := m["name"].(string); ok {
		c.Name = v
	}
	if v, ok := m["value"].(string); ok {
		c.Value = v
	}
	if v, ok := extractInt(m, "max_age"); ok {
		c.MaxAge = v
	}
	if v, ok := m["path"].(string); ok {
		c.Path = v
	}
	if v, ok := m["domain"].(string); ok {
		c.Domain = v
	}
	if v, ok := m["secure"].(bool); ok {
		c.Secure = v
	}
	if v, ok := m["http_only"].(bool); ok {
		c.HTTPOnly = v
	}
	return c
}

func extractInt(m map[string]any, key string) (int, bool) {
	v, ok := m[key]
	if !ok {
		return 0, false
	}
	switch n := v.(type) {
	case float64:
		return int(n), true
	case int:
		return n, true
	case int64:
		return int(n), true
	}
	return 0, false
}

func mapOrEmpty(m map[string]string) map[string]any {
	if m == nil {
		return map[string]any{}
	}
	result := make(map[string]any, len(m))
	for k, v := range m {
		result[k] = v
	}
	return result
}
