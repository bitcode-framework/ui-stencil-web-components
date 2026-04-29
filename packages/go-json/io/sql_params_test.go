package io

import (
	"testing"
)

func TestTranslateQuery_Positional_SQLite(t *testing.T) {
	q, args, err := TranslateQuery("SELECT * FROM users WHERE id = ? AND status = ?", "sqlite", []any{1, "active"})
	if err != nil {
		t.Fatal(err)
	}
	if q != "SELECT * FROM users WHERE id = ? AND status = ?" {
		t.Errorf("got %q", q)
	}
	if len(args) != 2 {
		t.Errorf("expected 2 args, got %d", len(args))
	}
}

func TestTranslateQuery_Positional_Postgres(t *testing.T) {
	q, args, err := TranslateQuery("SELECT * FROM users WHERE id = ? AND status = ?", "postgres", []any{1, "active"})
	if err != nil {
		t.Fatal(err)
	}
	if q != "SELECT * FROM users WHERE id = $1 AND status = $2" {
		t.Errorf("got %q", q)
	}
	if len(args) != 2 {
		t.Errorf("expected 2 args, got %d", len(args))
	}
}

func TestTranslateQuery_Positional_SQLServer(t *testing.T) {
	q, _, err := TranslateQuery("SELECT * FROM users WHERE id = ?", "sqlserver", []any{1})
	if err != nil {
		t.Fatal(err)
	}
	if q != "SELECT * FROM users WHERE id = @p1" {
		t.Errorf("got %q", q)
	}
}

func TestTranslateQuery_Named_Postgres(t *testing.T) {
	q, args, err := TranslateQuery("SELECT * FROM users WHERE id = :id AND status = :status", "postgres", map[string]any{"id": 1, "status": "active"})
	if err != nil {
		t.Fatal(err)
	}
	if q != "SELECT * FROM users WHERE id = $1 AND status = $2" {
		t.Errorf("got %q", q)
	}
	if len(args) != 2 {
		t.Errorf("expected 2 args, got %d", len(args))
	}
}

func TestTranslateQuery_Named_SQLite(t *testing.T) {
	q, args, err := TranslateQuery("SELECT * FROM users WHERE id = :id", "sqlite", map[string]any{"id": 42})
	if err != nil {
		t.Fatal(err)
	}
	if q != "SELECT * FROM users WHERE id = ?" {
		t.Errorf("got %q", q)
	}
	if len(args) != 1 || args[0] != 42 {
		t.Errorf("expected [42], got %v", args)
	}
}

func TestTranslateQuery_EscapeDoubleQuestion(t *testing.T) {
	q, _, err := TranslateQuery("SELECT * FROM t WHERE col ?? 'key'", "postgres", nil)
	if err != nil {
		t.Fatal(err)
	}
	if q != "SELECT * FROM t WHERE col ? 'key'" {
		t.Errorf("got %q", q)
	}
}

func TestTranslateQuery_QuestionInsideString(t *testing.T) {
	q, _, err := TranslateQuery("SELECT * FROM t WHERE col = 'what?'", "postgres", nil)
	if err != nil {
		t.Fatal(err)
	}
	if q != "SELECT * FROM t WHERE col = 'what?'" {
		t.Errorf("got %q", q)
	}
}

func TestTranslateQuery_MixedError(t *testing.T) {
	_, _, err := TranslateQuery("SELECT * FROM t WHERE id = ? AND name = :name", "postgres", nil)
	if err == nil {
		t.Fatal("expected error for mixed placeholders")
	}
}

func TestTranslateQuery_ArgCountMismatch(t *testing.T) {
	_, _, err := TranslateQuery("SELECT * FROM t WHERE id = ? AND name = ?", "sqlite", []any{1})
	if err == nil {
		t.Fatal("expected error for arg count mismatch")
	}
}

func TestTranslateQuery_NamedParamNotFound(t *testing.T) {
	_, _, err := TranslateQuery("SELECT * FROM t WHERE id = :id", "sqlite", map[string]any{"name": "test"})
	if err == nil {
		t.Fatal("expected error for missing named param")
	}
}

func TestTranslateQuery_EmptyQuery(t *testing.T) {
	q, args, err := TranslateQuery("", "sqlite", nil)
	if err != nil {
		t.Fatal(err)
	}
	if q != "" || args != nil {
		t.Errorf("expected empty result")
	}
}

func TestTranslateQuery_NoPlaceholders(t *testing.T) {
	q, _, err := TranslateQuery("SELECT 1", "postgres", nil)
	if err != nil {
		t.Fatal(err)
	}
	if q != "SELECT 1" {
		t.Errorf("got %q", q)
	}
}
