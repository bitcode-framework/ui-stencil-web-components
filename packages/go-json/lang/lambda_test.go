package lang

import (
	"testing"
)

func TestParseLambda_SingleParam(t *testing.T) {
	lambda := ParseLambda("fn(x) => x * 2")
	if lambda == nil {
		t.Fatal("expected lambda, got nil")
	}
	if len(lambda.Params) != 1 || lambda.Params[0] != "x" {
		t.Fatalf("expected params [x], got %v", lambda.Params)
	}
	if lambda.Body != "x * 2" {
		t.Fatalf("expected body 'x * 2', got '%s'", lambda.Body)
	}
}

func TestParseLambda_MultiParam(t *testing.T) {
	lambda := ParseLambda("fn(a, b) => a + b")
	if lambda == nil {
		t.Fatal("expected lambda, got nil")
	}
	if len(lambda.Params) != 2 || lambda.Params[0] != "a" || lambda.Params[1] != "b" {
		t.Fatalf("expected params [a, b], got %v", lambda.Params)
	}
	if lambda.Body != "a + b" {
		t.Fatalf("expected body 'a + b', got '%s'", lambda.Body)
	}
}

func TestParseLambda_NoParams(t *testing.T) {
	lambda := ParseLambda("fn() => 'hello'")
	if lambda == nil {
		t.Fatal("expected lambda, got nil")
	}
	if len(lambda.Params) != 0 {
		t.Fatalf("expected no params, got %v", lambda.Params)
	}
	if lambda.Body != "'hello'" {
		t.Fatalf("expected body \"'hello'\", got '%s'", lambda.Body)
	}
}

func TestParseLambda_NestedParensInBody(t *testing.T) {
	lambda := ParseLambda("fn(x) => max(x, 0)")
	if lambda == nil {
		t.Fatal("expected lambda, got nil")
	}
	if lambda.Body != "max(x, 0)" {
		t.Fatalf("expected body 'max(x, 0)', got '%s'", lambda.Body)
	}
}

func TestParseLambda_StringLiteralInBody(t *testing.T) {
	lambda := ParseLambda("fn(x) => 'hello ' + x")
	if lambda == nil {
		t.Fatal("expected lambda, got nil")
	}
	if lambda.Body != "'hello ' + x" {
		t.Fatalf("expected body \"'hello ' + x\", got '%s'", lambda.Body)
	}
}

func TestParseLambda_MultiStatementBody(t *testing.T) {
	lambda := ParseLambda("fn(x) => let y = x * 2; y + 10")
	if lambda == nil {
		t.Fatal("expected lambda, got nil")
	}
	if lambda.Body != "let y = x * 2; y + 10" {
		t.Fatalf("expected body 'let y = x * 2; y + 10', got '%s'", lambda.Body)
	}
}

func TestParseLambda_WithTernary(t *testing.T) {
	lambda := ParseLambda("fn(age) => age >= 18 ? 'adult' : 'minor'")
	if lambda == nil {
		t.Fatal("expected lambda, got nil")
	}
	if lambda.Body != "age >= 18 ? 'adult' : 'minor'" {
		t.Fatalf("unexpected body: '%s'", lambda.Body)
	}
}

func TestParseLambda_WithWhitespace(t *testing.T) {
	lambda := ParseLambda("  fn( x , y )  =>  x + y  ")
	if lambda == nil {
		t.Fatal("expected lambda, got nil")
	}
	if len(lambda.Params) != 2 || lambda.Params[0] != "x" || lambda.Params[1] != "y" {
		t.Fatalf("expected params [x, y], got %v", lambda.Params)
	}
	if lambda.Body != "x + y" {
		t.Fatalf("expected body 'x + y', got '%s'", lambda.Body)
	}
}

func TestParseLambda_RejectsInvalid(t *testing.T) {
	cases := []struct {
		name  string
		input string
	}{
		{"not a lambda", "x * 2"},
		{"no arrow", "fn(x) x * 2"},
		{"empty body", "fn(x) => "},
		{"unclosed paren", "fn(x => x * 2"},
		{"invalid param name", "fn(123) => x"},
		{"empty param", "fn(x,,y) => x"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			result := ParseLambda(tc.input)
			if result != nil {
				t.Fatalf("expected nil for input '%s', got %+v", tc.input, result)
			}
		})
	}
}

func TestFindInlineLambdas_Single(t *testing.T) {
	expr := "mapFn(items, fn(x) => x * 2)"
	positions := FindInlineLambdas(expr)
	if len(positions) != 1 {
		t.Fatalf("expected 1 lambda, got %d", len(positions))
	}
	lambdaStr := expr[positions[0][0]:positions[0][1]]
	if lambdaStr != "fn(x) => x * 2" {
		t.Fatalf("expected 'fn(x) => x * 2', got '%s'", lambdaStr)
	}
}

func TestFindInlineLambdas_Multiple(t *testing.T) {
	expr := "mapFn(filterFn(items, fn(x) => x > 0), fn(x) => x * 2)"
	positions := FindInlineLambdas(expr)
	if len(positions) != 2 {
		t.Fatalf("expected 2 lambdas, got %d", len(positions))
	}
	first := expr[positions[0][0]:positions[0][1]]
	if first != "fn(x) => x > 0" {
		t.Fatalf("first lambda: expected 'fn(x) => x > 0', got '%s'", first)
	}
	second := expr[positions[1][0]:positions[1][1]]
	if second != "fn(x) => x * 2" {
		t.Fatalf("second lambda: expected 'fn(x) => x * 2', got '%s'", second)
	}
}

