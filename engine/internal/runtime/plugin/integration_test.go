package plugin

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

func findRuntimeJS() string {
	candidates := []string{
		filepath.Join("plugins", "node", "runtime.js"),
		filepath.Join("..", "..", "..", "plugins", "node", "runtime.js"),
	}
	for _, c := range candidates {
		if _, err := os.Stat(c); err == nil {
			abs, _ := filepath.Abs(c)
			return abs
		}
	}
	return ""
}

func nodeAvailable() bool {
	_, err := exec.LookPath("node")
	return err == nil
}

func TestIntegrationSimpleScript(t *testing.T) {
	if !nodeAvailable() {
		t.Skip("Node.js not available")
	}
	runtimeJS := findRuntimeJS()
	if runtimeJS == "" {
		t.Skip("runtime.js not found")
	}

	tmpDir := t.TempDir()
	scriptPath := filepath.Join(tmpDir, "test_simple.js")
	os.WriteFile(scriptPath, []byte(`
		module.exports = {
			async execute(bitcode, params) {
				return { greeting: "hello " + (params.name || "world") };
			}
		};
	`), 0644)

	cmd := exec.Command("node", runtimeJS)
	cmd.Dir = filepath.Dir(runtimeJS)
	stdin, _ := cmd.StdinPipe()
	stdoutPipe, _ := cmd.StdoutPipe()
	cmd.Stderr = os.Stderr

	if err := cmd.Start(); err != nil {
		t.Fatalf("failed to start node: %v", err)
	}
	defer func() {
		stdin.Close()
		cmd.Process.Kill()
		cmd.Wait()
	}()

	scanner := bufio.NewScanner(stdoutPipe)
	scanner.Buffer(make([]byte, 0, 1024*1024), 10*1024*1024)

	execParams := map[string]any{
		"script":  scriptPath,
		"params":  map[string]any{"name": "bitcode"},
		"module":  "",
		"session": map[string]any{},
	}
	paramsJSON, _ := json.Marshal(execParams)

	msg := map[string]any{
		"type":   "execute",
		"id":     1,
		"params": json.RawMessage(paramsJSON),
	}
	data, _ := json.Marshal(msg)
	stdin.Write(append(data, '\n'))

	if !readWithTimeout(scanner, 5*time.Second) {
		t.Fatal("timeout waiting for response")
	}

	var resp Message
	if err := json.Unmarshal(scanner.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal response: %v (raw: %s)", err, scanner.Text())
	}

	if resp.Type != "execute_complete" {
		t.Fatalf("expected execute_complete, got %s (error: %+v)", resp.Type, resp.Error)
	}
	if resp.ID != 1 {
		t.Errorf("expected id 1, got %d", resp.ID)
	}

	var result map[string]any
	json.Unmarshal(resp.Result, &result)
	if result["greeting"] != "hello bitcode" {
		t.Errorf("expected greeting 'hello bitcode', got %v", result["greeting"])
	}
}

