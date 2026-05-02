# Plugins

Plugins let you write custom logic in TypeScript, JavaScript, Python, or Go when JSON processes aren't enough. Scripts call `bitcode.*` bridge methods to interact with the engine (database, email, cache, etc.).

## Architecture

Runtime engines are extracted to `packages/go-json-runtimes/` (separate go.mod). BitCode imports this package and provides a `VMAdapter` that converts `*bridge.Context` → `map[string]any` for the decoupled runtimes.

```
Engine (Go)
├── Embedded runtimes (in-process): goja, quickjs, yaegi
│   └── VMAdapter → go-json-runtimes VM (InjectBridge(map[string]any))
└── External runtimes (child process): Node.js, Python
    └── JSON-RPC over stdin/stdout → Process Pool
```

## How It Works

1. Engine spawns pools of Node.js and Python processes at startup (or uses embedded goja/quickjs/yaegi in-process)
2. Script execution request sent to an available process (routed by file extension)
3. Script calls `bitcode.*` methods → JSON-RPC request sent to Go (external) or direct Go function call (embedded)
4. Go executes the bridge method (DB query, HTTP call, etc.) and returns result
5. Script continues with real data, may call more bridge methods
6. Script finishes, final result returned to engine

go-json programs can also call external scripts via `script:` imports — see `packages/go-json/docs/language-reference.md`.

## Script Example

```typescript
export default {
  async execute(bitcode, params) {
    const lead = params.input;

    // Real database operations via bridge
    await bitcode.model("activity").create({
      lead_id: lead.id,
      type: "task",
      summary: "Send welcome package",
    });

    // Send actual email
    await bitcode.email.send({
      to: "manager@company.com",
      subject: "Deal Won: " + lead.name,
      body: "<h1>Revenue: $" + lead.expected_revenue + "</h1>",
    });

    bitcode.log("info", "Deal won processed", { leadId: lead.id });
    return { success: true };
  },
};
```

## Using in Process

```json
{
  "type": "script",
  "runtime": "node",
  "script": "scripts/on_deal_won.ts"
}
```

The `runtime` field is optional — `.ts` files default to `"node"`, `.js` files default to `"javascript"` (embedded goja). Set `"runtime": "node"` explicitly for `.js` files that need npm packages.

## Bridge API (`bitcode.*`)

| Namespace | Methods |
|-----------|---------|
| `bitcode.model(name)` | `search`, `get`, `create`, `write`, `delete`, `count`, `sum`, `upsert`, `createMany`, `writeMany`, `deleteMany`, `upsertMany`, `addRelation`, `removeRelation`, `setRelation`, `loadRelation`, `sudo()` |
| `bitcode.db` | `query(sql, ...args)`, `execute(sql, ...args)` |
| `bitcode.http` | `get`, `post`, `put`, `patch`, `delete` |
| `bitcode.cache` | `get`, `set`, `del` |
| `bitcode.fs` | `read`, `write`, `exists`, `list`, `mkdir`, `remove` |
| `bitcode.email` | `send(opts)` |
| `bitcode.notify` | `send(opts)`, `broadcast(channel, data)` |
| `bitcode.storage` | `upload`, `url`, `download`, `delete` |
| `bitcode.security` | `permissions`, `hasGroup`, `groups` |
| `bitcode.audit` | `log(opts)` |
| `bitcode.crypto` | `encrypt`, `decrypt`, `hash`, `verify` |
| `bitcode.execution` | `search`, `get`, `current`, `retry`, `cancel` |
| `bitcode.env(key)` | Read environment variable (security-filtered) |
| `bitcode.config(key)` | Read config value |
| `bitcode.session` | `userId`, `username`, `tenantId`, `groups`, `locale` (injected, no RPC) |
| `bitcode.log(level, msg, data)` | Log to engine logger |
| `bitcode.emit(event, data)` | Emit domain event |
| `bitcode.call(process, input)` | Call another process |
| `bitcode.exec(cmd, args, opts)` | Execute system command |
| `bitcode.t(key)` | Translate string |
| `bitcode.tx(fn)` | Run function in database transaction |

## Parameters Available to Scripts

| Property | Description |
|----------|-------------|
| `params.input` | Process input data |
| `params.variables` | Process variables |
| `params.result` | Previous step result |
| `params.user_id` | Current user ID |

## Per-Module npm Dependencies

Each module can have its own `package.json` and `node_modules/`:

```
modules/crm/
├── package.json          ← npm dependencies for this module
├── node_modules/         ← isolated, only for this module
└── scripts/
    └── crawl_leads.js    ← can require('axios')
```

`require()` resolves from the module directory first, then falls back to project-level.

## TypeScript Support

