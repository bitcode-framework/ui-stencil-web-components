package lang

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"sync/atomic"
	"time"
)

// Sentinel types for control flow signals.
type returnValue struct{ value any }
type breakSignal struct{}
type continueSignal struct{}

type Logger interface {
	Log(level, message string, data map[string]any)
}

type VMOption func(*VM)

func WithContext(ctx context.Context) VMOption {
	return func(vm *VM) { vm.ctx = ctx }
}

func WithDebugger(d Debugger) VMOption {
	return func(vm *VM) { vm.debugger = d }
}

func WithLogger(l Logger) VMOption {
	return func(vm *VM) { vm.logger = l }
}

func WithTrace(enabled bool) VMOption {
	return func(vm *VM) {
		if enabled {
			vm.trace = NewExecutionTrace()
		}
	}
}

func WithSession(session map[string]any) VMOption {
	return func(vm *VM) { vm.session = session }
}

func WithExecutionID(id string) VMOption {
	return func(vm *VM) { vm.executionID = id }
}

type VM struct {
	program         *CompiledProgram
	engine          ExprEngine
	scope           *Scope
	ctx             context.Context
	cancel          context.CancelFunc
	debugger        Debugger
	logger          Logger
	trace           *ExecutionTrace
	stepCount       int
	depth           int
	callStack       []StackFrame
	limits          ResolvedLimits
	session         map[string]any
	executionID     string
	startTime       time.Time
	sharedStepCount *int64 // shared across parallel branches for global step limit
}

type ExecutionResult struct {
	Value    any
	Trace    []TraceEntry
	Steps    int
	Duration time.Duration
}

func NewVM(program *CompiledProgram, engine ExprEngine, opts ...VMOption) *VM {
	vm := &VM{
		program: program,
		engine:  engine,
		limits:  program.Limits,
	}

	for _, opt := range opts {
		opt(vm)
	}

	if vm.ctx == nil {
		vm.ctx = context.Background()
	}

	// Apply timeout.
	if vm.limits.Timeout > 0 {
		vm.ctx, vm.cancel = context.WithTimeout(vm.ctx, vm.limits.Timeout)
	}

	return vm
}

func (vm *VM) Execute(input map[string]any) (*ExecutionResult, error) {
	start := time.Now()
	vm.startTime = start

	if vm.cancel != nil {
		defer vm.cancel()
	}

	vm.scope = NewScope("main")

	if input != nil {
		vm.scope.Declare("input", input, "map")
	} else {
		vm.scope.Declare("input", map[string]any{}, "map")
	}

	if vm.session != nil {
		vm.scope.Declare("session", vm.session, "map")
	} else {
		vm.scope.Declare("session", map[string]any{}, "map")
	}

	vm.scope.Declare("execution", map[string]any{
		"id":         vm.executionID,
		"program":    vm.program.Name,
		"started_at": start.Format(time.RFC3339),
		"depth":      0,
		"step_count": 0,
	}, "map")

	// Inject constants into scope.
	if vm.program.AST.Constants != nil {
		for name, value := range vm.program.AST.Constants {
			vm.scope.Declare(name, value, "const")
		}
	}

	// Inject enums into scope.
	if vm.program.AST.Enums != nil {
		for name, def := range vm.program.AST.Enums {
			switch v := def.(type) {
			case []any:
				enumMap := make(map[string]any, len(v))
				for _, item := range v {
					s := fmt.Sprintf("%v", item)
					enumMap[s] = s
				}
				vm.scope.Declare(name, enumMap, "enum")
			case map[string]any:
				vm.scope.Declare(name, v, "enum")
			}
		}
	}

	// Register functions in scope for expression-level calls.
	for name, fn := range vm.program.Functions {
		wrapped := vm.wrapFunction(fn)
		vm.scope.Declare(name, wrapped, "func")
	}

	result, err := vm.executeSteps(vm.program.AST.Steps)
	if err != nil {
		return nil, err
	}

	// Unwrap return sentinel.
	var value any
	if rv, ok := result.(returnValue); ok {
		value = rv.value
	} else {
		value = result
	}

	if err := vm.checkOutputSize(value, -1); err != nil {
		return nil, err
	}

	er := &ExecutionResult{
		Value:    value,
		Steps:    vm.stepCount,
		Duration: time.Since(start),
	}
	if vm.trace != nil {
		er.Trace = vm.trace.Entries()
	}

	return er, nil
}

// ExecuteFunction calls a specific compiled function with named arguments.
// All program functions are available for inter-function calls.
func (vm *VM) ExecuteFunction(fn *CompiledFunc, args map[string]any, env map[string]any) (any, error) {
	start := time.Now()
	vm.startTime = start

	if vm.cancel != nil {
		defer vm.cancel()
	}

	vm.scope = NewScope("main")

	for k, v := range env {
		vm.scope.Declare(k, v, "any")
	}

	for name, ffn := range vm.program.Functions {
		wrapped := vm.wrapFunction(ffn)
		vm.scope.Declare(name, wrapped, "func")
	}

	funcScope := vm.scope.IsolatedChild("func:" + fn.Name)

	for _, param := range fn.Params {
		if val, ok := args[param.Name]; ok {
			funcScope.Declare(param.Name, val, param.Type)
		} else if param.HasDefault {
			funcScope.Declare(param.Name, param.Default, param.Type)
		} else {
			funcScope.Declare(param.Name, nil, param.Type)
		}
	}

	for name, ffn := range vm.program.Functions {
		wrapped := vm.wrapFunction(ffn)
		funcScope.Declare(name, wrapped, "func")
	}

	vm.callStack = append(vm.callStack, StackFrame{Function: fn.Name, Step: -1})

	result, err := vm.withScope(funcScope, func() (any, error) {
		return vm.executeSteps(fn.Steps)
	})
	if err != nil {
		return nil, err
	}

	if rv, ok := result.(returnValue); ok {
		return rv.value, nil
	}
	return nil, nil
}

func (vm *VM) executeSteps(steps []Node) (any, error) {
	for _, step := range steps {
		// Context check (timeout/cancellation).
		select {
		case <-vm.ctx.Done():
			return nil, LimitError("TIMEOUT",
				fmt.Sprintf("execution timeout (%s) exceeded at step %d", vm.limits.Timeout, step.Meta().StepIndex),
				step.Meta().StepIndex)
		default:
		}

		vm.stepCount++
		if vm.sharedStepCount != nil {
			if int(atomic.AddInt64(vm.sharedStepCount, 1)) > vm.limits.MaxSteps {
				return nil, LimitError("STEP_LIMIT",
					fmt.Sprintf("step limit (%d) exceeded", vm.limits.MaxSteps),
					step.Meta().StepIndex)
			}
		} else if vm.stepCount > vm.limits.MaxSteps {
			return nil, LimitError("STEP_LIMIT",
				fmt.Sprintf("step limit (%d) exceeded", vm.limits.MaxSteps),
				step.Meta().StepIndex)
		}

		// Debug hook.
		if vm.debugger != nil {
			action := vm.debugger.OnStep(StepInfo{
				Index: step.Meta().StepIndex,
				Type:  step.nodeType(),
				Node:  step,
			})
			if action == DebugPause {
				// In a real implementation, this would block.
				// For now, just continue.
			}
		}

		stepStart := time.Now()

		result, err := vm.executeStep(step)
		if err != nil {
			if vm.debugger != nil {
				vm.debugger.OnError(err, step.Meta().StepIndex)
			}
			return nil, err
		}

		if vm.trace != nil {
			entry := TraceEntry{
				Step:       step.Meta().StepIndex,
				Type:       step.nodeType(),
				DurationUs: time.Since(stepStart).Microseconds(),
			}
			vm.enrichTraceEntry(&entry, step, result)
			vm.trace.AddStep(entry)
		}

		// Handle control flow signals.
		if result != nil {
			switch result.(type) {
			case returnValue, breakSignal, continueSignal:
				return result, nil
			}
		}
	}
	return nil, nil
}

