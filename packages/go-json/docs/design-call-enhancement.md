# Design: `call` Step Enhancement — Unified Function Calling

**Status**: Draft (Reviewed)
**Date**: 29 April 2026
**Affects**: `lang/ast.go`, `lang/parser.go`, `lang/vm.go`, `lang/compiler.go`
**Review**: Reviewed by Confucius — 7 findings addressed (see §14)

---

## 1. Problem

The `call` step can only invoke **go-json defined functions** (in the `functions` block) and **struct methods**. It cannot call I/O module functions, stdlib namespaced functions, or extension functions:

```json
// ✅ Works — go-json defined function
{"call": "calculateDiscount", "with": {"price": "100", "tier": "'gold'"}}

// ✅ Works — struct method
{"call": "person.birthday"}

// ❌ Fails — I/O module (FUNC_NOT_FOUND or NOT_STRUCT)
{"call": "http.get", "with": {"url": "'https://api.com'"}}

// ❌ Fails — extension function
{"call": "bc.model", "with": {"name": "'lead'"}}
```

Users must use `expr` for I/O/extension calls, which creates two problems:

1. **No fire-and-forget for side effects** — must use `{"let": "_", "expr": "fs.write(...)"}` hack
2. **Inconsistent mental model** — user must know which functions are "program functions" vs "environment functions"

Additionally, `with` only accepts `map[string]string` (named expression args). This forces string literal wrapping with single quotes (`'...'`), which causes escaping issues with content containing quotes, backticks, or markdown.

---

## 2. Solution

### 2.1 `call` Falls Back to Scope Lookup

When `call` encounters a dot-path like `http.get`, it currently only checks struct methods. The enhancement adds a fallback: if the object has no `_type` (not a struct), check if the member is a callable function in the scope.

**Resolution order for `call "X.Y"`:**
1. Look up `X` in scope
2. If `X` is a struct (has `_type`) → call method `Y` (existing behavior)
3. If `X` is a `map[string]any` without `_type` → look up `Y` in the map, call it as a native Go function (NEW)
4. If neither → error

**Resolution order for `call "X"` (no dot):**
1. Look up `X` in `program.Functions` → call as go-json function (existing behavior)
2. Look up `X` in scope → if it's a callable function, call it (NEW)
3. If neither → error with "did you mean?" suggestions

### 2.2 `with` Supports Array (Positional Args)

Currently `with` is always `map[string]string`. The enhancement adds array support:

```json
// Object → named args, each value is expression string (EXISTING)
{"call": "calculateDiscount", "with": {"price": "100", "tier": "'gold'"}}

// Array → positional args, each element is expression string (NEW)
{"call": "http.get", "with": ["'https://api.com'", "myHeaders"]}
```

Detection is by JSON type — array vs object. Mutually exclusive, unambiguous.

### 2.3 `args` — Literal JSON Values (NEW key)

A new `args` key provides **literal JSON values** — no expression evaluation, no quote wrapping needed:

```json
// "args" → pure JSON values, like a REST API request body
{"call": "fs.write", "args": ["./log.txt", "Don't forget `backtick` and \"quotes\""]}
{"call": "redis.set", "args": ["user:123", {"name": "Alice", "age": 30}, 3600]}
```

**Rules:**
- `args` is always an **array** (positional)
- Each element is a **literal JSON value** — string is string, number is number, no evaluation
- `with` and `args` are **mutually exclusive** — using both is a compile error
- `args` works for all function types (go-json functions, methods, I/O, extensions)

**Why `args` and not overloading `with`?**

The key insight from the design discussion: in JSON, types are self-describing (`"Alice"` is a string, `42` is a number). But in `with`, every string is an expression. There's no way to distinguish "I want the literal string `Alice`" from "I want to evaluate the variable `Alice`" without wrapping in quotes (`'Alice'`).

`args` solves this cleanly: `"Alice"` in `args` is always the literal string `Alice`. If you need computed values, use `with`.

