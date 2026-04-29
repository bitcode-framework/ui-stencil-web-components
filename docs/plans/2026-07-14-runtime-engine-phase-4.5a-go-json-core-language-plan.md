# Phase 4.5a — go-json Core Language: Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Build the go-json core language engine from scratch — a standalone JSON/JSONC programming language embeddable in Go, powered by expr-lang/expr for expression evaluation.

**Architecture:** New Go package `packages/go-json/`. Pipeline: JSONC pre-process → JSON parse → AST → compile (type check + expr compile) → immutable Program → VM execution with debug hooks. Expression evaluation delegated to expr-lang via ExprEngine abstraction layer. Stdlib split into Layer 1 (expr-lang built-ins, ~68 functions, zero work) and Layer 2 (go-json additions, ~19 functions).

**Tech Stack:** Go 1.24+, expr-lang/expr v1.16+, encoding/json (stdlib), sync (stdlib), context (stdlib), time (stdlib), regexp (stdlib), math/rand (stdlib)

**Design Doc:** `2026-07-14-runtime-engine-phase-4.5a-go-json-core-language.md`
**Decisions:** `2026-04-28-go-json-brainstorming-design.md`

---

## Critical Path

```
Task 1 (scaffold)
  │
  ├─► Task 2 (AST) ──► Task 8 (parser) ──► Task 10 (compiler) ──► Task 11 (VM)
  ├─► Task 3 (errors)                                                │
  ├─► Task 4 (preprocess) ──► Task 8                                 │
  ├─► Task 5 (scope) ──► Task 11                                     │
  ├─► Task 6 (types) ──► Task 10                                     │
  └─► Task 7 (expr engine) ──► Task 10                               │
                                                                      │
  Task 9 (program) ──► Task 10                                        │
  Task 12 (debugger) ──► Task 11                                      │
                                                                      ▼
  Task 13 (limits) ──► Task 16 (runtime API) ──► Task 17-21 (stdlib)
  Task 14 (context) ──► Task 16                     │
  Task 15 (logger) ──► Task 16                      ▼
                                              Task 22-25 (tests)
```

**Bottleneck:** Task 8 (parser) and Task 11 (VM) are the largest tasks and on the critical path.

---

## Batch 1: Foundation (no dependencies)

### Task 1: Project Scaffold

**Files:** Create `packages/go-json/` directory structure + `go.mod`
**Effort:** Small
**Depends on:** Nothing

**Steps:**
1. Create directory structure:
   ```
   packages/go-json/
   ├── go.mod
   ├── lang/
   ├── stdlib/
   ├── runtime/
   ├── cmd/go-json/
   ├── codegen/
   ├── io/
   └── testdata/
   ```
2. Initialize `go.mod`:
   ```
   module github.com/bitcode-framework/go-json
   go 1.24
   require github.com/expr-lang/expr v1.16.9
   ```
3. Create placeholder `cmd/go-json/main.go`

**Acceptance criteria:**
- `go build ./...` succeeds
- `go vet ./...` passes

**Commit:** `feat(go-json): scaffold project structure and go.mod`

---

### Task 2: AST Node Types

**Files:** Create `lang/ast.go`
**Effort:** Medium
**Depends on:** Task 1

**Steps:**
1. Define `NodeMeta` (Comment, Comments, StepIndex) and `Node` interface
2. Define `Program` node (Name, GoJSON, Input, Imports, Structs, Functions, Steps, Limits)
3. Define all step nodes:
   - `LetNode` — with Name, Type, Value/Expr/With, Call shorthand, New shorthand
   - `SetNode` — with Target (dot path), Value/Expr/With
   - `IfNode` — with Condition, Then, Elif ([]ElifBlock), Else
   - `SwitchNode` — with Expr, Cases map
   - `ForNode` — with Variable, In/Range, Index, Steps
   - `WhileNode` — with Condition, Steps
   - `BreakNode`, `ContinueNode`
   - `ReturnNode` — overloaded: string (expr), map (with/value/expr)
   - `CallNode` — with Function, With
   - `TryNode` — with Try, Catch (As + Steps), Finally
   - `ErrorNode` — overloaded: string (message expr), map (code/message/details)
   - `LogNode` — overloaded: string (simple), map (message/level/data)
   - `CommentNode` — standalone `_c` only
