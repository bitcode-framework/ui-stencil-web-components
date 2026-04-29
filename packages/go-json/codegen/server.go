package codegen

import (
	"fmt"
	"sync"

	"github.com/bitcode-framework/go-json/lang"
)

// ServerCodegenAdapter generates server code for a specific language×framework combination.
type ServerCodegenAdapter interface {
	GenerateServer(program *lang.CompiledProgram) (map[string]string, error)
	Framework() string
	Language() string
}

var (
	serverMu              sync.RWMutex
	serverCodegenRegistry = map[string]map[string]ServerCodegenAdapter{}
)

var defaultFrameworks = map[string]string{
	"go":         "fiber",
	"javascript": "express",
	"python":     "fastapi",
}

// RegisterServerCodegen registers a server codegen adapter for a language×framework pair.
func RegisterServerCodegen(language, framework string, adapter ServerCodegenAdapter) {
	serverMu.Lock()
	defer serverMu.Unlock()
	if serverCodegenRegistry[language] == nil {
		serverCodegenRegistry[language] = map[string]ServerCodegenAdapter{}
	}
	serverCodegenRegistry[language][framework] = adapter
}

// GetServerCodegen returns the adapter for a language×framework pair.
func GetServerCodegen(language, framework string) (ServerCodegenAdapter, error) {
	serverMu.RLock()
	defer serverMu.RUnlock()
	frameworks, ok := serverCodegenRegistry[language]
	if !ok {
		return nil, fmt.Errorf("no server codegen for language %q", language)
	}
	adapter, ok := frameworks[framework]
	if !ok {
		return nil, fmt.Errorf("no server codegen for %s+%s", language, framework)
	}
	return adapter, nil
}

// DefaultFramework returns the default framework for a language.
func DefaultFramework(language string) string {
	if fw, ok := defaultFrameworks[language]; ok {
		return fw
	}
	return ""
}

// ResolveFramework determines the framework to use based on explicit flag, server config, and defaults.
func ResolveFramework(language, explicit, serverConfigFramework string) string {
	if explicit != "" {
		return explicit
	}
	if serverConfigFramework != "" {
		return mapRuntimeToCodegenFramework(language, serverConfigFramework)
	}
	return DefaultFramework(language)
}

func mapRuntimeToCodegenFramework(language, runtimeFramework string) string {
	mapping := map[string]map[string]string{
		"go": {
			"fiber":   "fiber",
			"nethttp": "nethttp",
			"echo":    "echo",
			"gin":     "gin",
			"chi":     "chi",
		},
		"javascript": {
			"fiber": "express",
		},
		"python": {
			"fiber": "fastapi",
		},
	}
	if langMap, ok := mapping[language]; ok {
		if fw, ok := langMap[runtimeFramework]; ok {
			return fw
		}
	}
	return DefaultFramework(language)
}
