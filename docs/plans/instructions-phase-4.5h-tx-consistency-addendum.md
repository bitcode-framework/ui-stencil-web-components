# Addendum: Phase 4.5h — Unified Imperative Transaction API (All Runtimes)

**Date**: 15 July 2026
**Context**: This addendum extends Phase 4.5h to unify the transaction API across ALL runtimes.
**Decision**: Replace callback-based `tx(fn)` in goja/yaegi with imperative `tx.begin/commit/rollback` for API consistency.

---

## Background: Why This Change

Currently, transaction support is inconsistent across runtimes:

| Runtime | Current API | Style |
|---------|-------------|-------|
| **goja** | `bitcode.tx(function() { ... })` | Callback |
| **yaegi** | `bitcode.Tx(func() error { ... })` | Callback |
| **Node.js** | `bitcode.tx.begin()` / `bitcode.tx.commit()` / `bitcode.tx.rollback()` | Imperative |
| **Python** | `bitcode.tx.begin()` / `bitcode.tx.commit()` / `bitcode.tx.rollback()` | Imperative |
| **go-json** | ❌ None (removed in 4.5c-fix) | — |

**Problem**: A developer writing a goja script uses `bitcode.tx(fn)`. If they later move that logic to Node.js or go-json, they must rewrite the transaction pattern. This is tech debt.

**Decision**: ALL runtimes use imperative `begin/commit/rollback`. Callback style is removed from goja and yaegi.

---

## Design Principle: Consistent CONCEPT, Language-Appropriate SYNTAX

All runtimes share the same **conceptual API**: three operations — begin, commit, rollback. But each runtime expresses it in its **native language convention**:

| Runtime | Language | Convention | Tx API |
|---------|----------|-----------|--------|
| **go-json** | JSON DSL | dot-access namespace | `bc.tx.begin()` / `bc.tx.commit()` / `bc.tx.rollback()` |
| **goja** | JavaScript | dot-access namespace | `bitcode.tx.begin()` / `bitcode.tx.commit()` / `bitcode.tx.rollback()` |
| **Node.js** | TypeScript/JS | dot-access namespace | `bitcode.tx.begin()` / `bitcode.tx.commit()` / `bitcode.tx.rollback()` |
| **Python** | Python | dot-access namespace | `bitcode.tx.begin()` / `bitcode.tx.commit()` / `bitcode.tx.rollback()` |
| **yaegi** | Go | accessor + method (Go convention) | `bitcode.Tx().Begin()` / `bitcode.Tx().Commit()` / `bitcode.Tx().Rollback()` |

### Why yaegi Uses `Tx().Begin()` Instead of `tx.begin()`

yaegi exposes the bridge as a **Go package** (`import "bitcode"`). Go packages cannot have nested namespaces like `bitcode.tx.begin()`. The existing yaegi convention for ALL namespaces is the **accessor pattern**:

```go
// Existing yaegi patterns (already in production):
bitcode.DB().Query(sql)         // not bitcode.db.query(sql)
bitcode.HTTP().Get(url)         // not bitcode.http.get(url)
bitcode.Email().Send(opts)      // not bitcode.email.send(opts)
bitcode.Cache().Get(key)        // not bitcode.cache.get(key)
bitcode.Storage().Upload(opts)  // not bitcode.storage.upload(opts)

// NEW — follows the SAME pattern:
bitcode.Tx().Begin()            // not bitcode.tx.begin()
bitcode.Tx().Commit()           // not bitcode.tx.commit()
bitcode.Tx().Rollback()         // not bitcode.tx.rollback()
```

This is NOT an inconsistency — it IS the yaegi convention. yaegi is a Go interpreter; Go has PascalCase exports and accessor methods. Forcing JavaScript-style `bitcode.tx.begin()` into yaegi would be the actual inconsistency.

---

## After This Change

