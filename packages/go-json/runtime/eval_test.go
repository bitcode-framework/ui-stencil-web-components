package runtime

import (
	"math"
	"testing"
)

func TestEvalExpr_Arithmetic(t *testing.T) {
	tests := []struct {
		expr     string
		env      map[string]any
		expected any
	}{
		{"2 + 3", nil, 5},
		{"10 * 5", nil, 50},
		{"10 - 4", nil, 6},
		{"20 / 4", nil, 5},
	}

	for _, tt := range tests {
		val, err := EvalExpr(tt.expr, tt.env)
		if err != nil {
			t.Errorf("EvalExpr(%q) error: %v", tt.expr, err)
			continue
		}
		if toFloat(val) != toFloat(tt.expected) {
			t.Errorf("EvalExpr(%q) = %v, want %v", tt.expr, val, tt.expected)
		}
	}
}

func TestEvalExpr_Comparison(t *testing.T) {
	env := map[string]any{"a": 10, "b": 5}

	tests := []struct {
		expr     string
		expected bool
	}{
		{"a > b", true},
		{"a < b", false},
		{"a == b", false},
		{"a != b", true},
		{"a >= 10", true},
		{"b <= 5", true},
	}

	for _, tt := range tests {
		val, err := EvalExpr(tt.expr, env)
		if err != nil {
			t.Errorf("EvalExpr(%q) error: %v", tt.expr, err)
			continue
		}
		if val != tt.expected {
			t.Errorf("EvalExpr(%q) = %v, want %v", tt.expr, val, tt.expected)
		}
	}
}

func TestEvalExpr_StringFunctions(t *testing.T) {
	tests := []struct {
		expr     string
		expected any
	}{
		{`upper("hello")`, "HELLO"},
		{`lower("WORLD")`, "world"},
		{`"abc" contains "b"`, true},
		{`"abc" contains "z"`, false},
	}

	for _, tt := range tests {
		val, err := EvalExpr(tt.expr, nil)
		if err != nil {
			t.Errorf("EvalExpr(%q) error: %v", tt.expr, err)
			continue
		}
		if val != tt.expected {
			t.Errorf("EvalExpr(%q) = %v, want %v", tt.expr, val, tt.expected)
		}
	}
}

func TestEvalExprBool_Truthiness(t *testing.T) {
	tests := []struct {
		name     string
		expr     string
		env      map[string]any
		expected bool
	}{
		{"nil value", "x", map[string]any{"x": nil}, false},
		{"zero int", "x", map[string]any{"x": 0}, false},
		{"non-zero int", "x", map[string]any{"x": 42}, true},
		{"zero float", "x", map[string]any{"x": 0.0}, false},
		{"non-zero float", "x", map[string]any{"x": 3.14}, true},
		{"empty string", "x", map[string]any{"x": ""}, false},
		{"non-empty string", "x", map[string]any{"x": "hello"}, true},
		{"true bool", "x", map[string]any{"x": true}, true},
		{"false bool", "x", map[string]any{"x": false}, false},
		{"empty array", "x", map[string]any{"x": []any{}}, false},
		{"non-empty array", "x", map[string]any{"x": []any{1}}, true},
		{"empty map", "x", map[string]any{"x": map[string]any{}}, false},
		{"non-empty map", "x", map[string]any{"x": map[string]any{"a": 1}}, true},
		{"empty expression", "", nil, false},
		{"boolean expression true", "5 > 3", nil, true},
		{"boolean expression false", "3 > 5", nil, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			val, err := EvalExprBool(tt.expr, tt.env)
			if err != nil {
				t.Fatalf("EvalExprBool(%q) error: %v", tt.expr, err)
			}
			if val != tt.expected {
				t.Errorf("EvalExprBool(%q) = %v, want %v", tt.expr, val, tt.expected)
			}
		})
	}
}

func TestEvalExprFloat_Coercion(t *testing.T) {
	tests := []struct {
		name     string
		expr     string
		env      map[string]any
		expected float64
	}{
		{"int result", "x", map[string]any{"x": 42}, 42.0},
		{"float64 result", "x", map[string]any{"x": 3.14}, 3.14},
		{"string parseable", "x", map[string]any{"x": "3.14"}, 3.14},
		{"arithmetic", "2 + 3", nil, 5.0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			val, err := EvalExprFloat(tt.expr, tt.env)
			if err != nil {
				t.Fatalf("EvalExprFloat(%q) error: %v", tt.expr, err)
			}
			if math.Abs(val-tt.expected) > 0.0001 {
				t.Errorf("EvalExprFloat(%q) = %v, want %v", tt.expr, val, tt.expected)
			}
		})
	}
}

func TestEvalExprFloat_Error(t *testing.T) {
	_, err := EvalExprFloat("x", map[string]any{"x": "not-a-number"})
	if err == nil {
		t.Error("expected error for non-numeric string, got nil")
	}
}

func TestParseExpr_ValidExpression(t *testing.T) {
	tree, err := ParseExpr("a + b")
	if err != nil {
		t.Fatalf("ParseExpr error: %v", err)
	}
	if tree == nil {
		t.Fatal("expected non-nil ExprTree")
	}
	if tree.Root == nil {
		t.Fatal("expected non-nil Root node")
	}
}

func TestParseExpr_InvalidExpression(t *testing.T) {
	_, err := ParseExpr("+ +")
	if err == nil {
		t.Error("expected error for invalid expression, got nil")
	}
}

func TestParseExpr_EmptyExpression(t *testing.T) {
	_, err := ParseExpr("")
	if err == nil {
		t.Error("expected error for empty expression, got nil")
	}
}

