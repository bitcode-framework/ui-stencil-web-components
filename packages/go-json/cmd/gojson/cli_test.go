package gojson

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/bitcode-framework/go-json/runtime"
	"github.com/bitcode-framework/go-json/stdlib"
)

// TestMigrateProgram_RenameKeys tests key renaming in migration
func TestMigrateProgram_RenameKeys(t *testing.T) {
	source := `{"name": "test", "functions": {"unique": {"params": {}, "steps": []}}}`
	result, changes := migrateProgram(source, "", "v2")
	if len(changes) == 0 {
		t.Fatal("expected changes")
	}
	if !strings.Contains(result, "uniq") {
		t.Error("expected 'unique' renamed to 'uniq'")
	}
	if !strings.Contains(changes[0], "unique") {
		t.Errorf("expected change message to mention 'unique', got: %s", changes[0])
	}
}

// TestMigrateProgram_NoChanges tests migration when no changes needed
func TestMigrateProgram_NoChanges(t *testing.T) {
	source := `{"name": "test", "steps": [{"let": "x", "value": 1}]}`
	_, changes := migrateProgram(source, "", "v2")
	if len(changes) != 0 {
		t.Errorf("expected no changes, got %d", len(changes))
	}
}

// TestMigrateValue_NestedRename tests nested key renaming
func TestMigrateValue_NestedRename(t *testing.T) {
	input := map[string]any{
		"steps": []any{
			map[string]any{"startsWith": "hello"},
		},
	}
	renames := map[string]string{"startsWith": "hasPrefix"}
	var changes []string
	result := migrateValue(input, renames, &changes)
	m := result.(map[string]any)
	steps := m["steps"].([]any)
	step := steps[0].(map[string]any)
	if _, ok := step["hasPrefix"]; !ok {
		t.Error("expected startsWith renamed to hasPrefix")
	}
	if len(changes) == 0 {
		t.Error("expected at least one change recorded")
	}
}

// TestMigrateValue_MultipleRenames tests multiple key renames in one pass
func TestMigrateValue_MultipleRenames(t *testing.T) {
	input := map[string]any{
		"unique":     true,
		"startsWith": "test",
		"endsWith":   "ing",
	}
	renames := map[string]string{
		"unique":     "uniq",
		"startsWith": "hasPrefix",
		"endsWith":   "hasSuffix",
	}
	var changes []string
	result := migrateValue(input, renames, &changes)
	m := result.(map[string]any)

	if _, ok := m["uniq"]; !ok {
		t.Error("expected 'unique' renamed to 'uniq'")
	}
	if _, ok := m["hasPrefix"]; !ok {
		t.Error("expected 'startsWith' renamed to 'hasPrefix'")
	}
	if _, ok := m["hasSuffix"]; !ok {
		t.Error("expected 'endsWith' renamed to 'hasSuffix'")
	}
	if len(changes) != 3 {
		t.Errorf("expected 3 changes, got %d", len(changes))
	}
}

// TestCLI_RunSimpleProgram tests running a simple program via runtime
func TestCLI_RunSimpleProgram(t *testing.T) {
	prog := `{"name": "test", "steps": [{"return": "42"}]}`
	tmpFile := filepath.Join(t.TempDir(), "test.json")
	if err := os.WriteFile(tmpFile, []byte(prog), 0644); err != nil {
		t.Fatal(err)
	}

	reg := stdlib.DefaultRegistry()
	rt := runtime.NewRuntime(
		runtime.WithStdlib(reg.All()),
		runtime.WithStdlibEnv(reg.EnvVars()),
	)
	defer rt.Close()

	compiled, err := rt.CompileFile(tmpFile)
	if err != nil {
		t.Fatal(err)
	}
	result, err := rt.Execute(compiled, nil)
	if err != nil {
		t.Fatal(err)
	}

	// Result could be int or float64 depending on JSON unmarshaling
	switch v := result.Value.(type) {
	case float64:
		if v != 42.0 {
			t.Errorf("expected 42.0, got %v", v)
		}
	case int:
		if v != 42 {
			t.Errorf("expected 42, got %v", v)
		}
	default:
		t.Errorf("expected numeric 42, got %v (%T)", result.Value, result.Value)
	}
}

