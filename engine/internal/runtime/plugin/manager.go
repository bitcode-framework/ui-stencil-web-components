package plugin

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os/exec"
	"strings"
	"sync"

	"github.com/bitcode-framework/bitcode/internal/runtime/bridge"
)

type RuntimeConfig struct {
	NodeEnabled    string
	NodeCommand    string
	NodeMinVersion string
	WorkerPool     PoolConfig
	BackgroundPool PoolConfig
}

func DefaultRuntimeConfig() RuntimeConfig {
	return RuntimeConfig{
		NodeEnabled:    "auto",
		NodeCommand:    "",
		NodeMinVersion: "20.0.0",
		WorkerPool:     DefaultWorkerPoolConfig(),
		BackgroundPool: DefaultBackgroundPoolConfig(),
	}
}

type Manager struct {
	workerPool     *ProcessPool
	backgroundPool *ProcessPool
	bridgeHandler  *BridgeHandler
	bridgeFactory  *bridge.Factory
	txStore        *txStore
	runtimeConfig  RuntimeConfig
	mu             sync.RWMutex
	nodeAvailable  bool
}

func NewManager() *Manager {
	return &Manager{
		bridgeHandler: &BridgeHandler{},
		txStore:       newTxStore(),
		runtimeConfig: DefaultRuntimeConfig(),
	}
}

func (m *Manager) SetBridgeFactory(f *bridge.Factory) {
	m.bridgeFactory = f
}

func (m *Manager) SetRuntimeConfig(cfg RuntimeConfig) {
	m.runtimeConfig = cfg
}

func (m *Manager) StartNodePool() error {
	cfg := m.runtimeConfig

	if cfg.NodeEnabled == "false" {
		log.Println("[PLUGIN] Node.js runtime disabled by config")
		return nil
	}

	command, engine, err := detectJSEngine(cfg.NodeCommand)
	if err != nil {
		if cfg.NodeEnabled == "true" {
			return fmt.Errorf("Node.js runtime required but not found: %w", err)
		}
		log.Printf("[PLUGIN] Node.js runtime not available: %v", err)
		return nil
	}

	log.Printf("[PLUGIN] detected %s engine: %s", engine, command)

	args := []string{"plugins/node/runtime.js"}

	workerPool := NewProcessPool(command, args, cfg.WorkerPool)
	if err := workerPool.Start(); err != nil {
		return fmt.Errorf("failed to start Node.js worker pool: %w", err)
	}

	var bgPool *ProcessPool
	if cfg.BackgroundPool.Size > 0 {
		bgPool = NewProcessPool(command, args, cfg.BackgroundPool)
		if err := bgPool.Start(); err != nil {
			log.Printf("[WARN] failed to start background pool: %v — using worker pool as fallback", err)
			bgPool = nil
		}
	}

	m.mu.Lock()
	m.workerPool = workerPool
	m.backgroundPool = bgPool
	m.nodeAvailable = true
	m.mu.Unlock()

	totalSize := cfg.WorkerPool.Size
	if bgPool != nil {
		totalSize += cfg.BackgroundPool.Size
	}
	log.Printf("[PLUGIN] Node.js runtime started (worker: %d, background: %d, engine: %s)",
		cfg.WorkerPool.Size, cfg.BackgroundPool.Size, engine)
	return nil
}

func (m *Manager) IsRunning(name string) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	switch name {
	case "node", "typescript":
		return m.nodeAvailable
	}
	return false
}

func (m *Manager) Run(ctx context.Context, script string, params map[string]any) (any, error) {
	return m.Execute(ctx, script, params, nil)
}

