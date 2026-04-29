# Phase 4.5e — Unified Expression Engine

**Status**: ✅ Done
**Date**: 29 July 2026
**Design Decisions**: Based on analysis in this conversation session (expr-lang capabilities, Odoo record rule patterns, bitcode engine expression audit)
**Depends on**: Phase 4.5c (go-json I/O + Integration)
**Blocks**: Phase 7 (Module "setting")

---

## §1. Problem Statement

The bitcode engine currently has **3 separate, hand-rolled expression evaluators** (plus a DAG copy-paste duplicate and a template interpolation system). Each was written independently, with different capabilities, different bugs, and no shared code.

### 1.1 Current Expression Systems

| # | System | Location | Lines | Purpose | Known Issues |
|---|--------|----------|-------|---------|--------------|
| 1 | **Computed Field Evaluator** | `runtime/expression/evaluator.go` | 786 | Computed fields, formulas, aggregates on child collections | Limited to 10 functions; no string functions, no date functions, no conditionals beyond `if()` |
| 2 | **Process Condition Evaluator** | `executor/steps/control.go` | 63 | `if`/`switch`/`loop` conditions, `{{var}}` interpolation | `" > "` always returns `true` (bug); string-based comparison only |
| 3 | **Hook Condition Evaluator** | `runtime/hook/expr.go` | 133 | Agent/event trigger conditions | `&&`/`||` parsed via `strings.Split` (fragile); no operator precedence |
| — | **DAG Edge Evaluator** (duplicate of #2) | `executor/dag.go` | 41 | DAG process edge conditions | Copy-paste of process evaluator; same `" > "` bug |
| — | **Record Rule Interpolation** (not an evaluator) | 3 files, ~43 lines | 43 | `{{user.id}}` → value in SQL WHERE filters | String replace only; no type safety; no validation of template variables |

**3 distinct evaluators + 1 duplicate + 1 interpolation system = ~1,066 lines of duplicated, inconsistent, buggy expression handling.**

Meanwhile, go-json already wraps `expr-lang/expr` — a production-grade expression engine with:
- 68+ built-in functions
- Bytecode-compiled VM
- Compile-time type checking
- AST visitor/patcher pattern
- Caching (compile once, run many)

And the engine already depends on go-json (`go.mod`: `github.com/bitcode-framework/go-json`).

### 1.2 Goals

1. **Replace all 3 hand-rolled evaluators** with go-json's `ExprEngine` (backed by expr-lang/expr)
2. **Upgrade record rules** from `{{}}` string replace to expr-lang with `ctx.*` namespace and AST-to-WHERE conversion
3. **Expose a public `EvalExpr()` API** from go-json for engine consumption without requiring a full VM
4. **Maintain backward compatibility** with existing `{{}}` syntax via pre-processor
5. **Fix all known bugs** in condition evaluation

### 1.3 Non-Goals

- Replacing the process executor step dispatcher (that stays in Go)
- Changing the process JSON format (steps, types, etc.)
- Replacing Go `html/template` usage (email templates, view templates — those are a different concern)
- Replacing record rule `domain_filter` storage format in DB (old format still readable)

---

## §2. Architecture

### 2.1 New Public API in go-json

```go
// packages/go-json/runtime/eval.go
//
// NOTE: This file lives in packages/go-json/ (the go-json module).
// Engine files (aggregate.go, record_rule_expr.go) live in engine/internal/.
// The engine depends on go-json, not the other way around.

// EvalExpr evaluates a single expression string against an environment.
// This is the lightweight entry point for engine consumers that need
// expression evaluation without a full go-json program/VM.
//
// Uses the shared ExprLangEngine with compilation caching.
func EvalExpr(expression string, env map[string]any) (any, error)

// EvalExprFloat is a convenience wrapper that coerces the result to float64.
// Non-numeric results return an error.
func EvalExprFloat(expression string, env map[string]any) (float64, error)

// EvalExprBool evaluates an expression and returns a boolean result.
// Applies truthiness coercion for backward compatibility:
//   - bool → as-is
//   - nil → false
//   - numeric 0 → false, non-zero → true
//   - "" → false, non-empty string → true
//   - empty array/map → false, non-empty → true
//   - any other non-nil value → true
//
// This matches the current engine's truthiness semantics, ensuring
// existing conditions like {"condition": "{{input.name}}"} continue
// to work (non-empty string → true).
func EvalExprBool(expression string, env map[string]any) (bool, error)

// ParseExpr parses an expression and returns the AST root node.
// Returns expr-lang's ast.Node wrapped in ExprTree for stability.
// Used by record rule AST-to-WHERE conversion.
func ParseExpr(expression string) (*ExprTree, error)

// ExprTree wraps expr-lang's parsed AST to provide a stable public API.
// This prevents leaking the expr-lang dependency to engine consumers.
type ExprTree struct {
    Root ast.Node  // expr-lang AST root node (from github.com/expr-lang/expr/ast)
}

// ValidateExpr validates an expression against a known environment
// without executing it. Returns nil if valid.
func ValidateExpr(expression string, env map[string]any) error
```

### 2.2 Shared ExprLangEngine Instance

```go
// Singleton shared engine with compilation cache.
// All engine subsystems use this — computed fields, process conditions,
// hook conditions, record rules.
var sharedEngine = lang.NewExprLangEngine()
```

The `ExprLangEngine` already has:
- Thread-safe compilation cache (`sync.RWMutex` + `map[string]CompiledExpr`)
- Panic recovery (expr-lang panics caught and wrapped)
- MaxNodes limit (default 1000)
- Error enrichment with context

### 2.3 Component Replacement Map

```
BEFORE                              AFTER
──────                              ─────
expression/evaluator.go (786 lines) → runtime.EvalExpr() + custom aggregate functions
expression/hydrator.go (168 lines)  → HydratorV2 using runtime.EvalExpr()
steps/control.go evaluateCondition  → runtime.EvalExprBool()
steps/control.go interpolate        → runtime.EvalExpr() with env binding
dag.go evaluateEdgeCondition        → shared evaluateCondition (deduplicated)
hook/expr.go evaluateSimpleExpr     → runtime.EvalExprBool()
record_rule interpolation           → AST-to-WHERE converter + ctx.* namespace
```

---

## §3. Computed Fields (Model Layer)

### 3.1 Current Behavior

Model fields can declare `computed` or `formula` expressions:

```json
{
  "total": {"type": "decimal", "computed": "sum(lines.subtotal)"},
  "full_name": {"type": "string", "formula": "first_name + ' ' + last_name"},
  "weighted": {"type": "decimal", "computed": "expected_revenue * probability / 100"}
}
```

The `Hydrator` evaluates these after every record load (FindOne, FindAll).

### 3.2 New Behavior

Same JSON format. Same `Hydrator` interface. Different engine underneath.

**Environment for computed field evaluation:**

```go
env := map[string]any{
    // All record fields as top-level variables
    "quantity":   record["quantity"],
    "unit_price": record["unit_price"],
    "first_name": record["first_name"],
    // ...all fields from record

    // Aggregate functions for child collections
    "sum":   aggregateFunc("sum", childCollections),
    "count": aggregateFunc("count", childCollections),
    "avg":   aggregateFunc("avg", childCollections),
    "min":   aggregateFunc("min", childCollections),
    "max":   aggregateFunc("max", childCollections),
}
```

**Aggregate function registration:**

The current evaluator handles `sum(lines.subtotal)` by detecting `collection.field` patterns. With expr-lang, we register custom functions:

```go
// Register aggregate functions scoped to THIS evaluation context only.
// These are NOT registered on the shared global engine — they are
// passed as expr.Env() options per-evaluation to avoid shadowing
// expr-lang's built-in sum/count/min/max which operate on arrays.
//
// Naming: childSum, childCount, etc. to avoid collision with built-in sum().
// Backward compat: also register "sum" in the scoped env, since existing
// computed fields use sum(lines.subtotal). This works because the scoped
// env takes precedence over built-ins for this specific evaluation.
opts := []expr.Option{
    expr.Function("childSum", aggregateFunc("sum", childCollections),
        new(func(string) float64)),
    expr.Function("childCount", aggregateFunc("count", childCollections),
        new(func(string) float64)),
    // ... childAvg, childMin, childMax
}
// For backward compat, also register "sum" etc. in the env map:
env["sum"] = func(arg any) float64 {
    // If arg is string like "lines.subtotal" → aggregate over child collection
    // If arg is []any → delegate to built-in sum behavior
    return resolveAggregateOrBuiltin("sum", arg, childCollections)
}
```

**Key design decision:** Aggregate functions are registered **per-evaluation**, not on the shared global engine. This prevents shadowing expr-lang's built-in `sum([1,2,3])` in other contexts (process conditions, hook conditions, record rules). Only computed field evaluation gets the aggregate overloads.

### 3.3 Gained Capabilities

Computed fields can now use the full expr-lang expression language:

```json
// Before: only arithmetic + 10 functions
"computed": "quantity * unit_price"

// After: full expr-lang (68+ functions + go-json stdlib)
"computed": "quantity * unit_price * (1 - discount / 100)"
"computed": "upper(first_name) + ' ' + upper(last_name)"
"computed": "status == 'active' ? amount : 0"
"computed": "len(filter(lines, .quantity > 0))"
"computed": "sum(lines.subtotal) + sum(lines.tax)"
```

### 3.4 Backward Compatibility

All existing computed/formula expressions remain valid — expr-lang is a superset of the current evaluator's capabilities. The 10 functions (`sum`, `count`, `avg`, `min`, `max`, `abs`, `round`, `ceil`, `floor`, `if`) are all available in expr-lang.

---

## §4. Process Conditions (Runtime Layer)

### 4.1 Current Behavior

Process steps use `{{variable}}` interpolation and string-based condition evaluation:

```json
{"type": "if", "condition": "{{input.total}} > 100", "then": "approve", "else": "reject"}
```

The `evaluateCondition()` function:
1. Replaces `{{input.key}}` and `{{var}}` with string values
2. Checks for `" > "` substring → **always returns true** (bug)
3. Checks for `" == "` substring → string comparison
4. Falls back to variable truthiness

### 4.2 New Behavior

Conditions are evaluated via `runtime.EvalExprBool()` with proper environment binding:

```go
env := map[string]any{
    "input": execCtx.Input,       // all input fields
}
for k, v := range execCtx.Variables {
    env[k] = v                    // all process variables
}

result, err := runtime.EvalExprBool(condition, env)
```

**Backward compatibility for `{{}}` syntax:**

A pre-processor converts `{{identifier}}` patterns to bare identifiers before evaluation:

```go
// normalizeCondition converts {{identifier.path}} to bare identifier.path
// Uses targeted regex — only strips {{ }} around valid identifier patterns,
// NOT blind string replacement (which would break literal {{ in strings).
var conditionVarRe = regexp.MustCompile(`\{\{(\w+(?:\.\w+)*)\}\}`)

func normalizeCondition(condition string) string {
    // "{{input.total}} > 100" → "input.total > 100"
    // "{{active}}" → "active"
    // "status == '{{pending}}'" → "status == 'pending'" (identifier inside string — still works)
    return conditionVarRe.ReplaceAllString(condition, "$1")
}
```

This works because:
- `{{input.total}}` → `input.total` — valid expr-lang (member access on `input` map)
- `{{active}}` → `active` — valid expr-lang (variable lookup)
- `{{t('key')}}` — translation interpolation handled separately BEFORE this step (see §4.3)
- Only `{{word.word}}` patterns are matched — literal `{{` inside strings without matching identifier pattern are left as-is

### 4.3 Translation Interpolation

The `{{t('key')}}` pattern is NOT an expression — it's a translation lookup. This is handled as a separate pre-processing step before expression evaluation:

```go
func preprocessTranslations(s string, t func(string) string) string {
    // Replace {{t('key')}} with translated string
    // This runs BEFORE expression evaluation
    re := regexp.MustCompile(`\{\{t\('([^']+)'\)\}\}`)
    return re.ReplaceAllStringFunc(s, func(match string) string {
        key := re.FindStringSubmatch(match)[1]
        return t(key)
    })
}
```

### 4.4 String Interpolation in Non-Condition Contexts

For HTTP URLs, log messages, and validation error messages, the current `interpolate()` function does `{{var}}` → value string replacement. These are NOT expressions — they are template strings.

New approach: use `fmt.Sprintf`-style or keep simple replacement, but with proper escaping:

```go
func interpolateString(template string, execCtx *executor.Context) string {
    // For non-expression contexts (URLs, messages), keep simple replacement
    // but validate that all {{}} references resolve
    result := template
    result = preprocessTranslations(result, execCtx.T)
    for key, val := range execCtx.Input {
        result = strings.ReplaceAll(result, "{{input."+key+"}}", fmt.Sprintf("%v", val))
    }
    for key, val := range execCtx.Variables {
        result = strings.ReplaceAll(result, "{{"+key+"}}", fmt.Sprintf("%v", val))
    }
    return result
}
```

This is intentionally kept as string replacement — these are template strings, not expressions. The key difference from the current implementation: **conditions use expr-lang, templates use string replacement**.

### 4.5 DAG Edge Conditions

The DAG executor's `evaluateEdgeCondition()` is replaced by the same `evaluateCondition()` used by the sequential executor. No more code duplication.

---

## §5. Hook/Agent Conditions (Event Layer)

### 5.1 Current Behavior

Agent event handlers can have conditions:

```json
{
  "process": "send_notification",
  "condition": "status == 'active' && amount > 1000"
}
```

The `evaluateSimpleExpr()` function parses this via `strings.Split(" && ")` and `strings.Contains(" == ")`.

### 5.2 New Behavior

Direct evaluation via `runtime.EvalExprBool()`:

```go
func evaluateHandlerCondition(condition string, data map[string]any) bool {
    result, err := runtime.EvalExprBool(condition, data)
    if err != nil {
        // Log error, return false (fail-closed)
        return false
    }
    return result
}
```

The `data` map already contains record fields, `__old` (previous values), and `__session` (session context). These are remapped to clean names:

```go
env := map[string]any{}
// Copy record fields
for k, v := range eventCtx.Data {
    env[k] = v
}
// Remap special keys
if old, ok := eventCtx.Data["__old"].(map[string]any); ok {
    env["old"] = old
}
if sess, ok := eventCtx.Data["__session"].(map[string]any); ok {
    env["session"] = sess
}
```

So conditions can use:
```
status == 'active'                    // record field
old.status != status                  // compare with previous value
session.role == 'admin'               // session context
amount > 1000 && status != old.status // combined
```

---

## §6. Record Rules (Security Layer)

### 6.1 Current Behavior

Record rules use `domain_filter` with `{{user.id}}` string replacement:

```json
{
  "domain_filter": "[[\"created_by\", \"=\", \"{{user.id}}\"]]"
}
```

The `InterpolateDomainFilters()` function replaces `{{key}}` with values from a `map[string]string`.

### 6.2 New Behavior: expr-lang with `ctx.*` Namespace

New record rules use `domain_filter_expr` with expr-lang syntax:

```json
{
  "domain_filter_expr": "created_by == ctx.user_id"
}
```

**`ctx` is a reserved namespace** — it MUST NOT be used as a model field name. The engine validates this at model registration time.

### 6.3 Context Object

```go
type RecordRuleContext struct {
    UserID        string   `expr:"user_id"`
    CompanyID     string   `expr:"company_id"`
    CompanyIDs    []string `expr:"company_ids"`
    DepartmentID  string   `expr:"department_id"`
    DepartmentIDs []string `expr:"department_ids"`
    GroupIDs      []string `expr:"group_ids"`
    Groups        []string `expr:"groups"`
    Role          string   `expr:"role"`
    TenantID      string   `expr:"tenant_id"`
    Now           time.Time `expr:"now"`
    Today         string   `expr:"today"`
}
```

### 6.4 AST-to-WHERE Conversion Pipeline

```
Expression: "created_by == ctx.user_id && department_id in ctx.department_ids"

Step 1: Parse → AST
  BinaryNode("&&",
    BinaryNode("==", Ident("created_by"), Member("ctx","user_id")),
    BinaryNode("in", Ident("department_id"), Member("ctx","department_ids")))

Step 2: Resolve ctx.* → concrete values
  BinaryNode("&&",
    BinaryNode("==", Ident("created_by"), String("usr-42")),
    BinaryNode("in", Ident("department_id"), Array(["dept-1","dept-2"])))

Step 3: Emit WhereClause[]
  [{Field:"created_by", Op:"=", Value:"usr-42"},
   {Field:"department_id", Op:"in", Value:["dept-1","dept-2"]}]

Step 4: Apply via Query.Where() → parameterized SQL
  WHERE created_by = ? AND department_id IN (?,?)
```

### 6.5 AST Walker: `ExprToFilters`

The walker produces a **recursive filter tree**, not a flat list. This correctly handles nested boolean logic like `(a == 1 || b == 2) && c == 3`.

```go
// FilterNode is either a single condition or a group of conditions.
type FilterNode struct {
    // Exactly one of Condition or Group is set.
    Condition *WhereClause  // leaf: field op value
    Group     *FilterGroup  // branch: AND/OR of children
}

// FilterGroup is a set of FilterNodes joined by a connector.
type FilterGroup struct {
    Connector string       // "AND" or "OR"
    Children  []FilterNode
}

// ExprToFilters walks an expr-lang AST and produces a FilterNode tree.
type ExprToFilters struct {
    ctxData     map[string]any
    modelFields map[string]bool
    errors      []string
}

// Convert is the entry point — takes a parsed AST root and returns a filter tree.
func (w *ExprToFilters) Convert(node ast.Node) (*FilterNode, error) {
    result := w.walkNode(node)
    if len(w.errors) > 0 {
        return nil, fmt.Errorf("record rule errors: %v", w.errors)
    }
    return result, nil
}

func (w *ExprToFilters) walkNode(node ast.Node) *FilterNode {
    switch n := node.(type) {
    case *ast.BinaryNode:
        switch n.Operator {
        case "&&":
            left := w.walkNode(n.Left)
            right := w.walkNode(n.Right)
            return &FilterNode{Group: &FilterGroup{
                Connector: "AND", Children: []FilterNode{*left, *right},
            }}
        case "||":
            left := w.walkNode(n.Left)
            right := w.walkNode(n.Right)
            return &FilterNode{Group: &FilterGroup{
                Connector: "OR", Children: []FilterNode{*left, *right},
            }}
        case "==", "!=", ">", "<", ">=", "<=":
            return w.extractComparison(n)
        case "in", "not in":
            return w.extractInClause(n)
        case "contains", "startsWith", "endsWith":
            return w.extractLikeClause(n) // BinaryNode operators → SQL LIKE
        default:
            w.errors = append(w.errors, fmt.Sprintf("unsupported operator: %s", n.Operator))
            return nil
        }
    default:
        w.errors = append(w.errors, fmt.Sprintf("unsupported node type: %T", node))
        return nil
    }
}

func (w *ExprToFilters) extractComparison(n *ast.BinaryNode) *FilterNode {
    field := w.resolveFieldName(n.Left)
    value := w.resolveValue(n.Right)

    if field == "" {
        w.errors = append(w.errors, "left side must be a model field")
        return nil
    }
    if !w.modelFields[field] {
        w.errors = append(w.errors, fmt.Sprintf("unknown field: %s", field))
        return nil
    }

    return &FilterNode{Condition: &WhereClause{
        Field: field, Operator: mapOperator(n.Operator), Value: value,
    }}
}
```

The `FilterNode` tree maps directly to the existing `Query` builder's `WhereClause` / `ConditionGroup` structure (which already supports nested AND/OR groups).

**Rejected operators in record rules:**

| Operator | Reason | Action |
|----------|--------|--------|
| `matches` | expr-lang `matches` uses Go regex; no safe SQL equivalent. ReDoS risk on DB. | **Rejected** — use `contains` or `startsWith` operators instead |
| `contains`, `startsWith`, `endsWith` | These are expr-lang **operators** (BinaryNode), not functions (CallNode). Syntax: `field contains "value"`, `field startsWith "prefix"`, `field endsWith "suffix"` | **Allowed** — converted to SQL `LIKE` patterns (`%value%`, `value%`, `%value`) |
| `**`, `%`, `+`, `-`, `*`, `/` | Arithmetic — not meaningful for WHERE filters | **Rejected** in record rules |

### 6.6 Security Validations

| Check | When | What |
|-------|------|------|
| **Field name validation** | AST walk | Left side of comparison must be a known model field via `IsSafeFieldName()` |
| **Context access whitelist** | AST walk | Only `ctx.*` members allowed; no `sql.*`, `http.*`, `exec.*` |
| **No function calls** | AST walk | Reject `CallNode` (except whitelisted: `len`). Note: `contains`/`startsWith`/`endsWith` are BinaryNode operators, not CallNode — they are handled in the operator switch. |
| **No tautology** | AST walk | Reject expressions with no field reference (e.g., `1 == 1`) |
| **Empty array check** | Value resolution | `ctx.department_ids == []` → deny all (fail-closed) |
| **Nil context check** | Value resolution | `ctx.user_id == nil` → deny all (fail-closed) |
| **Parameterized output** | Query builder | All values pass through `?` parameters — never string-concatenated to SQL |
| **MaxNodes limit** | Parse time | Expression complexity capped (default 200 for record rules) |

### 6.7 Conditional Record Rules

For expressions that cannot be fully converted to WHERE (e.g., ternary):

```
ctx.role == 'admin' || created_by == ctx.user_id
```

This IS convertible — it becomes `WHERE (role = 'admin') OR (created_by = ?)`.

```
ctx.role == 'admin' ? true : created_by == ctx.user_id
```

This requires two-phase evaluation:
1. Evaluate `ctx.role == 'admin'` → `true` or `false`
2. If `true` → no filter (see all records)
3. If `false` → convert `created_by == ctx.user_id` to WHERE

### 6.8 Backward Compatibility

| Format | Field | Behavior |
|--------|-------|----------|
| Old | `domain_filter` | `[[\"field\",\"op\",\"{{var}}\"]]` — legacy interpolation path, unchanged |
| New | `domain_filter_expr` | `field == ctx.var` — expr-lang AST-to-WHERE path |

Both can coexist on the same model. Rules are combined following existing Odoo-compatible semantics:

- **Global rules** (no group) → INTERSECT (AND) — all must pass
- **Group rules** (with group) → UNION (OR) — any one must pass
- **Final** = AND(global rules) AND OR(group rules)

Within this composition, each individual rule can use either `domain_filter` (legacy) or `domain_filter_expr` (new). They produce the same output type (`[][]any` filter arrays) and are combined identically. If a single rule has both fields set, `domain_filter_expr` takes precedence for that rule.

Migration tool converts simple cases:
```
["created_by", "=", "{{user.id}}"]  →  created_by == ctx.user_id
["company_id", "in", "{{user.company_ids}}"]  →  company_id in ctx.company_ids
```

### 6.9 `{{}}` Syntactic Sugar

For user convenience, `{{}}` syntax is accepted in `domain_filter_expr` and auto-converted:

```go
// preprocessRuleExpr converts {{}} sugar to ctx.* namespace.
// Handles all known interpolation keys from the legacy system.
var ruleExprVarRe = regexp.MustCompile(`\{\{(\w+)\.(\w+)\}\}`)

func preprocessRuleExpr(input string) string {
    // {{session.user_id}} → ctx.user_id
    // {{user.id}}         → ctx.user_id  (alias mapping)
    // {{user.company_id}} → ctx.company_id
    // {{user.company_ids}} → ctx.company_ids
    result := ruleExprVarRe.ReplaceAllStringFunc(input, func(match string) string {
        parts := ruleExprVarRe.FindStringSubmatch(match)
        prefix, key := parts[1], parts[2]
        switch prefix {
        case "session", "user":
            // Map legacy user.id → ctx.user_id
            if prefix == "user" && key == "id" {
                return "ctx.user_id"
            }
            return "ctx." + key
        default:
            return match // unknown prefix — leave as-is
        }
    })
    // Handle standalone variables
    result = strings.ReplaceAll(result, "{{now}}", "ctx.now")
    result = strings.ReplaceAll(result, "{{today}}", "ctx.today")
    return result
}
```

---

## §7. Reserved Namespaces

To prevent collision between model fields and injected context variables:

| Namespace | Used In | Contains |
|-----------|---------|----------|
| `ctx` | Record rules | User/session/tenant context for security filters |
| `input` | Process conditions | Process input data |
| `old` | Hook conditions | Previous record values (before update) |
| `session` | Hook conditions | Session context in event handlers |
| `self` | go-json structs | Current struct instance (Phase 4.5b) |

**Validation:** Model field names are checked against reserved namespaces at model registration time.

- **New models:** A field named `ctx`, `input`, `old`, or `session` is rejected with a clear error message at startup.
- **Existing models:** If a model already has a field with a reserved name, the engine logs a **warning** (not error) at startup and the field continues to work normally. Record rules using `domain_filter_expr` on that model will not be able to use the conflicting namespace — they must use the legacy `domain_filter` format instead. This is a soft deprecation; a future major version may make it a hard error.
- **Likelihood:** `ctx`, `input`, `old` are uncommon field names. `session` is possible (e.g., in a session-tracking model) but rare as a direct field name (usually `session_id`).

---

## §8. Files Changed

**Module boundary:** Files prefixed with `packages/go-json/` live in the go-json module. Files prefixed with `engine/internal/` live in the bitcode engine module. The engine depends on go-json, not the other way around.

### 8.1 Files Deleted

| File | Lines | Reason |
|------|-------|--------|
| `runtime/expression/evaluator.go` | 786 | Replaced by `runtime.EvalExpr()` |
| `runtime/hook/expr.go` | 133 | Replaced by `runtime.EvalExprBool()` |

### 8.2 Files Rewritten

| File | Lines | What Changes |
|------|-------|-------------|
| `runtime/expression/hydrator.go` | 168 | Internal calls switch from `Evaluate()` to `runtime.EvalExpr()` with aggregate function registration |
| `executor/steps/control.go` | ~63 lines affected | `evaluateCondition()` → `runtime.EvalExprBool()` with `{{}}` normalization; `interpolate()` kept for template strings; DAG duplicate removed |
| `executor/dag.go` | ~41 lines removed | `evaluateEdgeCondition()`, `interpolateCondition()`, `resolveConditionVar()` removed — uses shared evaluator |

### 8.3 Files Modified (Import/Type Changes)

| File | Lines Affected | What Changes |
|------|---------------|-------------|
| `app.go` | ~8 | Hydrator initialization updated |
| `infrastructure/persistence/repository.go` | ~14 | Hydrator type reference updated |
| `presentation/api/router.go` | ~7 | Hydrator type reference updated |
| `presentation/view/renderer.go` | ~5 | Hydrator type reference updated |
| `runtime/hook/dispatcher.go` | ~3 | `evaluateHandlerCondition()` calls `runtime.EvalExprBool()` |

### 8.4 Files Added

| File | Purpose |
|------|---------|
| `packages/go-json/runtime/eval.go` | Public `EvalExpr()`, `EvalExprBool()`, `EvalExprFloat()`, `ParseExpr()`, `ValidateExpr()` API |
| `engine/internal/runtime/expression/aggregate.go` | Aggregate function registration for computed fields (`sum`, `count`, `avg`, `min`, `max` over child collections) |
| `engine/internal/infrastructure/persistence/record_rule_expr.go` | AST-to-WHERE converter for expr-lang record rules |
| `compiler/parser/model.go` (modified) | Add `DomainFilterExpr` field to record rule struct |
| `infrastructure/persistence/record_rule_service.go` (modified) | Add expression evaluation path alongside legacy interpolation |

### 8.5 Test Files

| File | Action | Reason |
|------|--------|--------|
| `runtime/expression/evaluator_test.go` | **Rewrite** | Test `runtime.EvalExpr()` instead of custom evaluator; same test cases, different calls |
| `executor/steps/steps_test.go` | **Update** | ~12 tests adjusted for `{{}}` normalization behavior |
| `runtime/hook/dispatcher_test.go` | **Update** | 3 `TestExprEvaluator_*` tests use `runtime.EvalExprBool()` |
| `infrastructure/persistence/record_rule_expr_test.go` | **New** | AST-to-WHERE conversion tests |
| `packages/go-json/runtime/eval_test.go` | **New** | Public API tests |

### 8.6 Documentation

| File | Action |
|------|--------|
| `engine/docs/features/models.md` | Update computed/formula section |
| `engine/docs/features/processes.md` | Update interpolation/condition section |
| `engine/docs/features/security.md` | Add `domain_filter_expr` documentation |
| `engine/docs/architecture.md` | Update expression engine section |
| `engine/AGENTS.md` | Update expression engine reference |
| `packages/go-json/AGENTS.md` | Add note about engine integration API |

### 8.7 Database Migration

| Change | Detail |
|--------|--------|
| Add column `domain_filter_expr` to `record_rule` table | TEXT, nullable, stores expr-lang expression |

---

## §9. Edge Cases

| Scenario | Behavior |
|----------|----------|
| Expression compilation fails at runtime | Return error; for conditions → `false` (fail-closed); for computed fields → field value = `nil` |
| Computed field references non-existent child collection | `childSum("nonexistent.field")` → `0.0` (same as current behavior) |
| Record rule `ctx.*` field not in context struct | expr-lang compile error at rule load time — caught early |
| `{{}}` preprocessor on expression without `{{}}` | No-op — regex finds no matches, returns input unchanged |
| Process condition returns non-bool | `EvalExprBool` applies truthiness coercion (see §2.1) |
| Record rule expression returns non-filter | Error at evaluation time — must produce filter-compatible output |
| Concurrent access to shared ExprLangEngine | Thread-safe — `sync.RWMutex` on compilation cache |
| Very long expression (DoS attempt) | `MaxNodes` limit (1000 default, 200 for record rules) rejects at parse time |
| Circular reference in computed fields | Not an expression engine concern — handled by Hydrator's existing field iteration order |
| `domain_filter_expr` with `ctx.department_ids = []` (empty) | Emit deny-all filter: `WHERE 1 = 0` (fail-closed, no records visible) |
| `domain_filter_expr` with `ctx.user_id = nil` | Emit deny-all filter: `WHERE 1 = 0` (fail-closed) |
| Model has field named `ctx` (reserved namespace conflict) | Warning at startup; model works normally; `domain_filter_expr` unavailable for that model |

---

## §10. Impact Summary

```
Production code:
  Deleted:   ~919 lines  (evaluator.go + hook/expr.go)
  Rewritten: ~230 lines  (hydrator, control.go conditions, dag.go dedup)
  Added:     ~500 lines  (eval.go API, aggregate.go, record_rule_expr.go)
  Modified:  ~50 lines   (import/type updates across 5 files)

Tests:
  Rewritten: ~374 lines  (evaluator_test.go)
  Updated:   ~100 lines  (steps_test.go, dispatcher_test.go)
  New:       ~300 lines  (eval_test.go, record_rule_expr_test.go)

Docs:        6 files

Net production code: ~-189 lines (less code, more capability)
```

---

## §11. Risks and Mitigations

| Risk | Severity | Mitigation |
|------|----------|------------|
| `{{}}` syntax breaking change | Medium | Pre-processor strips `{{`/`}}` before eval; all existing expressions remain valid |
| Performance regression on computed fields | Low | expr-lang uses bytecode VM + compilation cache — faster than custom tree-walk evaluator |
| Aggregate `sum(lines.subtotal)` pattern change | Medium | Register custom expr-lang functions that match current behavior exactly |
| expr-lang error messages differ from current | Low | Wrap errors with context (already done in go-json `ExprEngine`) |
| Record rule `domain_filter_expr` migration | Low | Old `domain_filter` format unchanged; new format opt-in; migration tool provided |
| Reserved namespace `ctx` conflicts with existing field | Low | Validate at model registration; `ctx` is not a common field name |
| Tautology in record rules (`1 == 1`) | High | AST validation rejects expressions without field references |
| Empty `ctx.department_ids` → invalid SQL `IN ()` | Medium | Detect empty array → return deny-all filter (fail-closed) |
