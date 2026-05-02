# Agent Briefing: Phase 4.5h — Bridge API Ergonomics

**Read this FIRST before starting work.**

---

## Your Primary Instruction File

```
docs/plans/instructions-implement-bridge-ergonomics.md
```

This file contains the full implementation plan for Phase 4.5h with 3 tasks:
- Task 1: Fluent Model API
- Task 2: Transaction Support (`bc.tx.*`)
- Task 3: Email Template Shorthand

**Follow that file as your main guide.** Everything in it is correct EXCEPT for Task 2 scope — which is EXPANDED by the addendum below.

---

## CRITICAL ADDENDUM for Task 2 (Transactions)

```
docs/plans/instructions-phase-4.5h-tx-consistency-addendum.md
```

**Read this addendum BEFORE implementing Task 2.** It expands the scope of Task 2 to include goja and yaegi changes for API consistency.

### Summary of What the Addendum Adds

The original instruction file only covers adding `bc.tx.begin/commit/rollback` to the **go-json extension** (`gojson_adapter.go`). The addendum EXTENDS this to also:

1. **Replace callback `tx` in goja** (`proxy.go` line 83-90) with imperative `"tx": map[string]any{begin, commit, rollback}`
2. **Replace callback `Tx` in yaegi** (`symbols.go` line 54-61) with `"Tx": func() *goTxProxy` accessor pattern (same as `DB()`, `HTTP()`, `Email()`)
3. **Update yaegi test** `TestYaegiBridgeTxSwapsContext` → imperative style
4. **Add auto-rollback safety nets** (on VM close + 30s timeout) for goja and yaegi
5. **Add new tests** for goja and yaegi imperative tx

### Why This Expansion

- Node.js and Python ALREADY use imperative `tx.begin/commit/rollback`
- goja and yaegi used callback style — inconsistent with other runtimes
- Zero tech debt policy requires fixing this NOW, not later
- Effort is small (~2-3 hours extra) and the code changes are straightforward

---

## CRITICAL CLARIFICATIONS (Design Decisions)

These clarifications resolve ambiguities between the main instruction file and the addendum. **Follow these over any conflicting guidance in the other docs.**

---

### Clarification #1: `txBridge` Design for go-json — WHY It Must Be Different from goja/yaegi

**Problem**: The main instruction file designs `txBridge` as a struct that manages `*gorm.DB` directly. The addendum shows goja/yaegi using `CloneWithGormTx()` to swap the entire bridge context. These are DIFFERENT approaches. Which one for go-json?

**Answer: go-json MUST use the `txBridge` struct approach (from main instruction file). It CANNOT use `CloneWithGormTx` swap.**

**Why**: 
- `BuildGoJSONExtension(bc)` is called ONCE at setup time. All function closures capture `bc` at that moment.
- goja can swap because it calls `v.rt.Set("bitcode", v.buildBitcodeObject(txCtx))` — rebuilds the entire bridge object at runtime.
- yaegi can swap because `bridgeHolder.ctx` is a mutable pointer — all proxies call `h.get()` which returns the current pointer.
- go-json extension closures capture `bc` directly. There is NO mechanism to swap `bc` after build. The closures are frozen.

**Therefore**: Model operations in go-json must query `txBridge` to get the effective DB handle. When a tx is active, `txBridge` returns the tx handle; otherwise, the normal DB.

**Implementation approach — modify `Context` to route through `txBridge`**:

