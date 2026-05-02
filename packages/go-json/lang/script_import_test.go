package lang

import (
	"testing"
)

func TestScriptImportParsing(t *testing.T) {
	program, err := Parse([]byte(`{
		"import": {
			"ml": "script:./plugins/predict.py",
			"utils": "script:./helpers/utils.js",
			"fast": "script:./compute/heavy.go"
		},
		"steps": []
	}`))
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	if len(program.Imports) != 3 {
		t.Fatalf("expected 3 imports, got %d", len(program.Imports))
	}

	types := map[string]string{}
	paths := map[string]string{}
	for _, imp := range program.Imports {
		types[imp.Alias] = imp.PathType
		paths[imp.Alias] = imp.Path
	}

	if types["ml"] != "script" {
		t.Errorf("expected ml PathType=script, got %s", types["ml"])
	}
	if types["utils"] != "script" {
		t.Errorf("expected utils PathType=script, got %s", types["utils"])
	}
	if types["fast"] != "script" {
		t.Errorf("expected fast PathType=script, got %s", types["fast"])
	}

	if paths["ml"] != "script:./plugins/predict.py" {
		t.Errorf("expected ml path preserved, got %s", paths["ml"])
	}
}

func TestScriptImportPathType(t *testing.T) {
	program, err := Parse([]byte(`{
		"import": {
			"models": "./models.json",
			"v": "stdlib:validators",
			"db": "io:database",
			"bc": "ext:bitcode",
			"ml": "script:./predict.py"
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

	if types["models"] != "relative" {
		t.Errorf("expected models=relative, got %s", types["models"])
	}
	if types["v"] != "stdlib" {
		t.Errorf("expected v=stdlib, got %s", types["v"])
	}
	if types["db"] != "io" {
		t.Errorf("expected db=io, got %s", types["db"])
	}
	if types["bc"] != "ext" {
		t.Errorf("expected bc=ext, got %s", types["bc"])
	}
	if types["ml"] != "script" {
		t.Errorf("expected ml=script, got %s", types["ml"])
	}
}

func TestScriptImportRecordedInRequestedModules(t *testing.T) {
	program, err := Parse([]byte(`{
		"import": {
			"ml": "script:./plugins/predict.py",
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

	if imp, ok := program.RequestedModules["ml"]; !ok {
		t.Error("expected 'ml' in RequestedModules")
	} else if imp.PathType != "script" {
		t.Errorf("expected ml PathType=script, got %s", imp.PathType)
	}

	if _, ok := program.RequestedModules["http"]; !ok {
		t.Error("expected 'http' in RequestedModules")
	}
}

func TestDetectImportPathType_Script(t *testing.T) {
	tests := []struct {
		path     string
		expected string
	}{
		{"script:./plugin.py", "script"},
		{"script:./helpers/utils.js", "script"},
		{"script:../shared/compute.go", "script"},
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
