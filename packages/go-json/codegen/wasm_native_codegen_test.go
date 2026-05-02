package codegen

import (
	"strings"
	"testing"

	"github.com/bitcode-framework/go-json/lang"
)

func TestDetectFeatures_Wasm(t *testing.T) {
	program := &lang.CompiledProgram{
		AST: &lang.Program{
			RequestedModules: map[string]lang.ImportDef{
				"img": {Alias: "img", Path: "wasm:./plugins/imgproc.wasm", PathType: "wasm"},
			},
		},
	}

	features := DetectFeatures(program)
	if !features.HasWasm {
		t.Error("expected HasWasm=true")
	}
	if features.HasNative {
		t.Error("expected HasNative=false")
	}
}

func TestDetectFeatures_Native(t *testing.T) {
	program := &lang.CompiledProgram{
		AST: &lang.Program{
			RequestedModules: map[string]lang.ImportDef{
				"fast": {Alias: "fast", Path: "plugin:./plugins/fast.so", PathType: "plugin"},
			},
		},
	}

	features := DetectFeatures(program)
	if !features.HasNative {
		t.Error("expected HasNative=true")
	}
	if features.HasWasm {
		t.Error("expected HasWasm=false")
	}
}

func TestDetectFeatures_WasmAndNative(t *testing.T) {
	program := &lang.CompiledProgram{
		AST: &lang.Program{
			RequestedModules: map[string]lang.ImportDef{
				"img":  {Alias: "img", Path: "wasm:./imgproc.wasm", PathType: "wasm"},
				"fast": {Alias: "fast", Path: "plugin:./fast.so", PathType: "plugin"},
				"db":   {Alias: "db", Path: "io:sql", PathType: "io"},
			},
		},
	}

	features := DetectFeatures(program)
	if !features.HasWasm {
		t.Error("expected HasWasm=true")
	}
	if !features.HasNative {
		t.Error("expected HasNative=true")
	}
	if !features.HasSQL {
		t.Error("expected HasSQL=true")
	}
}

func TestGenerateGoMod_WithWasm(t *testing.T) {
	features := DetectedFeatures{HasWasm: true}
	gomod := GenerateGoMod("myapp", features)

	if !strings.Contains(gomod, "wazero") {
		t.Error("expected wazero dependency in go.mod when HasWasm=true")
	}
}

func TestGenerateGoMod_WithoutWasm(t *testing.T) {
	features := DetectedFeatures{HasSQL: true}
	gomod := GenerateGoMod("myapp", features)

	if strings.Contains(gomod, "wazero") {
		t.Error("expected no wazero dependency when HasWasm=false")
	}
}
