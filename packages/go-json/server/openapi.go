package server

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/bitcode-framework/go-json/lang"
	"github.com/bitcode-framework/go-json/server/adapters"
)

// GenerateOpenAPISpec generates an OpenAPI 3.0 spec from a compiled program.
func GenerateOpenAPISpec(program *lang.CompiledProgram) map[string]any {
	cfg := program.AST.Server
	if cfg == nil {
		cfg = &lang.ServerConfig{}
	}
	MergeDefaults(cfg)

	flatRoutes := FlattenRoutes(program.AST.Routes, "", program.AST.Middleware)

	spec := map[string]any{
		"openapi": "3.0.3",
		"info": map[string]any{
			"title":   program.Name,
			"version": "1.0.0",
		},
		"servers": []map[string]any{
			{"url": fmt.Sprintf("http://localhost:%d", cfg.Port)},
		},
	}

	paths := make(map[string]any)
	tags := make(map[string]bool)

	for _, route := range flatRoutes {
		oaPath := convertToOpenAPIPath(route.FullPath)
		method := strings.ToLower(route.Method)

		operation := map[string]any{
			"operationId": route.Handler,
		}

		tag := extractTag(route.FullPath)
		if tag != "" {
			operation["tags"] = []string{tag}
			tags[tag] = true
		}

		if hasAuthMiddleware(route.Middleware) {
			operation["security"] = []map[string]any{
				{"bearerAuth": []string{}},
			}
		}

		if route.API != nil {
			applyAPIAnnotation(operation, route.API)
		} else {
			operation["responses"] = map[string]any{
				"200": map[string]any{
					"description": "Successful response",
				},
			}
		}

		if _, ok := paths[oaPath]; !ok {
			paths[oaPath] = map[string]any{}
		}
		paths[oaPath].(map[string]any)[method] = operation
	}

	spec["paths"] = paths

	securitySchemes := buildSecuritySchemes(cfg)
	if len(securitySchemes) > 0 {
		spec["components"] = map[string]any{
			"securitySchemes": securitySchemes,
		}
	}

	if len(tags) > 0 {
		tagList := make([]map[string]any, 0, len(tags))
		for t := range tags {
			tagList = append(tagList, map[string]any{"name": t})
		}
		spec["tags"] = tagList
	}

	return spec
}

// RegisterDocsEndpoints registers /docs and /docs/openapi.json endpoints.
func RegisterDocsEndpoints(adapter adapters.ServerAdapter, program *lang.CompiledProgram) {
	spec := GenerateOpenAPISpec(program)

	adapter.RegisterRoute(adapters.RouteRegistration{
		Method: "GET",
		Path:   "/docs/openapi.json",
		Handler: func(ctx *adapters.RequestContext) *adapters.Response {
			return &adapters.Response{
				Status:  200,
				Body:    spec,
				Headers: map[string]string{"Content-Type": "application/json"},
			}
		},
	})

	adapter.RegisterRoute(adapters.RouteRegistration{
		Method: "GET",
		Path:   "/docs",
		Handler: func(ctx *adapters.RequestContext) *adapters.Response {
			return &adapters.Response{
				Status: 200,
				Headers: map[string]string{
					"Content-Type": "text/html; charset=utf-8",
				},
				Body: json.RawMessage(`"` + escapeJSONString(swaggerUIHTML) + `"`),
			}
		},
	})
}

func convertToOpenAPIPath(path string) string {
	parts := strings.Split(path, "/")
	for i, part := range parts {
		if strings.HasPrefix(part, ":") {
			parts[i] = "{" + part[1:] + "}"
		}
	}
	return strings.Join(parts, "/")
}

func extractTag(path string) string {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) > 0 && parts[0] != "" {
		return parts[0]
	}
	return ""
}

func hasAuthMiddleware(middleware []string) bool {
	for _, m := range middleware {
		if m == "jwt" || m == "auth" || strings.HasPrefix(m, "auth:") {
			return true
		}
	}
	return false
}

