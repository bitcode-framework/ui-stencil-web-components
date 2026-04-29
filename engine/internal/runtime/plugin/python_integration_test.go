package plugin

import (
	"bufio"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func findRuntimePy() string {
	candidates := []string{
		filepath.Join("plugins", "python", "runtime.py"),
		filepath.Join("..", "..", "..", "plugins", "python", "runtime.py"),
	}
	for _, c := range candidates {
		if _, err := os.Stat(c); err == nil {
			abs, _ := filepath.Abs(c)
			return abs
		}
	}
	return ""
}

func pythonAvailable() (string, bool) {
	for _, cmd := range []string{"python", "python3"} {
		p, err := exec.LookPath(cmd)
		if err != nil {
			continue
		}
		out, err := exec.Command(p, "--version").Output()
		if err != nil {
			continue
		}
		ver := strings.TrimSpace(string(out))
		ver = strings.TrimPrefix(ver, "Python ")
		if isVersionAtLeast(ver, 3, 10, 0) {
			return p, true
		}
	}
	return "", false
}

func startPython(t *testing.T) (*pythonTestHelper, func()) {
	t.Helper()
	pythonBin, ok := pythonAvailable()
	if !ok {
		t.Skip("Python 3.10+ not available")
	}
	runtimePy := findRuntimePy()
	if runtimePy == "" {
		t.Skip("runtime.py not found")
	}

	cmd := exec.Command(pythonBin, runtimePy)
	cmd.Dir = filepath.Dir(runtimePy)
	stdin, err := cmd.StdinPipe()
	if err != nil {
		t.Fatalf("stdin pipe: %v", err)
	}
	stdoutPipe, err := cmd.StdoutPipe()
	if err != nil {
		t.Fatalf("stdout pipe: %v", err)
	}
	cmd.Stderr = os.Stderr

	if err := cmd.Start(); err != nil {
		t.Fatalf("failed to start python: %v", err)
	}

	scanner := bufio.NewScanner(stdoutPipe)
	scanner.Buffer(make([]byte, 0, 1024*1024), 10*1024*1024)

	h := &pythonTestHelper{
		t:       t,
		stdin:   stdin,
		scanner: scanner,
	}

	cleanup := func() {
		stdin.Close()
		cmd.Process.Kill()
		cmd.Wait()
	}

	return h, cleanup
}

type pythonTestHelper struct {
	t       *testing.T
	stdin   interface{ Write([]byte) (int, error) }
	scanner *bufio.Scanner
}

func (h *pythonTestHelper) sendExecute(id int, scriptPath string, params map[string]any) {
	execParams := map[string]any{
		"script":  scriptPath,
		"params":  params,
		"module":  "",
		"session": map[string]any{},
	}
	paramsJSON, _ := json.Marshal(execParams)
	msg := map[string]any{
		"type":   "execute",
		"id":     id,
		"params": json.RawMessage(paramsJSON),
	}
	data, _ := json.Marshal(msg)
	h.stdin.Write(append(data, '\n'))
}

func (h *pythonTestHelper) sendBridgeResponse(id int, result any) {
	resultJSON, _ := json.Marshal(result)
	resp := map[string]any{
		"type":   "bridge_response",
		"id":     id,
		"result": json.RawMessage(resultJSON),
	}
	data, _ := json.Marshal(resp)
	h.stdin.Write(append(data, '\n'))
}

func (h *pythonTestHelper) sendBridgeError(id int, code, message string) {
	resp := map[string]any{
		"type": "bridge_response",
		"id":   id,
		"error": map[string]any{
			"code":    code,
			"message": message,
		},
	}
	data, _ := json.Marshal(resp)
	h.stdin.Write(append(data, '\n'))
}

func (h *pythonTestHelper) readMessage(timeout time.Duration) *Message {
	if !readWithTimeout(h.scanner, timeout) {
		h.t.Fatalf("timeout waiting for message after %v", timeout)
	}
	var msg Message
	if err := json.Unmarshal(h.scanner.Bytes(), &msg); err != nil {
		h.t.Fatalf("unmarshal: %v (raw: %s)", err, h.scanner.Text())
	}
	return &msg
}

func TestPythonIntegrationSimpleScript(t *testing.T) {
	h, cleanup := startPython(t)
	defer cleanup()

	tmpDir := t.TempDir()
	script := filepath.Join(tmpDir, "test_simple.py")
	os.WriteFile(script, []byte(`
def execute(bitcode, params):
    return {"greeting": "hello " + params.get("name", "world")}
`), 0644)

	h.sendExecute(1, script, map[string]any{"name": "bitcode"})
	resp := h.readMessage(5 * time.Second)

	if resp.Type != "execute_complete" {
		t.Fatalf("expected execute_complete, got %s (error: %+v)", resp.Type, resp.Error)
	}
	var result map[string]any
	json.Unmarshal(resp.Result, &result)
	if result["greeting"] != "hello bitcode" {
		t.Errorf("expected 'hello bitcode', got %v", result["greeting"])
	}
}

func TestPythonIntegrationLegacySignature(t *testing.T) {
	h, cleanup := startPython(t)
	defer cleanup()

	tmpDir := t.TempDir()
	script := filepath.Join(tmpDir, "test_legacy.py")
	os.WriteFile(script, []byte(`
def execute(params):
    return {"legacy": True, "got_name": params.get("name", "")}
`), 0644)

	h.sendExecute(1, script, map[string]any{"name": "test"})
	resp := h.readMessage(5 * time.Second)

	if resp.Type != "execute_complete" {
		t.Fatalf("expected execute_complete, got %s (error: %+v)", resp.Type, resp.Error)
	}
	var result map[string]any
	json.Unmarshal(resp.Result, &result)
	if result["legacy"] != true {
		t.Error("expected legacy=true")
	}
	if result["got_name"] != "test" {
		t.Errorf("expected got_name=test, got %v", result["got_name"])
	}
}

func TestPythonIntegrationBridgeCall(t *testing.T) {
	h, cleanup := startPython(t)
	defer cleanup()

	tmpDir := t.TempDir()
	script := filepath.Join(tmpDir, "test_bridge.py")
	os.WriteFile(script, []byte(`
def execute(bitcode, params):
    result = bitcode.env("APP_NAME")
    return {"app_name": result}
`), 0644)

	h.sendExecute(1, script, map[string]any{})

	bridgeReq := h.readMessage(5 * time.Second)
	if bridgeReq.Type != "bridge_request" {
		t.Fatalf("expected bridge_request, got %s", bridgeReq.Type)
	}
	if bridgeReq.Method != "env.get" {
		t.Errorf("expected method env.get, got %s", bridgeReq.Method)
	}

	h.sendBridgeResponse(bridgeReq.ID, "BitCode Engine")

	resp := h.readMessage(5 * time.Second)
	if resp.Type != "execute_complete" {
		t.Fatalf("expected execute_complete, got %s (error: %+v)", resp.Type, resp.Error)
	}
	var result map[string]any
	json.Unmarshal(resp.Result, &result)
	if result["app_name"] != "BitCode Engine" {
		t.Errorf("expected 'BitCode Engine', got %v", result["app_name"])
	}
}

func TestPythonIntegrationMultipleBridgeCalls(t *testing.T) {
	h, cleanup := startPython(t)
	defer cleanup()

	tmpDir := t.TempDir()
	script := filepath.Join(tmpDir, "test_multi.py")
	os.WriteFile(script, []byte(`
def execute(bitcode, params):
    v1 = bitcode.env("KEY1")
    v2 = bitcode.env("KEY2")
    return {"key1": v1, "key2": v2}
`), 0644)

	h.sendExecute(1, script, map[string]any{})

	for i := 0; i < 2; i++ {
		req := h.readMessage(5 * time.Second)
		if req.Type != "bridge_request" {
			t.Fatalf("call %d: expected bridge_request, got %s", i+1, req.Type)
		}
		h.sendBridgeResponse(req.ID, "value"+string(rune('1'+i)))
	}

	resp := h.readMessage(5 * time.Second)
	if resp.Type != "execute_complete" {
		t.Fatalf("expected execute_complete, got %s", resp.Type)
	}
	var result map[string]any
	json.Unmarshal(resp.Result, &result)
	if result["key1"] != "value1" {
		t.Errorf("expected key1=value1, got %v", result["key1"])
	}
	if result["key2"] != "value2" {
		t.Errorf("expected key2=value2, got %v", result["key2"])
	}
}

func TestPythonIntegrationBridgeError(t *testing.T) {
	h, cleanup := startPython(t)
	defer cleanup()

	tmpDir := t.TempDir()
	script := filepath.Join(tmpDir, "test_bridge_err.py")
	os.WriteFile(script, []byte(`
def execute(bitcode, params):
    try:
        bitcode.env("SECRET")
        return {"error": "should have raised"}
    except Exception as e:
        return {"caught": True, "code": e.code, "message": str(e)}
`), 0644)

	h.sendExecute(1, script, map[string]any{})

	req := h.readMessage(5 * time.Second)
	if req.Type != "bridge_request" {
		t.Fatalf("expected bridge_request, got %s", req.Type)
	}

	h.sendBridgeError(req.ID, "ENV_ACCESS_DENIED", "access to SECRET is denied")

	resp := h.readMessage(5 * time.Second)
	if resp.Type != "execute_complete" {
		t.Fatalf("expected execute_complete, got %s (error: %+v)", resp.Type, resp.Error)
	}
	var result map[string]any
	json.Unmarshal(resp.Result, &result)
	if result["caught"] != true {
		t.Error("expected error to be caught")
	}
	if result["code"] != "ENV_ACCESS_DENIED" {
		t.Errorf("expected code ENV_ACCESS_DENIED, got %v", result["code"])
	}
}

func TestPythonIntegrationScriptError(t *testing.T) {
	h, cleanup := startPython(t)
	defer cleanup()

	tmpDir := t.TempDir()
	script := filepath.Join(tmpDir, "test_error.py")
	os.WriteFile(script, []byte(`
def execute(bitcode, params):
    raise ValueError("intentional test error")
`), 0644)

	h.sendExecute(1, script, map[string]any{})
	resp := h.readMessage(5 * time.Second)

	if resp.Type != "execute_error" {
		t.Fatalf("expected execute_error, got %s", resp.Type)
	}
	if resp.Error == nil {
		t.Fatal("expected error details")
	}
	if !strings.Contains(resp.Error.Message, "intentional test error") {
		t.Errorf("expected 'intentional test error', got %q", resp.Error.Message)
	}
}

func TestPythonIntegrationScriptNotFound(t *testing.T) {
	h, cleanup := startPython(t)
	defer cleanup()

	h.sendExecute(1, "/nonexistent/path/script.py", map[string]any{})
	resp := h.readMessage(5 * time.Second)

	if resp.Type != "execute_error" {
		t.Fatalf("expected execute_error, got %s", resp.Type)
	}
	if resp.Error == nil {
		t.Fatal("expected error details")
	}
	if !strings.Contains(resp.Error.Message, "not found") {
		t.Errorf("expected 'not found' in error, got %q", resp.Error.Message)
	}
}

func TestPythonIntegrationModelSearch(t *testing.T) {
	h, cleanup := startPython(t)
	defer cleanup()

	tmpDir := t.TempDir()
	script := filepath.Join(tmpDir, "test_model.py")
	os.WriteFile(script, []byte(`
def execute(bitcode, params):
    leads = bitcode.model("lead").search({"domain": [["status", "=", "new"]]})
    return {"count": len(leads)}
`), 0644)

	h.sendExecute(1, script, map[string]any{})

	req := h.readMessage(5 * time.Second)
	if req.Type != "bridge_request" {
		t.Fatalf("expected bridge_request, got %s", req.Type)
	}
	if req.Method != "model.search" {
		t.Errorf("expected model.search, got %s", req.Method)
	}

	var reqParams map[string]any
	json.Unmarshal(req.Params, &reqParams)
	if reqParams["model"] != "lead" {
		t.Errorf("expected model=lead, got %v", reqParams["model"])
	}

	h.sendBridgeResponse(req.ID, []map[string]any{
		{"id": "1", "name": "Lead A", "status": "new"},
		{"id": "2", "name": "Lead B", "status": "new"},
	})

	resp := h.readMessage(5 * time.Second)
	if resp.Type != "execute_complete" {
		t.Fatalf("expected execute_complete, got %s (error: %+v)", resp.Type, resp.Error)
	}
	var result map[string]any
	json.Unmarshal(resp.Result, &result)
	if result["count"] != float64(2) {
		t.Errorf("expected count=2, got %v", result["count"])
	}
}

func TestPythonIntegrationSnakeCaseToCamelCase(t *testing.T) {
	h, cleanup := startPython(t)
	defer cleanup()

	tmpDir := t.TempDir()
	script := filepath.Join(tmpDir, "test_snake.py")
	os.WriteFile(script, []byte(`
def execute(bitcode, params):
    bitcode.model("lead").create_many([{"name": "A"}, {"name": "B"}])
    bitcode.model("lead").write_many(["1", "2"], {"status": "done"})
    bitcode.model("lead").delete_many(["3"])
    bitcode.model("lead").add_relation("1", "tags", ["t1"])
    bitcode.model("lead").load_relation("1", "tags")
    bitcode.security.has_group("admin")
    return {"done": True}
`), 0644)

	h.sendExecute(1, script, map[string]any{})

	expectedMethods := []string{
		"model.createMany",
		"model.writeMany",
		"model.deleteMany",
		"model.addRelation",
		"model.loadRelation",
		"security.hasGroup",
	}

	for _, expected := range expectedMethods {
		req := h.readMessage(5 * time.Second)
		if req.Type != "bridge_request" {
			t.Fatalf("expected bridge_request for %s, got %s", expected, req.Type)
		}
		if req.Method != expected {
			t.Errorf("expected method %s, got %s", expected, req.Method)
		}
		h.sendBridgeResponse(req.ID, nil)
	}

	resp := h.readMessage(5 * time.Second)
	if resp.Type != "execute_complete" {
		t.Fatalf("expected execute_complete, got %s (error: %+v)", resp.Type, resp.Error)
	}
}

func TestPythonIntegrationSysPathRestore(t *testing.T) {
	h, cleanup := startPython(t)
	defer cleanup()

	tmpDir := t.TempDir()
	script1 := filepath.Join(tmpDir, "test_path1.py")
	os.WriteFile(script1, []byte(`
import sys
def execute(bitcode, params):
    return {"path_len": len(sys.path)}
`), 0644)

	h.sendExecute(1, script1, map[string]any{})
	resp1 := h.readMessage(5 * time.Second)
	if resp1.Type != "execute_complete" {
		t.Fatalf("exec 1: expected execute_complete, got %s (error: %+v)", resp1.Type, resp1.Error)
	}
	var result1 map[string]any
	json.Unmarshal(resp1.Result, &result1)
	pathLen1 := result1["path_len"]

	script2 := filepath.Join(tmpDir, "test_path2.py")
	os.WriteFile(script2, []byte(`
import sys
def execute(bitcode, params):
    return {"path_len": len(sys.path)}
`), 0644)

	h.sendExecute(2, script2, map[string]any{})
	resp2 := h.readMessage(5 * time.Second)
	if resp2.Type != "execute_complete" {
		t.Fatalf("exec 2: expected execute_complete, got %s (error: %+v)", resp2.Type, resp2.Error)
	}
	var result2 map[string]any
	json.Unmarshal(resp2.Result, &result2)
	pathLen2 := result2["path_len"]

	if pathLen1 != pathLen2 {
		t.Errorf("sys.path pollution: exec1 had %v entries, exec2 had %v", pathLen1, pathLen2)
	}
}

func TestPythonIntegrationSequentialExecutions(t *testing.T) {
	h, cleanup := startPython(t)
	defer cleanup()

	tmpDir := t.TempDir()
	script := filepath.Join(tmpDir, "test_seq.py")
	os.WriteFile(script, []byte(`
def execute(bitcode, params):
    return {"n": params.get("n", 0)}
`), 0644)

	for i := 1; i <= 3; i++ {
		h.sendExecute(i, script, map[string]any{"n": float64(i)})
		resp := h.readMessage(5 * time.Second)
		if resp.Type != "execute_complete" {
			t.Fatalf("exec %d: expected execute_complete, got %s", i, resp.Type)
		}
		if resp.ID != i {
			t.Errorf("exec %d: expected id %d, got %d", i, i, resp.ID)
		}
		var result map[string]any
		json.Unmarshal(resp.Result, &result)
		if result["n"] != float64(i) {
			t.Errorf("exec %d: expected n=%d, got %v", i, i, result["n"])
		}
	}
}

func TestPythonIntegrationHotReload(t *testing.T) {
	h, cleanup := startPython(t)
	defer cleanup()

	tmpDir := t.TempDir()
	script := filepath.Join(tmpDir, "test_reload.py")

	os.WriteFile(script, []byte(`
def execute(bitcode, params):
    return {"version": 1}
`), 0644)

	h.sendExecute(1, script, map[string]any{})
	resp1 := h.readMessage(5 * time.Second)
	if resp1.Type != "execute_complete" {
		t.Fatalf("v1: expected execute_complete, got %s", resp1.Type)
	}
	var result1 map[string]any
	json.Unmarshal(resp1.Result, &result1)
	if result1["version"] != float64(1) {
		t.Errorf("expected version=1, got %v", result1["version"])
	}

	os.WriteFile(script, []byte(`
def execute(bitcode, params):
    return {"version": 2}
`), 0644)

	h.sendExecute(2, script, map[string]any{})
	resp2 := h.readMessage(5 * time.Second)
	if resp2.Type != "execute_complete" {
		t.Fatalf("v2: expected execute_complete, got %s", resp2.Type)
	}
	var result2 map[string]any
	json.Unmarshal(resp2.Result, &result2)
	if result2["version"] != float64(2) {
		t.Errorf("expected version=2 after hot reload, got %v", result2["version"])
	}
}
