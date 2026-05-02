package lang

import (
	"testing"
)

func TestWasmImportParsing(t *testing.T) {
	program, err := Parse([]byte(`{
		"import": {
			"img": "wasm:./plugins/imgproc.wasm",
			"math": "wasm:./compute/math.wasm"
		},
		"steps": []
	}`))
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	if len(program.Imports) != 2 {
		t.Fatalf("expected 2 imports, got %d", len(program.Imports))
	}

	types := map[string]string{}
	paths := map[string]string{}
	for _, imp := range program.Imports {
		types[imp.Alias] = imp.PathType
		paths[imp.Alias] = imp.Path
	}

	if types["img"] != "wasm" {
		t.Errorf("expected img PathType=wasm, got %s", types["img"])
	}
	if types["math"] != "wasm" {
		t.Errorf("expected math PathType=wasm, got %s", types["math"])
	}
	if paths["img"] != "wasm:./plugins/imgproc.wasm" {
		t.Errorf("expected img path preserved, got %s", paths["img"])
	}
}

func TestPluginImportParsing(t *testing.T) {
	program, err := Parse([]byte(`{
		"import": {
			"fast": "plugin:./plugins/fast.so",
			"crypto": "plugin:./plugins/crypto.dylib"
		},
		"steps": []
	}`))
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	if len(program.Imports) != 2 {
		t.Fatalf("expected 2 imports, got %d", len(program.Imports))
	}

	types := map[string]string{}
	paths := map[string]string{}
	for _, imp := range program.Imports {
		types[imp.Alias] = imp.PathType
		paths[imp.Alias] = imp.Path
	}

	if types["fast"] != "plugin" {
		t.Errorf("expected fast PathType=plugin, got %s", types["fast"])
	}
	if types["crypto"] != "plugin" {
		t.Errorf("expected crypto PathType=plugin, got %s", types["crypto"])
	}
	if paths["fast"] != "plugin:./plugins/fast.so" {
		t.Errorf("expected fast path preserved, got %s", paths["fast"])
	}
}

func TestDetectImportPathType_WasmAndPlugin(t *testing.T) {
	tests := []struct {
		path     string
		expected string
	}{
		{"wasm:./plugin.wasm", "wasm"},
		{"wasm:./plugins/imgproc.wasm", "wasm"},
		{"wasm:../shared/compute.wasm", "wasm"},
		{"plugin:./fast.so", "plugin"},
		{"plugin:./plugins/crypto.dylib", "plugin"},
		{"plugin:../shared/lib.so", "plugin"},
		{"script:./plugin.py", "script"},
		{"./relative.json", "relative"},
		{"io:http", "io"},
		{"ext:bitcode", "ext"},
		{"stdlib:math", "stdlib"},
	}

	for _, tt := range tests {
		got := detectImportPathType(tt.path)
		if got != tt.expected {
			t.Errorf("detectImportPathType(%q) = %q, want %q", tt.path, got, tt.expected)
		}
	}
}

func TestWasmImportRecordedInRequestedModules(t *testing.T) {
	program, err := Parse([]byte(`{
		"import": {
			"img": "wasm:./plugins/imgproc.wasm",
			"http": "io:http"
		},
		"steps": []
	}`))
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	resolver := NewImportResolver()
	err = resolver.ResolveImports(program, t.TempDir(), nil)
	if err != nil {
		t.Fatalf("resolve error: %v", err)
	}

	if program.RequestedModules == nil {
		t.Fatal("expected RequestedModules to be populated")
	}

	if imp, ok := program.RequestedModules["img"]; !ok {
		t.Error("expected 'img' in RequestedModules")
	} else if imp.PathType != "wasm" {
		t.Errorf("expected img PathType=wasm, got %s", imp.PathType)
	}

	if _, ok := program.RequestedModules["http"]; !ok {
		t.Error("expected 'http' in RequestedModules")
	}
}

func TestPluginImportRecordedInRequestedModules(t *testing.T) {
	program, err := Parse([]byte(`{
		"import": {
			"fast": "plugin:./plugins/fast.so",
			"db": "io:sql"
		},
		"steps": []
	}`))
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	resolver := NewImportResolver()
	err = resolver.ResolveImports(program, t.TempDir(), nil)
	if err != nil {
		t.Fatalf("resolve error: %v", err)
	}

	if program.RequestedModules == nil {
		t.Fatal("expected RequestedModules to be populated")
	}

	if imp, ok := program.RequestedModules["fast"]; !ok {
		t.Error("expected 'fast' in RequestedModules")
	} else if imp.PathType != "plugin" {
		t.Errorf("expected fast PathType=plugin, got %s", imp.PathType)
	}

	if _, ok := program.RequestedModules["db"]; !ok {
		t.Error("expected 'db' in RequestedModules")
	}
}

func TestWasmAndPluginImportPathTypes(t *testing.T) {
	program, err := Parse([]byte(`{
		"import": {
			"models": "./models.json",
			"v": "stdlib:validators",
			"db": "io:database",
			"bc": "ext:bitcode",
			"ml": "script:./predict.py",
			"img": "wasm:./imgproc.wasm",
			"fast": "plugin:./fast.so"
		},
		"steps": []
	}`))
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	types := map[string]string{}
	for _, imp := range program.Imports {
		types[imp.Alias] = imp.PathType
	}

	expected := map[string]string{
		"models": "relative",
		"v":      "stdlib",
		"db":     "io",
		"bc":     "ext",
		"ml":     "script",
		"img":    "wasm",
		"fast":   "plugin",
	}

	for alias, want := range expected {
		if got := types[alias]; got != want {
			t.Errorf("expected %s=%s, got %s", alias, want, got)
		}
	}
}