| Runtime | API | Notes |
|---------|-----|-------|
| **go-json** | `bc.tx.begin()` / `bc.tx.commit()` / `bc.tx.rollback()` | NEW — imperative namespace |
| **goja** | `bitcode.tx.begin()` / `bitcode.tx.commit()` / `bitcode.tx.rollback()` | CHANGED — was callback |
| **Node.js** | `await bitcode.tx.begin()` / `await bitcode.tx.commit()` / `await bitcode.tx.rollback()` | UNCHANGED |
| **Python** | `bitcode.tx.begin()` / `bitcode.tx.commit()` / `bitcode.tx.rollback()` | UNCHANGED |
| **yaegi** | `bitcode.Tx().Begin()` / `bitcode.Tx().Commit()` / `bitcode.Tx().Rollback()` | CHANGED — was callback, now follows Go accessor pattern |

---

## What to Do

### 1. go-json Extension (`gojson_adapter.go`)

**Add** `"tx"` namespace to `BuildGoJSONExtension()`:

```go
// Add to BuildGoJSONExtension in gojson_adapter.go:
//
// Transaction namespace: imperative begin/commit/rollback for go-json.
// Design note: go-json cannot produce Go callbacks, so callback-based
// bc.Tx(fn) (previously used by goja/yaegi) is not possible here.
// All runtimes now use imperative style for consistency.
// Previously removed in Phase 4.5c-fix Task 11 (was callback-based).
// Re-added here as namespace with completely different design.
"tx": map[string]any{
    "begin": func() (any, error) {
        return nil, bc.txBridge.Begin()
    },
    "commit": func() (any, error) {
        return nil, bc.txBridge.Commit()
    },
    "rollback": func() (any, error) {
        return nil, bc.txBridge.Rollback()
    },
},
```

**Update** test `TestBuildGoJSONExtension_NoTxFunction` (line 603-609 of `gojson_adapter_test.go`):

```go
// DELETE the old test and REPLACE with:
func TestBuildGoJSONExtension_TxNamespace(t *testing.T) {
    bc := newTestContext()
    ext := BuildGoJSONExtension(bc)

    // tx MUST be present as map[string]any namespace (not function)
    txNs, ok := ext.Functions["tx"]
    if !ok {
        t.Fatal("tx namespace should exist in extension")
    }
    txMap, ok := txNs.(map[string]any)
    if !ok {
        t.Fatal("tx should be map[string]any (namespace), not a function")
    }
    if _, ok := txMap["begin"]; !ok {
        t.Error("tx.begin should exist")
    }
    if _, ok := txMap["commit"]; !ok {
        t.Error("tx.commit should exist")
    }
    if _, ok := txMap["rollback"]; !ok {
        t.Error("tx.rollback should exist")
    }
}
```

---

### 2. goja Runtime (`engine/internal/runtime/embedded/goja/proxy.go`)

**Replace** callback-based `"tx"` (line 83-90) with imperative namespace:

**BEFORE** (line 83-90):
```go
"tx": func(fn goja.Callable) error {
    return bc.Tx(func(txCtx *bridge.Context) error {
        v.rt.Set("bitcode", v.buildBitcodeObject(txCtx))
        defer v.rt.Set("bitcode", v.buildBitcodeObject(bc))
        _, err := fn(goja.Undefined())
        return err
    })
},
```

**AFTER**:
```go
"tx": map[string]any{
    "begin": func() error {
        db := bc.GormDB()
        if db == nil {
            return fmt.Errorf("database not available for transactions")
        }
        if v.txGormTx != nil {
            return fmt.Errorf("transaction already active — commit or rollback first")
        }
        gormTx := db.Begin()
        if gormTx.Error != nil {
            return gormTx.Error
        }
        txCtx := bc.CloneWithGormTx(gormTx)
        // Swap the entire bitcode object to use tx context
        v.txOriginalBC = bc
        v.txGormTx = gormTx
        v.rt.Set("bitcode", v.buildBitcodeObject(txCtx))
        return nil
    },
    "commit": func() error {
        if v.txGormTx == nil {
            return fmt.Errorf("no active transaction to commit")
        }
        err := v.txGormTx.Commit().Error
        // Restore original bitcode object
        v.rt.Set("bitcode", v.buildBitcodeObject(v.txOriginalBC))
        v.txGormTx = nil
        v.txOriginalBC = nil
        return err
    },
    "rollback": func() error {
        if v.txGormTx == nil {
            return fmt.Errorf("no active transaction to rollback")
        }
        err := v.txGormTx.Rollback().Error
        // Restore original bitcode object
        v.rt.Set("bitcode", v.buildBitcodeObject(v.txOriginalBC))
        v.txGormTx = nil
        v.txOriginalBC = nil
        return err
    },
},
```

