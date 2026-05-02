package wasm

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/api"
	"github.com/tetratelabs/wazero/imports/wasi_snapshot_preview1"
)

// Runtime implements the ScriptRuntime interface for WebAssembly modules via wazero.
type Runtime struct {
	engine wazero.Runtime
	cache  map[string]wazero.CompiledModule
	pools  map[string]*InstancePool
	config Config
	mu     sync.RWMutex
}

// New creates a new WASM runtime with the given config.
func New(config Config) *Runtime {
	ctx := context.Background()

	runtimeConfig := wazero.NewRuntimeConfig()
	if config.MaxMemoryMB > 0 {
		pages := uint32(config.MaxMemoryMB * 16) // 1 page = 64KB
		runtimeConfig = runtimeConfig.WithMemoryLimitPages(pages)
	}

	engine := wazero.NewRuntimeWithConfig(ctx, runtimeConfig)

	return &Runtime{
		engine: engine,
		cache:  make(map[string]wazero.CompiledModule),
		pools:  make(map[string]*InstancePool),
		config: config,
	}
}

func (w *Runtime) Name() string              { return "wasm" }
func (w *Runtime) Extensions() []string       { return []string{".wasm"} }
func (w *Runtime) CanHandle(ext string) bool  { return ext == ".wasm" }
func (w *Runtime) Validate() error            { return nil }

func (w *Runtime) Execute(ctx context.Context, script, function string, params, bridge map[string]any) (any, error) {
	compiled, err := w.getOrCompile(ctx, script)
	if err != nil {
		return nil, err
	}

	hostCloser, err := w.defineHostModule(ctx, bridge)
	if err != nil {
		return nil, fmt.Errorf("wasm host module: %w", err)
	}
	defer hostCloser.Close(ctx)

	if w.config.WASIEnabled {
		wasi_snapshot_preview1.MustInstantiate(ctx, w.engine)
	}

	instance, pooled, err := w.acquireInstance(ctx, script, compiled)
	if err != nil {
		return nil, fmt.Errorf("wasm instantiate: %w", err)
	}
	if pooled {
		defer w.releaseInstance(ctx, script, instance)
	} else {
		defer instance.Close(ctx)
	}

	argsJSON, err := json.Marshal(params)
	if err != nil {
		return nil, fmt.Errorf("wasm args marshal: %w", err)
	}

	malloc := instance.ExportedFunction("malloc")
	if malloc == nil {
		return nil, fmt.Errorf("wasm module missing required 'malloc' export")
	}

	allocResult, err := malloc.Call(ctx, uint64(len(argsJSON)))
	if err != nil {
		return nil, fmt.Errorf("wasm malloc: %w", err)
	}
	argPtr := uint32(allocResult[0])

	if !instance.Memory().Write(argPtr, argsJSON) {
		return nil, fmt.Errorf("wasm memory write failed (ptr=%d, len=%d)", argPtr, len(argsJSON))
	}

	fn := instance.ExportedFunction(function)
	if fn == nil {
		return nil, fmt.Errorf("wasm function '%s' not exported", function)
	}

	timeoutCtx := ctx
	var cancel context.CancelFunc
	if w.config.MaxExecTime > 0 {
		if _, hasDeadline := ctx.Deadline(); !hasDeadline {
			timeoutCtx, cancel = context.WithTimeout(ctx, w.config.MaxExecTime)
			defer cancel()
		}
	}

	results, err := fn.Call(timeoutCtx, uint64(argPtr), uint64(len(argsJSON)))
	if err != nil {
		return nil, fmt.Errorf("wasm call '%s': %w", function, err)
	}

	if len(results) == 0 {
		return nil, nil
	}

	resultPtr, resultLen := unpackPtrLen(results[0])
	if resultLen == 0 {
		return nil, nil
	}

	resultJSON, ok := instance.Memory().Read(resultPtr, resultLen)
	if !ok {
		return nil, fmt.Errorf("wasm result read failed (ptr=%d, len=%d)", resultPtr, resultLen)
	}

	if free := instance.ExportedFunction("free"); free != nil {
		free.Call(ctx, uint64(resultPtr), uint64(resultLen))
	}

	var result any
	if err := json.Unmarshal(resultJSON, &result); err != nil {
		return nil, fmt.Errorf("wasm result unmarshal: %w", err)
	}

	return result, nil
}

func (w *Runtime) Close() error {
	ctx := context.Background()
	w.mu.Lock()
	for _, pool := range w.pools {
		pool.Close(ctx)
	}
	w.pools = make(map[string]*InstancePool)
	w.mu.Unlock()
	return w.engine.Close(ctx)
}

func (w *Runtime) acquireInstance(ctx context.Context, path string, compiled wazero.CompiledModule) (api.Module, bool, error) {
	if w.config.PoolSize <= 0 {
		instance, err := w.instantiateModule(ctx, compiled)
		return instance, false, err
	}

	absPath, err := filepath.Abs(path)
	if err != nil {
		absPath = path
	}

	w.mu.RLock()
	pool, ok := w.pools[absPath]
	w.mu.RUnlock()

	if !ok {
		pool, err = w.getOrCreatePool(ctx, absPath, compiled)
		if err != nil {
			instance, err2 := w.instantiateModule(ctx, compiled)
			return instance, false, err2
		}
	}

	instance, err := pool.Acquire(ctx)
	if err != nil {
		instance, err2 := w.instantiateModule(ctx, compiled)
		return instance, false, err2
	}
	return instance, true, nil
}

func (w *Runtime) releaseInstance(ctx context.Context, path string, instance api.Module) {
	absPath, _ := filepath.Abs(path)

	w.mu.RLock()
	pool, ok := w.pools[absPath]
	w.mu.RUnlock()

	if ok {
		pool.Release(ctx, instance)
	} else {
		instance.Close(ctx)
	}
}

func (w *Runtime) getOrCreatePool(ctx context.Context, absPath string, compiled wazero.CompiledModule) (*InstancePool, error) {
	w.mu.Lock()
	defer w.mu.Unlock()

	if pool, ok := w.pools[absPath]; ok {
		return pool, nil
	}

	pool, err := NewInstancePool(ctx, w.engine, compiled, PoolConfig{Size: w.config.PoolSize})
	if err != nil {
		return nil, err
	}
	w.pools[absPath] = pool
	return pool, nil
}

func (w *Runtime) getOrCompile(ctx context.Context, path string) (wazero.CompiledModule, error) {
	absPath, err := filepath.Abs(path)
	if err != nil {
		absPath = path
	}

	if w.config.CompileCache {
		w.mu.RLock()
		if compiled, ok := w.cache[absPath]; ok {
			w.mu.RUnlock()
			return compiled, nil
		}
		w.mu.RUnlock()
	}

	wasmBytes, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read WASM file '%s': %w", filepath.Base(path), err)
	}

	compiled, err := w.engine.CompileModule(ctx, wasmBytes)
	if err != nil {
		return nil, fmt.Errorf("failed to compile WASM module '%s': %w", filepath.Base(path), err)
	}

	if w.config.CompileCache {
		w.mu.Lock()
		w.cache[absPath] = compiled
		w.mu.Unlock()
	}

	return compiled, nil
}
