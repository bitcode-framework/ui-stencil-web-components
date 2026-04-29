#!/usr/bin/env python3
"""BitCode Python Runtime - Bidirectional JSON-RPC bridge.

Architecture (single process, pool model):
- Router thread: reads ALL stdin, routes messages by type
- Main thread: picks execute requests from queue, runs scripts
- bridge_call(): blocks on threading.Event, unblocked by router thread

Each Python process handles ONE execution at a time.
Go's ProcessPool routes executions to available processes.
"""

import sys
import json
import threading
import inspect
import traceback
import os
import platform
import base64
import asyncio
import queue

# --- Communication Layer ---

_stdout_lock = threading.Lock()
_pending_lock = threading.Lock()
_pending_requests = {}
_next_bridge_id = 0
_bridge_id_lock = threading.Lock()
_execute_queue = queue.Queue()
_SHUTDOWN_SENTINEL = object()


def send_to_go(message):
    """Send JSON message to Go via stdout."""
    with _stdout_lock:
        sys.stdout.write(json.dumps(message) + "\n")
        sys.stdout.flush()


def bridge_call(method, params, tx_id=None):
    """Send bridge request to Go and BLOCK until response arrives.

    This is called from the script thread (synchronous).
    The reader thread will receive the response and signal us.
    """
    global _next_bridge_id
    with _bridge_id_lock:
        _next_bridge_id += 1
        req_id = _next_bridge_id

    event = threading.Event()
    with _pending_lock:
        _pending_requests[req_id] = {"event": event, "result": None, "error": None}

    msg = {
        "type": "bridge_request",
        "id": req_id,
        "method": method,
        "params": serialize_params(params),
    }
    if tx_id:
        msg["txId"] = tx_id
    send_to_go(msg)

    event.wait(timeout=60)

    with _pending_lock:
        entry = _pending_requests.pop(req_id, None)
    if entry is None or not event.is_set():
        raise BridgeError("BRIDGE_TIMEOUT", "Bridge call '{}' timed out after 60s".format(method))

    if entry["error"]:
        err = entry["error"]
        raise BridgeError(
            err.get("code", "INTERNAL_ERROR"),
            err.get("message", "Unknown error"),
            err.get("details"),
            err.get("retryable", False),
        )

    return deserialize_from_json(entry["result"])


# --- Binary Data Serialization ---


def serialize_for_json(value):
    """Recursively convert Python values to JSON-safe types.

    Handles bytes->base64, sets->lists, and other non-serializable types
    that Python scripts may return but JSON cannot represent.
    """
    if value is None:
        return value
    if isinstance(value, bytes):
        return {
            "_type": "binary",
            "encoding": "base64",
            "data": base64.b64encode(value).decode(),
        }
    if isinstance(value, dict):
        return {k: serialize_for_json(v) for k, v in value.items()}
    if isinstance(value, (list, tuple)):
        return [serialize_for_json(v) for v in value]
    if isinstance(value, set):
        return [serialize_for_json(v) for v in value]
    if isinstance(value, (int, float, str, bool)):
        return value
    return str(value)


serialize_params = serialize_for_json


def deserialize_from_json(value):
    """Recursively convert JSON-RPC response back to native Python types.

    Reverses the binary encoding: {"_type": "binary", "encoding": "base64", "data": "..."} -> bytes.
    Without this, every developer using storage.download() or any binary-returning
    bridge method must manually base64-decode — leaking protocol internals.
    """
    if value is None:
        return value
    if isinstance(value, dict):
        if value.get("_type") == "binary" and value.get("encoding") == "base64":
            return base64.b64decode(value.get("data", ""))
        return {k: deserialize_from_json(v) for k, v in value.items()}
    if isinstance(value, list):
        return [deserialize_from_json(v) for v in value]
    return value


# --- Error Type ---


class BridgeError(Exception):
    """Structured error from bridge operations."""

    def __init__(self, code, message, details=None, retryable=False):
        super().__init__(message)
        self.code = code
        self.details = details
        self.retryable = retryable


# --- Bridge Proxy (bitcode.*) ---


