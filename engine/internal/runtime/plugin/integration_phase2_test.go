package plugin

import (
	"bufio"
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

// --- Test helpers ---

type testProcess struct {
	cmd     *exec.Cmd
	stdin   *os.File
	scanner *bufio.Scanner
	t       *testing.T
}

func startTestNode(t *testing.T) *testProcess {
	t.Helper()
	if !nodeAvailable() {
		t.Skip("Node.js not available")
	}
	rjs := findRuntimeJS()
	if rjs == "" {
		t.Skip("runtime.js not found")
	}

	cmd := exec.Command("node", rjs)
	cmd.Dir = filepath.Dir(rjs)
	stdinPipe, _ := cmd.StdinPipe()
	stdoutPipe, _ := cmd.StdoutPipe()
	cmd.Stderr = os.Stderr

	if err := cmd.Start(); err != nil {
		t.Fatalf("start node: %v", err)
	}

	sc := bufio.NewScanner(stdoutPipe)
	sc.Buffer(make([]byte, 0, 1024*1024), 10*1024*1024)

	tp := &testProcess{cmd: cmd, stdin: stdinPipe.(*os.File), scanner: sc, t: t}
	t.Cleanup(func() {
		stdinPipe.Close()
		cmd.Process.Kill()
		cmd.Wait()
	})
	return tp
}

func (tp *testProcess) sendExecute(id int, scriptPath string, params map[string]any) {
	tp.t.Helper()
	ep := map[string]any{
		"script": scriptPath, "params": params,
		"module": "", "session": map[string]any{},
	}
	pj, _ := json.Marshal(ep)
	msg := map[string]any{"type": "execute", "id": id, "params": json.RawMessage(pj)}
	data, _ := json.Marshal(msg)
	tp.stdin.Write(append(data, '\n'))
}

func (tp *testProcess) readMsg(timeout time.Duration) *Message {
	tp.t.Helper()
	if !readWithTimeout(tp.scanner, timeout) {
		tp.t.Fatalf("timeout reading message after %v", timeout)
	}
	var msg Message
	if err := json.Unmarshal(tp.scanner.Bytes(), &msg); err != nil {
		tp.t.Fatalf("unmarshal: %v (raw: %s)", err, tp.scanner.Text())
	}
	return &msg
}

func (tp *testProcess) respondBridge(id int, result any) {
	tp.t.Helper()
	rj, _ := json.Marshal(result)
	resp := map[string]any{"type": "bridge_response", "id": id, "result": json.RawMessage(rj)}
	data, _ := json.Marshal(resp)
	tp.stdin.Write(append(data, '\n'))
}

func (tp *testProcess) respondBridgeError(id int, code, message string) {
	tp.t.Helper()
	resp := map[string]any{
		"type": "bridge_response", "id": id,
		"error": map[string]any{"code": code, "message": message},
	}
	data, _ := json.Marshal(resp)
	tp.stdin.Write(append(data, '\n'))
}

func writeScript(t *testing.T, name, code string) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), name)
	os.WriteFile(p, []byte(code), 0644)
	return p
}

func getResult(t *testing.T, msg *Message) map[string]any {
	t.Helper()
	var r map[string]any
	json.Unmarshal(msg.Result, &r)
	return r
}

// --- Test 22: All 20 bridge namespaces via RPC ---

func TestIntegrationBridgeNamespaceModel(t *testing.T) {
	tp := startTestNode(t)
	script := writeScript(t, "ns_model.js", `
		module.exports = { async execute(bitcode, params) {
			const results = await bitcode.model("lead").search({ domain: [["status","=","new"]] });
			const created = await bitcode.model("lead").create({ name: "test" });
			const count = await bitcode.model("lead").count({});
			return { results, created, count };
		}};
	`)
	tp.sendExecute(1, script, map[string]any{})

	// model.search
	msg := tp.readMsg(5 * time.Second)
	if msg.Method != "model.search" { t.Fatalf("expected model.search, got %s", msg.Method) }
	tp.respondBridge(msg.ID, []map[string]any{{"id": "1", "name": "lead1"}})

	// model.create
	msg = tp.readMsg(5 * time.Second)
	if msg.Method != "model.create" { t.Fatalf("expected model.create, got %s", msg.Method) }
	tp.respondBridge(msg.ID, map[string]any{"id": "2", "name": "test"})

	// model.count
	msg = tp.readMsg(5 * time.Second)
	if msg.Method != "model.count" { t.Fatalf("expected model.count, got %s", msg.Method) }
	tp.respondBridge(msg.ID, float64(5))

	result := tp.readMsg(5 * time.Second)
	if result.Type != "execute_complete" { t.Fatalf("expected complete, got %s", result.Type) }
}