func (vm *VM) executeStep(node Node) (any, error) {
	switch n := node.(type) {
	case *LetNode:
		return nil, vm.executeLet(n)
	case *SetNode:
		return nil, vm.executeSet(n)
	case *IfNode:
		return vm.executeIf(n)
	case *SwitchNode:
		return vm.executeSwitch(n)
	case *ForNode:
		return vm.executeFor(n)
	case *WhileNode:
		return vm.executeWhile(n)
	case *ReturnNode:
		return vm.executeReturn(n)
	case *CallNode:
		return nil, vm.executeCall(n)
	case *TryNode:
		return vm.executeTry(n)
	case *ErrorNode:
		return nil, vm.executeError(n)
	case *LogNode:
		return nil, vm.executeLog(n)
	case *ParallelNode:
		return vm.executeParallel(n)
	case *MatchNode:
		return vm.executeMatch(n)
	case *SleepNode:
		return nil, vm.executeSleep(n)
	case *RetryNode:
		return vm.executeRetry(n)
	case *AssertNode:
		return nil, vm.executeAssert(n)
	case *BreakNode:
		return breakSignal{}, nil
	case *ContinueNode:
		return continueSignal{}, nil
	case *CommentNode:
		return nil, nil
	default:
		return nil, RuntimeError("UNKNOWN_STEP", fmt.Sprintf("unknown step type: %T", node), node.Meta().StepIndex)
	}
}

func (vm *VM) executeLet(n *LetNode) error {
	idx := n.StepIndex

	if err := vm.checkShadowing(n.Name, idx); err != nil {
		return err
	}

	if n.Call != "" {
		var result any
		var err error
		if strings.Contains(n.Call, ".") {
			parts := strings.SplitN(n.Call, ".", 2)
			result, err = vm.callMethodOrNamespace(parts[0], parts[1], n.CallWith, n.CallWithArgs, n.CallArgs, idx)
		} else {
			result, err = vm.callFunctionUnified(n.Call, n.CallWith, n.CallWithArgs, n.CallArgs, idx)
		}
		if err != nil {
			return err
		}
		typ := InferType(result)
		if n.Type != "" {
			typ = n.Type
		}
		return vm.scope.Declare(n.Name, result, typ)
	}

	if n.New != "" {
		result, err := vm.executeNew(n.New, n.NewWith, idx)
		if err != nil {
			return err
		}
		typ := n.New
		if n.Type != "" {
			typ = n.Type
		}
		return vm.scope.Declare(n.Name, result, typ)
	}

	var value any
	var err error

	if n.HasValue {
		value = n.Value
	} else if n.HasExpr {
		value, err = vm.evalExpr(n.Expr, idx)
		if err != nil {
			return err
		}
	} else if n.HasWith {
		value, err = vm.evalWith(n.With, idx)
		if err != nil {
			return err
		}
	}

	typ := InferType(value)
	if n.Type != "" {
		typ = n.Type
	}

	if err := vm.scope.Declare(n.Name, value, typ); err != nil {
		if gjErr, ok := err.(*GoJSONError); ok {
			gjErr.Step = idx
		}
		return err
	}

	if err := vm.checkVariableCount(idx); err != nil {
		return err
	}
	if err := vm.checkVariableSize(n.Name, value, idx); err != nil {
		return err
	}

	if vm.debugger != nil {
		vm.debugger.OnVariable(n.Name, value, vm.scope.Name())
	}

	return nil
}

func (vm *VM) executeSet(n *SetNode) error {
	idx := n.StepIndex

	var value any
	var err error

	if n.HasValue {
		value = n.Value
	} else if n.HasExpr {
		value, err = vm.evalExpr(n.Expr, idx)
		if err != nil {
			return err
		}
	} else if n.HasWith {
		value, err = vm.evalWith(n.With, idx)
		if err != nil {
			return err
		}
	}

	// Handle dot-path mutation: "a.b.c" or "items[0].name"
	if strings.Contains(n.Target, ".") || strings.Contains(n.Target, "[") {
		return vm.setNestedProperty(n.Target, value, idx)
	}

	newType := InferType(value)
	if err := vm.scope.Set(n.Target, value, newType); err != nil {
		if gjErr, ok := err.(*GoJSONError); ok {
			gjErr.Step = idx
		}
		return err
	}

	if vm.debugger != nil {
		vm.debugger.OnVariable(n.Target, value, vm.scope.Name())
	}

	return nil
}

func (vm *VM) executeIf(n *IfNode) (any, error) {
	condResult, err := vm.evalExpr(n.Condition, n.StepIndex)
	if err != nil {
		return nil, err
	}

	if toBool(condResult) {
		childScope := vm.scope.NewChild("if-then")
		return vm.withScope(childScope, func() (any, error) {
			return vm.executeSteps(n.Then)
		})
	}

	for _, elif := range n.Elif {
		condResult, err := vm.evalExpr(elif.Condition, n.StepIndex)
		if err != nil {
			return nil, err
		}
		if toBool(condResult) {
			childScope := vm.scope.NewChild("elif")
			return vm.withScope(childScope, func() (any, error) {
				return vm.executeSteps(elif.Then)
			})
		}
	}

	if len(n.Else) > 0 {
		childScope := vm.scope.NewChild("else")
		return vm.withScope(childScope, func() (any, error) {
			return vm.executeSteps(n.Else)
		})
	}

	return nil, nil
}

func (vm *VM) executeSwitch(n *SwitchNode) (any, error) {
	exprResult, err := vm.evalExpr(n.Expr, n.StepIndex)
	if err != nil {
		return nil, err
	}

	key := fmt.Sprintf("%v", exprResult)

	if steps, ok := n.Cases[key]; ok {
		childScope := vm.scope.NewChild("case-" + key)
		return vm.withScope(childScope, func() (any, error) {
			return vm.executeSteps(steps)
		})
	}

	if steps, ok := n.Cases["default"]; ok {
		childScope := vm.scope.NewChild("case-default")
		return vm.withScope(childScope, func() (any, error) {
			return vm.executeSteps(steps)
		})
	}

	return nil, nil
}

func (vm *VM) executeFor(n *ForNode) (any, error) {
	idx := n.StepIndex

	var items []any

	if n.In != "" {
		result, err := vm.evalExpr(n.In, idx)
		if err != nil {
			return nil, err
		}
		arr, ok := result.([]any)
		if !ok {
			return nil, RuntimeError("NOT_ITERABLE",
				fmt.Sprintf("for-in expression must evaluate to array, got %T", result), idx)
		}
		items = arr
	} else if n.Range != nil {
		generated, err := generateRange(n.Range)
		if err != nil {
			return nil, RuntimeError("INVALID_RANGE", err.Error(), idx)
		}
		items = generated
	}

	for i, item := range items {
		if i >= vm.limits.MaxLoopIterations {
			return nil, LimitError("LOOP_LIMIT",
				fmt.Sprintf("loop iteration limit (%d) exceeded", vm.limits.MaxLoopIterations), idx)
		}

		childScope := vm.scope.NewChild("for-iteration")
		childScope.Declare(n.Variable, item, InferType(item))
		if n.Index != "" {
			childScope.Declare(n.Index, i, "int")
		}

		result, err := vm.withScope(childScope, func() (any, error) {
			return vm.executeSteps(n.Steps)
		})
		if err != nil {
			return nil, err
		}

		if result != nil {
			switch result.(type) {
			case breakSignal:
				return nil, nil
			case continueSignal:
				continue
			case returnValue:
				return result, nil
			}
		}
	}

	return nil, nil
}

func (vm *VM) executeWhile(n *WhileNode) (any, error) {
	idx := n.StepIndex
	iterations := 0

	for {
		if iterations >= vm.limits.MaxLoopIterations {
			return nil, LimitError("LOOP_LIMIT",
				fmt.Sprintf("loop iteration limit (%d) exceeded", vm.limits.MaxLoopIterations), idx)
		}

		condResult, err := vm.evalExpr(n.Condition, idx)
		if err != nil {
			return nil, err
		}

		if !toBool(condResult) {
			break
		}

		childScope := vm.scope.NewChild("while-iteration")
		result, err := vm.withScope(childScope, func() (any, error) {
			return vm.executeSteps(n.Steps)
		})
		if err != nil {
			return nil, err
		}

		if result != nil {
			switch result.(type) {
			case breakSignal:
				return nil, nil
			case continueSignal:
				iterations++
				continue
			case returnValue:
				return result, nil
			}
		}

		iterations++
	}

	return nil, nil
}

func (vm *VM) executeReturn(n *ReturnNode) (any, error) {
	idx := n.StepIndex

	if n.HasNew {
		result, err := vm.executeNew(n.New, n.NewWith, idx)
		if err != nil {
			return nil, err
		}
		return returnValue{value: result}, nil
	}

	if n.HasValue {
		return returnValue{value: n.Value}, nil
	}

	if n.HasExpr {
		result, err := vm.evalExpr(n.Expr, idx)
		if err != nil {
			return nil, err
		}
		return returnValue{value: result}, nil
	}

	if n.HasWith {
		result, err := vm.evalWith(n.With, idx)
		if err != nil {
			return nil, err
		}
		return returnValue{value: result}, nil
	}

	return returnValue{value: nil}, nil
}

