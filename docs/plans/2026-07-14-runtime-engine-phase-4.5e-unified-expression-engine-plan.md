# Phase 4.5e — Unified Expression Engine: Implementation Plan

**Goal:** Replace 3 hand-rolled expression evaluators with go-json's ExprEngine (expr-lang/expr), upgrade record rules to expr-lang with AST-to-WHERE conversion, and expose a public `EvalExpr()` API.

**Architecture:** Single shared ExprLangEngine instance with compilation cache. Computed fields use scoped aggregate functions. Process/hook conditions use `EvalExprBool()` with truthiness coercion. Record rules use AST walker to produce recursive `FilterNode` trees converted to parameterized SQL.

**Tech Stack:** Go 1.24+, expr-lang/expr v1.17.8, go-json ExprEngine

**Design Doc:** `2026-07-14-runtime-engine-phase-4.5e-unified-expression-engine.md`
**Depends on:** Phase 4.5c complete

---

## Critical Path

```
                    ┌──► Task 2 (computed fields)
                    │
Task 1 (EvalExpr) ──┼──► Task 3 (process conditions) ──► Task 5 (DAG dedup)
                    │
                    ├──► Task 4 (hook conditions)
                    │
                    └──► Task 7 (AST-to-WHERE) ──► Task 8 (record rule service) ──► Task 9 (DB migration)
                         ▲
Task 6 (rule parser) ───┘

Task 10 (namespace validation) — independent, can start anytime

Tasks 11-12 (tests) — after Tasks 2-5, 7-8
Task 13 (docs) — after all
```

**Parallelism:** Tasks 2, 3, 4 can run in parallel after Task 1. Task 6 and Task 10 have no dependencies and can start immediately.

## Deferred

**Migration tool** (convert legacy `domain_filter` → `domain_filter_expr`): Deferred to a future task. Both formats coexist indefinitely. The migration tool is a convenience, not a blocker — old format continues to work via the legacy interpolation path.

---

## Batch 1: go-json Public API

### Task 1: EvalExpr Public API

**Files:** Create `packages/go-json/runtime/eval.go`, `packages/go-json/runtime/eval_test.go`
**Effort:** Small
**Depends on:** Phase 4.5c complete

**Steps:**
1. Create `runtime/eval.go` with shared `ExprLangEngine` singleton
2. Implement `EvalExpr(expression string, env map[string]any) (any, error)` — delegates to shared engine's `Eval()`
3. Implement `EvalExprFloat()` — calls `EvalExpr()` + coerce to float64 (handle int, float32, string-parseable)
4. Implement `EvalExprBool()` — calls `EvalExpr()` + truthiness coercion:
   - `bool` → as-is
   - `nil` → `false`
   - numeric `0` → `false`, non-zero → `true`
   - `""` → `false`, non-empty → `true`
   - empty array/map → `false`, non-empty → `true`
   - any other non-nil → `true`
5. Implement `ParseExpr()` — calls `parser.Parse()`, wraps result in `ExprTree{Root: tree.Node}`
6. Implement `ValidateExpr()` — calls `Compile()` without `Run()`, returns compile error or nil
7. Define `ExprTree` struct wrapping `ast.Node`
8. Write tests:
   - `TestEvalExpr_Arithmetic` — `2 + 3`, `10 * 5`
   - `TestEvalExpr_Comparison` — `a > b`, `a == b`
   - `TestEvalExpr_StringFunctions` — `upper("hello")`, `"abc" contains "b"`
   - `TestEvalExprBool_Truthiness` — nil, 0, "", empty array, non-empty string, etc.
   - `TestEvalExprFloat_Coercion` — int, float64, string "3.14"
   - `TestParseExpr_ValidExpression` — returns ExprTree with BinaryNode
   - `TestParseExpr_InvalidExpression` — returns error
   - `TestValidateExpr_Valid` — returns nil
   - `TestValidateExpr_Invalid` — returns error
   - `TestEvalExpr_CacheHit` — same expression compiled once

**Acceptance criteria:** `go test ./runtime/ -run TestEval -v` passes; API usable from engine without full VM

**Commit:** `feat(go-json): public EvalExpr API for engine expression evaluation`

---

## Batch 2: Computed Field Migration

### Task 2: Replace Computed Field Evaluator