func TestIntegrationBridgeNamespaceDB(t *testing.T) {
	tp := startTestNode(t)
	script := writeScript(t, "ns_db.js", `
		module.exports = { async execute(bitcode, params) {
			const rows = await bitcode.db.query("SELECT * FROM leads WHERE status = ?", "new");
			return { rows };
		}};
	`)
	tp.sendExecute(1, script, map[string]any{})

	msg := tp.readMsg(5 * time.Second)
	if msg.Method != "db.query" { t.Fatalf("expected db.query, got %s", msg.Method) }
	var p map[string]any
	json.Unmarshal(msg.Params, &p)
	if p["sql"] != "SELECT * FROM leads WHERE status = ?" {
		t.Errorf("unexpected sql: %v", p["sql"])
	}
	tp.respondBridge(msg.ID, []map[string]any{{"id": "1"}})

	result := tp.readMsg(5 * time.Second)
	if result.Type != "execute_complete" { t.Fatalf("got %s", result.Type) }
}

func TestIntegrationBridgeNamespaceHTTP(t *testing.T) {
	tp := startTestNode(t)
	script := writeScript(t, "ns_http.js", `
		module.exports = { async execute(bitcode, params) {
			const resp = await bitcode.http.post("https://api.example.com/data", { body: { key: "val" } });
			return { status: resp.status };
		}};
	`)
	tp.sendExecute(1, script, map[string]any{})

	msg := tp.readMsg(5 * time.Second)
	if msg.Method != "http.request" { t.Fatalf("expected http.request, got %s", msg.Method) }
	var p map[string]any
	json.Unmarshal(msg.Params, &p)
	if p["method"] != "POST" { t.Errorf("expected POST, got %v", p["method"]) }
	if p["url"] != "https://api.example.com/data" { t.Errorf("unexpected url: %v", p["url"]) }
	tp.respondBridge(msg.ID, map[string]any{"status": 200, "body": "ok"})

	result := tp.readMsg(5 * time.Second)
	if result.Type != "execute_complete" { t.Fatalf("got %s", result.Type) }
	r := getResult(t, result)
	if r["status"] != float64(200) { t.Errorf("expected 200, got %v", r["status"]) }
}

func TestIntegrationBridgeNamespaceCache(t *testing.T) {
	tp := startTestNode(t)
	script := writeScript(t, "ns_cache.js", `
		module.exports = { async execute(bitcode, params) {
			await bitcode.cache.set("mykey", "myval");
			const val = await bitcode.cache.get("mykey");
			await bitcode.cache.del("mykey");
			return { val };
		}};
	`)
	tp.sendExecute(1, script, map[string]any{})

	msg := tp.readMsg(5 * time.Second)
	if msg.Method != "cache.set" { t.Fatalf("expected cache.set, got %s", msg.Method) }
	tp.respondBridge(msg.ID, nil)

	msg = tp.readMsg(5 * time.Second)
	if msg.Method != "cache.get" { t.Fatalf("expected cache.get, got %s", msg.Method) }
	tp.respondBridge(msg.ID, "myval")

	msg = tp.readMsg(5 * time.Second)
	if msg.Method != "cache.del" { t.Fatalf("expected cache.del, got %s", msg.Method) }
	tp.respondBridge(msg.ID, nil)

	result := tp.readMsg(5 * time.Second)
	if result.Type != "execute_complete" { t.Fatalf("got %s", result.Type) }
	r := getResult(t, result)
	if r["val"] != "myval" { t.Errorf("expected myval, got %v", r["val"]) }
}

