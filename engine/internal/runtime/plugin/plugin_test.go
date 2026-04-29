package plugin

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"
)

func TestMessageMarshal(t *testing.T) {
	msg := Message{
		Type:   MsgTypeExecute,
		ID:     1,
		Method: "execute",
	}
	data, err := json.Marshal(msg)
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}
	var decoded Message
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
	if decoded.Type != MsgTypeExecute {
		t.Errorf("expected type %q, got %q", MsgTypeExecute, decoded.Type)
	}
	if decoded.ID != 1 {
		t.Errorf("expected id 1, got %d", decoded.ID)
	}
}

func TestMessageErrorMarshal(t *testing.T) {
	msg := Message{
		Type: MsgTypeExecuteError,
		ID:   42,
		Error: &MessageError{
			Code:    "SCRIPT_ERROR",
			Message: "something went wrong",
			Details: map[string]any{"line": 10},
		},
	}
	data, err := json.Marshal(msg)
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}
	var decoded Message
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
	if decoded.Error == nil {
		t.Fatal("expected error, got nil")
	}
	if decoded.Error.Code != "SCRIPT_ERROR" {
		t.Errorf("expected code SCRIPT_ERROR, got %q", decoded.Error.Code)
	}
	if decoded.Error.Details["line"] != float64(10) {
		t.Errorf("expected line 10, got %v", decoded.Error.Details["line"])
	}
}

func TestExecuteParamsMarshal(t *testing.T) {
	ep := ExecuteParams{
		Script: "scripts/test.js",
		Params: map[string]any{"key": "value"},
		Module: "crm",
		Session: SessionInfo{
			UserID:   "user-1",
			Username: "admin",
			Locale:   "en",
			Groups:   []string{"admin", "manager"},
		},
	}
	data, err := json.Marshal(ep)
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}
	var decoded ExecuteParams
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
	if decoded.Script != "scripts/test.js" {
		t.Errorf("expected script scripts/test.js, got %q", decoded.Script)
	}
	if decoded.Module != "crm" {
		t.Errorf("expected module crm, got %q", decoded.Module)
	}
	if decoded.Session.UserID != "user-1" {
		t.Errorf("expected userId user-1, got %q", decoded.Session.UserID)
	}
	if len(decoded.Session.Groups) != 2 {
		t.Errorf("expected 2 groups, got %d", len(decoded.Session.Groups))
	}
}

func TestBridgeRequestMarshal(t *testing.T) {
	br := BridgeRequest{
		Method: "model.search",
		Params: map[string]any{
			"model": "lead",
			"opts":  map[string]any{"limit": 10},
		},
		TxID: "tx-123",
	}
	data, err := json.Marshal(br)
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}
	var decoded BridgeRequest
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
	if decoded.Method != "model.search" {
		t.Errorf("expected method model.search, got %q", decoded.Method)
	}
	if decoded.TxID != "tx-123" {
		t.Errorf("expected txId tx-123, got %q", decoded.TxID)
	}
}

func TestMessageConstants(t *testing.T) {
	if MsgTypeExecute != "execute" {
		t.Errorf("MsgTypeExecute = %q", MsgTypeExecute)
	}
	if MsgTypeBridgeResponse != "bridge_response" {
		t.Errorf("MsgTypeBridgeResponse = %q", MsgTypeBridgeResponse)
	}
	if MsgTypeBridgeRequest != "bridge_request" {
		t.Errorf("MsgTypeBridgeRequest = %q", MsgTypeBridgeRequest)
	}
	if MsgTypeExecuteComplete != "execute_complete" {
		t.Errorf("MsgTypeExecuteComplete = %q", MsgTypeExecuteComplete)
	}
	if MsgTypeExecuteError != "execute_error" {
		t.Errorf("MsgTypeExecuteError = %q", MsgTypeExecuteError)
	}
}

func TestPoolConfig(t *testing.T) {
	worker := DefaultWorkerPoolConfig()
	if worker.Size != 4 {
		t.Errorf("expected worker pool size 4, got %d", worker.Size)
	}
	if worker.MaxExecutions != 1000 {
		t.Errorf("expected max executions 1000, got %d", worker.MaxExecutions)
	}

	bg := DefaultBackgroundPoolConfig()
	if bg.Size != 2 {
		t.Errorf("expected background pool size 2, got %d", bg.Size)
	}
	if bg.MaxExecutions != 100 {
		t.Errorf("expected max executions 100, got %d", bg.MaxExecutions)
	}
}