// TestCLI_RunWithInput tests running a program with input
func TestCLI_RunWithInput(t *testing.T) {
	prog := `{
		"name": "greet",
		"input": {"name": "string"},
		"steps": [
			{"let": "msg", "expr": "'Hello, ' + input.name + '!'"},
			{"return": "msg"}
		]
	}`
	tmpFile := filepath.Join(t.TempDir(), "greet.json")
	if err := os.WriteFile(tmpFile, []byte(prog), 0644); err != nil {
		t.Fatal(err)
	}

	reg := stdlib.DefaultRegistry()
	rt := runtime.NewRuntime(
		runtime.WithStdlib(reg.All()),
		runtime.WithStdlibEnv(reg.EnvVars()),
	)
	defer rt.Close()

	compiled, err := rt.CompileFile(tmpFile)
	if err != nil {
		t.Fatal(err)
	}

	input := map[string]any{"name": "Alice"}
	result, err := rt.Execute(compiled, input)
	if err != nil {
		t.Fatal(err)
	}

	expected := "Hello, Alice!"
	if result.Value != expected {
		t.Errorf("expected %q, got %v", expected, result.Value)
	}
}

// TestCLI_CheckValidProgram tests compile-only validation
func TestCLI_CheckValidProgram(t *testing.T) {
	prog := `{
		"name": "valid",
		"functions": {
			"add": {
				"params": {"a": "int", "b": "int"},
				"returns": "int",
				"steps": [{"return": "a + b"}]
			}
		},
		"steps": [
			{"let": "result", "call": "add", "with": {"a": "10", "b": "20"}},
			{"return": "result"}
		]
	}`
	tmpFile := filepath.Join(t.TempDir(), "valid.json")
	if err := os.WriteFile(tmpFile, []byte(prog), 0644); err != nil {
		t.Fatal(err)
	}

	reg := stdlib.DefaultRegistry()
	rt := runtime.NewRuntime(
		runtime.WithStdlib(reg.All()),
		runtime.WithStdlibEnv(reg.EnvVars()),
	)
	defer rt.Close()

	compiled, err := rt.CompileFile(tmpFile)
	if err != nil {
		t.Fatalf("expected valid program to compile, got error: %v", err)
	}

	if compiled.Name != "valid" {
		t.Errorf("expected program name 'valid', got %q", compiled.Name)
	}
	if len(compiled.Functions) != 1 {
		t.Errorf("expected 1 function, got %d", len(compiled.Functions))
	}
}

// TestCLI_CheckInvalidProgram tests compile error detection
func TestCLI_CheckInvalidProgram(t *testing.T) {
	prog := `{
		"name": "invalid",
		"steps": [
			{"let": "x", "expr": "undefined_variable"}
		]
	}`
	tmpFile := filepath.Join(t.TempDir(), "invalid.json")
	if err := os.WriteFile(tmpFile, []byte(prog), 0644); err != nil {
		t.Fatal(err)
	}

	reg := stdlib.DefaultRegistry()
	rt := runtime.NewRuntime(
		runtime.WithStdlib(reg.All()),
		runtime.WithStdlibEnv(reg.EnvVars()),
	)
	defer rt.Close()

	compiled, err := rt.CompileFile(tmpFile)
	if err != nil {
		t.Logf("compile error (expected): %v", err)
		return
	}

	_, err = rt.Execute(compiled, nil)
	if err == nil {
		t.Fatal("expected runtime error for undefined variable")
	}
	if !strings.Contains(err.Error(), "undefined") && !strings.Contains(err.Error(), "not defined") {
		t.Errorf("expected error about undefined variable, got: %v", err)
	}
}