class ModelHandle:
    """Proxy for bitcode.model(name) operations."""

    def __init__(self, model_name, sudo=False, tenant=None, skip_val=False):
        self._model = model_name
        self._sudo = sudo
        self._tenant = tenant
        self._skip_val = skip_val

    def _params(self, **kwargs):
        p = {"model": self._model, **kwargs}
        if self._sudo:
            p["sudo"] = True
        if self._tenant:
            p["tenant"] = self._tenant
        if self._skip_val:
            p["skipValidation"] = True
        return p

    # Single record CRUD
    def search(self, opts=None):
        return bridge_call("model.search", self._params(opts=opts or {}))

    def get(self, id, opts=None):
        return bridge_call("model.get", self._params(id=id, opts=opts))

    def create(self, data):
        return bridge_call("model.create", self._params(data=data))

    def write(self, id, data):
        return bridge_call("model.write", self._params(id=id, data=data))

    def delete(self, id):
        return bridge_call("model.delete", self._params(id=id))

    def count(self, opts=None):
        return bridge_call("model.count", self._params(opts=opts or {}))

    def sum(self, field, opts=None):
        return bridge_call("model.sum", self._params(field=field, opts=opts or {}))

    def upsert(self, data, unique):
        return bridge_call("model.upsert", self._params(data=data, unique=unique))

    # Bulk operations
    def create_many(self, records):
        return bridge_call("model.createMany", self._params(records=records))

    def write_many(self, ids, data):
        return bridge_call("model.writeMany", self._params(ids=ids, data=data))

    def delete_many(self, ids):
        return bridge_call("model.deleteMany", self._params(ids=ids))

    def upsert_many(self, records, unique):
        return bridge_call("model.upsertMany", self._params(records=records, unique=unique))

    # Relation operations
    def add_relation(self, id, field, related_ids):
        return bridge_call(
            "model.addRelation",
            self._params(id=id, field=field, relatedIds=related_ids),
        )

    def remove_relation(self, id, field, related_ids):
        return bridge_call(
            "model.removeRelation",
            self._params(id=id, field=field, relatedIds=related_ids),
        )

    def set_relation(self, id, field, related_ids):
        return bridge_call(
            "model.setRelation",
            self._params(id=id, field=field, relatedIds=related_ids),
        )

    def load_relation(self, id, field):
        return bridge_call("model.loadRelation", self._params(id=id, field=field))

    # Mode switching
    def sudo(self):
        return SudoModelHandle(self._model)


class SudoModelHandle(ModelHandle):
    """ModelHandle with sudo mode - bypasses permissions."""

    def __init__(self, model_name, tenant=None, skip_val=False):
        super().__init__(model_name, sudo=True, tenant=tenant, skip_val=skip_val)

    def hard_delete(self, id):
        return bridge_call("model.hardDelete", self._params(id=id))

    def hard_delete_many(self, ids):
        return bridge_call("model.hardDeleteMany", self._params(ids=ids))

    def with_tenant(self, tenant_id):
        return SudoModelHandle(self._model, tenant=tenant_id, skip_val=self._skip_val)

    def skip_validation(self):
        return SudoModelHandle(self._model, tenant=self._tenant, skip_val=True)


class DBProxy:
    def query(self, sql, *args):
        return bridge_call("db.query", {"sql": sql, "args": list(args)})

    def execute(self, sql, *args):
        return bridge_call("db.execute", {"sql": sql, "args": list(args)})


class HTTPProxy:
    def get(self, url, **opts):
        return bridge_call("http.request", {"method": "GET", "url": url, **opts})

    def post(self, url, **opts):
        return bridge_call("http.request", {"method": "POST", "url": url, **opts})

    def put(self, url, **opts):
        return bridge_call("http.request", {"method": "PUT", "url": url, **opts})

    def patch(self, url, **opts):
        return bridge_call("http.request", {"method": "PATCH", "url": url, **opts})

    def delete(self, url, **opts):
        return bridge_call("http.request", {"method": "DELETE", "url": url, **opts})


class CacheProxy:
    def get(self, key):
        return bridge_call("cache.get", {"key": key})

    def set(self, key, value, ttl=None):
        opts = {"key": key, "value": value}
        if ttl:
            opts["ttl"] = ttl
        return bridge_call("cache.set", opts)

    def delete(self, key):
        return bridge_call("cache.del", {"key": key})


class FSProxy:
    def read(self, path):
        return bridge_call("fs.read", {"path": path})

    def write(self, path, content):
        return bridge_call("fs.write", {"path": path, "content": content})

    def exists(self, path):
        return bridge_call("fs.exists", {"path": path})

    def list(self, path):
        return bridge_call("fs.list", {"path": path})

    def mkdir(self, path):
        return bridge_call("fs.mkdir", {"path": path})

    def remove(self, path):
        return bridge_call("fs.remove", {"path": path})


class EmailProxy:
    def send(self, **opts):
        return bridge_call("email.send", opts)


