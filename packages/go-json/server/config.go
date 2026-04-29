package server

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/bitcode-framework/go-json/lang"
)

var validFrameworks = map[string]bool{
	"fiber":   true,
	"nethttp": true,
	"echo":    true,
	"gin":     true,
	"chi":     true,
}

var validJWTAlgorithms = map[string]bool{
	"HS256": true,
	"HS384": true,
	"HS512": true,
}

var validRateLimitBy = map[string]bool{
	"ip":   true,
	"user": true,
}

var validAuthStrategyTypes = map[string]bool{
	"bearer":  true,
	"apikey":  true,
	"basic":   true,
	"custom":  true,
}

// DefaultServerConfig returns a ServerConfig with sensible defaults.
func DefaultServerConfig() *lang.ServerConfig {
	return &lang.ServerConfig{
		Framework:        "fiber",
		Port:             3000,
		Host:             "0.0.0.0",
		GracefulShutdown: "10s",
		ReadTimeout:      "30s",
		WriteTimeout:     "30s",
		MaxBodySize:      "10mb",
	}
}

// MergeDefaults fills zero-value fields in cfg with defaults.
func MergeDefaults(cfg *lang.ServerConfig) {
	defaults := DefaultServerConfig()
	if cfg.Framework == "" {
		cfg.Framework = defaults.Framework
	}
	if cfg.Port == 0 {
		cfg.Port = defaults.Port
	}
	if cfg.Host == "" {
		cfg.Host = defaults.Host
	}
	if cfg.GracefulShutdown == "" {
		cfg.GracefulShutdown = defaults.GracefulShutdown
	}
	if cfg.ReadTimeout == "" {
		cfg.ReadTimeout = defaults.ReadTimeout
	}
	if cfg.WriteTimeout == "" {
		cfg.WriteTimeout = defaults.WriteTimeout
	}
	if cfg.MaxBodySize == "" {
		cfg.MaxBodySize = defaults.MaxBodySize
	}
}

// Validate checks the server config for errors.
func Validate(cfg *lang.ServerConfig) error {
	if !validFrameworks[cfg.Framework] {
		return fmt.Errorf("unsupported framework: %q (valid: fiber, nethttp, echo, gin, chi)", cfg.Framework)
	}

	if cfg.Port < 1 || cfg.Port > 65535 {
		return fmt.Errorf("port must be between 1 and 65535, got %d", cfg.Port)
	}

	if _, err := parseDuration(cfg.GracefulShutdown); err != nil {
		return fmt.Errorf("invalid graceful_shutdown duration: %w", err)
	}
	if _, err := parseDuration(cfg.ReadTimeout); err != nil {
		return fmt.Errorf("invalid read_timeout duration: %w", err)
	}
	if _, err := parseDuration(cfg.WriteTimeout); err != nil {
		return fmt.Errorf("invalid write_timeout duration: %w", err)
	}
	if _, err := parseByteSize(cfg.MaxBodySize); err != nil {
		return fmt.Errorf("invalid max_body_size: %w", err)
	}

	if cfg.JWT != nil {
		if err := validateJWTConfig(cfg.JWT); err != nil {
			return fmt.Errorf("jwt config: %w", err)
		}
	}

	if cfg.Auth != nil {
		if err := validateAuthConfig(cfg.Auth); err != nil {
			return fmt.Errorf("auth config: %w", err)
		}
	}

	if cfg.RateLimit != nil {
		if err := validateRateLimitConfig(cfg.RateLimit); err != nil {
			return fmt.Errorf("rate_limit config: %w", err)
		}
	}

	if cfg.CORS != nil {
		if len(cfg.CORS.Origins) == 0 {
			return fmt.Errorf("cors.origins must not be empty when cors is configured")
		}
	}

	return nil
}

func validateJWTConfig(cfg *lang.JWTConfig) error {
	if cfg.SecretEnv == "" {
		return fmt.Errorf("secret_env is required")
	}
	if cfg.Algorithm != "" && !validJWTAlgorithms[cfg.Algorithm] {
		return fmt.Errorf("unsupported algorithm: %q (valid: HS256, HS384, HS512)", cfg.Algorithm)
	}
	if cfg.Expiry != "" {
		if _, err := parseDuration(cfg.Expiry); err != nil {
			return fmt.Errorf("invalid expiry duration: %w", err)
		}
	}
	return nil
}

func validateAuthConfig(cfg *lang.AuthConfig) error {
	if cfg.Default != "" && cfg.Strategies != nil {
		if _, ok := cfg.Strategies[cfg.Default]; !ok {
			return fmt.Errorf("default strategy %q not found in strategies", cfg.Default)
		}
	}
	for name, sc := range cfg.Strategies {
		if sc.Type == "" {
			return fmt.Errorf("strategy %q: type is required", name)
		}
		if !validAuthStrategyTypes[sc.Type] {
			return fmt.Errorf("strategy %q: unsupported type %q (valid: bearer, apikey, basic, custom)", name, sc.Type)
		}
		switch sc.Type {
		case "bearer":
			if sc.SecretEnv == "" {
				return fmt.Errorf("strategy %q (bearer): secret_env is required", name)
			}
		case "apikey":
			if sc.KeysEnv == "" {
				return fmt.Errorf("strategy %q (apikey): keys_env is required", name)
			}
		case "basic":
			if sc.UsersEnv == "" {
				return fmt.Errorf("strategy %q (basic): users_env is required", name)
			}
		case "custom":
			if sc.Handler == "" {
				return fmt.Errorf("strategy %q (custom): handler is required", name)
			}
		}
	}
	return nil
}

func validateRateLimitConfig(cfg *lang.RateLimitConfig) error {
	if cfg.Requests <= 0 {
		return fmt.Errorf("requests must be positive, got %d", cfg.Requests)
	}
	if cfg.Window == "" {
		return fmt.Errorf("window is required")
	}
	if _, err := parseDuration(cfg.Window); err != nil {
		return fmt.Errorf("invalid window duration: %w", err)
	}
	if cfg.By != "" && !validRateLimitBy[cfg.By] {
		return fmt.Errorf("unsupported by: %q (valid: ip, user)", cfg.By)
	}
	return nil
}

// parseDuration parses a duration string like "30s", "5m", "1h".
func parseDuration(s string) (time.Duration, error) {
	return time.ParseDuration(s)
}

// parseByteSize parses a byte size string like "10mb", "1gb", "512kb".
func parseByteSize(s string) (int64, error) {
	s = strings.TrimSpace(strings.ToLower(s))
	multipliers := map[string]int64{
		"b":  1,
		"kb": 1024,
		"mb": 1024 * 1024,
		"gb": 1024 * 1024 * 1024,
	}

	for suffix, mult := range multipliers {
		if strings.HasSuffix(s, suffix) {
			numStr := strings.TrimSuffix(s, suffix)
			numStr = strings.TrimSpace(numStr)
			n, err := strconv.ParseFloat(numStr, 64)
			if err != nil {
				return 0, fmt.Errorf("invalid number in %q", s)
			}
			return int64(n * float64(mult)), nil
		}
	}

	n, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid byte size: %q", s)
	}
	return n, nil
}

// ParseDurationConfig is exported for use by other server packages.
func ParseDurationConfig(s string) (time.Duration, error) {
	return parseDuration(s)
}

// ParseByteSizeConfig is exported for use by other server packages.
func ParseByteSizeConfig(s string) (int64, error) {
	return parseByteSize(s)
}