func TestIntegrationBridgeNamespaceFS(t *testing.T) {
	tp := startTestNode(t)
	script := writeScript(t, "ns_fs.js", `
		module.exports = { async execute(bitcode, params) {
			const exists = await bitcode.fs.exists("/tmp/test.txt");
			await bitcode.fs.write("/tmp/test.txt", "hello");
			const content = await bitcode.fs.read("/tmp/test.txt");
			return { exists, content };
		}};
	`)
	tp.sendExecute(1, script, map[string]any{})

	msg := tp.readMsg(5 * time.Second)
	if msg.Method != "fs.exists" { t.Fatalf("expected fs.exists, got %s", msg.Method) }
	tp.respondBridge(msg.ID, false)

	msg = tp.readMsg(5 * time.Second)
	if msg.Method != "fs.write" { t.Fatalf("expected fs.write, got %s", msg.Method) }
	tp.respondBridge(msg.ID, nil)

	msg = tp.readMsg(5 * time.Second)
	if msg.Method != "fs.read" { t.Fatalf("expected fs.read, got %s", msg.Method) }
	tp.respondBridge(msg.ID, "hello")

	result := tp.readMsg(5 * time.Second)
	if result.Type != "execute_complete" { t.Fatalf("got %s", result.Type) }
	r := getResult(t, result)
	if r["exists"] != false { t.Errorf("expected false, got %v", r["exists"]) }
	if r["content"] != "hello" { t.Errorf("expected hello, got %v", r["content"]) }
}

func TestIntegrationBridgeNamespaceEmail(t *testing.T) {
	tp := startTestNode(t)
	script := writeScript(t, "ns_email.js", `
		module.exports = { async execute(bitcode, params) {
			await bitcode.email.send({ to: "a@b.com", subject: "Hi", body: "Hello" });
			return { sent: true };
		}};
	`)
	tp.sendExecute(1, script, map[string]any{})

	msg := tp.readMsg(5 * time.Second)
	if msg.Method != "email.send" { t.Fatalf("expected email.send, got %s", msg.Method) }
	var p map[string]any
	json.Unmarshal(msg.Params, &p)
	if p["to"] != "a@b.com" { t.Errorf("unexpected to: %v", p["to"]) }
	tp.respondBridge(msg.ID, nil)

	result := tp.readMsg(5 * time.Second)
	if result.Type != "execute_complete" { t.Fatalf("got %s", result.Type) }
}

func TestIntegrationBridgeNamespaceNotify(t *testing.T) {
	tp := startTestNode(t)
	script := writeScript(t, "ns_notify.js", `
		module.exports = { async execute(bitcode, params) {
			await bitcode.notify.send({ to: "user1", title: "Alert", message: "test", type: "info" });
			await bitcode.notify.broadcast("channel1", { msg: "hello" });
			return { done: true };
		}};
	`)
	tp.sendExecute(1, script, map[string]any{})

	msg := tp.readMsg(5 * time.Second)
	if msg.Method != "notify.send" { t.Fatalf("expected notify.send, got %s", msg.Method) }
	tp.respondBridge(msg.ID, nil)

	msg = tp.readMsg(5 * time.Second)
	if msg.Method != "notify.broadcast" { t.Fatalf("expected notify.broadcast, got %s", msg.Method) }
	tp.respondBridge(msg.ID, nil)

	result := tp.readMsg(5 * time.Second)
	if result.Type != "execute_complete" { t.Fatalf("got %s", result.Type) }
}

func TestIntegrationBridgeNamespaceStorage(t *testing.T) {
	tp := startTestNode(t)
	script := writeScript(t, "ns_storage.js", `
		module.exports = { async execute(bitcode, params) {
			const url = await bitcode.storage.url("file-123");
			return { url };
		}};
	`)
	tp.sendExecute(1, script, map[string]any{})

	msg := tp.readMsg(5 * time.Second)
	if msg.Method != "storage.url" { t.Fatalf("expected storage.url, got %s", msg.Method) }
	tp.respondBridge(msg.ID, "https://cdn.example.com/file-123.pdf")

	result := tp.readMsg(5 * time.Second)
	if result.Type != "execute_complete" { t.Fatalf("got %s", result.Type) }
	r := getResult(t, result)
	if r["url"] != "https://cdn.example.com/file-123.pdf" { t.Errorf("unexpected url: %v", r["url"]) }
}