`.ts` files are transpiled on-the-fly via esbuild (<10ms). No build step needed.

## Python Scripts

`.py` files use the same bidirectional JSON-RPC protocol. Python bridge uses snake_case (Pythonic):

```python
def execute(bitcode, params):
    leads = bitcode.model("lead").search({
        "domain": [["status", "=", "new"]],
    })
    for lead in leads:
        bitcode.model("lead").write(lead["id"], {"score": 85})
    bitcode.log("info", "Scored leads", {"count": len(leads)})
    return {"processed": len(leads)}
```

Using in process:

```json
{
  "type": "script",
  "runtime": "python",
  "script": "scripts/analyze_pipeline.py"
}
```

The `runtime` field is optional — `.py` files default to `"python"`.

### Per-Module pip Dependencies

Each module can have its own `requirements.txt` and `.venv/`:

```
modules/analytics/
├── requirements.txt      ← pip dependencies
├── .venv/                ← virtual environment (created by CLI)
└── scripts/
    └── analyze_data.py   ← can import pandas, numpy, etc.
```

```bash
bitcode module install-deps analytics    # creates .venv + pip install -r requirements.txt
bitcode module recreate-venv analytics   # delete + recreate .venv from scratch
bitcode module freeze analytics          # pip freeze > requirements.txt
```

### Backward Compatibility (Python)

Old-style scripts (`def execute(params)`) still work — the runtime auto-detects the signature.

### Async Python Scripts

Scripts can use `asyncio` for concurrent bridge calls:

```python
import asyncio

async def execute(bitcode, params):
    tasks = [
        asyncio.to_thread(bitcode.http.get, "https://api1.com"),
        asyncio.to_thread(bitcode.http.get, "https://api2.com"),
    ]
    results = await asyncio.gather(*tasks)
    return {"results": results}
```

## Backward Compatibility

The old `definePlugin` pattern still works:

```typescript
import { definePlugin } from '@bitcode/sdk';
export default definePlugin({
  async execute(ctx, params) { /* ctx === bitcode */ }
});
```

`runtime: "typescript"` is mapped to `"node"` internally.

## Transactions

Wrap multiple operations in a database transaction:

```typescript
export default {
  async execute(bitcode, params) {
    await bitcode.tx(async (tx) => {
      await tx.model("order").create({ total: 100 });
      await tx.model("inventory").write(itemId, { stock: newStock });
    });
  },
};
```

All operations inside `tx()` use the same database transaction. On error, everything is rolled back automatically. Transaction timeout: 30 seconds.

## CLI Commands

```bash
# Node.js (npm)
bitcode module install-deps crm          # npm install in modules/crm/
bitcode module install-deps --all        # all modules (npm + pip)
bitcode module add-package crm axios     # npm install axios in modules/crm/
bitcode module remove-package crm axios  # npm uninstall axios from modules/crm/

# Python (pip + venv)
bitcode module install-deps analytics    # creates .venv + pip install -r requirements.txt
bitcode module recreate-venv analytics   # delete + recreate .venv from scratch
bitcode module freeze analytics          # pip freeze > requirements.txt
```

## Configuration

```yaml
runtime:
  node:
    enabled: "auto"        # "auto" | "true" | "false"
    command: ""            # empty = auto-detect (bun > node)
  python:
    enabled: "auto"        # "auto" | "true" | "false"
    command: "python3"     # python binary (tries python3, then python)
    min_version: "3.10.0"  # minimum Python version
  worker:
    pool_size: 4           # processes for fast scripts (shared by all runtimes)
    max_executions: 1000   # recycle after N executions
  background:
    pool_size: 2           # processes for long-running scripts
    max_executions: 100
```

## Plugin Manager

The plugin manager handles:
- Dual process pool: worker (fast, 4 procs) + background (long-running, 2 procs)
- Version validation: Node.js 20+, Bun 1.2.15+ (auto-detected)
- Crash recovery with exponential backoff (up to 5 restart attempts)
- Process recycling after N executions (configurable)
- Real transactions over JSON-RPC with 30s auto-rollback timeout
- Bidirectional JSON-RPC for real bridge method execution
- Auto-detection of Bun vs Node.js (Bun preferred if available)
- Graceful degradation when Node.js or Python is not installed
- Python 3.10+ runtime with same bidirectional protocol
- Per-module venv isolation for Python (requirements.txt + .venv/)
- Cross-platform venv paths (Windows Scripts/ vs Unix bin/)
- Python script signature auto-detection (legacy 1-param vs new 2-param)
- Async Python script support via asyncio
- Binary data serialization/deserialization (bytes ↔ base64)
- Transaction-scoped context for Python (TxBitcodeContext)
- Graceful shutdown via stdin EOF sentinel