func TestIntegrationBridgeCall(t *testing.T) {
	if !nodeAvailable() {
		t.Skip("Node.js not available")
	}
	runtimeJS := findRuntimeJS()
	if runtimeJS == "" {
		t.Skip("runtime.js not found")
	}

	tmpDir := t.TempDir()
	scriptPath := filepath.Join(tmpDir, "test_bridge.js")
	os.WriteFile(scriptPath, []byte(`
		module.exports = {
			async execute(bitcode, params) {
				const result = await bitcode.env("APP_NAME");
				return { appName: result };
			}
		};
	`), 0644)

	cmd := exec.Command("node", runtimeJS)
	cmd.Dir = filepath.Dir(runtimeJS)
	stdin, _ := cmd.StdinPipe()
	stdoutPipe, _ := cmd.StdoutPipe()
	cmd.Stderr = os.Stderr

	if err := cmd.Start(); err != nil {
		t.Fatalf("failed to start node: %v", err)
	}
	defer func() {
		stdin.Close()
		cmd.Process.Kill()
		cmd.Wait()
	}()

	scanner := bufio.NewScanner(stdoutPipe)
	scanner.Buffer(make([]byte, 0, 1024*1024), 10*1024*1024)

	execParams := map[string]any{
		"script":  scriptPath,
		"params":  map[string]any{},
		"module":  "",
		"session": map[string]any{},
	}
	paramsJSON, _ := json.Marshal(execParams)
	msg := map[string]any{
		"type":   "execute",
		"id":     1,
		"params": json.RawMessage(paramsJSON),
	}
	data, _ := json.Marshal(msg)
	stdin.Write(append(data, '\n'))

	if !readWithTimeout(scanner, 5*time.Second) {
		t.Fatal("timeout waiting for bridge request")
	}

	var bridgeReq Message
	if err := json.Unmarshal(scanner.Bytes(), &bridgeReq); err != nil {
		t.Fatalf("unmarshal bridge request: %v", err)
	}

	if bridgeReq.Type != "bridge_request" {
		t.Fatalf("expected bridge_request, got %s", bridgeReq.Type)
	}
	if bridgeReq.Method != "env.get" {
		t.Errorf("expected method env.get, got %s", bridgeReq.Method)
	}

	resultJSON, _ := json.Marshal("BitCode Engine")
	bridgeResp := map[string]any{
		"type":   "bridge_response",
		"id":     bridgeReq.ID,
		"result": json.RawMessage(resultJSON),
	}
	respData, _ := json.Marshal(bridgeResp)
	stdin.Write(append(respData, '\n'))

	if !readWithTimeout(scanner, 5*time.Second) {
		t.Fatal("timeout waiting for execute_complete")
	}

	var execResp Message
	json.Unmarshal(scanner.Bytes(), &execResp)

	if execResp.Type != "execute_complete" {
		t.Fatalf("expected execute_complete, got %s (error: %+v)", execResp.Type, execResp.Error)
	}

	var result map[string]any
	json.Unmarshal(execResp.Result, &result)
	if result["appName"] != "BitCode Engine" {
		t.Errorf("expected appName 'BitCode Engine', got %v", result["appName"])
	}
}

func TestIntegrationScriptError(t *testing.T) {
	if !nodeAvailable() {
		t.Skip("Node.js not available")
	}
	runtimeJS := findRuntimeJS()
	if runtimeJS == "" {
		t.Skip("runtime.js not found")
	}

	tmpDir := t.TempDir()
	scriptPath := filepath.Join(tmpDir, "test_error.js")
	os.WriteFile(scriptPath, []byte(`
		module.exports = {
			async execute(bitcode, params) {
				throw new Error("intentional test error");
			}
		};
	`), 0644)

	cmd := exec.Command("node", runtimeJS)
	cmd.Dir = filepath.Dir(runtimeJS)
	stdin, _ := cmd.StdinPipe()
	stdoutPipe, _ := cmd.StdoutPipe()
	cmd.Stderr = os.Stderr

	if err := cmd.Start(); err != nil {
		t.Fatalf("failed to start node: %v", err)
	}
	defer func() {
		stdin.Close()
		cmd.Process.Kill()
		cmd.Wait()
	}()

	scanner := bufio.NewScanner(stdoutPipe)
	scanner.Buffer(make([]byte, 0, 1024*1024), 10*1024*1024)

	execParams := map[string]any{
		"script":  scriptPath,
		"params":  map[string]any{},
		"module":  "",
		"session": map[string]any{},
	}
	paramsJSON, _ := json.Marshal(execParams)
	msg := map[string]any{
		"type":   "execute",
		"id":     1,
		"params": json.RawMessage(paramsJSON),
	}
	data, _ := json.Marshal(msg)
	stdin.Write(append(data, '\n'))

	if !readWithTimeout(scanner, 5*time.Second) {
		t.Fatal("timeout waiting for error response")
	}

	var resp Message
	json.Unmarshal(scanner.Bytes(), &resp)

	if resp.Type != "execute_error" {
		t.Fatalf("expected execute_error, got %s", resp.Type)
	}
	if resp.Error == nil {
		t.Fatal("expected error details")
	}
	if !strings.Contains(resp.Error.Message, "intentional test error") {
		t.Errorf("expected error message to contain 'intentional test error', got %q", resp.Error.Message)
	}
}

