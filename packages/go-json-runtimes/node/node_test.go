package node

import (
	"testing"

	runtimes "github.com/bitcode-framework/go-json-runtimes"
)

func TestNodeRuntime_Name(t *testing.T) {
	rt := New()
	if rt.Name() != "node" {
		t.Errorf("expected 'node', got %q", rt.Name())
	}
}

func TestNodeRuntime_NewAuto(t *testing.T) {
	rt := NewAuto()
	if rt.config.Enabled != "auto" {
		t.Errorf("expected enabled='auto', got %q", rt.config.Enabled)
	}
	if rt.config.Command != "node" {
		t.Errorf("expected command='node', got %q", rt.config.Command)
	}
}

func TestNodeRuntime_DisabledValidation(t *testing.T) {
	rt := New(runtimes.NodeConfig{Enabled: "false"})
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
		{"20.0.0", "20.0", true},
		{"22.1.0", "20.0", true},
		{"18.0.0", "20.0", false},
		{"20.0.0", "20.0.0", true},
		{"3.12.0", "3.10.0", true},
		{"3.8.0", "3.10.0", false},
	}

	for _, tt := range tests {
		got := meetsMinVersion(tt.current, tt.minimum)
		if got != tt.expected {
			t.Errorf("meetsMinVersion(%q, %q) = %v, want %v", tt.current, tt.minimum, got, tt.expected)
		}
	}
}

func TestNodeRuntime_Close(t *testing.T) {
	rt := New()
	if err := rt.Close(); err != nil {
		t.Fatalf("Close error: %v", err)
	}
}
