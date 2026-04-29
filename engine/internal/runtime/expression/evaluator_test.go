package expression

import (
	"math"
	"testing"

	"github.com/bitcode-framework/go-json/runtime"
)

func evalFloat(expr string, record map[string]any, children map[string][]map[string]any) (float64, error) {
	env := buildComputedFieldEnv(record, children)
	return runtime.EvalExprFloat(expr, env)
}

func eval(expr string, record map[string]any, children map[string][]map[string]any) (any, error) {
	env := buildComputedFieldEnv(record, children)
	return runtime.EvalExpr(expr, env)
}

func noChildren() map[string][]map[string]any {
	return make(map[string][]map[string]any)
}

func TestBasicArithmetic(t *testing.T) {
	tests := []struct {
		expr     string
		expected float64
	}{
		{"2 + 3", 5},
		{"10 - 4", 6},
		{"3 * 7", 21},
		{"20 / 4", 5},
		{"10 % 3", 1},
		{"2 + 3 * 4", 14},
		{"(2 + 3) * 4", 20},
		{"-5 + 10", 5},
	}

	for _, tt := range tests {
		val, err := evalFloat(tt.expr, map[string]any{}, noChildren())
		if err != nil {
			t.Errorf("evalFloat(%q) error: %v", tt.expr, err)
			continue
		}
		if math.Abs(val-tt.expected) > 0.0001 {
			t.Errorf("evalFloat(%q) = %v, want %v", tt.expr, val, tt.expected)
		}
	}
}

func TestFieldReferences(t *testing.T) {
	record := map[string]any{
		"quantity":   10.0,
		"unit_price": 25.5,
		"discount":   5.0,
	}

	val, err := evalFloat("quantity * unit_price", record, noChildren())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if math.Abs(val-255.0) > 0.0001 {
		t.Errorf("got %v, want 255.0", val)
	}

	val, err = evalFloat("quantity * unit_price - discount", record, noChildren())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if math.Abs(val-250.0) > 0.0001 {
		t.Errorf("got %v, want 250.0", val)
	}
}

func TestComputedFieldFormula(t *testing.T) {
	record := map[string]any{
		"expected_revenue": 100000.0,
		"probability":      75.0,
	}

	val, err := evalFloat("expected_revenue * probability / 100", record, noChildren())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if math.Abs(val-75000.0) > 0.0001 {
		t.Errorf("got %v, want 75000.0", val)
	}
}

func TestAggregateSum(t *testing.T) {
	children := map[string][]map[string]any{
		"lines": {
			{"subtotal": 100.0},
			{"subtotal": 200.0},
			{"subtotal": 50.0},
		},
	}

	val, err := evalFloat(`sum("lines.subtotal")`, map[string]any{}, children)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if math.Abs(val-350.0) > 0.0001 {
		t.Errorf("got %v, want 350.0", val)
	}
}

func TestAggregateCount(t *testing.T) {
	children := map[string][]map[string]any{
		"items": {
			{"qty": 1},
			{"qty": 2},
			{"qty": 3},
		},
	}

	val, err := evalFloat(`count("items.qty")`, map[string]any{}, children)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if math.Abs(val-3.0) > 0.0001 {
		t.Errorf("got %v, want 3.0", val)
	}
}

func TestAggregateAvg(t *testing.T) {
	children := map[string][]map[string]any{
		"scores": {
			{"value": 80.0},
			{"value": 90.0},
			{"value": 100.0},
		},
	}

	val, err := evalFloat(`avg("scores.value")`, map[string]any{}, children)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if math.Abs(val-90.0) > 0.0001 {
		t.Errorf("got %v, want 90.0", val)
	}
}

func TestAggregateMinMax(t *testing.T) {
	children := map[string][]map[string]any{
		"prices": {
			{"amount": 10.0},
			{"amount": 50.0},
			{"amount": 30.0},
		},
	}

	val, err := evalFloat(`min("prices.amount")`, map[string]any{}, children)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if math.Abs(val-10.0) > 0.0001 {
		t.Errorf("min got %v, want 10.0", val)
	}

	val, err = evalFloat(`max("prices.amount")`, map[string]any{}, children)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if math.Abs(val-50.0) > 0.0001 {
		t.Errorf("max got %v, want 50.0", val)
	}
}