func TestIntegrationTypeScript(t *testing.T) {
	if !nodeAvailable() {
		t.Skip("Node.js not available")
	}
	runtimeJS := findRuntimeJS()
	if runtimeJS == "" {
		t.Skip("runtime.js not found")
	}

	esbuildPath := filepath.Join(filepath.Dir(runtimeJS), "node_modules", "esbuild")
	if _, err := os.Stat(esbuildPath); os.IsNotExist(err) {
		t.Skip("esbuild not installed in plugins/node/")
	}

	tmpDir := t.TempDir()
	scriptPath := filepath.Join(tmpDir, "test_ts.ts")
	os.WriteFile(scriptPath, []byte(`
		interface Result {
			typed: boolean;
			value: number;
		}
		export default {
			async execute(bitcode: any, params: any): Promise<Result> {
				return { typed: true, value: 42 };
			}
		};
	`), 0644)

	cmd := exec.Command("node", runtimeJS)
	cmd.Dir = filepath.Dir(runtimeJS)
	stdin, _ := cmd.StdinPipe()
	stdoutPipe, _ := cmd.StdoutPipe()
	cmd.Stderr = os.Stderr

	if err := cmd.Start(); err != nil {
		t.Fatalf("failed to start node: %v", err)
	}
	defer func() {
		stdin.Close()
		cmd.Process.Kill()
		cmd.Wait()
	}()

	scanner := bufio.NewScanner(stdoutPipe)
	scanner.Buffer(make([]byte, 0, 1024*1024), 10*1024*1024)

	execParams := map[string]any{
		"script":  scriptPath,
		"params":  map[string]any{},
		"module":  "",
		"session": map[string]any{},
	}
	paramsJSON, _ := json.Marshal(execParams)
	msg := map[string]any{
		"type":   "execute",
		"id":     1,
		"params": json.RawMessage(paramsJSON),
	}
	data, _ := json.Marshal(msg)
	stdin.Write(append(data, '\n'))

	if !readWithTimeout(scanner, 10*time.Second) {
		t.Fatal("timeout waiting for TS response")
	}

	var resp Message
	json.Unmarshal(scanner.Bytes(), &resp)

	if resp.Type != "execute_complete" {
		t.Fatalf("expected execute_complete, got %s (error: %+v)", resp.Type, resp.Error)
	}

	var result map[string]any
	json.Unmarshal(resp.Result, &result)
	if result["typed"] != true {
		t.Errorf("expected typed=true, got %v", result["typed"])
	}
	if result["value"] != float64(42) {
		t.Errorf("expected value=42, got %v", result["value"])
	}
}