func TestFindInlineLambdas_InStringLiteral(t *testing.T) {
	expr := "'fn(x) => x' + y"
	positions := FindInlineLambdas(expr)
	if len(positions) != 0 {
		t.Fatalf("expected 0 lambdas inside string literal, got %d", len(positions))
	}
}

func TestFindInlineLambdas_NotPartOfIdentifier(t *testing.T) {
	expr := "xfn(x) + 1"
	positions := FindInlineLambdas(expr)
	if len(positions) != 0 {
		t.Fatalf("expected 0 lambdas (xfn is not fn), got %d", len(positions))
	}
}

func TestFindInlineLambdas_NestedFunctionCallInBody(t *testing.T) {
	expr := "mapFn(items, fn(x) => max(x, 0))"
	positions := FindInlineLambdas(expr)
	if len(positions) != 1 {
		t.Fatalf("expected 1 lambda, got %d", len(positions))
	}
	lambdaStr := expr[positions[0][0]:positions[0][1]]
	if lambdaStr != "fn(x) => max(x, 0)" {
		t.Fatalf("expected 'fn(x) => max(x, 0)', got '%s'", lambdaStr)
	}
}

func TestFindInlineLambdas_NoLambdas(t *testing.T) {
	expr := "x + y * 2"
	positions := FindInlineLambdas(expr)
	if len(positions) != 0 {
		t.Fatalf("expected 0 lambdas, got %d", len(positions))
	}
}

func TestFindBodyEnd_SimpleExpression(t *testing.T) {
	expr := "x * 2)"
	end := findBodyEnd(expr, 0)
	if end != 5 {
		t.Fatalf("expected end at 5, got %d (body: '%s')", end, expr[:end])
	}
}

func TestFindBodyEnd_WithNestedParens(t *testing.T) {
	expr := "max(x, 0))"
	end := findBodyEnd(expr, 0)
	if end != 9 {
		t.Fatalf("expected end at 9, got %d (body: '%s')", end, expr[:end])
	}
}

func TestFindBodyEnd_CommaDelimited(t *testing.T) {
	expr := "x * 2, fn(y) => y + 1)"
	end := findBodyEnd(expr, 0)
	if end != 5 {
		t.Fatalf("expected end at 5 (comma), got %d (body: '%s')", end, expr[:end])
	}
}

func TestParseLambda_Named(t *testing.T) {
	lambda := ParseLambda("fn factorial(n) => n <= 1 ? 1 : n * factorial(n - 1)")
	if lambda == nil {
		t.Fatal("expected lambda, got nil")
	}
	if lambda.Name != "factorial" {
		t.Fatalf("expected name 'factorial', got '%s'", lambda.Name)
	}
	if len(lambda.Params) != 1 || lambda.Params[0] != "n" {
		t.Fatalf("expected params [n], got %v", lambda.Params)
	}
	if lambda.Body != "n <= 1 ? 1 : n * factorial(n - 1)" {
		t.Fatalf("unexpected body: '%s'", lambda.Body)
	}
}

func TestParseLambda_NamedMultiParam(t *testing.T) {
	lambda := ParseLambda("fn add(a, b) => a + b")
	if lambda == nil {
		t.Fatal("expected lambda, got nil")
	}
	if lambda.Name != "add" {
		t.Fatalf("expected name 'add', got '%s'", lambda.Name)
	}
	if len(lambda.Params) != 2 || lambda.Params[0] != "a" || lambda.Params[1] != "b" {
		t.Fatalf("expected params [a, b], got %v", lambda.Params)
	}
}

func TestParseLambda_NamedNoParams(t *testing.T) {
	lambda := ParseLambda("fn greet() => 'hello'")
	if lambda == nil {
		t.Fatal("expected lambda, got nil")
	}
	if lambda.Name != "greet" {
		t.Fatalf("expected name 'greet', got '%s'", lambda.Name)
	}
	if len(lambda.Params) != 0 {
		t.Fatalf("expected no params, got %v", lambda.Params)
	}
}

func TestParseLambda_AnonymousHasNoName(t *testing.T) {
	lambda := ParseLambda("fn(x) => x * 2")
	if lambda == nil {
		t.Fatal("expected lambda, got nil")
	}
	if lambda.Name != "" {
		t.Fatalf("expected empty name for anonymous lambda, got '%s'", lambda.Name)
	}
}

func TestFindInlineLambdas_Named(t *testing.T) {
	expr := "mapFn([1,2,3], fn double(x) => x * 2)"
	positions := FindInlineLambdas(expr)
	if len(positions) != 1 {
		t.Fatalf("expected 1 lambda, got %d", len(positions))
	}
	lambdaStr := expr[positions[0][0]:positions[0][1]]
	if lambdaStr != "fn double(x) => x * 2" {
		t.Fatalf("expected 'fn double(x) => x * 2', got '%s'", lambdaStr)
	}
}