4. Define `FuncDef` (Params, Returns, Steps)
5. Implement `nodeType()` and `Meta()` for all types

**Edge cases:**
- `return` can be string, map, number, bool, null — parser must detect
- `let` + `call` shorthand: `{"let": "x", "call": "fn", "with": {...}}`
- `_c` as array of strings for multi-line comments

**Acceptance criteria:** All node types compile, `go vet` passes

**Commit:** `feat(go-json): define AST node types for all step types`

---

### Task 3: Error Types

**Files:** Create `lang/errors.go`
**Effort:** Medium
**Depends on:** Task 1

**Steps:**
1. Define `ErrorCategory` (compile, runtime, limit, io), `StackFrame`, `GoJSONError` struct with all fields (Code, Title, Category, Message, Fix, Suggestion, Step, Function, Program, Context, Stack)
2. Implement `Error()` — full formatted multi-line output
3. Implement `JSON()` — structured map for visual editor
4. Implement `Short()` — one-line summary
5. Implement `levenshtein()` distance function
6. Implement `SuggestSimilar(target, candidates, maxResults, maxDistance)` — returns similar names sorted by distance
7. Create constructor helpers: `CompileError()`, `RuntimeError()`, `LimitError()`
8. Create fluent builder methods: `WithFix()`, `WithSuggestions()`, `WithContext()`, `InFunction()`, `InProgram()`, `WithStack()`

**Edge cases:**
- Levenshtein with empty strings → return length of other string
- Unicode variable names → use rune-based comparison
- Max distance 3 to avoid false positives on short names

**Acceptance criteria:**
- `GoJSONError` implements `error` interface
- `SuggestSimilar("user_name", ["username", "user_id", "email"], 3, 3)` returns `["username", "user_id"]`

**Commit:** `feat(go-json): error types with enrichment, suggestions, and structured output`

---

### Task 4: JSONC Pre-processor

**Files:** Create `lang/preprocess.go`, `lang/preprocess_test.go`
**Effort:** Small
**Depends on:** Task 1

**Steps:**
1. Implement `StripComments(input []byte) []byte`:
   - Track string context (inside `"..."`) — don't strip comments in strings
   - Handle escaped quotes `\"` inside strings
   - Strip `//` line comments (to end of line)
   - Strip `/* */` block comments
   - Strip trailing commas before `]` and `}`
2. Write comprehensive tests:
   - No comments → unchanged
   - Line comment → stripped
   - Block comment → stripped
   - Comment inside string → preserved
   - Escaped quotes → handled correctly
   - Trailing commas → removed
   - Empty input → empty output
   - Unterminated block comment → strip to end

**Edge cases:**
- `//` inside `"url": "https://example.com"` → must NOT strip
- `/* */` inside string → must NOT strip
- Nested `*/` inside string → must NOT strip
- Multiple trailing commas `[1,,,]` → strip all
- Whitespace between comma and `]`/`}` → still strip

**Acceptance criteria:** `go test ./lang/ -run TestStripComments -v` passes

**Commit:** `feat(go-json): JSONC pre-processor — strip comments and trailing commas`

---

## Batch 2: Core Engine (depends on Batch 1)

### Task 5: Scope System

**Files:** Create `lang/scope.go`
**Effort:** Medium
**Depends on:** Task 1

**Steps:**
1. Define `VarInfo` (Value, Type, Declared) and `Scope` struct (vars map, parent, name, mutex)
2. Implement `Declare(name, value, type)` — error if already exists in current scope
3. Implement `Get(name)` — search up scope chain, return value + type + found
4. Implement `Set(name, value, newType)` — search up chain, type compatibility check
5. Implement `Has(name)` — current scope only
6. Implement `AllNames()` — all accessible names (for "did you mean" suggestions)
7. Implement `NewChild(name)` — child scope with parent link
8. Implement `IsolatedChild(name)` — child scope WITHOUT parent (for function calls)
9. Implement `ToMap()` — export all variables as `map[string]any` for expr-lang env

**Edge cases:**
- `Set` on variable in grandparent scope → must traverse chain
- `Declare` same name in child scope → allowed (shadowing)
- `Set` type mismatch with `any` type → allowed
- Concurrent access → mutex protects

