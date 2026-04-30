package io

import (
	"encoding/base64"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func newTestFSModule(tmpDir string) *FSModule {
	security := DefaultSecurityConfig()
	security.FS.AllowedPaths = nil
	security.FS.BlockedPaths = nil
	security.FS.AllowWrite = true
	return NewFSModule(security)
}

func TestFSModule_Read(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")
	content := "Hello, World!"
	if err := os.WriteFile(testFile, []byte(content), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	module := newTestFSModule(tmpDir)

	result, err := module.fsRead(testFile)
	if err != nil {
		t.Fatalf("fsRead failed: %v", err)
	}

	if result != content {
		t.Errorf("expected %q, got %q", content, result)
	}
}

func TestFSModule_ReadNonExistent(t *testing.T) {
	tmpDir := t.TempDir()
	nonExistent := filepath.Join(tmpDir, "nonexistent.txt")

	security := DefaultSecurityConfig()
	security.FS.AllowedPaths = nil
	module := NewFSModule(security)

	_, err := module.fsRead(nonExistent)
	if err == nil {
		t.Fatal("expected error for non-existent file")
	}
	if !strings.Contains(err.Error(), "no such file") && !strings.Contains(err.Error(), "cannot find") {
		t.Errorf("expected file not found error, got: %v", err)
	}
}

func TestFSModule_ReadDirectory(t *testing.T) {
	tmpDir := t.TempDir()

	security := DefaultSecurityConfig()
	security.FS.AllowedPaths = nil
	module := NewFSModule(security)

	_, err := module.fsRead(tmpDir)
	if err == nil {
		t.Fatal("expected error when reading directory")
	}
	if !strings.Contains(err.Error(), "cannot read directory") {
		t.Errorf("expected directory error, got: %v", err)
	}
}

func TestFSModule_Write(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "write_test.txt")
	content := "Test content"

	security := DefaultSecurityConfig()
	security.FS.AllowedPaths = nil
	security.FS.AllowWrite = true
	module := NewFSModule(security)

	_, err := module.fsWrite(testFile, content)
	if err != nil {
		t.Fatalf("fsWrite failed: %v", err)
	}

	data, err := os.ReadFile(testFile)
	if err != nil {
		t.Fatalf("failed to read written file: %v", err)
	}

	if string(data) != content {
		t.Errorf("expected %q, got %q", content, string(data))
	}
}

func TestFSModule_WriteDisabled(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")

	security := DefaultSecurityConfig()
	security.FS.AllowedPaths = nil
	security.FS.AllowWrite = false
	module := NewFSModule(security)

	_, err := module.fsWrite(testFile, "content")
	if err == nil {
		t.Fatal("expected error when write is disabled")
	}
	if !strings.Contains(err.Error(), "write operations disabled") {
		t.Errorf("expected write disabled error, got: %v", err)
	}
}

func TestFSModule_WriteEmpty(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "empty.txt")

	security := DefaultSecurityConfig()
	security.FS.AllowedPaths = nil
	security.FS.AllowWrite = true
	module := NewFSModule(security)

	_, err := module.fsWrite(testFile, "")
	if err != nil {
		t.Fatalf("fsWrite with empty content failed: %v", err)
	}

	data, err := os.ReadFile(testFile)
	if err != nil {
		t.Fatalf("failed to read file: %v", err)
	}

	if len(data) != 0 {
		t.Errorf("expected empty file, got %d bytes", len(data))
	}
}

func TestFSModule_Append(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "append_test.txt")

	security := DefaultSecurityConfig()
	security.FS.AllowedPaths = nil
	security.FS.AllowWrite = true
	module := NewFSModule(security)

	if err := os.WriteFile(testFile, []byte("Line 1\n"), 0644); err != nil {
		t.Fatalf("failed to create initial file: %v", err)
	}

	_, err := module.fsAppend(testFile, "Line 2\n")
	if err != nil {
		t.Fatalf("fsAppend failed: %v", err)
	}

	data, err := os.ReadFile(testFile)
	if err != nil {
		t.Fatalf("failed to read file: %v", err)
	}

	expected := "Line 1\nLine 2\n"
	if string(data) != expected {
		t.Errorf("expected %q, got %q", expected, string(data))
	}
}

