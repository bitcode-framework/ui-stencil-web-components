package server

import (
	"testing"

	"github.com/bitcode-framework/go-json/lang"
)

func TestDefaultServerConfig(t *testing.T) {
	cfg := DefaultServerConfig()
	if cfg.Framework != "fiber" {
		t.Errorf("expected fiber, got %s", cfg.Framework)
	}
	if cfg.Port != 3000 {
		t.Errorf("expected 3000, got %d", cfg.Port)
	}
	if cfg.Host != "0.0.0.0" {
		t.Errorf("expected 0.0.0.0, got %s", cfg.Host)
	}
	if cfg.MaxBodySize != "10mb" {
		t.Errorf("expected 10mb, got %s", cfg.MaxBodySize)
	}
}

func TestMergeDefaults(t *testing.T) {
	cfg := &lang.ServerConfig{Port: 8080}
	MergeDefaults(cfg)
	if cfg.Framework != "fiber" {
		t.Errorf("expected fiber, got %s", cfg.Framework)
	}
	if cfg.Port != 8080 {
		t.Errorf("expected 8080, got %d", cfg.Port)
	}
}

func TestValidate_ValidConfig(t *testing.T) {
	cfg := DefaultServerConfig()
	if err := Validate(cfg); err != nil {
		t.Errorf("expected no error, got %v", err)
	}
}

func TestValidate_InvalidFramework(t *testing.T) {
	cfg := DefaultServerConfig()
	cfg.Framework = "invalid"
	if err := Validate(cfg); err == nil {
		t.Error("expected error for invalid framework")
	}
}

func TestValidate_InvalidPort(t *testing.T) {
	cfg := DefaultServerConfig()
	cfg.Port = 0
	if err := Validate(cfg); err == nil {
		t.Error("expected error for port 0")
	}
	cfg.Port = 70000
	if err := Validate(cfg); err == nil {
		t.Error("expected error for port 70000")
	}
}

func TestValidate_InvalidTimeout(t *testing.T) {
	cfg := DefaultServerConfig()
	cfg.ReadTimeout = "invalid"
	if err := Validate(cfg); err == nil {
		t.Error("expected error for invalid timeout")
	}
}

func TestValidate_JWTConfig(t *testing.T) {
	cfg := DefaultServerConfig()
	cfg.JWT = &lang.JWTConfig{SecretEnv: "JWT_SECRET", Algorithm: "HS256"}
	if err := Validate(cfg); err != nil {
		t.Errorf("expected no error, got %v", err)
	}

	cfg.JWT.Algorithm = "RS256"
	if err := Validate(cfg); err == nil {
		t.Error("expected error for unsupported algorithm")
	}
}

func TestValidate_RateLimitConfig(t *testing.T) {
	cfg := DefaultServerConfig()
	cfg.RateLimit = &lang.RateLimitConfig{Requests: 100, Window: "1m"}
	if err := Validate(cfg); err != nil {
		t.Errorf("expected no error, got %v", err)
	}

	cfg.RateLimit.Requests = 0
	if err := Validate(cfg); err == nil {
		t.Error("expected error for zero requests")
	}
}

func TestValidate_AuthConfig(t *testing.T) {
	cfg := DefaultServerConfig()
	cfg.Auth = &lang.AuthConfig{
		Default: "jwt",
		Strategies: map[string]*lang.StrategyConfig{
			"jwt": {Type: "bearer", SecretEnv: "JWT_SECRET"},
		},
	}
	if err := Validate(cfg); err != nil {
		t.Errorf("expected no error, got %v", err)
	}

	cfg.Auth.Default = "nonexistent"
	if err := Validate(cfg); err == nil {
		t.Error("expected error for missing default strategy")
	}
}

func TestParseByteSize(t *testing.T) {
	tests := []struct {
		input    string
		expected int64
	}{
		{"10mb", 10 * 1024 * 1024},
		{"1gb", 1024 * 1024 * 1024},
		{"512kb", 512 * 1024},
		{"100b", 100},
		{"1024", 1024},
	}
	for _, tt := range tests {
		result, err := parseByteSize(tt.input)
		if err != nil {
			t.Errorf("parseByteSize(%q): %v", tt.input, err)
			continue
		}
		if result != tt.expected {
			t.Errorf("parseByteSize(%q) = %d, want %d", tt.input, result, tt.expected)
		}
	}
}
