# Implement Phase 4.5k: WASM + Native Plugins

## Prerequisites

**Phase 4.5j MUST be complete before starting this phase.** This phase depends on:
- `ScriptRuntime` interface (defined in Phase 4.5j)
- `packages/go-json-runtimes/` package (created in Phase 4.5j)
- `script:` import infrastructure in go-json parser/VM (Phase 4.5j)
- CLI default runtime registration pattern (Phase 4.5j)

If Phase 4.5j is not complete, **STOP — do not proceed**.

## References

| Doc | Purpose |
|-----|---------|
| `docs/plans/2026-07-15-phase-4.5k-wasm-native-plugins.md` | **PRIMARY** — Full design doc with WASM protocol, native plugin design, code, checklists |
| `docs/plans/instructions-implement-wasm-native-plugins.md` | Implementation instruction — task-by-task guide |
| `docs/plans/2026-07-14-runtime-engine-redesign-master.md` | Master doc — phase table, dependency graph |
| `docs/plans/2026-07-15-phase-4.5j-script-plugins-runtime-extraction.md` | Phase 4.5j design — ScriptRuntime interface that WASM/native implement |

Read the design doc FIRST (all 1456 lines). It contains everything needed:
- WASM runtime via wazero — pure Go, zero CGO (Section 4)
- JSON-over-linear-memory protocol — malloc/free/packed i64 return (Section 4.2-4.3)
- Host functions — `bridge_call` for WASM→host bridge access (Section 4.4)
- WASM module lifecycle — compile cache, instantiation, timeout (Section 4.5)
- WASM instance pooling + bitcode.yaml mapping (Section 4.7-4.8)
- Native plugin via Go `plugin` package — Linux/macOS only (Section 5)
- Native plugin protocol — Manifest() + `func(map[string]any) (any, error)` (Section 5.2-5.3)
- Codegen strategy — WASM portable, native stubs (Section 7)
- CLI default: WASM enabled, native NOT enabled (Section Phase 5 checklist)
- Documentation outlines — wasm.md, native.md, script-runtimes.md update (Section 15)
- Full implementation checklist — 6 phases, ~70 items (Section 11)

## Target

```
packages/go-json/lang/                    ← wasm: and plugin: import types in parser
packages/go-json/runtime/                 ← import resolution for wasm:/plugin:
packages/go-json-runtimes/wasm/           ← NEW — WasmRuntime (wazero)
packages/go-json-runtimes/native/         ← NEW — NativeRuntime (Go plugin package)
packages/go-json-runtimes/options.go      ← WithWasm(), WithNative() options
packages/go-json/cmd/go-json/             ← register WASM as CLI default
packages/go-json/codegen/                 ← WASM/native codegen
```

## Verify

```bash
# After each phase:
cd packages/go-json && go build ./... && go vet ./... && go test ./... -v
cd packages/go-json-runtimes && go build ./... && go vet ./... && go test ./wasm/ -v && go test ./native/ -v
cd engine && go build ./... && go vet ./... && go test ./... -v
```

## Cara Kerja

1. Baca design doc lengkap (`2026-07-15-phase-4.5k-wasm-native-plugins.md`) + instruction file (`instructions-implement-wasm-native-plugins.md`)
2. Kerjakan task-by-task sesuai stream order di Section 11 (Phase 1 → 2 → 3 → 4 → 5 → 6)
3. Setiap stream selesai: review kritis dan jujur (edge cases, backward compat). Jika ada yang perlu diperbaiki dan disesuaikan, langsung fix (kecuali memang blocker). Ulang fase ini (review → need fix & not blocker → fix → review lagi, sampai tidak ada yang perlu di-fix)
4. Update doc-doc terkait (lihat Section 11 Phase 6 — split per scope 6a-6e), lalu commit

## Stream Order

### Stream 1: go-json Core — Parser Changes (Section 11 Phase 1)
- Add `wasm:` to `classifyImportPath()` in parser
- Add `plugin:` to `classifyImportPath()` in parser
- Handle `wasm:` imports in `Runtime.Execute()` — same pattern as `script:` (resolve by extension `.wasm`)
- Handle `plugin:` imports in `Runtime.Execute()` — resolve by extension `.so`/`.dylib`
- Same for `Runtime.ExecuteFunction()`
- Write parser tests
- **Verify**: all existing go-json tests pass, zero new dependencies

### Stream 2: WASM Runtime (Section 11 Phase 2) — BULK OF WORK
- Create `packages/go-json-runtimes/wasm/`
- Add `github.com/tetratelabs/wazero` to `packages/go-json-runtimes/go.mod`
- Implement `WasmRuntime` struct implementing `ScriptRuntime` interface
- Implement module compilation with caching (`getOrCompile`)
- Implement host functions: `bridge_call` (WASM calls host bridge functions)
- Implement JSON-over-linear-memory protocol:
  - Host → WASM: serialize args to JSON, call `malloc`, write to memory, call function with `(ptr, len)`
  - WASM → Host: function returns packed `i64` (`result_ptr << 32 | result_len`), host reads memory, calls `free`
- Implement timeout via `context.WithTimeout` (wazero respects `ctx`)
- Implement memory limit via `wazero.NewRuntimeConfig().WithMemoryLimitPages()`
- `Validate()` returns nil — wazero is embedded, always available
- Add `WithWasm()` option to go-json-runtimes
- Create test WASM modules (pre-compiled `.wasm` files in `testdata/wasm/`)
- Write all tests from checklist
- **Verify**: `go test ./wasm/ -v` passes

