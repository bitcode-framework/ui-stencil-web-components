package wasm

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"

	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/api"
)

// PoolConfig configures WASM instance pooling for high-throughput scenarios.
type PoolConfig struct {
	Size int // number of pre-instantiated modules (0 = no pooling)
}

// InstancePool manages a pool of pre-instantiated WASM modules for reuse.
// Each instance has its own linear memory — instances are NOT shared across goroutines.
// After use, instances are returned to the pool for the next caller.
type InstancePool struct {
	compiled wazero.CompiledModule
	engine   wazero.Runtime
	pool     chan api.Module
	config   PoolConfig
	active   atomic.Int64
	mu       sync.Mutex
	closed   bool
}

// NewInstancePool creates a pool of pre-instantiated WASM modules.
func NewInstancePool(ctx context.Context, engine wazero.Runtime, compiled wazero.CompiledModule, config PoolConfig) (*InstancePool, error) {
	if config.Size <= 0 {
		return nil, fmt.Errorf("pool size must be > 0")
	}

	p := &InstancePool{
		compiled: compiled,
		engine:   engine,
		pool:     make(chan api.Module, config.Size),
		config:   config,
	}

	for i := 0; i < config.Size; i++ {
		instance, err := engine.InstantiateModule(ctx, compiled, wazero.NewModuleConfig().WithName(""))
		if err != nil {
			p.Close(ctx)
			return nil, fmt.Errorf("failed to pre-instantiate pool instance %d: %w", i, err)
		}
		p.pool <- instance
	}

	return p, nil
}

// Acquire gets an instance from the pool. Blocks if all instances are in use.
// Returns an error if the pool is closed or the context is cancelled.
func (p *InstancePool) Acquire(ctx context.Context) (api.Module, error) {
	select {
	case instance := <-p.pool:
		if instance == nil {
			return nil, fmt.Errorf("pool closed")
		}
		p.active.Add(1)
		return instance, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

// Release returns an instance to the pool for reuse.
// If the pool is closed, the instance is closed instead.
func (p *InstancePool) Release(ctx context.Context, instance api.Module) {
	p.active.Add(-1)

	p.mu.Lock()
	closed := p.closed
	p.mu.Unlock()

	if closed {
		instance.Close(ctx)
		return
	}

	select {
	case p.pool <- instance:
	default:
		instance.Close(ctx)
	}
}

// Close drains the pool and closes all instances.
func (p *InstancePool) Close(ctx context.Context) error {
	p.mu.Lock()
	if p.closed {
		p.mu.Unlock()
		return nil
	}
	p.closed = true
	p.mu.Unlock()

	close(p.pool)
	var firstErr error
	for instance := range p.pool {
		if instance != nil {
			if err := instance.Close(ctx); err != nil && firstErr == nil {
				firstErr = err
			}
		}
	}
	return firstErr
}

// Size returns the configured pool size.
func (p *InstancePool) Size() int {
	return p.config.Size
}

// Available returns the number of instances currently available in the pool.
func (p *InstancePool) Available() int {
	return len(p.pool)
}

// Active returns the number of instances currently in use.
func (p *InstancePool) Active() int64 {
	return p.active.Load()
}
