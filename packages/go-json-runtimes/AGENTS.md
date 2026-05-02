# AGENTS.md — go-json-runtimes

## Overview

Script runtime engines for go-json. Separate `go.mod` — independently importable. Implements go-json's `ScriptRuntime` interface with goja, quickjs, yaegi, Node.js, and Python runtimes.

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
├── node/               Node.js child process runtime
├── python/             Python child process runtime
└── pool/               Process pool manager (worker + background)
```

## Key Design Decisions

1. **Bridge is `map[string]any`** — runtimes NEVER import BitCode types
2. **Pool manager is optional** — standalone users can use without pool
3. **Each runtime is opt-in** — `WithGoja()`, `WithNode()`, etc.
4. **Timeout is NOT pool's job** — enforced by caller via `context.Context`

## Testing

```bash
cd packages/go-json-runtimes
go test ./... -v          # All tests
go test ./goja/ -v        # Goja tests
go test ./quickjs/ -v     # QuickJS tests
go test ./yaegi/ -v       # Yaegi tests
go test ./node/ -v        # Node.js tests
go test ./python/ -v      # Python tests
go test ./pool/ -v        # Pool tests
```

## Conventions

- Follow root `AGENTS.md` conventions
- All exported types need Go doc comments
- `go build ./...` and `go vet ./...` must pass
- Bridge is always `map[string]any` — never typed interfaces
