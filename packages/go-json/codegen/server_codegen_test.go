package codegen

import (
	"strings"
	"testing"

	"github.com/bitcode-framework/go-json/lang"
)

func TestDefaultFramework(t *testing.T) {
	tests := []struct {
		language string
		expected string
	}{
		{"go", "fiber"},
		{"javascript", "express"},
		{"python", "fastapi"},
		{"unknown", ""},
	}

	for _, tt := range tests {
		result := DefaultFramework(tt.language)
		if result != tt.expected {
			t.Errorf("DefaultFramework(%q) = %q, want %q", tt.language, result, tt.expected)
		}
	}
}

func TestResolveFramework_Explicit(t *testing.T) {
	result := ResolveFramework("go", "nethttp", "fiber")
	if result != "nethttp" {
		t.Errorf("ResolveFramework with explicit = %q, want %q", result, "nethttp")
	}
}

func TestResolveFramework_FromConfig(t *testing.T) {
	result := ResolveFramework("go", "", "nethttp")
	if result != "nethttp" {
		t.Errorf("ResolveFramework from config = %q, want %q", result, "nethttp")
	}
}

func TestResolveFramework_Default(t *testing.T) {
	result := ResolveFramework("go", "", "")
	if result != "fiber" {
		t.Errorf("ResolveFramework default = %q, want %q", result, "fiber")
	}
}

func TestGetServerCodegen_GoFiber(t *testing.T) {
	adapter, err := GetServerCodegen("go", "fiber")
	if err != nil {
		t.Fatalf("GetServerCodegen(go, fiber) error: %v", err)
	}
	if adapter == nil {
		t.Fatal("expected non-nil adapter")
	}
	if adapter.Language() != "go" {
		t.Errorf("adapter.Language() = %q, want %q", adapter.Language(), "go")
	}
	if adapter.Framework() != "fiber" {
		t.Errorf("adapter.Framework() = %q, want %q", adapter.Framework(), "fiber")
	}
}

func TestGetServerCodegen_Unknown(t *testing.T) {
	_, err := GetServerCodegen("rust", "actix")
	if err == nil {
		t.Error("expected error for unknown language+framework, got nil")
	}
}

func makeServerProgram() *lang.CompiledProgram {
	return &lang.CompiledProgram{
		Name: "test-api",
		AST: &lang.Program{
			Server:     &lang.ServerConfig{Port: 3000, Framework: "fiber"},
			Routes:     []lang.RouteConfig{{Method: "GET", Path: "/users", Handler: "listUsers"}},
			Middleware: []string{"logger"},
		},
		Functions: map[string]*lang.CompiledFunc{
			"listUsers": {Steps: []lang.Node{&lang.ReturnNode{HasExpr: true, Expr: "'ok'"}}},
		},
	}
}

func TestDetectFeatures_Server(t *testing.T) {
	program := makeServerProgram()
	features := DetectFeatures(program)

	if !features.HasServer {
		t.Error("expected HasServer = true")
	}
	if features.Framework != "fiber" {
		t.Errorf("expected Framework = %q, got %q", "fiber", features.Framework)
	}
}

func TestDetectFeatures_JWT(t *testing.T) {
	program := &lang.CompiledProgram{
		Name: "jwt-api",
		AST: &lang.Program{
			Server: &lang.ServerConfig{
				Port:      3000,
				Framework: "fiber",
				JWT:       &lang.JWTConfig{SecretEnv: "JWT_SECRET"},
			},
		},
		Functions: map[string]*lang.CompiledFunc{},
	}

	features := DetectFeatures(program)
	if !features.HasJWT {
		t.Error("expected HasJWT = true")
	}
}

func TestDetectFeatures_IOModules(t *testing.T) {
	program := &lang.CompiledProgram{
		Name: "io-api",
		AST: &lang.Program{
			RequestedModules: map[string]lang.ImportDef{
				"db":   {Alias: "db", Path: "io:sql", PathType: "io"},
				"http": {Alias: "http", Path: "io:http", PathType: "io"},
			},
		},
		Functions: map[string]*lang.CompiledFunc{},
	}

	features := DetectFeatures(program)
	if !features.HasSQL {
		t.Error("expected HasSQL = true")
	}
	if !features.HasHTTP {
		t.Error("expected HasHTTP = true")
	}
}