**Acceptance criteria:**
- Block scope: variable in if-then not visible outside
- Scope chain: inner reads outer
- Mutation: inner can `set` outer
- Isolation: `IsolatedChild` cannot access parent

**Commit:** `feat(go-json): scope system with block scope, chain lookup, and isolation`

---

### Task 6: Type System

**Files:** Create `lang/types.go`
**Effort:** Medium
**Depends on:** Task 1

**Steps:**
1. Implement `InferType(value any) string` — detect Go type → go-json type string. Handle JSON's float64-for-all-numbers: detect integers via `float64(int64(f)) == f`
2. Implement `IsNullable(typ)`, `BaseType(typ)` — nullable type helpers
3. Implement `TypesCompatible(existingType, newType) bool` — strict-after-first-assignment rules. `any` accepts all. `?T` accepts nil. Same base type required otherwise.
4. Implement `TypeFromJSON(jsonType)` — map JSON type strings to internal names

**Edge cases:**
- JSON `42` parsed as `float64` by Go → detect as `int`
- `[]any` with all same type → keep as `[]any` (don't over-infer)
- `?[]string` — nullable array
- Empty type `""` → treat as `any`

**Acceptance criteria:**
- `InferType(42.0)` → `"int"` (when float64 is whole number)
- `InferType(3.14)` → `"float"`
- `TypesCompatible("int", "string")` → `false`
- `TypesCompatible("?string", "nil")` → `true`

**Commit:** `feat(go-json): type system with inference, nullable, and compatibility checking`

---

### Task 7: ExprEngine Abstraction

**Files:** Create `lang/expr_engine.go`
**Effort:** Large
**Depends on:** Task 3 (errors)

**Steps:**
1. Define `ExprEngine` interface: `Compile`, `Run`, `Eval`, `Validate`, `ReturnType`
2. Implement `ExprLangEngine` with:
   - Compiled expression cache (`map[string]*vm.Program` + `sync.RWMutex`)
   - Base options: `MaxNodes`, `Timezone`, custom `Function` registrations
   - `Compile` — check cache first, compile with `expr.Compile`, cache result
   - `Run` — `expr.Run` with error enrichment
   - `Eval` — compile + run shortcut
   - `Validate` — compile only, discard result
   - `ReturnType` — compile, inspect `Node().Type()`
3. Implement error enrichment:
   - Detect "undefined variable" → extract var name, suggest similar from env
   - Detect "type mismatch" → suggest conversion functions
   - Detect "nil pointer" → suggest optional chaining
   - Detect "division by zero" → suggest guard check
   - Fallback: wrap raw error with expression context

**Edge cases:**
- Cache key collision (same expression, different env types) → acceptable for Phase 4.5a
- Concurrent compilation → mutex protects cache
- expr-lang panic → wrap in defer/recover
- Very large cache → no eviction for now (programs are finite)

**Acceptance criteria:**
- `Eval("1 + 2", nil)` → `3`
- `Eval("undefined_var", env)` → enriched error with suggestions
- Second `Compile` with same expression → cache hit (no recompile)

**Commit:** `feat(go-json): ExprEngine abstraction with expr-lang, caching, and error enrichment`

---

## Batch 3: Parsing & Compilation (depends on Batch 2)

### Task 8: Parser

**Files:** Create `lang/parser.go`
**Effort:** Large
**Depends on:** Task 2, 3, 4

**Steps:**
1. Implement `Parse(input []byte) (*Program, error)` — pre-process → JSON unmarshal → build AST
2. Implement `ParseFile(path string)` — read file + parse
3. Implement `parseProgram(raw map[string]any)` — extract name, go_json, input, imports, functions, limits, steps, _c
4. Implement `parseSteps(rawSteps []any) ([]Node, error)` — iterate and dispatch
5. Implement `parseStep(m map[string]any, index int)` — detect step type by key presence, dispatch to specific parser
6. Implement individual step parsers:
   - `parseLetNode` — handle value/expr/with exclusivity, call shorthand, new shorthand
   - `parseSetNode` — target + value/expr/with
   - `parseIfNode` — condition + then + elif + else, recursive step parsing
   - `parseSwitchNode` — expr + cases map
   - `parseForNode` — variable + in/range + index + steps
   - `parseWhileNode` — condition + steps
   - `parseReturnNode` — detect string vs map vs literal
   - `parseCallNode` — function + with
   - `parseTryNode` — try + catch (as + steps) + finally
   - `parseErrorNode` — detect string vs map (code/message/details)
   - `parseLogNode` — detect string vs map (message/level/data)
   - `parseBreakNode`, `parseContinueNode`
7. Implement `parseComment(meta, m)` — handle string and array `_c`
8. Implement `parseFuncDef(name, raw)` — params, returns, steps
9. Implement `isCommentOnly(m)` — detect standalone `_c` step

**Edge cases:**
- `{"return": 42}` — literal number return
- `{"return": null}` — nil return
- `{"error": "msg"}` vs `{"error": {"code": "X"}}` — type detection
- `{"_c": "only comment"}` — no other keys → CommentNode
- Empty steps `[]` → valid, returns nil
- Missing `steps` key → library file (valid)
- JSON number as float64 → detect integers

**Acceptance criteria:**
- Parse all 15 step types
- Detect unknown step types with error
- Validate value mode exclusivity
- Handle overloaded return/error/log
- Parse functions, input schema, limits

**Commit:** `feat(go-json): JSON/JSONC parser — all step types, functions, input schema`

---

### Task 9: Compiled Program

**Files:** Create `lang/program.go`
**Effort:** Small
**Depends on:** Task 2, 7

**Steps:**
1. Define `CompiledProgram` — immutable struct with Name, GoJSON, AST, Functions map, Input, Limits
2. Define `CompiledFunc` — Name, Params (ordered []ParamDef), Returns, Steps
3. Define `ParamDef` — Name, Type, Default, HasDefault
4. Define `ResolvedLimits` — all limit values resolved to concrete numbers

**Acceptance criteria:** Struct compiles, no mutable methods, safe for concurrent use

**Commit:** `feat(go-json): immutable CompiledProgram struct`

---

### Task 10: Compiler

**Files:** Create `lang/compiler.go`
**Effort:** Large
**Depends on:** Task 2, 6, 7, 8, 9

**Steps:**
1. Implement `Compile(program, engine, limits) (*CompiledProgram, error)`
2. Compile functions — parse params in order, build `CompiledFunc`
3. Build base environment for expression validation (input schema + function names)
4. Validate all expressions in all steps via `engine.Validate()`
5. Validate all expressions in all functions (with function params in env)
6. Resolve limits — merge program limits with engine limits (most restrictive wins)
7. Detect `break`/`continue` outside loop → compile error
8. Detect `let` variable shadowing built-in functions → compile warning

**Edge cases:**
- Function references itself (recursion) → allowed
- Function references undefined function → compile error with suggestions
- Expression references undeclared variable → compile error with suggestions
- `break`/`continue` outside loop → compile error
- Nested `with` → recursive validation

**Acceptance criteria:**
- Valid programs compile without error
- Invalid expressions produce enriched errors
- Limits resolved correctly

**Commit:** `feat(go-json): compiler — AST to CompiledProgram with type checking`

---

## Batch 4: Execution (depends on Batch 3)

### Task 11: VM (Tree-Walk Interpreter)

**Files:** Create `lang/vm.go`
**Effort:** Large (largest task)
**Depends on:** Task 5, 7, 9, 10, 12

**Steps:**
1. Define `VM` struct — program, engine, scope, context, debugger, logger, trace, counters (stepCount, depth, callStack), limits
2. Define `ExecutionResult` — Value, Trace, Steps, Duration
3. Define sentinel types: `returnValue`, `breakSignal`, `continueSignal`
4. Implement `NewVM(program, engine, ...opts)` — create VM with context/timeout
5. Implement `Execute(input map[string]any) (*ExecutionResult, error)`:
   - Create root scope, inject input, register functions
   - Execute steps, return result
6. Implement `executeSteps(steps)` — iterate steps with:
   - Context check (timeout/cancellation)
   - Step limit check
   - Debug hook (OnStep)
   - Trace capture (timing)
   - Dispatch to specific executor
   - Handle return/break/continue signals
7. Implement step executors:
   - `executeLet` — evaluate value/expr/with/call, infer type, declare in scope
   - `executeSet` — evaluate value, handle dot-path mutation, type check
   - `executeIf` — evaluate condition, execute then/elif/else in child scopes
   - `executeSwitch` — evaluate expr, match cases, execute matched case
   - `executeFor` — evaluate in/range, iterate with child scope per iteration, handle break/continue
   - `executeWhile` — loop with condition check, iteration limit
   - `executeReturn` — evaluate expr/with/value, return sentinel
   - `executeCall` — call function (step-level, without let)
   - `executeTry` — execute try steps, catch errors, normalize error object, finally
   - `executeError` — evaluate message/code/details, throw GoJSONError
   - `executeLog` — evaluate message, call logger
8. Implement `callFunction(name, args, stepIndex)`:
   - Depth limit check + call stack push
   - Find function (with "did you mean" on miss)
   - Create isolated scope
   - Bind params (evaluate arg expressions in caller scope, bind in func scope)
   - Register functions for recursion
   - Execute function steps
   - Debug hooks (OnFunctionCall, OnFunctionReturn)
9. Implement `evalExpr(expression, stepIndex)` — scope.ToMap() → engine.Eval()
10. Implement `evalWith(with)` — recursive: strings = expressions, maps = recursive, others = literal
11. Implement `withScope(scope, fn)` — temporary scope switch
12. Implement `wrapFunction(fn)` — wrap CompiledFunc as Go func for expr-lang env (positional params)
13. Implement `setNestedProperty(path, value, stepIndex)` — parse dot/bracket path, traverse, mutate leaf
14. Implement `generateRange(rangeSpec)` — [start, end] or [start, end, step] → []any
15. Implement `normalizeError(err, stepIndex)` — any error → structured error map with code, message, details, step, stack

**Edge cases:**
- `return` inside loop → exits loop AND function
- `break` inside nested loops → exits innermost only
- `continue` inside if inside loop → continues loop
- Recursive function → depth limit with call stack trace
- Mutual recursion (A→B→A) → same depth counter
- `set "a.b.c"` where `a.b` is nil → runtime error
- `set "items[5]"` where array has 3 elements → runtime error
- `while true` → iteration limit
- Timeout during long expression evaluation → context cancellation
- `try` with error in `finally` → finally error replaces original
- `error` with string → auto-normalize to structured
- `log` with structured data → evaluate data expressions
- Empty steps array → return nil
- Function with no return → return nil

**Acceptance criteria:**
- Execute all step types correctly
- Scope isolation for functions
- Block scope for if/for/while
- Resource limits enforced (steps, depth, iterations, timeout)
- Debug hooks called at every step
- Trace captured when enabled
- Error messages enriched with context

**Commit:** `feat(go-json): VM tree-walk interpreter with all step types`

---

### Task 12: Debugger Interface

**Files:** Create `lang/debugger.go`
**Effort:** Small
**Depends on:** Task 2

**Steps:**
1. Define `DebugAction` enum: Continue, StepOver, StepInto, Pause
2. Define `StepInfo` struct: Index, Type, Node
3. Define `Debugger` interface: OnStep, OnVariable, OnError, OnFunctionCall, OnFunctionReturn
4. Define `ExecutionTrace` struct with `AddStep(TraceEntry)` method
5. Define `TraceEntry`: Step, Type, Var, Value, Condition, Result, DurationUs

**Acceptance criteria:** Interface compiles, VM can use it

**Commit:** `feat(go-json): debugger interface and execution trace`

---

## Batch 5: Runtime Layer (depends on Batch 4)

### Task 13: Resource Limits

**Files:** Create `runtime/limits.go`
**Effort:** Small
**Depends on:** Task 1

**Steps:**
1. Define `Limits` struct with all fields + defaults
2. Implement `DefaultLimits()`, `HardLimits()`
3. Implement `Resolve(engine, program Limits) ResolvedLimits` — most restrictive wins

**Commit:** `feat(go-json): resource limits with defaults and resolution`

---

### Task 14: Execution Context

**Files:** Create `runtime/context.go`
**Effort:** Small
**Depends on:** Task 1

**Steps:**
1. Define `Session` struct (UserID, Locale, TenantID, Groups)
2. Define `ExecutionMeta` struct (ID, Program, StartedAt, Depth, StepCount)
3. Implement injection into VM scope as `session.*` and `execution.*`

**Commit:** `feat(go-json): execution context — session and metadata`

---

### Task 15: Logger

**Files:** Create `runtime/logger.go`
**Effort:** Small
**Depends on:** Task 1

**Steps:**
1. Define `Logger` interface: `Log(level, message string, data map[string]any)`
2. Implement `DefaultLogger` — writes to stdout with timestamp
3. Define log levels: debug, info, warn, error

**Commit:** `feat(go-json): logger interface with default stdout implementation`

---

### Task 16: Runtime API

**Files:** Create `runtime/runtime.go`
**Effort:** Medium
**Depends on:** Task 11, 13, 14, 15

**Steps:**
1. Define `Runtime` struct with program cache, expr engine, limits, logger, debugger, extensions
2. Implement `NewRuntime(opts ...Option)` with options:
   - `WithStdlib(funcs)`, `WithLimits(limits)`, `WithLogger(logger)`
   - `WithDebugger(debugger)`, `WithTrace(bool)`
   - `WithExtension(name, ext)`, `WithIO(modules)`, `WithoutIO()`
   - `WithSession(session)`, `WithContext(ctx)`
3. Implement `Compile(input []byte) (*CompiledProgram, error)` — parse + compile with caching
4. Implement `Execute(program, input) (*ExecutionResult, error)` — create VM + execute
5. Implement `ExecuteJSON(programJSON []byte, input) (*ExecutionResult, error)` — compile + execute (with cache)
6. Program cache: `map[string]*CompiledProgram` keyed by content hash, protected by `sync.RWMutex`

**Edge cases:**
- Concurrent Execute calls with same program → cache hit, each gets own VM/scope
- Nil input → empty input map
- Missing program name → generate from hash

**Acceptance criteria:**
- `NewRuntime()` creates working runtime
- `Execute` runs programs correctly
- Program cache prevents recompilation
- Concurrent execution safe

**Commit:** `feat(go-json): runtime API with compile-once-run-many and program cache`

---

## Batch 6: Stdlib Layer 2 (depends on Batch 5)

### Task 17: Stdlib Registry

**Files:** Create `stdlib/registry.go`
**Effort:** Small
**Depends on:** Task 7

**Steps:**
1. Define `Registry` struct with function map
2. Implement `Register(name, fn, types, deprecated)` — register function with expr.Function() format
3. Implement `All() []expr.Option` — return all registered functions as expr options
4. Implement deprecation support — log warning when deprecated function called

**Commit:** `feat(go-json): stdlib registry with deprecation support`

---

### Task 18: Stdlib Math (Layer 2)

**Files:** Create `stdlib/math.go`
**Effort:** Small
**Depends on:** Task 17

**Functions (7):** `clamp(x, min, max)`, `sign(x)`, `randomInt(min, max)`, `randomFloat(min, max)`, `pow(base, exp)`, `sqrt(x)`, `mod(a, b)`

**Note:** `abs`, `ceil`, `floor`, `round`, `min`, `max`, `sum`, `mean`, `median` already in expr-lang — DO NOT reimplement.

**Edge cases:**
- `clamp(5, 10, 20)` → 10 (x < min)
- `clamp(25, 10, 20)` → 20 (x > max)
- `clamp(15, 10, 20)` → 15 (in range)
- `clamp(5, 20, 10)` → error (min > max)
- `randomInt(10, 10)` → 10 (min == max)
- `pow(2, -1)` → 0.5
- `sqrt(-1)` → NaN or error
- `mod(10, 0)` → error (division by zero)

**Commit:** `feat(go-json): stdlib math Layer 2 — clamp, sign, random, pow, sqrt, mod`

---

### Task 19: Stdlib Strings (Layer 2)

**Files:** Create `stdlib/strings.go`
**Effort:** Small
**Depends on:** Task 17

**Functions (5):** `padLeft(str, len, char)`, `padRight(str, len, char)`, `substring(str, start, end)`, `format(template, args...)`, `matches(str, pattern)` (regex)

**Note:** `upper`, `lower`, `trim`, `split`, `replace`, `repeat`, `indexOf`, `lastIndexOf`, `hasPrefix`, `hasSuffix`, `trimPrefix`, `trimSuffix`, `contains` already in expr-lang.

**Edge cases:**
- `padLeft("hi", 5, "0")` → `"000hi"`
- `padLeft("hello", 3, "0")` → `"hello"` (already longer)
- `substring("hello", 1, 3)` → `"el"`
- `substring("hello", -1, 3)` → error or clamp to 0
- `matches("hello", "[invalid")` → error (invalid regex)
- `format("%s is %d", "age", 30)` → `"age is 30"`

**Commit:** `feat(go-json): stdlib strings Layer 2 — padLeft, padRight, substring, format, matches`

---

### Task 20: Stdlib Arrays (Layer 2)

**Files:** Create `stdlib/arrays.go`
**Effort:** Small
**Depends on:** Task 17

**Functions (5):** `append(arr, item)`, `prepend(arr, item)`, `slice(arr, start, end)`, `chunk(arr, size)`, `zip(arr1, arr2)`

**Note:** `filter`, `map`, `reduce`, `find`, `sort`, `sortBy`, `reverse`, `concat`, `flatten`, `uniq`, `first`, `last`, `take`, `groupBy`, `count`, `all`, `any`, `one`, `none`, `findIndex`, `findLast`, `findLastIndex`, `join`, `sum`, `mean`, `median` already in expr-lang.

**Edge cases:**
- `append([], 1)` → `[1]`
- `slice([1,2,3,4,5], 1, 3)` → `[2,3]`
- `slice([1,2,3], 5, 10)` → `[]` (out of bounds → empty, not error)
- `chunk([1,2,3,4,5], 2)` → `[[1,2],[3,4],[5]]`
- `chunk([], 2)` → `[]`
- `chunk([1,2,3], 0)` → error (size must be > 0)
- `zip([1,2,3], ["a","b"])` → `[[1,"a"],[2,"b"]]` (shorter array determines length)

**Commit:** `feat(go-json): stdlib arrays Layer 2 — append, prepend, slice, chunk, zip`

---

### Task 21: Stdlib Types (Layer 2)

**Files:** Create `stdlib/types.go`
**Effort:** Small
**Depends on:** Task 17

**Functions (2):** `bool(x)`, `isNil(x)`

**Note:** `int`, `float`, `string`, `type`, `toJSON`, `fromJSON`, `toBase64`, `fromBase64`, `toPairs`, `fromPairs` already in expr-lang.

**Edge cases:**
- `bool(0)` → `false`, `bool(1)` → `true`
- `bool("")` → `false`, `bool("hello")` → `true`
- `bool(nil)` → `false`
- `bool([])` → `false`, `bool([1])` → `true`
- `isNil(nil)` → `true`, `isNil(0)` → `false`, `isNil("")` → `false`

**Commit:** `feat(go-json): stdlib types Layer 2 — bool, isNil`

---

## Batch 7: Tests (depends on all above)

### Task 22: Test Fixtures

**Files:** Create `testdata/*.json` and `testdata/*.jsonc`
**Effort:** Medium
**Depends on:** All above

**Create test programs:**
- `testdata/hello.json` — minimal program
- `testdata/hello.jsonc` — same with JSONC comments
- `testdata/variables.json` — let, set, value/expr/with modes
- `testdata/control_flow.json` — if/elif/else, switch
- `testdata/loops.json` — for, while, range, break, continue
- `testdata/functions.json` — definition, call, recursion (factorial)
- `testdata/error_handling.json` — try/catch/finally, error throw
- `testdata/stdlib_math.json` — Layer 2 math functions
- `testdata/stdlib_strings.json` — Layer 2 string functions
- `testdata/stdlib_arrays.json` — Layer 2 array functions
- `testdata/limits.json` — programs that exceed limits
- `testdata/comments.json` — _c inline, standalone, multi-line
- `testdata/types.json` — type inference, strict assignment, any, nullable

**Commit:** `test(go-json): test fixture programs for all features`

---

### Task 23: Unit Tests

**Files:** Create `lang/*_test.go`, `stdlib/*_test.go`, `runtime/*_test.go`
**Effort:** Large
**Depends on:** Task 22

**Test coverage:**
- `preprocess_test.go` — comment stripping, trailing commas, edge cases
- `scope_test.go` — declare, get, set, chain, isolation, shadowing
- `types_test.go` — inference, compatibility, nullable
- `expr_engine_test.go` — eval, compile, cache, error enrichment, suggestions
- `parser_test.go` — parse all step types, error cases, overloaded nodes
- `compiler_test.go` — compile valid/invalid programs, limit resolution
- `vm_test.go` — execute all step types, scope behavior, return/break/continue signals
- `errors_test.go` — error formatting, JSON output, levenshtein, suggestions
- `stdlib/math_test.go`, `strings_test.go`, `arrays_test.go`, `types_test.go`
- `runtime/runtime_test.go` — NewRuntime, Execute, cache, concurrent execution

**Commit:** `test(go-json): comprehensive unit tests for all components`

---

### Task 24: Integration Tests

**Files:** Create `lang/integration_test.go`
**Effort:** Medium
**Depends on:** Task 23

**Test scenarios:**
- Full program execution: parse → compile → execute → verify result
- Factorial recursive function
- Complex control flow (nested if/for/while)
- Error handling (try/catch with structured errors)
- Resource limits (step limit, depth limit, timeout)
- Concurrent execution (same program, different inputs, goroutines)
- JSONC programs (comments stripped, same result as JSON)
- Programs with _c comments (preserved in AST, not executed)
- Type checking (strict-after-assignment, any opt-in, nullable)

**Commit:** `test(go-json): integration tests — full program execution scenarios`

---

### Task 25: Edge Case Tests

**Files:** Create `lang/edge_cases_test.go`
**Effort:** Medium
**Depends on:** Task 23

**Test scenarios:**
- Nil handling: nil property access, nil in expressions, nil coalescing
- Type errors: assign string to int variable, wrong function arg type
- Overflow: very deep recursion, very long loops, very large variables
- Empty programs: no steps, no functions, no input
- Malformed JSON: syntax errors, missing required fields
- Deeply nested structures: 10+ levels of if/for nesting
- Unicode: variable names with unicode, string values with unicode
- Large programs: 1000+ steps, 100+ functions
- Concurrent stress: 100 goroutines executing same program

**Commit:** `test(go-json): edge case tests — nil, types, overflow, unicode, concurrency`

---

## Summary

| Task | Description | Effort | Depends On | Batch |
|------|-------------|--------|------------|-------|
| 1 | Project scaffold | Small | — | 1 |
| 2 | AST node types | Medium | 1 | 1 |
| 3 | Error types | Medium | 1 | 1 |
| 4 | JSONC pre-processor | Small | 1 | 1 |
| 5 | Scope system | Medium | 1 | 2 |
| 6 | Type system | Medium | 1 | 2 |
| 7 | ExprEngine abstraction | Large | 3 | 2 |
| 8 | Parser | Large | 2,3,4 | 3 |
| 9 | Compiled Program | Small | 2,7 | 3 |
| 10 | Compiler | Large | 2,6,7,8,9 | 3 |
| 11 | VM | Large | 5,7,9,10,12 | 4 |
| 12 | Debugger interface | Small | 2 | 4 |
| 13 | Resource limits | Small | 1 | 5 |
| 14 | Execution context | Small | 1 | 5 |
| 15 | Logger | Small | 1 | 5 |
| 16 | Runtime API | Medium | 11,13,14,15 | 5 |
| 17 | Stdlib registry | Small | 7 | 6 |
| 18 | Stdlib math | Small | 17 | 6 |
| 19 | Stdlib strings | Small | 17 | 6 |
| 20 | Stdlib arrays | Small | 17 | 6 |
| 21 | Stdlib types | Small | 17 | 6 |
| 22 | Test fixtures | Medium | All | 7 |
| 23 | Unit tests | Large | 22 | 7 |
| 24 | Integration tests | Medium | 23 | 7 |
| 25 | Edge case tests | Medium | 23 | 7 |

**Total: 25 tasks across 7 batches**
**Critical path: Task 1 → 2 → 8 → 10 → 11 → 16 → 23**
**Estimated total effort: ~3-4 weeks for one developer**

**Effort breakdown:**
- Small (1-2 hours): 11 tasks
- Medium (3-6 hours): 9 tasks
- Large (1-2 days): 5 tasks
