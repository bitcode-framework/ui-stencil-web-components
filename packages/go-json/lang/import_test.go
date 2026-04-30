package lang

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeTestFile(t *testing.T, dir, name, content string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write test file %s: %v", path, err)
	}
	return path
}

func TestImport_RelativeFile(t *testing.T) {
	dir := t.TempDir()

	writeTestFile(t, dir, "types.json", `{
		"structs": {
			"Point": {
				"fields": {
					"x": "int",
					"y": "int"
				}
			}
		}
	}`)

	mainPath := writeTestFile(t, dir, "main.json", `{
		"import": {
			"t": "./types.json"
		},
		"steps": [
			{"let": "p", "new": "t.Point", "with": {"x": "3", "y": "4"}},
			{"return": "p.x + p.y"}
		]
	}`)

	program, err := ParseFile(mainPath)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	resolver := NewImportResolver()
	if err := resolver.ResolveImports(program, dir, []string{mainPath}); err != nil {
		t.Fatalf("import error: %v", err)
	}

	if _, ok := program.Structs["t.Point"]; !ok {
		t.Error("expected t.Point struct to be imported")
	}
}

func TestImport_CircularDetection(t *testing.T) {
	dir := t.TempDir()

	writeTestFile(t, dir, "a.json", `{
		"import": {"b": "./b.json"},
		"steps": []
	}`)
	writeTestFile(t, dir, "b.json", `{
		"import": {"a": "./a.json"},
		"steps": []
	}`)

	program, err := ParseFile(filepath.Join(dir, "a.json"))
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	resolver := NewImportResolver()
	err = resolver.ResolveImports(program, dir, []string{filepath.Join(dir, "a.json")})
	if err == nil {
		t.Fatal("expected circular import error")
	}
	if !strings.Contains(err.Error(), "circular") {
		t.Errorf("expected 'circular' error, got: %v", err)
	}
}

func TestImport_DiamondNotCircular(t *testing.T) {
	dir := t.TempDir()

	writeTestFile(t, dir, "shared.json", `{
		"structs": {
			"Shared": {
				"fields": {"id": "int"}
			}
		}
	}`)
	writeTestFile(t, dir, "a.json", `{
		"import": {"s": "./shared.json"},
		"structs": {
			"A": {
				"fields": {"name": "string"}
			}
		}
	}`)
	writeTestFile(t, dir, "b.json", `{
		"import": {"s": "./shared.json"},
		"structs": {
			"B": {
				"fields": {"value": "int"}
			}
		}
	}`)

	mainPath := writeTestFile(t, dir, "main.json", `{
		"import": {
			"a": "./a.json",
			"b": "./b.json"
		},
		"steps": []
	}`)

	program, err := ParseFile(mainPath)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	resolver := NewImportResolver()
	err = resolver.ResolveImports(program, dir, []string{mainPath})
	if err != nil {
		t.Fatalf("diamond import should not be circular, got: %v", err)
	}
}

func TestImport_FileNotFound(t *testing.T) {
	dir := t.TempDir()

	mainPath := writeTestFile(t, dir, "main.json", `{
		"import": {
			"x": "./nonexistent.json"
		},
		"steps": []
	}`)

	program, err := ParseFile(mainPath)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	resolver := NewImportResolver()
	err = resolver.ResolveImports(program, dir, []string{mainPath})
	if err == nil {
		t.Fatal("expected error for missing import file")
	}
}

