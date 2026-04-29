package stdlib

import "testing"

func TestPadLeft_Padding(t *testing.T) {
	result := evalExpr(t, `padLeft("hi", 5, "0")`, nil)
	if result != "000hi" {
		t.Errorf("expected '000hi', got %v", result)
	}
}

func TestPadLeft_NoTruncation(t *testing.T) {
	result := evalExpr(t, `padLeft("hello", 3, "0")`, nil)
	if result != "hello" {
		t.Errorf("expected 'hello', got %v", result)
	}
}

func TestPadLeft_ExactLength(t *testing.T) {
	result := evalExpr(t, `padLeft("abc", 3, "0")`, nil)
	if result != "abc" {
		t.Errorf("expected 'abc', got %v", result)
	}
}

func TestPadLeft_DefaultPad(t *testing.T) {
	result := evalExpr(t, `padLeft("hi", 5)`, nil)
	if result != "   hi" {
		t.Errorf("expected '   hi', got %q", result)
	}
}

func TestPadRight_Padding(t *testing.T) {
	result := evalExpr(t, `padRight("hi", 5, "0")`, nil)
	if result != "hi000" {
		t.Errorf("expected 'hi000', got %v", result)
	}
}

func TestPadRight_NoTruncation(t *testing.T) {
	result := evalExpr(t, `padRight("hello", 3, "0")`, nil)
	if result != "hello" {
		t.Errorf("expected 'hello', got %v", result)
	}
}

func TestPadRight_DefaultPad(t *testing.T) {
	result := evalExpr(t, `padRight("hi", 5)`, nil)
	if result != "hi   " {
		t.Errorf("expected 'hi   ', got %q", result)
	}
}

func TestSubstring_WithStartAndEnd(t *testing.T) {
	result := evalExpr(t, `substring("hello", 1, 3)`, nil)
	if result != "el" {
		t.Errorf("expected 'el', got %v", result)
	}
}

func TestSubstring_FromStart(t *testing.T) {
	result := evalExpr(t, `substring("hello", 0)`, nil)
	if result != "hello" {
		t.Errorf("expected 'hello', got %v", result)
	}
}

func TestSubstring_FromMiddle(t *testing.T) {
	result := evalExpr(t, `substring("hello", 2)`, nil)
	if result != "llo" {
		t.Errorf("expected 'llo', got %v", result)
	}
}

func TestSubstring_BeyondLength(t *testing.T) {
	result := evalExpr(t, `substring("hello", 10)`, nil)
	if result != "" {
		t.Errorf("expected '', got %v", result)
	}
}

func TestFormat_StringAndInt(t *testing.T) {
	result := evalExpr(t, `format("%s is %d", "age", 30)`, nil)
	if result != "age is 30" {
		t.Errorf("expected 'age is 30', got %v", result)
	}
}

func TestFormat_SingleArg(t *testing.T) {
	result := evalExpr(t, `format("hello %s", "world")`, nil)
	if result != "hello world" {
		t.Errorf("expected 'hello world', got %v", result)
	}
}

func TestStrMatches_Match(t *testing.T) {
	result := evalExpr(t, `strMatches("hello123", "[0-9]+")`, nil)
	if result != true {
		t.Errorf("expected true, got %v", result)
	}
}

func TestStrMatches_NoMatch(t *testing.T) {
	result := evalExpr(t, `strMatches("hello", "^[0-9]+$")`, nil)
	if result != false {
		t.Errorf("expected false, got %v", result)
	}
}

func TestStrContains_Found(t *testing.T) {
	result := evalExpr(t, `strContains("hello world", "world")`, nil)
	if result != true {
		t.Errorf("expected true, got %v", result)
	}
}

func TestStrContains_NotFound(t *testing.T) {
	result := evalExpr(t, `strContains("hello world", "xyz")`, nil)
	if result != false {
		t.Errorf("expected false, got %v", result)
	}
}

func TestStrStartsWith_Match(t *testing.T) {
	result := evalExpr(t, `strStartsWith("hello", "hel")`, nil)
	if result != true {
		t.Errorf("expected true, got %v", result)
	}
}

func TestStrStartsWith_NoMatch(t *testing.T) {
	result := evalExpr(t, `strStartsWith("hello", "xyz")`, nil)
	if result != false {
		t.Errorf("expected false, got %v", result)
	}
}

func TestStrEndsWith_Match(t *testing.T) {
	result := evalExpr(t, `strEndsWith("hello", "llo")`, nil)
	if result != true {
		t.Errorf("expected true, got %v", result)
	}
}

func TestStrEndsWith_NoMatch(t *testing.T) {
	result := evalExpr(t, `strEndsWith("hello", "xyz")`, nil)
	if result != false {
		t.Errorf("expected false, got %v", result)
	}
}
