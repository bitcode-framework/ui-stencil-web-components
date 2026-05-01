package lang

import (
	"math"
	"testing"

	"github.com/bitcode-framework/go-json/stdlib"
)

func compileAndRunWithStdlib(t *testing.T, jsonProgram string, input map[string]any) *ExecutionResult {
	t.Helper()
	program, err := Parse([]byte(jsonProgram))
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	reg := stdlib.DefaultRegistry()
	engine := NewExprLangEngine(reg.All()...)
	compiled, err := Compile(program, engine, DefaultLimits())
	if err != nil {
		t.Fatalf("compile error: %v", err)
	}

	if input == nil {
		input = make(map[string]any)
	}
	for k, v := range reg.EnvVars() {
		input[k] = v
	}

	vm := NewVM(compiled, engine)
	result, err := vm.Execute(input)
	if err != nil {
		t.Fatalf("execute error: %v", err)
	}
	return result
}

func TestLambda_SingleExpr(t *testing.T) {
	result := compileAndRun(t, `{
		"steps": [
			{"let": "double", "expr": "fn(x) => x * 2"},
			{"let": "result", "expr": "double(5)"},
			{"return": "result"}
		]
	}`, nil)
	if !numEq(result.Value, 10) {
		t.Fatalf("expected 10, got %v", result.Value)
	}
}

func TestLambda_MultiParam(t *testing.T) {
	result := compileAndRun(t, `{
		"steps": [
			{"let": "add", "expr": "fn(a, b) => a + b"},
			{"let": "result", "expr": "add(3, 7)"},
			{"return": "result"}
		]
	}`, nil)
	if !numEq(result.Value, 10) {
		t.Fatalf("expected 10, got %v", result.Value)
	}
}

func TestLambda_NoParams(t *testing.T) {
	result := compileAndRun(t, `{
		"steps": [
			{"let": "greet", "expr": "fn() => 'Hello!'"},
			{"let": "result", "expr": "greet()"},
			{"return": "result"}
		]
	}`, nil)
	if result.Value != "Hello!" {
		t.Fatalf("expected 'Hello!', got %v", result.Value)
	}
}

func TestLambda_ScopeCapture(t *testing.T) {
	result := compileAndRun(t, `{
		"steps": [
			{"let": "factor", "value": 3},
			{"let": "multiply", "expr": "fn(x) => x * factor"},
			{"set": "factor", "value": 10},
			{"let": "result", "expr": "multiply(5)"},
			{"return": "result"}
		]
	}`, nil)
	if !numEq(result.Value, 15) {
		t.Fatalf("expected 15 (captured factor=3), got %v", result.Value)
	}
}

func TestLambda_Composition(t *testing.T) {
	result := compileAndRun(t, `{
		"steps": [
			{"let": "addTax", "expr": "fn(p) => p * 1.1"},
			{"let": "discount", "expr": "fn(p) => p * 0.8"},
			{"let": "final", "expr": "fn(p) => discount(addTax(p))"},
			{"let": "result", "expr": "final(100)"},
			{"return": "result"}
		]
	}`, nil)
	f, ok := result.Value.(float64)
	if !ok {
		t.Fatalf("expected float64, got %T: %v", result.Value, result.Value)
	}
	if math.Abs(f-88.0) > 0.01 {
		t.Fatalf("expected ~88.0, got %v", f)
	}
}

func TestLambda_WithTernary(t *testing.T) {
	result := compileAndRun(t, `{
		"steps": [
			{"let": "classify", "expr": "fn(age) => age >= 18 ? 'adult' : 'minor'"},
			{"let": "r1", "expr": "classify(20)"},
			{"let": "r2", "expr": "classify(15)"},
			{"return": {"value": {"r1": "placeholder", "r2": "placeholder"}}}
		]
	}`, nil)
	_ = result

	r1 := compileAndRun(t, `{
		"steps": [
			{"let": "classify", "expr": "fn(age) => age >= 18 ? 'adult' : 'minor'"},
			{"let": "result", "expr": "classify(20)"},
			{"return": "result"}
		]
	}`, nil)
	if r1.Value != "adult" {
		t.Fatalf("expected 'adult', got %v", r1.Value)
	}

	r2 := compileAndRun(t, `{
		"steps": [
			{"let": "classify", "expr": "fn(age) => age >= 18 ? 'adult' : 'minor'"},
			{"let": "result", "expr": "classify(15)"},
			{"return": "result"}
		]
	}`, nil)
	if r2.Value != "minor" {
		t.Fatalf("expected 'minor', got %v", r2.Value)
	}
}

func TestLambda_WithInput(t *testing.T) {
	result := compileAndRun(t, `{
		"steps": [
			{"let": "greet", "expr": "fn(name) => 'Hello, ' + name + '!'"},
			{"let": "result", "expr": "greet(input.name)"},
			{"return": "result"}
		]
	}`, map[string]any{"name": "Alice"})
	if result.Value != "Hello, Alice!" {
		t.Fatalf("expected 'Hello, Alice!', got %v", result.Value)
	}
}

