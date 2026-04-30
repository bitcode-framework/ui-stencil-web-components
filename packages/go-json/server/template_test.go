package server

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestTranslateDateFormat_YYYY(t *testing.T) {
	result := translateDateFormat("YYYY-MM-DD")
	if result != "2006-01-02" {
		t.Errorf("got %q, want %q", result, "2006-01-02")
	}
}

func TestTranslateDateFormat_Full(t *testing.T) {
	result := translateDateFormat("YYYY-MM-DD HH:mm:ss")
	if result != "2006-01-02 15:04:05" {
		t.Errorf("got %q, want %q", result, "2006-01-02 15:04:05")
	}
}

func TestTranslateDateFormat_GoLayout(t *testing.T) {
	result := translateDateFormat("2006-01-02")
	if result != "2006-01-02" {
		t.Errorf("got %q, want %q", result, "2006-01-02")
	}
}

func TestTemplateEngine_RenderSimple(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "hello.html"), []byte(`<h1>Hello</h1>`), 0644); err != nil {
		t.Fatal(err)
	}

	te, err := NewTemplateEngine(dir, false)
	if err != nil {
		t.Fatal(err)
	}

	result, err := te.Render("hello.html", nil)
	if err != nil {
		t.Fatal(err)
	}
	if result != "<h1>Hello</h1>" {
		t.Errorf("got %q, want %q", result, "<h1>Hello</h1>")
	}
}

func TestTemplateEngine_RenderWithData(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "page.html"), []byte(`<h1>{{.title}}</h1>`), 0644); err != nil {
		t.Fatal(err)
	}

	te, err := NewTemplateEngine(dir, false)
	if err != nil {
		t.Fatal(err)
	}

	result, err := te.Render("page.html", map[string]any{"title": "Welcome"})
	if err != nil {
		t.Fatal(err)
	}
	if result != "<h1>Welcome</h1>" {
		t.Errorf("got %q, want %q", result, "<h1>Welcome</h1>")
	}
}

func TestTemplateEngine_RenderNotFound(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "exists.html"), []byte(`ok`), 0644); err != nil {
		t.Fatal(err)
	}

	te, err := NewTemplateEngine(dir, false)
	if err != nil {
		t.Fatal(err)
	}

	_, err = te.Render("nonexistent.html", nil)
	if err == nil {
		t.Error("expected error for non-existent template")
	}
}

func TestTemplateEngine_BuiltinFuncs_Upper(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "upper.html"), []byte(`{{upper .name}}`), 0644); err != nil {
		t.Fatal(err)
	}

	te, err := NewTemplateEngine(dir, false)
	if err != nil {
		t.Fatal(err)
	}

	result, err := te.Render("upper.html", map[string]any{"name": "hello"})
	if err != nil {
		t.Fatal(err)
	}
	if result != "HELLO" {
		t.Errorf("got %q, want %q", result, "HELLO")
	}
}

func TestTemplateEngine_BuiltinFuncs_Truncate(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "trunc.html"), []byte(`{{truncate .text 10}}`), 0644); err != nil {
		t.Fatal(err)
	}

	te, err := NewTemplateEngine(dir, false)
	if err != nil {
		t.Fatal(err)
	}

	result, err := te.Render("trunc.html", map[string]any{"text": "Hello World, this is long"})
	if err != nil {
		t.Fatal(err)
	}
	if result != "Hello W..." {
		t.Errorf("got %q, want %q", result, "Hello W...")
	}
}

func TestTemplateEngine_BuiltinFuncs_Default(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "def.html"), []byte(`{{default "fallback" .missing}}`), 0644); err != nil {
		t.Fatal(err)
	}

	te, err := NewTemplateEngine(dir, false)
	if err != nil {
		t.Fatal(err)
	}

	result, err := te.Render("def.html", map[string]any{})
	if err != nil {
		t.Fatal(err)
	}
	if result != "fallback" {
		t.Errorf("got %q, want %q", result, "fallback")
	}
}

func TestTemplateEngine_BuiltinFuncs_Add(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "add.html"), []byte(`{{add 1 2}}`), 0644); err != nil {
		t.Fatal(err)
	}

	te, err := NewTemplateEngine(dir, false)
	if err != nil {
		t.Fatal(err)
	}

	result, err := te.Render("add.html", map[string]any{})
	if err != nil {
		t.Fatal(err)
	}
	if result != "3" {
		t.Errorf("got %q, want %q", result, "3")
	}
}

func TestTemplateEngine_DevModeReload(t *testing.T) {
	dir := t.TempDir()
	tmplPath := filepath.Join(dir, "reload.html")
	if err := os.WriteFile(tmplPath, []byte(`version1`), 0644); err != nil {
		t.Fatal(err)
	}

	te, err := NewTemplateEngine(dir, true)
	if err != nil {
		t.Fatal(err)
	}

	result, err := te.Render("reload.html", nil)
	if err != nil {
		t.Fatal(err)
	}
	if result != "version1" {
		t.Errorf("got %q, want %q", result, "version1")
	}

	if err := os.WriteFile(tmplPath, []byte(`version2`), 0644); err != nil {
		t.Fatal(err)
	}

	result, err = te.Render("reload.html", nil)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(result, "version2") {
		t.Errorf("expected version2 after reload, got %q", result)
	}
}
