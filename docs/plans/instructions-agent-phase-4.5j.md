# Implement Phase 4.5j: Script Plugin Interface + Runtime Extraction

## References

| Doc | Purpose |
|-----|---------|
| `docs/plans/2026-07-15-phase-4.5j-script-plugins-runtime-extraction.md` | **PRIMARY** — Full design doc with architecture, interfaces, code, checklists |
| `docs/plans/instructions-implement-script-plugins.md` | Implementation instruction — task-by-task guide |
| `docs/plans/2026-07-14-runtime-engine-redesign-master.md` | Master doc — phase table, dependency graph, naming conventions |
| `docs/plans/handoff-phase-4.5h-to-4.5k.md` | Handoff — codebase context, what to read, gotchas |

Read the design doc FIRST (all 1776 lines). It contains everything needed:
- `ScriptRuntime` interface definition (Section 4)
- `script:` import parser + VM changes (Section 5)
- `go-json-runtimes` package structure + go.mod (Section 6)
- Pool manager with `HardMaxMemoryMB` + `TimeoutConfig` (Section 7)
- BitCode config compatibility — resolution hierarchy preserved (Section 7b)
- Migration plan — file mapping, what stays in BitCode (Section 8)
- CLI default runtimes — goja/yaegi/quickjs embedded (Section 5b checklist)
- Documentation outlines — exact content for each doc (Section 16)
- Full implementation checklist — 6 phases, ~80 items (Section 13)

## Target

```
packages/go-json/runtime/          ← ScriptRuntime interface, WithScriptRuntime option
packages/go-json/lang/             ← script: import parser + VM call dispatch
packages/go-json-runtimes/         ← NEW package (separate go.mod)
packages/go-json/cmd/go-json/     ← CLI default runtime registration
engine/internal/runtime/embedded/  ← thin wrapper (VMAdapter)
```

## Verify

```bash
# After each phase:
cd packages/go-json && go build ./... && go vet ./... && go test ./... -v
cd packages/go-json-runtimes && go build ./... && go vet ./... && go test ./... -v
cd engine && go build ./... && go vet ./... && go test ./... -v
```

## Cara Kerja

1. Baca design doc lengkap (`2026-07-15-phase-4.5j-script-plugins-runtime-extraction.md`) + instruction file (`instructions-implement-script-plugins.md`)
2. Kerjakan task-by-task sesuai stream order di Section 13 (Phase 1 → 2 → 3 → 4 → 5 → 5b → 6)
3. Setiap stream selesai: review kritis dan jujur (edge cases, backward compat). Jika ada yang perlu diperbaiki dan disesuaikan, langsung fix (kecuali memang blocker). Ulang fase ini (review → need fix & not blocker → fix → review lagi, sampai tidak ada yang perlu di-fix)
4. Update doc-doc terkait (lihat Section 13 Phase 6 — split per scope 6a/6b/6c/6d), lalu commit

## Stream Order

### Stream 1: go-json Core (Section 13 Phase 1)
- Define `ScriptRuntime` interface in `packages/go-json/runtime/script_runtime.go`
- Define `ScriptRuntimeRegistry` in same file
- Add `WithScriptRuntime()`, `WithScriptBridge()` options
- Modify parser: add `script:` to import path classification
- Modify VM: step-level `call` dispatch for `"ml.predict"` → split on `.`, check script imports
- Modify `Runtime.Execute()` + `ExecuteFunction()`: inject script proxy for `script:` imports
- Script proxy: `map[string]any{"call": func(function string, args ...any) (any, error) {...}}`
- Path validation: reject absolute paths, prevent traversal
- Write all tests from checklist
- **Verify**: all 969+ existing go-json tests still pass, zero new dependencies in go.mod

### Stream 2: go-json-runtimes Package (Section 13 Phase 2)
- Create `packages/go-json-runtimes/` with `go.mod` (go 1.24)
- Write interfaces: `VM`, `EmbeddedRuntime`, `ExternalRuntime` — `InjectBridge(map[string]any)` NOT `*bridge.Context`
- Write executor, registry, loader, helpers (from BitCode, decoupled)
- Write `PoolConfig` with ALL fields: `Size`, `MaxExecutions`, `MaxMemoryMB`, `HardMaxMemoryMB`, `MaxIdleTime`, `CrashRecovery`, `MaxBackoff`
- Write `TimeoutConfig`: `DefaultStepTimeout`, `MaxStepTimeout`, `MaxProcessTimeout`
- Write `PoolWithTimeoutConfig`, `DualPoolConfig`, `DualPool`
- Write functional options: `WithGoja()`, `WithQuickJS()`, `WithYaegi()`, `WithNode()`, `WithPython()`
- **Verify**: `go build ./...` passes

