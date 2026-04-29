# Plugins

Plugins let you write custom logic in TypeScript or JavaScript when JSON processes aren't enough. Scripts call `bitcode.*` bridge methods to interact with the engine (database, email, cache, etc.) via bidirectional JSON-RPC.

## How It Works

```
Engine (Go) ◄──Bidirectional JSON-RPC over stdin/stdout──► Node.js Process Pool
```

1. Engine spawns a pool of Node.js processes at startup
2. Script execution request sent to an available process
3. Script calls `bitcode.*` methods → JSON-RPC request sent to Go
4. Go executes the bridge method (DB query, HTTP call, etc.) and returns result
5. Script continues with real data, may call more bridge methods
6. Script finishes, final result returned to engine

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

## Backward Compatibility

The old `definePlugin` pattern still works:

```typescript
import { definePlugin } from '@bitcode/sdk';
export default definePlugin({
  async execute(ctx, params) { /* ctx === bitcode */ }
});
```

`runtime: "typescript"` is mapped to `"node"` internally.

## Plugin Manager

The plugin manager handles:
- Process pool with configurable size (default: 4 worker, 2 background)
- Crash recovery with exponential backoff (up to 5 restart attempts)
- Process recycling after N executions (default: 1000)
- Bidirectional JSON-RPC for real bridge method execution
- Auto-detection of Bun vs Node.js (Bun preferred if available)
- Graceful degradation when Node.js is not installed
