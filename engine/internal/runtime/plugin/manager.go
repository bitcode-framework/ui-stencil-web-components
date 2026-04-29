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
	nodePool       *ProcessPool
	bridgeHandler  *BridgeHandler
	bridgeFactory  *bridge.Factory
	runtimeConfig  RuntimeConfig
	mu             sync.RWMutex
	nodeAvailable  bool
}

func NewManager() *Manager {
	return &Manager{
		bridgeHandler: &BridgeHandler{},
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

	pool := NewProcessPool(command, []string{"plugins/node/runtime.js"}, cfg.WorkerPool)
	if err := pool.Start(); err != nil {
		return fmt.Errorf("failed to start Node.js pool: %w", err)
	}

	m.mu.Lock()
	m.nodePool = pool
	m.nodeAvailable = true
	m.mu.Unlock()

	log.Printf("[PLUGIN] Node.js runtime started (pool: %d, engine: %s)", cfg.WorkerPool.Size, engine)
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
	pool := m.nodePool
	available := m.nodeAvailable
	m.mu.RUnlock()

	if !available || pool == nil {
		return nil, &bridge.BridgeError{
			Code:    "RUNTIME_NOT_AVAILABLE",
			Message: "Node.js runtime is not available. Install Node.js 20+ or set runtime.node.enabled in bitcode.yaml",
		}
	}

	moduleName := ""
	if runtimeParam, ok := params["__runtime"]; ok {
		delete(params, "__runtime")
		_ = runtimeParam
	}
	if mn, ok := params["__module"]; ok {
		moduleName = fmt.Sprintf("%v", mn)
		delete(params, "__module")
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

	for {
		resp, err := proc.receive()
		if err != nil {
			return nil, fmt.Errorf("receive from process: %w", err)
		}

		switch resp.Type {
		case MsgTypeBridgeRequest:
			m.handleBridgeRequest(ctx, proc, bc, resp)

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

func (m *Manager) handleBridgeRequest(ctx context.Context, proc *PluginProcess, bc *bridge.Context, msg *Message) {
	if bc == nil {
		proc.send(Message{
			Type: MsgTypeBridgeResponse,
			ID:   msg.ID,
			Error: &MessageError{
				Code:    "NO_BRIDGE_CONTEXT",
				Message: "bridge context not available for this execution",
			},
		})
		return
	}

	var params map[string]any
	if msg.Params != nil {
		json.Unmarshal(msg.Params, &params)
	}
	if params == nil {
		params = make(map[string]any)
	}

	var result any
	var bridgeErr *bridge.BridgeError

	if strings.HasPrefix(msg.Method, "tx.") {
		result, bridgeErr = m.handleTxMethod(ctx, bc, msg.Method, params)
	} else {
		result, bridgeErr = m.bridgeHandler.Handle(ctx, bc, msg.Method, params)
	}

	if bridgeErr != nil {
		resultJSON, _ := json.Marshal(nil)
		proc.send(Message{
			Type:   MsgTypeBridgeResponse,
			ID:     msg.ID,
			Result: resultJSON,
			Error: &MessageError{
				Code:      bridgeErr.Code,
				Message:   bridgeErr.Message,
				Details:   bridgeErr.Details,
				Retryable: bridgeErr.Retryable,
			},
		})
	} else {
		resultJSON, _ := json.Marshal(result)
		proc.send(Message{
			Type:   MsgTypeBridgeResponse,
			ID:     msg.ID,
			Result: resultJSON,
		})
	}
}

func (m *Manager) handleTxMethod(ctx context.Context, bc *bridge.Context, method string, params map[string]any) (any, *bridge.BridgeError) {
	switch method {
	case "tx.begin":
		return map[string]any{"txId": "tx-not-implemented"}, nil
	case "tx.commit":
		return nil, nil
	case "tx.rollback":
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
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.nodePool != nil {
		m.nodePool.Stop()
		m.nodePool = nil
		m.nodeAvailable = false
	}
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
		p, err := exec.LookPath(forceCommand)
		if err != nil {
			return "", "", fmt.Errorf("%s not found in PATH", forceCommand)
		}
		if strings.Contains(forceCommand, "bun") {
			return p, "bun", nil
		}
		return p, "nodejs", nil
	}

	if p, err := exec.LookPath("bun"); err == nil {
		return p, "bun", nil
	}

	if p, err := exec.LookPath("node"); err == nil {
		return p, "nodejs", nil
	}

	return "", "", fmt.Errorf("neither Bun nor Node.js found in PATH")
}