```go
// tx_bridge.go — NEW file
type txBridge struct {
    mu         sync.Mutex
    db         *gorm.DB      // normal DB (from factory)
    gormTx     *gorm.DB      // active transaction handle (nil when no tx)
    active     bool
    timeout    *time.Timer
}

func newTxBridge(db *gorm.DB) *txBridge {
    return &txBridge{db: db}
}

// EffectiveDB returns the tx handle if active, otherwise normal DB.
// This is what modelFactory and dbBridge should use.
func (t *txBridge) EffectiveDB() *gorm.DB {
    t.mu.Lock()
    defer t.mu.Unlock()
    if t.active && t.gormTx != nil {
        return t.gormTx
    }
    return t.db
}

func (t *txBridge) Begin() error {
    t.mu.Lock()
    defer t.mu.Unlock()
    if t.active {
        return fmt.Errorf("transaction already active — commit or rollback first")
    }
    gormTx := t.db.Begin()
    if gormTx.Error != nil {
        return fmt.Errorf("failed to begin transaction: %w", gormTx.Error)
    }
    t.gormTx = gormTx
    t.active = true
    t.timeout = time.AfterFunc(30*time.Second, t.autoRollback)
    return nil
}

func (t *txBridge) Commit() error {
    t.mu.Lock()
    defer t.mu.Unlock()
    if !t.active {
        return fmt.Errorf("no active transaction to commit")
    }
    if t.timeout != nil {
        t.timeout.Stop()
        t.timeout = nil
    }
    err := t.gormTx.Commit().Error
    t.gormTx = nil
    t.active = false
    if err != nil {
        return fmt.Errorf("commit failed: %w", err)
    }
    return nil
}

func (t *txBridge) Rollback() error {
    t.mu.Lock()
    defer t.mu.Unlock()
    if !t.active {
        return fmt.Errorf("no active transaction to rollback")
    }
    if t.timeout != nil {
        t.timeout.Stop()
        t.timeout = nil
    }
    err := t.gormTx.Rollback().Error
    t.gormTx = nil
    t.active = false
    if err != nil {
        return fmt.Errorf("rollback failed: %w", err)
    }
    return nil
}

func (t *txBridge) IsActive() bool {
    t.mu.Lock()
    defer t.mu.Unlock()
    return t.active
}

func (t *txBridge) Cleanup() {
    t.mu.Lock()
    defer t.mu.Unlock()
    if t.active && t.gormTx != nil {
        if t.timeout != nil {
            t.timeout.Stop()
            t.timeout = nil
        }
        t.gormTx.Rollback()
        t.gormTx = nil
        t.active = false
    }
}

func (t *txBridge) autoRollback() {
    t.mu.Lock()
    defer t.mu.Unlock()
    if t.active && t.gormTx != nil {
        t.gormTx.Rollback()
        t.gormTx = nil
        t.active = false
    }
}
```

**Wiring into existing code — TWO places need to use `txBridge.EffectiveDB()`**:

1. **`modelFactory`** — currently receives `*gorm.DB` in constructor. Change `repoFactory` closure to use `txBridge.EffectiveDB()`:

```go
// factory.go — modify NewContext:
func (f *Factory) NewContext(moduleName string, session Session, rules SecurityRules) *Context {
    txb := newTxBridge(f.DB)
    return &Context{
        txManager: newTxManager(f.DB),
        txBridge:  txb,  // ← NEW field
        model:     newModelFactoryWithTx(f.DB, f.ModelRegistry, f.PermService, txb),  // ← pass txBridge
        db:        newDBBridgeWithTx(f.DB, txb),  // ← pass txBridge
        // ... rest unchanged ...
    }
}
```

2. **`modelFactory.repoFactory`** — use `txBridge.EffectiveDB()` instead of raw `f.db`:

```go
// model.go — modify newModelFactory or create newModelFactoryWithTx:
func newModelFactoryWithTx(db *gorm.DB, registry *model.Registry, permService *persistence.PermissionService, txb *txBridge) *modelFactory {
    return &modelFactory{
        db:          db,
        registry:    registry,
        permService: permService,
        repoFactory: func(modelName string, session Session, _ *gorm.DB) (*persistence.GenericRepository, *parser.ModelDefinition, error) {
            effectiveDB := txb.EffectiveDB()  // ← uses tx handle when active
            modelDef, err := registry.Get(modelName)
            if err != nil || modelDef == nil {
                return nil, nil, ErrModelNotFoundFor(modelName)
            }
            tableName := registry.TableName(modelName)
            if tableName == "" {
                tableName = modelName
            }
            var repo *persistence.GenericRepository
            if session.TenantID != "" {
                repo = persistence.NewGenericRepositoryWithModelAndTenant(effectiveDB, tableName, modelDef, session.TenantID)
            } else {
                repo = persistence.NewGenericRepositoryWithModel(effectiveDB, tableName, modelDef)
            }
            repo.SetCurrentUser(session.UserID)
            repo.SetLocale(session.Locale)
            return repo, modelDef, nil
        },
    }
}
```

3. **`dbBridge`** — same pattern for `bc.db.query()` / `bc.db.execute()`:

```go
// db.go — modify or create newDBBridgeWithTx:
type dbBridge struct {
    db  *gorm.DB
    txb *txBridge  // optional — if set, use EffectiveDB()
}

func (d *dbBridge) effectiveDB() *gorm.DB {
    if d.txb != nil {
        return d.txb.EffectiveDB()
    }
    return d.db
}

func (d *dbBridge) Query(sql string, args ...any) ([]map[string]any, error) {
    db := d.effectiveDB()  // ← uses tx when active
    // ... existing query logic using db ...
}

func (d *dbBridge) Execute(sql string, args ...any) (*ExecDBResult, error) {
    db := d.effectiveDB()  // ← uses tx when active
    // ... existing execute logic using db ...
}
```