func TestLambda_CallingStdlib(t *testing.T) {
	result := compileAndRun(t, `{
		"steps": [
			{"let": "normalize", "expr": "fn(s) => lower(trim(s))"},
			{"let": "result", "expr": "normalize('  HELLO  ')"},
			{"return": "result"}
		]
	}`, nil)
	if result.Value != "hello" {
		t.Fatalf("expected 'hello', got %v", result.Value)
	}
}

func TestLambda_ReturnsFunction(t *testing.T) {
	result := compileAndRun(t, `{
		"steps": [
			{"let": "double", "expr": "fn(x) => x * 2"},
			{"let": "result", "expr": "double(21)"},
			{"return": "result"}
		]
	}`, nil)
	if !numEq(result.Value, 42) {
		t.Fatalf("expected 42, got %v", result.Value)
	}
}

func TestLambda_MultipleCallsSameFunction(t *testing.T) {
	result := compileAndRun(t, `{
		"steps": [
			{"let": "inc", "expr": "fn(x) => x + 1"},
			{"let": "a", "expr": "inc(1)"},
			{"let": "b", "expr": "inc(10)"},
			{"let": "c", "expr": "inc(100)"},
			{"return": "a + b + c"}
		]
	}`, nil)
	if !numEq(result.Value, 114) {
		t.Fatalf("expected 114, got %v", result.Value)
	}
}

func TestLambda_HigherOrder_MapFn(t *testing.T) {
	result := compileAndRunWithStdlib(t, `{
		"steps": [
			{"let": "double", "expr": "fn(x) => x * 2"},
			{"let": "result", "expr": "mapFn([1, 2, 3], double)"},
			{"return": "result"}
		]
	}`, nil)
	arr, ok := result.Value.([]any)
	if !ok {
		t.Fatalf("expected []any, got %T", result.Value)
	}
	if len(arr) != 3 || !numEq(arr[0], 2) || !numEq(arr[1], 4) || !numEq(arr[2], 6) {
		t.Fatalf("expected [2, 4, 6], got %v", arr)
	}
}

func TestLambda_HigherOrder_MapFn_Inline(t *testing.T) {
	result := compileAndRunWithStdlib(t, `{
		"steps": [
			{"let": "result", "expr": "mapFn([1, 2, 3, 4], fn(x) => x * x)"},
			{"return": "result"}
		]
	}`, nil)
	arr, ok := result.Value.([]any)
	if !ok {
		t.Fatalf("expected []any, got %T", result.Value)
	}
	if len(arr) != 4 || !numEq(arr[0], 1) || !numEq(arr[1], 4) || !numEq(arr[2], 9) || !numEq(arr[3], 16) {
		t.Fatalf("expected [1, 4, 9, 16], got %v", arr)
	}
}

func TestLambda_HigherOrder_FilterFn(t *testing.T) {
	result := compileAndRunWithStdlib(t, `{
		"steps": [
			{"let": "result", "expr": "filterFn([1, 2, 3, 4, 5, 6], fn(x) => x % 2 == 0)"},
			{"return": "result"}
		]
	}`, nil)
	arr, ok := result.Value.([]any)
	if !ok {
		t.Fatalf("expected []any, got %T", result.Value)
	}
	if len(arr) != 3 || !numEq(arr[0], 2) || !numEq(arr[1], 4) || !numEq(arr[2], 6) {
		t.Fatalf("expected [2, 4, 6], got %v", arr)
	}
}

func TestLambda_HigherOrder_ReduceFn(t *testing.T) {
	result := compileAndRunWithStdlib(t, `{
		"steps": [
			{"let": "total", "expr": "reduceFn([1, 2, 3, 4, 5], fn(acc, x) => acc + x, 0)"},
			{"return": "total"}
		]
	}`, nil)
	if !numEq(result.Value, 15) {
		t.Fatalf("expected 15, got %v", result.Value)
	}
}

func TestLambda_HigherOrder_ReduceFn_NoInitial(t *testing.T) {
	result := compileAndRunWithStdlib(t, `{
		"steps": [
			{"let": "total", "expr": "reduceFn([10, 20, 30], fn(acc, x) => acc + x)"},
			{"return": "total"}
		]
	}`, nil)
	if !numEq(result.Value, 60) {
		t.Fatalf("expected 60, got %v", result.Value)
	}
}

func TestLambda_HigherOrder_SortFn(t *testing.T) {
	result := compileAndRunWithStdlib(t, `{
		"steps": [
			{"let": "result", "expr": "sortFn([3, 1, 4, 1, 5], fn(a, b) => a < b)"},
			{"return": "result"}
		]
	}`, nil)
	arr, ok := result.Value.([]any)
	if !ok {
		t.Fatalf("expected []any, got %T", result.Value)
	}
	if len(arr) != 5 || !numEq(arr[0], 1) || !numEq(arr[1], 1) || !numEq(arr[2], 3) || !numEq(arr[3], 4) || !numEq(arr[4], 5) {
		t.Fatalf("expected [1, 1, 3, 4, 5], got %v", arr)
	}
}

