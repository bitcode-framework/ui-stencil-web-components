package server

import (
	"fmt"
	"strings"

	"github.com/bitcode-framework/go-json/lang"
)

// FlatRoute is a fully resolved route with absolute path and merged middleware.
type FlatRoute struct {
	Method     string
	FullPath   string
	Handler    string
	Middleware []string
	Render     string
	API        *lang.APIAnnotation
}

// FlattenRoutes recursively flattens route groups into a flat list.
func FlattenRoutes(routes []lang.RouteConfig, parentPrefix string, parentMiddleware []string) []FlatRoute {
	var result []FlatRoute

	for _, r := range routes {
		if r.Prefix != "" || len(r.Routes) > 0 {
			prefix := joinPath(parentPrefix, r.Prefix)
			merged := mergeMiddleware(parentMiddleware, r.Middleware)
			result = append(result, FlattenRoutes(r.Routes, prefix, merged)...)
			continue
		}

		fullPath := joinPath(parentPrefix, r.Path)
		merged := mergeMiddleware(parentMiddleware, r.Middleware)

		result = append(result, FlatRoute{
			Method:     strings.ToUpper(r.Method),
			FullPath:   fullPath,
			Handler:    r.Handler,
			Middleware: merged,
			Render:     r.Render,
			API:        r.API,
		})
	}

	return result
}

// ValidateRoutes checks for common route configuration errors.
func ValidateRoutes(routes []FlatRoute, functions map[string]*lang.CompiledFunc) []error {
	var errs []error
	seen := make(map[string]bool)

	for _, r := range routes {
		if r.Method == "" {
			errs = append(errs, fmt.Errorf("route %q: method is required", r.FullPath))
		}
		if r.FullPath == "" {
			errs = append(errs, fmt.Errorf("route with handler %q: path is required", r.Handler))
		}
		if r.Handler == "" {
			errs = append(errs, fmt.Errorf("route %s %s: handler is required", r.Method, r.FullPath))
			continue
		}

		if functions != nil {
			if _, ok := functions[r.Handler]; !ok {
				errs = append(errs, fmt.Errorf("route %s %s: handler function %q not found", r.Method, r.FullPath, r.Handler))
			}
		}

		key := r.Method + " " + r.FullPath
		if seen[key] {
			errs = append(errs, fmt.Errorf("duplicate route: %s", key))
		}
		seen[key] = true
	}

	return errs
}

func joinPath(prefix, path string) string {
	prefix = strings.TrimRight(prefix, "/")
	if path == "" {
		return prefix
	}
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	return prefix + path
}

func mergeMiddleware(parent, child []string) []string {
	if len(parent) == 0 && len(child) == 0 {
		return nil
	}
	result := make([]string, 0, len(parent)+len(child))
	result = append(result, parent...)
	result = append(result, child...)
	return result
}
