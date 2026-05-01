package stdlib

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/expr-lang/expr"
)

// EnvResolver resolves environment variable values.
// Default: os.Getenv. BitCode overrides with viper.GetString.
type EnvResolver func(key string) string

// EnvAccessConfig controls which env vars are accessible.
type EnvAccessConfig struct {
	Allow []string
	Deny  []string
}

// EnvHandle is a mutable handle to the env() function's configuration.
// Runtime uses this to override resolver/access after registry construction.
type EnvHandle struct {
	mu       sync.RWMutex
	resolver EnvResolver
	access   *EnvAccessConfig
}

// SetResolver overrides the env resolver.
func (h *EnvHandle) SetResolver(resolver EnvResolver) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.resolver = resolver
}

// SetAccess overrides the env access config.
func (h *EnvHandle) SetAccess(access *EnvAccessConfig) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.access = access
}

func (h *EnvHandle) resolve(key string) (string, error) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	if h.access != nil {
		if err := checkEnvAccess(key, h.access); err != nil {
			return "", err
		}
	}
	return h.resolver(key), nil
}

// RegisterEnvFunc registers the env() function with optional resolver and access control.
// Returns an EnvHandle that can be used to override resolver/access after registration.
func RegisterEnvFunc(r *Registry, resolver EnvResolver, access *EnvAccessConfig) *EnvHandle {
	if resolver == nil {
		resolver = os.Getenv
	}

	handle := &EnvHandle{
		resolver: resolver,
		access:   access,
	}

	r.Register(expr.Function("env", func(params ...any) (any, error) {
		if len(params) < 1 {
			return nil, fmt.Errorf("env: requires at least a key argument")
		}
		key, ok := params[0].(string)
		if !ok {
			return nil, fmt.Errorf("env: key must be a string")
		}

		val, err := handle.resolve(key)
		if err != nil {
			return nil, err
		}

		if val == "" && len(params) > 1 {
			if def, ok := params[1].(string); ok {
				return def, nil
			}
			return fmt.Sprintf("%v", params[1]), nil
		}
		return val, nil
	}))

	return handle
}

func checkEnvAccess(key string, config *EnvAccessConfig) error {
	upper := strings.ToUpper(key)

	for _, pattern := range config.Deny {
		if matched, _ := filepath.Match(strings.ToUpper(pattern), upper); matched {
			return fmt.Errorf("env: access denied for key '%s'", key)
		}
	}

	if len(config.Allow) > 0 {
		for _, pattern := range config.Allow {
			if matched, _ := filepath.Match(strings.ToUpper(pattern), upper); matched {
				return nil
			}
		}
		return fmt.Errorf("env: access denied for key '%s'", key)
	}

	return nil
}
