package main

import (
	"encoding/json"
	"math"
	"os"
	"path/filepath"
	"testing"
)

func TestDeepEqual_Floats(t *testing.T) {
	if !deepEqual(0.3, 0.1+0.2) {
		t.Error("float tolerance should handle 0.1+0.2 == 0.3")
	}

	if !deepEqual(1.0, 1.0) {
		t.Error("equal floats should match")
	}

	if deepEqual(1.0, 2.0) {
		t.Error("different floats should not match")
	}

	if !deepEqual(math.Pi, math.Pi) {
		t.Error("Pi should equal Pi")
	}
}

func TestDeepEqual_Ints(t *testing.T) {
	if !deepEqual(42, 42) {
		t.Error("equal ints should match")
	}
	if deepEqual(42, 43) {
		t.Error("different ints should not match")
	}

	if !deepEqual(0, 0) {
		t.Error("zero should equal zero")
	}

	if deepEqual(-1, 1) {
		t.Error("negative and positive should not match")
	}
}

func TestDeepEqual_IntAndFloat(t *testing.T) {
	if !deepEqual(42, 42.0) {
		t.Error("int 42 should equal float 42.0")
	}

	if !deepEqual(42.0, 42) {
		t.Error("float 42.0 should equal int 42")
	}

	if deepEqual(42, 42.1) {
		t.Error("int 42 should not equal float 42.1")
	}
}

func TestDeepEqual_Strings(t *testing.T) {
	if !deepEqual("hello", "hello") {
		t.Error("equal strings should match")
	}

	if deepEqual("hello", "world") {
		t.Error("different strings should not match")
	}

	if !deepEqual("", "") {
		t.Error("empty strings should match")
	}
}

func TestDeepEqual_Bools(t *testing.T) {
	if !deepEqual(true, true) {
		t.Error("true should equal true")
	}

	if !deepEqual(false, false) {
		t.Error("false should equal false")
	}

	if deepEqual(true, false) {
		t.Error("true should not equal false")
	}
}

func TestDeepEqual_Maps(t *testing.T) {
	a := map[string]any{"name": "Alice", "age": float64(30)}
	b := map[string]any{"name": "Alice", "age": float64(30)}
	if !deepEqual(a, b) {
		t.Error("equal maps should match")
	}

	c := map[string]any{"name": "Bob", "age": float64(30)}
	if deepEqual(a, c) {
		t.Error("maps with different values should not match")
	}

	d := map[string]any{"name": "Alice"}
	if deepEqual(a, d) {
		t.Error("maps with different keys should not match")
	}
}

func TestDeepEqual_NestedMaps(t *testing.T) {
	a := map[string]any{
		"user": map[string]any{
			"name": "Alice",
			"age":  float64(30),
		},
	}
	b := map[string]any{
		"user": map[string]any{
			"name": "Alice",
			"age":  float64(30),
		},
	}
	if !deepEqual(a, b) {
		t.Error("equal nested maps should match")
	}

	c := map[string]any{
		"user": map[string]any{
			"name": "Bob",
			"age":  float64(30),
		},
	}
	if deepEqual(a, c) {
		t.Error("nested maps with different values should not match")
	}
}

func TestDeepEqual_Arrays(t *testing.T) {
	a := []any{1.0, 2.0, 3.0}
	b := []any{1.0, 2.0, 3.0}
	if !deepEqual(a, b) {
		t.Error("equal arrays should match")
	}

	c := []any{1.0, 2.0, 4.0}
	if deepEqual(a, c) {
		t.Error("arrays with different values should not match")
	}

	d := []any{1.0, 2.0}
	if deepEqual(a, d) {
		t.Error("arrays with different lengths should not match")
	}
}

func TestDeepEqual_ArrayOrder(t *testing.T) {
	a := []any{1.0, 2.0, 3.0}
	b := []any{3.0, 2.0, 1.0}
	if deepEqual(a, b) {
		t.Error("arrays with different order should not match")
	}
}

func TestDeepEqual_NestedArrays(t *testing.T) {
	a := []any{
		[]any{1.0, 2.0},
		[]any{3.0, 4.0},
	}
	b := []any{
		[]any{1.0, 2.0},
		[]any{3.0, 4.0},
	}
	if !deepEqual(a, b) {
		t.Error("equal nested arrays should match")
	}

	c := []any{
		[]any{1.0, 2.0},
		[]any{3.0, 5.0},
	}
	if deepEqual(a, c) {
		t.Error("nested arrays with different values should not match")
	}
}