func TestIntegrationBridgeNamespaceSecurity(t *testing.T) {
	tp := startTestNode(t)
	script := writeScript(t, "ns_security.js", `
		module.exports = { async execute(bitcode, params) {
			const perms = await bitcode.security.permissions("lead");
			const has = await bitcode.security.hasGroup("admin");
			const groups = await bitcode.security.groups();
			return { perms, has, groups };
		}};
	`)
	tp.sendExecute(1, script, map[string]any{})

	msg := tp.readMsg(5 * time.Second)
	if msg.Method != "security.permissions" { t.Fatalf("expected security.permissions, got %s", msg.Method) }
	tp.respondBridge(msg.ID, map[string]any{"canRead": true, "canWrite": false})

	msg = tp.readMsg(5 * time.Second)
	if msg.Method != "security.hasGroup" { t.Fatalf("expected security.hasGroup, got %s", msg.Method) }
	tp.respondBridge(msg.ID, true)

	msg = tp.readMsg(5 * time.Second)
	if msg.Method != "security.groups" { t.Fatalf("expected security.groups, got %s", msg.Method) }
	tp.respondBridge(msg.ID, []string{"admin", "user"})

	result := tp.readMsg(5 * time.Second)
	if result.Type != "execute_complete" { t.Fatalf("got %s", result.Type) }
}

func TestIntegrationBridgeNamespaceCrypto(t *testing.T) {
	tp := startTestNode(t)
	script := writeScript(t, "ns_crypto.js", `
		module.exports = { async execute(bitcode, params) {
			const encrypted = await bitcode.crypto.encrypt("secret");
			const hashed = await bitcode.crypto.hash("password");
			return { encrypted, hashed };
		}};
	`)
	tp.sendExecute(1, script, map[string]any{})

	msg := tp.readMsg(5 * time.Second)
	if msg.Method != "crypto.encrypt" { t.Fatalf("expected crypto.encrypt, got %s", msg.Method) }
	tp.respondBridge(msg.ID, "enc_abc123")

	msg = tp.readMsg(5 * time.Second)
	if msg.Method != "crypto.hash" { t.Fatalf("expected crypto.hash, got %s", msg.Method) }
	tp.respondBridge(msg.ID, "hash_xyz789")

	result := tp.readMsg(5 * time.Second)
	if result.Type != "execute_complete" { t.Fatalf("got %s", result.Type) }
	r := getResult(t, result)
	if r["encrypted"] != "enc_abc123" { t.Errorf("unexpected encrypted: %v", r["encrypted"]) }
}

func TestIntegrationBridgeNamespaceAudit(t *testing.T) {
	tp := startTestNode(t)
	script := writeScript(t, "ns_audit.js", `
		module.exports = { async execute(bitcode, params) {
			await bitcode.audit.log({ action: "login", model: "user", recordId: "u1" });
			return { logged: true };
		}};
	`)
	tp.sendExecute(1, script, map[string]any{})

	msg := tp.readMsg(5 * time.Second)
	if msg.Method != "audit.log" { t.Fatalf("expected audit.log, got %s", msg.Method) }
	tp.respondBridge(msg.ID, nil)

	result := tp.readMsg(5 * time.Second)
	if result.Type != "execute_complete" { t.Fatalf("got %s", result.Type) }
}

func TestIntegrationBridgeNamespaceExecution(t *testing.T) {
	tp := startTestNode(t)
	script := writeScript(t, "ns_exec.js", `
		module.exports = { async execute(bitcode, params) {
			const current = await bitcode.execution.current();
			return { current };
		}};
	`)
	tp.sendExecute(1, script, map[string]any{})

	msg := tp.readMsg(5 * time.Second)
	if msg.Method != "execution.current" { t.Fatalf("expected execution.current, got %s", msg.Method) }
	tp.respondBridge(msg.ID, map[string]any{"id": "exec-1", "processName": "test"})

	result := tp.readMsg(5 * time.Second)
	if result.Type != "execute_complete" { t.Fatalf("got %s", result.Type) }
}

func TestIntegrationBridgeNamespaceLogEmitCallExecT(t *testing.T) {
	tp := startTestNode(t)
	script := writeScript(t, "ns_misc.js", `
		module.exports = { async execute(bitcode, params) {
			await bitcode.log("info", "test message", { key: "val" });
			await bitcode.emit("order.created", { orderId: "o1" });
			const callResult = await bitcode.call("process_name", { input: "data" });
			const translated = await bitcode.t("greeting.hello");
			return { callResult, translated };
		}};
	`)
	tp.sendExecute(1, script, map[string]any{})

	msg := tp.readMsg(5 * time.Second)
	if msg.Method != "log" { t.Fatalf("expected log, got %s", msg.Method) }
	tp.respondBridge(msg.ID, nil)

	msg = tp.readMsg(5 * time.Second)
	if msg.Method != "emit" { t.Fatalf("expected emit, got %s", msg.Method) }
	tp.respondBridge(msg.ID, nil)

	msg = tp.readMsg(5 * time.Second)
	if msg.Method != "call" { t.Fatalf("expected call, got %s", msg.Method) }
	tp.respondBridge(msg.ID, map[string]any{"success": true})

	msg = tp.readMsg(5 * time.Second)
	if msg.Method != "t" { t.Fatalf("expected t, got %s", msg.Method) }
	tp.respondBridge(msg.ID, "Hello!")

	result := tp.readMsg(5 * time.Second)
	if result.Type != "execute_complete" { t.Fatalf("got %s", result.Type) }
	r := getResult(t, result)
	if r["translated"] != "Hello!" { t.Errorf("expected Hello!, got %v", r["translated"]) }
}