func TestRuntimeConfig(t *testing.T) {
	cfg := DefaultRuntimeConfig()
	if cfg.NodeEnabled != "auto" {
		t.Errorf("expected node enabled auto, got %q", cfg.NodeEnabled)
	}
	if cfg.NodeMinVersion != "20.0.0" {
		t.Errorf("expected min version 20.0.0, got %q", cfg.NodeMinVersion)
	}
}

func TestDetectRuntime(t *testing.T) {
	m := NewManager()

	tests := []struct {
		script   string
		explicit string
		want     string
	}{
		{"scripts/test.ts", "", "node"},
		{"scripts/test.js", "", "node"},
		{"scripts/test.py", "", "python"},
		{"scripts/test.go", "", "go"},
		{"scripts/test.ts", "typescript", "node"},
		{"scripts/test.js", "node", "node"},
	}

	for _, tt := range tests {
		got := m.detectRuntime(tt.script, tt.explicit)
		if got != tt.want {
			t.Errorf("detectRuntime(%q, %q) = %q, want %q", tt.script, tt.explicit, got, tt.want)
		}
	}
}

func TestManagerIsRunning(t *testing.T) {
	m := NewManager()
	if m.IsRunning("node") {
		t.Error("expected node not running before start")
	}
	if m.IsRunning("typescript") {
		t.Error("expected typescript not running before start")
	}
	if m.IsRunning("python") {
		t.Error("expected python not running")
	}
}

func TestBridgeHandlerHelpers(t *testing.T) {
	m := map[string]any{
		"str":   "hello",
		"num":   float64(42),
		"bool":  true,
		"slice": []any{"a", "b"},
		"map":   map[string]any{"key": "val"},
	}

	if s := getString(m, "str"); s != "hello" {
		t.Errorf("getString = %q", s)
	}
	if s := getString(m, "missing"); s != "" {
		t.Errorf("getString missing = %q", s)
	}
	if b := getBool(m, "bool"); !b {
		t.Error("getBool = false")
	}
	if n := getInt(m, "num"); n != 42 {
		t.Errorf("getInt = %d", n)
	}
	if mm := getMap(m, "map"); mm == nil || mm["key"] != "val" {
		t.Errorf("getMap = %v", mm)
	}
	if ss := getStringSlice(m, "slice"); len(ss) != 2 || ss[0] != "a" {
		t.Errorf("getStringSlice = %v", ss)
	}
}

func TestDecodeBinaryFields(t *testing.T) {
	params := map[string]any{
		"content": map[string]any{
			"_type":    "binary",
			"encoding": "base64",
			"data":     "SGVsbG8=",
		},
		"name": "test.txt",
	}

	decodeBinaryFields(params)

	content, ok := params["content"].([]byte)
	if !ok {
		t.Fatalf("expected []byte, got %T", params["content"])
	}
	if string(content) != "Hello" {
		t.Errorf("expected Hello, got %q", string(content))
	}
	if params["name"] != "test.txt" {
		t.Errorf("name should be unchanged, got %v", params["name"])
	}
}

func TestToBridgeError(t *testing.T) {
	if toBridgeError(nil) != nil {
		t.Error("expected nil for nil error")
	}

	be := toBridgeError(fmt.Errorf("test error"))
	if be == nil {
		t.Fatal("expected non-nil bridge error")
	}
	if be.Code != "INTERNAL_ERROR" {
		t.Errorf("expected INTERNAL_ERROR, got %q", be.Code)
	}
}