func TestDeepEqual_Nil(t *testing.T) {
	if !deepEqual(nil, nil) {
		t.Error("nil == nil should be true")
	}
	if deepEqual(nil, "something") {
		t.Error("nil != something")
	}
	if deepEqual("something", nil) {
		t.Error("something != nil")
	}
	if deepEqual(nil, 0) {
		t.Error("nil != 0")
	}
}

func TestDeepEqual_MixedTypes(t *testing.T) {
	if deepEqual("42", 42) {
		t.Error("string '42' should not equal int 42")
	}

	if deepEqual(true, 1) {
		t.Error("bool true should not equal int 1")
	}

	if deepEqual([]any{1, 2}, map[string]any{"0": 1, "1": 2}) {
		t.Error("array should not equal map")
	}
}

func TestDeepEqual_ComplexStructure(t *testing.T) {
	a := map[string]any{
		"users": []any{
			map[string]any{"name": "Alice", "age": float64(30)},
			map[string]any{"name": "Bob", "age": float64(25)},
		},
		"count": float64(2),
	}
	b := map[string]any{
		"users": []any{
			map[string]any{"name": "Alice", "age": float64(30)},
			map[string]any{"name": "Bob", "age": float64(25)},
		},
		"count": float64(2),
	}
	if !deepEqual(a, b) {
		t.Error("equal complex structures should match")
	}
}

func TestTestRunner_FindTestFiles(t *testing.T) {
	dir := t.TempDir()

	testFile := `{"name": "my_test", "test": true, "import": {"calc": "./calc.json"}, "cases": [{"_c": "basic", "call": "calc.add", "with": {"a": "1", "b": "2"}, "expect": 3}]}`
	if err := os.WriteFile(filepath.Join(dir, "my_test.json"), []byte(testFile), 0644); err != nil {
		t.Fatal(err)
	}

	nonTest := `{"name": "not_a_test", "steps": []}`
	if err := os.WriteFile(filepath.Join(dir, "other.json"), []byte(nonTest), 0644); err != nil {
		t.Fatal(err)
	}

	files := findTestFiles(dir)
	if len(files) != 1 {
		t.Errorf("expected 1 test file, got %d", len(files))
	}

	if len(files) > 0 && !filepath.IsAbs(files[0]) {
		t.Error("expected absolute path")
	}
}

func TestTestRunner_FindTestFiles_Nested(t *testing.T) {
	dir := t.TempDir()
	subdir := filepath.Join(dir, "subdir")
	if err := os.MkdirAll(subdir, 0755); err != nil {
		t.Fatal(err)
	}

	testFile1 := `{"name": "test1", "test": true, "cases": []}`
	if err := os.WriteFile(filepath.Join(dir, "test1.json"), []byte(testFile1), 0644); err != nil {
		t.Fatal(err)
	}

	testFile2 := `{"name": "test2", "test": true, "cases": []}`
	if err := os.WriteFile(filepath.Join(subdir, "test2.json"), []byte(testFile2), 0644); err != nil {
		t.Fatal(err)
	}

	files := findTestFiles(dir)
	if len(files) != 2 {
		t.Errorf("expected 2 test files (including nested), got %d", len(files))
	}
}

func TestTestRunner_FindTestFiles_EmptyDir(t *testing.T) {
	dir := t.TempDir()
	files := findTestFiles(dir)
	if len(files) != 0 {
		t.Errorf("expected 0 test files in empty dir, got %d", len(files))
	}
}

func TestTestRunner_ParseTestFile(t *testing.T) {
	data := []byte(`{"name": "test_math", "test": true, "import": {"m": "./math.json"}, "cases": [{"_c": "add", "call": "m.add", "with": {"a": "1", "b": "2"}, "expect": 3}]}`)

	var tf testFile
	if err := json.Unmarshal(data, &tf); err != nil {
		t.Fatal(err)
	}
	if tf.Name != "test_math" {
		t.Errorf("expected name 'test_math', got %q", tf.Name)
	}
	if !tf.Test {
		t.Error("expected test=true")
	}
	if len(tf.Cases) != 1 {
		t.Errorf("expected 1 case, got %d", len(tf.Cases))
	}
	if tf.Cases[0].Call != "m.add" {
		t.Errorf("expected call 'm.add', got %q", tf.Cases[0].Call)
	}
	if tf.Cases[0].Comment != "add" {
		t.Errorf("expected comment 'add', got %q", tf.Cases[0].Comment)
	}
}

func TestTestRunner_ParseTestFile_MultipleCases(t *testing.T) {
	data := []byte(`{
		"name": "test_math",
		"test": true,
		"import": {"m": "./math.json"},
		"cases": [
			{"_c": "add", "call": "m.add", "with": {"a": "1", "b": "2"}, "expect": 3},
			{"_c": "subtract", "call": "m.sub", "with": {"a": "5", "b": "3"}, "expect": 2}
		]
	}`)

	var tf testFile
	if err := json.Unmarshal(data, &tf); err != nil {
		t.Fatal(err)
	}
	if len(tf.Cases) != 2 {
		t.Errorf("expected 2 cases, got %d", len(tf.Cases))
	}
}