class NotifyProxy:
    def send(self, **opts):
        return bridge_call("notify.send", opts)

    def broadcast(self, channel, data):
        return bridge_call("notify.broadcast", {"channel": channel, "data": data})


class StorageProxy:
    def upload(self, **opts):
        return bridge_call("storage.upload", opts)

    def url(self, id):
        return bridge_call("storage.url", {"id": id})

    def download(self, id):
        return bridge_call("storage.download", {"id": id})

    def delete(self, id):
        return bridge_call("storage.delete", {"id": id})


class SecurityProxy:
    def permissions(self, model):
        return bridge_call("security.permissions", {"model": model})

    def has_group(self, group):
        return bridge_call("security.hasGroup", {"group": group})

    def groups(self):
        return bridge_call("security.groups", {})


class AuditProxy:
    def log(self, **opts):
        return bridge_call("audit.log", opts)


class CryptoProxy:
    def encrypt(self, text):
        return bridge_call("crypto.encrypt", {"text": text})

    def decrypt(self, text):
        return bridge_call("crypto.decrypt", {"text": text})

    def hash(self, value):
        return bridge_call("crypto.hash", {"value": value})

    def verify(self, value, hash_str):
        return bridge_call("crypto.verify", {"value": value, "hash": hash_str})


class ExecutionProxy:
    def search(self, **opts):
        return bridge_call("execution.search", opts)

    def get(self, id, **opts):
        return bridge_call("execution.get", {"id": id, **opts})

    def current(self):
        return bridge_call("execution.current", {})

    def retry(self, id):
        return bridge_call("execution.retry", {"id": id})

    def cancel(self, id):
        return bridge_call("execution.cancel", {"id": id})


class BitcodeContext:
    """Main bridge context - exposed as 'bitcode' to scripts."""

    def __init__(self, session):
        self.session = session
        self.db = DBProxy()
        self.http = HTTPProxy()
        self.cache = CacheProxy()
        self.fs = FSProxy()
        self.email = EmailProxy()
        self.notify = NotifyProxy()
        self.storage = StorageProxy()
        self.security = SecurityProxy()
        self.audit = AuditProxy()
        self.crypto = CryptoProxy()
        self.execution = ExecutionProxy()

    def model(self, name):
        return ModelHandle(name)

    def env(self, key):
        return bridge_call("env.get", {"key": key})

    def config(self, key):
        return bridge_call("config.get", {"key": key})

    def log(self, level, msg, data=None):
        bridge_call("log", {"level": level, "msg": msg, "data": data})

    def emit(self, event, data=None):
        return bridge_call("emit", {"event": event, "data": data or {}})

    def call(self, process, input_data=None):
        return bridge_call("call", {"process": process, "input": input_data or {}})

    def exec(self, cmd, args=None, **opts):
        return bridge_call("exec", {"cmd": cmd, "args": args or [], **opts})

    def t(self, key):
        return bridge_call("t", {"key": key})

    def tx(self, fn):
        """Execute fn inside a database transaction.

        Creates a tx-scoped context where all bridge calls carry the txId,
        mirroring Node.js runtime's txBitcode pattern.
        """
        result = bridge_call("tx.begin", {})
        tx_id = result.get("txId") if isinstance(result, dict) else None
        tx_ctx = TxBitcodeContext(self.session, tx_id)
        try:
            ret = fn(tx_ctx)
            bridge_call("tx.commit", {"txId": tx_id})
            return ret
        except Exception:
            try:
                bridge_call("tx.rollback", {"txId": tx_id})
            except Exception:
                pass
            raise


