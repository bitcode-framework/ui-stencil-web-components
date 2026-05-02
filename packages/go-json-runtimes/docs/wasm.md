# WASM Plugin Authoring Guide

## Overview

WASM plugins run WebAssembly modules inside a sandboxed wazero runtime. They execute in-process with zero CGO dependencies, providing near-native performance with strong isolation guarantees.

**When to use WASM plugins:**

- Cross-platform plugins (Linux, macOS, Windows — all supported)
- Untrusted or third-party code that needs sandboxing
- Performance-sensitive logic that can't tolerate child-process overhead
- Plugins authored in Rust, Go (TinyGo), C/C++, or AssemblyScript

**When NOT to use WASM:**

- You need full filesystem/network access (use Node.js/Python instead)
- Your plugin requires Go standard library (use native or yaegi instead)
- Simple scripts that don't need sandboxing (use goja/quickjs instead)

## Quick Start

### 1. Write a plugin (Rust)

```rust
// src/lib.rs
use std::alloc::{alloc, dealloc, Layout};
use std::slice;

// Required: memory allocator for the host
#[no_mangle]
pub extern "C" fn malloc(size: u32) -> *mut u8 {
    let layout = Layout::from_size_align(size as usize, 1).unwrap();
    unsafe { alloc(layout) }
}

// Required: memory deallocator
#[no_mangle]
pub extern "C" fn free(ptr: *mut u8, size: u32) {
    let layout = Layout::from_size_align(size as usize, 1).unwrap();
    unsafe { dealloc(ptr, layout) }
}

// Your plugin function
#[no_mangle]
pub extern "C" fn transform(ptr: u32, len: u32) -> u64 {
    // Read JSON input from memory
    let input = unsafe {
        let slice = slice::from_raw_parts(ptr as *const u8, len as usize);
        String::from_utf8_lossy(slice).to_string()
    };

    // Parse input, do work
    let args: serde_json::Value = serde_json::from_str(&input).unwrap_or_default();
    let result = serde_json::json!({
        "transformed": true,
        "input": args
    });

    // Write result to memory and return packed ptr|len
    let output = serde_json::to_vec(&result).unwrap();
    let out_len = output.len() as u32;
    let out_ptr = malloc(out_len);
    unsafe {
        std::ptr::copy_nonoverlapping(output.as_ptr(), out_ptr, out_len as usize);
    }

    // Pack pointer (high 32 bits) and length (low 32 bits) into i64
    ((out_ptr as u64) << 32) | (out_len as u64)
}
```

### 2. Build to WASM

```bash
cargo build --target wasm32-unknown-unknown --release
cp target/wasm32-unknown-unknown/release/my_plugin.wasm plugins/
```

### 3. Call from go-json

```jsonc
{
  "steps": [
    {
      "type": "script",
      "script": "plugins/my_plugin.wasm",
      "function": "transform",
      "args": { "name": "hello" }
    }
  ]
}
```

### 4. Configure the runtime (Go)

```go
import "github.com/bitcode-framework/go-json-runtimes"

runtimes.WithWasm(runtimes.WasmRuntimeConfig{
    Enabled:      true,
    MaxMemoryMB:  64,
    MaxExecTime:  30 * time.Second,
    CompileCache: true,
    WASIEnabled:  false,
})
```

## Plugin Protocol

Every WASM plugin must export these functions:

### Required Exports

| Export | Signature | Purpose |
|--------|-----------|---------|
| `memory` | (memory) | Linear memory, at least 1 page (64KB) |
| `malloc` | `(i32) -> i32` | Allocate `n` bytes, return pointer |
| Your function | `(i32, i32) -> i64` | Receive `(ptr, len)`, return packed result |

### Optional Exports

| Export | Signature | Purpose |
|--------|-----------|---------|
| `free` | `(i32, i32) -> ()` | Free `(ptr, len)` — called after reading result |

### Calling Convention

1. Host serializes `params` to JSON bytes
2. Host calls `malloc(len)` → gets `ptr`
3. Host writes JSON bytes to `memory[ptr..ptr+len]`
4. Host calls `your_function(ptr, len)` → gets packed `i64`
5. Host unpacks result: `result_ptr = i64 >> 32`, `result_len = i64 & 0xFFFFFFFF`
6. Host reads `memory[result_ptr..result_ptr+result_len]` as JSON
7. Host calls `free(result_ptr, result_len)` if exported

### Return Value Encoding

The return value is a single `i64` with pointer and length packed:

```
┌─────────────────────────────────────────────────────────────────┐
│  bits 63..32: pointer (u32)  │  bits 31..0: length (u32)       │
└─────────────────────────────────────────────────────────────────┘
```

Return `0` (i64) to indicate no result (nil).

### Host Functions (imports from "env")

Your WASM module can import these from the `"env"` module:

| Import | Signature | Purpose |
|--------|-----------|---------|
| `bridge_call` | `(fn_ptr, fn_len, args_ptr, args_len) -> i64` | Call a bridge function by dotted path |
| `log` | `(level, msg_ptr, msg_len) -> ()` | Log a message (level: 0=debug, 1=info, 2=warn, 3=error) |