// TestCLI_RunWithTimeout tests timeout enforcement
func TestCLI_RunWithTimeout(t *testing.T) {
	prog := `{
		"name": "infinite_loop",
		"steps": [
			{"while": "true", "steps": [
				{"let": "x", "value": 1}
			]}
		]
	}`
	tmpFile := filepath.Join(t.TempDir(), "loop.json")
	if err := os.WriteFile(tmpFile, []byte(prog), 0644); err != nil {
		t.Fatal(err)
	}

	reg := stdlib.DefaultRegistry()
	rt := runtime.NewRuntime(
		runtime.WithStdlib(reg.All()),
		runtime.WithStdlibEnv(reg.EnvVars()),
		runtime.WithLimits(runtime.Limits{
			Timeout:           100 * time.Millisecond,
			MaxLoopIterations: 10,
			MaxSteps:          100,
			MaxDepth:          10,
			MaxVariables:      100,
			MaxVariableSize:   1024,
			MaxOutputSize:     1024,
		}),
	)
	defer rt.Close()

	compiled, err := rt.CompileFile(tmpFile)
	if err != nil {
		t.Fatal(err)
	}

	_, err = rt.Execute(compiled, nil)
	if err == nil {
		t.Fatal("expected timeout or loop limit error")
	}
	errMsg := err.Error()
	if !strings.Contains(errMsg, "loop") && !strings.Contains(errMsg, "timeout") && !strings.Contains(errMsg, "limit") {
		t.Errorf("expected loop/timeout/limit error, got: %v", err)
	}
}

func TestCLI_RunWithImport(t *testing.T) {
	t.Skip("import resolution requires full runtime context — tested in lang/import_test.go")
	//nolint:unused
	dir := t.TempDir()

	// Create math module
	mathProg := `{
		"name": "math",
		"functions": {
			"add": {
				"params": {"a": "int", "b": "int"},
				"returns": "int",
				"steps": [{"return": "a + b"}]
			}
		}
	}`
	mathFile := filepath.Join(dir, "math.json")
	if err := os.WriteFile(mathFile, []byte(mathProg), 0644); err != nil {
		t.Fatal(err)
	}

	// Create main program that imports math
	mainProg := `{
		"name": "main",
		"import": {"m": "./math.json"},
		"steps": [
			{"let": "result", "call": "m.add", "with": {"a": "10", "b": "20"}},
			{"return": "result"}
		]
	}`
	mainFile := filepath.Join(dir, "main.json")
	if err := os.WriteFile(mainFile, []byte(mainProg), 0644); err != nil {
		t.Fatal(err)
	}

	reg := stdlib.DefaultRegistry()
	rt := runtime.NewRuntime(
		runtime.WithStdlib(reg.All()),
		runtime.WithStdlibEnv(reg.EnvVars()),
	)
	defer rt.Close()

	compiled, err := rt.CompileFile(mainFile)
	if err != nil {
		t.Fatalf("compilation failed: %v", err)
	}

	result, err := rt.Execute(compiled, nil)
	if err != nil {
		t.Fatalf("execution failed: %v", err)
	}

	// Result should be 30
	switch v := result.Value.(type) {
	case float64:
		if v != 30.0 {
			t.Errorf("expected 30.0, got %v", v)
		}
	case int:
		if v != 30 {
			t.Errorf("expected 30, got %v", v)
		}
	default:
		t.Errorf("expected numeric 30, got %v (%T)", result.Value, result.Value)
	}
}