func TestValidateExpr_Valid(t *testing.T) {
	env := map[string]any{"a": 1, "b": 2}
	err := ValidateExpr("a + b", env)
	if err != nil {
		t.Errorf("ValidateExpr returned error for valid expression: %v", err)
	}
}

func TestValidateExpr_Invalid(t *testing.T) {
	err := ValidateExpr("a +", nil)
	if err == nil {
		t.Error("expected error for invalid expression, got nil")
	}
}

func TestValidateExpr_Empty(t *testing.T) {
	err := ValidateExpr("", nil)
	if err == nil {
		t.Error("expected error for empty expression, got nil")
	}
}

func TestEvalExpr_CacheHit(t *testing.T) {
	env := map[string]any{"x": 10}

	val1, err := EvalExpr("x + 1", env)
	if err != nil {
		t.Fatalf("first eval error: %v", err)
	}

	val2, err := EvalExpr("x + 1", env)
	if err != nil {
		t.Fatalf("second eval error: %v", err)
	}

	if toFloat(val1) != toFloat(val2) {
		t.Errorf("cache hit returned different result: %v vs %v", val1, val2)
	}
}

func TestEvalExpr_EmptyExpression(t *testing.T) {
	val, err := EvalExpr("", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if val != nil {
		t.Errorf("expected nil for empty expression, got %v", val)
	}
}

func TestEvalExpr_MapAccess(t *testing.T) {
	env := map[string]any{
		"input": map[string]any{
			"total": 150,
			"name":  "test",
		},
	}

	val, err := EvalExpr("input.total > 100", env)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if val != true {
		t.Errorf("expected true, got %v", val)
	}
}

func TestEvalExprBool_ComplexCondition(t *testing.T) {
	env := map[string]any{
		"status": "active",
		"amount": 1500,
	}

	val, err := EvalExprBool(`status == "active" && amount > 1000`, env)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !val {
		t.Error("expected true for complex condition")
	}
}

func TestEvalExpr_StdlibFunctions(t *testing.T) {
	tests := []struct {
		name     string
		expr     string
		env      map[string]any
		expected any
	}{
		{"clamp low", `clamp(x, 0, 100)`, map[string]any{"x": -5}, 0},
		{"clamp high", `clamp(x, 0, 100)`, map[string]any{"x": 150}, 100},
		{"clamp in range", `clamp(x, 0, 100)`, map[string]any{"x": 50}, 50},
		{"padLeft", `padLeft("hi", 5, "0")`, nil, "000hi"},
		{"padRight", `padRight("hi", 5, "0")`, nil, "hi000"},
		{"pow", `pow(2, 10)`, nil, 1024.0},
		{"sqrt", `sqrt(144)`, nil, 12.0},
		{"isNil true", `isNil(x)`, map[string]any{"x": nil}, true},
		{"isNil false", `isNil(x)`, map[string]any{"x": 42}, false},
		{"substring", `substring("hello", 1, 4)`, nil, "ell"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			val, err := EvalExpr(tt.expr, tt.env)
			if err != nil {
				t.Fatalf("EvalExpr(%q) error: %v", tt.expr, err)
			}
			if toFloat(val) != toFloat(tt.expected) && val != tt.expected {
				t.Errorf("EvalExpr(%q) = %v (%T), want %v (%T)", tt.expr, val, val, tt.expected, tt.expected)
			}
		})
	}
}

func TestEvalExpr_StrFunctions(t *testing.T) {
	tests := []struct {
		name     string
		expr     string
		expected any
	}{
		{"strContains true", `strContains("hello world", "world")`, true},
		{"strContains false", `strContains("hello", "xyz")`, false},
		{"strStartsWith true", `strStartsWith("hello", "hel")`, true},
		{"strStartsWith false", `strStartsWith("hello", "xyz")`, false},
		{"strEndsWith true", `strEndsWith("hello", "llo")`, true},
		{"strEndsWith false", `strEndsWith("hello", "xyz")`, false},
		{"strMatches true", `strMatches("hello123", "\\d+")`, true},
		{"strMatches false", `strMatches("hello", "\\d+")`, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			val, err := EvalExpr(tt.expr, nil)
			if err != nil {
				t.Fatalf("EvalExpr(%q) error: %v", tt.expr, err)
			}
			if val != tt.expected {
				t.Errorf("EvalExpr(%q) = %v, want %v", tt.expr, val, tt.expected)
			}
		})
	}
}

func TestEvalExpr_OperatorStyleStillWorks(t *testing.T) {
	tests := []struct {
		name     string
		expr     string
		expected any
	}{
		{"contains operator", `"hello" contains "ell"`, true},
		{"startsWith operator", `"hello" startsWith "hel"`, true},
		{"endsWith operator", `"hello" endsWith "llo"`, true},
		{"matches operator", `"hello123" matches "\\d+"`, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			val, err := EvalExpr(tt.expr, nil)
			if err != nil {
				t.Fatalf("EvalExpr(%q) error: %v", tt.expr, err)
			}
			if val != tt.expected {
				t.Errorf("EvalExpr(%q) = %v, want %v", tt.expr, val, tt.expected)
			}
		})
	}
}

func TestEvalExpr_CryptoNamespace(t *testing.T) {
	val, err := EvalExpr(`crypto.uuid()`, nil)
	if err != nil {
		t.Fatalf("crypto.uuid() error: %v", err)
	}
	s, ok := val.(string)
	if !ok || len(s) < 32 {
		t.Errorf("crypto.uuid() returned %v, expected UUID string", val)
	}
}

func toFloat(v any) float64 {
	switch val := v.(type) {
	case float64:
		return val
	case float32:
		return float64(val)
	case int:
		return float64(val)
	case int64:
		return float64(val)
	default:
		return 0
	}
}