func TestParseSearchOptions(t *testing.T) {
	m := map[string]any{
		"order":  "name asc",
		"limit":  float64(10),
		"offset": float64(5),
		"fields": []any{"name", "email"},
		"domain": []any{
			[]any{"status", "=", "active"},
		},
	}

	opts := parseSearchOptions(m)
	if opts.Order != "name asc" {
		t.Errorf("order = %q", opts.Order)
	}
	if opts.Limit != 10 {
		t.Errorf("limit = %d", opts.Limit)
	}
	if opts.Offset != 5 {
		t.Errorf("offset = %d", opts.Offset)
	}
	if len(opts.Fields) != 2 {
		t.Errorf("fields len = %d", len(opts.Fields))
	}
	if len(opts.Domain) != 1 {
		t.Errorf("domain len = %d", len(opts.Domain))
	}
}

func TestParseSearchOptionsNil(t *testing.T) {
	opts := parseSearchOptions(nil)
	if opts.Limit != 0 || opts.Order != "" {
		t.Error("expected zero values for nil input")
	}
}

func TestIsVersionAtLeast(t *testing.T) {
	tests := []struct {
		version              string
		major, minor, patch  int
		want                 bool
	}{
		{"22.5.0", 20, 0, 0, true},
		{"20.0.0", 20, 0, 0, true},
		{"20.0.1", 20, 0, 0, true},
		{"19.9.9", 20, 0, 0, false},
		{"18.20.4", 20, 0, 0, false},
		{"1.2.15", 1, 2, 15, true},
		{"1.2.16", 1, 2, 15, true},
		{"1.2.14", 1, 2, 15, false},
		{"1.3.0", 1, 2, 15, true},
		{"2.0.0", 1, 2, 15, true},
		{"0.9.0", 1, 2, 15, false},
		{"22.5.0-nightly", 20, 0, 0, true},
		{"20.0.0-rc.1", 20, 0, 0, true},
		{"18.0.0-rc.1", 20, 0, 0, false},
	}

	for _, tt := range tests {
		got := isVersionAtLeast(tt.version, tt.major, tt.minor, tt.patch)
		if got != tt.want {
			t.Errorf("isVersionAtLeast(%q, %d, %d, %d) = %v, want %v",
				tt.version, tt.major, tt.minor, tt.patch, got, tt.want)
		}
	}
}

func TestGetEngineVersion(t *testing.T) {
	ver := getEngineVersion("nonexistent-binary-xyz")
	if ver != "" {
		t.Errorf("expected empty for nonexistent binary, got %q", ver)
	}
}

func TestTxStoreBasic(t *testing.T) {
	store := newTxStore()
	if store == nil {
		t.Fatal("newTxStore returned nil")
	}
	if len(store.entries) != 0 {
		t.Errorf("expected empty entries, got %d", len(store.entries))
	}
}

func TestTxStoreGetContextMissing(t *testing.T) {
	store := newTxStore()
	ctx := store.GetContext("nonexistent-tx-id")
	if ctx != nil {
		t.Error("expected nil for nonexistent tx")
	}
}

func TestTxStoreRollbackMissing(t *testing.T) {
	store := newTxStore()
	err := store.Rollback("nonexistent-tx-id")
	if err != nil {
		t.Errorf("rollback of nonexistent tx should not error, got: %v", err)
	}
}

func TestTxStoreCommitMissing(t *testing.T) {
	store := newTxStore()
	err := store.Commit("nonexistent-tx-id")
	if err == nil {
		t.Error("commit of nonexistent tx should error")
	}
}

func TestTxStoreCleanupAll(t *testing.T) {
	store := newTxStore()
	store.CleanupAll()
	if len(store.entries) != 0 {
		t.Errorf("expected empty after cleanup, got %d", len(store.entries))
	}
}

func TestRuntimeConfigPythonDefaults(t *testing.T) {
	cfg := DefaultRuntimeConfig()
	if cfg.PythonEnabled != "auto" {
		t.Errorf("expected python enabled auto, got %q", cfg.PythonEnabled)
	}
	if cfg.PythonCommand != "python3" {
		t.Errorf("expected python command python3, got %q", cfg.PythonCommand)
	}
	if cfg.PythonMinVersion != "3.10.0" {
		t.Errorf("expected python min version 3.10.0, got %q", cfg.PythonMinVersion)
	}
}

