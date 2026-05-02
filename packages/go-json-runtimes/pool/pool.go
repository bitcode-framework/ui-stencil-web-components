package pool

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"
)

type Process interface {
	Execute(ctx context.Context, request []byte) ([]byte, error)
	Kill() error
	Pid() int
	Executions() int
	MemoryMB() int
}

type ProcessFactory interface {
	Spawn(ctx context.Context) (Process, error)
}

type Pool struct {
	config  Config
	factory ProcessFactory
	procs   chan Process
	mu      sync.Mutex
	closed  bool
	stats   Stats
}

type Config struct {
	Size            int
	MaxExecutions   int
	MaxMemoryMB     int
	HardMaxMemoryMB int
	MaxIdleTime     time.Duration
	CrashRecovery   bool
	MaxBackoff      time.Duration
}

type Stats struct {
	TotalExecutions atomic.Int64
	TotalSpawns     atomic.Int64
	TotalRecycles   atomic.Int64
	TotalCrashes    atomic.Int64
	ActiveProcesses atomic.Int32
	IdleProcesses   atomic.Int32
}

func NewPool(config Config, factory ProcessFactory) *Pool {
	if config.Size <= 0 {
		config.Size = 1
	}
	return &Pool{
		config:  config,
		factory: factory,
		procs:   make(chan Process, config.Size),
	}
}

func (p *Pool) Execute(ctx context.Context, request []byte) ([]byte, error) {
	proc, err := p.acquire(ctx)
	if err != nil {
		return nil, err
	}

	p.stats.ActiveProcesses.Add(1)
	defer p.stats.ActiveProcesses.Add(-1)

	result, err := proc.Execute(ctx, request)
	p.stats.TotalExecutions.Add(1)

	if err != nil {
		p.stats.TotalCrashes.Add(1)
		_ = proc.Kill()
		if p.config.CrashRecovery {
			p.spawnReplacement()
		}
		return nil, err
	}

	if p.shouldRecycle(proc) {
		p.stats.TotalRecycles.Add(1)
		_ = proc.Kill()
		p.spawnReplacement()
		return result, nil
	}

	p.release(proc)
	return result, nil
}

func (p *Pool) acquire(ctx context.Context) (Process, error) {
	select {
	case proc := <-p.procs:
		p.stats.IdleProcesses.Add(-1)
		return proc, nil
	default:
	}

	p.mu.Lock()
	currentSize := int(p.stats.ActiveProcesses.Load()) + int(p.stats.IdleProcesses.Load())
	canSpawn := currentSize < p.config.Size
	p.mu.Unlock()

	if canSpawn {
		return p.spawn(ctx)
	}

	select {
	case proc := <-p.procs:
		p.stats.IdleProcesses.Add(-1)
		return proc, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

func (p *Pool) release(proc Process) {
	p.mu.Lock()
	if p.closed {
		p.mu.Unlock()
		_ = proc.Kill()
		return
	}
	p.mu.Unlock()

	p.stats.IdleProcesses.Add(1)
	select {
	case p.procs <- proc:
	default:
		p.stats.IdleProcesses.Add(-1)
		_ = proc.Kill()
	}
}

func (p *Pool) spawn(ctx context.Context) (Process, error) {
	proc, err := p.factory.Spawn(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to spawn process: %w", err)
	}
	p.stats.TotalSpawns.Add(1)
	return proc, nil
}

func (p *Pool) spawnReplacement() {
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		proc, err := p.spawn(ctx)
		if err != nil {
			return
		}
		p.release(proc)
	}()
}

func (p *Pool) shouldRecycle(proc Process) bool {
	if p.config.MaxExecutions > 0 && proc.Executions() >= p.config.MaxExecutions {
		return true
	}
	if p.config.HardMaxMemoryMB > 0 && proc.MemoryMB() > p.config.HardMaxMemoryMB {
		return true
	}
	if p.config.MaxMemoryMB > 0 && proc.MemoryMB() > p.config.MaxMemoryMB {
		return true
	}
	return false
}

type StatsSnapshot struct {
	TotalExecutions int64
	TotalSpawns     int64
	TotalRecycles   int64
	TotalCrashes    int64
	ActiveProcesses int32
	IdleProcesses   int32
}

func (p *Pool) GetStats() StatsSnapshot {
	return StatsSnapshot{
		TotalExecutions: p.stats.TotalExecutions.Load(),
		TotalSpawns:     p.stats.TotalSpawns.Load(),
		TotalRecycles:   p.stats.TotalRecycles.Load(),
		TotalCrashes:    p.stats.TotalCrashes.Load(),
		ActiveProcesses: p.stats.ActiveProcesses.Load(),
		IdleProcesses:   p.stats.IdleProcesses.Load(),
	}
}

func (p *Pool) Close() error {
	p.mu.Lock()
	if p.closed {
		p.mu.Unlock()
		return nil
	}
	p.closed = true
	p.mu.Unlock()

	close(p.procs)
	var firstErr error
	for proc := range p.procs {
		if err := proc.Kill(); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	return firstErr
}