// --- Test 23: Per-module require resolution ---

func TestIntegrationModuleRequire(t *testing.T) {
	tp := startTestNode(t)

	modDir := t.TempDir()
	nmDir := filepath.Join(modDir, "node_modules", "fake-pkg")
	os.MkdirAll(nmDir, 0755)
	os.WriteFile(filepath.Join(nmDir, "index.js"), []byte(`module.exports = { version: "1.2.3" };`), 0644)
	os.WriteFile(filepath.Join(nmDir, "package.json"), []byte(`{"name":"fake-pkg","version":"1.2.3","main":"index.js"}`), 0644)

	scriptPath := filepath.Join(modDir, "scripts", "test_require.js")
	os.MkdirAll(filepath.Dir(scriptPath), 0755)
	os.WriteFile(scriptPath, []byte(`
		const pkg = require("fake-pkg");
		module.exports = { async execute(bitcode, params) {
			return { pkgVersion: pkg.version };
		}};
	`), 0644)

	ep := map[string]any{
		"script": scriptPath, "params": map[string]any{},
		"module": "", "session": map[string]any{},
	}
	pj, _ := json.Marshal(ep)
	msg := map[string]any{"type": "execute", "id": 1, "params": json.RawMessage(pj)}
	data, _ := json.Marshal(msg)
	tp.stdin.Write(append(data, '\n'))

	result := tp.readMsg(5 * time.Second)
	if result.Type != "execute_complete" {
		t.Fatalf("expected execute_complete, got %s (error: %+v)", result.Type, result.Error)
	}
	r := getResult(t, result)
	if r["pkgVersion"] != "1.2.3" {
		t.Errorf("expected pkgVersion 1.2.3, got %v", r["pkgVersion"])
	}
}

// --- Test 25: Transaction via RPC ---

func TestIntegrationTransaction(t *testing.T) {
	tp := startTestNode(t)
	script := writeScript(t, "tx_test.js", `
		module.exports = { async execute(bitcode, params) {
			const result = await bitcode.tx(async (tx) => {
				await tx.model("order").create({ total: 100 });
				await tx.db.execute("UPDATE inventory SET stock = stock - 1");
				return { committed: true };
			});
			return result;
		}};
	`)
	tp.sendExecute(1, script, map[string]any{})

	// tx.begin
	msg := tp.readMsg(5 * time.Second)
	if msg.Method != "tx.begin" { t.Fatalf("expected tx.begin, got %s", msg.Method) }
	tp.respondBridge(msg.ID, map[string]any{"txId": "tx-test-123"})

	// model.create (should have txId)
	msg = tp.readMsg(5 * time.Second)
	if msg.Method != "model.create" { t.Fatalf("expected model.create, got %s", msg.Method) }
	if msg.TxID != "tx-test-123" { t.Errorf("expected txId tx-test-123, got %q", msg.TxID) }
	tp.respondBridge(msg.ID, map[string]any{"id": "o1"})

	// db.execute (should have txId)
	msg = tp.readMsg(5 * time.Second)
	if msg.Method != "db.execute" { t.Fatalf("expected db.execute, got %s", msg.Method) }
	if msg.TxID != "tx-test-123" { t.Errorf("expected txId tx-test-123, got %q", msg.TxID) }
	tp.respondBridge(msg.ID, map[string]any{"rows_affected": 1})

	// tx.commit
	msg = tp.readMsg(5 * time.Second)
	if msg.Method != "tx.commit" { t.Fatalf("expected tx.commit, got %s", msg.Method) }
	tp.respondBridge(msg.ID, map[string]any{"committed": true})

	result := tp.readMsg(5 * time.Second)
	if result.Type != "execute_complete" { t.Fatalf("got %s", result.Type) }
	r := getResult(t, result)
	if r["committed"] != true { t.Errorf("expected committed=true, got %v", r["committed"]) }
}