func TestLambda_HigherOrder_FilterThenMap(t *testing.T) {
	result := compileAndRunWithStdlib(t, `{
		"steps": [
			{"let": "isEven", "expr": "fn(x) => x % 2 == 0"},
			{"let": "double", "expr": "fn(x) => x * 2"},
			{"let": "evens", "expr": "filterFn([1, 2, 3, 4, 5, 6], isEven)"},
			{"let": "result", "expr": "mapFn(evens, double)"},
			{"return": "result"}
		]
	}`, nil)
	arr, ok := result.Value.([]any)
	if !ok {
		t.Fatalf("expected []any, got %T", result.Value)
	}
	if len(arr) != 3 || !numEq(arr[0], 4) || !numEq(arr[1], 8) || !numEq(arr[2], 12) {
		t.Fatalf("expected [4, 8, 12], got %v", arr)
	}
}

func TestLambda_HigherOrder_RejectFn(t *testing.T) {
	result := compileAndRunWithStdlib(t, `{
		"steps": [
			{"let": "odds", "expr": "rejectFn([1, 2, 3, 4, 5, 6], fn(x) => x % 2 == 0)"},
			{"return": "odds"}
		]
	}`, nil)
	arr, ok := result.Value.([]any)
	if !ok {
		t.Fatalf("expected []any, got %T", result.Value)
	}
	if len(arr) != 3 || !numEq(arr[0], 1) || !numEq(arr[1], 3) || !numEq(arr[2], 5) {
		t.Fatalf("expected [1, 3, 5], got %v", arr)
	}
}

func TestLambda_HigherOrder_TakeWhileFn(t *testing.T) {
	result := compileAndRunWithStdlib(t, `{
		"steps": [
			{"let": "result", "expr": "takeWhileFn([1, 2, 3, 5, 1], fn(x) => x < 4)"},
			{"return": "result"}
		]
	}`, nil)
	arr, ok := result.Value.([]any)
	if !ok {
		t.Fatalf("expected []any, got %T", result.Value)
	}
	if len(arr) != 3 || !numEq(arr[0], 1) || !numEq(arr[1], 2) || !numEq(arr[2], 3) {
		t.Fatalf("expected [1, 2, 3], got %v", arr)
	}
}

func TestLambda_HigherOrder_DropWhileFn(t *testing.T) {
	result := compileAndRunWithStdlib(t, `{
		"steps": [
			{"let": "result", "expr": "dropWhileFn([1, 2, 3, 5, 1], fn(x) => x < 4)"},
			{"return": "result"}
		]
	}`, nil)
	arr, ok := result.Value.([]any)
	if !ok {
		t.Fatalf("expected []any, got %T", result.Value)
	}
	if len(arr) != 2 || !numEq(arr[0], 5) || !numEq(arr[1], 1) {
		t.Fatalf("expected [5, 1], got %v", arr)
	}
}

func TestLambda_HigherOrder_FindFn(t *testing.T) {
	result := compileAndRunWithStdlib(t, `{
		"steps": [
			{"let": "found", "expr": "findFn([1, 2, 3, 4, 5], fn(x) => x > 3)"},
			{"return": "found"}
		]
	}`, nil)
	if !numEq(result.Value, 4) {
		t.Fatalf("expected 4, got %v", result.Value)
	}
}

func TestLambda_HigherOrder_EveryFn(t *testing.T) {
	result := compileAndRunWithStdlib(t, `{
		"steps": [
			{"let": "allPositive", "expr": "everyFn([1, 2, 3], fn(x) => x > 0)"},
			{"return": "allPositive"}
		]
	}`, nil)
	if result.Value != true {
		t.Fatalf("expected true, got %v", result.Value)
	}
}

func TestLambda_HigherOrder_SomeFn(t *testing.T) {
	result := compileAndRunWithStdlib(t, `{
		"steps": [
			{"let": "hasNeg", "expr": "someFn([1, -2, 3], fn(x) => x < 0)"},
			{"return": "hasNeg"}
		]
	}`, nil)
	if result.Value != true {
		t.Fatalf("expected true, got %v", result.Value)
	}
}

func TestLambda_HigherOrder_PartitionFn(t *testing.T) {
	result := compileAndRunWithStdlib(t, `{
		"steps": [
			{"let": "parts", "expr": "partitionFn([1, 2, 3, 4, 5, 6], fn(x) => x % 2 == 0)"},
			{"return": "parts"}
		]
	}`, nil)
	arr, ok := result.Value.([]any)
	if !ok {
		t.Fatalf("expected []any, got %T", result.Value)
	}
	if len(arr) != 2 {
		t.Fatalf("expected 2 partitions, got %d", len(arr))
	}
	evens := arr[0].([]any)
	odds := arr[1].([]any)
	if len(evens) != 3 {
		t.Fatalf("expected 3 evens, got %v", evens)
	}
	if len(odds) != 3 {
		t.Fatalf("expected 3 odds, got %v", odds)
	}
}
