package wasm

import (
	"context"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/tetratelabs/wazero"
)

func setupPoolTest(t *testing.T) (wazero.Runtime, wazero.CompiledModule, string) {
	t.Helper()
	ctx := context.Background()
	engine := wazero.NewRuntime(ctx)

	wasmBytes := validMinimalWasm()
	compiled, err := engine.CompileModule(ctx, wasmBytes)
	if err != nil {
		t.Fatalf("compile error: %v", err)
	}

	tmpDir := t.TempDir()
	wasmFile := filepath.Join(tmpDir, "test.wasm")
	os.WriteFile(wasmFile, wasmBytes, 0644)

	return engine, compiled, tmpDir
}

func TestInstancePool_Create(t *testing.T) {
	ctx := context.Background()
	engine, compiled, _ := setupPoolTest(t)
	defer engine.Close(ctx)

	pool, err := NewInstancePool(ctx, engine, compiled, PoolConfig{Size: 3})
	if err != nil {
		t.Fatalf("pool creation error: %v", err)
	}
	defer pool.Close(ctx)

	if pool.Size() != 3 {
		t.Errorf("expected size 3, got %d", pool.Size())
	}
	if pool.Available() != 3 {
		t.Errorf("expected 3 available, got %d", pool.Available())
	}
	if pool.Active() != 0 {
		t.Errorf("expected 0 active, got %d", pool.Active())
	}
}

func TestInstancePool_InvalidSize(t *testing.T) {
	ctx := context.Background()
	engine, compiled, _ := setupPoolTest(t)
	defer engine.Close(ctx)

	_, err := NewInstancePool(ctx, engine, compiled, PoolConfig{Size: 0})
	if err == nil {
		t.Fatal("expected error for size 0")
	}

	_, err = NewInstancePool(ctx, engine, compiled, PoolConfig{Size: -1})
	if err == nil {
		t.Fatal("expected error for negative size")
	}
}

func TestInstancePool_AcquireRelease(t *testing.T) {
	ctx := context.Background()
	engine, compiled, _ := setupPoolTest(t)
	defer engine.Close(ctx)

	pool, err := NewInstancePool(ctx, engine, compiled, PoolConfig{Size: 2})
	if err != nil {
		t.Fatalf("pool creation error: %v", err)
	}
	defer pool.Close(ctx)

	instance, err := pool.Acquire(ctx)
	if err != nil {
		t.Fatalf("acquire error: %v", err)
	}
	if instance == nil {
		t.Fatal("expected non-nil instance")
	}
	if pool.Available() != 1 {
		t.Errorf("expected 1 available after acquire, got %d", pool.Available())
	}
	if pool.Active() != 1 {
		t.Errorf("expected 1 active after acquire, got %d", pool.Active())
	}

	pool.Release(ctx, instance)
	if pool.Available() != 2 {
		t.Errorf("expected 2 available after release, got %d", pool.Available())
	}
	if pool.Active() != 0 {
		t.Errorf("expected 0 active after release, got %d", pool.Active())
	}
}

func TestInstancePool_AcquireAll(t *testing.T) {
	ctx := context.Background()
	engine, compiled, _ := setupPoolTest(t)
	defer engine.Close(ctx)

	pool, err := NewInstancePool(ctx, engine, compiled, PoolConfig{Size: 2})
	if err != nil {
		t.Fatalf("pool creation error: %v", err)
	}
	defer pool.Close(ctx)

	i1, _ := pool.Acquire(ctx)
	i2, _ := pool.Acquire(ctx)

	if pool.Available() != 0 {
		t.Errorf("expected 0 available, got %d", pool.Available())
	}
	if pool.Active() != 2 {
		t.Errorf("expected 2 active, got %d", pool.Active())
	}

	pool.Release(ctx, i1)
	pool.Release(ctx, i2)
}

func TestInstancePool_AcquireBlocksWhenEmpty(t *testing.T) {
	ctx := context.Background()
	engine, compiled, _ := setupPoolTest(t)
	defer engine.Close(ctx)

	pool, err := NewInstancePool(ctx, engine, compiled, PoolConfig{Size: 1})
	if err != nil {
		t.Fatalf("pool creation error: %v", err)
	}
	defer pool.Close(ctx)

	instance, _ := pool.Acquire(ctx)

	timeoutCtx, cancel := context.WithTimeout(ctx, 50*time.Millisecond)
	defer cancel()

	_, err = pool.Acquire(timeoutCtx)
	if err == nil {
		t.Fatal("expected timeout error when pool is empty")
	}

	pool.Release(ctx, instance)
}

func TestInstancePool_ConcurrentAccess(t *testing.T) {
	ctx := context.Background()
	engine, compiled, _ := setupPoolTest(t)
	defer engine.Close(ctx)

	poolSize := 4
	pool, err := NewInstancePool(ctx, engine, compiled, PoolConfig{Size: poolSize})
	if err != nil {
		t.Fatalf("pool creation error: %v", err)
	}
	defer pool.Close(ctx)

	var wg sync.WaitGroup
	var successCount atomic.Int64
	goroutines := 20
	iterations := 10

	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < iterations; j++ {
				instance, err := pool.Acquire(ctx)
				if err != nil {
					return
				}

				fn := instance.ExportedFunction("malloc")
				if fn != nil {
					fn.Call(ctx, 64)
				}

				pool.Release(ctx, instance)
				successCount.Add(1)
			}
		}()
	}

	wg.Wait()

	expected := int64(goroutines * iterations)
	if successCount.Load() != expected {
		t.Errorf("expected %d successful operations, got %d", expected, successCount.Load())
	}

	if pool.Available() != poolSize {
		t.Errorf("expected all %d instances returned to pool, got %d available", poolSize, pool.Available())
	}
}

func TestInstancePool_Close(t *testing.T) {
	ctx := context.Background()
	engine, compiled, _ := setupPoolTest(t)
	defer engine.Close(ctx)

	pool, err := NewInstancePool(ctx, engine, compiled, PoolConfig{Size: 3})
	if err != nil {
		t.Fatalf("pool creation error: %v", err)
	}

	if err := pool.Close(ctx); err != nil {
		t.Fatalf("close error: %v", err)
	}

	_, err = pool.Acquire(ctx)
	if err == nil {
		t.Fatal("expected error acquiring from closed pool")
	}
}

func TestInstancePool_DoubleClose(t *testing.T) {
	ctx := context.Background()
	engine, compiled, _ := setupPoolTest(t)
	defer engine.Close(ctx)

	pool, err := NewInstancePool(ctx, engine, compiled, PoolConfig{Size: 1})
	if err != nil {
		t.Fatalf("pool creation error: %v", err)
	}

	pool.Close(ctx)
	if err := pool.Close(ctx); err != nil {
		t.Errorf("second close should not error, got: %v", err)
	}
}

func TestInstancePool_ReleaseAfterClose(t *testing.T) {
	ctx := context.Background()
	engine, compiled, _ := setupPoolTest(t)
	defer engine.Close(ctx)

	pool, err := NewInstancePool(ctx, engine, compiled, PoolConfig{Size: 1})
	if err != nil {
		t.Fatalf("pool creation error: %v", err)
	}

	instance, _ := pool.Acquire(ctx)
	pool.Close(ctx)

	pool.Release(ctx, instance)
}