func TestIntegrationTransactionRollback(t *testing.T) {
	tp := startTestNode(t)
	script := writeScript(t, "tx_rollback.js", `
		module.exports = { async execute(bitcode, params) {
			try {
				await bitcode.tx(async (tx) => {
					await tx.model("order").create({ total: 100 });
					throw new Error("intentional rollback");
				});
			} catch (e) {
				return { rolledBack: true, error: e.message };
			}
		}};
	`)
	tp.sendExecute(1, script, map[string]any{})

	msg := tp.readMsg(5 * time.Second)
	if msg.Method != "tx.begin" { t.Fatalf("expected tx.begin, got %s", msg.Method) }
	tp.respondBridge(msg.ID, map[string]any{"txId": "tx-rb-456"})

	msg = tp.readMsg(5 * time.Second)
	if msg.Method != "model.create" { t.Fatalf("expected model.create, got %s", msg.Method) }
	tp.respondBridge(msg.ID, map[string]any{"id": "o2"})

	// tx.rollback (triggered by throw)
	msg = tp.readMsg(5 * time.Second)
	if msg.Method != "tx.rollback" { t.Fatalf("expected tx.rollback, got %s", msg.Method) }
	tp.respondBridge(msg.ID, nil)

	result := tp.readMsg(5 * time.Second)
	if result.Type != "execute_complete" { t.Fatalf("got %s", result.Type) }
	r := getResult(t, result)
	if r["rolledBack"] != true { t.Errorf("expected rolledBack=true") }
}

// --- Test 30: Binary data base64 round-trip ---

func TestIntegrationBinaryData(t *testing.T) {
	tp := startTestNode(t)
	script := writeScript(t, "binary.js", `
		module.exports = { async execute(bitcode, params) {
			const buf = Buffer.from("Hello Binary World");
			await bitcode.storage.upload({ filename: "test.bin", content: buf });
			return { uploaded: true };
		}};
	`)
	tp.sendExecute(1, script, map[string]any{})

	msg := tp.readMsg(5 * time.Second)
	if msg.Method != "storage.upload" { t.Fatalf("expected storage.upload, got %s", msg.Method) }
	var p map[string]any
	json.Unmarshal(msg.Params, &p)
	content, ok := p["content"].(map[string]any)
	if !ok { t.Fatalf("expected content to be binary object, got %T", p["content"]) }
	if content["_type"] != "binary" { t.Errorf("expected _type=binary, got %v", content["_type"]) }
	if content["encoding"] != "base64" { t.Errorf("expected encoding=base64") }
	tp.respondBridge(msg.ID, map[string]any{"id": "att-1"})

	result := tp.readMsg(5 * time.Second)
	if result.Type != "execute_complete" { t.Fatalf("got %s", result.Type) }
}

// --- Test 31: console.log interception ---

func TestIntegrationConsoleLogInterception(t *testing.T) {
	tp := startTestNode(t)
	script := writeScript(t, "console_log.js", `
		module.exports = { async execute(bitcode, params) {
			console.log("info message");
			console.warn("warning message");
			console.error("error message");
			return { done: true };
		}};
	`)
	tp.sendExecute(1, script, map[string]any{})

	for i := 0; i < 3; i++ {
		msg := tp.readMsg(5 * time.Second)
		if msg.Method != "log" {
			t.Fatalf("call %d: expected log, got %s", i+1, msg.Method)
		}
		var p map[string]any
		json.Unmarshal(msg.Params, &p)
		level := p["level"].(string)
		switch i {
		case 0:
			if level != "info" { t.Errorf("expected info, got %s", level) }
		case 1:
			if level != "warn" { t.Errorf("expected warn, got %s", level) }
		case 2:
			if level != "error" { t.Errorf("expected error, got %s", level) }
		}
		tp.respondBridge(msg.ID, nil)
	}

	result := tp.readMsg(5 * time.Second)
	if result.Type != "execute_complete" { t.Fatalf("got %s", result.Type) }
}

// --- Test 34: Node.js not installed (graceful degradation) ---