**Add fields** to `GojaVM` struct (in `vm.go`):

```go
type GojaVM struct {
    rt             *goja.Runtime
    timeout        time.Duration
    // Transaction state for imperative tx.begin/commit/rollback
    txOriginalBC   *bridge.Context
    txGormTx       *gorm.DB
}
```

**Add cleanup** to `GojaVM.Close()`:

```go
func (v *GojaVM) Close() {
    // Auto-rollback dangling transaction on VM close
    if v.txGormTx != nil {
        v.txGormTx.Rollback()
        v.txGormTx = nil
        v.txOriginalBC = nil
    }
}
```

**Usage from goja script (JavaScript)**:
```javascript
// BEFORE (callback — REMOVED):
// bitcode.tx(function() {
//     bitcode.model("account").write(id, {balance: newBalance});
// });

// AFTER (imperative):
bitcode.tx.begin();
try {
    bitcode.model("account").write(fromId, {balance: fromBalance - amount});
    bitcode.model("account").write(toId, {balance: toBalance + amount});
    bitcode.tx.commit();
} catch(e) {
    bitcode.tx.rollback();
    throw e;
}
```

---

### 3. yaegi Runtime (`engine/internal/runtime/embedded/yaegi/symbols.go`)

yaegi uses the **Go accessor pattern** for all namespaces (`DB()`, `HTTP()`, `Email()`, etc.). The transaction API follows the same pattern using a proxy struct.

**Replace** callback-based `"Tx"` (line 54-61) with accessor that returns proxy struct:

**BEFORE** (line 54-61):
```go
"Tx": reflect.ValueOf(func(fn func() error) error {
    return h.get().Tx(func(txCtx *bridge.Context) error {
        original := h.ctx
        h.ctx = txCtx
        defer func() { h.ctx = original }()
        return fn()
    })
}),
```

**AFTER**:
```go
"Tx": reflect.ValueOf(func() *goTxProxy { return newTxProxy(h) }),
```

**Add `goTxProxy` struct** (in `symbols.go`, alongside other proxy structs):

```go
// --- Tx proxy (follows same pattern as DB, HTTP, Email, etc.) ---

type goTxProxy struct {
    h *bridgeHolder
}

func newTxProxy(h *bridgeHolder) *goTxProxy { return &goTxProxy{h: h} }

func (t *goTxProxy) Begin() error {
    db := t.h.get().GormDB()
    if db == nil {
        return fmt.Errorf("database not available for transactions")
    }
    if t.h.txGormTx != nil {
        return fmt.Errorf("transaction already active — commit or rollback first")
    }
    gormTx := db.Begin()
    if gormTx.Error != nil {
        return gormTx.Error
    }
    txCtx := t.h.get().CloneWithGormTx(gormTx)
    t.h.txOriginalCtx = t.h.ctx
    t.h.txGormTx = gormTx
    t.h.ctx = txCtx
    return nil
}

func (t *goTxProxy) Commit() error {
    if t.h.txGormTx == nil {
        return fmt.Errorf("no active transaction to commit")
    }
    err := t.h.txGormTx.Commit().Error
    t.h.ctx = t.h.txOriginalCtx
    t.h.txGormTx = nil
    t.h.txOriginalCtx = nil
    return err
}

func (t *goTxProxy) Rollback() error {
    if t.h.txGormTx == nil {
        return fmt.Errorf("no active transaction to rollback")
    }
    err := t.h.txGormTx.Rollback().Error
    t.h.ctx = t.h.txOriginalCtx
    t.h.txGormTx = nil
    t.h.txOriginalCtx = nil
    return err
}
```

