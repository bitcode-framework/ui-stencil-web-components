# go-json-runtimes

Script runtime engines for [go-json](../go-json/). Provides goja (JavaScript), yaegi (Go), quickjs (JavaScript ES2023), WASM (wazero), Node.js, and Python runtimes that implement go-json's `ScriptRuntime` interface.

## Installation

```bash
go get github.com/bitcode-framework/go-json-runtimes
```

## Quick Start

```go
import (
    gojson "github.com/bitcode-framework/go-json/runtime"
    "github.com/bitcode-framework/go-json-runtimes/goja"
    "github.com/bitcode-framework/go-json-runtimes/quickjs"
    "github.com/bitcode-framework/go-json-runtimes/yaegi"
)

rt := gojson.NewRuntime(
    gojson.WithScriptRuntime(goja.New()),      // .js
    gojson.WithScriptRuntime(quickjs.New()),    // .js (explicit)
    gojson.WithScriptRuntime(yaegi.New()),      // .go
)
defer rt.Close()
```

## Available Runtimes

| Runtime | Package | File Extensions | Type |
|---------|---------|----------------|------|
| goja | `goja/` | `.js` | Embedded (pure Go) |
| quickjs | `quickjs/` | `.js` | Embedded (via wazero) |
| yaegi | `yaegi/` | `.go` | Embedded (Go interpreter) |
| wazero (WASM) | `wasm/` | `.wasm` | Embedded (pure Go, zero CGO) |
| Node.js | `node/` | `.ts`, `.mjs` | External (child process) |
| Python | `python/` | `.py`, `.pyw` | External (child process) |
| Native | `native/` | `.so`, `.dylib` | Native (Go plugin, Linux/macOS only) |

## Configuration

```go
// Embedded runtimes — always available, no system dependencies
runtimes.WithGoja()
runtimes.WithQuickJS()
runtimes.WithYaegi(runtimes.YaegiConfig{
    BridgesDir: "bridges/",
})

// WASM runtime — embedded, always available
runtimes.WithWasm(runtimes.WasmRuntimeConfig{
    Enabled:      true,
    MaxMemoryMB:  64,
    MaxExecTime:  30 * time.Second,
    CompileCache: true,
})

// External runtimes — require system installation
runtimes.WithNode(runtimes.NodeConfig{
    Enabled:    "auto",
    Command:    "node",
    MinVersion: "20.0",
})
runtimes.WithPython(runtimes.PythonConfig{
    Enabled:    "auto",
    Command:    "python3",
    MinVersion: "3.10.0",
})

// Native plugins — Linux/macOS only, NOT enabled by default (security)
runtimes.WithNative(runtimes.NativeRuntimeConfig{
    Enabled:     true,
    AllowedDirs: []string{"plugins/"},
})
```

## Pool Manager

For production use with Node.js/Python, configure process pools:

```go
runtimes.WithNode(runtimes.NodeConfig{
    Pool: &runtimes.DualPoolRef{
        Worker: runtimes.PoolWithTimeoutConfig{
            Pool: pool.Config{Size: 4, MaxExecutions: 1000, CrashRecovery: true},
        },
        Background: runtimes.PoolWithTimeoutConfig{
            Pool: pool.Config{Size: 2, MaxExecutions: 100, CrashRecovery: true},
        },
    },
})
```

## For BitCode Users

BitCode imports this package automatically. Runtime configuration is read from `bitcode.yaml` and passed to go-json-runtimes via functional options. No manual setup needed.

## Bridge Contract

All runtimes receive the bridge as `map[string]any`. Runtimes never import BitCode types. The bridge is built by the host application (BitCode or standalone).

## Architecture

```
go-json (core)          ← zero external deps, defines ScriptRuntime interface
     ↑
go-json-runtimes        ← imports go-json (for interface), goja, wazero, etc.
     ↑
BitCode engine          ← imports go-json-runtimes, builds bridge map
```
