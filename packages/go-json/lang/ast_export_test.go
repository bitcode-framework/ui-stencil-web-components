package lang

import (
	"encoding/json"
	"testing"
)

func TestASTExport_Roundtrip(t *testing.T) {
	programJSON := `{
		"name": "test_program",
		"go_json": "1",
		"input": {"x": "int", "y": "int"},
		"functions": {
			"add": {
				"params": {"a": "int", "b": "int"},
				"returns": "int",
				"steps": [
					{"return": "a + b"}
				]
			}
		},
		"steps": [
			{"let": "sum", "call": "add", "with": {"a": "input.x", "b": "input.y"}},
			{"return": "sum"}
		]
	}`

	program, err := Parse([]byte(programJSON))
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	astJSON, err := json.MarshalIndent(program, "", "  ")
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	var astMap map[string]any
	err = json.Unmarshal(astJSON, &astMap)
	if err != nil {
		t.Fatalf("Unmarshal to map failed: %v", err)
	}

	if astMap["Name"] != program.Name {
		t.Errorf("Name mismatch: expected %s, got %v", program.Name, astMap["Name"])
	}

	if astMap["GoJSON"] != program.GoJSON {
		t.Errorf("GoJSON mismatch: expected %s, got %v", program.GoJSON, astMap["GoJSON"])
	}

	input, ok := astMap["Input"].([]any)
	if !ok || len(input) != len(program.Input) {
		t.Errorf("Input length mismatch: expected %d, got %d", len(program.Input), len(input))
	}

	functions, ok := astMap["Functions"].(map[string]any)
	if !ok || len(functions) != len(program.Functions) {
		t.Errorf("Functions length mismatch: expected %d, got %d", len(program.Functions), len(functions))
	}

	steps, ok := astMap["Steps"].([]any)
	if !ok || len(steps) != len(program.Steps) {
		t.Errorf("Steps length mismatch: expected %d, got %d", len(program.Steps), len(steps))
	}
}

func TestASTExport_AllStepTypes(t *testing.T) {
	programJSON := `{
		"name": "all_steps",
		"go_json": "1",
		"steps": [
			{"let": "x", "value": 10},
			{"set": "x", "expr": "x + 1"},
			{"if": "x > 5", "then": [{"log": "x is greater than 5"}]},
			{"switch": "x", "cases": {"10": [{"log": "x is 10"}]}},
			{"for": "i", "in": "[1, 2, 3]", "steps": [{"log": "i"}]},
			{"while": "x < 20", "steps": [{"set": "x", "expr": "x + 1"}]},
			{"break": true},
			{"continue": true},
			{"return": "x"},
			{"call": "someFunc"},
			{"try": [{"log": "trying"}], "catch": "e", "do": [{"log": "caught"}]},
			{"error": "Something went wrong"},
			{"log": "test message"},
			{"parallel": {"branch1": [{"log": "a"}], "branch2": [{"log": "b"}]}},
			{"_c": "This is a comment"}
		]
	}`

	program, err := Parse([]byte(programJSON))
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if len(program.Steps) != 15 {
		t.Errorf("Expected 15 steps, got %d", len(program.Steps))
	}

	expectedTypes := []string{
		"let", "set", "if", "switch", "for", "while", "break", "continue",
		"return", "call", "try", "error", "log", "parallel", "comment",
	}

	for i, step := range program.Steps {
		nodeType := step.nodeType()
		if i < len(expectedTypes) && nodeType != expectedTypes[i] {
			t.Errorf("Step %d: expected type %s, got %s", i, expectedTypes[i], nodeType)
		}
	}
}

