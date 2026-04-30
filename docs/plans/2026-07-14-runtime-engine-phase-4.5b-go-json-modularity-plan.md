# Phase 4.5b — go-json Modularity: Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Add struct/methods, import system, parallel execution, and extended stdlib to the go-json engine built in Phase 4.5a.

**Architecture:** Extends `packages/go-json/`. Structs are first-class types with methods and `self` binding. Import system uses alias-as-namespace. Parallel execution uses goroutines with Go context cancellation. Stdlib Layer 2 extended with map, datetime, encoding, crypto, format functions.

**Tech Stack:** Go 1.24+, expr-lang/expr v1.16+, crypto/md5, crypto/sha256, crypto/hmac, net/url, github.com/google/uuid

**Design Doc:** `2026-07-14-runtime-engine-phase-4.5b-go-json-modularity.md`
**Decisions:** `2026-04-28-go-json-brainstorming-design.md` (OQ-2, OQ-5, OQ-6, HP-1, HP-2, §9 Struct Methods)
**Depends on:** Phase 4.5a complete

---

## Critical Path

```
Task 1 (struct def) ──► Task 3 (construction) ──► Task 5 (methods) ──► Task 8 (invocation)
  │                       │                                               │
  └─► Task 2 (type reg) ─┘                                               │
  └─► Task 4 (field access) ──► Task 9 (deep mutation)                   │
                                                                          │
Task 10 (import parser) ──► Task 11 (resolver) ──► Task 12 (circular) ──► Task 13 (barrel)
                                                                          │
Task 14-18 (stdlib) ─────────────────────────────────────────────────────┘
                                                                          │
Task 19 (parallel parser) ──► Task 20 (parallel engine) ──► Task 21 (error modes)
                                                                          │
Task 22-23 (nullable) ───────────────────────────────────────────────────┘
                                                                          │
                                                                    Task 24-29 (tests)
```

---

## Batch 1: Struct System

### Task 1: Struct Definition Parser

**Files:** Modify `lang/ast.go`, `lang/parser.go`
**Effort:** Medium
**Depends on:** Phase 4.5a complete

**Steps:**
1. Add AST nodes to `ast.go`:
   ```go
   type StructDef struct {
       NodeMeta
       Name    string
       Frozen  bool                    `json:"frozen,omitempty"`
       Fields  map[string]*FieldDef    `json:"fields"`
       Methods map[string]*MethodDef   `json:"methods,omitempty"`
   }

   type FieldDef struct {
       Type       string  // "string", "int", "Person", "[]string", "?Address"
       Default    any     // nil if required
       HasDefault bool
   }

   type MethodDef struct {
       NodeMeta
       Params  map[string]any `json:"params,omitempty"`
       Returns string         `json:"returns,omitempty"`
       Steps   []Node         `json:"steps"`
   }
   ```
2. Add struct parsing to `parser.go` — parse `structs` block, handle short form (`"field": "type"`) and long form (`"field": {"type": "T", "default": "expr"}`)
3. Parse `frozen: true` flag
4. Parse nested struct references (`"address": "Address"`)
5. Parse nullable fields (`"nickname": "?string"`)
6. Parse array of struct fields (`"addresses": "[]Address"`)

**Edge cases:**
- Field type references struct not yet defined (forward reference) → defer validation to Task 2
- `frozen` missing → default false (mutable)
- Field with default that is expression string → evaluate at construction time, not parse time

**Acceptance criteria:** Parse struct definitions from JSON, build StructDef AST nodes

**Commit:** `feat(go-json): struct definition parser with frozen support`

---

### Task 2: Struct Type Registration

**Files:** Modify `lang/types.go`, `lang/compiler.go`
**Effort:** Medium
**Depends on:** Task 1

**Steps:**
1. Add struct type registry to compiler — map of struct name → StructDef
2. Validate field types — check that referenced types exist (built-in or other structs)
3. Handle forward references — two-pass: first register all struct names, then validate field types
4. Detect circular struct definitions (`A has field of type A` → allowed for nullable, error for non-nullable)
5. Register struct types in expr-lang environment

**Edge cases:**
- Struct A has field `"b": "B"`, struct B has field `"a": "?A"` → allowed (nullable breaks cycle)
- Struct A has field `"a": "A"` (non-nullable) → compile error (infinite size)
- Imported struct references → resolved after import system (Task 11)