**Files:** Rewrite `engine/internal/runtime/expression/hydrator.go`, create `engine/internal/runtime/expression/aggregate.go`, delete `engine/internal/runtime/expression/evaluator.go`
**Effort:** Medium
**Depends on:** Task 1

**Steps:**
1. Create `aggregate.go`:
   - Implement `resolveAggregateOrBuiltin(funcName string, arg any, collections map[string][]map[string]any) float64`
   - If `arg` is string with dot (e.g., `"lines.subtotal"`) → split into collection + field → aggregate over child records
   - If `arg` is `[]any` → delegate to standard sum/count/avg/min/max
   - Register as expr-lang functions via `expr.Function()` options
2. Rewrite `hydrator.go`:
   - Keep `Hydrator` struct, `SetTableNameResolver()`, `HydrateRecord()`, `HydrateRecords()` interfaces unchanged
   - Replace internal `Evaluate(expr, evalCtx)` calls with `runtime.EvalExpr(expr, env)` where `env` is built from record fields + aggregate functions
   - Build env per-record: all record fields as top-level keys + scoped aggregate functions
   - For child collections: load children (existing `loadChildren()` logic unchanged), register aggregate functions scoped to this evaluation
3. Delete `evaluator.go` (786 lines)
4. Update `app.go`:
   - `expression.NewHydrator(db, modelReg)` → update if constructor signature changes
   - Hydrator no longer needs custom `EvalContext` — uses `map[string]any` env
5. Verify `repository.go`, `router.go`, `renderer.go` — these call `SetHydrator()` and `HydrateRecord()`/`HydrateRecords()` which remain unchanged
6. Rewrite `evaluator_test.go`:
   - Same test cases (arithmetic, field references, computed formulas, aggregates, comparisons, boolean logic, built-in functions, if-function, nil fields, string concat, integer/string field values)
   - Change calls from `Evaluate(expr, evalCtx)` / `EvaluateFloat(expr, evalCtx)` to `runtime.EvalExpr(expr, env)` / `runtime.EvalExprFloat(expr, env)`
   - `EvalContext{Record, ChildCollections}` → `map[string]any` with aggregate functions

**Acceptance criteria:**
- All 17 existing evaluator tests pass with new engine
- `go test ./internal/runtime/expression/ -v` passes
- `evaluator.go` deleted
- Computed fields work end-to-end (manual test: create model with computed field, load record, verify computed value)

**Commit:** `refactor(engine): replace computed field evaluator with go-json ExprEngine`

---

## Batch 3: Process & Hook Condition Migration

### Task 3: Replace Process Condition Evaluator

**Files:** Modify `engine/internal/runtime/executor/steps/control.go`
**Effort:** Small
**Depends on:** Task 1

**Steps:**
1. Add `normalizeCondition()` function using targeted regex (`conditionVarRe`) per design doc §4.2
2. Keep `preprocessTranslations()` — move before `normalizeCondition()` in the pipeline
3. Rewrite `evaluateCondition()`:
   ```
   condition = preprocessTranslations(condition, execCtx.T)
   condition = normalizeCondition(condition)
   env = {input: execCtx.Input, ...execCtx.Variables}
   result = runtime.EvalExprBool(condition, env)
   ```
4. Keep `interpolateString()` for non-condition contexts (HTTP URLs, log messages, validation errors) — this remains string replacement, NOT expression evaluation
5. Keep `resolveVariable()` for `SwitchHandler` field resolution and `LoopHandler` over resolution — these resolve data references, not expressions
6. Update `steps_test.go`:
   - `TestIfHandler` — verify `{{input.total}} > 100` works (normalized to `input.total > 100`)
   - `TestIfHandler_GreaterThan` — verify the `" > "` bug is FIXED (no longer always true)
   - `TestInterpolateTranslation` — unchanged (preprocessor still works)
   - Add test: condition without `{{}}` works directly (`input.total > 100`)

**Acceptance criteria:**
- `" > "` bug fixed — `evaluateCondition("5 > 10", ...)` returns `false`
- `{{}}` backward compat — `evaluateCondition("{{input.x}} > 100", ...)` works
- `go test ./internal/runtime/executor/steps/ -v` passes

**Commit:** `fix(engine): replace buggy process condition evaluator with expr-lang`

---

### Task 4: Replace Hook Condition Evaluator

