package stdlib

import "testing"

func TestBool_ZeroInt(t *testing.T) {
	env := map[string]any{"val": 0}
	result := evalExpr(t, "bool(val)", env)
	if result != false {
		t.Errorf("expected false, got %v", result)
	}
}

func TestBool_NonZeroInt(t *testing.T) {
	env := map[string]any{"val": 1}
	result := evalExpr(t, "bool(val)", env)
	if result != true {
		t.Errorf("expected true, got %v", result)
	}
}

func TestBool_EmptyString(t *testing.T) {
	result := evalExpr(t, `bool("")`, nil)
	if result != false {
		t.Errorf("expected false, got %v", result)
	}
}

func TestBool_NonEmptyString(t *testing.T) {
	result := evalExpr(t, `bool("hello")`, nil)
	if result != true {
		t.Errorf("expected true, got %v", result)
	}
}

func TestBool_Nil(t *testing.T) {
	env := map[string]any{"val": nil}
	result := evalExpr(t, "bool(val)", env)
	if result != false {
		t.Errorf("expected false, got %v", result)
	}
}

func TestBool_EmptyArray(t *testing.T) {
	env := map[string]any{"val": []any{}}
	result := evalExpr(t, "bool(val)", env)
	if result != false {
		t.Errorf("expected false, got %v", result)
	}
}

func TestBool_NonEmptyArray(t *testing.T) {
	env := map[string]any{"val": []any{1}}
	result := evalExpr(t, "bool(val)", env)
	if result != true {
		t.Errorf("expected true, got %v", result)
	}
}

func TestBool_EmptyMap(t *testing.T) {
	env := map[string]any{"val": map[string]any{}}
	result := evalExpr(t, "bool(val)", env)
	if result != false {
		t.Errorf("expected false, got %v", result)
	}
}

func TestBool_NonEmptyMap(t *testing.T) {
	env := map[string]any{"val": map[string]any{"a": 1}}
	result := evalExpr(t, "bool(val)", env)
	if result != true {
		t.Errorf("expected true, got %v", result)
	}
}

func TestBool_TrueValue(t *testing.T) {
	env := map[string]any{"val": true}
	result := evalExpr(t, "bool(val)", env)
	if result != true {
		t.Errorf("expected true, got %v", result)
	}
}

func TestBool_FalseValue(t *testing.T) {
	env := map[string]any{"val": false}
	result := evalExpr(t, "bool(val)", env)
	if result != false {
		t.Errorf("expected false, got %v", result)
	}
}

func TestBool_ZeroFloat(t *testing.T) {
	env := map[string]any{"val": 0.0}
	result := evalExpr(t, "bool(val)", env)
	if result != false {
		t.Errorf("expected false, got %v", result)
	}
}

func TestBool_NonZeroFloat(t *testing.T) {
	env := map[string]any{"val": 3.14}
	result := evalExpr(t, "bool(val)", env)
	if result != true {
		t.Errorf("expected true, got %v", result)
	}
}

func TestIsNil_NilValue(t *testing.T) {
	env := map[string]any{"val": nil}
	result := evalExpr(t, "isNil(val)", env)
	if result != true {
		t.Errorf("expected true, got %v", result)
	}
}

func TestIsNil_ZeroInt(t *testing.T) {
	env := map[string]any{"val": 0}
	result := evalExpr(t, "isNil(val)", env)
	if result != false {
		t.Errorf("expected false, got %v", result)
	}
}

func TestIsNil_EmptyString(t *testing.T) {
	result := evalExpr(t, `isNil("")`, nil)
	if result != false {
		t.Errorf("expected false, got %v", result)
	}
}

func TestIsNil_NonNilValue(t *testing.T) {
	env := map[string]any{"val": "hello"}
	result := evalExpr(t, "isNil(val)", env)
	if result != false {
		t.Errorf("expected false, got %v", result)
	}
}

func TestIsNil_EmptyArray(t *testing.T) {
	env := map[string]any{"val": []any{}}
	result := evalExpr(t, "isNil(val)", env)
	if result != false {
		t.Errorf("expected false, got %v", result)
	}
}
