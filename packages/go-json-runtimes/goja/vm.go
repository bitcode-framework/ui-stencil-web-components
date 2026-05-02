package goja

import (
	"fmt"
	"os"
	"sync"

	runtimes "github.com/bitcode-framework/go-json-runtimes"
	"github.com/dop251/goja"
)

var compiledScripts sync.Map

type GojaVM struct {
	rt   *goja.Runtime
	opts runtimes.VMOptions
}

func (v *GojaVM) InjectBridge(bridge map[string]any) error {
	v.rt.Set("bitcode", bridge)

	logFn := func(level string) func(args ...any) {
		return func(args ...any) {
			if logFnRaw, ok := bridge["log"]; ok {
				if logFunc, ok := logFnRaw.(func(...any) (any, error)); ok {
					logFunc(level, fmt.Sprint(args...))
				}
			}
		}
	}
	v.rt.Set("console", map[string]any{
		"log":   logFn("info"),
		"warn":  logFn("warn"),
		"error": logFn("error"),
		"debug": logFn("debug"),
	})
	return nil
}

func (v *GojaVM) InjectParams(params map[string]any) error {
	v.rt.Set("params", params)
	return nil
}

func (v *GojaVM) Execute(code string, filename string) (any, error) {
	program, err := v.getCompiled(filename, code)
	if err != nil {
		return nil, fmt.Errorf("syntax error: %w", err)
	}

	val, err := v.rt.RunProgram(program)
	if err != nil {
		return v.handleError(err)
	}

	return v.resolveResult(val)
}

func (v *GojaVM) Interrupt(reason string) {
	v.rt.Interrupt(reason)
}

func (v *GojaVM) Close() {
	v.rt.ClearInterrupt()
}

func (v *GojaVM) getCompiled(filename, code string) (*goja.Program, error) {
	info, statErr := os.Stat(filename)
	if statErr != nil {
		return goja.Compile(filename, code, true)
	}

	cacheKey := filename + ":" + info.ModTime().String()
	if cached, ok := compiledScripts.Load(cacheKey); ok {
		return cached.(*goja.Program), nil
	}

	program, err := goja.Compile(filename, code, true)
	if err != nil {
		return nil, err
	}
	compiledScripts.Store(cacheKey, program)
	return program, nil
}

func (v *GojaVM) handleError(err error) (any, error) {
	if interrupted, ok := err.(*goja.InterruptedError); ok {
		return nil, fmt.Errorf("step timeout: %v", interrupted.Value())
	}
	if exception, ok := err.(*goja.Exception); ok {
		return nil, fmt.Errorf("script error: %s", exception.String())
	}
	return nil, fmt.Errorf("script error: %w", err)
}

func (v *GojaVM) resolveResult(val goja.Value) (any, error) {
	if val == nil || goja.IsUndefined(val) || goja.IsNull(val) {
		return nil, nil
	}

	exported := val.Export()

	if m, ok := exported.(map[string]any); ok {
		if execFn, exists := m["execute"]; exists {
			if callable, ok := goja.AssertFunction(v.rt.ToValue(execFn)); ok {
				return v.callExecuteFunc(callable)
			}
		}
	}

	if callable, ok := goja.AssertFunction(val); ok {
		return v.callExecuteFunc(callable)
	}

	return exported, nil
}

func (v *GojaVM) callExecuteFunc(fn goja.Callable) (any, error) {
	bitcodeVal := v.rt.Get("bitcode")
	paramsVal := v.rt.Get("params")

	result, err := fn(goja.Undefined(), bitcodeVal, paramsVal)
	if err != nil {
		return v.handleError(err)
	}

	if result == nil || goja.IsUndefined(result) || goja.IsNull(result) {
		return nil, nil
	}
	return result.Export(), nil
}
