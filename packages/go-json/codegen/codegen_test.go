package codegen

import (
	"strings"
	"testing"

	"github.com/bitcode-framework/go-json/lang"
	"github.com/bitcode-framework/go-json/stdlib"
)

var factorialJSON = []byte(`{
	"name": "factorial",
	"functions": {
		"factorial": {
			"params": {"n": "int"},
			"returns": "int",
			"steps": [
				{"if": "n <= 1", "then": [{"return": "1"}]},
				{"let": "sub", "call": "factorial", "with": {"n": "n - 1"}},
				{"return": "n * sub"}
			]
		}
	},
	"steps": [
		{"let": "result", "call": "factorial", "with": {"n": "10"}},
		{"return": "result"}
	]
}`)

func compileTestProgram(t *testing.T) *lang.CompiledProgram {
	t.Helper()
	program, err := lang.Parse(factorialJSON)
	if err != nil {
		t.Fatalf("parse error: %s", err.Error())
	}

	engine := lang.NewExprLangEngine()
	reg := stdlib.DefaultRegistry()
	engine.AddOptions(reg.All()...)

	compiled, err := lang.Compile(program, engine, lang.DefaultLimits())
	if err != nil {
		t.Fatalf("compile error: %s", err.Error())
	}
	return compiled
}

func TestGoGenerator_Factorial(t *testing.T) {
	compiled := compileTestProgram(t)
	gen := &GoGenerator{PackageName: "main"}

	code, err := gen.Generate(compiled)
	if err != nil {
		t.Fatalf("generate error: %s", err.Error())
	}

	if !strings.Contains(code, "package main") {
		t.Error("expected 'package main'")
	}
	if !strings.Contains(code, "func factorial") {
		t.Error("expected 'func factorial'")
	}
	if !strings.Contains(code, "func main()") {
		t.Error("expected 'func main()'")
	}
	if !strings.Contains(code, "return") {
		t.Error("expected return statement")
	}
}

func TestGoGenerator_Language(t *testing.T) {
	gen := &GoGenerator{}
	if gen.Language() != "go" {
		t.Errorf("expected 'go', got %q", gen.Language())
	}
}

func TestJSGenerator_Factorial(t *testing.T) {
	compiled := compileTestProgram(t)
	gen := &JSGenerator{}

	code, err := gen.Generate(compiled)
	if err != nil {
		t.Fatalf("generate error: %s", err.Error())
	}

	if !strings.Contains(code, "function factorial") {
		t.Error("expected 'function factorial'")
	}
	if !strings.Contains(code, "return") {
		t.Error("expected return statement")
	}
}

func TestJSGenerator_Language(t *testing.T) {
	gen := &JSGenerator{}
	if gen.Language() != "javascript" {
		t.Errorf("expected 'javascript', got %q", gen.Language())
	}
}

func TestPythonGenerator_Factorial(t *testing.T) {
	compiled := compileTestProgram(t)
	gen := &PythonGenerator{}

	code, err := gen.Generate(compiled)
	if err != nil {
		t.Fatalf("generate error: %s", err.Error())
	}

	if !strings.Contains(code, "def factorial") {
		t.Error("expected 'def factorial'")
	}
	if !strings.Contains(code, "if __name__") {
		t.Error("expected 'if __name__' guard")
	}
	if !strings.Contains(code, "return") {
		t.Error("expected return statement")
	}
}

func TestPythonGenerator_Language(t *testing.T) {
	gen := &PythonGenerator{}
	if gen.Language() != "python" {
		t.Errorf("expected 'python', got %q", gen.Language())
	}
}

func TestTransformExpr_Python(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"a && b", "a  and  b"},
		{"a || b", "a  or  b"},
		{"true", "True"},
		{"false", "False"},
		{"nil", "None"},
	}

	for _, tt := range tests {
		result := transformExpr(tt.input, "python")
		if result != tt.expected {
			t.Errorf("transformExpr(%q, python) = %q, want %q", tt.input, result, tt.expected)
		}
	}
}

