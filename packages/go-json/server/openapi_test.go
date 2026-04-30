package server

import (
	"testing"

	"github.com/bitcode-framework/go-json/lang"
)

func TestConvertToOpenAPIPath(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"/users/:id", "/users/{id}"},
		{"/users/:id/posts/:postId", "/users/{id}/posts/{postId}"},
		{"/users", "/users"},
		{"/:id", "/{id}"},
	}

	for _, tt := range tests {
		result := convertToOpenAPIPath(tt.input)
		if result != tt.expected {
			t.Errorf("convertToOpenAPIPath(%q) = %q, want %q", tt.input, result, tt.expected)
		}
	}
}

func TestExtractTag(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"/api/users", "api"},
		{"/users", "users"},
		{"/", ""},
	}

	for _, tt := range tests {
		result := extractTag(tt.input)
		if result != tt.expected {
			t.Errorf("extractTag(%q) = %q, want %q", tt.input, result, tt.expected)
		}
	}
}

func TestHasAuthMiddleware(t *testing.T) {
	tests := []struct {
		middleware []string
		expected   bool
	}{
		{[]string{"jwt"}, true},
		{[]string{"auth"}, true},
		{[]string{"auth:apikey"}, true},
		{[]string{"logger"}, false},
		{[]string{"logger", "jwt"}, true},
		{[]string{}, false},
		{nil, false},
	}

	for _, tt := range tests {
		result := hasAuthMiddleware(tt.middleware)
		if result != tt.expected {
			t.Errorf("hasAuthMiddleware(%v) = %v, want %v", tt.middleware, result, tt.expected)
		}
	}
}

func TestBuildSecuritySchemes_JWT(t *testing.T) {
	cfg := &lang.ServerConfig{
		JWT: &lang.JWTConfig{
			SecretEnv: "JWT_SECRET",
			Algorithm: "HS256",
		},
	}

	schemes := buildSecuritySchemes(cfg)
	bearer, ok := schemes["bearerAuth"]
	if !ok {
		t.Fatal("expected bearerAuth scheme")
	}

	m := bearer.(map[string]any)
	if m["type"] != "http" {
		t.Errorf("type = %v, want http", m["type"])
	}
	if m["scheme"] != "bearer" {
		t.Errorf("scheme = %v, want bearer", m["scheme"])
	}
	if m["bearerFormat"] != "JWT" {
		t.Errorf("bearerFormat = %v, want JWT", m["bearerFormat"])
	}
}

func TestBuildSecuritySchemes_APIKey(t *testing.T) {
	cfg := &lang.ServerConfig{
		Auth: &lang.AuthConfig{
			Strategies: map[string]*lang.StrategyConfig{
				"mykey": {
					Type:   "apikey",
					Header: "X-Custom-Key",
				},
			},
		},
	}

	schemes := buildSecuritySchemes(cfg)
	apiKey, ok := schemes["mykeyAuth"]
	if !ok {
		t.Fatal("expected mykeyAuth scheme")
	}

	m := apiKey.(map[string]any)
	if m["type"] != "apiKey" {
		t.Errorf("type = %v, want apiKey", m["type"])
	}
	if m["in"] != "header" {
		t.Errorf("in = %v, want header", m["in"])
	}
	if m["name"] != "X-Custom-Key" {
		t.Errorf("name = %v, want X-Custom-Key", m["name"])
	}
}

func TestBuildSecuritySchemes_Empty(t *testing.T) {
	cfg := &lang.ServerConfig{}

	schemes := buildSecuritySchemes(cfg)
	if len(schemes) != 0 {
		t.Errorf("expected empty schemes, got %v", schemes)
	}
}

func TestGenerateOpenAPISpec_Basic(t *testing.T) {
	program := &lang.CompiledProgram{
		Name: "test-api",
		AST: &lang.Program{
			Server: &lang.ServerConfig{Port: 3000},
			Routes: []lang.RouteConfig{
				{Method: "GET", Path: "/users", Handler: "listUsers"},
			},
		},
		Functions: map[string]*lang.CompiledFunc{
			"listUsers": {},
		},
	}

	spec := GenerateOpenAPISpec(program)

	if spec["openapi"] != "3.0.3" {
		t.Errorf("openapi = %v, want 3.0.3", spec["openapi"])
	}

	info, ok := spec["info"].(map[string]any)
	if !ok {
		t.Fatal("expected info map")
	}
	if info["title"] != "test-api" {
		t.Errorf("info.title = %v, want test-api", info["title"])
	}

	paths, ok := spec["paths"].(map[string]any)
	if !ok {
		t.Fatal("expected paths map")
	}
	if _, ok := paths["/users"]; !ok {
		t.Error("expected /users path")
	}
}

func TestGenerateOpenAPISpec_WithAuth(t *testing.T) {
	program := &lang.CompiledProgram{
		Name: "auth-api",
		AST: &lang.Program{
			Server: &lang.ServerConfig{
				Port: 3000,
				JWT:  &lang.JWTConfig{SecretEnv: "JWT_SECRET"},
			},
			Routes: []lang.RouteConfig{
				{Method: "GET", Path: "/protected", Handler: "protectedHandler", Middleware: []string{"jwt"}},
			},
		},
		Functions: map[string]*lang.CompiledFunc{
			"protectedHandler": {},
		},
	}

	spec := GenerateOpenAPISpec(program)

	paths := spec["paths"].(map[string]any)
	protectedPath := paths["/protected"].(map[string]any)
	getOp := protectedPath["get"].(map[string]any)

	security, ok := getOp["security"]
	if !ok {
		t.Fatal("expected security field on authenticated route")
	}

	secList := security.([]map[string]any)
	if len(secList) == 0 {
		t.Fatal("expected non-empty security list")
	}
	if _, ok := secList[0]["bearerAuth"]; !ok {
		t.Error("expected bearerAuth in security")
	}
}

func TestGenerateOpenAPISpec_WithAPIAnnotation(t *testing.T) {
	program := &lang.CompiledProgram{
		Name: "annotated-api",
		AST: &lang.Program{
			Server: &lang.ServerConfig{Port: 3000},
			Routes: []lang.RouteConfig{
				{
					Method:     "POST",
					Path:       "/users",
					Handler:    "createUser",
					Middleware: []string{"jwt"},
					API: &lang.APIAnnotation{
						Summary: "Create user",
						Tags:    []string{"users"},
						Responses: map[string]*lang.APIResponseAnnotation{
							"201": {Description: "Created"},
						},
					},
				},
			},
		},
		Functions: map[string]*lang.CompiledFunc{
			"createUser": {},
		},
	}

	spec := GenerateOpenAPISpec(program)

	paths := spec["paths"].(map[string]any)
	usersPath := paths["/users"].(map[string]any)
	postOp := usersPath["post"].(map[string]any)

	if postOp["summary"] != "Create user" {
		t.Errorf("summary = %v, want 'Create user'", postOp["summary"])
	}

	tags := postOp["tags"].([]string)
	if len(tags) == 0 || tags[0] != "users" {
		t.Errorf("tags = %v, want [users]", tags)
	}

	responses := postOp["responses"].(map[string]any)
	resp201, ok := responses["201"]
	if !ok {
		t.Fatal("expected 201 response")
	}
	respMap := resp201.(map[string]any)
	if respMap["description"] != "Created" {
		t.Errorf("201 description = %v, want Created", respMap["description"])
	}
}