func TestTestRunner_ParseTestFile_EmptyCases(t *testing.T) {
	data := []byte(`{"name": "empty_test", "test": true, "cases": []}`)

	var tf testFile
	if err := json.Unmarshal(data, &tf); err != nil {
		t.Fatal(err)
	}
	if len(tf.Cases) != 0 {
		t.Errorf("expected 0 cases, got %d", len(tf.Cases))
	}
}

func TestTestRunner_ParseTestFile_NoImport(t *testing.T) {
	data := []byte(`{"name": "no_import", "test": true, "cases": []}`)

	var tf testFile
	if err := json.Unmarshal(data, &tf); err != nil {
		t.Fatal(err)
	}
	if tf.Import != nil && len(tf.Import) > 0 {
		t.Error("expected nil or empty import map")
	}
}

func TestBuildTestWrapper(t *testing.T) {
	with := map[string]any{
		"a": "1 + 1",
		"b": float64(2),
	}
	wrapper := buildTestWrapper("./math.json", "add", with)

	if wrapper == "" {
		t.Fatal("expected non-empty wrapper")
	}

	var parsed map[string]any
	if err := json.Unmarshal([]byte(wrapper), &parsed); err != nil {
		t.Fatalf("wrapper should be valid JSON: %v", err)
	}

	imports, ok := parsed["import"].(map[string]any)
	if !ok || imports["_m"] != "./math.json" {
		t.Error("expected import with alias '_m'")
	}

	steps, ok := parsed["steps"].([]any)
	if !ok || len(steps) != 2 {
		t.Error("expected 2 steps in wrapper")
	}
}

func TestToFloat_Float64(t *testing.T) {
	f, ok := toFloat(42.5)
	if !ok {
		t.Error("expected float64 to convert")
	}
	if f != 42.5 {
		t.Errorf("expected 42.5, got %v", f)
	}
}

func TestToFloat_Int(t *testing.T) {
	f, ok := toFloat(42)
	if !ok {
		t.Error("expected int to convert")
	}
	if f != 42.0 {
		t.Errorf("expected 42.0, got %v", f)
	}
}

func TestToFloat_Int64(t *testing.T) {
	f, ok := toFloat(int64(42))
	if !ok {
		t.Error("expected int64 to convert")
	}
	if f != 42.0 {
		t.Errorf("expected 42.0, got %v", f)
	}
}

func TestToFloat_String(t *testing.T) {
	_, ok := toFloat("42")
	if ok {
		t.Error("expected string to not convert")
	}
}

func TestToFloat_Nil(t *testing.T) {
	_, ok := toFloat(nil)
	if ok {
		t.Error("expected nil to not convert")
	}
}

func TestToFloat_JSONNumber(t *testing.T) {
	num := json.Number("42.5")
	f, ok := toFloat(num)
	if !ok {
		t.Error("expected json.Number to convert")
	}
	if f != 42.5 {
		t.Errorf("expected 42.5, got %v", f)
	}
}

func TestToFloat_InvalidJSONNumber(t *testing.T) {
	num := json.Number("not-a-number")
	_, ok := toFloat(num)
	if ok {
		t.Error("expected invalid json.Number to not convert")
	}
}

func TestDeepEqual_FloatTolerance(t *testing.T) {
	tests := []struct {
		a, b     float64
		expected bool
	}{
		{0.1 + 0.2, 0.3, true},
		{1.0000000001, 1.0, true},
		{1.0, 1.0000000001, true},
		{1.0, 1.1, false},
		{0.0, 0.0, true},
		{-0.1 - 0.2, -0.3, true},
	}

	for _, tt := range tests {
		result := deepEqual(tt.a, tt.b)
		if result != tt.expected {
			t.Errorf("deepEqual(%v, %v) = %v, expected %v", tt.a, tt.b, result, tt.expected)
		}
	}
}

func TestDeepEqual_JSONSerialization(t *testing.T) {
	a := map[string]any{
		"name": "Alice",
		"tags": []any{"admin", "user"},
	}
	b := map[string]any{
		"name": "Alice",
		"tags": []any{"admin", "user"},
	}

	if !deepEqual(a, b) {
		t.Error("equal structures should match via JSON serialization")
	}

	c := map[string]any{
		"name": "Alice",
		"tags": []any{"user", "admin"},
	}

	if deepEqual(a, c) {
		t.Error("structures with different array order should not match")
	}
}