func TestIntegrationMultipleBridgeCalls(t *testing.T) {
	if !nodeAvailable() {
		t.Skip("Node.js not available")
	}
	runtimeJS := findRuntimeJS()
	if runtimeJS == "" {
		t.Skip("runtime.js not found")
	}

	tmpDir := t.TempDir()
	scriptPath := filepath.Join(tmpDir, "test_multi.js")
	os.WriteFile(scriptPath, []byte(`
		module.exports = {
			async execute(bitcode, params) {
				const env1 = await bitcode.env("KEY1");
				const env2 = await bitcode.env("KEY2");
				return { key1: env1, key2: env2 };
			}
		};
	`), 0644)

	cmd := exec.Command("node", runtimeJS)
	cmd.Dir = filepath.Dir(runtimeJS)
	stdin, _ := cmd.StdinPipe()
	stdoutPipe, _ := cmd.StdoutPipe()
	cmd.Stderr = os.Stderr

	if err := cmd.Start(); err != nil {
		t.Fatalf("failed to start node: %v", err)
	}
	defer func() {
		stdin.Close()
		cmd.Process.Kill()
		cmd.Wait()
	}()

	scanner := bufio.NewScanner(stdoutPipe)
	scanner.Buffer(make([]byte, 0, 1024*1024), 10*1024*1024)

	execParams := map[string]any{
		"script":  scriptPath,
		"params":  map[string]any{},
		"module":  "",
		"session": map[string]any{},
	}
	paramsJSON, _ := json.Marshal(execParams)
	msg := map[string]any{
		"type":   "execute",
		"id":     1,
		"params": json.RawMessage(paramsJSON),
	}
	data, _ := json.Marshal(msg)
	stdin.Write(append(data, '\n'))

	for i := 0; i < 2; i++ {
		if !readWithTimeout(scanner, 5*time.Second) {
			t.Fatalf("timeout waiting for bridge request %d", i+1)
		}

		var bridgeReq Message
		json.Unmarshal(scanner.Bytes(), &bridgeReq)

		if bridgeReq.Type != "bridge_request" {
			t.Fatalf("call %d: expected bridge_request, got %s", i+1, bridgeReq.Type)
		}

		value := fmt.Sprintf("value%d", i+1)
		resultJSON, _ := json.Marshal(value)
		bridgeResp := map[string]any{
			"type":   "bridge_response",
			"id":     bridgeReq.ID,
			"result": json.RawMessage(resultJSON),
		}
		respData, _ := json.Marshal(bridgeResp)
		stdin.Write(append(respData, '\n'))
	}

	if !readWithTimeout(scanner, 5*time.Second) {
		t.Fatal("timeout waiting for execute_complete")
	}

	var execResp Message
	json.Unmarshal(scanner.Bytes(), &execResp)

	if execResp.Type != "execute_complete" {
		t.Fatalf("expected execute_complete, got %s", execResp.Type)
	}

	var result map[string]any
	json.Unmarshal(execResp.Result, &result)
	if result["key1"] != "value1" {
		t.Errorf("expected key1=value1, got %v", result["key1"])
	}
	if result["key2"] != "value2" {
		t.Errorf("expected key2=value2, got %v", result["key2"])
	}
}

func TestIntegrationBridgeError(t *testing.T) {
	if !nodeAvailable() {
		t.Skip("Node.js not available")
	}
	runtimeJS := findRuntimeJS()
	if runtimeJS == "" {
		t.Skip("runtime.js not found")
	}

	tmpDir := t.TempDir()
	scriptPath := filepath.Join(tmpDir, "test_bridge_err.js")
	os.WriteFile(scriptPath, []byte(`
		module.exports = {
			async execute(bitcode, params) {
				try {
					await bitcode.env("SECRET");
					return { error: "should have thrown" };
				} catch (e) {
					return { caught: true, code: e.code, message: e.message };
				}
			}
		};
	`), 0644)

	cmd := exec.Command("node", runtimeJS)
	cmd.Dir = filepath.Dir(runtimeJS)
	stdin, _ := cmd.StdinPipe()
	stdoutPipe, _ := cmd.StdoutPipe()
	cmd.Stderr = os.Stderr

	if err := cmd.Start(); err != nil {
		t.Fatalf("failed to start node: %v", err)
	}
	defer func() {
		stdin.Close()
		cmd.Process.Kill()
		cmd.Wait()
	}()

	scanner := bufio.NewScanner(stdoutPipe)
	scanner.Buffer(make([]byte, 0, 1024*1024), 10*1024*1024)

	execParams := map[string]any{
		"script":  scriptPath,
		"params":  map[string]any{},
		"module":  "",
		"session": map[string]any{},
	}
	paramsJSON, _ := json.Marshal(execParams)
	msg := map[string]any{
		"type":   "execute",
		"id":     1,
		"params": json.RawMessage(paramsJSON),
	}
	msgData, _ := json.Marshal(msg)
	stdin.Write(append(msgData, '\n'))

	if !readWithTimeout(scanner, 5*time.Second) {
		t.Fatal("timeout waiting for bridge request")
	}

	var bridgeReq Message
	json.Unmarshal(scanner.Bytes(), &bridgeReq)

	bridgeResp := map[string]any{
		"type": "bridge_response",
		"id":   bridgeReq.ID,
		"error": map[string]any{
			"code":    "ENV_ACCESS_DENIED",
			"message": "access to SECRET is denied",
		},
	}
	respData, _ := json.Marshal(bridgeResp)
	stdin.Write(append(respData, '\n'))

	if !readWithTimeout(scanner, 5*time.Second) {
		t.Fatal("timeout waiting for execute_complete")
	}

	var execResp Message
	json.Unmarshal(scanner.Bytes(), &execResp)

	if execResp.Type != "execute_complete" {
		t.Fatalf("expected execute_complete, got %s", execResp.Type)
	}

	var result map[string]any
	json.Unmarshal(execResp.Result, &result)
	if result["caught"] != true {
		t.Error("expected error to be caught")
	}
	if result["code"] != "ENV_ACCESS_DENIED" {
		t.Errorf("expected code ENV_ACCESS_DENIED, got %v", result["code"])
	}
}