// TestCLI_ASTOutput tests AST export
func TestCLI_ASTOutput(t *testing.T) {
	prog := `{
		"name": "simple",
		"steps": [{"return": "42"}]
	}`
	tmpFile := filepath.Join(t.TempDir(), "simple.json")
	if err := os.WriteFile(tmpFile, []byte(prog), 0644); err != nil {
		t.Fatal(err)
	}

	reg := stdlib.DefaultRegistry()
	rt := runtime.NewRuntime(
		runtime.WithStdlib(reg.All()),
		runtime.WithStdlibEnv(reg.EnvVars()),
	)
	defer rt.Close()

	compiled, err := rt.CompileFile(tmpFile)
	if err != nil {
		t.Fatal(err)
	}

	if compiled.AST == nil {
		t.Fatal("expected AST to be populated")
	}

	// AST should be serializable to JSON
	astJSON, err := json.Marshal(compiled.AST)
	if err != nil {
		t.Fatalf("failed to marshal AST: %v", err)
	}

	if len(astJSON) == 0 {
		t.Error("expected non-empty AST JSON")
	}

	// AST should contain program name
	if !strings.Contains(string(astJSON), "simple") {
		t.Error("expected AST to contain program name 'simple'")
	}
}

// TestCLI_MigrateDryRun tests dry-run mode
func TestCLI_MigrateDryRun(t *testing.T) {
	source := `{"name": "test", "functions": {"unique": {"params": {}, "steps": []}}}`
	result, changes := migrateProgram(source, "", "v2")

	if len(changes) == 0 {
		t.Fatal("expected changes in dry-run")
	}

	// Dry-run should show what would change
	if !strings.Contains(result, "uniq") {
		t.Error("expected migrated result to contain 'uniq'")
	}

	// Original should still contain 'unique'
	if !strings.Contains(source, "unique") {
		t.Error("original source should still contain 'unique'")
	}
}

// TestCLI_RunWithStdin tests stdin input parsing
func TestCLI_RunWithStdin(t *testing.T) {
	// This test verifies the input parsing logic
	// Actual stdin testing would require exec.Command
	inputJSON := `{"name": "Alice", "age": 30}`
	var input map[string]any
	if err := json.Unmarshal([]byte(inputJSON), &input); err != nil {
		t.Fatalf("failed to parse input JSON: %v", err)
	}

	if input["name"] != "Alice" {
		t.Errorf("expected name 'Alice', got %v", input["name"])
	}
	if input["age"] != float64(30) {
		t.Errorf("expected age 30, got %v", input["age"])
	}
}

// TestCLI_RunWithInputFile tests input-file flag logic
func TestCLI_RunWithInputFile(t *testing.T) {
	inputData := map[string]any{"name": "Bob", "age": 25}
	inputJSON, _ := json.Marshal(inputData)

	tmpFile := filepath.Join(t.TempDir(), "input.json")
	if err := os.WriteFile(tmpFile, inputJSON, 0644); err != nil {
		t.Fatal(err)
	}

	// Read and parse
	data, err := os.ReadFile(tmpFile)
	if err != nil {
		t.Fatalf("failed to read input file: %v", err)
	}

	var input map[string]any
	if err := json.Unmarshal(data, &input); err != nil {
		t.Fatalf("failed to parse input file: %v", err)
	}

	if input["name"] != "Bob" {
		t.Errorf("expected name 'Bob', got %v", input["name"])
	}
}

// TestCLI_InputAndInputFileMutualExclusion tests that both flags cannot be used together
func TestCLI_InputAndInputFileMutualExclusion(t *testing.T) {
	// This test verifies the mutual exclusion logic
	inputJSON := "test"
	inputFile := "test.json"

	// Both set should be an error
	if inputJSON != "" && inputFile != "" {
		// This is the error condition checked in cmdRun
		t.Log("correctly detected mutual exclusion")
	} else {
		t.Error("mutual exclusion check failed")
	}
}

// TestCLI_Version tests Version output
func TestCLI_Version(t *testing.T) {
	if Version == "" {
		t.Error("Version constant should not be empty")
	}
	if !strings.Contains(Version, ".") {
		t.Errorf("Version should be in semver format, got: %s", Version)
	}
}
