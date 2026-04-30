package io

import (
	"os"
	"runtime"
	"strings"
	"testing"
)

func TestExecModule_SimpleCommand(t *testing.T) {
	security := DefaultSecurityConfig()
	security.Exec.AllowedCommands = []string{"go"}

	m := NewExecModule(security)

	result, err := m.execRun("go", map[string]any{"args": []any{"version"}})
	if err != nil {
		t.Fatalf("go version failed: %v", err)
	}

	resultMap, ok := result.(map[string]any)
	if !ok {
		t.Fatalf("Expected map result, got %T", result)
	}

	exitCode := resultMap["exit_code"].(int)
	if exitCode != 0 {
		t.Errorf("Expected exit_code=0, got %d", exitCode)
	}

	stdout := resultMap["stdout"].(string)
	if !strings.Contains(stdout, "go version") {
		t.Errorf("Expected 'go version' in stdout, got: %s", stdout)
	}
}

func TestExecModule_CaptureStdout(t *testing.T) {
	security := DefaultSecurityConfig()

	var cmdName string
	var args []any

	if runtime.GOOS == "windows" {
		cmdName = "cmd"
		args = []any{"/c", "echo", "hello world"}
		security.Exec.AllowedCommands = []string{"cmd"}
	} else {
		cmdName = "echo"
		args = []any{"hello world"}
		security.Exec.AllowedCommands = []string{"echo"}
	}

	m := NewExecModule(security)

	result, err := m.execRun(cmdName, map[string]any{"args": args})
	if err != nil {
		t.Fatalf("echo failed: %v", err)
	}

	resultMap := result.(map[string]any)
	stdout := resultMap["stdout"].(string)

	if !strings.Contains(stdout, "hello world") {
		t.Errorf("Expected 'hello world' in stdout, got: %s", stdout)
	}
}

func TestExecModule_NonZeroExitCode(t *testing.T) {
	security := DefaultSecurityConfig()

	var cmdName string
	var args []any

	if runtime.GOOS == "windows" {
		cmdName = "cmd"
		args = []any{"/c", "exit", "42"}
		security.Exec.AllowedCommands = []string{"cmd"}
	} else {
		cmdName = "sh"
		args = []any{"-c", "exit 42"}
		security.Exec.AllowedCommands = []string{"sh"}
	}

	m := NewExecModule(security)

	result, err := m.execRun(cmdName, map[string]any{"args": args})
	if err != nil {
		t.Fatalf("Command failed: %v", err)
	}

	resultMap := result.(map[string]any)
	exitCode := resultMap["exit_code"].(int)

	if exitCode != 42 {
		t.Errorf("Expected exit_code=42, got %d", exitCode)
	}
}

func TestExecModule_NotInWhitelist(t *testing.T) {
	security := DefaultSecurityConfig()
	security.Exec.AllowedCommands = []string{"go"}

	m := NewExecModule(security)

	_, err := m.execRun("python", map[string]any{"args": []any{"--version"}})
	if err == nil {
		t.Fatal("Expected error for command not in whitelist, got nil")
	}

	if !strings.Contains(err.Error(), "not in allowed list") {
		t.Errorf("Expected 'not in allowed list' error, got: %v", err)
	}
}

func TestExecModule_DeniedCommand(t *testing.T) {
	security := DefaultSecurityConfig()
	security.Exec.AllowedCommands = []string{"rm"}

	m := NewExecModule(security)

	_, err := m.execRun("rm", map[string]any{"args": []any{"-rf", "/"}})
	if err == nil {
		t.Fatal("Expected error for denied command, got nil")
	}

	if !strings.Contains(err.Error(), "permanently blocked") {
		t.Errorf("Expected 'permanently blocked' error, got: %v", err)
	}
}