| Key | Element type | `"Alice"` means | `42` means | Use when |
|-----|-------------|----------------|-----------|----------|
| `with` (object) | expression string | variable lookup `Alice` | expression `42` → number | Named args, computed values |
| `with` (array) | expression string | variable lookup `Alice` | expression `42` → number | Positional args, computed values |
| `args` (array) | literal JSON value | string `"Alice"` | number `42` | Literal data, no computation needed |

---

## 3. Syntax Summary

After this enhancement, `call` supports these forms:

```json
// 1. Named expression args (EXISTING — unchanged)
{"call": "calculateDiscount", "with": {"price": "input.price", "tier": "input.tier"}}

// 2. Positional expression args (NEW)
{"call": "http.get", "with": ["url", "headers"]}
{"call": "calculateDiscount", "with": ["input.price", "5", "'gold'"]}

// 3. Positional literal args (NEW)
{"call": "fs.write", "args": ["./output.txt", "Hello, World!"]}
{"call": "redis.set", "args": ["user:123", {"name": "Alice"}, 3600]}

// 4. No args (EXISTING — unchanged)
{"call": "person.birthday"}

// 5. let + call — all forms work (EXISTING + NEW)
{"let": "resp", "call": "http.get", "with": ["url"]}
{"let": "resp", "call": "http.get", "args": ["https://api.example.com"]}
{"let": "disc", "call": "calculateDiscount", "with": {"price": "100", "tier": "'gold'"}}
```

---

## 4. AST Changes

### 4.1 CallNode

```go
// Before
type CallNode struct {
    NodeMeta
    Function string
    With     map[string]string
}

// After
type CallNode struct {
    NodeMeta
    Function string
    With     map[string]string // named expression args (object with)
    WithArgs []string          // positional expression args (array with)
    Args     []any             // positional literal args (args key)
}
```

### 4.2 LetNode

```go
// Before
type LetNode struct {
    // ...
    Call     string
    CallWith map[string]string
    // ...
}

// After
type LetNode struct {
    // ...
    Call         string
    CallWith     map[string]string // named expression args
    CallWithArgs []string          // positional expression args
    CallArgs     []any             // positional literal args
    // ...
}
```

### 4.3 Invariant

Exactly zero or one of `With`/`WithArgs`/`Args` is set. Multiple = compile error.

---

## 5. Parser Changes

### 5.1 `parseCallNode` — detect `with` type + `args`

```go
func parseCallNode(m map[string]any, index int) (*CallNode, error) {
    node := &CallNode{}
    node.StepIndex = index
    parseComment(&node.NodeMeta, m)

    funcName, ok := m["call"].(string)
    if !ok {
        return nil, CompileError("INVALID_CALL", "call function name must be a string", index)
    }
    node.Function = funcName

    hasWith := false
    hasArgs := false

    if withRaw, ok := m["with"]; ok {
        hasWith = true
        switch w := withRaw.(type) {
        case map[string]any:
            node.With = toStringMap(w)
        case []any:
            node.WithArgs = toStringSlice(w)
        default:
            return nil, CompileError("INVALID_WITH", "with must be an object or array", index)
        }
    }

    if argsRaw, ok := m["args"]; ok {
        hasArgs = true
        arr, ok := argsRaw.([]any)
        if !ok {
            return nil, CompileError("INVALID_ARGS", "args must be an array", index)
        }
        node.Args = arr
    }

    if hasWith && hasArgs {
        return nil, CompileError("WITH_ARGS_CONFLICT",
            "cannot use both 'with' and 'args' in the same call", index)
    }

    return node, nil
}
```

### 5.2 `parseLetCallNode` — same pattern

Same detection logic for `LetNode.CallWith`/`CallWithArgs`/`CallArgs`.

### 5.3 New helper: `toStringSlice`

