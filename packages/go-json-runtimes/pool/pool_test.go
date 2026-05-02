package pool

import (
	"context"
	"sync/atomic"
	"testing"
	"time"
)

type mockProcess struct {
	executions atomic.Int32
	memoryMB   int
	killed     bool
}

func (p *mockProcess) Execute(ctx context.Context, request []byte) ([]byte, error) {
	p.executions.Add(1)
	return append([]byte("response:"), request...), nil
}

func (p *mockProcess) Kill() error {
	p.killed = true
	return nil
}

func (p *mockProcess) Pid() int        { return 1234 }
func (p *mockProcess) Executions() int { return int(p.executions.Load()) }
func (p *mockProcess) MemoryMB() int   { return p.memoryMB }

type mockFactory struct {
	spawned atomic.Int32
}

func (f *mockFactory) Spawn(ctx context.Context) (Process, error) {
	f.spawned.Add(1)
	return &mockProcess{}, nil
}

func TestPool_BasicExecution(t *testing.T) {
	factory := &mockFactory{}
	p := NewPool(Config{Size: 2}, factory)
	defer p.Close()

	ctx := context.Background()
	result, err := p.Execute(ctx, []byte("hello"))
	if err != nil {
		t.Fatalf("Execute error: %v", err)
	}
	if string(result) != "response:hello" {
		t.Errorf("expected 'response:hello', got %q", string(result))
	}
}

func TestPool_ReuseProcess(t *testing.T) {
	factory := &mockFactory{}
	p := NewPool(Config{Size: 1}, factory)
	defer p.Close()

	ctx := context.Background()
	for i := 0; i < 5; i++ {
		_, err := p.Execute(ctx, []byte("test"))
		if err != nil {
			t.Fatalf("Execute %d error: %v", i, err)
		}
	}

	stats := p.GetStats()
	if stats.TotalExecutions != 5 {
		t.Errorf("expected 5 executions, got %d", stats.TotalExecutions)
	}
	if stats.TotalSpawns > 2 {
		t.Errorf("expected at most 2 spawns (reuse), got %d", stats.TotalSpawns)
	}
}

func TestPool_MaxExecutionsRecycle(t *testing.T) {
	factory := &mockFactory{}
	p := NewPool(Config{Size: 1, MaxExecutions: 2, CrashRecovery: true}, factory)
	defer p.Close()

	ctx := context.Background()
	for i := 0; i < 4; i++ {
		_, err := p.Execute(ctx, []byte("test"))
		if err != nil {
			t.Fatalf("Execute %d error: %v", i, err)
		}
	}

	time.Sleep(100 * time.Millisecond)
	stats := p.GetStats()
	if stats.TotalRecycles == 0 {
		t.Error("expected at least one recycle")
	}
}

func TestPool_ContextCancellation(t *testing.T) {
	slowFactory := &slowProcessFactory{delay: 500 * time.Millisecond}
	p := NewPool(Config{Size: 1}, slowFactory)
	defer p.Close()

	go func() {
		ctx := context.Background()
		_, _ = p.Execute(ctx, []byte("occupy"))
	}()

	time.Sleep(50 * time.Millisecond)

	ctx2, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	_, err := p.Execute(ctx2, []byte("should-timeout"))
	if err == nil {
		t.Fatal("expected context cancellation error")
	}
}

type slowProcessFactory struct {
	delay time.Duration
}

func (f *slowProcessFactory) Spawn(ctx context.Context) (Process, error) {
	return &slowProcess{delay: f.delay}, nil
}

type slowProcess struct {
	delay      time.Duration
	executions atomic.Int32
	killed     bool
}

func (p *slowProcess) Execute(ctx context.Context, request []byte) ([]byte, error) {
	p.executions.Add(1)
	time.Sleep(p.delay)
	return request, nil
}

func (p *slowProcess) Kill() error    { p.killed = true; return nil }
func (p *slowProcess) Pid() int       { return 9999 }
func (p *slowProcess) Executions() int { return int(p.executions.Load()) }
func (p *slowProcess) MemoryMB() int  { return 0 }

func TestPool_Close(t *testing.T) {
	factory := &mockFactory{}
	p := NewPool(Config{Size: 2}, factory)

	ctx := context.Background()
	_, _ = p.Execute(ctx, []byte("test"))

	err := p.Close()
	if err != nil {
		t.Fatalf("Close error: %v", err)
	}
}

func TestDualPool_WorkerAndBackground(t *testing.T) {
	factory := &mockFactory{}
	dp := NewDualPool(DualPoolConfig{
		Worker:     Config{Size: 2},
		Background: Config{Size: 1},
	}, factory)
	defer dp.Close()

	ctx := context.Background()

	result, err := dp.Execute(ctx, "worker", []byte("w"))
	if err != nil {
		t.Fatalf("worker execute error: %v", err)
	}
	if string(result) != "response:w" {
		t.Errorf("expected 'response:w', got %q", string(result))
	}

	result, err = dp.Execute(ctx, "background", []byte("b"))
	if err != nil {
		t.Fatalf("background execute error: %v", err)
	}
	if string(result) != "response:b" {
		t.Errorf("expected 'response:b', got %q", string(result))
	}
}