func TestIntegrationDefinePluginCompat(t *testing.T) {
	if !nodeAvailable() {
		t.Skip("Node.js not available")
	}
	runtimeJS := findRuntimeJS()
	if runtimeJS == "" {
		t.Skip("runtime.js not found")
	}

	esbuildPath := filepath.Join(filepath.Dir(runtimeJS), "node_modules", "esbuild")
	if _, err := os.Stat(esbuildPath); os.IsNotExist(err) {
		t.Skip("esbuild not installed")
	}

	tmpDir := t.TempDir()
	scriptPath := filepath.Join(tmpDir, "test_compat.ts")
	os.WriteFile(scriptPath, []byte(`
		import { definePlugin } from '@bitcode/sdk';
		export default definePlugin({
			async execute(ctx, params) {
				return { compat: true, hasModel: typeof ctx.model === 'function' };
			}
		});
	`), 0644)

	cmd := exec.Command("node", runtimeJS)
	cmd.Dir = filepath.Dir(runtimeJS)
	stdin, _ := cmd.StdinPipe()
	stdoutPipe, _ := cmd.StdoutPipe()
	cmd.Stderr = os.Stderr

	if err := cmd.Start(); err != nil {
		t.Fatalf("failed to start node: %v", err)
	}
	defer func() {
		stdin.Close()
		cmd.Process.Kill()
		cmd.Wait()
	}()

	scanner := bufio.NewScanner(stdoutPipe)
	scanner.Buffer(make([]byte, 0, 1024*1024), 10*1024*1024)

	execParams := map[string]any{
		"script":  scriptPath,
		"params":  map[string]any{},
		"module":  "",
		"session": map[string]any{},
	}
	paramsJSON, _ := json.Marshal(execParams)
	msg := map[string]any{
		"type":   "execute",
		"id":     1,
		"params": json.RawMessage(paramsJSON),
	}
	msgData, _ := json.Marshal(msg)
	stdin.Write(append(msgData, '\n'))

	if !readWithTimeout(scanner, 10*time.Second) {
		t.Fatal("timeout waiting for response")
	}

	var resp Message
	json.Unmarshal(scanner.Bytes(), &resp)

	if resp.Type != "execute_complete" {
		t.Fatalf("expected execute_complete, got %s (error: %+v)", resp.Type, resp.Error)
	}

	var result map[string]any
	json.Unmarshal(resp.Result, &result)
	if result["compat"] != true {
		t.Error("expected compat=true")
	}
	if result["hasModel"] != true {
		t.Error("expected hasModel=true — ctx should have model function")
	}
}

func readWithTimeout(scanner *bufio.Scanner, timeout time.Duration) bool {
	done := make(chan bool, 1)
	go func() {
		done <- scanner.Scan()
	}()
	select {
	case result := <-done:
		return result
	case <-time.After(timeout):
		return false
	}
}

func init() {
	_ = runtime.GOOS
	_ = fmt.Sprintf
}