func (vm *VM) executeCall(n *CallNode) error {
	if strings.Contains(n.Function, ".") {
		parts := strings.SplitN(n.Function, ".", 2)
		_, err := vm.callMethodOrNamespace(parts[0], parts[1], n.With, n.WithArgs, n.Args, n.StepIndex)
		return err
	}
	_, err := vm.callFunctionUnified(n.Function, n.With, n.WithArgs, n.Args, n.StepIndex)
	return err
}

func (vm *VM) executeTry(n *TryNode) (any, error) {
	var tryResult any
	var tryErr error

	func() {
		defer func() {
			if r := recover(); r != nil {
				tryErr = RuntimeError("PANIC", fmt.Sprintf("%v", r), n.StepIndex)
			}
		}()

		childScope := vm.scope.NewChild("try")
		tryResult, tryErr = vm.withScope(childScope, func() (any, error) {
			return vm.executeSteps(n.Try)
		})
	}()

	if tryErr != nil && n.Catch != nil {
		errObj := vm.normalizeError(tryErr, n.StepIndex)

		catchScope := vm.scope.NewChild("catch")
		catchScope.Declare(n.Catch.As, errObj, "map")

		catchResult, catchErr := vm.withScope(catchScope, func() (any, error) {
			return vm.executeSteps(n.Catch.Steps)
		})

		if n.Finally != nil {
			finallyScope := vm.scope.NewChild("finally")
			_, finallyErr := vm.withScope(finallyScope, func() (any, error) {
				return vm.executeSteps(n.Finally)
			})
			if finallyErr != nil {
				return nil, finallyErr
			}
		}

		if catchErr != nil {
			return nil, catchErr
		}
		return catchResult, nil
	}

	if n.Finally != nil {
		finallyScope := vm.scope.NewChild("finally")
		_, finallyErr := vm.withScope(finallyScope, func() (any, error) {
			return vm.executeSteps(n.Finally)
		})
		if finallyErr != nil {
			return nil, finallyErr
		}
	}

	if tryErr != nil {
		return nil, tryErr
	}

	return tryResult, nil
}

func (vm *VM) executeError(n *ErrorNode) error {
	idx := n.StepIndex

	if n.IsStructured {
		code := "ERROR"
		message := ""
		var details any

		if n.Code != "" {
			result, err := vm.evalExpr(n.Code, idx)
			if err != nil {
				return err
			}
			code = fmt.Sprintf("%v", result)
		}
		if n.Message != "" {
			result, err := vm.evalExpr(n.Message, idx)
			if err != nil {
				return err
			}
			message = fmt.Sprintf("%v", result)
		}
		if n.Details != "" {
			result, err := vm.evalExpr(n.Details, idx)
			if err != nil {
				return err
			}
			details = result
		}

		return RuntimeError(code, message, idx).
			WithContext(map[string]any{"details": details}).
			WithStack(vm.callStack)
	}

	// Simple string error.
	result, err := vm.evalExpr(n.Message, idx)
	if err != nil {
		return err
	}
	return RuntimeError("ERROR", fmt.Sprintf("%v", result), idx).
		WithStack(vm.callStack)
}

func (vm *VM) executeLog(n *LogNode) error {
	idx := n.StepIndex

	if vm.logger == nil {
		return nil
	}

	if n.IsStructured {
		message := ""
		level := "info"
		var data map[string]any

		if n.Message != "" {
			result, err := vm.evalExpr(n.Message, idx)
			if err != nil {
				return err
			}
			message = fmt.Sprintf("%v", result)
		}
		if n.Level != "" {
			result, err := vm.evalExpr(n.Level, idx)
			if err != nil {
				return err
			}
			level = fmt.Sprintf("%v", result)
		}
		if n.Data != nil {
			data = make(map[string]any)
			for k, expr := range n.Data {
				result, err := vm.evalExpr(expr, idx)
				if err != nil {
					return err
				}
				data[k] = result
			}
		}

		vm.logger.Log(level, message, data)
		return nil
	}

	// Simple string log.
	result, err := vm.evalExpr(n.Message, idx)
	if err != nil {
		return err
	}
	vm.logger.Log("info", fmt.Sprintf("%v", result), nil)
	return nil
}

// --- Function calling ---

func (vm *VM) callFunction(name string, withExprs map[string]string, stepIndex int) (any, error) {
	// Depth limit check.
	vm.depth++
	defer func() { vm.depth-- }()

	if vm.depth > vm.limits.MaxDepth {
		return nil, LimitError("DEPTH_LIMIT",
			fmt.Sprintf("call depth limit (%d) exceeded at function '%s'", vm.limits.MaxDepth, name),
			stepIndex).WithStack(vm.callStack)
	}

	fn, ok := vm.program.Functions[name]
	if !ok {
		// "Did you mean?" suggestion.
		funcNames := make([]string, 0, len(vm.program.Functions))
		for n := range vm.program.Functions {
			funcNames = append(funcNames, n)
		}
		gjErr := RuntimeError("FUNC_NOT_FOUND",
			fmt.Sprintf("function '%s' not defined", name), stepIndex)
		if suggestions := SuggestSimilar(name, funcNames, 3, 3); len(suggestions) > 0 {
			gjErr.WithSuggestions(suggestions...)
		}
		return nil, gjErr
	}

	// Push call stack.
	vm.callStack = append(vm.callStack, StackFrame{Function: name, Step: stepIndex})
	defer func() { vm.callStack = vm.callStack[:len(vm.callStack)-1] }()

	if vm.debugger != nil {
		args := make(map[string]any)
		if withExprs != nil {
			for k, v := range withExprs {
				args[k] = v
			}
		}
		vm.debugger.OnFunctionCall(name, args)
	}

	// Create isolated scope.
	funcScope := vm.scope.IsolatedChild("func:" + name)

	// Evaluate arg expressions in CALLER scope, bind in FUNCTION scope.
	for _, param := range fn.Params {
		if withExprs != nil {
			if expr, ok := withExprs[param.Name]; ok {
				val, err := vm.evalExpr(expr, stepIndex)
				if err != nil {
					return nil, err
				}
				funcScope.Declare(param.Name, val, param.Type)
				continue
			}
		}
		if param.HasDefault {
			funcScope.Declare(param.Name, param.Default, param.Type)
		} else {
			funcScope.Declare(param.Name, nil, param.Type)
		}
	}

	// Register functions for recursion.
	for fname, ffn := range vm.program.Functions {
		wrapped := vm.wrapFunction(ffn)
		funcScope.Declare(fname, wrapped, "func")
	}

	// Execute function steps.
	result, err := vm.withScope(funcScope, func() (any, error) {
		return vm.executeSteps(fn.Steps)
	})
	if err != nil {
		return nil, err
	}

	// Unwrap return sentinel.
	var retVal any
	if rv, ok := result.(returnValue); ok {
		retVal = rv.value
	}

	if vm.debugger != nil {
		vm.debugger.OnFunctionReturn(name, retVal)
	}

	return retVal, nil
}

