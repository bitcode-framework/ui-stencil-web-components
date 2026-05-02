package wasm

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/api"
)

func (w *Runtime) defineHostModule(ctx context.Context, bridge map[string]any) (api.Closer, error) {
	builder := w.engine.NewHostModuleBuilder("env")

	builder.NewFunctionBuilder().
		WithFunc(func(ctx context.Context, m api.Module, fnNamePtr, fnNameLen, argsPtr, argsLen uint32) uint64 {
			fnName := readStringFromMemory(m, fnNamePtr, fnNameLen)
			argsJSON := readBytesFromMemory(m, argsPtr, argsLen)

			result, err := invokeBridgeFunction(bridge, fnName, argsJSON)
			return writeResultToMemory(ctx, m, result, err)
		}).
		WithParameterNames("fn_ptr", "fn_len", "args_ptr", "args_len").
		Export("bridge_call")

	builder.NewFunctionBuilder().
		WithFunc(func(ctx context.Context, m api.Module, level uint32, msgPtr, msgLen uint32) {
			// log host function — currently a no-op, can be wired to a logger
			_ = readStringFromMemory(m, msgPtr, msgLen)
		}).
		WithParameterNames("level", "msg_ptr", "msg_len").
		Export("log")

	return builder.Instantiate(ctx)
}

func (w *Runtime) instantiateModule(ctx context.Context, compiled wazero.CompiledModule) (api.Module, error) {
	moduleConfig := wazero.NewModuleConfig().
		WithName("").
		WithStartFunctions()

	return w.engine.InstantiateModule(ctx, compiled, moduleConfig)
}

func writeResultToMemory(ctx context.Context, m api.Module, result any, err error) uint64 {
	var response []byte
	if err != nil {
		response, _ = json.Marshal(map[string]any{"error": err.Error()})
	} else {
		response, _ = json.Marshal(map[string]any{"value": result})
	}

	malloc := m.ExportedFunction("malloc")
	if malloc == nil {
		return 0
	}
	results, callErr := malloc.Call(ctx, uint64(len(response)))
	if callErr != nil || len(results) == 0 {
		return 0
	}
	ptr := uint32(results[0])

	m.Memory().Write(ptr, response)
	return packPtrLen(ptr, uint32(len(response)))
}

func invokeBridgeFunction(bridge map[string]any, fnPath string, argsJSON []byte) (any, error) {
	if bridge == nil {
		return nil, fmt.Errorf("bridge not available")
	}

	var args []any
	if len(argsJSON) > 0 {
		json.Unmarshal(argsJSON, &args)
	}

	parts := strings.Split(fnPath, ".")
	var current any = bridge
	for i, part := range parts {
		m, ok := current.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("bridge path '%s' not found", fnPath)
		}
		current = m[part]
		if current == nil {
			return nil, fmt.Errorf("bridge function '%s' not found", fnPath)
		}
		if i < len(parts)-1 {
			continue
		}
		switch fn := current.(type) {
		case func(string, ...any) (any, error):
			if len(args) > 0 {
				str, _ := args[0].(string)
				return fn(str, args[1:]...)
			}
			return fn("")
		case func(...any) (any, error):
			return fn(args...)
		case func(map[string]any) (any, error):
			if len(args) > 0 {
				if m, ok := args[0].(map[string]any); ok {
					return fn(m)
				}
			}
			return fn(nil)
		default:
			return nil, fmt.Errorf("bridge '%s' is not callable (type %T)", fnPath, current)
		}
	}
	return nil, fmt.Errorf("bridge function '%s' not found", fnPath)
}