**Key insight**: When `bc.tx.begin()` is called, ALL subsequent `bc.model(...)` and `bc.db.*` operations automatically use the transaction handle — because they query `txBridge.EffectiveDB()` on every call. When `bc.tx.commit()` or `bc.tx.rollback()` is called, subsequent operations go back to normal DB. No bridge rebuild needed.

**This is DIFFERENT from goja/yaegi** where the entire bridge object is swapped. For go-json, only the DB handle is swapped (via `txBridge`), which is sufficient because all DB operations go through `txBridge.EffectiveDB()`.

---

### Clarification #2: `TestBuildGoJSONExtension_Structure` — Update `expectedKeys`

The test at line 253-268 of `gojson_adapter_test.go` checks exact function count. After adding `"tx"`, you MUST update `expectedKeys`:

```go
expectedKeys := []string{
    "model", "db", "http", "cache", "fs", "env", "config", "session",
    "log", "emit", "call", "exec", "email", "notify", "storage",
    "t", "security", "audit", "crypto", "execution",
    "tx",  // ← ADD THIS
}
```

**Also check**: Does `BuildGoJSONExtension` currently export keys NOT in this list (e.g., `"meta"`, `"refresh"`, `"script"`)? If yes, add those too. The test asserts `len(ext.Functions) == len(expectedKeys)` — any mismatch fails.

---

### Clarification #3: Mock `ModelHandle` for `query_builder_test.go`

The existing `mockModelHandle` in `gojson_adapter_test.go` (line 167-210) is MISSING `MorphAttach`, `MorphDetach`, `MorphSync` methods. The `ModelHandle` interface (in `interfaces.go`) requires them.

**Action**: Create a NEW, COMPLETE mock in `query_builder_test.go`:

```go
// query_builder_test.go

// testModelHandle is a complete mock of ModelHandle for query builder tests.
type testModelHandle struct {
    searchCalled  bool
    searchOpts    SearchOptions
    searchResult  []map[string]any
    searchErr     error
    getCalled     bool
    getID         string
    getResult     map[string]any
    getErr        error
    writeCalled   bool
    writeID       string
    writeData     map[string]any
    writeErr      error
    deleteCalled  bool
    deleteID      string
    deleteErr     error
    countResult   int64
    countErr      error
    sumResult     float64
    sumErr        error
}

func (m *testModelHandle) Search(opts SearchOptions) ([]map[string]any, error) {
    m.searchCalled = true
    m.searchOpts = opts
    return m.searchResult, m.searchErr
}
func (m *testModelHandle) Get(id string, opts ...GetOptions) (map[string]any, error) {
    m.getCalled = true
    m.getID = id
    return m.getResult, m.getErr
}
func (m *testModelHandle) Create(data map[string]any) (map[string]any, error) {
    data["id"] = "new-1"
    return data, nil
}
func (m *testModelHandle) Write(id string, data map[string]any) error {
    m.writeCalled = true
    m.writeID = id
    m.writeData = data
    return m.writeErr
}
func (m *testModelHandle) Delete(id string) error {
    m.deleteCalled = true
    m.deleteID = id
    return m.deleteErr
}
func (m *testModelHandle) Count(opts SearchOptions) (int64, error) {
    m.searchOpts = opts
    return m.countResult, m.countErr
}
func (m *testModelHandle) Sum(field string, opts SearchOptions) (float64, error) {
    m.searchOpts = opts
    return m.sumResult, m.sumErr
}
func (m *testModelHandle) Upsert(data map[string]any, uniqueFields []string) (map[string]any, error) {
    return data, nil
}
func (m *testModelHandle) CreateMany(records []map[string]any) ([]map[string]any, error) {
    return records, nil
}
func (m *testModelHandle) WriteMany(ids []string, data map[string]any) (*BulkResult, error) {
    return &BulkResult{Affected: int64(len(ids))}, nil
}
func (m *testModelHandle) DeleteMany(ids []string) (*BulkResult, error) {
    return &BulkResult{Affected: int64(len(ids))}, nil
}
func (m *testModelHandle) UpsertMany(records []map[string]any, uniqueFields []string) ([]map[string]any, error) {
    return records, nil
}
func (m *testModelHandle) AddRelation(id, field string, relatedIDs []string) error    { return nil }
func (m *testModelHandle) RemoveRelation(id, field string, relatedIDs []string) error { return nil }
func (m *testModelHandle) SetRelation(id, field string, relatedIDs []string) error    { return nil }
func (m *testModelHandle) LoadRelation(id, field string) ([]map[string]any, error) {
    return nil, nil
}
func (m *testModelHandle) MorphAttach(id, relation string, relatedIDs []string) error  { return nil }
func (m *testModelHandle) MorphDetach(id, relation string, relatedIDs []string) error  { return nil }
func (m *testModelHandle) MorphSync(id, relation string, relatedIDs []string) error    { return nil }
func (m *testModelHandle) Sudo() SudoModelHandle                                       { return nil }
```