func (vm *VM) callFunctionUnified(name string, namedWith map[string]string, positionalWith []string, literalArgs []any, stepIndex int) (any, error) {
	if len(namedWith) > 0 {
		return vm.callFunction(name, namedWith, stepIndex)
	}

	fn, ok := vm.program.Functions[name]
	if !ok {
		// Dynamic call_ref: check if name is a variable containing a function name or callable.
		if val, _, found := vm.scope.Get(name); found {
			switch v := val.(type) {
			case string:
				fn, ok = vm.program.Functions[v]
				if !ok {
					if scopeVal, _, scopeFound := vm.scope.Get(v); scopeFound {
						if callable, isFunc := scopeVal.(func(...any) (any, error)); isFunc {
							return vm.callNativeFunc(callable, positionalWith, literalArgs, stepIndex)
						}
					}
					return nil, RuntimeError("FUNC_NOT_FOUND",
						fmt.Sprintf("function '%s' (resolved from variable '%s') not defined", v, name), stepIndex)
				}
			case func(...any) (any, error):
				return vm.callNativeFunc(v, positionalWith, literalArgs, stepIndex)
			}
		}

		if !ok {
			funcNames := make([]string, 0, len(vm.program.Functions))
			for n := range vm.program.Functions {
				funcNames = append(funcNames, n)
			}
			gjErr := RuntimeError("FUNC_NOT_FOUND",
				fmt.Sprintf("function '%s' not defined", name), stepIndex)
			if suggestions := SuggestSimilar(name, funcNames, 3, 3); len(suggestions) > 0 {
				gjErr.WithSuggestions(suggestions...)
			}
			return nil, gjErr
		}
	}

	vm.depth++
	defer func() { vm.depth-- }()

	if vm.depth > vm.limits.MaxDepth {
		return nil, LimitError("DEPTH_LIMIT",
			fmt.Sprintf("call depth limit (%d) exceeded at function '%s'", vm.limits.MaxDepth, name),
			stepIndex).WithStack(vm.callStack)
	}

	vm.callStack = append(vm.callStack, StackFrame{Function: name, Step: stepIndex})
	defer func() { vm.callStack = vm.callStack[:len(vm.callStack)-1] }()

	if vm.debugger != nil {
		args := make(map[string]any)
		if positionalWith != nil {
			for i, expr := range positionalWith {
				if i < len(fn.Params) {
					args[fn.Params[i].Name] = expr
				}
			}
		} else if literalArgs != nil {
			for i, val := range literalArgs {
				if i < len(fn.Params) {
					args[fn.Params[i].Name] = val
				}
			}
		}
		vm.debugger.OnFunctionCall(name, args)
	}

	funcScope := vm.scope.IsolatedChild("func:" + name)

	if len(positionalWith) > 0 {
		for i, param := range fn.Params {
			if i < len(positionalWith) {
				val, err := vm.evalExpr(positionalWith[i], stepIndex)
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
	} else if len(literalArgs) > 0 {
		for i, param := range fn.Params {
			if i < len(literalArgs) {
				funcScope.Declare(param.Name, literalArgs[i], param.Type)
			} else if param.HasDefault {
				funcScope.Declare(param.Name, param.Default, param.Type)
			} else {
				funcScope.Declare(param.Name, nil, param.Type)
			}
		}
	} else {
		for _, param := range fn.Params {
			if param.HasDefault {
				funcScope.Declare(param.Name, param.Default, param.Type)
			} else {
				funcScope.Declare(param.Name, nil, param.Type)
			}
		}
	}

	for fname, ffn := range vm.program.Functions {
		wrapped := vm.wrapFunction(ffn)
		funcScope.Declare(fname, wrapped, "func")
	}

	result, err := vm.withScope(funcScope, func() (any, error) {
		return vm.executeSteps(fn.Steps)
	})
	if err != nil {
		return nil, err
	}

	var retVal any
	if rv, ok := result.(returnValue); ok {
		retVal = rv.value
	}

	if vm.debugger != nil {
		vm.debugger.OnFunctionReturn(name, retVal)
	}

	return retVal, nil
}

func (vm *VM) callMethodOrNamespace(objectName, memberPath string,
	withExprs map[string]string, withArgs []string, args []any, stepIndex int,
) (any, error) {
	objVal, _, found := vm.scope.Get(objectName)
	if !found {
		// Fallback: check inside input map (for I/O modules and extensions injected by runtime).
		if inputVal, _, inputFound := vm.scope.Get("input"); inputFound {
			if inputMap, ok := inputVal.(map[string]any); ok {
				if nsVal, exists := inputMap[objectName]; exists {
					objVal = nsVal
					found = true
				}
			}
		}
	}
	if !found {
		return nil, RuntimeError("VAR_NOT_FOUND",
			fmt.Sprintf("variable '%s' not defined", objectName), stepIndex)
	}

	obj, ok := objVal.(map[string]any)
	if !ok {
		return nil, RuntimeError("NOT_CALLABLE",
			fmt.Sprintf("cannot call method on %T — expected struct or namespace. "+
				"Did you shadow an I/O module import?", objVal), stepIndex)
	}

	typeName, _ := obj["_type"].(string)
	if typeName != "" {
		return vm.callMethod(objectName, memberPath, withExprs, stepIndex)
	}

	if withExprs != nil && len(withExprs) > 0 {
		return nil, RuntimeError("NAMED_WITH_NAMESPACE",
			fmt.Sprintf("cannot use named 'with' for namespace function '%s.%s' — "+
				"use array 'with' for expression args or 'args' for literal values",
				objectName, memberPath), stepIndex)
	}

	return vm.callNamespaceFunction(obj, objectName, memberPath, withArgs, args, stepIndex)
}

func (vm *VM) callNamespaceFunction(namespace map[string]any, nsName, funcPath string,
	withArgs []string, args []any, stepIndex int,
) (any, error) {
	current := namespace
	parts := strings.Split(funcPath, ".")

	for i := 0; i < len(parts)-1; i++ {
		next, exists := current[parts[i]]
		if !exists {
			return nil, RuntimeError("NAMESPACE_NOT_FOUND",
				fmt.Sprintf("'%s' not found in namespace '%s'", parts[i], nsName), stepIndex)
		}
		nextMap, ok := next.(map[string]any)
		if !ok {
			return nil, RuntimeError("NOT_NAMESPACE",
				fmt.Sprintf("'%s.%s' is not a namespace (type: %T)", nsName, parts[i], next), stepIndex)
		}
		current = nextMap
	}

	funcName := parts[len(parts)-1]
	fnVal, exists := current[funcName]
	if !exists {
		names := make([]string, 0, len(current))
		for k := range current {
			names = append(names, k)
		}
		gjErr := RuntimeError("FUNC_NOT_FOUND",
			fmt.Sprintf("function '%s' not found in namespace '%s'", funcName, nsName), stepIndex)
		if suggestions := SuggestSimilar(funcName, names, 3, 3); len(suggestions) > 0 {
			gjErr.WithSuggestions(suggestions...)
		}
		return nil, gjErr
	}

	var callArgs []any
	if args != nil {
		callArgs = args
	} else if withArgs != nil {
		callArgs = make([]any, len(withArgs))
		for i, expr := range withArgs {
			val, err := vm.evalExpr(expr, stepIndex)
			if err != nil {
				return nil, err
			}
			callArgs[i] = val
		}
	}

	vm.depth++
	defer func() { vm.depth-- }()
	if vm.depth > vm.limits.MaxDepth {
		return nil, LimitError("DEPTH_LIMIT",
			fmt.Sprintf("call depth limit (%d) exceeded at '%s.%s'", vm.limits.MaxDepth, nsName, funcPath),
			stepIndex)
	}

	fullName := nsName + "." + funcPath
	vm.callStack = append(vm.callStack, StackFrame{Function: fullName, Step: stepIndex})
	defer func() { vm.callStack = vm.callStack[:len(vm.callStack)-1] }()

	if vm.debugger != nil {
		debugArgs := make(map[string]any)
		for i, a := range callArgs {
			debugArgs[fmt.Sprintf("arg%d", i)] = a
		}
		vm.debugger.OnFunctionCall(fullName, debugArgs)
	}

	callable, ok := fnVal.(func(...any) (any, error))
	if !ok {
		return nil, RuntimeError("NOT_CALLABLE",
			fmt.Sprintf("'%s' is not a function (type: %T)", fullName, fnVal), stepIndex)
	}

	var result any
	var callErr error
	func() {
		defer func() {
			if r := recover(); r != nil {
				callErr = RuntimeError("PANIC",
					fmt.Sprintf("panic in namespace function '%s': %v", fullName, r), stepIndex)
			}
		}()
		result, callErr = callable(callArgs...)
	}()

	if vm.debugger != nil {
		vm.debugger.OnFunctionReturn(fullName, result)
	}

	return result, callErr
}

// callMethodUnified calls a struct method with positional with or literal args.
func (vm *VM) callMethodUnified(objectName, methodName string, positionalWith []string, literalArgs []any, stepIndex int) (any, error) {
	objVal, _, _ := vm.scope.Get(objectName)
	obj := objVal.(map[string]any)
	typeName := obj["_type"].(string)
	cs := vm.program.Structs[typeName]
	cm := cs.Methods[methodName]

	vm.depth++
	defer func() { vm.depth-- }()

	if vm.depth > vm.limits.MaxDepth {
		return nil, LimitError("DEPTH_LIMIT",
			fmt.Sprintf("call depth limit (%d) exceeded at method '%s.%s'", vm.limits.MaxDepth, typeName, methodName),
			stepIndex)
	}

	vm.callStack = append(vm.callStack, StackFrame{Function: typeName + "." + methodName, Step: stepIndex})
	defer func() { vm.callStack = vm.callStack[:len(vm.callStack)-1] }()

	methodScope := vm.scope.IsolatedChild("method:" + typeName + "." + methodName)
	methodScope.Declare("self", obj, typeName)

	if len(positionalWith) > 0 {
		for i, param := range cm.Params {
			if i < len(positionalWith) {
				val, err := vm.evalExpr(positionalWith[i], stepIndex)
				if err != nil {
					return nil, err
				}
				methodScope.Declare(param.Name, val, param.Type)
			} else if param.HasDefault {
				methodScope.Declare(param.Name, param.Default, param.Type)
			} else {
				methodScope.Declare(param.Name, nil, param.Type)
			}
		}
	} else if len(literalArgs) > 0 {
		for i, param := range cm.Params {
			if i < len(literalArgs) {
				methodScope.Declare(param.Name, literalArgs[i], param.Type)
			} else if param.HasDefault {
				methodScope.Declare(param.Name, param.Default, param.Type)
			} else {
				methodScope.Declare(param.Name, nil, param.Type)
			}
		}
	} else {
		for _, param := range cm.Params {
			if param.HasDefault {
				methodScope.Declare(param.Name, param.Default, param.Type)
			} else {
				methodScope.Declare(param.Name, nil, param.Type)
			}
		}
	}

	for fname, ffn := range vm.program.Functions {
		wrapped := vm.wrapFunction(ffn)
		methodScope.Declare(fname, wrapped, "func")
	}

	result, err := vm.withScope(methodScope, func() (any, error) {
		return vm.executeSteps(cm.Steps)
	})
	if err != nil {
		return nil, err
	}

	selfVal, _, _ := methodScope.Get("self")
	if selfMap, ok := selfVal.(map[string]any); ok {
		for k, v := range selfMap {
			obj[k] = v
		}
	}

	if rv, ok := result.(returnValue); ok {
		return rv.value, nil
	}
	return obj, nil
}

// wrapFunction wraps a CompiledFunc as a Go func for expr-lang expression-level calls.
// Positional params: createUser('Alice', 30) → name='Alice', age=30
func (vm *VM) wrapFunction(fn *CompiledFunc) func(...any) (any, error) {
	return func(args ...any) (any, error) {
		funcScope := vm.scope.IsolatedChild("func:" + fn.Name)

		for i, param := range fn.Params {
			if i < len(args) {
				funcScope.Declare(param.Name, args[i], param.Type)
			} else if param.HasDefault {
				funcScope.Declare(param.Name, param.Default, param.Type)
			} else {
				funcScope.Declare(param.Name, nil, param.Type)
			}
		}

		// Register functions for recursion.
		for fname, ffn := range vm.program.Functions {
			wrapped := vm.wrapFunction(ffn)
			funcScope.Declare(fname, wrapped, "func")
		}

		vm.depth++
		defer func() { vm.depth-- }()

		if vm.depth > vm.limits.MaxDepth {
			return nil, LimitError("DEPTH_LIMIT",
				fmt.Sprintf("call depth limit (%d) exceeded at function '%s'", vm.limits.MaxDepth, fn.Name), -1)
		}

		vm.callStack = append(vm.callStack, StackFrame{Function: fn.Name, Step: -1})
		defer func() { vm.callStack = vm.callStack[:len(vm.callStack)-1] }()

		result, err := vm.withScope(funcScope, func() (any, error) {
			return vm.executeSteps(fn.Steps)
		})
		if err != nil {
			return nil, err
		}

		if rv, ok := result.(returnValue); ok {
			return rv.value, nil
		}
		return nil, nil
	}
}

// --- Parallel execution ---

func (vm *VM) executeParallel(n *ParallelNode) (any, error) {
	if len(n.Branches) == 0 {
		if n.Into != "" {
			vm.scope.Declare(n.Into, map[string]any{}, "map")
		}
		return nil, nil
	}

	join := n.Join
	if join == "" {
		join = "all"
	}
	onError := n.OnError
	if onError == "" {
		onError = "cancel_all"
	}

	type branchResult struct {
		name  string
		value any
		err   error
	}

	ctx, cancel := context.WithCancel(vm.ctx)
	defer cancel()

	resultCh := make(chan branchResult, len(n.Branches))

	parentEnv := vm.scope.ToMap()

	var sharedSteps int64
	if vm.sharedStepCount != nil {
		sharedSteps = atomic.LoadInt64(vm.sharedStepCount)
	} else {
		sharedSteps = int64(vm.stepCount)
	}
	parentShared := &sharedSteps

	for branchName, steps := range n.Branches {
		bName := branchName
		bSteps := steps
		go func() {
			defer func() {
				if r := recover(); r != nil {
					resultCh <- branchResult{name: bName, err: RuntimeError("PANIC", fmt.Sprintf("%v", r), n.StepIndex)}
				}
			}()

			branchVM := &VM{
				program:         vm.program,
				engine:          vm.engine,
				ctx:             ctx,
				debugger:        vm.debugger,
				logger:          vm.logger,
				limits:          vm.limits,
				sharedStepCount: parentShared,
			}

			branchScope := NewScope("parallel:" + bName)
			for k, v := range parentEnv {
				branchScope.Declare(k, v, InferType(v))
			}
			branchVM.scope = branchScope

			result, err := branchVM.executeSteps(bSteps)
			if err != nil {
				resultCh <- branchResult{name: bName, err: err}
				return
			}

			var val any
			if rv, ok := result.(returnValue); ok {
				val = rv.value
			}
			resultCh <- branchResult{name: bName, value: val}
		}()
	}

	results := make(map[string]any)
	branchCount := len(n.Branches)
	var firstErr error
	collected := 0

	for collected < branchCount {
		br := <-resultCh
		collected++
		if br.err != nil {
			if join == "settled" {
				results[br.name] = map[string]any{
					"error":   true,
					"message": br.err.Error(),
				}
			} else {
				switch onError {
				case "cancel_all":
					cancel()
					for collected < branchCount {
						extra := <-resultCh
						collected++
						if extra.err == nil {
							results[extra.name] = extra.value
						}
					}
					if n.Into != "" {
						results[br.name] = nil
						if vm.scope.Has(n.Into) {
							vm.scope.Set(n.Into, results, "map")
						} else {
							vm.scope.Declare(n.Into, results, "map")
						}
					}
					return nil, br.err
				case "continue":
					results[br.name] = nil
				case "collect":
					results[br.name] = map[string]any{
						"error":   true,
						"message": br.err.Error(),
					}
				}
			}
			if firstErr == nil {
				firstErr = br.err
			}
		} else {
			results[br.name] = br.value

			if join == "any" {
				cancel()
				for collected < branchCount {
					<-resultCh
					collected++
				}
				break
			}
		}
	}

	if n.Into != "" {
		if vm.scope.Has(n.Into) {
			vm.scope.Set(n.Into, results, "map")
		} else {
			vm.scope.Declare(n.Into, results, "map")
		}
	}

	return nil, nil
}

// --- Struct construction ---

func (vm *VM) executeNew(structName string, withArgs map[string]any, stepIndex int) (map[string]any, error) {
	cs, ok := vm.program.Structs[structName]
	if !ok {
		structNames := make([]string, 0, len(vm.program.Structs))
		for n := range vm.program.Structs {
			structNames = append(structNames, n)
		}
		gjErr := RuntimeError("STRUCT_NOT_FOUND",
			fmt.Sprintf("struct '%s' not defined", structName), stepIndex)
		if suggestions := SuggestSimilar(structName, structNames, 3, 3); len(suggestions) > 0 {
			gjErr.WithSuggestions(suggestions...)
		}
		return nil, gjErr
	}

	instance := map[string]any{
		"_type": structName,
	}

	for fieldName, fd := range cs.Fields {
		if withArgs != nil {
			if arg, ok := withArgs[fieldName]; ok {
				val, err := vm.evalNewArg(arg, stepIndex)
				if err != nil {
					return nil, err
				}
				if err := validateFieldType(structName, fieldName, fd, val); err != nil {
					return nil, err.(*GoJSONError).AtStep(stepIndex)
				}
				instance[fieldName] = val
				continue
			}
		}
		if fd.HasDefault {
			switch def := fd.Default.(type) {
			case string:
				val, err := vm.evalExpr(def, stepIndex)
				if err != nil {
					return nil, err
				}
				instance[fieldName] = val
			default:
				instance[fieldName] = fd.Default
			}
			continue
		}
		if IsNullable(fd.Type) {
			instance[fieldName] = nil
			continue
		}
		return nil, RuntimeError("MISSING_FIELD",
			fmt.Sprintf("struct '%s' requires field '%s' (type %s)", structName, fieldName, fd.Type),
			stepIndex)
	}

	return instance, nil
}

func validateFieldType(structName, fieldName string, fd *FieldDef, val any) error {
	if val == nil {
		if IsNullable(fd.Type) {
			return nil
		}
		return RuntimeError("TYPE_MISMATCH",
			fmt.Sprintf("struct '%s' field '%s' (type %s) cannot be nil", structName, fieldName, fd.Type), -1)
	}

	actualType := InferType(val)
	baseType := BaseType(fd.Type)

	if strings.HasPrefix(baseType, "[]") {
		if actualType != "[]any" {
			return RuntimeError("TYPE_MISMATCH",
				fmt.Sprintf("struct '%s' field '%s' expected %s, got %s", structName, fieldName, fd.Type, actualType), -1)
		}
		return nil
	}

	switch baseType {
	case "any", "":
		return nil
	case "string":
		if actualType != "string" {
			return RuntimeError("TYPE_MISMATCH",
				fmt.Sprintf("struct '%s' field '%s' expected string, got %s", structName, fieldName, actualType), -1)
		}
	case "int":
		if actualType != "int" {
			return RuntimeError("TYPE_MISMATCH",
				fmt.Sprintf("struct '%s' field '%s' expected int, got %s", structName, fieldName, actualType), -1)
		}
	case "float":
		if actualType != "float" && actualType != "int" {
			return RuntimeError("TYPE_MISMATCH",
				fmt.Sprintf("struct '%s' field '%s' expected float, got %s", structName, fieldName, actualType), -1)
		}
	case "bool":
		if actualType != "bool" {
			return RuntimeError("TYPE_MISMATCH",
				fmt.Sprintf("struct '%s' field '%s' expected bool, got %s", structName, fieldName, actualType), -1)
		}
	case "map":
		if actualType != "map" {
			return RuntimeError("TYPE_MISMATCH",
				fmt.Sprintf("struct '%s' field '%s' expected map, got %s", structName, fieldName, actualType), -1)
		}
	default:
		if actualType == "map" {
			if m, ok := val.(map[string]any); ok {
				if typeName, ok := m["_type"].(string); ok && typeName != baseType {
					return RuntimeError("TYPE_MISMATCH",
						fmt.Sprintf("struct '%s' field '%s' expected %s, got %s", structName, fieldName, baseType, typeName), -1)
				}
			}
		}
	}
	return nil
}

// evalNewArg evaluates a single with-arg value based on its parsed type:
//   - string → expression
//   - *NewConstruction → recursive struct construction
//   - anything else → literal value
func (vm *VM) evalNewArg(arg any, stepIndex int) (any, error) {
	switch v := arg.(type) {
	case string:
		return vm.evalExpr(v, stepIndex)
	case *NewConstruction:
		return vm.executeNew(v.StructName, v.With, stepIndex)
	default:
		return v, nil
	}
}

// --- Method invocation ---

func (vm *VM) callMethod(objectName, methodName string, withExprs map[string]string, stepIndex int) (any, error) {
	objVal, _, found := vm.scope.Get(objectName)
	if !found {
		return nil, RuntimeError("VAR_NOT_FOUND",
			fmt.Sprintf("variable '%s' not defined", objectName), stepIndex)
	}

	obj, ok := objVal.(map[string]any)
	if !ok {
		return nil, RuntimeError("NOT_STRUCT",
			fmt.Sprintf("cannot call method on %T — expected struct", objVal), stepIndex)
	}

	typeName, _ := obj["_type"].(string)
	if typeName == "" {
		return nil, RuntimeError("NOT_STRUCT",
			fmt.Sprintf("cannot call method — object has no _type metadata"), stepIndex)
	}

	cs, ok := vm.program.Structs[typeName]
	if !ok {
		return nil, RuntimeError("STRUCT_NOT_FOUND",
			fmt.Sprintf("struct type '%s' not defined", typeName), stepIndex)
	}

	if cs.Methods == nil {
		return nil, RuntimeError("METHOD_NOT_FOUND",
			fmt.Sprintf("struct '%s' has no methods", typeName), stepIndex)
	}

	cm, ok := cs.Methods[methodName]
	if !ok {
		methodNames := make([]string, 0, len(cs.Methods))
		for n := range cs.Methods {
			methodNames = append(methodNames, n)
		}
		gjErr := RuntimeError("METHOD_NOT_FOUND",
			fmt.Sprintf("method '%s' not defined on struct '%s'", methodName, typeName), stepIndex)
		if suggestions := SuggestSimilar(methodName, methodNames, 3, 3); len(suggestions) > 0 {
			gjErr.WithSuggestions(suggestions...)
		}
		return nil, gjErr
	}

	vm.depth++
	defer func() { vm.depth-- }()

	if vm.depth > vm.limits.MaxDepth {
		return nil, LimitError("DEPTH_LIMIT",
			fmt.Sprintf("call depth limit (%d) exceeded at method '%s.%s'", vm.limits.MaxDepth, typeName, methodName),
			stepIndex)
	}

	vm.callStack = append(vm.callStack, StackFrame{Function: typeName + "." + methodName, Step: stepIndex})
	defer func() { vm.callStack = vm.callStack[:len(vm.callStack)-1] }()

	methodScope := vm.scope.IsolatedChild("method:" + typeName + "." + methodName)
	methodScope.Declare("self", obj, typeName)

	for _, param := range cm.Params {
		if withExprs != nil {
			if expr, ok := withExprs[param.Name]; ok {
				val, err := vm.evalExpr(expr, stepIndex)
				if err != nil {
					return nil, err
				}
				methodScope.Declare(param.Name, val, param.Type)
				continue
			}
		}
		if param.HasDefault {
			methodScope.Declare(param.Name, param.Default, param.Type)
		} else {
			methodScope.Declare(param.Name, nil, param.Type)
		}
	}

	for fname, ffn := range vm.program.Functions {
		wrapped := vm.wrapFunction(ffn)
		methodScope.Declare(fname, wrapped, "func")
	}

	result, err := vm.withScope(methodScope, func() (any, error) {
		return vm.executeSteps(cm.Steps)
	})
	if err != nil {
		return nil, err
	}

	selfVal, _, _ := methodScope.Get("self")
	if selfMap, ok := selfVal.(map[string]any); ok {
		for k, v := range selfMap {
			obj[k] = v
		}
	}

	if rv, ok := result.(returnValue); ok {
		return rv.value, nil
	}
	// Methods without explicit return implicitly return self for chaining.
	return obj, nil
}

// wrapMethod wraps a CompiledMethod as a Go func for expr-lang expression-level calls.
func (vm *VM) wrapMethod(obj map[string]any, typeName string, cm *CompiledMethod) func(...any) (any, error) {
	return func(args ...any) (any, error) {
		methodScope := vm.scope.IsolatedChild("method:" + typeName + "." + cm.Name)
		methodScope.Declare("self", obj, typeName)

		for i, param := range cm.Params {
			if i < len(args) {
				methodScope.Declare(param.Name, args[i], param.Type)
			} else if param.HasDefault {
				methodScope.Declare(param.Name, param.Default, param.Type)
			} else {
				methodScope.Declare(param.Name, nil, param.Type)
			}
		}

		for fname, ffn := range vm.program.Functions {
			wrapped := vm.wrapFunction(ffn)
			methodScope.Declare(fname, wrapped, "func")
		}

		vm.depth++
		defer func() { vm.depth-- }()

		if vm.depth > vm.limits.MaxDepth {
			return nil, LimitError("DEPTH_LIMIT",
				fmt.Sprintf("call depth limit (%d) exceeded at method '%s.%s'", vm.limits.MaxDepth, typeName, cm.Name), -1)
		}

		vm.callStack = append(vm.callStack, StackFrame{Function: typeName + "." + cm.Name, Step: -1})
		defer func() { vm.callStack = vm.callStack[:len(vm.callStack)-1] }()

		result, err := vm.withScope(methodScope, func() (any, error) {
			return vm.executeSteps(cm.Steps)
		})
		if err != nil {
			return nil, err
		}

		selfVal, _, _ := methodScope.Get("self")
		if selfMap, ok := selfVal.(map[string]any); ok {
			for k, v := range selfMap {
				obj[k] = v
			}
		}

		if rv, ok := result.(returnValue); ok {
			return rv.value, nil
		}
		return obj, nil
	}
}

// --- Expression evaluation ---

func (vm *VM) evalExpr(expression string, stepIndex int) (any, error) {
	env := vm.scope.ToMap()

	// Inject method wrappers directly into struct instance maps.
	// Must use the same map reference so mutations from method calls
	// are visible to subsequent chained calls.
	for _, val := range env {
		obj, ok := val.(map[string]any)
		if !ok {
			continue
		}
		typeName, _ := obj["_type"].(string)
		if typeName == "" {
			continue
		}
		cs, ok := vm.program.Structs[typeName]
		if !ok || cs.Methods == nil {
			continue
		}
		for mName, cm := range cs.Methods {
			obj[mName] = vm.wrapMethod(obj, typeName, cm)
		}
	}

	// Lambda preprocessing: detect and compile lambda expressions.
	processedExpr, lambdaFuncs := PreprocessLambdas(expression, vm.engine, env, vm.limits.MaxDepth)
	if lambdaFuncs != nil {
		if directFn, isDirect := lambdaFuncs["__direct_lambda"]; isDirect {
			return directFn, nil
		}
		for name, fn := range lambdaFuncs {
			env[name] = fn
		}
		expression = processedExpr
	}

	result, err := vm.engine.Eval(expression, env)
	if err != nil {
		if gjErr, ok := err.(*GoJSONError); ok {
			gjErr.Step = stepIndex
			if len(vm.callStack) > 0 {
				gjErr.Function = vm.callStack[len(vm.callStack)-1].Function
			}
			return nil, gjErr
		}
		return nil, RuntimeError("EXPR_ERROR", err.Error(), stepIndex)
	}
	return result, nil
}

func (vm *VM) evalWith(with map[string]string, stepIndex int) (map[string]any, error) {
	result := make(map[string]any, len(with))
	for key, expr := range with {
		val, err := vm.evalExpr(expr, stepIndex)
		if err != nil {
			return nil, err
		}
		result[key] = val
	}
	return result, nil
}

// --- Scope management ---

func (vm *VM) withScope(scope *Scope, fn func() (any, error)) (any, error) {
	prev := vm.scope
	vm.scope = scope
	result, err := fn()
	vm.scope = prev
	return result, err
}

// --- Nested property mutation ---

// setNestedProperty handles dot-path and bracket notation: "a.b.c", "items[0].name"
func (vm *VM) setNestedProperty(path string, value any, stepIndex int) error {
	parts := parseDotPath(path)
	if len(parts) == 0 {
		return RuntimeError("INVALID_PATH", "empty property path", stepIndex)
	}

	// Get the root variable.
	rootName := parts[0]
	rootVal, _, found := vm.scope.Get(rootName)
	if !found {
		return RuntimeError("VAR_NOT_FOUND",
			fmt.Sprintf("variable '%s' not defined", rootName), stepIndex)
	}

	if len(parts) == 1 {
		return vm.scope.Set(rootName, value, InferType(value))
	}

	// Traverse to the parent of the leaf.
	current := rootVal
	for i := 1; i < len(parts)-1; i++ {
		current = traverseProperty(current, parts[i])
		if current == nil {
			return RuntimeError("NIL_ACCESS",
				fmt.Sprintf("cannot access '%s' — intermediate value is nil at '%s'",
					path, strings.Join(parts[:i+1], ".")), stepIndex)
		}
	}

	// Set the leaf.
	leafKey := parts[len(parts)-1]

	if m, ok := current.(map[string]any); ok {
		m[leafKey] = value
		return nil
	}

	if arr, ok := current.([]any); ok {
		idx, err := strconv.Atoi(leafKey)
		if err != nil {
			return RuntimeError("INVALID_INDEX",
				fmt.Sprintf("array index '%s' is not a number", leafKey), stepIndex)
		}
		if idx < 0 || idx >= len(arr) {
			return RuntimeError("INDEX_OUT_OF_BOUNDS",
				fmt.Sprintf("array index %d out of bounds (length %d)", idx, len(arr)), stepIndex)
		}
		arr[idx] = value
		return nil
	}

	return RuntimeError("NOT_SETTABLE",
		fmt.Sprintf("cannot set property '%s' on %T", leafKey, current), stepIndex)
}

// parseDotPath splits "a.b[0].c" into ["a", "b", "0", "c"]
func parseDotPath(path string) []string {
	var parts []string
	current := ""

	for _, ch := range path {
		switch ch {
		case '.':
			if current != "" {
				parts = append(parts, current)
				current = ""
			}
		case '[':
			if current != "" {
				parts = append(parts, current)
				current = ""
			}
		case ']':
			if current != "" {
				parts = append(parts, current)
				current = ""
			}
		default:
			current += string(ch)
		}
	}
	if current != "" {
		parts = append(parts, current)
	}

	return parts
}

func traverseProperty(obj any, key string) any {
	if m, ok := obj.(map[string]any); ok {
		return m[key]
	}
	if arr, ok := obj.([]any); ok {
		idx, err := strconv.Atoi(key)
		if err != nil || idx < 0 || idx >= len(arr) {
			return nil
		}
		return arr[idx]
	}
	return nil
}

// --- Range generation ---

func generateRange(rangeSpec []any) ([]any, error) {
	if len(rangeSpec) < 2 || len(rangeSpec) > 3 {
		return nil, fmt.Errorf("range requires [start, end] or [start, end, step]")
	}

	start, ok := toFloat(rangeSpec[0])
	if !ok {
		return nil, fmt.Errorf("range start must be a number")
	}
	end, ok := toFloat(rangeSpec[1])
	if !ok {
		return nil, fmt.Errorf("range end must be a number")
	}

	step := 1.0
	if len(rangeSpec) == 3 {
		s, ok := toFloat(rangeSpec[2])
		if !ok {
			return nil, fmt.Errorf("range step must be a number")
		}
		if s == 0 {
			return nil, fmt.Errorf("range step cannot be zero")
		}
		step = s
	}

	var items []any
	if step > 0 {
		for i := start; i < end; i += step {
			if float64(int64(i)) == i {
				items = append(items, int(i))
			} else {
				items = append(items, i)
			}
		}
	} else {
		for i := start; i > end; i += step {
			if float64(int64(i)) == i {
				items = append(items, int(i))
			} else {
				items = append(items, i)
			}
		}
	}

	return items, nil
}

func toFloat(v any) (float64, bool) {
	switch n := v.(type) {
	case float64:
		return n, true
	case int:
		return float64(n), true
	case int64:
		return float64(n), true
	}
	return 0, false
}

// --- Error normalization ---

func (vm *VM) normalizeError(err error, stepIndex int) map[string]any {
	result := map[string]any{
		"message": err.Error(),
		"code":    "ERROR",
		"details": nil,
		"step":    stepIndex,
		"stack":   []string{},
	}

	if gjErr, ok := err.(*GoJSONError); ok {
		result["message"] = gjErr.Message
		result["code"] = gjErr.Code
		result["step"] = gjErr.Step
		if gjErr.Context != nil {
			if details, ok := gjErr.Context["details"]; ok {
				result["details"] = details
			}
		}
		stackStrs := make([]string, len(gjErr.Stack))
		for i, f := range gjErr.Stack {
			stackStrs[i] = fmt.Sprintf("%s() step %d", f.Function, f.Step)
		}
		result["stack"] = stackStrs
	}

	return result
}

func (vm *VM) enrichTraceEntry(entry *TraceEntry, node Node, result any) {
	switch n := node.(type) {
	case *LetNode:
		entry.Var = n.Name
		if val, _, ok := vm.scope.Get(n.Name); ok {
			entry.Value = val
		}
	case *SetNode:
		entry.Var = n.Target
		if val, _, ok := vm.scope.Get(n.Target); ok {
			entry.Value = val
		}
	case *IfNode:
		entry.Condition = n.Condition
		if result != nil {
			entry.Result = true
		}
	case *WhileNode:
		entry.Condition = n.Condition
	case *ForNode:
		entry.Var = n.Variable
	case *ReturnNode:
		if rv, ok := result.(returnValue); ok {
			entry.Value = rv.value
		}
	case *SwitchNode:
		entry.Condition = n.Expr
	}
}

// --- Built-in name protection ---

// Reserved names that cannot be used as variable names.
// Only includes names where shadowing would cause real confusion:
// - implicit scope variables (input, session, execution)
// - commonly-called utility functions where shadowing breaks expressions
// Excludes common-word functions (count, filter, map, sort, etc.) that are
// also natural variable names — expr-lang handles these via method syntax.
var builtinNames = map[string]bool{
	"input": true, "session": true, "execution": true,
	"len": true, "abs": true, "ceil": true, "floor": true, "round": true,
	"min": true, "max": true, "sum": true,
	"upper": true, "lower": true, "trim": true, "split": true, "join": true,
	"replace": true, "contains": true, "hasPrefix": true, "hasSuffix": true,
	"strContains": true, "strStartsWith": true, "strEndsWith": true, "strMatches": true,
	"int": true, "float": true, "string": true, "type": true,
	"toJSON": true, "fromJSON": true, "toBase64": true, "fromBase64": true,
	"now": true, "date": true, "duration": true,
	"clamp": true, "randomInt": true, "randomFloat": true,
	"pow": true, "sqrt": true, "mod": true,
	"padLeft": true, "padRight": true, "substring": true, "format": true,
	"append": true, "prepend": true, "slice": true, "chunk": true, "zip": true,
	"isNil": true,
}

func (vm *VM) checkShadowing(name string, stepIndex int) error {
	if builtinNames[name] {
		return CompileError("SHADOWS_BUILTIN",
			fmt.Sprintf("variable '%s' shadows built-in function", name), stepIndex).
			WithFix("use a different variable name to avoid shadowing built-in '" + name + "'")
	}
	return nil
}

// --- Limit enforcement ---

func (vm *VM) checkVariableCount(stepIndex int) error {
	count := vm.scope.VarCount()
	if count > vm.limits.MaxVariables {
		return LimitError("VARIABLE_LIMIT",
			fmt.Sprintf("variable limit (%d) exceeded", vm.limits.MaxVariables), stepIndex)
	}
	return nil
}

func (vm *VM) checkVariableSize(name string, value any, stepIndex int) error {
	size := estimateSize(value)
	if size > vm.limits.MaxVariableSize {
		return LimitError("VARIABLE_SIZE_LIMIT",
			fmt.Sprintf("variable '%s' exceeds size limit (%d bytes, max %d)", name, size, vm.limits.MaxVariableSize), stepIndex)
	}
	return nil
}

func (vm *VM) checkOutputSize(value any, stepIndex int) error {
	size := estimateSize(value)
	if size > vm.limits.MaxOutputSize {
		return LimitError("OUTPUT_SIZE_LIMIT",
			fmt.Sprintf("program output exceeds size limit (%d bytes, max %d)", size, vm.limits.MaxOutputSize), stepIndex)
	}
	return nil
}

func estimateSize(v any) int {
	if v == nil {
		return 0
	}
	switch val := v.(type) {
	case bool:
		return 1
	case int:
		return 8
	case int64:
		return 8
	case float64:
		return 8
	case string:
		return len(val)
	case []any:
		total := 24
		for _, item := range val {
			total += estimateSize(item)
		}
		return total
	case map[string]any:
		total := 24
		for k, item := range val {
			total += len(k) + estimateSize(item)
		}
		return total
	default:
		return 64
	}
}

// --- Pattern Matching ---

func (vm *VM) executeMatch(n *MatchNode) (any, error) {
	subject, err := vm.evalExpr(n.Subject, n.StepIndex)
	if err != nil {
		return nil, err
	}

	for _, c := range n.Cases {
		bindings := make(map[string]any)
		if !vm.doMatch(subject, c.Pattern, bindings) {
			continue
		}

		if c.Guard != "" {
			guardScope := vm.scope.NewChild("match-guard")
			for name, value := range bindings {
				guardScope.Declare(name, value, "any")
			}
			guardResult, err := vm.withScope(guardScope, func() (any, error) {
				return vm.evalExpr(c.Guard, n.StepIndex)
			})
			if err != nil {
				return nil, err
			}
			if !toBool(guardResult) {
				continue
			}
		}

		matchScope := vm.scope.NewChild("match")
		for name, value := range bindings {
			matchScope.Declare(name, value, "any")
		}

		return vm.withScope(matchScope, func() (any, error) {
			return vm.executeSteps(c.Steps)
		})
	}

	return nil, nil
}

func (vm *VM) doMatch(subject any, pattern any, bindings map[string]any) bool {
	switch p := pattern.(type) {
	case string:
		if p == "_" {
			return true
		}
		if len(p) > 1 && p[0] == '$' {
			varName := p[1:]
			if varName != "_" {
				bindings[varName] = subject
			}
			return true
		}
		subStr, ok := subject.(string)
		return ok && subStr == p

	case float64:
		switch s := subject.(type) {
		case float64:
			return s == p
		case int:
			return float64(s) == p
		case int64:
			return float64(s) == p
		}
		return false

	case bool:
		subBool, ok := subject.(bool)
		return ok && subBool == p

	case nil:
		return subject == nil

	case map[string]any:
		subMap, ok := subject.(map[string]any)
		if !ok {
			return false
		}
		for key, patternValue := range p {
			subValue, exists := subMap[key]
			if !exists {
				if patternValue == nil {
					continue
				}
				if ps, ok := patternValue.(string); ok {
					if ps == "_" {
						continue
					}
					if len(ps) > 1 && ps[0] == '$' {
						varName := ps[1:]
						if varName != "_" {
							bindings[varName] = nil
						}
						continue
					}
				}
				return false
			}
			if !vm.doMatch(subValue, patternValue, bindings) {
				return false
			}
		}
		return true

	case []any:
		subArr, ok := subject.([]any)
		if !ok {
			return false
		}
		if len(subArr) != len(p) {
			return false
		}
		for i, patternElem := range p {
			if !vm.doMatch(subArr[i], patternElem, bindings) {
				return false
			}
		}
		return true
	}

	return false
}

func (vm *VM) callNativeFunc(fn func(...any) (any, error), positionalWith []string, literalArgs []any, stepIndex int) (any, error) {
	var callArgs []any
	if literalArgs != nil {
		callArgs = literalArgs
	} else if positionalWith != nil {
		callArgs = make([]any, len(positionalWith))
		for i, expr := range positionalWith {
			val, err := vm.evalExpr(expr, stepIndex)
			if err != nil {
				return nil, err
			}
			callArgs[i] = val
		}
	}
	return fn(callArgs...)
}

// --- Sleep / Retry / Assert ---

func (vm *VM) executeSleep(n *SleepNode) error {
	var ms int
	switch v := n.Duration.(type) {
	case int:
		ms = v
	case string:
		val, err := vm.evalExpr(v, n.StepIndex)
		if err != nil {
			return err
		}
		f, ok := toFloat64Val(val)
		if !ok {
			return RuntimeError("TYPE_ERROR", "sleep duration must evaluate to a number", n.StepIndex)
		}
		ms = int(f)
	}

	if ms <= 0 {
		return nil
	}
	if ms > 300000 {
		return LimitError("SLEEP_LIMIT", "sleep duration cannot exceed 300000ms (5 minutes)", n.StepIndex)
	}

	select {
	case <-time.After(time.Duration(ms) * time.Millisecond):
		return nil
	case <-vm.ctx.Done():
		return LimitError("TIMEOUT", "execution cancelled during sleep", n.StepIndex)
	}
}

func (vm *VM) executeRetry(n *RetryNode) (any, error) {
	var lastErr error
	for attempt := 1; attempt <= n.Max; attempt++ {
		childScope := vm.scope.NewChild(fmt.Sprintf("retry-attempt-%d", attempt))
		result, err := vm.withScope(childScope, func() (any, error) {
			return vm.executeSteps(n.Steps)
		})
		if err == nil {
			return result, nil
		}
		lastErr = err

		if attempt < n.Max {
			delay := calculateBackoff(n.Delay, attempt, n.Backoff)
			select {
			case <-time.After(time.Duration(delay) * time.Millisecond):
			case <-vm.ctx.Done():
				return nil, LimitError("TIMEOUT", "execution cancelled during retry delay", n.StepIndex)
			}
		}
	}
	return nil, RuntimeError("RETRY_EXHAUSTED",
		fmt.Sprintf("retry exhausted after %d attempts: %v", n.Max, lastErr), n.StepIndex)
}

func calculateBackoff(baseDelay, attempt int, strategy string) int {
	switch strategy {
	case "linear":
		return baseDelay * attempt
	case "exponential":
		return baseDelay * (1 << (attempt - 1))
	default:
		return baseDelay
	}
}

func (vm *VM) executeAssert(n *AssertNode) error {
	result, err := vm.evalExpr(n.Condition, n.StepIndex)
	if err != nil {
		return err
	}
	if !toBool(result) {
		msg := fmt.Sprintf("assertion failed: %s", n.Condition)
		if n.Message != "" {
			msgVal, err := vm.evalExpr(n.Message, n.StepIndex)
			if err == nil {
				msg = fmt.Sprintf("%v", msgVal)
			}
		}
		return RuntimeError("ASSERTION_FAILED", msg, n.StepIndex)
	}
	return nil
}

func toFloat64Val(v any) (float64, bool) {
	switch n := v.(type) {
	case int:
		return float64(n), true
	case int64:
		return float64(n), true
	case float64:
		return n, true
	}
	return 0, false
}

// --- Helpers ---

func toBool(v any) bool {
	if v == nil {
		return false
	}
	switch b := v.(type) {
	case bool:
		return b
	case int:
		return b != 0
	case int64:
		return b != 0
	case float64:
		return b != 0
	case string:
		return b != ""
	case []any:
		return len(b) > 0
	case map[string]any:
		return len(b) > 0
	default:
		return true
	}
}
