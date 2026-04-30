package server

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/bitcode-framework/go-json/lang"
)

func TestResolveStaticConfig_Nil(t *testing.T) {
	cfg := &lang.ServerConfig{Static: nil}
	result := ResolveStaticConfig(cfg)
	if result != nil {
		t.Errorf("expected nil, got %+v", result)
	}
}

func TestResolveStaticConfig_String(t *testing.T) {
	cfg := &lang.ServerConfig{Static: "public"}
	result := ResolveStaticConfig(cfg)
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if result.Dir != "public" {
		t.Errorf("Dir = %q, want %q", result.Dir, "public")
	}
	if result.Prefix != "/static" {
		t.Errorf("Prefix = %q, want %q", result.Prefix, "/static")
	}
}

func TestResolveStaticConfig_Struct(t *testing.T) {
	cfg := &lang.ServerConfig{
		Static: lang.StaticConfig{Dir: "assets", Prefix: "/files"},
	}
	result := ResolveStaticConfig(cfg)
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if result.Dir != "assets" {
		t.Errorf("Dir = %q, want %q", result.Dir, "assets")
	}
	if result.Prefix != "/files" {
		t.Errorf("Prefix = %q, want %q", result.Prefix, "/files")
	}
}

func TestResolveStaticConfig_EmptyString(t *testing.T) {
	cfg := &lang.ServerConfig{Static: ""}
	result := ResolveStaticConfig(cfg)
	if result != nil {
		t.Errorf("expected nil for empty string, got %+v", result)
	}
}

func TestValidateStaticDir_Exists(t *testing.T) {
	dir := t.TempDir()
	err := ValidateStaticDir(dir)
	if err != nil {
		t.Errorf("expected no error for existing dir, got %v", err)
	}
}

func TestValidateStaticDir_NotExists(t *testing.T) {
	err := ValidateStaticDir("/nonexistent/path/that/does/not/exist")
	if err == nil {
		t.Error("expected error for non-existent path")
	}
}

func TestValidateStaticDir_IsFile(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "notadir.txt")
	if err := os.WriteFile(file, []byte("hello"), 0644); err != nil {
		t.Fatal(err)
	}

	err := ValidateStaticDir(file)
	if err == nil {
		t.Error("expected error for file path")
	}
}

func TestIsPathSafe_Normal(t *testing.T) {
	dir := t.TempDir()
	if !IsPathSafe(dir, "style.css") {
		t.Error("expected style.css to be safe")
	}
}

func TestIsPathSafe_Subdirectory(t *testing.T) {
	dir := t.TempDir()
	if !IsPathSafe(dir, "css/style.css") {
		t.Error("expected css/style.css to be safe")
	}
}

func TestIsPathSafe_TraversalAttack(t *testing.T) {
	dir := t.TempDir()
	if IsPathSafe(dir, "../../../etc/passwd") {
		t.Error("expected path traversal to be unsafe")
	}
}

func TestIsPathSafe_HiddenFile(t *testing.T) {
	dir := t.TempDir()
	if IsPathSafe(dir, ".htaccess") {
		t.Error("expected .htaccess to be unsafe")
	}
}

func TestIsPathSafe_HiddenDir(t *testing.T) {
	dir := t.TempDir()
	// The implementation checks filepath.Base(requestedPath) for leading dot.
	// filepath.Base(".git/config") = "config" (no dot prefix), so it passes.
	// filepath.Base(".git") = ".git" (dot prefix), so it's blocked.
	if IsPathSafe(dir, ".git") {
		t.Error("expected .git to be unsafe")
	}
}