`bridge_call` returns a packed `i64` (same ptr\|len format) containing JSON `{"value": ...}` or `{"error": "..."}`.

## Writing Plugins in Rust (Recommended)

Rust produces the smallest, fastest WASM binaries with excellent tooling.

### Setup

```bash
rustup target add wasm32-unknown-unknown
cargo init --lib my_plugin
```

**Cargo.toml:**

```toml
[package]
name = "my_plugin"
version = "0.1.0"
edition = "2021"

[lib]
crate-type = ["cdylib"]

[dependencies]
serde_json = "1"

[profile.release]
opt-level = "z"    # Optimize for size
lto = true
strip = true
```

### Template

```rust
use std::alloc::{alloc, dealloc, Layout};
use std::slice;

#[no_mangle]
pub extern "C" fn malloc(size: u32) -> *mut u8 {
    let layout = Layout::from_size_align(size as usize, 1).unwrap();
    unsafe { alloc(layout) }
}

#[no_mangle]
pub extern "C" fn free(ptr: *mut u8, size: u32) {
    let layout = Layout::from_size_align(size as usize, 1).unwrap();
    unsafe { dealloc(ptr, layout) }
}

/// Helper: read JSON input from WASM memory
fn read_input(ptr: u32, len: u32) -> serde_json::Value {
    let bytes = unsafe { slice::from_raw_parts(ptr as *const u8, len as usize) };
    serde_json::from_slice(bytes).unwrap_or_default()
}

/// Helper: write JSON output and return packed ptr|len
fn write_output(value: &serde_json::Value) -> u64 {
    let bytes = serde_json::to_vec(value).unwrap();
    let out_len = bytes.len() as u32;
    let out_ptr = malloc(out_len);
    unsafe {
        std::ptr::copy_nonoverlapping(bytes.as_ptr(), out_ptr, out_len as usize);
    }
    ((out_ptr as u64) << 32) | (out_len as u64)
}

#[no_mangle]
pub extern "C" fn my_function(ptr: u32, len: u32) -> u64 {
    let input = read_input(ptr, len);
    let result = serde_json::json!({ "ok": true, "echo": input });
    write_output(&result)
}
```

### Build

```bash
cargo build --target wasm32-unknown-unknown --release
# Output: target/wasm32-unknown-unknown/release/my_plugin.wasm

# Optional: strip further with wasm-opt
wasm-opt -Oz -o my_plugin.wasm target/wasm32-unknown-unknown/release/my_plugin.wasm
```

## Writing Plugins in Go (TinyGo)

TinyGo compiles Go to WASM. Standard Go compiler does NOT support `wasm32-unknown-unknown` (only `GOOS=js`).

### Setup

```bash
# Install TinyGo: https://tinygo.org/getting-started/install/
tinygo version  # requires 0.30+
```

### Template

```go
package main

import "unsafe"

// Required exports
//export malloc
func malloc(size uint32) *byte {
    buf := make([]byte, size)
    return &buf[0]
}

//export free
func free(ptr *byte, size uint32) {
    // GC handles this in TinyGo
}

//export transform
func transform(ptr *byte, size uint32) uint64 {
    // Read input
    input := unsafe.Slice(ptr, size)

    // Process (simple echo for demo)
    output := make([]byte, len(input))
    copy(output, input)

    // Write output
    outPtr := malloc(uint32(len(output)))
    outSlice := unsafe.Slice(outPtr, len(output))
    copy(outSlice, output)

    // Pack ptr|len
    return (uint64(uintptr(unsafe.Pointer(outPtr))) << 32) | uint64(len(output))
}

func main() {}
```

### Build

```bash
tinygo build -o my_plugin.wasm -target wasm -no-debug ./
```

> **Note:** TinyGo WASM binaries are larger than Rust (~100KB+ vs ~10KB) due to the Go runtime. Use Rust for size-sensitive deployments.

## Writing Plugins in AssemblyScript

AssemblyScript is TypeScript-like syntax that compiles to WASM.

### Setup

```bash
npm init -y
npm install assemblyscript
npx asinit .
```

### Template (assembly/index.ts)

```typescript
// Memory is managed by AssemblyScript's runtime
export function malloc(size: u32): usize {
  return heap.alloc(size) as usize;
}

export function free(ptr: usize, size: u32): void {
  heap.free(ptr);
}

export function greet(ptr: usize, len: u32): u64 {
  // Read input JSON
  let input = String.UTF8.decodeUnsafe(ptr, len);

  // Build output
  let output = '{"greeting":"Hello from AssemblyScript!","input":' + input + '}';
  let encoded = String.UTF8.encode(output);
  let outLen = encoded.byteLength as u32;
  let outPtr = malloc(outLen);

  // Copy to output buffer
  memory.copy(outPtr, changetype<usize>(encoded), outLen);

  // Pack ptr|len into i64
  return (u64(outPtr) << 32) | u64(outLen);
}
```

### Build

```bash
npx asc assembly/index.ts -o my_plugin.wasm --optimize --exportRuntime
```