**Do NOT modify** the existing `mockModelHandle` in `gojson_adapter_test.go` — it may be missing morph methods but if tests pass, leave it alone. Create your own fresh mock in the new test file.

---

### Clarification #4: Timeout Safety Net — `time.AfterFunc(30s)` in `txBridge.Begin()`

**Yes, implement exactly as described.** The pattern matches `txStore.watchTimeout()` used by Node.js/Python (see `engine/internal/runtime/plugin/tx_store.go` line 100-113).

```go
func (t *txBridge) Begin() error {
    // ... validation, db.Begin() ...
    t.timeout = time.AfterFunc(30*time.Second, t.autoRollback)
    return nil
}

func (t *txBridge) Commit() error {
    // ... validation ...
    if t.timeout != nil {
        t.timeout.Stop()
        t.timeout = nil
    }
    // ... commit ...
}

func (t *txBridge) Rollback() error {
    // ... validation ...
    if t.timeout != nil {
        t.timeout.Stop()
        t.timeout = nil
    }
    // ... rollback ...
}

func (t *txBridge) autoRollback() {
    t.mu.Lock()
    defer t.mu.Unlock()
    if t.active && t.gormTx != nil {
        t.gormTx.Rollback()
        t.gormTx = nil
        t.active = false
    }
}
```

**Same pattern for goja and yaegi** (in their respective `begin()` implementations — see addendum).

---

## Execution Order

