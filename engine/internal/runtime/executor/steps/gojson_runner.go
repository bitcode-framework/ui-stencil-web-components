package steps

import (
	"context"
	"fmt"
	"path/filepath"
	"time"

	"github.com/bitcode-framework/go-json/runtime"
	"github.com/bitcode-framework/go-json/stdlib"

	"github.com/bitcode-framework/bitcode/internal/runtime/bridge"
	"github.com/bitcode-framework/bitcode/internal/runtime/plugin"
	wasmRT "github.com/bitcode-framework/go-json-runtimes/wasm"
)

// GoJSONRunner executes go-json programs as bitcode process steps.
type GoJSONRunner struct {
	BridgeCtx     *bridge.Context
	ScriptDir     string
	RuntimeConfig *plugin.RuntimeConfig
}

func (r *GoJSONRunner) CanHandle(rt string) bool {
	return rt == "go-json"
}

func (r *GoJSONRunner) Run(ctx context.Context, script string, params map[string]any) (any, error) {
	scriptPath := script
	if !filepath.IsAbs(scriptPath) && r.ScriptDir != "" {
		scriptPath = filepath.Join(r.ScriptDir, script)
	}

	reg := stdlib.DefaultRegistry()
	opts := []runtime.Option{
		runtime.WithStdlib(reg.All()),
		runtime.WithStdlibEnv(reg.EnvVars()),
		runtime.WithRuntimeContext(ctx),
		runtime.WithoutIO(),
	}

	if r.BridgeCtx != nil {
		ext := bridge.BuildGoJSONExtension(r.BridgeCtx)
		opts = append(opts, runtime.WithExtension("bitcode", ext))

		s := r.BridgeCtx.Session()
		opts = append(opts, runtime.WithSession(&runtime.Session{
			UserID:   s.UserID,
			Locale:   s.Locale,
			TenantID: s.TenantID,
			Groups:   s.Groups,
		}))

		scriptBridge := bridge.BuildScriptBridge(r.BridgeCtx)
		if scriptBridge != nil {
			opts = append(opts, runtime.WithScriptBridge(scriptBridge))
		}
	}

	opts = append(opts, r.buildRuntimeOpts()...)

	rt := runtime.NewRuntime(opts...)
	defer rt.Close()

	compiled, err := rt.CompileFile(scriptPath)
	if err != nil {
		return nil, fmt.Errorf("go-json: compile error in %s: %w", script, err)
	}

	input := make(map[string]any)
	if params != nil {
		for k, v := range params {
			input[k] = v
		}
	}

	result, err := rt.Execute(compiled, input)
	if err != nil {
		return nil, fmt.Errorf("go-json: execution error in %s: %w", script, err)
	}

	return result.Value, nil
}

func (r *GoJSONRunner) buildRuntimeOpts() []runtime.Option {
	if r.RuntimeConfig == nil {
		return nil
	}

	var opts []runtime.Option

	if r.RuntimeConfig.WasmEnabled {
		cfg := wasmRT.DefaultConfig()
		if r.RuntimeConfig.WasmMaxMemoryMB > 0 {
			cfg.MaxMemoryMB = r.RuntimeConfig.WasmMaxMemoryMB
		}
		if r.RuntimeConfig.WasmMaxExecTime != "" {
			if d, err := time.ParseDuration(r.RuntimeConfig.WasmMaxExecTime); err == nil {
				cfg.MaxExecTime = d
			}
		}
		cfg.CompileCache = r.RuntimeConfig.WasmCompileCache
		opts = append(opts, runtime.WithScriptRuntime(wasmRT.New(cfg)))
	}

	return opts
}