### Stream 3: Extract Embedded Runtimes (Section 13 Phase 3)
- Copy goja/ from BitCode → change `InjectBridge(*bridge.Context)` to `InjectBridge(map[string]any)`
- Copy quickjs/ — same change
- Copy yaegi/ — same change + keep bridges loader
- Migrate tests
- **Verify**: all extracted tests pass

### Stream 4: Extract External Runtimes (Section 13 Phase 4)
- Extract Node.js from `engine/internal/runtime/plugin/`
- Extract Python from same
- Decouple from engine types, integrate pool manager
- Migrate tests (78 Node.js + 11 Python)
- **Verify**: all extracted tests pass

### Stream 5: BitCode Integration (Section 13 Phase 5)
- Add `go-json-runtimes` to `engine/go.mod`
- Create `VMAdapter` in `engine/internal/runtime/embedded/` — converts `*bridge.Context` → `map[string]any`
- Update BitCode runtime initialization to use go-json-runtimes
- Remove duplicated code from BitCode
- **CRITICAL**: `bitcode.yaml` runtime config format UNCHANGED (see Section 7b)
- **CRITICAL**: Resolution hierarchy (step → process → module → project → hardcoded) PRESERVED — BitCode executor resolves, passes via `context.Context`
- **Verify**: ALL existing BitCode tests pass — this is refactoring, not rewrite

### Stream 5b: CLI Default Runtimes (Section 13 Phase 5b)
- Register goja, yaegi, quickjs as default in `packages/go-json/cmd/go-json/main.go`
- Register Node.js, Python with `Enabled: "auto"`
- **Verify**: `go-json run program.json` with `script:./plugin.js` works out of the box
- **Verify**: `packages/go-json/go.mod` has ZERO new dependencies (runtimes only in CLI binary)

### Stream 6: Documentation (Section 13 Phase 6)
- Follow Section 16 outlines EXACTLY for new docs
- Update all docs listed in Phase 6 checklist (6a/6b/6c/6d)
- **CRITICAL**: `engine/docs/features/plugins.md` needs multiple changes (line-by-line specified in checklist)
- Run verification items in 6d

## Aturan

- WAJIB berpikir kritis, detail, mateng, lengkap dan jujur
- Saya tidak ingin tech debt di phase ini, jadi pastikan tidak ada tech debt (sampaikan rekomendasinya setelah selesai semua pengerjaan)
- Jika ada blocker, jangan kerjakan dulu, konfirmasikan ke saya terlebih dahulu
- Go version: **1.24** (semua go.mod)
- go-json core MUST remain zero-dependency — `ScriptRuntime` is interface only
- Bridge is `map[string]any` — runtimes NEVER import BitCode types
- Pool manager is optional — standalone users can use without pool
- All existing tests must pass after every stream

## Key Gotchas (Read Before Starting)

1. **`bridge_helper.go` STAYS in BitCode** — it imports `bridge.SearchOptions`, `bridge.HTTPOptions`. Only copy generic helpers (`ToInt`, `ToStringSlice`) to go-json-runtimes.

2. **goja `buildBitcodeObject` is no longer needed in go-json-runtimes** — bridge is ALREADY `map[string]any`. goja just does `v.rt.Set("bitcode", bridge)`.

3. **yaegi `bridgeHolder` pattern must be preserved** — all proxies call `h.get()` which returns current context pointer. This enables tx swap (Phase 4.5h).

4. **Node.js/Python `txStore` must be extracted** — it's in `engine/internal/runtime/plugin/tx_store.go`. Move to `go-json-runtimes/node/` (shared with Python).

5. **Timeout is NOT pool's job** — pool manages process lifecycle. Timeout is enforced by caller via `context.Context`. `TimeoutConfig` is stored for introspection, not enforcement.

6. **`script:` expression-level calls use `ml.call("predict", args)`** — NOT `ml.predict(args)`. Dot-notation is future enhancement. See Section 5.7 Tier 2.