**Add fields** to `bridgeHolder` struct:

```go
type bridgeHolder struct {
    ctx            *bridge.Context
    // Transaction state for imperative Tx().Begin/Commit/Rollback
    txOriginalCtx  *bridge.Context
    txGormTx       *gorm.DB
}
```

**Add cleanup** — yaegi VM close should rollback dangling tx. In `vm.go`, ensure the VM's Close or cleanup path calls:

```go
// In YaegiVM close/cleanup:
if v.holder != nil && v.holder.txGormTx != nil {
    v.holder.txGormTx.Rollback()
    v.holder.txGormTx = nil
    v.holder.txOriginalCtx = nil
}
```

**Usage from yaegi script (Go)**:
```go
// BEFORE (callback — REMOVED):
// bitcode.Tx(func() error {
//     bitcode.Model("contact").Create(map[string]any{"name": "InTx"})
//     return nil
// })

// AFTER (imperative — follows same pattern as DB(), HTTP(), etc.):
if err := bitcode.Tx().Begin(); err != nil {
    return nil, err
}
_, err := bitcode.Model("contact").Create(map[string]any{"name": "InTx"})
if err != nil {
    bitcode.Tx().Rollback()
    return nil, err
}
if err := bitcode.Tx().Commit(); err != nil {
    return nil, err
}
```

**Note**: `bitcode.Tx()` returns the same `*goTxProxy` every time (stateless accessor — state lives in `bridgeHolder`). This is identical to how `bitcode.DB()` returns `*goDBProxy` — the proxy is a thin wrapper over the shared `bridgeHolder`.

---

### 4. Node.js / Python — NO CHANGES

Node.js and Python already use imperative `tx.begin/commit/rollback` via JSON-RPC through `txStore`. **Do not modify** `engine/internal/runtime/plugin/tx_store.go` or `manager.go`.

---

### 5. Tests to Update

#### 5.1 yaegi test: `TestYaegiBridgeTxSwapsContext`

File: `engine/internal/runtime/embedded/yaegi/yaegi_test.go` (around line 645)

**BEFORE**:
```go
func TestYaegiBridgeTxSwapsContext(t *testing.T) {
    // ... setup ...
    code := `
package main

import "bitcode"

func Execute(ctx context.Context, params map[string]any) (any, error) {
    err := bitcode.Tx(func() error {
        _, err := bitcode.Model("contact").Create(map[string]any{"name": "InTx"})
        return err
    })
    if err != nil {
        return nil, err
    }
    return map[string]any{"tx": "ok"}, nil
}
`
    // ...
}
```

**AFTER**:
```go
func TestYaegiBridgeTxImperative(t *testing.T) {
    // ... setup ...
    code := `
package main

import "bitcode"

func Execute(ctx context.Context, params map[string]any) (any, error) {
    if err := bitcode.Tx().Begin(); err != nil {
        return nil, err
    }
    _, err := bitcode.Model("contact").Create(map[string]any{"name": "InTx"})
    if err != nil {
        bitcode.Tx().Rollback()
        return nil, err
    }
    if err := bitcode.Tx().Commit(); err != nil {
        return nil, err
    }
    return map[string]any{"tx": "ok"}, nil
}
`
    // ...
}
```

#### 5.2 goja test (if any tx test exists)

Search for any goja test that uses callback `tx`. Update to imperative style:
```javascript
// Old: bitcode.tx(function() { ... })
// New: bitcode.tx.begin(); try { ... bitcode.tx.commit(); } catch(e) { bitcode.tx.rollback(); }
```

#### 5.3 New tests to add