**Acceptance criteria:** All struct field types validated, forward references resolved

**Commit:** `feat(go-json): struct type registration with forward reference resolution`

---

### Task 3: Struct Construction (`new` + `with`)

**Files:** Modify `lang/vm.go`
**Effort:** Medium
**Depends on:** Task 1, 2

**Steps:**
1. Implement `executeNew(structName, withArgs)` in VM:
   - Look up struct definition
   - Evaluate `with` args (recursive — all string values are expressions)
   - Validate required fields provided (fields without defaults must be in `with`)
   - Apply defaults for missing optional fields
   - Type-check field values against field types
   - Return constructed struct as `map[string]any` with `_type` metadata
2. Handle nested construction: `{"new": "Address", "with": {...}}` inside `with`
3. Handle `with` value types: string → expression, number/bool/null → literal, map with `new` → nested construction, map without `new` → literal object

**Edge cases:**
- Missing required field → compile error with field name
- Wrong type for field → runtime error with expected vs actual
- `with` value `"age": 30` (number, not string) → literal, not expression
- Nested `new` inside `with` → recursive construction
- Default value is expression string → evaluate at construction time

**Acceptance criteria:**
- Construct struct with all fields
- Apply defaults
- Reject missing required fields
- Type-check field values

**Commit:** `feat(go-json): struct construction with new + with`

---

### Task 4: Struct Field Access

**Files:** Modify `lang/vm.go`, `lang/expr_engine.go`
**Effort:** Medium
**Depends on:** Task 3

**Steps:**
1. Register struct instances in expr-lang environment so `person.name`, `person.address.city` work in expressions
2. Struct instances are `map[string]any` — expr-lang already supports map field access via dot notation
3. Ensure optional chaining works: `person?.address?.city`
4. Ensure bracket notation works: `person["name"]`

**Edge cases:**
- Access field on nil struct → runtime error (use `?.` for safe access)
- Access non-existent field → runtime error with "did you mean" suggestion
- Nested struct access `person.address.city` → chain of map lookups

**Acceptance criteria:** All field access patterns work in expressions

**Commit:** `feat(go-json): struct field access in expressions`

---

## Batch 2: Struct Methods

### Task 5: Method Definition Parser

**Files:** Modify `lang/parser.go`
**Effort:** Small
**Depends on:** Task 1

**Steps:**
1. Parse `methods` block inside struct definition
2. Methods have same structure as functions: params, returns, steps
3. `self` is NOT declared in params — it's implicit

**Commit:** `feat(go-json): method definition parser`

---

### Task 6: Method `self` Binding

**Files:** Modify `lang/vm.go`
**Effort:** Medium
**Depends on:** Task 5

**Steps:**
1. When method is called, create scope with `self` bound to struct instance
2. `self.field` reads field value
3. `self.method()` calls another method on same instance
4. `self` cannot be reassigned: `{"set": "self", ...}` → compile error
5. `self` can be passed to functions: function receives copy (isolated scope)

**Edge cases:**
- `self` in nested function call → `self` not available (function has isolated scope)
- `self.method()` that calls `self.otherMethod()` → works (same instance)
- Method modifies `self.field`, then reads it → sees updated value

**Acceptance criteria:** `self` binding works correctly in all method contexts

**Commit:** `feat(go-json): method self binding with scope isolation`

---

### Task 7: Method Mutation + Frozen Check

**Files:** Modify `lang/vm.go`, `lang/compiler.go`
**Effort:** Medium
**Depends on:** Task 6

**Steps:**
1. `set "self.field"` mutates struct instance in-place
2. For `frozen: true` structs: compiler scans all methods for `set "self.*"` → compile error
3. Read-only methods on frozen structs → allowed
4. Methods returning new instance on frozen structs → allowed