class TxModelHandle(ModelHandle):
    """ModelHandle that passes txId on every bridge call."""

    def __init__(self, model_name, tx_id, sudo=False, tenant=None, skip_val=False):
        super().__init__(model_name, sudo=sudo, tenant=tenant, skip_val=skip_val)
        self._tx_id = tx_id

    def _bridge(self, method, params):
        return bridge_call(method, params, tx_id=self._tx_id)

    def search(self, opts=None):
        return self._bridge("model.search", self._params(opts=opts or {}))

    def get(self, id, opts=None):
        return self._bridge("model.get", self._params(id=id, opts=opts))

    def create(self, data):
        return self._bridge("model.create", self._params(data=data))

    def write(self, id, data):
        return self._bridge("model.write", self._params(id=id, data=data))

    def delete(self, id):
        return self._bridge("model.delete", self._params(id=id))

    def count(self, opts=None):
        return self._bridge("model.count", self._params(opts=opts or {}))

    def sum(self, field, opts=None):
        return self._bridge("model.sum", self._params(field=field, opts=opts or {}))

    def upsert(self, data, unique):
        return self._bridge("model.upsert", self._params(data=data, unique=unique))

    def create_many(self, records):
        return self._bridge("model.createMany", self._params(records=records))

    def write_many(self, ids, data):
        return self._bridge("model.writeMany", self._params(ids=ids, data=data))

    def delete_many(self, ids):
        return self._bridge("model.deleteMany", self._params(ids=ids))

    def upsert_many(self, records, unique):
        return self._bridge("model.upsertMany", self._params(records=records, unique=unique))

    def add_relation(self, id, field, related_ids):
        return self._bridge(
            "model.addRelation",
            self._params(id=id, field=field, relatedIds=related_ids),
        )

    def remove_relation(self, id, field, related_ids):
        return self._bridge(
            "model.removeRelation",
            self._params(id=id, field=field, relatedIds=related_ids),
        )

    def set_relation(self, id, field, related_ids):
        return self._bridge(
            "model.setRelation",
            self._params(id=id, field=field, relatedIds=related_ids),
        )

    def load_relation(self, id, field):
        return self._bridge("model.loadRelation", self._params(id=id, field=field))

    def sudo(self):
        return TxSudoModelHandle(self._model, self._tx_id)


class TxSudoModelHandle(TxModelHandle):
    def __init__(self, model_name, tx_id, tenant=None, skip_val=False):
        super().__init__(model_name, tx_id, sudo=True, tenant=tenant, skip_val=skip_val)

    def hard_delete(self, id):
        return self._bridge("model.hardDelete", self._params(id=id))

    def hard_delete_many(self, ids):
        return self._bridge("model.hardDeleteMany", self._params(ids=ids))

    def with_tenant(self, tenant_id):
        return TxSudoModelHandle(self._model, self._tx_id, tenant=tenant_id, skip_val=self._skip_val)

    def skip_validation(self):
        return TxSudoModelHandle(self._model, self._tx_id, tenant=self._tenant, skip_val=True)


class TxDBProxy:
    def __init__(self, tx_id):
        self._tx_id = tx_id

    def query(self, sql, *args):
        return bridge_call("db.query", {"sql": sql, "args": list(args)}, tx_id=self._tx_id)

    def execute(self, sql, *args):
        return bridge_call("db.execute", {"sql": sql, "args": list(args)}, tx_id=self._tx_id)


class TxBitcodeContext(BitcodeContext):
    """Transaction-scoped context — all bridge calls carry txId."""

    def __init__(self, session, tx_id):
        super().__init__(session)
        self._tx_id = tx_id
        self.db = TxDBProxy(tx_id)

    def model(self, name):
        return TxModelHandle(name, self._tx_id)

    def tx(self, fn):
        raise BridgeError("TX_NESTED", "nested transactions are not supported")


# --- venv Path Resolution (Cross-Platform) ---


def get_venv_site_packages(module_dir):
    """Get the venv site-packages path, handling OS differences.

    Linux/macOS: .venv/lib/python3.X/site-packages/
    Windows:     .venv/Lib/site-packages/
    """
    venv_dir = os.path.join(module_dir, ".venv")
    if not os.path.isdir(venv_dir):
        return None

    if platform.system() == "Windows":
        site = os.path.join(venv_dir, "Lib", "site-packages")
    else:
        site = os.path.join(
            venv_dir,
            "lib",
            "python{}.{}".format(sys.version_info.major, sys.version_info.minor),
            "site-packages",
        )

    return site if os.path.isdir(site) else None


# --- Script Execution ---


