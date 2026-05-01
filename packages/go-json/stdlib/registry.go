package stdlib

import (
	"github.com/expr-lang/expr"
)

type Registry struct {
	functions []expr.Option
	envVars   map[string]any
}

func NewRegistry() *Registry {
	return &Registry{
		envVars: make(map[string]any),
	}
}

func (r *Registry) Register(opt expr.Option) {
	r.functions = append(r.functions, opt)
}

// RegisterEnv registers a named variable for injection into the expression environment.
func (r *Registry) RegisterEnv(name string, value any) {
	r.envVars[name] = value
}

// All returns all registered stdlib functions as expr.Option slice.
func (r *Registry) All() []expr.Option {
	return r.functions
}

// EnvVars returns all registered environment variables for scope injection.
func (r *Registry) EnvVars() map[string]any {
	return r.envVars
}

func DefaultRegistry() *Registry {
	r := NewRegistry()
	RegisterMath(r)
	RegisterMathExt(r)
	RegisterStrings(r)
	RegisterStringsExt(r)
	RegisterArrays(r)
	RegisterArraysExt(r)
	RegisterTypes(r)
	RegisterMaps(r)
	RegisterMapsExt(r)
	RegisterDateTime(r)
	RegisterDateTimeExt(r)
	RegisterEncoding(r)
	RegisterFormat(r)
	RegisterFormatExt(r)
	RegisterRegex(r)
	RegisterPath(r)
	RegisterJSON(r)
	RegisterEnvFunc(r, nil, nil)
	r.RegisterEnv("crypto", CryptoNamespace())
	r.RegisterEnv("validate", ValidateNamespace())
	return r
}

// DefaultRegistryWithEnv creates a registry with custom env() configuration.
func DefaultRegistryWithEnv(resolver EnvResolver, access *EnvAccessConfig) *Registry {
	r := NewRegistry()
	RegisterMath(r)
	RegisterMathExt(r)
	RegisterStrings(r)
	RegisterStringsExt(r)
	RegisterArrays(r)
	RegisterArraysExt(r)
	RegisterTypes(r)
	RegisterMaps(r)
	RegisterMapsExt(r)
	RegisterDateTime(r)
	RegisterDateTimeExt(r)
	RegisterEncoding(r)
	RegisterFormat(r)
	RegisterFormatExt(r)
	RegisterRegex(r)
	RegisterPath(r)
	RegisterJSON(r)
	RegisterEnvFunc(r, resolver, access)
	r.RegisterEnv("crypto", CryptoNamespace())
	r.RegisterEnv("validate", ValidateNamespace())
	return r
}