func TestASTExport_CommentPreserved(t *testing.T) {
	programJSON := `{
		"name": "with_comments",
		"go_json": "1",
		"steps": [
			{"_c": "Initialize counter", "let": "counter", "value": 0},
			{"_c": ["Multi-line comment", "Second line"], "set": "counter", "expr": "counter + 1"},
			{"return": "counter"}
		]
	}`

	program, err := Parse([]byte(programJSON))
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if len(program.Steps) != 3 {
		t.Fatalf("Expected 3 steps, got %d", len(program.Steps))
	}

	firstStep := program.Steps[0]
	meta := firstStep.Meta()
	if meta.Comment != "Initialize counter" {
		t.Errorf("Expected comment 'Initialize counter', got %s", meta.Comment)
	}

	secondStep := program.Steps[1]
	meta2 := secondStep.Meta()
	if len(meta2.Comments) != 2 {
		t.Errorf("Expected 2 comments in array, got %d", len(meta2.Comments))
	}
	if len(meta2.Comments) >= 2 {
		if meta2.Comments[0] != "Multi-line comment" {
			t.Errorf("Expected first comment 'Multi-line comment', got %s", meta2.Comments[0])
		}
		if meta2.Comments[1] != "Second line" {
			t.Errorf("Expected second comment 'Second line', got %s", meta2.Comments[1])
		}
	}

	astJSON, err := json.MarshalIndent(program, "", "  ")
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	if !json.Valid(astJSON) {
		t.Error("Generated AST JSON is not valid")
	}
}

func TestASTExport_FunctionsPreserved(t *testing.T) {
	programJSON := `{
		"name": "with_functions",
		"go_json": "1",
		"functions": {
			"greet": {
				"params": {"name": "string"},
				"returns": "string",
				"steps": [
					{"return": "'Hello, ' + name"}
				]
			},
			"add": {
				"params": {"a": "int", "b": "int"},
				"returns": "int",
				"steps": [
					{"return": "a + b"}
				]
			}
		},
		"steps": [
			{"let": "msg", "call": "greet", "with": {"name": "'World'"}},
			{"return": "msg"}
		]
	}`

	program, err := Parse([]byte(programJSON))
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if len(program.Functions) != 2 {
		t.Errorf("Expected 2 functions, got %d", len(program.Functions))
	}

	greetFunc, ok := program.Functions["greet"]
	if !ok {
		t.Fatal("Function 'greet' not found")
	}

	if greetFunc.Returns != "string" {
		t.Errorf("Expected greet returns 'string', got %s", greetFunc.Returns)
	}

	if len(greetFunc.Params) != 1 {
		t.Errorf("Expected 1 param, got %d", len(greetFunc.Params))
	}

	if greetFunc.Params[0].Name != "name" || greetFunc.Params[0].Type != "string" {
		t.Errorf("Expected param (name, string), got (%s, %s)", greetFunc.Params[0].Name, greetFunc.Params[0].Type)
	}

	astJSON, err := json.MarshalIndent(program, "", "  ")
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	var astMap map[string]any
	err = json.Unmarshal(astJSON, &astMap)
	if err != nil {
		t.Fatalf("Unmarshal to map failed: %v", err)
	}

	functions, ok := astMap["Functions"].(map[string]any)
	if !ok {
		t.Fatal("Functions not found in AST")
	}

	if len(functions) != 2 {
		t.Errorf("Expected 2 functions in AST, got %d", len(functions))
	}
}