def execute_script(script_path, params, module_name, session, security_rules):
    """Load and execute a Python script with bitcode bridge.

    Handles:
    - sys.path save/restore (prevents pollution across executions)
    - Per-module venv site-packages resolution
    - Module cache clearing (hot reload)
    - Signature detection (1 param legacy vs 2 param new style)
    - Async script detection (inspect.iscoroutinefunction)
    """
    bitcode = BitcodeContext(session)

    if not os.path.exists(script_path):
        raise FileNotFoundError("script not found: {}".format(script_path))

    # Determine module directory
    module_dir = (
        os.path.abspath(os.path.join("modules", module_name))
        if module_name
        else os.path.dirname(os.path.abspath(script_path))
    )

    # Save sys.path to restore after execution (prevent pollution)
    original_path = sys.path.copy()

    try:
        # Resolve venv site-packages (cross-platform)
        venv_site = get_venv_site_packages(module_dir)

        # Prepend module paths to sys.path (module-level packages first)
        if venv_site and venv_site not in sys.path:
            sys.path.insert(0, venv_site)
        if os.path.isdir(module_dir) and module_dir not in sys.path:
            sys.path.insert(0, module_dir)

        import types
        with open(script_path, "r") as f:
            source = f.read()
        code = compile(source, script_path, "exec")
        mod = types.ModuleType("bitcode_script")
        mod.__file__ = script_path
        exec(code, mod.__dict__)

        # Call execute function
        if hasattr(mod, "execute"):
            func = mod.execute
            sig = inspect.signature(func)
            param_count = len(sig.parameters)

            # Detect async scripts
            if inspect.iscoroutinefunction(func):
                if param_count >= 2:
                    return asyncio.run(func(bitcode, params))
                else:
                    return asyncio.run(func(params))

            # Sync scripts - detect signature
            if param_count >= 2:
                return func(bitcode, params)  # new style
            else:
                return func(params)  # legacy style

        elif hasattr(mod, "main"):
            return mod.main(params)  # alternative legacy
        else:
            return {"executed": True, "script": script_path}

    finally:
        # Restore sys.path to prevent pollution across executions
        sys.path = original_path


# --- Message Loop (Main Thread + Reader Thread) ---


def message_router():
    """Single reader thread that reads ALL messages from stdin and routes them.

    This solves the race condition: only ONE thread reads stdin.
    Messages are routed by type:
    - "execute" -> put in execute_queue for main thread
    - "bridge_response" -> route to pending bridge_call() via threading.Event

    This is the same pattern as Node.js readline - one reader, multiple consumers.
    """
    for line in sys.stdin:
        line = line.strip()
        if not line:
            continue

        try:
            message = json.loads(line)
        except json.JSONDecodeError:
            continue

        msg_type = message.get("type")

        if msg_type == "bridge_response":
            req_id = message.get("id")
            with _pending_lock:
                entry = _pending_requests.get(req_id)
            if entry:
                if "error" in message and message["error"]:
                    entry["error"] = message["error"]
                else:
                    entry["result"] = message.get("result")
                entry["event"].set()

        elif msg_type == "execute":
            _execute_queue.put(message)

    # stdin closed (Go closed pipe) — unblock any pending bridge calls, then signal main thread
    with _pending_lock:
        for entry in _pending_requests.values():
            entry["error"] = {"code": "PROCESS_SHUTDOWN", "message": "process shutting down"}
            entry["event"].set()
    _execute_queue.put(_SHUTDOWN_SENTINEL)


def main():
    """Main entry point.

    Architecture (single process, pool model):
    - Router thread: reads ALL stdin, routes messages by type
    - Main thread: picks execute requests from queue, runs scripts
    - bridge_call(): blocks on threading.Event, unblocked by router thread

    Flow:
    1. Router thread reads stdin, routes "execute" to queue
    2. Main thread picks from queue, runs script
    3. Script calls bridge_call() -> sends request via stdout
    4. Router thread receives bridge_response -> unblocks bridge_call()
    5. Script completes -> main thread sends result -> picks next from queue
    """
    sys.stderr.write("[plugin:python] ready\n")
    sys.stderr.flush()

    # Start router thread (single reader for stdin)
    router = threading.Thread(target=message_router, daemon=True)
    router.start()

    while True:
        message = _execute_queue.get()
        if message is _SHUTDOWN_SENTINEL:
            break
        exec_id = message.get("id")
        params = message.get("params", {})

        # Handle params as raw dict or JSON-encoded
        if isinstance(params, str):
            try:
                params = json.loads(params)
            except json.JSONDecodeError:
                params = {}

        try:
            result = execute_script(
                params.get("script", ""),
                params.get("params", {}),
                params.get("module", ""),
                params.get("session", {}),
                params.get("securityRules", {}),
            )
            send_to_go({
                "type": "execute_complete",
                "id": exec_id,
                "result": serialize_for_json(result),
            })
        except BridgeError as e:
            send_to_go({
                "type": "execute_error",
                "id": exec_id,
                "error": {
                    "code": e.code,
                    "message": str(e),
                    "details": e.details,
                },
            })
        except Exception as e:
            send_to_go({
                "type": "execute_error",
                "id": exec_id,
                "error": {
                    "code": "SCRIPT_ERROR",
                    "message": str(e),
                    "stack": traceback.format_exc(),
                },
            })


if __name__ == "__main__":
    main()