func TestGoTypeMap(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"string", "string"},
		{"int", "int64"},
		{"float", "float64"},
		{"bool", "bool"},
		{"any", "any"},
		{"[]string", "[]string"},
	}

	for _, tt := range tests {
		result := goTypeMap(tt.input)
		if result != tt.expected {
			t.Errorf("goTypeMap(%q) = %q, want %q", tt.input, result, tt.expected)
		}
	}
}

func TestPythonTypeMap(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"string", "str"},
		{"int", "int"},
		{"float", "float"},
		{"bool", "bool"},
		{"any", "Any"},
		{"", ""},
	}

	for _, tt := range tests {
		result := pythonTypeMap(tt.input)
		if result != tt.expected {
			t.Errorf("pythonTypeMap(%q) = %q, want %q", tt.input, result, tt.expected)
		}
	}
}

// Test data for comprehensive step type coverage
var allStepsJSON = []byte(`{
	"name": "all_steps",
	"go_json": "1",
	"functions": {
		"process": {
			"params": {"items": "[]any", "mode": "string"},
			"returns": "any",
			"steps": [
				{"_c": "Initialize counter"},
				{"let": "count", "value": 0},
				{"set": "count", "expr": "count + 1"},
				{"if": "mode == 'fast'", "then": [{"return": "'fast mode'"}], "elif": [{"condition": "mode == 'slow'", "then": [{"log": "'slow mode'"}]}], "else": [{"return": "'normal'"}]},
				{"switch": "mode", "cases": {"fast": [{"return": "'f'"}], "slow": [{"return": "'s'"}], "default": [{"return": "'d'"}]}},
				{"for": "item", "in": "items", "steps": [{"log": "item"}]},
				{"for": "i", "range": [0, 10], "steps": [{"if": "i > 5", "then": [{"break": true}]}]},
				{"while": "count < 100", "steps": [{"set": "count", "expr": "count + 1"}, {"if": "count == 50", "then": [{"continue": true}]}]},
				{"call": "process", "with": {"items": "[]", "mode": "'test'"}},
				{"try": [{"let": "x", "expr": "1 / 0"}], "catch": {"as": "err", "steps": [{"log": "err"}]}, "finally": [{"log": "'done'"}]},
				{"error": "'something went wrong'"},
				{"parallel": {"a": [{"let": "r1", "value": 1}], "b": [{"let": "r2", "value": 2}]}, "into": "results"},
				{"return": "count"}
			]
		}
	},
	"steps": [
		{"let": "result", "call": "process", "with": {"items": "[1, 2, 3]", "mode": "'fast'"}},
		{"return": "result"}
	]
}`)

var structJSON = []byte(`{
	"name": "struct_test",
	"go_json": "1",
	"structs": {
		"Point": {
			"fields": {"x": "int", "y": "int"}
		}
	},
	"steps": [
		{"let": "p", "new": "Point", "with": {"x": 10, "y": 20}},
		{"return": "p"}
	]
}`)

var commentJSON = []byte(`{
	"name": "comment_test",
	"go_json": "1",
	"steps": [
		{"_c": "This is a comment"},
		{"let": "x", "value": 42},
		{"_c": "Return the value"},
		{"return": "x"}
	]
}`)

func compileProgram(t *testing.T, data []byte) *lang.CompiledProgram {
	t.Helper()
	program, err := lang.Parse(data)
	if err != nil {
		t.Fatalf("parse error: %s", err.Error())
	}

	engine := lang.NewExprLangEngine()
	reg := stdlib.DefaultRegistry()
	engine.AddOptions(reg.All()...)

	compiled, err := lang.Compile(program, engine, lang.DefaultLimits())
	if err != nil {
		t.Fatalf("compile error: %s", err.Error())
	}
	return compiled
}

func TestGoGenerator_AllStepTypes(t *testing.T) {
	compiled := compileProgram(t, allStepsJSON)
	gen := &GoGenerator{PackageName: "main"}

	code, err := gen.Generate(compiled)
	if err != nil {
		t.Fatalf("generate error: %s", err.Error())
	}

	expectedConstructs := []string{
		"package main",
		"func process",
		"func main()",
		"count :=",
		"count =",
		"switch mode",
		"for",
		"break",
		"continue",
		"return",
	}

	for _, expected := range expectedConstructs {
		if !strings.Contains(code, expected) {
			t.Errorf("expected %q in generated code", expected)
		}
	}
}