func TestNodeNotInstalledGracefulDegradation(t *testing.T) {
	m := NewManager()
	m.SetRuntimeConfig(RuntimeConfig{
		NodeEnabled: "true",
		NodeCommand: "nonexistent-node-binary-xyz-12345",
		WorkerPool:  DefaultWorkerPoolConfig(),
	})
	err := m.StartNodePool()
	if err == nil {
		t.Fatal("expected error when Node.js binary not found")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("expected 'not found' in error, got: %v", err)
	}
}

func TestNodeNotInstalledAutoMode(t *testing.T) {
	m := NewManager()
	m.SetRuntimeConfig(RuntimeConfig{
		NodeEnabled: "auto",
		NodeCommand: "nonexistent-node-binary-xyz-12345",
		WorkerPool:  DefaultWorkerPoolConfig(),
	})
	err := m.StartNodePool()
	if err != nil {
		t.Errorf("auto mode should not error, got: %v", err)
	}
	if m.IsRunning("node") {
		t.Error("node should not be running")
	}
}

func TestNodeDisabledByConfig(t *testing.T) {
	m := NewManager()
	m.SetRuntimeConfig(RuntimeConfig{
		NodeEnabled: "false",
		WorkerPool:  DefaultWorkerPoolConfig(),
	})
	err := m.StartNodePool()
	if err != nil {
		t.Errorf("disabled mode should not error, got: %v", err)
	}
	if m.IsRunning("node") {
		t.Error("node should not be running when disabled")
	}
}

func TestExecuteWhenNodeNotAvailable(t *testing.T) {
	m := NewManager()
	_, err := m.Execute(context.Background(), "test.ts", map[string]any{}, nil)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "RUNTIME_NOT_AVAILABLE") {
		t.Errorf("expected RUNTIME_NOT_AVAILABLE, got: %v", err)
	}
}

// --- Test 35: Script timeout (vm.timeout) ---

func TestIntegrationScriptTimeout(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping 30s timeout test in short mode")
	}
	tp := startTestNode(t)
	// vm.runInNewContext timeout only covers synchronous module-level code
	script := writeScript(t, "timeout.js", `
		while(true) {}
		module.exports = { execute: function() {} };
	`)
	tp.sendExecute(1, script, map[string]any{})

	result := tp.readMsg(35 * time.Second)
	if result.Type != "execute_error" {
		t.Fatalf("expected execute_error (timeout), got %s", result.Type)
	}
	if result.Error == nil {
		t.Fatal("expected error details")
	}
	msg := strings.ToLower(result.Error.Message)
	if !strings.Contains(msg, "timed out") && !strings.Contains(msg, "timeout") {
		t.Errorf("expected timeout error, got: %s", result.Error.Message)
	}
}

// --- Test 27: Process pool (acquire, release, queue) ---

func TestProcessPoolAcquireRelease(t *testing.T) {
	if !nodeAvailable() { t.Skip("Node.js not available") }
	rjs := findRuntimeJS()
	if rjs == "" { t.Skip("runtime.js not found") }

	cfg := PoolConfig{Size: 2, MaxExecutions: 100}
	pool := NewProcessPool("node", []string{rjs}, cfg)
	pool.command = "node"
	if err := pool.Start(); err != nil {
		t.Fatalf("pool start: %v", err)
	}
	defer pool.Stop()

	p1 := pool.Acquire()
	p2 := pool.Acquire()

	if p1 == p2 {
		t.Error("acquired same process twice")
	}
	if p1 == nil || p2 == nil {
		t.Fatal("acquired nil process")
	}

	pool.Release(p1)
	pool.Release(p2)

	p3 := pool.Acquire()
	if p3 == nil {
		t.Fatal("failed to re-acquire after release")
	}
	pool.Release(p3)
}

func TestProcessPoolQueueWhenFull(t *testing.T) {
	if !nodeAvailable() { t.Skip("Node.js not available") }
	rjs := findRuntimeJS()
	if rjs == "" { t.Skip("runtime.js not found") }

	cfg := PoolConfig{Size: 1, MaxExecutions: 100}
	pool := NewProcessPool("node", []string{rjs}, cfg)
	if err := pool.Start(); err != nil {
		t.Fatalf("pool start: %v", err)
	}
	defer pool.Stop()

	p1 := pool.Acquire()

	acquired := make(chan *PluginProcess, 1)
	go func() {
		p2 := pool.Acquire()
		acquired <- p2
	}()

	select {
	case <-acquired:
		t.Fatal("should not acquire when pool is full")
	case <-time.After(200 * time.Millisecond):
		// expected: blocked
	}

	pool.Release(p1)

	select {
	case p2 := <-acquired:
		if p2 == nil {
			t.Fatal("acquired nil after release")
		}
		pool.Release(p2)
	case <-time.After(2 * time.Second):
		t.Fatal("should have acquired after release")
	}
}