func TestASTExport_ImportsPreserved(t *testing.T) {
	programJSON := `{
		"name": "with_imports",
		"go_json": "1",
		"import": {
			"http": "io:http",
			"fs": "io:fs",
			"utils": "./utils.json"
		},
		"steps": [
			{"let": "resp", "call": "http.get", "args": ["https://api.example.com"]},
			{"return": "resp.body"}
		]
	}`

	program, err := Parse([]byte(programJSON))
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if len(program.Imports) != 3 {
		t.Errorf("Expected 3 imports, got %d", len(program.Imports))
	}

	foundHTTP := false
	foundFS := false
	foundUtils := false

	for _, imp := range program.Imports {
		switch imp.Alias {
		case "http":
			foundHTTP = true
			if imp.Path != "io:http" {
				t.Errorf("Expected http path 'io:http', got %s", imp.Path)
			}
			if imp.PathType != "io" {
				t.Errorf("Expected http PathType 'io', got %s", imp.PathType)
			}
		case "fs":
			foundFS = true
			if imp.Path != "io:fs" {
				t.Errorf("Expected fs path 'io:fs', got %s", imp.Path)
			}
		case "utils":
			foundUtils = true
			if imp.Path != "./utils.json" {
				t.Errorf("Expected utils path './utils.json', got %s", imp.Path)
			}
			if imp.PathType != "relative" {
				t.Errorf("Expected utils PathType 'relative', got %s", imp.PathType)
			}
		}
	}

	if !foundHTTP || !foundFS || !foundUtils {
		t.Errorf("Not all imports found: http=%v, fs=%v, utils=%v", foundHTTP, foundFS, foundUtils)
	}

	astJSON, err := json.MarshalIndent(program, "", "  ")
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	var astMap map[string]any
	err = json.Unmarshal(astJSON, &astMap)
	if err != nil {
		t.Fatalf("Unmarshal to map failed: %v", err)
	}

	imports, ok := astMap["Imports"].([]any)
	if !ok {
		t.Fatal("Imports not found in AST")
	}

	if len(imports) != 3 {
		t.Errorf("Expected 3 imports in AST, got %d", len(imports))
	}
}

func TestASTExport_StructsPreserved(t *testing.T) {
	programJSON := `{
		"name": "with_structs",
		"go_json": "1",
		"structs": {
			"Person": {
				"fields": {
					"name": {"type": "string"},
					"age": {"type": "int", "default": 0}
				},
				"methods": {
					"greet": {
						"returns": "string",
						"steps": [
							{"return": "'Hello, ' + self.name"}
						]
					}
				}
			}
		},
		"steps": [
			{"let": "p", "new": "Person", "with": {"name": "Alice", "age": 30}},
			{"return": "p.greet()"}
		]
	}`

	program, err := Parse([]byte(programJSON))
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if len(program.Structs) != 1 {
		t.Errorf("Expected 1 struct, got %d", len(program.Structs))
	}

	personStruct, ok := program.Structs["Person"]
	if !ok {
		t.Fatal("Struct 'Person' not found")
	}

	if len(personStruct.Fields) != 2 {
		t.Errorf("Expected 2 fields, got %d", len(personStruct.Fields))
	}

	nameField, ok := personStruct.Fields["name"]
	if !ok {
		t.Fatal("Field 'name' not found")
	}

	if nameField.Type != "string" {
		t.Errorf("Expected name type 'string', got %s", nameField.Type)
	}

	ageField, ok := personStruct.Fields["age"]
	if !ok {
		t.Fatal("Field 'age' not found")
	}

	if !ageField.HasDefault || ageField.Default != float64(0) {
		t.Errorf("Expected age default=0, got %v (HasDefault=%v)", ageField.Default, ageField.HasDefault)
	}

	if len(personStruct.Methods) != 1 {
		t.Errorf("Expected 1 method, got %d", len(personStruct.Methods))
	}

	greetMethod, ok := personStruct.Methods["greet"]
	if !ok {
		t.Fatal("Method 'greet' not found")
	}

	if greetMethod.Returns != "string" {
		t.Errorf("Expected greet returns 'string', got %s", greetMethod.Returns)
	}

	astJSON, err := json.MarshalIndent(program, "", "  ")
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	var astMap map[string]any
	err = json.Unmarshal(astJSON, &astMap)
	if err != nil {
		t.Fatalf("Unmarshal to map failed: %v", err)
	}

	structs, ok := astMap["Structs"].(map[string]any)
	if !ok {
		t.Fatal("Structs not found in AST")
	}

	if len(structs) != 1 {
		t.Errorf("Expected 1 struct in AST, got %d", len(structs))
	}
}
