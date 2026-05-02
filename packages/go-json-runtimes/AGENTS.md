# AGENTS.md — go-json-runtimes

## Overview

Script runtime engines for go-json. Separate `go.mod` — independently importable. Implements go-json's `ScriptRuntime` interface with goja, quickjs, yaegi, WASM (wazero), Node.js, Python, and native plugin runtimes.

## Package Structure

```
packages/go-json-runtimes/
├── go.mod              Module definition (separate from go-json)
├── runtime.go          VM, EmbeddedRuntime, ExternalRuntime interfaces
├── runtimeset.go       RuntimeSet aggregator + ScriptRuntime adapters
├── config.go           Configuration types (GojaConfig, NodeConfig, etc.)
├── options.go          Functional options (WithGoja, WithNode, etc.)
├── loader.go           Script file loader
├── helpers.go          Type conversion (ToInt, ToStringSlice, etc.)
├── goja/               Goja JavaScript runtime (pure Go, ES5.1+)
├── quickjs/            QuickJS JavaScript runtime (ES2023 via wazero)
├── yaegi/              Yaegi Go interpreter runtime
├── wasm/               WASM runtime (wazero v1.9.0, pure Go, zero CGO)
├── native/             Native plugin runtime (Go plugin package, Linux/macOS)
├── node/               Node.js child process runtime
├── python/             Python child process runtime
├── pool/               Process pool manager (worker + background)
└── docs/               Runtime documentation (wasm.md, native.md)
```

## Key Design Decisions

1. **Bridge is `map[string]any`** — runtimes NEVER import BitCode types
2. **Pool manager is optional** — standalone users can use without pool
3. **Each runtime is opt-in** — `WithGoja()`, `WithNode()`, `WithWasm()`, `WithNative()`, etc.
4. **Timeout is NOT pool's job** — enforced by caller via `context.Context`
5. **WASM uses instance pooling** — `PoolSize > 0` enables chan-based instance reuse (different from process pooling)
6. **Native plugins are NOT default** — security risk, must be explicitly enabled

## Runtime Versions

| Runtime | Package | Version |
|---------|---------|---------|
| goja | github.com/dop251/goja | v0.0.0-20260311... |
| yaegi | github.com/traefik/yaegi | v0.16.1 |
| quickjs | github.com/fastschema/qjs | v0.0.6 |
| wazero | github.com/tetratelabs/wazero | v1.9.0 |

## Testing

```bash
cd packages/go-json-runtimes
go test ./... -v          # All tests
go test ./goja/ -v        # Goja tests
go test ./quickjs/ -v     # QuickJS tests
go test ./yaegi/ -v       # Yaegi tests
go test ./wasm/ -v        # WASM tests
go test ./native/ -v      # Native plugin tests (Linux/macOS)
go test ./node/ -v        # Node.js tests
go test ./python/ -v      # Python tests
go test ./pool/ -v        # Pool tests
```

## Conventions

- Follow root `AGENTS.md` conventions
- All exported types need Go doc comments
- `go build ./...` and `go vet ./...` must pass
- Bridge is always `map[string]any` — never typed interfaces