func TestDetectRuntimePython(t *testing.T) {
	m := NewManager()
	tests := []struct {
		script   string
		explicit string
		want     string
	}{
		{"scripts/analyze.py", "", "python"},
		{"modules/crm/scripts/test.py", "", "python"},
		{"scripts/test.py", "python", "python"},
	}
	for _, tt := range tests {
		got := m.detectRuntime(tt.script, tt.explicit)
		if got != tt.want {
			t.Errorf("detectRuntime(%q, %q) = %q, want %q", tt.script, tt.explicit, got, tt.want)
		}
	}
}

func TestManagerIsRunningPython(t *testing.T) {
	m := NewManager()
	if m.IsRunning("python") {
		t.Error("expected python not running before start")
	}
	m.mu.Lock()
	m.pythonAvailable = true
	m.mu.Unlock()
	if !m.IsRunning("python") {
		t.Error("expected python running after setting pythonAvailable")
	}
}

func TestStartPythonPoolDisabled(t *testing.T) {
	m := NewManager()
	cfg := DefaultRuntimeConfig()
	cfg.PythonEnabled = "false"
	m.SetRuntimeConfig(cfg)
	if err := m.StartPythonPool(); err != nil {
		t.Errorf("expected nil error when disabled, got: %v", err)
	}
	if m.IsRunning("python") {
		t.Error("python should not be running when disabled")
	}
}

func TestDetectPythonEngineVersionCheck(t *testing.T) {
	tests := []struct {
		version string
		min     string
		want    bool
	}{
		{"3.12.1", "3.10.0", true},
		{"3.10.0", "3.10.0", true},
		{"3.11.5", "3.10.0", true},
		{"3.9.18", "3.10.0", false},
		{"3.8.0", "3.10.0", false},
		{"2.7.18", "3.10.0", false},
	}
	for _, tt := range tests {
		var minMajor, minMinor, minPatch int
		fmt.Sscanf(tt.min, "%d.%d.%d", &minMajor, &minMinor, &minPatch)
		got := isVersionAtLeast(tt.version, minMajor, minMinor, minPatch)
		if got != tt.want {
			t.Errorf("isVersionAtLeast(%q, %s) = %v, want %v", tt.version, tt.min, got, tt.want)
		}
	}
}

func TestNewProcessPoolWithPrefix(t *testing.T) {
	pool := NewProcessPoolWithPrefix("echo", []string{"test"}, DefaultWorkerPoolConfig(), "plugin:python")
	if pool.logPrefix != "plugin:python" {
		t.Errorf("expected logPrefix plugin:python, got %q", pool.logPrefix)
	}
	if pool.command != "echo" {
		t.Errorf("expected command echo, got %q", pool.command)
	}
}

func TestNewProcessPoolDefaultPrefix(t *testing.T) {
	pool := NewProcessPool("echo", []string{"test"}, DefaultWorkerPoolConfig())
	if pool.logPrefix != "plugin" {
		t.Errorf("expected default logPrefix plugin, got %q", pool.logPrefix)
	}
}

func TestManagerStopAllWithPython(t *testing.T) {
	m := NewManager()
	m.mu.Lock()
	m.pythonAvailable = true
	m.mu.Unlock()
	m.StopAll()
	if m.IsRunning("python") {
		t.Error("python should not be running after StopAll")
	}
	if m.IsRunning("node") {
		t.Error("node should not be running after StopAll")
	}
}

func TestExecuteRejectsPythonWhenNotAvailable(t *testing.T) {
	m := NewManager()
	_, err := m.Execute(context.Background(), "test.py", map[string]any{}, nil)
	if err == nil {
		t.Fatal("expected error for python when not available")
	}
	if !strings.Contains(err.Error(), "not available") {
		t.Errorf("expected 'not available' error, got: %v", err)
	}
}

func TestProcessShouldRecycle(t *testing.T) {
	proc := &PluginProcess{executionCount: 999}
	cfg := PoolConfig{MaxExecutions: 1000}
	if proc.shouldRecycle(cfg) {
		t.Error("should not recycle at 999")
	}

	proc.executionCount = 1000
	if !proc.shouldRecycle(cfg) {
		t.Error("should recycle at 1000")
	}

	proc.executionCount = 50
	cfg.MaxExecutions = 0
	if proc.shouldRecycle(cfg) {
		t.Error("should not recycle when MaxExecutions=0")
	}
}