```go
func toStringSlice(arr []any, stepIndex int) ([]string, error) {
    result := make([]string, len(arr))
    for i, v := range arr {
        switch val := v.(type) {
        case string:
            result[i] = val
        case float64:
            // JSON numbers → valid expression strings (42, 3.14)
            result[i] = fmt.Sprintf("%v", val)
        case bool:
            // JSON booleans → valid expression strings (true, false)
            result[i] = fmt.Sprintf("%v", val)
        case nil:
            result[i] = "nil"
        default:
            // Objects and arrays in expression position are ambiguous
            return nil, CompileError("INVALID_WITH_ELEMENT",
                fmt.Sprintf("with array element %d is %T — must be a string (expression). "+
                    "Did you mean to use 'args' for literal values?", i, v), stepIndex)
        }
    }
    return result, nil
}
```

This catches the common mistake of putting objects/arrays in `with` (expression mode) when the user meant `args` (literal mode). Primitives (number, bool, nil) are coerced to expression strings since they're unambiguous.

---

## 6. VM Changes

### 6.1 `callMethod` — fallback to namespace function with multi-level dot support

```go
func (vm *VM) callMethod(objectName, methodName string, ...) (any, error) {
    objVal, _, found := vm.scope.Get(objectName)
    if !found {
        return nil, RuntimeError("VAR_NOT_FOUND", ...)
    }

    obj, ok := objVal.(map[string]any)
    if !ok {
        return nil, RuntimeError("NOT_CALLABLE",
            fmt.Sprintf("cannot call method on %T — expected struct or namespace", objVal), ...)
    }

    typeName, _ := obj["_type"].(string)

    // NEW: If no _type, this is a namespace map (I/O module, extension, etc.)
    if typeName == "" {
        // Support multi-level dot paths: "db.query" in namespace
        // methodName could be "query" (simple) or "db.query" (nested)
        return vm.callNamespaceFunction(obj, methodName, ...)
    }

    // EXISTING: struct method dispatch
    // ... (unchanged)
}
```

### 6.1b `executeCall` — multi-level dot path dispatch

```go
func (vm *VM) executeCall(n *CallNode) error {
    if strings.Contains(n.Function, ".") {
        // Split on FIRST dot only: "bc.db.query" → root="bc", rest="db.query"
        parts := strings.SplitN(n.Function, ".", 2)
        _, err := vm.callMethod(parts[0], parts[1], n.With, n.WithArgs, n.Args, n.StepIndex)
        return err
    }
    _, err := vm.callFunction(n.Function, n.With, n.WithArgs, n.Args, n.StepIndex)
    return err
}
```

In `callNamespaceFunction`, the `funcName` parameter may contain dots (e.g., `"db.query"`). The function traverses nested maps:

```go
func (vm *VM) callNamespaceFunction(namespace map[string]any, funcPath string, ...) (any, error) {
    // Traverse nested namespaces: "db.query" → namespace["db"]["query"]
    current := namespace
    parts := strings.Split(funcPath, ".")
    
    for i := 0; i < len(parts)-1; i++ {
        next, exists := current[parts[i]]
        if !exists {
            return nil, RuntimeError("NAMESPACE_NOT_FOUND", ...)
        }
        nextMap, ok := next.(map[string]any)
        if !ok {
            return nil, RuntimeError("NOT_NAMESPACE",
                fmt.Sprintf("'%s' is not a namespace (type: %T)", parts[i], next), ...)
        }
        current = nextMap
    }
    
    funcName := parts[len(parts)-1]
    fnVal, exists := current[funcName]
    if !exists {
        // "Did you mean?" suggestions from current namespace level
        ...
    }
    
    // ... call fnVal ...
}
```

### 6.2 New: `callNamespaceFunction`

