package plugin

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os/exec"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

type PoolConfig struct {
	Size            int
	MaxExecutions   int
	MaxMemoryMB     int
	DefaultTimeout  time.Duration
	MaxTimeout      time.Duration
	ProcessTimeout  time.Duration
}

func DefaultWorkerPoolConfig() PoolConfig {
	return PoolConfig{
		Size:           4,
		MaxExecutions:  1000,
		DefaultTimeout: 30 * time.Second,
		MaxTimeout:     5 * time.Minute,
		ProcessTimeout: 10 * time.Minute,
	}
}

func DefaultBackgroundPoolConfig() PoolConfig {
	return PoolConfig{
		Size:           2,
		MaxExecutions:  100,
		DefaultTimeout: 5 * time.Minute,
	}
}

type PluginProcess struct {
	cmd            *exec.Cmd
	stdin          io.WriteCloser
	scanner        *bufio.Scanner
	mu             sync.Mutex
	executionCount int64
	startedAt      time.Time
	id             int
	recycled       bool
}

func (p *PluginProcess) send(msg any) error {
	data, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("marshal message: %w", err)
	}
	data = append(data, '\n')
	_, err = p.stdin.Write(data)
	return err
}

func (p *PluginProcess) receive() (*Message, error) {
	if !p.scanner.Scan() {
		if err := p.scanner.Err(); err != nil {
			return nil, fmt.Errorf("read from process: %w", err)
		}
		return nil, fmt.Errorf("process closed stdout")
	}
	var msg Message
	if err := json.Unmarshal(p.scanner.Bytes(), &msg); err != nil {
		return nil, fmt.Errorf("unmarshal message: %w", err)
	}
	return &msg, nil
}

func (p *PluginProcess) shouldRecycle(config PoolConfig) bool {
	if config.MaxExecutions > 0 && int(atomic.LoadInt64(&p.executionCount)) >= config.MaxExecutions {
		return true
	}
	return false
}

type ProcessPool struct {
	command   string
	args      []string
	config    PoolConfig
	available chan *PluginProcess
	mu        sync.Mutex
	processes []*PluginProcess
	nextProcID int
	stopped   bool
}

func NewProcessPool(command string, args []string, config PoolConfig) *ProcessPool {
	return &ProcessPool{
		command:   command,
		args:      args,
		config:    config,
		available: make(chan *PluginProcess, config.Size),
	}
}

func (pool *ProcessPool) Start() error {
	pool.mu.Lock()
	defer pool.mu.Unlock()

	for i := 0; i < pool.config.Size; i++ {
		proc, err := pool.startProcess()
		if err != nil {
			pool.stopAllLocked()
			return fmt.Errorf("failed to start process %d: %w", i, err)
		}
		pool.processes = append(pool.processes, proc)
		pool.available <- proc
	}
	return nil
}

func (pool *ProcessPool) startProcess() (*PluginProcess, error) {
	cmd := exec.Command(pool.command, pool.args...)

	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, fmt.Errorf("stdin pipe: %w", err)
	}

	stdoutPipe, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("stdout pipe: %w", err)
	}

	cmd.Stderr = &logWriter{prefix: fmt.Sprintf("[plugin:node:%d] ", pool.nextProcID+1)}

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("start process: %w", err)
	}

	pool.nextProcID++
	proc := &PluginProcess{
		cmd:       cmd,
		stdin:     stdin,
		scanner:   bufio.NewScanner(stdoutPipe),
		startedAt: time.Now(),
		id:        pool.nextProcID,
	}

	proc.scanner.Buffer(make([]byte, 0, 1024*1024), 10*1024*1024)

	go pool.monitorProcess(proc)

	return proc, nil
}

func (pool *ProcessPool) monitorProcess(proc *PluginProcess) {
	err := proc.cmd.Wait()

	pool.mu.Lock()
	if pool.stopped || proc.recycled {
		pool.mu.Unlock()
		return
	}
	pool.mu.Unlock()

	if err != nil {
		log.Printf("[WARN] plugin process %d crashed: %v", proc.id, err)
	}

	pool.removeProcess(proc)

	backoff := time.Second
	for attempts := 0; attempts < 5; attempts++ {
		time.Sleep(backoff)

		pool.mu.Lock()
		if pool.stopped {
			pool.mu.Unlock()
			return
		}
		pool.mu.Unlock()

		newProc, err := pool.startProcess()
		if err == nil {
			pool.mu.Lock()
			pool.processes = append(pool.processes, newProc)
			pool.mu.Unlock()
			pool.available <- newProc
			log.Printf("[INFO] plugin process restarted (id=%d)", newProc.id)
			return
		}
		log.Printf("[WARN] restart attempt %d failed: %v", attempts+1, err)
		backoff *= 2
	}
	log.Printf("[ERROR] failed to restart plugin process after 5 attempts")
}

func (pool *ProcessPool) removeProcess(proc *PluginProcess) {
	pool.mu.Lock()
	defer pool.mu.Unlock()
	for i, p := range pool.processes {
		if p == proc {
			pool.processes = append(pool.processes[:i], pool.processes[i+1:]...)
			break
		}
	}
}

func (pool *ProcessPool) Acquire() *PluginProcess {
	return <-pool.available
}

func (pool *ProcessPool) Release(proc *PluginProcess) {
	atomic.AddInt64(&proc.executionCount, 1)

	if proc.shouldRecycle(pool.config) {
		log.Printf("[INFO] recycling plugin process %d after %d executions", proc.id, proc.executionCount)
		go pool.recycleProcess(proc)
		return
	}

	pool.available <- proc
}

func (pool *ProcessPool) recycleProcess(proc *PluginProcess) {
	proc.recycled = true
	proc.stdin.Close()

	done := make(chan struct{}, 1)
	go func() {
		proc.cmd.Wait()
		done <- struct{}{}
	}()

	select {
	case <-done:
	case <-time.After(5 * time.Second):
		if proc.cmd.Process != nil {
			proc.cmd.Process.Kill()
		}
	}

	pool.removeProcess(proc)

	pool.mu.Lock()
	if pool.stopped {
		pool.mu.Unlock()
		return
	}
	pool.mu.Unlock()

	newProc, err := pool.startProcess()
	if err != nil {
		log.Printf("[ERROR] failed to start replacement process: %v", err)
		return
	}

	pool.mu.Lock()
	pool.processes = append(pool.processes, newProc)
	pool.mu.Unlock()
	pool.available <- newProc
}

func (pool *ProcessPool) Stop() {
	pool.mu.Lock()
	pool.stopped = true
	procs := make([]*PluginProcess, len(pool.processes))
	copy(procs, pool.processes)
	pool.mu.Unlock()

	for _, proc := range procs {
		proc.stdin.Close()
	}

	deadline := time.After(5 * time.Second)
	for _, proc := range procs {
		done := make(chan error, 1)
		go func(p *PluginProcess) { done <- p.cmd.Wait() }(proc)
		select {
		case <-done:
		case <-deadline:
			if proc.cmd.Process != nil {
				proc.cmd.Process.Kill()
			}
		}
	}
}

func (pool *ProcessPool) stopAllLocked() {
	pool.stopped = true
	for _, proc := range pool.processes {
		proc.stdin.Close()
		if proc.cmd.Process != nil {
			proc.cmd.Process.Kill()
		}
	}
	pool.processes = nil
}

type logWriter struct {
	prefix string
}

func (w *logWriter) Write(p []byte) (n int, err error) {
	log.Printf("%s%s", w.prefix, strings.TrimRight(string(p), "\n\r"))
	return len(p), nil
}
