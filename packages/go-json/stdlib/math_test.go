package stdlib

import (
	"testing"

	"github.com/expr-lang/expr"
)

func evalExpr(t *testing.T, expression string, env map[string]any) any {
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
		t.Fatalf("compile error for %q: %v", expression, err)
	}
	result, err := expr.Run(program, mergedEnv)
	if err != nil {
		t.Fatalf("run error for %q: %v", expression, err)
	}
	return result
}

func evalExprError(t *testing.T, expression string, env map[string]any) error {
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
		return err
	}
	_, err = expr.Run(program, mergedEnv)
	return err
}

func TestClamp_BelowMin(t *testing.T) {
	result := evalExpr(t, "clamp(5, 10, 20)", nil)
	if toF64(result) != 10 {
		t.Errorf("expected 10, got %v", result)
	}
}

func TestClamp_AboveMax(t *testing.T) {
	result := evalExpr(t, "clamp(25, 10, 20)", nil)
	if toF64(result) != 20 {
		t.Errorf("expected 20, got %v", result)
	}
}

func TestClamp_InRange(t *testing.T) {
	result := evalExpr(t, "clamp(15, 10, 20)", nil)
	if toF64(result) != 15 {
		t.Errorf("expected 15, got %v", result)
	}
}

func TestClamp_AtBoundaries(t *testing.T) {
	result := evalExpr(t, "clamp(10, 10, 20)", nil)
	if toF64(result) != 10 {
		t.Errorf("expected 10, got %v", result)
	}
	result = evalExpr(t, "clamp(20, 10, 20)", nil)
	if toF64(result) != 20 {
		t.Errorf("expected 20, got %v", result)
	}
}

func TestSign_Negative(t *testing.T) {
	result := evalExpr(t, "sign(-5)", nil)
	if result != -1 {
		t.Errorf("expected -1, got %v", result)
	}
}

func TestSign_Zero(t *testing.T) {
	result := evalExpr(t, "sign(0)", nil)
	if result != 0 {
		t.Errorf("expected 0, got %v", result)
	}
}

func TestSign_Positive(t *testing.T) {
	result := evalExpr(t, "sign(5)", nil)
	if result != 1 {
		t.Errorf("expected 1, got %v", result)
	}
}

func TestRandomInt_SameMinMax(t *testing.T) {
	result := evalExpr(t, "randomInt(10, 10)", nil)
	if result != 10 {
		t.Errorf("expected 10, got %v", result)
	}
}

func TestRandomInt_InRange(t *testing.T) {
	for i := 0; i < 50; i++ {
		result := evalExpr(t, "randomInt(1, 100)", nil)
		n, ok := result.(int)
		if !ok {
			t.Fatalf("expected int, got %T: %v", result, result)
		}
		if n < 1 || n > 100 {
			t.Errorf("randomInt(1, 100) = %d, out of range [1, 100]", n)
		}
	}
}

func TestRandomFloat_InRange(t *testing.T) {
	for i := 0; i < 50; i++ {
		result := evalExpr(t, "randomFloat(0, 1)", nil)
		f := toF64(result)
		if f < 0 || f > 1 {
			t.Errorf("randomFloat(0, 1) = %f, out of range [0, 1]", f)
		}
	}
}

func TestRandomFloat_CustomRange(t *testing.T) {
	for i := 0; i < 50; i++ {
		result := evalExpr(t, "randomFloat(10, 20)", nil)
		f := toF64(result)
		if f < 10 || f > 20 {
			t.Errorf("randomFloat(10, 20) = %f, out of range [10, 20]", f)
		}
	}
}

func TestPow_PositiveExponent(t *testing.T) {
	result := evalExpr(t, "pow(2, 3)", nil)
	if toF64(result) != 8 {
		t.Errorf("expected 8, got %v", result)
	}
}

func TestPow_NegativeExponent(t *testing.T) {
	result := evalExpr(t, "pow(2, -1)", nil)
	if toF64(result) != 0.5 {
		t.Errorf("expected 0.5, got %v", result)
	}
}

func TestPow_ZeroExponent(t *testing.T) {
	result := evalExpr(t, "pow(5, 0)", nil)
	if toF64(result) != 1 {
		t.Errorf("expected 1, got %v", result)
	}
}

func TestSqrt_PerfectSquare(t *testing.T) {
	result := evalExpr(t, "sqrt(9)", nil)
	if toF64(result) != 3 {
		t.Errorf("expected 3, got %v", result)
	}
}

func TestSqrt_Zero(t *testing.T) {
	result := evalExpr(t, "sqrt(0)", nil)
	if toF64(result) != 0 {
		t.Errorf("expected 0, got %v", result)
	}
}

func TestSqrt_NonPerfect(t *testing.T) {
	result := evalExpr(t, "sqrt(2)", nil)
	f := toF64(result)
	if f < 1.414 || f > 1.415 {
		t.Errorf("expected ~1.4142, got %v", f)
	}
}

func TestMod_Basic(t *testing.T) {
	result := evalExpr(t, "mod(10, 3)", nil)
	if toF64(result) != 1 {
		t.Errorf("expected 1, got %v", result)
	}
}

func TestMod_EvenDivision(t *testing.T) {
	result := evalExpr(t, "mod(10, 5)", nil)
	if toF64(result) != 0 {
		t.Errorf("expected 0, got %v", result)
	}
}

func TestMod_DivisionByZero(t *testing.T) {
	err := evalExprError(t, "mod(10, 0)", nil)
	if err == nil {
		t.Error("expected error for mod(10, 0), got nil")
	}
}

func toF64(v any) float64 {
	switch n := v.(type) {
	case float64:
		return n
	case int:
		return float64(n)
	case int64:
		return float64(n)
	}
	return 0
}
