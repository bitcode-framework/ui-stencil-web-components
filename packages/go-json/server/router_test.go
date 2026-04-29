package server

import (
	"testing"

	"github.com/bitcode-framework/go-json/lang"
)

func TestFlattenRoutes_Basic(t *testing.T) {
	routes := []lang.RouteConfig{
		{Method: "GET", Path: "/users", Handler: "listUsers"},
		{Method: "POST", Path: "/users", Handler: "createUser"},
	}
	flat := FlattenRoutes(routes, "", nil)
	if len(flat) != 2 {
		t.Fatalf("expected 2 routes, got %d", len(flat))
	}
	if flat[0].FullPath != "/users" || flat[0].Method != "GET" {
		t.Errorf("unexpected route: %+v", flat[0])
	}
}

func TestFlattenRoutes_Groups(t *testing.T) {
	routes := []lang.RouteConfig{
		{
			Prefix:     "/api",
			Middleware: []string{"auth"},
			Routes: []lang.RouteConfig{
				{Method: "GET", Path: "/users", Handler: "listUsers"},
				{Method: "POST", Path: "/users", Handler: "createUser"},
			},
		},
	}
	flat := FlattenRoutes(routes, "", nil)
	if len(flat) != 2 {
		t.Fatalf("expected 2 routes, got %d", len(flat))
	}
	if flat[0].FullPath != "/api/users" {
		t.Errorf("expected /api/users, got %s", flat[0].FullPath)
	}
	if len(flat[0].Middleware) != 1 || flat[0].Middleware[0] != "auth" {
		t.Errorf("expected [auth] middleware, got %v", flat[0].Middleware)
	}
}

func TestFlattenRoutes_NestedGroups(t *testing.T) {
	routes := []lang.RouteConfig{
		{
			Prefix: "/api",
			Routes: []lang.RouteConfig{
				{
					Prefix:     "/v1",
					Middleware: []string{"logger"},
					Routes: []lang.RouteConfig{
						{Method: "GET", Path: "/items", Handler: "listItems"},
					},
				},
			},
		},
	}
	flat := FlattenRoutes(routes, "", nil)
	if len(flat) != 1 {
		t.Fatalf("expected 1 route, got %d", len(flat))
	}
	if flat[0].FullPath != "/api/v1/items" {
		t.Errorf("expected /api/v1/items, got %s", flat[0].FullPath)
	}
}

func TestFlattenRoutes_MiddlewareMerging(t *testing.T) {
	routes := []lang.RouteConfig{
		{
			Prefix:     "/api",
			Middleware: []string{"cors"},
			Routes: []lang.RouteConfig{
				{Method: "GET", Path: "/public", Handler: "pub"},
				{Method: "POST", Path: "/private", Handler: "priv", Middleware: []string{"jwt"}},
			},
		},
	}
	flat := FlattenRoutes(routes, "", []string{"logger"})
	if len(flat) != 2 {
		t.Fatalf("expected 2 routes, got %d", len(flat))
	}
	if len(flat[0].Middleware) != 2 {
		t.Errorf("expected 2 middleware for public, got %v", flat[0].Middleware)
	}
	if len(flat[1].Middleware) != 3 {
		t.Errorf("expected 3 middleware for private, got %v", flat[1].Middleware)
	}
}

func TestValidateRoutes_DuplicateDetection(t *testing.T) {
	routes := []FlatRoute{
		{Method: "GET", FullPath: "/users", Handler: "a"},
		{Method: "GET", FullPath: "/users", Handler: "b"},
	}
	errs := ValidateRoutes(routes, nil)
	found := false
	for _, e := range errs {
		if e.Error() == "duplicate route: GET /users" {
			found = true
		}
	}
	if !found {
		t.Error("expected duplicate route error")
	}
}

func TestValidateRoutes_MissingHandler(t *testing.T) {
	routes := []FlatRoute{
		{Method: "GET", FullPath: "/users", Handler: ""},
	}
	errs := ValidateRoutes(routes, nil)
	if len(errs) == 0 {
		t.Error("expected error for missing handler")
	}
}