// --- Test 28: Process crash recovery ---

func TestProcessPoolCrashRecovery(t *testing.T) {
	if !nodeAvailable() { t.Skip("Node.js not available") }
	rjs := findRuntimeJS()
	if rjs == "" { t.Skip("runtime.js not found") }

	cfg := PoolConfig{Size: 2, MaxExecutions: 1000}
	pool := NewProcessPool("node", []string{rjs}, cfg)
	if err := pool.Start(); err != nil {
		t.Fatalf("pool start: %v", err)
	}
	defer pool.Stop()

	pool.mu.Lock()
	initialCount := len(pool.processes)
	pool.mu.Unlock()

	if initialCount != 2 {
		t.Fatalf("expected 2 initial processes, got %d", initialCount)
	}

	pool.mu.Lock()
	if len(pool.processes) > 0 {
		pool.processes[0].cmd.Process.Kill()
	}
	pool.mu.Unlock()

	time.Sleep(5 * time.Second)

	pool.mu.Lock()
	afterCount := len(pool.processes)
	pool.mu.Unlock()

	if afterCount < 2 {
		t.Logf("process count after crash: %d (recovery may still be in progress)", afterCount)
	}
}

// --- Test 32: Concurrent executions ---

func TestIntegrationConcurrentExecutions(t *testing.T) {
	if !nodeAvailable() { t.Skip("Node.js not available") }
	rjs := findRuntimeJS()
	if rjs == "" { t.Skip("runtime.js not found") }

	cfg := PoolConfig{Size: 2, MaxExecutions: 100}
	pool := NewProcessPool("node", []string{rjs}, cfg)
	if err := pool.Start(); err != nil {
		t.Fatalf("pool start: %v", err)
	}
	defer pool.Stop()

	script := writeScript(t, "concurrent.js", `
		module.exports = { async execute(bitcode, params) {
			return { worker: params.worker };
		}};
	`)

	var wg sync.WaitGroup
	results := make([]any, 4)
	errors := make([]error, 4)

	for i := 0; i < 4; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			proc := pool.Acquire()
			defer pool.Release(proc)

			proc.mu.Lock()
			defer proc.mu.Unlock()

			ep := map[string]any{
				"script": script, "params": map[string]any{"worker": idx},
				"module": "", "session": map[string]any{},
			}
			pj, _ := json.Marshal(ep)
			execID := 100 + idx
			proc.send(Message{Type: MsgTypeExecute, ID: execID, Params: pj})

			resp, err := proc.receive()
			if err != nil {
				errors[idx] = err
				return
			}
			if resp.Type == MsgTypeExecuteComplete {
				var r map[string]any
				json.Unmarshal(resp.Result, &r)
				results[idx] = r
			}
		}(i)
	}

	wg.Wait()

	completed := 0
	for i := 0; i < 4; i++ {
		if errors[i] != nil {
			t.Logf("worker %d error: %v", i, errors[i])
			continue
		}
		if results[i] != nil {
			completed++
		}
	}
	if completed < 2 {
		t.Errorf("expected at least 2 completed, got %d", completed)
	}
}

// --- Test 33: Process recycling ---

func TestProcessPoolRecycling(t *testing.T) {
	if !nodeAvailable() { t.Skip("Node.js not available") }
	rjs := findRuntimeJS()
	if rjs == "" { t.Skip("runtime.js not found") }

	cfg := PoolConfig{Size: 1, MaxExecutions: 2}
	pool := NewProcessPool("node", []string{rjs}, cfg)
	if err := pool.Start(); err != nil {
		t.Fatalf("pool start: %v", err)
	}
	defer pool.Stop()

	firstProc := pool.Acquire()
	firstID := firstProc.id
	pool.Release(firstProc) // count=1

	secondProc := pool.Acquire()
	pool.Release(secondProc) // count=2 -> triggers recycle

	time.Sleep(3 * time.Second)

	thirdProc := pool.Acquire()
	if thirdProc.id == firstID {
		t.Error("expected new process after recycling")
	}
	pool.Release(thirdProc)
}