**Files:** Delete `engine/internal/runtime/hook/expr.go`, modify `engine/internal/runtime/hook/dispatcher.go`
**Effort:** Small
**Depends on:** Task 1

**Steps:**
1. Delete `hook/expr.go` (133 lines)
2. In `dispatcher.go`, rewrite `evaluateHandlerCondition()`:
   ```go
   func evaluateHandlerCondition(condition string, data map[string]any) bool {
       env := buildHookEnv(data)
       result, err := runtime.EvalExprBool(condition, env)
       if err != nil { return false }
       return result
   }
   ```
3. Implement `buildHookEnv()` — remap `__old` → `old`, `__session` → `session`
4. Update `dispatcher_test.go`:
   - `TestExprEvaluator_Basic` — same test, different engine
   - `TestExprEvaluator_OldData` — `old.status != status` works
   - `TestExprEvaluator_AndOr` — proper operator precedence now
   - `TestDispatcher_ConditionSkip` — unchanged (integration test)

**Acceptance criteria:**
- `hook/expr.go` deleted
- `&&`/`||` operator precedence correct (no more `strings.Split` fragility)
- `go test ./internal/runtime/hook/ -v` passes

**Commit:** `refactor(engine): replace hook condition evaluator with expr-lang`

---

### Task 5: Deduplicate DAG Edge Conditions

**Files:** Modify `engine/internal/runtime/executor/dag.go`
**Effort:** Small
**Depends on:** Task 3

**Steps:**
1. Remove `evaluateEdgeCondition()` (17 lines)
2. Remove `interpolateCondition()` (10 lines)
3. Remove `resolveConditionVar()` (14 lines)
4. Import and call the shared `evaluateCondition()` from `steps/control.go` (or extract to a shared package if needed)
5. Update `dag_test.go` — verify edge conditions still work with `{{input.is_vip}}` syntax

**Acceptance criteria:**
- 41 lines of duplicate code removed
- DAG edge conditions use same evaluator as sequential process
- `go test ./internal/runtime/executor/ -v` passes

**Commit:** `refactor(engine): deduplicate DAG edge condition evaluator`

---

## Batch 4: Record Rule Upgrade

### Task 6: Record Rule Expression Parser

**Files:** Modify `engine/internal/compiler/parser/model.go`
**Effort:** Small
**Depends on:** None (can start in parallel with Batch 2-3)

**Steps:**
1. Add `DomainFilterExpr` field to record rule struct in `model.go`:
   ```go
   DomainFilterExpr string `json:"domain_filter_expr,omitempty"`
   ```
2. Add `preprocessRuleExpr()` function per design doc §6.9
3. In record rule parsing, if `domain_filter_expr` is set, run preprocessor to normalize `{{}}` sugar

**Acceptance criteria:** `DomainFilterExpr` field parsed from JSON; `{{user.id}}` sugar converted to `ctx.user_id`

**Commit:** `feat(engine): add domain_filter_expr field to record rule parser`

---

### Task 7: AST-to-WHERE Converter

**Files:** Create `engine/internal/infrastructure/persistence/record_rule_expr.go`
**Effort:** Large
**Depends on:** Task 1, Task 6

**Steps:**
1. Define `FilterNode` and `FilterGroup` types per design doc §6.5
2. Define `RecordRuleContext` struct per design doc §6.3 — typed struct with `UserID`, `CompanyID`, `CompanyIDs`, `DepartmentID`, `DepartmentIDs`, `GroupIDs`, `Groups`, `Role`, `TenantID`, `Now`, `Today`
3. Implement `ExprToFilters` walker:
   - `Convert(node ast.Node) (*FilterNode, error)` — entry point
   - `walkNode()` — recursive: `&&` → AND group, `||` → OR group, comparison → leaf
   - `extractComparison()` — resolve field name (left) + value (right, with ctx resolution)
   - `extractInClause()` — handle `in` / `not in` operators
   - `resolveFieldName()` — extract `IdentifierNode.Value`, validate against model fields
   - `resolveValue()` — resolve `MemberNode` (ctx.user_id → concrete value), `IntegerNode`, `StringNode`, `BoolNode`, `ArrayNode`
