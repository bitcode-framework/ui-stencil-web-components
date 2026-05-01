package stdlib

import (
	"testing"

	"github.com/expr-lang/expr"
)

func evalHO(t *testing.T, expression string, env map[string]any) any {
	t.Helper()
	reg := DefaultRegistry()
	opts := reg.All()
	mergedEnv := make(map[string]any)
	for k, v := range reg.EnvVars() {
		mergedEnv[k] = v
	}
	for k, v := range env {
		mergedEnv[k] = v
	}
	opts = append(opts, expr.Env(mergedEnv))
	program, err := expr.Compile(expression, opts...)
	if err != nil {
		t.Fatalf("compile error: %v", err)
	}
	result, err := expr.Run(program, mergedEnv)
	if err != nil {
		t.Fatalf("run error: %v", err)
	}
	return result
}

func TestRejectFn(t *testing.T) {
	isEven := func(args ...any) (any, error) {
		n, _ := toFloat64(args[0])
		return int(n)%2 == 0, nil
	}
	env := map[string]any{
		"arr":    []any{1, 2, 3, 4, 5, 6},
		"isEven": isEven,
	}
	result := evalHO(t, "rejectFn(arr, isEven)", env)
	arr := result.([]any)
	if len(arr) != 3 {
		t.Fatalf("expected 3 items, got %d: %v", len(arr), arr)
	}
}

func TestTakeWhileFn(t *testing.T) {
	lessThan4 := func(args ...any) (any, error) {
		n, _ := toFloat64(args[0])
		return n < 4, nil
	}
	env := map[string]any{
		"arr":      []any{1, 2, 3, 5, 1, 2},
		"lessThan4": lessThan4,
	}
	result := evalHO(t, "takeWhileFn(arr, lessThan4)", env)
	arr := result.([]any)
	if len(arr) != 3 {
		t.Fatalf("expected [1,2,3], got %v", arr)
	}
}

func TestDropWhileFn(t *testing.T) {
	lessThan4 := func(args ...any) (any, error) {
		n, _ := toFloat64(args[0])
		return n < 4, nil
	}
	env := map[string]any{
		"arr":      []any{1, 2, 3, 5, 1, 2},
		"lessThan4": lessThan4,
	}
	result := evalHO(t, "dropWhileFn(arr, lessThan4)", env)
	arr := result.([]any)
	if len(arr) != 3 {
		t.Fatalf("expected [5,1,2], got %v", arr)
	}
}

func TestFindFn(t *testing.T) {
	greaterThan3 := func(args ...any) (any, error) {
		n, _ := toFloat64(args[0])
		return n > 3, nil
	}
	env := map[string]any{
		"arr":         []any{1, 2, 3, 4, 5},
		"greaterThan3": greaterThan3,
	}
	result := evalHO(t, "findFn(arr, greaterThan3)", env)
	n, _ := toFloat64(result)
	if n != 4 {
		t.Fatalf("expected 4, got %v", result)
	}
}

func TestFindFn_NotFound(t *testing.T) {
	greaterThan10 := func(args ...any) (any, error) {
		n, _ := toFloat64(args[0])
		return n > 10, nil
	}
	env := map[string]any{
		"arr":          []any{1, 2, 3},
		"greaterThan10": greaterThan10,
	}
	result := evalHO(t, "findFn(arr, greaterThan10)", env)
	if result != nil {
		t.Fatalf("expected nil, got %v", result)
	}
}

func TestEveryFn(t *testing.T) {
	positive := func(args ...any) (any, error) {
		n, _ := toFloat64(args[0])
		return n > 0, nil
	}
	env := map[string]any{
		"arr":      []any{1, 2, 3},
		"positive": positive,
	}
	result := evalHO(t, "everyFn(arr, positive)", env)
	if result != true {
		t.Fatalf("expected true, got %v", result)
	}

	env["arr"] = []any{1, -2, 3}
	result = evalHO(t, "everyFn(arr, positive)", env)
	if result != false {
		t.Fatalf("expected false, got %v", result)
	}
}

func TestSomeFn(t *testing.T) {
	negative := func(args ...any) (any, error) {
		n, _ := toFloat64(args[0])
		return n < 0, nil
	}
	env := map[string]any{
		"arr":      []any{1, 2, -3},
		"negative": negative,
	}
	result := evalHO(t, "someFn(arr, negative)", env)
	if result != true {
		t.Fatalf("expected true, got %v", result)
	}

	env["arr"] = []any{1, 2, 3}
	result = evalHO(t, "someFn(arr, negative)", env)
	if result != false {
		t.Fatalf("expected false, got %v", result)
	}
}

func TestPartitionFn(t *testing.T) {
	isEven := func(args ...any) (any, error) {
		n, _ := toFloat64(args[0])
		return int(n)%2 == 0, nil
	}
	env := map[string]any{
		"arr":    []any{1, 2, 3, 4, 5, 6},
		"isEven": isEven,
	}
	result := evalHO(t, "partitionFn(arr, isEven)", env)
	parts := result.([]any)
	if len(parts) != 2 {
		t.Fatalf("expected 2 partitions, got %d", len(parts))
	}
	matches := parts[0].([]any)
	nonMatches := parts[1].([]any)
	if len(matches) != 3 {
		t.Fatalf("expected 3 matches (evens), got %d: %v", len(matches), matches)
	}
	if len(nonMatches) != 3 {
		t.Fatalf("expected 3 non-matches (odds), got %d: %v", len(nonMatches), nonMatches)
	}
}

func TestIdentity(t *testing.T) {
	env := map[string]any{"x": 42}
	result := evalHO(t, "identity(x)", env)
	n, _ := toFloat64(result)
	if n != 42 {
		t.Fatalf("expected 42, got %v", result)
	}
}

func TestIdentity_Nil(t *testing.T) {
	env := map[string]any{"x": nil}
	result := evalHO(t, "identity(x)", env)
	if result != nil {
		t.Fatalf("expected nil, got %v", result)
	}
}