func TestEmptyCollection(t *testing.T) {
	val, err := evalFloat(`sum("lines.subtotal")`, map[string]any{}, noChildren())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if val != 0 {
		t.Errorf("got %v, want 0", val)
	}
}

func TestComparisons(t *testing.T) {
	record := map[string]any{
		"a": 10.0,
		"b": 20.0,
	}

	tests := []struct {
		expr     string
		expected bool
	}{
		{"a < b", true},
		{"a > b", false},
		{"a == b", false},
		{"a != b", true},
		{"a <= 10", true},
		{"b >= 20", true},
	}

	for _, tt := range tests {
		val, err := eval(tt.expr, record, noChildren())
		if err != nil {
			t.Errorf("eval(%q) error: %v", tt.expr, err)
			continue
		}
		if val != tt.expected {
			t.Errorf("eval(%q) = %v, want %v", tt.expr, val, tt.expected)
		}
	}
}

func TestBooleanLogic(t *testing.T) {
	record := map[string]any{
		"active": true,
		"paid":   false,
	}

	val, err := eval("active && paid", record, noChildren())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if val != false {
		t.Errorf("got %v, want false", val)
	}

	val, err = eval("active || paid", record, noChildren())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if val != true {
		t.Errorf("got %v, want true", val)
	}
}

func TestBuiltinFunctions(t *testing.T) {
	val, err := evalFloat("abs(-42)", map[string]any{}, noChildren())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if val != 42 {
		t.Errorf("abs got %v, want 42", val)
	}

	val, err = evalFloat("ceil(3.2)", map[string]any{}, noChildren())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if val != 4 {
		t.Errorf("ceil got %v, want 4", val)
	}

	val, err = evalFloat("floor(3.8)", map[string]any{}, noChildren())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if val != 3 {
		t.Errorf("floor got %v, want 3", val)
	}
}

func TestTernaryCondition(t *testing.T) {
	record := map[string]any{
		"amount": 100.0,
	}

	val, err := evalFloat("amount > 50 ? amount * 2 : amount", record, noChildren())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if val != 200 {
		t.Errorf("got %v, want 200", val)
	}
}

func TestStringConcat(t *testing.T) {
	record := map[string]any{
		"first": "John",
		"last":  "Doe",
	}

	val, err := eval("first + ' ' + last", record, noChildren())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if val != "John Doe" {
		t.Errorf("got %v, want 'John Doe'", val)
	}
}

func TestEmptyExpression(t *testing.T) {
	val, err := runtime.EvalExpr("", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if val != nil {
		t.Errorf("got %v, want nil", val)
	}
}

func TestIntegerFieldValues(t *testing.T) {
	record := map[string]any{
		"qty":   5,
		"price": 10,
	}

	val, err := evalFloat("qty * price", record, noChildren())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if val != 50 {
		t.Errorf("got %v, want 50", val)
	}
}

func TestNilFieldCoercion(t *testing.T) {
	record := map[string]any{
		"quantity":   5.0,
		"unit_price": nil,
	}

	val, err := evalFloat("quantity * (unit_price ?? 0)", record, noChildren())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if val != 0 {
		t.Errorf("got %v, want 0 (nil field with ?? 0 should be 0)", val)
	}
}

func TestExprLangGainedCapabilities(t *testing.T) {
	record := map[string]any{
		"first_name": "john",
		"last_name":  "doe",
		"status":     "active",
		"amount":     100.0,
		"discount":   10.0,
	}

	tests := []struct {
		name     string
		expr     string
		expected any
	}{
		{"upper", `upper(first_name) + " " + upper(last_name)`, "JOHN DOE"},
		{"ternary", `status == "active" ? amount : 0`, 100.0},
		{"ternary with discount", `amount * (1 - discount / 100)`, 90.0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			val, err := eval(tt.expr, record, noChildren())
			if err != nil {
				t.Fatalf("eval(%q) error: %v", tt.expr, err)
			}
			switch expected := tt.expected.(type) {
			case float64:
				f, ok := val.(float64)
				if !ok {
					if i, ok2 := val.(int); ok2 {
						f = float64(i)
					}
				}
				if math.Abs(f-expected) > 0.0001 {
					t.Errorf("eval(%q) = %v, want %v", tt.expr, val, expected)
				}
			default:
				if val != tt.expected {
					t.Errorf("eval(%q) = %v, want %v", tt.expr, val, tt.expected)
				}
			}
		})
	}
}