4. Handle `contains`/`startsWith`/`endsWith` BinaryNode operators → SQL LIKE conversion:
   - `field contains "value"` → `WHERE field LIKE '%value%'` (BinaryNode with operator `"contains"`)
   - `field startsWith "value"` → `WHERE field LIKE 'value%'` (BinaryNode with operator `"startsWith"`)
   - `field endsWith "value"` → `WHERE field LIKE '%value'` (BinaryNode with operator `"endsWith"`)
   - Note: these are expr-lang **operators** (BinaryNode), not functions (CallNode)
5. Implement security validations per design doc §6.6:
   - Field name validation via `IsSafeFieldName()`
   - Context access whitelist (only `ctx.*`)
   - Reject `CallNode` (except whitelist: `len`). Note: `contains`/`startsWith`/`endsWith` are BinaryNode operators, handled in step 4.
   - Tautology detection (no field reference → error)
   - Empty array → deny-all filter
   - Nil context value → deny-all filter
6. Implement `FilterNodeToQuery()` — convert `FilterNode` tree to `Query` builder calls:
   - Leaf → `Query.Where(field, op, value)`
   - AND group → nested `Query.Where()` calls
   - OR group → `Query.OrWhere()` or `ConditionGroup{Connector: "OR"}`
7. Implement conditional rule handling (design doc §6.7):
   - Detect `ConditionalNode` (ternary) in AST
   - Evaluate condition part via `EvalExprBool()`
   - Convert the selected branch to filters
8. Set `MaxNodes` to 200 for record rule expressions

**Acceptance criteria:**
- `created_by == ctx.user_id` → `WHERE created_by = ?` with parameterized value
- `department_id in ctx.department_ids` → `WHERE department_id IN (?,?)`
- `(a == ctx.x || b == ctx.y) && c == ctx.z` → correct nested AND/OR groups
- `matches` operator → rejected with clear error
- `1 == 1` → rejected (tautology)
- Empty `ctx.department_ids` → deny-all filter
- All values parameterized (never string-concatenated to SQL)

**Commit:** `feat(engine): AST-to-WHERE converter for expr-lang record rules`

---

### Task 8: Record Rule Service Integration

**Files:** Modify `engine/internal/infrastructure/persistence/record_rule_service.go`
**Effort:** Medium
**Depends on:** Task 7

**Steps:**
1. Add `exprEngine` field to `RecordRuleService` (or use the global `runtime.EvalExpr`)
2. In `GetFilters()`, after loading rules from DB:
   - If rule has `DomainFilterExpr` → parse with `runtime.ParseExpr()`, convert with `ExprToFilters`, resolve ctx values
   - If rule has `DomainFilter` (legacy) → existing `parseDomainFilter()` + `InterpolateDomainFilters()` path
   - If both → `DomainFilterExpr` takes precedence
3. Build `RecordRuleContext` from user session data
4. Pass model field names to `ExprToFilters` for validation
5. Handle errors: if expression evaluation fails → log error, return deny-all filter (fail-closed)
6. Update middleware callers (`record_rule.go`, `graphql/resolver.go`, `websocket/crud.go`) to pass full user context

**Acceptance criteria:**
- New `domain_filter_expr` rules produce correct SQL WHERE clauses
- Old `domain_filter` rules continue to work unchanged
- Mixed old/new rules on same model combine correctly (AND global, OR group)
- Expression errors → deny-all (fail-closed, not fail-open)

**Commit:** `feat(engine): integrate expr-lang record rules into RecordRuleService`

---

### Task 9: Database Migration

**Files:** Create migration in `engine/internal/infrastructure/persistence/`
**Effort:** Small
**Depends on:** Task 8

**Steps:**
1. Add `domain_filter_expr TEXT` column to `record_rule` table (nullable)
2. Handle all 3 DB drivers (SQLite, PostgreSQL, MySQL)
3. No data migration needed — new column is opt-in

**Acceptance criteria:** Column exists after migration; existing record rules unaffected

**Commit:** `feat(engine): add domain_filter_expr column to record_rule table`

---

### Task 10: Reserved Namespace Validation

**Files:** Modify `engine/internal/domain/model/registry.go` or model registration path
**Effort:** Small
**Depends on:** None (can start in parallel)

**Steps:**
1. Define `ReservedNamespaces = []string{"ctx", "input", "old", "session", "self"}`
2. In model registration, check each field name against reserved list
3. New models → hard error with clear message
4. Existing models (loaded at startup) → warning log, model continues to work
5. Add test: model with field `ctx` → error; model with field `context` → OK