func TestExecModule_Timeout(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping timeout test in short mode")
	}

	security := DefaultSecurityConfig()
	security.Exec.MaxTimeout = 1

	var cmdName string
	var args []any

	if runtime.GOOS == "windows" {
		cmdName = "ping"
		args = []any{"-n", "30", "127.0.0.1"}
		security.Exec.AllowedCommands = []string{"ping"}
	} else {
		cmdName = "sleep"
		args = []any{"30"}
		security.Exec.AllowedCommands = []string{"sleep"}
	}

	m := NewExecModule(security)

	_, err := m.execRun(cmdName, map[string]any{"args": args, "timeout": 1})
	if err == nil {
		t.Fatal("Expected timeout error, got nil")
	}

	if !strings.Contains(err.Error(), "timed out") {
		t.Errorf("Expected 'timed out' error, got: %v", err)
	}
}

func TestExecModule_EnvIsolation(t *testing.T) {
	security := DefaultSecurityConfig()

	var cmdName string
	var args []any

	if runtime.GOOS == "windows" {
		cmdName = "cmd"
		args = []any{"/c", "echo", "%CUSTOM_VAR%"}
		security.Exec.AllowedCommands = []string{"cmd"}
	} else {
		cmdName = "sh"
		args = []any{"-c", "echo $CUSTOM_VAR"}
		security.Exec.AllowedCommands = []string{"sh"}
	}

	m := NewExecModule(security)

	customEnv := map[string]any{
		"CUSTOM_VAR": "test_value",
	}

	result, err := m.execRun(cmdName, map[string]any{
		"args": args,
		"env":  customEnv,
	})
	if err != nil {
		t.Fatalf("Command failed: %v", err)
	}

	resultMap := result.(map[string]any)
	stdout := resultMap["stdout"].(string)

	if !strings.Contains(stdout, "test_value") {
		t.Errorf("Expected 'test_value' in stdout, got: %s", stdout)
	}
}

func TestExecModule_EngineSecretsStripped(t *testing.T) {
	security := DefaultSecurityConfig()

	var cmdName string
	var args []any

	if runtime.GOOS == "windows" {
		cmdName = "cmd"
		args = []any{"/c", "set"}
		security.Exec.AllowedCommands = []string{"cmd"}
	} else {
		cmdName = "env"
		args = []any{}
		security.Exec.AllowedCommands = []string{"env"}
	}

	m := NewExecModule(security)

	os.Setenv("JWT_SECRET", "secret123")
	os.Setenv("DB_PASSWORD", "password123")
	defer func() {
		os.Unsetenv("JWT_SECRET")
		os.Unsetenv("DB_PASSWORD")
	}()

	result, err := m.execRun(cmdName, map[string]any{"args": args})
	if err != nil {
		t.Fatalf("Command failed: %v", err)
	}

	resultMap := result.(map[string]any)
	stdout := resultMap["stdout"].(string)

	if strings.Contains(stdout, "JWT_SECRET") {
		t.Error("JWT_SECRET should be stripped from environment")
	}

	if strings.Contains(stdout, "DB_PASSWORD") {
		t.Error("DB_PASSWORD should be stripped from environment")
	}
}

func TestExecModule_MaxOutputSize(t *testing.T) {
	security := DefaultSecurityConfig()
	security.Exec.MaxOutputSize = 100

	var cmdName string
	var args []any

	if runtime.GOOS == "windows" {
		cmdName = "cmd"
		args = []any{"/c", "echo", strings.Repeat("A", 200)}
		security.Exec.AllowedCommands = []string{"cmd"}
	} else {
		cmdName = "echo"
		args = []any{strings.Repeat("A", 200)}
		security.Exec.AllowedCommands = []string{"echo"}
	}

	m := NewExecModule(security)

	result, err := m.execRun(cmdName, map[string]any{"args": args})
	if err != nil {
		t.Fatalf("Command failed: %v", err)
	}

	resultMap := result.(map[string]any)
	stdout := resultMap["stdout"].(string)

	if len(stdout) > 100 {
		t.Errorf("Expected stdout truncated to 100 chars, got %d", len(stdout))
	}

	truncated, ok := resultMap["truncated"].(bool)
	if !ok || !truncated {
		t.Error("Expected truncated=true in result")
	}
}