**Edge cases:**
- Frozen struct method calls non-frozen struct method on a field → allowed (field's struct is not frozen)
- Frozen struct method uses `set "self.nested.field"` → compile error (any self mutation blocked)

**Acceptance criteria:**
- Mutation works on mutable structs
- Compile error on frozen struct mutation
- Read-only methods work on frozen structs

**Commit:** `feat(go-json): method mutation with frozen struct compile-time check`

---

### Task 8: Method Invocation

**Files:** Modify `lang/vm.go`, `lang/expr_engine.go`
**Effort:** Medium
**Depends on:** Task 6, 7

**Steps:**
1. Expression-level: `person.fullInfo()`, `person.greet('Hello')` — register methods in expr-lang env
2. Step-level: `{"call": "person.birthday"}`, `{"let": "msg", "call": "person.greet", "with": {"greeting": "'Hello'"}}`
3. Method chaining: `person.withName('Bob').withAge(30).fullInfo()` — natural via expr-lang
4. Parse dot in call target: `"call": "person.birthday"` → object = "person", method = "birthday"

**Edge cases:**
- Call method on nil variable → runtime error
- Call undefined method → error with "did you mean" suggestion
- Method chaining with mutation → each call modifies and returns same instance
- Method chaining with immutable pattern → each call returns new instance

**Acceptance criteria:** Methods callable at both expression and step level, chaining works

**Commit:** `feat(go-json): method invocation — expression and step level with chaining`

---

## Batch 3: Deep Property Mutation

### Task 9: Nested Property Mutation

**Files:** Modify `lang/vm.go`
**Effort:** Medium
**Depends on:** Task 4

**Steps:**
1. Implement `setNestedProperty(path, value, stepIndex)` — parse dot/bracket path, traverse, mutate leaf
2. Support: `set "person.address.city"`, `set "items[0].name"`, `set "data['key']"`
3. Parse path into segments: `["person", "address", "city"]` or `["items", 0, "name"]`
4. Traverse object, error on nil intermediate or wrong type

**Edge cases:**
- `set "a.b.c"` where `a.b` is nil → runtime error: "cannot set property 'c' on nil"
- `set "a[5]"` where array has 3 elements → runtime error: "index 5 out of bounds"
- `set "a.b"` where `a` is int → runtime error: "cannot set property on int"
- `set "a[0].b[1].c"` → mixed dot/bracket traversal

**Acceptance criteria:** All nested mutation patterns work with clear error messages

**Commit:** `feat(go-json): nested property mutation with dot and bracket notation`

---

## Batch 4: Import System

### Task 10: Import Parser

**Files:** Modify `lang/parser.go`
**Effort:** Small
**Depends on:** Phase 4.5a

**Steps:**
1. Parse `import` block — map of alias → path
2. Detect path types: relative (`./`), stdlib (`stdlib:`), extension (`ext:`), I/O (`io:`)
3. Store in Program AST

**Commit:** `feat(go-json): import path parser with type detection`

---

### Task 11: Import Resolver

**Files:** Create `lang/import.go`
**Effort:** Large
**Depends on:** Task 10

**Steps:**
1. Implement `ResolveImports(program, basePath)`:
   - For each import, resolve path type
   - Relative: resolve against current file directory, read + parse file
   - stdlib: load from built-in registry
   - ext: load from host-injected extensions
   - io: load from I/O module registry
2. Extract exportable items from imported file: structs + functions only
3. Register under alias in current scope: `alias.StructName`, `alias.functionName`
4. Handle `.json` and `.jsonc` files transparently

**Edge cases:**
- File not found → compile error with path
- File has JSON syntax error → compile error with import chain
- Imported file has compile error → propagate with import chain context
- Alias collision → compile error
- Diamond import (A→B, A→C, B→D, C→D) → D loaded once, cached

**Acceptance criteria:** Import, resolve, and use structs/functions from other files

**Commit:** `feat(go-json): import resolver with file loading and alias scoping`

---

### Task 12: Circular Import Detection

**Files:** Modify `lang/import.go`
**Effort:** Small
**Depends on:** Task 11

**Steps:**
1. Maintain import stack during resolution
2. Before loading file, check if already in stack → cycle detected
3. Report full cycle path in error message

**Edge cases:**
- A→B→A (direct cycle)
- A→B→C→A (indirect cycle)
- A→B, A→C, B→D, C→D (diamond — NOT a cycle, D loaded once)

**Acceptance criteria:** Cycles detected with clear error, diamonds handled correctly

**Commit:** `feat(go-json): circular import detection`

---

### Task 13: Re-export / Barrel Files

**Files:** Modify `lang/import.go`, `lang/parser.go`
**Effort:** Small
**Depends on:** Task 11

**Steps:**
1. Support `{"alias": "imported.Type"}` in structs block — re-export imported struct
2. Support `{"alias": "imported.func"}` in functions block — re-export imported function
3. Allow index.json files that aggregate exports from sub-files

**Acceptance criteria:** Barrel files work for re-exporting

**Commit:** `feat(go-json): barrel file re-export support`

---

## Batch 5: Stdlib Layer 2 Extensions

### Task 14: Map Functions

**Files:** Create `stdlib/maps.go`
**Effort:** Small
**Depends on:** Phase 4.5a stdlib registry

**Functions (5):** `has(map, key)`, `getIn(obj, dotPath [, separator])`, `merge(a, b)`, `pick(map, keys)`, `omit(map, keys)`

**Note:** `keys`, `values`, `toPairs`, `fromPairs` already in expr-lang. `get` renamed to `getIn` to avoid collision with expr-lang built-in.

**Edge cases:**
- `getIn(map, "a.b.c")` — dot path traversal, nil intermediate → return nil
- `merge(a, b)` — shallow merge, b overrides a
- `pick(map, [])` → empty map
- `omit(map, [])` → original map

**Commit:** `feat(go-json): stdlib maps — has, get, merge, pick, omit`

---

### Task 15: DateTime Functions

**Files:** Create `stdlib/datetime.go`
**Effort:** Small
**Depends on:** Phase 4.5a stdlib registry

**Functions (3):** `formatDate(dt, format)`, `addDuration(dt, dur)`, `diffDates(a, b)`

**Note:** `now`, `date`, `duration`, `timezone` already in expr-lang. `year/month/day/hour/minute` available as methods on date objects via expr-lang.

**Edge cases:**
- `addDuration(date, "invalid")` → error
- `diffDates(a, b)` where a < b → negative duration
- Format strings: Go format (`2006-01-02`) vs common format (`YYYY-MM-DD`) — use Go format, document clearly

**Commit:** `feat(go-json): stdlib datetime — formatDate, addDuration, diffDates`

---

### Task 16: Encoding Functions

**Files:** Create `stdlib/encoding.go`
**Effort:** Small
**Depends on:** Phase 4.5a stdlib registry

**Functions (2):** `urlEncode(str)`, `urlDecode(str)`

**Note:** `toJSON`, `fromJSON`, `toBase64`, `fromBase64` already in expr-lang.

**Commit:** `feat(go-json): stdlib encoding — urlEncode, urlDecode`

---

### Task 17: Crypto Functions

**Files:** Create `stdlib/crypto.go`
**Effort:** Small
**Depends on:** Phase 4.5a stdlib registry

**Functions (4):** `crypto.sha256(str)`, `crypto.md5(str)`, `crypto.uuid()`, `crypto.hmac(str, key, algo)`

**Namespaced** with `crypto.` prefix — registered as `crypto.sha256` etc.

**Edge cases:**
- `crypto.hmac(str, key, "sha512")` — support sha256 (default) and sha512
- `crypto.hmac(str, key, "invalid")` → error

**Commit:** `feat(go-json): stdlib crypto — sha256, md5, uuid, hmac (namespaced)`

---

### Task 18: Format Function

**Files:** Create `stdlib/fmt.go`
**Effort:** Small
**Depends on:** Phase 4.5a stdlib registry

**Functions (1):** `sprintf(format, args...)`

**Commit:** `feat(go-json): stdlib format — sprintf`

---

## Batch 6: Parallel Execution

### Task 19: Parallel Step Parser

**Files:** Modify `lang/parser.go`, `lang/ast.go`
**Effort:** Small
**Depends on:** Phase 4.5a

**Steps:**
1. Add `ParallelNode` to AST:
   ```go
   type ParallelNode struct {
       NodeMeta
       Branches map[string][]Node `json:"parallel"`
       Join     string            `json:"join,omitempty"`     // "all" (default), "any", "settled"
       OnError  string            `json:"on_error,omitempty"` // "cancel_all" (default), "continue", "collect"
       Into     string            `json:"into,omitempty"`
   }
   ```
2. Parse parallel step in parser

**Commit:** `feat(go-json): parallel step parser`

---

### Task 20: Parallel Execution Engine

**Files:** Modify `lang/vm.go`
**Effort:** Large
**Depends on:** Task 19

**Steps:**
1. Implement `executeParallel(n *ParallelNode)`:
   - Create child context with cancellation
   - For each branch: spawn goroutine with own scope (child of parent, read-only parent access)
   - Collect results into `into` variable
   - Wait for all/any based on `join` mode
2. **Compile-time check**: scan parallel branch steps for `set` targeting parent scope variables → compile error
3. Each branch scope: can `Declare` new variables, can `Get` from parent (read-only), CANNOT `Set` parent variables

**Edge cases:**
- Branch panics → recover, treat as error
- All branches complete before timeout → normal
- Timeout during parallel → cancel context, abort remaining branches
- Branch returns value → stored in results map
- Branch has no return → result is nil
- Empty branches map → no-op, into = empty map

**Acceptance criteria:**
- Branches execute concurrently
- Parent scope read-only enforced
- Results collected correctly
- Context cancellation works

**Commit:** `feat(go-json): parallel execution engine with scope isolation`

---

### Task 21: Parallel Error Handling

**Files:** Modify `lang/vm.go`
**Effort:** Medium
**Depends on:** Task 20

**Steps:**
1. Implement 3 modes:
   - `cancel_all` (default): first error → cancel context → abort others → propagate error
   - `continue`: errors → nil in results, log error → continue others → success with partial results
   - `collect`: errors → error objects as values in results → success → user checks per-branch
2. Use Go `context.WithCancel` for `cancel_all` mode

**Edge cases:**
- `cancel_all` but branch already completed → keep its result
- `continue` with all branches failing → results all nil, no error propagated
- `collect` → user must check `type(results.branch)` to detect errors

**Acceptance criteria:** All 3 modes work correctly

**Commit:** `feat(go-json): parallel error handling — cancel_all, continue, collect`

---

## Batch 7: Nullable Types

### Task 22: Nullable Type Support

**Files:** Modify `lang/types.go`, `lang/compiler.go`
**Effort:** Small
**Depends on:** Phase 4.5a type system

**Steps:**
1. `?T` types in field definitions, function params, variable declarations
2. Nil assignment only allowed for nullable types
3. Non-nullable + nil → compile error

**Commit:** `feat(go-json): nullable type support`

---

### Task 23: Optional Chaining Verification

**Files:** Create tests
**Effort:** Small
**Depends on:** Task 22

**Steps:**
1. Verify `a?.b?.c` works with go-json struct types
2. Already supported by expr-lang — just need integration tests

**Commit:** `test(go-json): optional chaining with struct types`

---

## Batch 8: Tests

### Task 24-29: Test Suites

**Effort:** Large (combined)

| Task | Scope | Files |
|------|-------|-------|
| 24 | Struct CRUD tests | `lang/struct_test.go` |
| 25 | Method tests (self, mutation, frozen, chaining) | `lang/method_test.go` |
| 26 | Import tests (relative, circular, diamond, barrel) | `lang/import_test.go` |
| 27 | Parallel tests (modes, scope isolation, timeout) | `lang/parallel_test.go` |
| 28 | Stdlib Tier 2 tests | `stdlib/*_test.go` |
| 29 | Integration tests (complex programs) | `lang/integration_4.5b_test.go` |

**Commit:** `test(go-json): Phase 4.5b comprehensive tests`

---

## Summary

| Task | Description | Effort | Batch |
|------|-------------|--------|-------|
| 1 | Struct definition parser | Medium | 1 |
| 2 | Struct type registration | Medium | 1 |
| 3 | Struct construction | Medium | 1 |
| 4 | Struct field access | Medium | 1 |
| 5 | Method definition parser | Small | 2 |
| 6 | Method self binding | Medium | 2 |
| 7 | Method mutation + frozen | Medium | 2 |
| 8 | Method invocation | Medium | 2 |
| 9 | Deep property mutation | Medium | 3 |
| 10 | Import parser | Small | 4 |
| 11 | Import resolver | Large | 4 |
| 12 | Circular import detection | Small | 4 |
| 13 | Barrel files | Small | 4 |
| 14 | Stdlib maps | Small | 5 |
| 15 | Stdlib datetime | Small | 5 |
| 16 | Stdlib encoding | Small | 5 |
| 17 | Stdlib crypto | Small | 5 |
| 18 | Stdlib format | Small | 5 |
| 19 | Parallel parser | Small | 6 |
| 20 | Parallel engine | Large | 6 |
| 21 | Parallel error handling | Medium | 6 |
| 22 | Nullable types | Small | 7 |
| 23 | Optional chaining verification | Small | 7 |
| 24-29 | Tests | Large | 8 |

**Total: 29 tasks across 8 batches**
**Critical path: Task 1 → 3 → 6 → 8 → 20 → 21 → tests**
**Estimated total effort: ~2-3 weeks for one developer**