```go
// goja:
func TestGojaTxBeginCommit(t *testing.T) { ... }
func TestGojaTxBeginRollback(t *testing.T) { ... }
func TestGojaTxDoubleBeginError(t *testing.T) { ... }
func TestGojaTxCommitWithoutBeginError(t *testing.T) { ... }
func TestGojaTxRollbackWithoutBeginError(t *testing.T) { ... }
func TestGojaTxAutoRollbackOnClose(t *testing.T) { ... }

// yaegi:
func TestYaegiTxBeginCommit(t *testing.T) { ... }
func TestYaegiTxBeginRollback(t *testing.T) { ... }
func TestYaegiTxDoubleBeginError(t *testing.T) { ... }
func TestYaegiTxCommitWithoutBeginError(t *testing.T) { ... }
func TestYaegiTxRollbackWithoutBeginError(t *testing.T) { ... }
func TestYaegiTxAutoRollbackOnClose(t *testing.T) { ... }
```

---

### 6. Safety Nets (CRITICAL)

All runtimes MUST have auto-rollback protection:

1. **On VM Close**: If a transaction is active when the VM is closed/destroyed, auto-rollback.
   - goja: in `GojaVM.Close()` — check `v.txGormTx != nil`, rollback
   - yaegi: in VM close path — check `holder.txGormTx != nil`, rollback
   - go-json: in `txBridge.Cleanup()` — already designed in instruction doc
   - Node.js/Python: already have `txStore.watchTimeout()` + `CleanupAll()`

2. **On Timeout**: Transactions that exceed 30 seconds are auto-rolled-back.
   - go-json: implement in `txBridge` (goroutine with timer, same as `txStore.watchTimeout`)
   - goja: start goroutine in `begin()` that auto-rollbacks after 30s
   - yaegi: same as goja
   - Node.js/Python: already have `txStore.watchTimeout(txID, 30*time.Second)`

   Implementation for goja/yaegi timeout:
   ```go
   // In begin():
   v.txTimeout = time.AfterFunc(30*time.Second, func() {
       // Auto-rollback if still active after 30s
       if v.txGormTx != nil {
           v.txGormTx.Rollback()
           v.txGormTx = nil
           v.txOriginalBC = nil
           // Restore bridge (best effort — VM may be executing)
       }
   })

   // In commit()/rollback():
   if v.txTimeout != nil {
       v.txTimeout.Stop()
       v.txTimeout = nil
   }
   ```

3. **Error messages**: Clear, actionable error messages (same across all runtimes):
   - `"no active transaction to commit"` — developer forgot `begin()`
   - `"no active transaction to rollback"` — already committed or never started
   - `"transaction already active — commit or rollback first"` — nested tx attempt
   - `"database not available for transactions"` — no DB configured

---

### 7. What NOT to Change

| Item | Reason |
|------|--------|
| `bridge/tx.go` (`txManager` struct) | Internal Go API. Still used by `Context.Tx()`. Keep for potential future use. |
| `bridge/interfaces.go` (`TxManager` interface) | Internal interface. Not exposed to scripts. |
| `bridge/context.go` (`Context.Tx(fn)` method) | Internal Go-level API. Not exposed to scripts. |
| `plugin/tx_store.go` | Already imperative for Node.js/Python — no changes needed. |
| `plugin/manager.go` `handleTxMethod()` | Already handles `tx.begin/commit/rollback` — no changes needed. |

**Rationale**: The callback-based `TxManager.RunTx()` and `Context.Tx(fn)` are internal Go APIs used by the engine itself (not by scripts). They may be useful for future Go-level transaction management within the engine. Removing them would be unnecessary churn with no benefit. The only thing removed is the **script-facing callback** in goja and yaegi.

---

### 8. Documents to Update

After implementation, update these documents:

| Document | What to Change |
|----------|---------------|
| `engine/AGENTS.md` | Bridge section: document unified imperative tx API across all runtimes, note yaegi uses `Tx().Begin()` pattern |
| `engine/docs/features/plugins.md` | Full tx API reference with examples for each runtime |
| `docs/plans/2026-07-15-phase-4.5h-bridge-ergonomics.md` | Section 3.2: add note that goja/yaegi callback is replaced |
| `docs/plans/instructions-implement-bridge-ergonomics.md` | Task 2: add goja/yaegi changes to scope |
| `docs/plans/handoff-phase-4.5h-to-4.5k.md` | Section 3.2 (tx gotcha): update — callback is now REMOVED from goja/yaegi, imperative everywhere |

---

### 9. Verification Checklist

```bash
# All must pass:
cd engine

# Bridge tests (go-json extension)
go test ./internal/runtime/bridge/ -v

# goja tests
go test ./internal/runtime/embedded/goja/ -v

# yaegi tests
go test ./internal/runtime/embedded/yaegi/ -v

# Node.js integration (if Node.js available)
go test ./internal/runtime/plugin/ -run TestNodeJS -v

# Python integration (if Python available)
go test ./internal/runtime/plugin/ -run TestPython -v

# Full build + vet
go build ./...
go vet ./...
```

---

## Summary of Changes

| File | Action |
|------|--------|
| `engine/internal/runtime/bridge/gojson_adapter.go` | ADD `"tx"` namespace with begin/commit/rollback |
| `engine/internal/runtime/bridge/gojson_adapter_test.go` | REPLACE `TestBuildGoJSONExtension_NoTxFunction` → `TestBuildGoJSONExtension_TxNamespace` |
| `engine/internal/runtime/bridge/tx_bridge.go` | NEW — imperative tx manager for go-json |
| `engine/internal/runtime/bridge/tx_bridge_test.go` | NEW — tests for txBridge |
| `engine/internal/runtime/embedded/goja/proxy.go` | REPLACE callback `"tx": func(fn goja.Callable)` → imperative `"tx": map[string]any{begin, commit, rollback}` |
| `engine/internal/runtime/embedded/goja/vm.go` | ADD `txOriginalBC`, `txGormTx`, `txTimeout` fields + cleanup in Close() |
| `engine/internal/runtime/embedded/yaegi/symbols.go` | REPLACE `"Tx": reflect.ValueOf(func(fn func() error) error {...})` → `"Tx": reflect.ValueOf(func() *goTxProxy {...})` |
| `engine/internal/runtime/embedded/yaegi/symbols.go` | ADD `goTxProxy` struct with Begin/Commit/Rollback methods |
| `engine/internal/runtime/embedded/yaegi/symbols.go` | ADD `txOriginalCtx`, `txGormTx` fields to `bridgeHolder` |
| `engine/internal/runtime/embedded/yaegi/vm.go` | ADD cleanup in Close() for dangling tx |
| `engine/internal/runtime/embedded/yaegi/yaegi_test.go` | UPDATE `TestYaegiBridgeTxSwapsContext` → `TestYaegiTxImperative` with `Tx().Begin()` pattern |
| Node.js/Python code | NO CHANGES |
| `bridge/tx.go` (existing txManager) | NO CHANGES (keep for internal use) |
| `bridge/interfaces.go` (TxManager interface) | NO CHANGES |
| `bridge/context.go` (Context.Tx method) | NO CHANGES |

---

## API Quick Reference (For Documentation)

### go-json
```jsonc
{"let": "_", "expr": "bc.tx.begin()"}
{"let": "_", "expr": "bc.tx.commit()"}
{"let": "_", "expr": "bc.tx.rollback()"}
```

### goja (JavaScript)
```javascript
bitcode.tx.begin();
bitcode.tx.commit();
bitcode.tx.rollback();
```

### yaegi (Go)
```go
bitcode.Tx().Begin()
bitcode.Tx().Commit()
bitcode.Tx().Rollback()
```

### Node.js (TypeScript)
```typescript
await bitcode.tx.begin();
await bitcode.tx.commit();
await bitcode.tx.rollback();
```

### Python
```python
bitcode.tx.begin()
bitcode.tx.commit()
bitcode.tx.rollback()
```