func TestFSModule_AppendCreatesFile(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "new_file.txt")

	security := DefaultSecurityConfig()
	security.FS.AllowedPaths = nil
	security.FS.AllowWrite = true
	module := NewFSModule(security)

	_, err := module.fsAppend(testFile, "First line\n")
	if err != nil {
		t.Fatalf("fsAppend failed: %v", err)
	}

	data, err := os.ReadFile(testFile)
	if err != nil {
		t.Fatalf("failed to read file: %v", err)
	}

	if string(data) != "First line\n" {
		t.Errorf("expected 'First line\\n', got %q", string(data))
	}
}

func TestFSModule_Exists(t *testing.T) {
	tmpDir := t.TempDir()
	existingFile := filepath.Join(tmpDir, "exists.txt")
	nonExistentFile := filepath.Join(tmpDir, "nonexistent.txt")

	if err := os.WriteFile(existingFile, []byte("test"), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	security := DefaultSecurityConfig()
	security.FS.AllowedPaths = nil
	module := NewFSModule(security)

	result, err := module.fsExists(existingFile)
	if err != nil {
		t.Fatalf("fsExists failed: %v", err)
	}
	if result != true {
		t.Error("expected true for existing file")
	}

	result, err = module.fsExists(nonExistentFile)
	if err != nil {
		t.Fatalf("fsExists failed: %v", err)
	}
	if result != false {
		t.Error("expected false for non-existent file")
	}
}

func TestFSModule_List(t *testing.T) {
	tmpDir := t.TempDir()

	files := []string{"file1.txt", "file2.txt", "file3.txt"}
	for _, f := range files {
		path := filepath.Join(tmpDir, f)
		if err := os.WriteFile(path, []byte("test"), 0644); err != nil {
			t.Fatalf("failed to create test file: %v", err)
		}
	}

	security := DefaultSecurityConfig()
	security.FS.AllowedPaths = nil
	module := NewFSModule(security)

	result, err := module.fsList(tmpDir)
	if err != nil {
		t.Fatalf("fsList failed: %v", err)
	}

	entries, ok := result.([]any)
	if !ok {
		t.Fatalf("expected []any, got %T", result)
	}

	if len(entries) != 3 {
		t.Errorf("expected 3 entries, got %d", len(entries))
	}

	found := make(map[string]bool)
	for _, entry := range entries {
		name, ok := entry.(string)
		if !ok {
			t.Errorf("expected string entry, got %T", entry)
			continue
		}
		found[name] = true
	}

	for _, f := range files {
		if !found[f] {
			t.Errorf("expected to find %s in list", f)
		}
	}
}

func TestFSModule_Mkdir(t *testing.T) {
	tmpDir := t.TempDir()
	newDir := filepath.Join(tmpDir, "subdir")

	security := DefaultSecurityConfig()
	security.FS.AllowedPaths = nil
	security.FS.AllowWrite = true
	module := NewFSModule(security)

	_, err := module.fsMkdir(newDir)
	if err != nil {
		t.Fatalf("fsMkdir failed: %v", err)
	}

	info, err := os.Stat(newDir)
	if err != nil {
		t.Fatalf("directory not created: %v", err)
	}

	if !info.IsDir() {
		t.Error("expected directory, got file")
	}
}

func TestFSModule_MkdirNested(t *testing.T) {
	tmpDir := t.TempDir()
	nestedDir := filepath.Join(tmpDir, "a", "b", "c")

	security := DefaultSecurityConfig()
	security.FS.AllowedPaths = nil
	security.FS.AllowWrite = true
	module := NewFSModule(security)

	_, err := module.fsMkdir(nestedDir)
	if err != nil {
		t.Fatalf("fsMkdir nested failed: %v", err)
	}

	info, err := os.Stat(nestedDir)
	if err != nil {
		t.Fatalf("nested directory not created: %v", err)
	}

	if !info.IsDir() {
		t.Error("expected directory, got file")
	}
}

func TestFSModule_Remove(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "remove_test.txt")

	if err := os.WriteFile(testFile, []byte("test"), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	security := DefaultSecurityConfig()
	security.FS.AllowedPaths = nil
	security.FS.AllowWrite = true
	module := NewFSModule(security)

	_, err := module.fsRemove(testFile)
	if err != nil {
		t.Fatalf("fsRemove failed: %v", err)
	}

	_, err = os.Stat(testFile)
	if !os.IsNotExist(err) {
		t.Error("file should have been removed")
	}
}

func TestFSModule_RemoveEmptyDir(t *testing.T) {
	tmpDir := t.TempDir()
	emptyDir := filepath.Join(tmpDir, "empty")

	if err := os.Mkdir(emptyDir, 0755); err != nil {
		t.Fatalf("failed to create directory: %v", err)
	}

	security := DefaultSecurityConfig()
	security.FS.AllowedPaths = nil
	security.FS.AllowWrite = true
	module := NewFSModule(security)

	_, err := module.fsRemove(emptyDir)
	if err != nil {
		t.Fatalf("fsRemove empty dir failed: %v", err)
	}

	_, err = os.Stat(emptyDir)
	if !os.IsNotExist(err) {
		t.Error("directory should have been removed")
	}
}

func TestFSModule_RemoveNonEmptyDirWithoutRecursive(t *testing.T) {
	tmpDir := t.TempDir()
	nonEmptyDir := filepath.Join(tmpDir, "nonempty")
	testFile := filepath.Join(nonEmptyDir, "file.txt")

	if err := os.Mkdir(nonEmptyDir, 0755); err != nil {
		t.Fatalf("failed to create directory: %v", err)
	}
	if err := os.WriteFile(testFile, []byte("test"), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	security := DefaultSecurityConfig()
	security.FS.AllowedPaths = nil
	security.FS.AllowWrite = true
	module := NewFSModule(security)

	_, err := module.fsRemove(nonEmptyDir)
	if err == nil {
		t.Fatal("expected error when removing non-empty directory without recursive")
	}
	if !strings.Contains(err.Error(), "not empty") {
		t.Errorf("expected 'not empty' error, got: %v", err)
	}
}

func TestFSModule_RemoveRecursive(t *testing.T) {
	tmpDir := t.TempDir()
	nonEmptyDir := filepath.Join(tmpDir, "nonempty")
	testFile := filepath.Join(nonEmptyDir, "file.txt")

	if err := os.Mkdir(nonEmptyDir, 0755); err != nil {
		t.Fatalf("failed to create directory: %v", err)
	}
	if err := os.WriteFile(testFile, []byte("test"), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	security := DefaultSecurityConfig()
	security.FS.AllowedPaths = nil
	security.FS.AllowWrite = true
	module := NewFSModule(security)

	_, err := module.fsRemove(nonEmptyDir, true)
	if err != nil {
		t.Fatalf("fsRemove recursive failed: %v", err)
	}

	_, err = os.Stat(nonEmptyDir)
	if !os.IsNotExist(err) {
		t.Error("directory should have been removed")
	}
}

func TestFSModule_SecurityPathTraversal(t *testing.T) {
	tmpDir := t.TempDir()

	security := DefaultSecurityConfig()
	security.FS.AllowedPaths = []string{tmpDir}
	module := NewFSModule(security)

	_, err := module.fsRead("../../etc/passwd")
	if err == nil {
		t.Fatal("expected error for path traversal attempt")
	}
	if !strings.Contains(err.Error(), "not in allowed paths") {
		t.Errorf("expected security error, got: %v", err)
	}
}

func TestFSModule_SecurityBlockedPath(t *testing.T) {
	security := DefaultSecurityConfig()
	security.FS.BlockedPaths = []string{filepath.Join(t.TempDir(), "blocked") + string(filepath.Separator)}
	blockedFile := filepath.Join(security.FS.BlockedPaths[0], "secret.txt")
	os.MkdirAll(security.FS.BlockedPaths[0], 0755)
	os.WriteFile(blockedFile, []byte("secret"), 0644)
	module := NewFSModule(security)

	_, err := module.fsRead(blockedFile)
	if err == nil {
		t.Fatal("expected error for blocked path")
	}
	if !strings.Contains(err.Error(), "blocked") {
		t.Errorf("expected blocked path error, got: %v", err)
	}
}

func TestFSModule_Base64Encoding(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "base64_test.txt")
	content := "Hello, Base64!"

	security := DefaultSecurityConfig()
	security.FS.AllowedPaths = nil
	security.FS.AllowWrite = true
	module := NewFSModule(security)

	encoded := base64.StdEncoding.EncodeToString([]byte(content))
	_, err := module.fsWrite(testFile, encoded, "base64")
	if err != nil {
		t.Fatalf("fsWrite with base64 failed: %v", err)
	}

	result, err := module.fsRead(testFile, "base64")
	if err != nil {
		t.Fatalf("fsRead with base64 failed: %v", err)
	}

	if result != encoded {
		t.Errorf("expected %q, got %q", encoded, result)
	}

	resultUTF8, err := module.fsRead(testFile)
	if err != nil {
		t.Fatalf("fsRead UTF-8 failed: %v", err)
	}

	if resultUTF8 != content {
		t.Errorf("expected %q, got %q", content, resultUTF8)
	}
}

func TestFSModule_Stat(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "stat_test.txt")
	content := "Test content for stat"

	if err := os.WriteFile(testFile, []byte(content), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	security := DefaultSecurityConfig()
	security.FS.AllowedPaths = nil
	module := NewFSModule(security)

	result, err := module.fsStat(testFile)
	if err != nil {
		t.Fatalf("fsStat failed: %v", err)
	}

	stat, ok := result.(map[string]any)
	if !ok {
		t.Fatalf("expected map result, got %T", result)
	}

	if stat["name"] != "stat_test.txt" {
		t.Errorf("expected name 'stat_test.txt', got %v", stat["name"])
	}

	if stat["size"] != int64(len(content)) {
		t.Errorf("expected size %d, got %v", len(content), stat["size"])
	}

	if stat["is_file"] != true {
		t.Error("expected is_file to be true")
	}

	if stat["is_dir"] != false {
		t.Error("expected is_dir to be false")
	}

	if stat["ext"] != ".txt" {
		t.Errorf("expected ext '.txt', got %v", stat["ext"])
	}
}

func TestFSModule_StatDirectory(t *testing.T) {
	tmpDir := t.TempDir()

	security := DefaultSecurityConfig()
	security.FS.AllowedPaths = nil
	module := NewFSModule(security)

	result, err := module.fsStat(tmpDir)
	if err != nil {
		t.Fatalf("fsStat failed: %v", err)
	}

	stat, ok := result.(map[string]any)
	if !ok {
		t.Fatalf("expected map result, got %T", result)
	}

	if stat["is_dir"] != true {
		t.Error("expected is_dir to be true")
	}

	if stat["is_file"] != false {
		t.Error("expected is_file to be false")
	}
}

func TestFSModule_Copy(t *testing.T) {
	tmpDir := t.TempDir()
	srcFile := filepath.Join(tmpDir, "source.txt")
	dstFile := filepath.Join(tmpDir, "destination.txt")
	content := "Content to copy"

	if err := os.WriteFile(srcFile, []byte(content), 0644); err != nil {
		t.Fatalf("failed to create source file: %v", err)
	}

	security := DefaultSecurityConfig()
	security.FS.AllowedPaths = nil
	security.FS.AllowWrite = true
	module := NewFSModule(security)

	_, err := module.fsCopy(srcFile, dstFile)
	if err != nil {
		t.Fatalf("fsCopy failed: %v", err)
	}

	data, err := os.ReadFile(dstFile)
	if err != nil {
		t.Fatalf("failed to read destination file: %v", err)
	}

	if string(data) != content {
		t.Errorf("expected %q, got %q", content, string(data))
	}

	srcData, err := os.ReadFile(srcFile)
	if err != nil {
		t.Fatalf("source file should still exist: %v", err)
	}

	if string(srcData) != content {
		t.Error("source file content should be unchanged")
	}
}

func TestFSModule_Move(t *testing.T) {
	tmpDir := t.TempDir()
	srcFile := filepath.Join(tmpDir, "source.txt")
	dstFile := filepath.Join(tmpDir, "destination.txt")
	content := "Content to move"

	if err := os.WriteFile(srcFile, []byte(content), 0644); err != nil {
		t.Fatalf("failed to create source file: %v", err)
	}

	security := DefaultSecurityConfig()
	security.FS.AllowedPaths = nil
	security.FS.AllowWrite = true
	module := NewFSModule(security)

	_, err := module.fsMove(srcFile, dstFile)
	if err != nil {
		t.Fatalf("fsMove failed: %v", err)
	}

	data, err := os.ReadFile(dstFile)
	if err != nil {
		t.Fatalf("failed to read destination file: %v", err)
	}

	if string(data) != content {
		t.Errorf("expected %q, got %q", content, string(data))
	}

	_, err = os.Stat(srcFile)
	if !os.IsNotExist(err) {
		t.Error("source file should have been removed")
	}
}

func TestFSModule_Glob(t *testing.T) {
	tmpDir := t.TempDir()

	files := []string{"test1.txt", "test2.txt", "data.json", "readme.md"}
	for _, f := range files {
		path := filepath.Join(tmpDir, f)
		if err := os.WriteFile(path, []byte("test"), 0644); err != nil {
			t.Fatalf("failed to create test file: %v", err)
		}
	}

	security := DefaultSecurityConfig()
	security.FS.AllowedPaths = nil
	module := NewFSModule(security)

	pattern := filepath.Join(tmpDir, "*.txt")
	result, err := module.fsGlob(pattern)
	if err != nil {
		t.Fatalf("fsGlob failed: %v", err)
	}

	matches, ok := result.([]any)
	if !ok {
		t.Fatalf("expected []any, got %T", result)
	}

	if len(matches) != 2 {
		t.Errorf("expected 2 matches, got %d", len(matches))
	}

	found := make(map[string]bool)
	for _, match := range matches {
		path, ok := match.(string)
		if !ok {
			t.Errorf("expected string match, got %T", match)
			continue
		}
		found[filepath.Base(path)] = true
	}

	if !found["test1.txt"] || !found["test2.txt"] {
		t.Error("expected to find test1.txt and test2.txt")
	}
}

func TestFSModule_MaxFileSize(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "large.txt")
	largeContent := strings.Repeat("x", 2048)

	if err := os.WriteFile(testFile, []byte(largeContent), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	security := DefaultSecurityConfig()
	security.FS.AllowedPaths = nil
	security.FS.MaxFileSize = 1024
	module := NewFSModule(security)

	_, err := module.fsRead(testFile)
	if err == nil {
		t.Fatal("expected error for file exceeding max size")
	}
	if !strings.Contains(err.Error(), "exceeds max size") {
		t.Errorf("expected max size error, got: %v", err)
	}
}

func TestFSModule_WriteMaxFileSize(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "large.txt")
	largeContent := strings.Repeat("x", 2048)

	security := DefaultSecurityConfig()
	security.FS.AllowedPaths = nil
	security.FS.AllowWrite = true
	security.FS.MaxFileSize = 1024
	module := NewFSModule(security)

	_, err := module.fsWrite(testFile, largeContent)
	if err == nil {
		t.Fatal("expected error for content exceeding max size")
	}
	if !strings.Contains(err.Error(), "exceeds max size") {
		t.Errorf("expected max size error, got: %v", err)
	}
}

func TestFSModule_MissingPath(t *testing.T) {
	module := NewFSModule(DefaultSecurityConfig())

	_, err := module.fsRead()
	if err == nil {
		t.Fatal("expected error for missing path")
	}
	if !strings.Contains(err.Error(), "path is required") {
		t.Errorf("expected 'path is required' error, got: %v", err)
	}
}

func TestFSModule_InvalidPathType(t *testing.T) {
	module := NewFSModule(DefaultSecurityConfig())

	_, err := module.fsRead(123)
	if err == nil {
		t.Fatal("expected error for invalid path type")
	}
	if !strings.Contains(err.Error(), "path must be a string") {
		t.Errorf("expected 'path must be a string' error, got: %v", err)
	}
}