## Configuration

```go
type WasmRuntimeConfig struct {
    Enabled      bool          // Enable WASM runtime
    MaxMemoryMB  int           // Memory limit (default: 64MB)
    MaxExecTime  time.Duration // Execution timeout (default: 30s)
    CompileCache bool          // Cache compiled modules (default: true)
    WASIEnabled  bool          // Enable WASI imports (default: false)
}
```

### BitCode YAML

```yaml
runtimes:
  wasm:
    enabled: true
    max_memory_mb: 64
    max_exec_time: 30s
    compile_cache: true
    wasi: false
```

| Field | Default | Description |
|-------|---------|-------------|
| `MaxMemoryMB` | `64` | Maximum linear memory in MB. Each WASM page = 64KB. Set to 0 for unlimited. |
| `MaxExecTime` | `30s` | Maximum execution time per call. Enforced via `context.WithTimeout`. |
| `CompileCache` | `true` | Cache compiled modules in memory. Eliminates re-compilation on repeated calls. |
| `WASIEnabled` | `false` | Enable WASI (WebAssembly System Interface). Required for plugins that need stdio/filesystem. |

## Runtime Version

| Component | Version | Notes |
|-----------|---------|-------|
| wazero | v1.9.0 | Pure Go, zero CGO, no system dependencies |
| WASI | snapshot_preview1 | Optional, disabled by default |
| Go | 1.24+ | Required for go-json-runtimes |

## Security

### Sandbox Guarantees

WASM plugins run in a **fully sandboxed** environment:

- **No filesystem access** — unless WASI is enabled with explicit permissions
- **No network access** — plugins cannot make HTTP calls or open sockets
- **No system calls** — only imported host functions are available
- **Memory isolation** — each module instance has its own linear memory
- **Deterministic execution** — no access to time, random, or OS state (without WASI)

### WASI Access Control

When `WASIEnabled: true`, the runtime instantiates WASI snapshot_preview1. By default this provides:

- `fd_write` to stdout/stderr (for debugging)
- No filesystem mounts (plugins cannot read/write files)
- No environment variables exposed
- No command-line arguments

> **Warning:** Enabling WASI relaxes the sandbox. Only enable it for trusted plugins that require stdio.

### Memory Limits

- `MaxMemoryMB` caps the linear memory a module can grow to
- Memory is measured in pages (1 page = 64KB, so 64MB = 1024 pages)
- Exceeding the limit causes a trap (runtime error), not OOM

### Execution Timeout

- `MaxExecTime` is enforced via Go's `context.WithTimeout`
- If the caller already provides a deadline, the runtime respects the shorter one
- Timeout causes immediate termination — no cleanup code runs in the module

## Performance

### Compile Caching

With `CompileCache: true` (default), modules are compiled once and cached in memory:

```
First call:  read .wasm → compile → instantiate → execute  (~5-50ms compile)
Second call: cache hit  →           instantiate → execute  (~0.1ms)
```

The cache key is the absolute file path. Restart the process to invalidate.

### JIT Compilation

wazero uses a **compiler backend** on supported platforms:

| Platform | Backend | Performance |
|----------|---------|-------------|
| `amd64` (x86_64) | JIT compiler | Near-native speed |
| `arm64` (Apple Silicon, ARM servers) | JIT compiler | Near-native speed |
| Other architectures | Interpreter | ~10x slower |

### Tips

- Keep WASM binaries small — smaller modules compile faster
- Use `wasm-opt -Oz` (from Binaryen) to shrink Rust/C output
- Avoid allocating large buffers inside WASM — pass data via shared memory
- Reuse the runtime instance across calls (it's thread-safe)

## Troubleshooting

### "wasm module missing required 'malloc' export"

Your module doesn't export a `malloc` function. Every plugin must export:
```
malloc: (i32) -> i32
```

### "wasm function 'X' not exported"

The function name you're calling doesn't exist in the module's exports. Check:
```bash
wasm-objdump -x my_plugin.wasm | grep "Export"
# or
wasm2wat my_plugin.wasm | grep export
```

### "wasm memory write failed"

The `malloc` returned a pointer outside valid memory bounds. Ensure your allocator returns valid pointers within the module's linear memory.

### "failed to compile WASM module"

The `.wasm` file is corrupted or uses unsupported features. Verify:
```bash
wasm-validate my_plugin.wasm  # from wabt toolkit
```

### "wasm call 'X': context deadline exceeded"

Execution exceeded `MaxExecTime`. Either:
- Increase the timeout in config
- Optimize the plugin (infinite loops will always timeout)
- Check for accidental infinite recursion

### Module too large / OOM

If compilation fails with memory errors:
- Reduce module size with `wasm-opt`
- Strip debug info: `wasm-strip my_plugin.wasm`
- Increase host process memory

### WASI functions not found

If your module imports WASI functions but `WASIEnabled` is `false`:
```
error: module "" function "fd_write" not found
```
Set `WASIEnabled: true` in config, or recompile without WASI (use `wasm32-unknown-unknown` target in Rust).