func applyAPIAnnotation(operation map[string]any, api *lang.APIAnnotation) {
	if api.Summary != "" {
		operation["summary"] = api.Summary
	}
	if api.Description != "" {
		operation["description"] = api.Description
	}
	if len(api.Tags) > 0 {
		operation["tags"] = api.Tags
	}

	if api.Body != nil {
		reqBody := map[string]any{}
		if api.Body.Required {
			reqBody["required"] = true
		}
		if api.Body.Content != nil {
			properties := make(map[string]any)
			required := []string{}
			for name, field := range api.Body.Content {
				prop := map[string]any{}
				if field.Type != "" {
					prop["type"] = field.Type
				}
				if field.Description != "" {
					prop["description"] = field.Description
				}
				if field.Format != "" {
					prop["format"] = field.Format
				}
				if len(field.Enum) > 0 {
					prop["enum"] = field.Enum
				}
				if field.Default != nil {
					prop["default"] = field.Default
				}
				properties[name] = prop
				if field.Required {
					required = append(required, name)
				}
			}
			schema := map[string]any{
				"type":       "object",
				"properties": properties,
			}
			if len(required) > 0 {
				schema["required"] = required
			}
			reqBody["content"] = map[string]any{
				"application/json": map[string]any{
					"schema": schema,
				},
			}
		}
		operation["requestBody"] = reqBody
	}

	if len(api.Query) > 0 {
		params := make([]map[string]any, 0)
		for name, param := range api.Query {
			p := map[string]any{
				"name": name,
				"in":   "query",
			}
			if param.Description != "" {
				p["description"] = param.Description
			}
			if param.Type != "" {
				p["schema"] = map[string]any{"type": param.Type}
			}
			if param.Default != nil {
				if s, ok := p["schema"].(map[string]any); ok {
					s["default"] = param.Default
				}
			}
			params = append(params, p)
		}
		operation["parameters"] = params
	}

	if len(api.Responses) > 0 {
		responses := make(map[string]any)
		for code, resp := range api.Responses {
			r := map[string]any{}
			if resp.Description != "" {
				r["description"] = resp.Description
			}
			if resp.Content != nil {
				properties := make(map[string]any)
				for name, field := range resp.Content {
					prop := map[string]any{}
					if field.Type != "" {
						prop["type"] = field.Type
					}
					properties[name] = prop
				}
				r["content"] = map[string]any{
					"application/json": map[string]any{
						"schema": map[string]any{
							"type":       "object",
							"properties": properties,
						},
					},
				}
			}
			responses[code] = r
		}
		operation["responses"] = responses
	} else {
		operation["responses"] = map[string]any{
			"200": map[string]any{"description": "Successful response"},
		}
	}
}

func buildSecuritySchemes(cfg *lang.ServerConfig) map[string]any {
	schemes := make(map[string]any)

	if cfg.JWT != nil {
		schemes["bearerAuth"] = map[string]any{
			"type":         "http",
			"scheme":       "bearer",
			"bearerFormat": "JWT",
		}
	}

	if cfg.Auth != nil {
		for name, sc := range cfg.Auth.Strategies {
			switch sc.Type {
			case "bearer":
				schemes[name+"Auth"] = map[string]any{
					"type":         "http",
					"scheme":       "bearer",
					"bearerFormat": "JWT",
				}
			case "apikey":
				header := sc.Header
				if header == "" {
					header = "X-API-Key"
				}
				schemes[name+"Auth"] = map[string]any{
					"type": "apiKey",
					"in":   "header",
					"name": header,
				}
			case "basic":
				schemes[name+"Auth"] = map[string]any{
					"type":   "http",
					"scheme": "basic",
				}
			}
		}
	}

	return schemes
}

const swaggerUIHTML = `<!DOCTYPE html>
<html>
<head>
<title>API Documentation</title>
<link rel="stylesheet" href="https://unpkg.com/swagger-ui-dist@5/swagger-ui.css">
</head>
<body>
<div id="swagger-ui"></div>
<script src="https://unpkg.com/swagger-ui-dist@5/swagger-ui-bundle.js"></script>
<script>
SwaggerUIBundle({url: '/docs/openapi.json', dom_id: '#swagger-ui'});
</script>
</body>
</html>`