```go
func (vm *VM) callNamespaceFunction(
    namespace map[string]any,
    funcName string,
    withExprs map[string]string,  // named expression args (may be nil)
    withArgs []string,            // positional expression args (may be nil)
    args []any,                   // literal args (may be nil)
    stepIndex int,
) (any, error) {
    fnVal, exists := namespace[funcName]
    if !exists {
        // "Did you mean?" for namespace functions
        names := make([]string, 0, len(namespace))
        for k := range namespace {
            names = append(names, k)
        }
        gjErr := RuntimeError("FUNC_NOT_FOUND",
            fmt.Sprintf("function '%s' not found in namespace", funcName), stepIndex)
        if suggestions := SuggestSimilar(funcName, names, 3, 3); len(suggestions) > 0 {
            gjErr.WithSuggestions(suggestions...)
        }
        return nil, gjErr
    }

    // Try multiple function signatures — extensions may use typed signatures
    var callResult any
    var callErr error

    switch fn := fnVal.(type) {
    case func(...any) (any, error):
        // Standard I/O module signature
        callResult, callErr = fn(callArgs...)
    case func(args ...any) (any, error):
        callResult, callErr = fn(callArgs...)
    default:
        // Use reflect for arbitrary function signatures (extension functions)
        callResult, callErr = callWithReflect(fnVal, callArgs)
        if callErr != nil && strings.Contains(callErr.Error(), "not a function") {
            return nil, RuntimeError("NOT_CALLABLE",
                fmt.Sprintf("'%s' is not a callable function (type: %T)", funcName, fnVal), stepIndex)
        }
    }

    // Build positional args
    var callArgs []any

    if args != nil {
        // Literal mode — use as-is
        callArgs = args
    } else if withArgs != nil {
        // Positional expression mode — evaluate each
        callArgs = make([]any, len(withArgs))
        for i, expr := range withArgs {
            val, err := vm.evalExpr(expr, stepIndex)
            if err != nil {
                return nil, err
            }
            callArgs[i] = val
        }
    } else if withExprs != nil {
        // Named expression mode — evaluate values in order
        // Note: Go map iteration order is random, but for native functions
        // we just pass all values. The function receives them as variadic.
        callArgs = make([]any, 0, len(withExprs))
        for _, expr := range withExprs {
            val, err := vm.evalExpr(expr, stepIndex)
            if err != nil {
                return nil, err
            }
            callArgs = append(callArgs, val)
        }
    }

    // Depth tracking + stack trace
    vm.depth++
    defer func() { vm.depth-- }()
    if vm.depth > vm.limits.MaxDepth {
        return nil, LimitError("DEPTH_LIMIT", ...)
    }

    vm.callStack = append(vm.callStack, StackFrame{Function: funcName, Step: stepIndex})
    defer func() { vm.callStack = vm.callStack[:len(vm.callStack)-1] }()

    // Debugger hook
    if vm.debugger != nil {
        vm.debugger.OnFunctionCall(funcName, map[string]any{"args": callArgs})
    }

    result, err := fn(callArgs...)
    if err != nil {
        return nil, RuntimeError("NATIVE_CALL_ERROR", err.Error(), stepIndex).
            WithStack(vm.callStack)
    }

    if vm.debugger != nil {
        vm.debugger.OnFunctionReturn(funcName, result)
    }

    return result, nil
}
```

### 6.3 Update `callFunction` — support positional args + literal args

The existing `callFunction` (for go-json defined functions) also needs to support `WithArgs` and `Args`:

```go
func (vm *VM) callFunction(name string, withExprs map[string]string,
    withArgs []string, args []any, stepIndex int) (any, error) {

    // ... existing depth check, function lookup ...

    funcScope := vm.scope.IsolatedChild("func:" + name)

    if args != nil {
        // Literal mode — bind positionally
        for i, param := range fn.Params {
            if i < len(args) {
                funcScope.Declare(param.Name, args[i], param.Type)
            } else if param.HasDefault {
                funcScope.Declare(param.Name, param.Default, param.Type)
            } else {
                funcScope.Declare(param.Name, nil, param.Type)
            }
        }
    } else if withArgs != nil {
        // Positional expression mode — evaluate and bind
        for i, param := range fn.Params {
            if i < len(withArgs) {
                val, err := vm.evalExpr(withArgs[i], stepIndex)
                if err != nil {
                    return nil, err
                }
                funcScope.Declare(param.Name, val, param.Type)
            } else if param.HasDefault {
                funcScope.Declare(param.Name, param.Default, param.Type)
            } else {
                funcScope.Declare(param.Name, nil, param.Type)
            }
        }
    } else if withExprs != nil {
        // Named expression mode (existing behavior)
        for _, param := range fn.Params {
            if expr, ok := withExprs[param.Name]; ok {
                val, err := vm.evalExpr(expr, stepIndex)
                // ...
            }
        }
    }

    // ... rest unchanged ...
}
```