func TestJSGenerator_AllStepTypes(t *testing.T) {
	compiled := compileProgram(t, allStepsJSON)
	gen := &JSGenerator{}

	code, err := gen.Generate(compiled)
	if err != nil {
		t.Fatalf("generate error: %s", err.Error())
	}

	expectedConstructs := []string{
		"function process",
		"count",
		"switch (mode)",
		"case",
		"for",
		"while",
		"return",
		"try {",
		"catch",
		"console.log",
	}

	for _, expected := range expectedConstructs {
		if !strings.Contains(code, expected) {
			t.Errorf("expected %q in generated code", expected)
		}
	}
}

func TestPythonGenerator_AllStepTypes(t *testing.T) {
	compiled := compileProgram(t, allStepsJSON)
	gen := &PythonGenerator{}

	code, err := gen.Generate(compiled)
	if err != nil {
		t.Fatalf("generate error: %s", err.Error())
	}

	expectedConstructs := []string{
		"def process",
		"count =",
		"if",
		"elif",
		"for item in items:",
		"for i in range(",
		"while",
		"break",
		"continue",
		"return",
		"try:",
		"except Exception",
		"raise Exception",
		"print(",
	}

	for _, expected := range expectedConstructs {
		if !strings.Contains(code, expected) {
			t.Errorf("expected %q in generated code", expected)
		}
	}
}

func TestGoGenerator_StructConstruction(t *testing.T) {
	compiled := compileProgram(t, structJSON)
	gen := &GoGenerator{PackageName: "main"}

	code, err := gen.Generate(compiled)
	if err != nil {
		t.Fatalf("generate error: %s", err.Error())
	}

	if !strings.Contains(code, "Point{") {
		t.Error("expected Point struct construction in generated code")
	}
}

func TestJSGenerator_StructConstruction(t *testing.T) {
	compiled := compileProgram(t, structJSON)
	gen := &JSGenerator{}

	code, err := gen.Generate(compiled)
	if err != nil {
		t.Fatalf("generate error: %s", err.Error())
	}

	if !strings.Contains(code, "new Point(") {
		t.Error("expected new Point() construction in generated code")
	}
}

func TestPythonGenerator_StructConstruction(t *testing.T) {
	compiled := compileProgram(t, structJSON)
	gen := &PythonGenerator{}

	code, err := gen.Generate(compiled)
	if err != nil {
		t.Fatalf("generate error: %s", err.Error())
	}

	if !strings.Contains(code, "Point(") {
		t.Error("expected Point() construction in generated code")
	}
}

func TestGoGenerator_Comments(t *testing.T) {
	compiled := compileProgram(t, commentJSON)
	gen := &GoGenerator{PackageName: "main"}

	code, err := gen.Generate(compiled)
	if err != nil {
		t.Fatalf("generate error: %s", err.Error())
	}

	// Verify comments are present
	expectedComments := []string{
		"// This is a comment",
		"// Return the value",
	}

	for _, expected := range expectedComments {
		if !strings.Contains(code, expected) {
			t.Errorf("expected comment %q in generated code", expected)
		}
	}
}

func TestJSGenerator_Comments(t *testing.T) {
	compiled := compileProgram(t, commentJSON)
	gen := &JSGenerator{}

	code, err := gen.Generate(compiled)
	if err != nil {
		t.Fatalf("generate error: %s", err.Error())
	}

	// Verify comments are present
	expectedComments := []string{
		"// This is a comment",
		"// Return the value",
	}

	for _, expected := range expectedComments {
		if !strings.Contains(code, expected) {
			t.Errorf("expected comment %q in generated code", expected)
		}
	}
}

func TestPythonGenerator_Comments(t *testing.T) {
	compiled := compileProgram(t, commentJSON)
	gen := &PythonGenerator{}

	code, err := gen.Generate(compiled)
	if err != nil {
		t.Fatalf("generate error: %s", err.Error())
	}

	// Verify comments are present
	expectedComments := []string{
		"# This is a comment",
		"# Return the value",
	}

	for _, expected := range expectedComments {
		if !strings.Contains(code, expected) {
			t.Errorf("expected comment %q in generated code", expected)
		}
	}
}
