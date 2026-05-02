Implement Phase 4.5k: WASM + Native Plugins.

**Prerequisite**: Phase 4.5j MUST be complete. If `packages/go-json-runtimes/` does not exist or `ScriptRuntime` interface is not defined, STOP and confirm with me.

Reference: `docs/plans/2026-07-15-phase-4.5k-wasm-native-plugins.md`
Instruction: `docs/plans/instructions-implement-wasm-native-plugins.md`
Master doc: `docs/plans/2026-07-14-runtime-engine-redesign-master.md`

This file contains everything needed:
- WASM runtime via wazero — pure Go, zero CGO (Section 4)
- JSON-over-linear-memory protocol — malloc/free/packed i64 return (Section 4.2-4.3)
- Host functions — `bridge_call` for WASM→host bridge access (Section 4.4)
- WASM instance pooling + bitcode.yaml config mapping (Section 4.7-4.8)
- Native plugin via Go `plugin` package — Linux/macOS only (Section 5)
- Native plugin protocol — Manifest() + `func(map[string]any) (any, error)` (Section 5.2-5.3)
- Codegen strategy — WASM portable across Go/JS/Python, native generates stubs (Section 7)
- CLI default: WASM enabled, native NOT enabled for security (Phase 5 checklist)
- Documentation outlines — wasm.md, native.md authoring guides (Section 15)
- Full implementation checklist — 6 phases, ~70 items (Section 11)

Target:
```
packages/go-json/lang/                    ← wasm: and plugin: import types in parser
packages/go-json/runtime/                 ← import resolution for wasm:/plugin:
packages/go-json-runtimes/wasm/           ← NEW — WasmRuntime (wazero)
packages/go-json-runtimes/native/         ← NEW — NativeRuntime (Go plugin package)
packages/go-json-runtimes/options.go      ← WithWasm(), WithNative() options
packages/go-json/cmd/go-json/             ← register WASM as CLI default
packages/go-json/codegen/                 ← WASM/native codegen
```

Verify:
```bash
cd packages/go-json && go build ./... && go vet ./... && go test ./... -v
cd packages/go-json-runtimes && go build ./... && go vet ./... && go test ./wasm/ -v && go test ./native/ -v
cd engine && go build ./... && go vet ./... && go test ./... -v
```

## Cara Kerja
1. Baca design doc + instruction file lengkap
2. Kerjakan task-by-task sesuai stream order (Section 11: Phase 1 → 2 → 3 → 4 → 5 → 6)
3. Setiap stream selesai: review kritis dan jujur (edge cases, backward compat), jika ada yg perlu di perbaiki dan sesuaikan, langsung fix (kecuali memang blocker), ulang fase ini (review → need fix & not blocker → fix → review lagi, sampai tidak ada yg perlu di fix)
4. Update doc-doc terkait (Section 11 Phase 6 — split per scope 6a-6e, ikuti outline di Section 15), lalu commit

## Aturan
- WAJIB berpikir kritis, detail, mateng, lengkap dan jujur
- Saya tidak ingin tech debt di phase ini, jadi pastikan tidak ada tech debt (sampaikan rekomendasinya setelah selesai semua pengerjaan)
- Jika ada blocker, jangan kerjakan dulu, konfirmasikan ke saya terlebih dahulu
- Go version: **1.24** (semua go.mod)
- go-json core MUST remain zero-dependency — wazero is in go-json-runtimes ONLY
- WASM default in CLI, native NOT default (security)
- All existing tests must pass after every stream
