package stdlib

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

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

// RegisterEnvFunc registers the env() function with optional resolver and access control.
func RegisterEnvFunc(r *Registry, resolver EnvResolver, access *EnvAccessConfig) {
	if resolver == nil {
		resolver = os.Getenv
	}

	r.Register(expr.Function("env", func(params ...any) (any, error) {
		if len(params) < 1 {
			return nil, fmt.Errorf("env: requires at least a key argument")
		}
		key, ok := params[0].(string)
		if !ok {
			return nil, fmt.Errorf("env: key must be a string")
		}

		if access != nil {
			if err := checkEnvAccess(key, access); err != nil {
				return nil, err
			}
		}

		val := resolver(key)
		if val == "" && len(params) > 1 {
			if def, ok := params[1].(string); ok {
				return def, nil
			}
			return fmt.Sprintf("%v", params[1]), nil
		}
		return val, nil
	}))
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
