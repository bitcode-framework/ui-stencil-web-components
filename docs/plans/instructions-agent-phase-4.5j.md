Implement Phase 4.5j: Script Plugin Interface + Runtime Extraction.

Reference: `docs/plans/2026-07-15-phase-4.5j-script-plugins-runtime-extraction.md`
Instruction: `docs/plans/instructions-implement-script-plugins.md`
Master doc: `docs/plans/2026-07-14-runtime-engine-redesign-master.md`
Handoff: `docs/plans/handoff-phase-4.5h-to-4.5k.md`

This file contains everything needed:
- `ScriptRuntime` interface definition (Section 4)
- `script:` import parser + VM changes — two-tier call design (Section 5)
- `go-json-runtimes` package structure + go.mod (Section 6)
- Pool manager with `HardMaxMemoryMB` + `TimeoutConfig` (Section 7)
- BitCode config compatibility — `bitcode.yaml` format UNCHANGED, resolution hierarchy preserved (Section 7b)
- Migration plan — file mapping, what stays in BitCode, what moves (Section 8)
- CLI default runtimes — goja/yaegi/quickjs embedded, Node.js/Python auto-detected (Phase 5b checklist)
- Documentation outlines — exact content for each doc to write (Section 16)
- Full implementation checklist — 6 phases, ~80 items (Section 13)

Target:
```
packages/go-json/runtime/          ← ScriptRuntime interface, WithScriptRuntime option
packages/go-json/lang/             ← script: import parser + VM call dispatch
packages/go-json-runtimes/         ← NEW package (separate go.mod, go 1.24)
packages/go-json/cmd/go-json/     ← CLI default runtime registration
engine/internal/runtime/embedded/  ← thin wrapper (VMAdapter)
```

Verify:
```bash
cd packages/go-json && go build ./... && go vet ./... && go test ./... -v
cd packages/go-json-runtimes && go build ./... && go vet ./... && go test ./... -v
cd engine && go build ./... && go vet ./... && go test ./... -v
```

## Cara Kerja
1. Baca design doc + instruction file lengkap
2. Kerjakan task-by-task sesuai stream order (Section 13: Phase 1 → 2 → 3 → 4 → 5 → 5b → 6)
3. Setiap stream selesai: review kritis dan jujur (edge cases, backward compat), jika ada yg perlu di perbaiki dan sesuaikan, langsung fix (kecuali memang blocker), ulang fase ini (review → need fix & not blocker → fix → review lagi, sampai tidak ada yg perlu di fix)
4. Update doc-doc terkait (Section 13 Phase 6 — split per scope 6a/6b/6c/6d, ikuti outline di Section 16), lalu commit

## Aturan
- WAJIB berpikir kritis, detail, mateng, lengkap dan jujur
- Saya tidak ingin tech debt di phase ini, jadi pastikan tidak ada tech debt (sampaikan rekomendasinya setelah selesai semua pengerjaan)
- Jika ada blocker, jangan kerjakan dulu, konfirmasikan ke saya terlebih dahulu
- Go version: **1.24** (semua go.mod)
- go-json core MUST remain zero-dependency — `ScriptRuntime` is interface only, TIDAK import go-json-runtimes
- Bridge is `map[string]any` — runtimes NEVER import BitCode types
- Pool manager is optional — standalone users can use without pool
- All existing tests must pass after every stream