1. **Task 1: Fluent Model API** — follow main instruction file exactly
2. **Task 2: Transaction Support** — follow this order:
   - a. Create `tx_bridge.go` with `txBridge` struct (see Clarification #1 above)
   - b. Modify `factory.go` to create `txBridge` and pass to model/db
   - c. Modify `model.go` — `newModelFactoryWithTx` or modify `repoFactory` to use `txBridge.EffectiveDB()`
   - d. Modify `dbBridge` to use `txBridge.EffectiveDB()`
   - e. Add `"tx"` namespace to `BuildGoJSONExtension` in `gojson_adapter.go`
   - f. Update `TestBuildGoJSONExtension_NoTxFunction` → `TestBuildGoJSONExtension_TxNamespace`
   - g. Update `TestBuildGoJSONExtension_Structure` → add `"tx"` to `expectedKeys`
   - h. Write `tx_bridge_test.go`
   - i. **THEN** follow addendum for goja changes (`proxy.go`, `vm.go`)
   - j. **THEN** follow addendum for yaegi changes (`symbols.go`, `vm.go`)
   - k. Update yaegi test `TestYaegiBridgeTxSwapsContext` → imperative
   - l. Add new goja/yaegi tx tests
3. **Task 3: Email Template Shorthand** — follow main instruction file exactly
4. **Docs** — follow main instruction file + addendum doc update list

---

## Quick Reference: What Changes Where

### From Main Instruction File:

| File | Action |
|------|--------|
| `engine/internal/runtime/bridge/query_builder.go` | NEW — fluent query builder |
| `engine/internal/runtime/bridge/query_builder_test.go` | NEW — unit tests (use fresh `testModelHandle` mock) |
| `engine/internal/runtime/bridge/tx_bridge.go` | NEW — imperative tx manager with `EffectiveDB()` pattern |
| `engine/internal/runtime/bridge/tx_bridge_test.go` | NEW — unit tests |
| `engine/internal/runtime/bridge/gojson_adapter.go` | MODIFY — add fluent methods, `"tx"` namespace, `email.template` |
| `engine/internal/runtime/bridge/gojson_adapter_test.go` | MODIFY — replace `NoTxFunction` test, update `expectedKeys`, add new tests |
| `engine/internal/runtime/bridge/context.go` | MODIFY — add `txBridge *txBridge` field |
| `engine/internal/runtime/bridge/factory.go` | MODIFY — create `txBridge`, pass to model/db factories |
| `engine/internal/runtime/bridge/model.go` | MODIFY — `repoFactory` uses `txBridge.EffectiveDB()` |
| `engine/internal/runtime/bridge/db.go` (or wherever `dbBridge` lives) | MODIFY — add `txb *txBridge` field, use `effectiveDB()` |

### From Addendum (ADDITIONAL):

| File | Action |
|------|--------|
| `engine/internal/runtime/embedded/goja/proxy.go` | REPLACE callback `"tx"` (line 83-90) → imperative `"tx": map[string]any{begin, commit, rollback}` |
| `engine/internal/runtime/embedded/goja/vm.go` | ADD `txOriginalBC`, `txGormTx`, `txTimeout` fields + cleanup in `Close()` |
| `engine/internal/runtime/embedded/yaegi/symbols.go` | REPLACE `"Tx"` callback (line 54-61) → `"Tx": reflect.ValueOf(func() *goTxProxy{...})` |
| `engine/internal/runtime/embedded/yaegi/symbols.go` | ADD `goTxProxy` struct with `Begin()`/`Commit()`/`Rollback()` methods |
| `engine/internal/runtime/embedded/yaegi/symbols.go` | ADD `txOriginalCtx`, `txGormTx` fields to `bridgeHolder` |
| `engine/internal/runtime/embedded/yaegi/vm.go` | ADD cleanup in Close() for dangling tx |
| `engine/internal/runtime/embedded/yaegi/yaegi_test.go` | UPDATE `TestYaegiBridgeTxSwapsContext` → `TestYaegiTxImperative` with `Tx().Begin()` |
| `engine/internal/runtime/embedded/goja/goja_test.go` | ADD new imperative tx tests |

### NOT Changed:

| File | Reason |
|------|--------|
| `engine/internal/runtime/plugin/` (Node.js/Python) | Already imperative — no changes |
| `engine/internal/runtime/bridge/tx.go` | Internal `txManager` — keep for engine use |
| `engine/internal/runtime/bridge/interfaces.go` | `TxManager` interface — internal, keep |
| `engine/internal/runtime/bridge/context.go` `Tx(fn)` method | Internal Go API — keep |

---

## Key Differences Between Runtimes (For Your Understanding)

| | go-json | goja | yaegi |
|---|---------|------|-------|
| **How bridge is built** | `BuildGoJSONExtension(bc)` — once, closures frozen | `buildBitcodeObject(bc)` — can rebuild at runtime | `bridgeHolder.ctx` — mutable pointer |
| **How tx swap works** | `txBridge.EffectiveDB()` queried on every DB call | Rebuild entire bridge: `v.rt.Set("bitcode", v.buildBitcodeObject(txCtx))` | Swap pointer: `h.ctx = txCtx` |
| **Why different** | Closures capture `bc` at build time, cannot swap | goja VM allows setting globals at any time | yaegi proxies call `h.get()` which returns current pointer |
| **Tx state lives in** | `txBridge` struct (field on `Context`) | `GojaVM` struct fields (`txOriginalBC`, `txGormTx`) | `bridgeHolder` struct fields (`txOriginalCtx`, `txGormTx`) |

---

## Final Verification

```bash
cd engine
go build ./...
go vet ./...
go test ./internal/runtime/bridge/ -v
go test ./internal/runtime/embedded/goja/ -v
go test ./internal/runtime/embedded/yaegi/ -v
go test ./internal/runtime/plugin/ -v  # ensure no regression
go test ./... -v  # full suite
```

ALL existing tests must pass. Tests that change:
- `TestBuildGoJSONExtension_NoTxFunction` → renamed to `TestBuildGoJSONExtension_TxNamespace`
- `TestBuildGoJSONExtension_Structure` → `expectedKeys` updated
- `TestYaegiBridgeTxSwapsContext` → rewritten as `TestYaegiTxImperative`

---

## Documents to Update After ALL Tasks Complete

| Document | Changes |
|----------|---------|
| `engine/AGENTS.md` | Bridge section: fluent API, unified imperative tx (all runtimes), email.template |
| `engine/docs/architecture.md` | Extends/overrides pattern documentation |
| `engine/docs/codebase.md` | New files (query_builder.go, tx_bridge.go) |
| `engine/docs/features/plugins.md` | Full bridge API reference with fluent + tx + email.template examples. Include per-runtime tx syntax. |
| `engine/docs/features/processes.md` | Note go-json as recommended runtime |
| `docs/features.md` | Root feature list update |
| `docs/plans/handoff-phase-4.5h-to-4.5k.md` | Section 3.2: update tx gotcha — callback REMOVED from goja/yaegi, imperative everywhere |