### 6.4 Update `executeCall` and `executeLet`

Pass the new fields through:

```go
func (vm *VM) executeCall(n *CallNode) error {
    if strings.Contains(n.Function, ".") {
        parts := strings.SplitN(n.Function, ".", 2)
        _, err := vm.callMethod(parts[0], parts[1], n.With, n.WithArgs, n.Args, n.StepIndex)
        return err
    }
    _, err := vm.callFunction(n.Function, n.With, n.WithArgs, n.Args, n.StepIndex)
    return err
}
```

### 6.5 Named `with` for Namespace Functions — Compile Error

When using `with` (object, named args) to call a namespace function, **Go map iteration order is not guaranteed**. Named args would arrive in random order to a positional native function — this is a **silent data corruption vector**, not just a cosmetic issue.

**Decision:** Named `with` (object) for namespace functions is a **compile error**:

```
compile error: cannot use named 'with' for namespace function 'http.get' — 
use array 'with' for expression args or 'args' for literal values
```

Detection: at runtime, when `callMethod` resolves to a namespace (no `_type`), check if `With` (object) was used. If so, return error.

Note: this cannot be caught at compile time because namespace resolution is runtime. The error is raised at the start of execution, before any args are evaluated.

For go-json defined functions and struct methods, named `with` (object) continues to work as before — they have declared parameter names.

---

## 7. Compiler Changes

### 7.1 Validate `with`/`args` mutual exclusivity

In the compiler's step validation (if any), verify that `With`, `WithArgs`, and `Args` are mutually exclusive.

### 7.2 Validate `args` element types

`args` elements must be valid JSON values. No validation needed beyond what JSON parsing already provides.

---

## 8. Backward Compatibility

| Existing code | Impact |
|---------------|--------|
| `{"call": "fn", "with": {"a": "expr"}}` | Unchanged — object `with` still works |
| `{"let": "x", "call": "fn", "with": {"a": "expr"}}` | Unchanged |
| `{"call": "obj.method"}` | Unchanged — struct methods still work |
| `{"call": "obj.method", "with": {"a": "expr"}}` | Unchanged |
| `{"let": "x", "expr": "http.get('...')"}` | Still works — `expr` is not affected |

No existing programs break. All changes are additive.

---

## 9. Examples — Before and After

### I/O Module Calls

```json
// BEFORE (only way)
{"let": "resp", "expr": "http.get('https://api.com/users')"}
{"let": "_", "expr": "fs.write('./log.txt', content)"}
{"let": "_", "expr": "redis.set('user:123', userData, 3600)"}

// AFTER — expression args
{"let": "resp", "call": "http.get", "with": ["url"]}
{"call": "fs.write", "with": ["'./log.txt'", "content"]}
{"call": "redis.set", "with": ["'user:' + id", "userData", "3600"]}

// AFTER — literal args (no escaping needed)
{"call": "fs.write", "args": ["./log.txt", "Don't forget `backtick`"]}
{"call": "redis.set", "args": ["user:123", {"name": "Alice"}, 3600]}
```

### go-json Function Calls

```json
// BEFORE (named args only)
{"call": "calculateDiscount", "with": {"price": "100", "tier": "'gold'"}}

// AFTER — also supports positional expression args
{"call": "calculateDiscount", "with": ["100", "'gold'"]}

// AFTER — also supports literal args
{"call": "calculateDiscount", "args": [100, "gold"]}
```

### Extension Calls

```json
// BEFORE
{"let": "leads", "expr": "bc.model('lead')"}

// AFTER
{"let": "leads", "call": "bc.model", "with": ["'lead'"]}
{"let": "leads", "call": "bc.model", "args": ["lead"]}
```

### Markdown Content (the escaping problem that motivated `args`)