func TestGenerateGoMod_Fiber(t *testing.T) {
	features := DetectedFeatures{HasServer: true, Framework: "fiber"}
	result := GenerateGoMod("myapp", features)

	if !strings.Contains(result, "gofiber") {
		t.Error("expected go.mod to contain gofiber dependency")
	}
	if !strings.Contains(result, "module myapp") {
		t.Error("expected go.mod to contain module name")
	}
}

func TestGenerateGoMod_WithJWT(t *testing.T) {
	features := DetectedFeatures{HasServer: true, Framework: "fiber", HasJWT: true}
	result := GenerateGoMod("myapp", features)

	if !strings.Contains(result, "golang-jwt") {
		t.Error("expected go.mod to contain golang-jwt dependency")
	}
}

func TestGeneratePackageJSON_Express(t *testing.T) {
	features := DetectedFeatures{HasServer: true}
	result := GeneratePackageJSON("my-api", features)

	if !strings.Contains(result, "express") {
		t.Error("expected package.json to contain express dependency")
	}
	if !strings.Contains(result, `"name": "my-api"`) {
		t.Error("expected package.json to contain project name")
	}
}

func TestGeneratePackageJSON_WithJWT(t *testing.T) {
	features := DetectedFeatures{HasServer: true, HasJWT: true}
	result := GeneratePackageJSON("my-api", features)

	if !strings.Contains(result, "jsonwebtoken") {
		t.Error("expected package.json to contain jsonwebtoken dependency")
	}
}

func TestGenerateRequirementsTxt_FastAPI(t *testing.T) {
	features := DetectedFeatures{HasServer: true}
	result := GenerateRequirementsTxt(features)

	if !strings.Contains(result, "fastapi") {
		t.Error("expected requirements.txt to contain fastapi")
	}
	if !strings.Contains(result, "uvicorn") {
		t.Error("expected requirements.txt to contain uvicorn")
	}
}

func TestGenerateEnvExample_WithJWT(t *testing.T) {
	program := &lang.CompiledProgram{
		Name: "jwt-api",
		AST: &lang.Program{
			Server: &lang.ServerConfig{
				Port: 3000,
				JWT:  &lang.JWTConfig{SecretEnv: "JWT_SECRET"},
			},
		},
		Functions: map[string]*lang.CompiledFunc{},
	}

	result := GenerateEnvExample(program)
	if !strings.Contains(result, "JWT_SECRET") {
		t.Error("expected .env.example to contain JWT_SECRET")
	}
}

func TestGoFiberCodegen_Generate(t *testing.T) {
	program := makeServerProgram()

	adapter, err := GetServerCodegen("go", "fiber")
	if err != nil {
		t.Fatalf("GetServerCodegen error: %v", err)
	}

	files, err := adapter.GenerateServer(program)
	if err != nil {
		t.Fatalf("GenerateServer error: %v", err)
	}

	if _, ok := files["main.go"]; !ok {
		t.Error("expected files map to contain main.go")
	}
}

func TestJSExpressCodegen_Generate(t *testing.T) {
	program := makeServerProgram()

	adapter, err := GetServerCodegen("javascript", "express")
	if err != nil {
		t.Fatalf("GetServerCodegen error: %v", err)
	}

	files, err := adapter.GenerateServer(program)
	if err != nil {
		t.Fatalf("GenerateServer error: %v", err)
	}

	if _, ok := files["index.js"]; !ok {
		t.Error("expected files map to contain index.js")
	}
}

func TestPyFastAPICodegen_Generate(t *testing.T) {
	program := makeServerProgram()

	adapter, err := GetServerCodegen("python", "fastapi")
	if err != nil {
		t.Fatalf("GetServerCodegen error: %v", err)
	}

	files, err := adapter.GenerateServer(program)
	if err != nil {
		t.Fatalf("GenerateServer error: %v", err)
	}

	if _, ok := files["main.py"]; !ok {
		t.Error("expected files map to contain main.py")
	}
}
