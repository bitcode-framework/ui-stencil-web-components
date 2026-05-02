package python

import (
	"testing"

	runtimes "github.com/bitcode-framework/go-json-runtimes"
)

func TestPythonRuntime_Name(t *testing.T) {
	rt := New()
	if rt.Name() != "python" {
		t.Errorf("expected 'python', got %q", rt.Name())
	}
}

func TestPythonRuntime_NewAuto(t *testing.T) {
	rt := NewAuto()
	if rt.config.Enabled != "auto" {
		t.Errorf("expected enabled='auto', got %q", rt.config.Enabled)
	}
}

func TestPythonRuntime_DisabledValidation(t *testing.T) {
	rt := New(runtimes.PythonConfig{Enabled: "false"})
	err := rt.Validate()
	if err == nil {
		t.Fatal("expected error for disabled runtime")
	}
}

func TestMeetsMinVersion(t *testing.T) {
	tests := []struct {
		current  string
		minimum  string
		expected bool
	}{
		{"3.12.0", "3.10.0", true},
		{"3.10.0", "3.10.0", true},
		{"3.8.0", "3.10.0", false},
		{"3.11.5", "3.10.0", true},
	}

	for _, tt := range tests {
		got := meetsMinVersion(tt.current, tt.minimum)
		if got != tt.expected {
			t.Errorf("meetsMinVersion(%q, %q) = %v, want %v", tt.current, tt.minimum, got, tt.expected)
		}
	}
}

func TestPythonRuntime_Close(t *testing.T) {
	rt := New()
	if err := rt.Close(); err != nil {
		t.Fatalf("Close error: %v", err)
	}
}