```json
// BEFORE — escaping hell
{"let": "content", "value": "Don't forget `console.log()` for \"debugging\""},
{"let": "_", "expr": "fs.write('./doc.md', content)"}

// AFTER — clean with args
{"call": "fs.write", "args": ["./doc.md", "Don't forget `console.log()` for \"debugging\""]}
```

---

## 10. What This Does NOT Change

- `expr` mode is unaffected — still works for all function types
- `value` mode is unaffected
- `with` (object) for go-json functions is unaffected
- `new` + `with` is unaffected
- Expression-level function calls (`factorial(10)`, `http.get('...')`) are unaffected
- No changes to I/O module interface (`IOModule`, `Functions() map[string]any`)
- No changes to extension interface (`Extension`)

---

## 11. Edge Cases

| Case | Behavior |
|------|----------|
| `with` object + `args` in same step | Compile error: "cannot use both 'with' and 'args'" |
| `with` array + `args` in same step | Compile error: same |
| `args` with empty array `[]` | Valid — calls function with zero args |
| `with` array with empty array `[]` | Valid — calls function with zero args |
| `call "http.get"` without `with` or `args` | Calls `http.get()` with zero args — function decides if that's valid |
| `call "unknown.func"` where `unknown` not in scope | Runtime error: "variable 'unknown' not defined" |
| `call "http.nonexistent"` | Runtime error: "function 'nonexistent' not found in namespace" with suggestions |
| `call "http"` (no dot, not a program function) | Falls through to scope lookup — `http` is a map, not callable → error |
| `args` with nested objects | Valid — `[{"key": "value"}, [1,2,3]]` passed as-is |
| Named `with` for namespace function | Runtime error: "cannot use named 'with' for namespace function — use array 'with' or 'args'" |
| `call "X"` where X is both program function and scope variable | Program function wins (backward compat). Scope variable unreachable via `call` — use `expr` |
| `call "http.get"` but user shadowed `http` with `let "http"` | Error: "cannot call method on string — expected struct or namespace. Did you shadow an I/O module import?" |
| Native function panics | Recovered via `defer recover()`, converted to `NATIVE_CALL_ERROR` runtime error |
| Extra positional args beyond param count (go-json function) | Runtime warning logged, extra args silently dropped |
| `with` array with object/array elements | Compile error: "with array element must be expression string. Did you mean 'args'?" |
| `call "x.y.z"` (multi-dot) | Recursive traversal: look up `x` in scope → get map → look up `y` → get nested map → look up `z` → call function. Works for any depth. |

---

## 12. Files Changed

| File | Change |
|------|--------|
| `lang/ast.go` | Add `WithArgs []string`, `Args []any` to `CallNode`. Add `CallWithArgs []string`, `CallArgs []any` to `LetNode` |
| `lang/parser.go` | Update `parseCallNode`, `parseLetCallNode` to detect array `with` and `args`. Add `toStringSlice` helper |
| `lang/vm.go` | Update `callMethod` with namespace fallback. Add `callNamespaceFunction`. Update `callFunction` for positional/literal args. Update `executeCall`, `executeLet` to pass new fields |
| `lang/compiler.go` | Add mutual exclusivity validation for `with`/`args` |
| `lang/errors.go` | Add error codes: `WITH_ARGS_CONFLICT`, `NOT_CALLABLE`, `NATIVE_CALL_ERROR`, `NAMESPACE_NOT_FOUND`, `NOT_NAMESPACE`, `INVALID_WITH_ELEMENT`, `NAMED_WITH_NAMESPACE` |

**Estimated: ~300 lines added, ~50 lines modified.**

---

## 13. Test Plan