func TestImport_PathTypeDetection(t *testing.T) {
	program, err := Parse([]byte(`{
		"import": {
			"models": "./models.json",
			"v": "stdlib:validators",
			"db": "io:database",
			"bc": "ext:bitcode"
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
}

func TestImport_BarrelFile(t *testing.T) {
	dir := t.TempDir()

	writeTestFile(t, dir, "address.json", `{
		"structs": {
			"Address": {
				"fields": {"city": "string"}
			}
		}
	}`)

	writeTestFile(t, dir, "index.json", `{
		"import": {
			"_addr": "./address.json"
		},
		"structs": {
			"Address": {"alias": "_addr.Address"}
		}
	}`)

	program, err := ParseFile(filepath.Join(dir, "index.json"))
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	resolver := NewImportResolver()
	err = resolver.ResolveImports(program, dir, []string{filepath.Join(dir, "index.json")})
	if err != nil {
		t.Fatalf("import error: %v", err)
	}

	addrDef, ok := program.Structs["Address"]
	if !ok {
		t.Fatal("expected Address struct in barrel file")
	}
	if addrDef.Alias != "_addr.Address" {
		t.Errorf("expected alias=_addr.Address, got %s", addrDef.Alias)
	}
}

func TestImport_IOModuleRecordedInRequestedModules(t *testing.T) {
	program, err := Parse([]byte(`{
		"import": {
			"http": "io:http",
			"db": "io:sql"
		},
		"steps": []
	}`))
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	resolver := NewImportResolver()
	if err := resolver.ResolveImports(program, "", nil); err != nil {
		t.Fatalf("resolve error: %v", err)
	}

	if program.RequestedModules == nil {
		t.Fatal("expected RequestedModules to be populated")
	}
	if imp, ok := program.RequestedModules["http"]; !ok {
		t.Error("expected 'http' in RequestedModules")
	} else if imp.PathType != "io" {
		t.Errorf("expected PathType=io, got %s", imp.PathType)
	}
	if imp, ok := program.RequestedModules["db"]; !ok {
		t.Error("expected 'db' in RequestedModules")
	} else if imp.Path != "io:sql" {
		t.Errorf("expected Path=io:sql, got %s", imp.Path)
	}
}

func TestImport_ExtModuleRecordedInRequestedModules(t *testing.T) {
	program, err := Parse([]byte(`{
		"import": {
			"bc": "ext:bitcode"
		},
		"steps": []
	}`))
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	resolver := NewImportResolver()
	if err := resolver.ResolveImports(program, "", nil); err != nil {
		t.Fatalf("resolve error: %v", err)
	}

	if program.RequestedModules == nil {
		t.Fatal("expected RequestedModules to be populated")
	}
	if imp, ok := program.RequestedModules["bc"]; !ok {
		t.Error("expected 'bc' in RequestedModules")
	} else {
		if imp.PathType != "ext" {
			t.Errorf("expected PathType=ext, got %s", imp.PathType)
		}
		if imp.Path != "ext:bitcode" {
			t.Errorf("expected Path=ext:bitcode, got %s", imp.Path)
		}
	}
}

func TestImport_StdlibNotInRequestedModules(t *testing.T) {
	program, err := Parse([]byte(`{
		"import": {
			"v": "stdlib:validators"
		},
		"steps": []
	}`))
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	resolver := NewImportResolver()
	if err := resolver.ResolveImports(program, "", nil); err != nil {
		t.Fatalf("resolve error: %v", err)
	}

	if program.RequestedModules != nil && len(program.RequestedModules) > 0 {
		t.Error("stdlib imports should not appear in RequestedModules")
	}
}

func TestParallel_CompileError_ParentWrite(t *testing.T) {
	program, err := Parse([]byte(`{
		"steps": [
			{"let": "counter", "value": 0},
			{
				"parallel": {
					"a": [{"set": "counter", "expr": "counter + 1"}]
				},
				"into": "results"
			}
		]
	}`))
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	engine := NewExprLangEngine()
	_, err = Compile(program, engine, DefaultLimits())
	if err == nil {
		t.Fatal("expected compile error for parallel parent write")
	}
	if !strings.Contains(err.Error(), "cannot mutate parent") {
		t.Errorf("expected 'cannot mutate parent' error, got: %v", err)
	}
}

func TestParallel_LocalLetThenSet_Allowed(t *testing.T) {
	result := compileAndRun(t, `{
		"steps": [
			{
				"parallel": {
					"a": [
						{"let": "x", "value": 10},
						{"set": "x", "value": 20},
						{"return": "x"}
					]
				},
				"into": "results"
			},
			{"return": "results.a"}
		]
	}`, nil)

	if !numEq(result.Value, 20) {
		t.Errorf("expected 20, got %v", result.Value)
	}
}

func TestImport_IndirectCircularDetection(t *testing.T) {
	dir := t.TempDir()

	writeTestFile(t, dir, "a.json", `{
		"import": {"b": "./b.json"},
		"steps": []
	}`)
	writeTestFile(t, dir, "b.json", `{
		"import": {"c": "./c.json"},
		"steps": []
	}`)
	writeTestFile(t, dir, "c.json", `{
		"import": {"a": "./a.json"},
		"steps": []
	}`)

	program, err := ParseFile(filepath.Join(dir, "a.json"))
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	resolver := NewImportResolver()
	err = resolver.ResolveImports(program, dir, []string{filepath.Join(dir, "a.json")})
	if err == nil {
		t.Fatal("expected circular import error for A→B→C→A")
	}
	if !strings.Contains(err.Error(), "circular") {
		t.Errorf("expected 'circular' error, got: %v", err)
	}
}

func TestImport_AliasCollision_Error(t *testing.T) {
	dir := t.TempDir()

	writeTestFile(t, dir, "types_a.json", `{
		"structs": {
			"Item": {
				"fields": {"name": "string"}
			}
		},
		"functions": {
			"helper": {
				"steps": [{"return": 1}]
			}
		}
	}`)
	writeTestFile(t, dir, "types_b.json", `{
		"structs": {
			"Item": {
				"fields": {"value": "int"}
			}
		}
	}`)

	mainPath := writeTestFile(t, dir, "main.json", `{
		"import": {
			"m": "./types_a.json",
			"m2": "./types_b.json"
		},
		"steps": []
	}`)

	program, err := ParseFile(mainPath)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	resolver := NewImportResolver()
	err = resolver.ResolveImports(program, dir, []string{mainPath})
	if err != nil {
		t.Fatalf("no collision expected for different aliases: %v", err)
	}

	if _, ok := program.Structs["m.Item"]; !ok {
		t.Error("expected m.Item")
	}
	if _, ok := program.Structs["m2.Item"]; !ok {
		t.Error("expected m2.Item")
	}
	if _, ok := program.Functions["m.helper"]; !ok {
		t.Error("expected m.helper")
	}
}

func TestImport_SameAliasCollision_Error(t *testing.T) {
	dir := t.TempDir()

	writeTestFile(t, dir, "types.json", `{
		"structs": {
			"Item": {
				"fields": {"name": "string"}
			}
		}
	}`)

	mainPath := writeTestFile(t, dir, "main.json", `{
		"steps": []
	}`)

	program, err := ParseFile(mainPath)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	if program.Structs == nil {
		program.Structs = make(map[string]*StructDef)
	}
	program.Structs["t.Item"] = &StructDef{Name: "Item", Fields: map[string]*FieldDef{"x": {Type: "int"}}}

	resolver := NewImportResolver()
	program.Imports = []*ImportDef{{Alias: "t", Path: "./types.json", PathType: "relative"}}
	err = resolver.ResolveImports(program, dir, []string{mainPath})
	if err == nil {
		t.Fatal("expected collision error for duplicate t.Item")
	}
	if !strings.Contains(err.Error(), "collision") {
		t.Errorf("expected 'collision' error, got: %v", err)
	}
}

func TestImport_FileWithJsonSyntaxError(t *testing.T) {
	dir := t.TempDir()

	writeTestFile(t, dir, "bad.json", `{invalid json content}`)

	mainPath := writeTestFile(t, dir, "main.json", `{
		"import": {
			"b": "./bad.json"
		},
		"steps": []
	}`)

	program, err := ParseFile(mainPath)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	resolver := NewImportResolver()
	err = resolver.ResolveImports(program, dir, []string{mainPath})
	if err == nil {
		t.Fatal("expected error for JSON syntax error in imported file")
	}
}
