//go:build linux || darwin

package native

import (
	"fmt"
	"path/filepath"
	"plugin"
)

func (n *Runtime) loadPlugin(absPath string) (*loadedPlugin, error) {
	p, err := plugin.Open(absPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open plugin '%s': %w", filepath.Base(absPath), err)
	}

	manifestSym, err := p.Lookup("Manifest")
	if err != nil {
		return nil, fmt.Errorf("plugin '%s' missing Manifest() export", filepath.Base(absPath))
	}
	manifestFn, ok := manifestSym.(func() []string)
	if !ok {
		return nil, fmt.Errorf("plugin '%s' Manifest has wrong signature (expected func() []string)", filepath.Base(absPath))
	}
	manifest := manifestFn()

	functions := make(map[string]func(map[string]any) (any, error))
	for _, name := range manifest {
		sym, err := p.Lookup(name)
		if err != nil {
			return nil, fmt.Errorf("plugin '%s' declares '%s' in manifest but symbol not found", filepath.Base(absPath), name)
		}
		fn, ok := sym.(func(map[string]any) (any, error))
		if !ok {
			return nil, fmt.Errorf("plugin '%s' function '%s' has wrong signature (expected func(map[string]any) (any, error))", filepath.Base(absPath), name)
		}
		functions[name] = fn
	}

	return &loadedPlugin{functions: functions, manifest: manifest}, nil
}