### Stream 3: Native Plugin Runtime (Section 11 Phase 3)
- Create `packages/go-json-runtimes/native/`
- Implement `NativeRuntime` struct implementing `ScriptRuntime` interface
- Platform guard: `runtime.GOOS == "windows"` → error
- Plugin loading via `plugin.Open()` with caching
- Manifest discovery: `Manifest() []string` required export
- Function signature: `func(map[string]any) (any, error)` — type-assert at load time
- Path validation: `AllowedDirs` whitelist
- Timeout via goroutine + `context.Context`
- Add `WithNative()` option to go-json-runtimes
- Build-tag tests for Linux/macOS only
- **Verify**: `go test ./native/ -v` passes (Linux/macOS)

### Stream 4: Codegen Support (Section 11 Phase 4)
- WASM codegen for Go target: wazero module loader
- WASM codegen for JavaScript target: `WebAssembly.instantiate()` API
- WASM codegen for Python target: wasmer/wasmtime import
- Native codegen for Go target: `plugin.Open()` + symbol lookup
- Native codegen for JS/Python targets: stub with clear error message
- Write codegen tests
- **Verify**: `go test ./codegen/ -v` passes

### Stream 5: CLI Default + BitCode Integration (Section 11 Phase 5 + Phase 5 checklist)
- Register WASM runtime as default in `packages/go-json/cmd/go-json/main.go` (alongside goja/yaegi/quickjs from Phase 4.5j)
- WASM is embedded (wazero = pure Go) → always available, no system dependency
- Native is NOT registered by default (security risk — must be explicitly enabled)
- BitCode integration: register WASM in engine startup
- BitCode integration: parse `wasm` and `native` config from `bitcode.yaml`
- **CRITICAL**: WASM uses same timeout enforcement as all runtimes — caller sets `context.WithTimeout`, wazero respects `ctx.Done()`
- **CRITICAL**: `MaxMemoryMB` and `HardMaxMemoryMB` from bitcode.yaml → wazero `WithMemoryLimitPages()`
- **Verify**: `go-json run program.json` with `wasm:./plugin.wasm` works out of the box
- **Verify**: ALL existing tests pass

### Stream 6: Documentation (Section 11 Phase 6)
- Follow Section 15 outlines EXACTLY for new docs
- Write `packages/go-json-runtimes/docs/wasm.md` — WASM plugin authoring guide (Section 15.1 outline)
- Write `packages/go-json-runtimes/docs/native.md` — native plugin guide (Section 15.2 outline)
- Update `packages/go-json/docs/script-runtimes.md` — add WASM + native sections (Section 15.3)
- Update all docs listed in Phase 6 checklist (6b/6c/6d)
- **CRITICAL**: `engine/docs/features/plugins.md` needs WASM + native sections, config examples
- Run verification items in 6e

## Aturan

- WAJIB berpikir kritis, detail, mateng, lengkap dan jujur
- Saya tidak ingin tech debt di phase ini, jadi pastikan tidak ada tech debt (sampaikan rekomendasinya setelah selesai semua pengerjaan)
- Jika ada blocker, jangan kerjakan dulu, konfirmasikan ke saya terlebih dahulu
- Go version: **1.24** (semua go.mod)
- go-json core MUST remain zero-dependency — wazero is in go-json-runtimes ONLY
- WASM default in CLI, native NOT default (security)
- All existing tests must pass after every stream

## Key Gotchas (Read Before Starting)

1. **wazero module naming** — use empty string `""` for anonymous modules (`wazero.NewModuleConfig().WithName("")`). Unique names required per instantiation.

2. **WASM memory is per-instance** — don't share memory references across instances. Each `Execute()` call gets its own instance (or pooled instance).

3. **Packed return value** — WASM function returns `i64`: `(result_ptr << 32) | result_len`. Unpack with bit shift:
   ```go
   resultPtr := uint32(packed >> 32)
   resultLen := uint32(packed & 0xFFFFFFFF)
   ```

4. **WASM module MUST export `malloc` and `free`** — without these, host cannot write data to WASM memory. Return clear error if missing: `"wasm module missing required 'malloc' export"`.

5. **Native plugins can't be unloaded** — Go's `plugin` package doesn't support unloading. Once loaded, stays in memory for process lifetime. Document this.

6. **Native plugins require SAME Go version** — plugin must be built with same Go version as host binary. Version mismatch = load failure. Document clearly.

7. **Test WASM modules** — pre-compile `.wasm` files and commit to repo. Use WAT (WebAssembly Text) format for simple inline tests if possible. Provide `Makefile` in `testdata/wasm/build/` for rebuilding.

8. **quickjs (`fastschema/qjs`) may use wazero internally** — potential version conflict with wazero added for WASM runtime. Check at implementation time. If conflict, pin compatible versions.

9. **`bridge_call` host function** — must resolve dotted paths (`"db.query"` → `bridge["db"].(map[string]any)["query"]`). Handle various function signatures via type switch. See Section 4.4 in design doc.

10. **WASM pool is different from process pool** — WASM instances are in-process (not child processes). Use instance pooling (`chan api.Module`), not process pooling. See Section 4.7-4.8.