func (m *Manager) Execute(ctx context.Context, script string, params map[string]any, bridgeCtx *bridge.Context) (any, error) {
	runtime := m.detectRuntime(script, "")
	if runtime != "node" && runtime != "typescript" {
		return nil, fmt.Errorf("unsupported runtime %q for external plugin", runtime)
	}

	m.mu.RLock()
	available := m.nodeAvailable
	workerPool := m.workerPool
	bgPool := m.backgroundPool
	m.mu.RUnlock()

	if !available || workerPool == nil {
		return nil, &bridge.BridgeError{
			Code:    "RUNTIME_NOT_AVAILABLE",
			Message: "Node.js runtime is not available. Install Node.js 20+ or set runtime.node.enabled in bitcode.yaml",
		}
	}

	moduleName := ""
	poolName := ""
	if _, ok := params["__runtime"]; ok {
		delete(params, "__runtime")
	}
	if mn, ok := params["__module"]; ok {
		moduleName = fmt.Sprintf("%v", mn)
		delete(params, "__module")
	}
	if pn, ok := params["__pool"]; ok {
		poolName = fmt.Sprintf("%v", pn)
		delete(params, "__pool")
	}

	pool := workerPool
	if poolName == "background" && bgPool != nil {
		pool = bgPool
	}

	var bc *bridge.Context
	if bridgeCtx != nil {
		bc = bridgeCtx
	} else if m.bridgeFactory != nil {
		session := bridge.Session{}
		if uid, ok := params["user_id"]; ok {
			session.UserID = fmt.Sprintf("%v", uid)
		}
		bc = m.bridgeFactory.NewContext(moduleName, session, bridge.SecurityRules{})
	}

	proc := pool.Acquire()
	defer pool.Release(proc)

	proc.mu.Lock()
	defer proc.mu.Unlock()

	execID := proc.id*100000 + int(proc.executionCount) + 1

	execParams := ExecuteParams{
		Script: script,
		Params: params,
		Module: moduleName,
	}
	if bc != nil {
		s := bc.Session()
		execParams.Session = SessionInfo{
			UserID:   s.UserID,
			Username: s.Username,
			Email:    s.Email,
			TenantID: s.TenantID,
			Groups:   s.Groups,
			Locale:   s.Locale,
			Context:  s.Context,
		}
	}

	paramsJSON, err := json.Marshal(execParams)
	if err != nil {
		return nil, fmt.Errorf("marshal execute params: %w", err)
	}

	msg := Message{
		Type:   MsgTypeExecute,
		ID:     execID,
		Params: paramsJSON,
	}

	if err := proc.send(msg); err != nil {
		return nil, fmt.Errorf("send execute request: %w", err)
	}

	type receiveResult struct {
		msg *Message
		err error
	}

	for {
		ch := make(chan receiveResult, 1)
		go func() {
			msg, err := proc.receive()
			ch <- receiveResult{msg, err}
		}()

		var rr receiveResult
		select {
		case <-ctx.Done():
			return nil, fmt.Errorf("script execution cancelled: %w", ctx.Err())
		case rr = <-ch:
		}

		if rr.err != nil {
			return nil, fmt.Errorf("receive from process: %w", rr.err)
		}
		resp := rr.msg

		switch resp.Type {
		case MsgTypeBridgeRequest:
			if err := m.handleBridgeRequest(ctx, proc, bc, resp); err != nil {
				return nil, fmt.Errorf("bridge request failed: %w", err)
			}

		case MsgTypeExecuteComplete:
			if resp.ID == execID {
				var result any
				if resp.Result != nil {
					json.Unmarshal(resp.Result, &result)
				}
				return result, nil
			}

		case MsgTypeExecuteError:
			if resp.ID == execID {
				if resp.Error != nil {
					return nil, &bridge.BridgeError{
						Code:    resp.Error.Code,
						Message: resp.Error.Message,
					}
				}
				return nil, fmt.Errorf("script execution failed")
			}

		default:
			log.Printf("[PLUGIN] unexpected message type: %s", resp.Type)
		}
	}
}

func (m *Manager) handleBridgeRequest(ctx context.Context, proc *PluginProcess, bc *bridge.Context, msg *Message) error {
	if bc == nil {
		return proc.send(Message{
			Type: MsgTypeBridgeResponse,
			ID:   msg.ID,
			Error: &MessageError{
				Code:    "NO_BRIDGE_CONTEXT",
				Message: "bridge context not available for this execution",
			},
		})
	}

	var params map[string]any
	if msg.Params != nil {
		json.Unmarshal(msg.Params, &params)
	}
	if params == nil {
		params = make(map[string]any)
	}

	effectiveCtx := bc
	if msg.TxID != "" && !strings.HasPrefix(msg.Method, "tx.") {
		if txCtx := m.txStore.GetContext(msg.TxID); txCtx != nil {
			effectiveCtx = txCtx
		}
	}

	var result any
	var bridgeErr *bridge.BridgeError

	if strings.HasPrefix(msg.Method, "tx.") {
		result, bridgeErr = m.handleTxMethod(ctx, bc, msg.Method, params)
	} else {
		result, bridgeErr = m.bridgeHandler.Handle(ctx, effectiveCtx, msg.Method, params)
	}

	if bridgeErr != nil {
		return proc.send(Message{
			Type: MsgTypeBridgeResponse,
			ID:   msg.ID,
			Error: &MessageError{
				Code:      bridgeErr.Code,
				Message:   bridgeErr.Message,
				Details:   bridgeErr.Details,
				Retryable: bridgeErr.Retryable,
			},
		})
	}

	resultJSON, _ := json.Marshal(result)
	return proc.send(Message{
		Type:   MsgTypeBridgeResponse,
		ID:     msg.ID,
		Result: resultJSON,
	})
}