| Test | Description |
|------|-------------|
| `TestCall_WithArray_GoJsonFunction` | `call` with positional expression args to program function |
| `TestCall_WithArray_Method` | `call` with positional expression args to struct method |
| `TestCall_Args_Literal` | `call` with literal args — string, number, bool, null, object, array |
| `TestCall_Args_StringNoEscaping` | `args` with strings containing quotes, backticks, markdown |
| `TestCall_Args_NamespaceFunction` | `call` I/O-style namespace function with `args` |
| `TestCall_WithArray_NamespaceFunction` | `call` I/O-style namespace function with array `with` |
| `TestCall_NamespaceFallback` | `call "ns.func"` where `ns` is a map (not struct) |
| `TestCall_NamespaceNotFound` | `call "ns.nonexistent"` — error with suggestions |
| `TestCall_WithArgsConflict` | `with` + `args` in same step → compile error |
| `TestCall_LetCall_WithArray` | `let` + `call` with array `with` |
| `TestCall_LetCall_Args` | `let` + `call` with `args` |
| `TestCall_Args_Empty` | `call` with `args: []` |
| `TestCall_BackwardCompat` | Existing `with` object tests still pass |
| `TestCall_MultiLevelNamespace` | `call "bc.db.query"` with nested namespace maps |
| `TestCall_NamedWithNamespace_Error` | Named `with` (object) for namespace function → runtime error |
| `TestCall_NativePanicRecovery` | Native function that panics → recovered as runtime error |
| `TestCall_ShadowedImport_Error` | User shadows I/O import with `let` → clear error message |
| `TestCall_ExtraArgs_Warning` | More positional args than params → warning, extra dropped |
| `TestCall_WithArray_NonString_Error` | Object/array in `with` array → compile error |
| `TestCall_ReflectSignature` | Extension function with typed signature (not variadic) → works via reflect |

---

## 14. Review Findings & Resolutions

Design reviewed by Confucius. Critical and major findings addressed:

| # | Finding | Severity | Resolution |
|---|---------|----------|------------|
| 1 | Named `with` for namespace functions: warning allows silent data corruption | **Critical** | Changed to **runtime error**. Named `with` only works for go-json functions and struct methods (which have declared param names). §6.5 updated. |
| 2 | Type assertion only handles `func(...any)(any, error)` | **Critical** | Added `reflect`-based fallback for arbitrary function signatures. Extension functions with typed params (e.g., `func(id string)`) now work. §6.2 updated. |
| 3 | Multi-level namespaces (`bc.db.query`) broken | **Critical** | Implemented recursive dot-path traversal in `callNamespaceFunction`. §6.1, §6.1b updated. |
| 4 | Shadowed import produces confusing error | **Major** | Error message now hints: "Did you shadow an I/O module import?" §11 updated. |
| 5 | `args`+`with` mutual exclusivity blocks mixed literal/expression | **Major** | Documented workaround pattern: pre-compute with `let`, then pass via `args` or `with`. See §9 examples. |
| 6 | `toStringSlice` silently coerces objects/arrays | **Major** | Changed to compile error for non-primitive elements. Primitives (number, bool, nil) still coerced since they're unambiguous expressions. §5.3 updated. |
| 7 | No panic recovery for native functions | **Minor** | Added `defer recover()` in `callNamespaceFunction`. §11 updated. |
| 8 | Extra positional args silently dropped | **Minor** | Added runtime warning for arg count mismatch on go-json functions. §11 updated. |
| 10 | `ExecuteFunction` not addressed | **Major** | Out of scope — `ExecuteFunction` is used by server handler bridge with its own arg passing. This enhancement targets the `call` step in programs. |

### Mixed Literal/Expression Pattern (Finding #5)

When you need both literal strings and computed values in the same call:

```json
// Pattern: pre-compute expressions, then pass all as literals
{"let": "params", "expr": "[userName, minAge]"},
{"call": "sql.query", "args": ["SELECT * FROM users WHERE name = ? AND age > ?", "params_wont_work_here"]}

// Better: use with (expression mode) for everything
{"call": "sql.query", "with": ["'SELECT * FROM users WHERE name = ? AND age > ?'", "[userName, minAge]"]}

// Or: pre-compute the SQL too if it has escaping issues
{"let": "query", "value": "SELECT * FROM users WHERE name = ? AND age > ?"},
{"call": "sql.query", "with": ["query", "[userName, minAge]"]}
```

The `let` + `value` + variable reference pattern is the recommended approach for complex strings that would have escaping issues in expression mode.