**Acceptance criteria:** `ctx` field rejected for new models; warning for existing; non-reserved fields unaffected

**Commit:** `feat(engine): validate model fields against reserved expression namespaces`

---

## Batch 5: Tests & Documentation

### Task 11: Record Rule Expression Tests

**Files:** Create `engine/internal/infrastructure/persistence/record_rule_expr_test.go`
**Effort:** Medium
**Depends on:** Task 7

**Steps:**
1. Test simple comparisons: `created_by == ctx.user_id`
2. Test `in` clause: `department_id in ctx.department_ids`
3. Test AND: `a == ctx.x && b == ctx.y`
4. Test OR: `a == ctx.x || b == ctx.y`
5. Test nested: `(a == ctx.x || b == ctx.y) && c == ctx.z`
6. Test conditional: `ctx.role == 'admin' || created_by == ctx.user_id`
7. Test security: `matches` rejected, tautology rejected, unknown field rejected
8. Test empty array → deny-all
9. Test nil context → deny-all
10. Test `{{}}` sugar preprocessing
11. Test backward compat: old `domain_filter` still works alongside new `domain_filter_expr`

**Acceptance criteria:** All tests pass; covers all edge cases from design doc §9

**Commit:** `test(engine): comprehensive record rule expression tests`

---

### Task 12: Integration Tests

**Files:** Update `engine/tests/` integration tests if they exist
**Effort:** Small
**Depends on:** Tasks 2-5, 8

**Steps:**
1. Verify computed fields work end-to-end with new engine
2. Verify process conditions work with `{{}}` syntax
3. Verify hook conditions work
4. Verify record rules work with both old and new format
5. Run full test suite: `go test ./... -v`

**Acceptance criteria:** `go test ./... -v` passes with zero failures

**Commit:** `test(engine): verify unified expression engine integration`

---

### Task 13: Documentation Updates

**Files:** 6 documentation files per design doc §8.6
**Effort:** Small
**Depends on:** All previous tasks

**Steps:**
1. `engine/docs/features/models.md` — update computed/formula section: now uses expr-lang, full function list available
2. `engine/docs/features/processes.md` — update condition evaluation: `{{}}` still works, expr-lang syntax also accepted, `" > "` bug fixed
3. `engine/docs/features/security.md` — add `domain_filter_expr` section: `ctx.*` namespace, examples, migration from `{{}}` format
4. `engine/docs/architecture.md` — update expression engine section: single ExprLangEngine, no more custom evaluators
5. `engine/AGENTS.md` — update expression engine reference
6. `packages/go-json/AGENTS.md` — add `EvalExpr` API documentation

**Acceptance criteria:** All docs accurate and consistent with implementation

**Commit:** `docs(engine): update documentation for unified expression engine`

---

## Task Summary

| # | Task | Effort | Depends On | Files |
|---|------|--------|------------|-------|
| 1 | EvalExpr Public API | Small | Phase 4.5c | `packages/go-json/runtime/eval.go` |
| 2 | Replace Computed Field Evaluator | Medium | Task 1 | `engine/.../expression/` (3 files) |
| 3 | Replace Process Condition Evaluator | Small | Task 1 | `engine/.../executor/steps/control.go` |
| 4 | Replace Hook Condition Evaluator | Small | Task 1 | `engine/.../hook/` (2 files) |
| 5 | Deduplicate DAG Edge Conditions | Small | Task 3 | `engine/.../executor/dag.go` |
| 6 | Record Rule Expression Parser | Small | — | `engine/.../compiler/parser/model.go` |
| 7 | AST-to-WHERE Converter | Large | Task 1, 6 | `engine/.../persistence/record_rule_expr.go` |
| 8 | Record Rule Service Integration | Medium | Task 7 | `engine/.../persistence/record_rule_service.go` |
| 9 | Database Migration | Small | Task 8 | migration file |
| 10 | Reserved Namespace Validation | Small | — | `engine/.../domain/model/` |
| 11 | Record Rule Expression Tests | Medium | Task 7 | test file |
| 12 | Integration Tests | Small | Tasks 2-5, 8 | `engine/tests/` |
| 13 | Documentation Updates | Small | All | 6 doc files |