func (m *Manager) handleTxMethod(ctx context.Context, bc *bridge.Context, method string, params map[string]any) (any, *bridge.BridgeError) {
	switch method {
	case "tx.begin":
		txID, _, err := m.txStore.Begin(bc)
		if err != nil {
			return nil, bridge.NewError(bridge.ErrInternalError, "failed to begin transaction: "+err.Error())
		}
		return map[string]any{"txId": txID}, nil

	case "tx.commit":
		txID := getString(params, "txId")
		if txID == "" {
			return nil, bridge.NewError(bridge.ErrValidation, "txId is required")
		}
		if err := m.txStore.Commit(txID); err != nil {
			return nil, bridge.NewError(bridge.ErrInternalError, "commit failed: "+err.Error())
		}
		return map[string]any{"committed": true}, nil

	case "tx.rollback":
		txID := getString(params, "txId")
		if txID == "" {
			return nil, bridge.NewError(bridge.ErrValidation, "txId is required")
		}
		if err := m.txStore.Rollback(txID); err != nil {
			return nil, bridge.NewError(bridge.ErrInternalError, "rollback failed: "+err.Error())
		}
		return nil, nil

	default:
		return nil, bridge.NewErrorf("UNKNOWN_METHOD", "unknown tx method: %s", method)
	}
}

func (m *Manager) detectRuntime(script string, explicitRuntime string) string {
	if explicitRuntime != "" {
		if explicitRuntime == "typescript" {
			return "node"
		}
		return explicitRuntime
	}
	if strings.HasSuffix(script, ".py") {
		return "python"
	}
	if strings.HasSuffix(script, ".go") {
		return "go"
	}
	return "node"
}

func (m *Manager) StopAll() {
	if m.txStore != nil {
		m.txStore.CleanupAll()
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.workerPool != nil {
		m.workerPool.Stop()
		m.workerPool = nil
	}
	if m.backgroundPool != nil {
		m.backgroundPool.Stop()
		m.backgroundPool = nil
	}
	m.nodeAvailable = false
}

// StartTypescript is kept for backward compatibility. Delegates to StartNodePool.
func (m *Manager) StartTypescript(nodeCmd string) error {
	if nodeCmd != "" {
		m.runtimeConfig.NodeCommand = nodeCmd
	}
	return m.StartNodePool()
}

func (m *Manager) StartPython(pythonCmd string) error {
	return nil
}

func (m *Manager) StopPlugin(name string) error {
	return nil
}

func detectJSEngine(forceCommand string) (command string, engine string, err error) {
	if forceCommand != "" {
		p, lookErr := exec.LookPath(forceCommand)
		if lookErr != nil {
			return "", "", fmt.Errorf("%s not found in PATH", forceCommand)
		}
		if strings.Contains(forceCommand, "bun") {
			ver := getEngineVersion(p)
			if ver != "" && !isVersionAtLeast(ver, 1, 2, 15) {
				return "", "", fmt.Errorf("Bun %s found but 1.2.15+ required for vm support", ver)
			}
			return p, "bun", nil
		}
		ver := getEngineVersion(p)
		if ver != "" && !isVersionAtLeast(ver, 20, 0, 0) {
			return "", "", fmt.Errorf("Node.js %s found but 20.0.0+ required", ver)
		}
		return p, "nodejs", nil
	}

	if p, lookErr := exec.LookPath("bun"); lookErr == nil {
		ver := getEngineVersion(p)
		if ver != "" && isVersionAtLeast(ver, 1, 2, 15) {
			log.Printf("[PLUGIN] Bun %s detected (faster startup, native TS)", ver)
			return p, "bun", nil
		}
		if ver != "" {
			log.Printf("[WARN] Bun %s found but 1.2.15+ required for vm support, skipping", ver)
		}
	}

	if p, lookErr := exec.LookPath("node"); lookErr == nil {
		ver := getEngineVersion(p)
		if ver != "" && isVersionAtLeast(ver, 20, 0, 0) {
			log.Printf("[PLUGIN] Node.js %s detected", ver)
			return p, "nodejs", nil
		}
		if ver != "" {
			log.Printf("[WARN] Node.js %s found but 20.0.0+ required, skipping", ver)
		}
	}

	return "", "", fmt.Errorf("neither Bun (1.2.15+) nor Node.js (20.0.0+) found in PATH")
}

func getEngineVersion(binPath string) string {
	out, err := exec.Command(binPath, "--version").Output()
	if err != nil {
		return ""
	}
	ver := strings.TrimSpace(string(out))
	ver = strings.TrimPrefix(ver, "v")
	if idx := strings.IndexByte(ver, '\n'); idx >= 0 {
		ver = ver[:idx]
	}
	return ver
}

func isVersionAtLeast(version string, minMajor, minMinor, minPatch int) bool {
	parts := strings.SplitN(version, "-", 2)
	nums := strings.SplitN(parts[0], ".", 3)

	major, minor, patch := 0, 0, 0
	if len(nums) >= 1 {
		fmt.Sscanf(nums[0], "%d", &major)
	}
	if len(nums) >= 2 {
		fmt.Sscanf(nums[1], "%d", &minor)
	}
	if len(nums) >= 3 {
		fmt.Sscanf(nums[2], "%d", &patch)
	}

	if major != minMajor {
		return major > minMajor
	}
	if minor != minMinor {
		return minor > minMinor
	}
	return patch >= minPatch
}
