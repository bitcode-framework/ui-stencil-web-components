package runtime

import (
	"fmt"
	"reflect"
	"strconv"

	"github.com/bitcode-framework/go-json/lang"
	"github.com/bitcode-framework/go-json/stdlib"
	"github.com/expr-lang/expr/ast"
	"github.com/expr-lang/expr/parser"
)

var sharedEngine = lang.NewExprLangEngine()
var sharedStdlibEnv map[string]any

func init() {
	reg := stdlib.DefaultRegistry()
	sharedEngine.AddOptions(reg.All()...)
	sharedStdlibEnv = reg.EnvVars()
}

// ExprTree wraps expr-lang's parsed AST to provide a stable public API.
// This prevents leaking the expr-lang dependency to engine consumers.
type ExprTree struct {
	Root ast.Node
}

// EvalExpr evaluates a single expression string against an environment.
// This is the lightweight entry point for engine consumers that need
// expression evaluation without a full go-json program/VM.
//
// Uses the shared ExprLangEngine with compilation caching.
func EvalExpr(expression string, env map[string]any) (any, error) {
	if expression == "" {
		return nil, nil
	}
	return sharedEngine.Eval(expression, mergeStdlibEnv(env))
}

func mergeStdlibEnv(env map[string]any) map[string]any {
	if len(sharedStdlibEnv) == 0 {
		return env
	}
	merged := make(map[string]any, len(sharedStdlibEnv)+len(env))
	for k, v := range sharedStdlibEnv {
		merged[k] = v
	}
	for k, v := range env {
		merged[k] = v
	}
	return merged
}

// EvalExprFloat is a convenience wrapper that coerces the result to float64.
// Non-numeric results return an error.
func EvalExprFloat(expression string, env map[string]any) (float64, error) {
	val, err := EvalExpr(expression, env)
	if err != nil {
		return 0, err
	}
	return coerceToFloat64(val)
}

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
func EvalExprBool(expression string, env map[string]any) (bool, error) {
	if expression == "" {
		return false, nil
	}
	val, err := EvalExpr(expression, env)
	if err != nil {
		return false, err
	}
	return coerceToBool(val), nil
}

// ParseExpr parses an expression and returns the AST root node.
// Returns expr-lang's ast.Node wrapped in ExprTree for stability.
// Used by record rule AST-to-WHERE conversion.
func ParseExpr(expression string) (*ExprTree, error) {
	if expression == "" {
		return nil, fmt.Errorf("empty expression")
	}
	tree, err := parser.Parse(expression)
	if err != nil {
		return nil, fmt.Errorf("parse error: %w", err)
	}
	return &ExprTree{Root: tree.Node}, nil
}

// ValidateExpr validates an expression without executing it.
// Returns nil if valid, or a compile error.
func ValidateExpr(expression string, env map[string]any) error {
	if expression == "" {
		return fmt.Errorf("empty expression")
	}
	return sharedEngine.Validate(expression, env)
}

// coerceToFloat64 converts a value to float64.
func coerceToFloat64(val any) (float64, error) {
	if val == nil {
		return 0, nil
	}
	switch v := val.(type) {
	case float64:
		return v, nil
	case float32:
		return float64(v), nil
	case int:
		return float64(v), nil
	case int8:
		return float64(v), nil
	case int16:
		return float64(v), nil
	case int32:
		return float64(v), nil
	case int64:
		return float64(v), nil
	case uint:
		return float64(v), nil
	case uint8:
		return float64(v), nil
	case uint16:
		return float64(v), nil
	case uint32:
		return float64(v), nil
	case uint64:
		return float64(v), nil
	case string:
		f, err := strconv.ParseFloat(v, 64)
		if err != nil {
			return 0, fmt.Errorf("cannot convert string %q to float64: %w", v, err)
		}
		return f, nil
	case bool:
		if v {
			return 1, nil
		}
		return 0, nil
	default:
		return 0, fmt.Errorf("cannot convert %T to float64", val)
	}
}

// coerceToBool applies truthiness coercion to a value.
func coerceToBool(val any) bool {
	if val == nil {
		return false
	}
	switch v := val.(type) {
	case bool:
		return v
	case int:
		return v != 0
	case int8:
		return v != 0
	case int16:
		return v != 0
	case int32:
		return v != 0
	case int64:
		return v != 0
	case uint:
		return v != 0
	case uint8:
		return v != 0
	case uint16:
		return v != 0
	case uint32:
		return v != 0
	case uint64:
		return v != 0
	case float32:
		return v != 0
	case float64:
		return v != 0
	case string:
		return v != ""
	default:
		rv := reflect.ValueOf(val)
		switch rv.Kind() {
		case reflect.Slice, reflect.Map:
			return rv.Len() > 0
		case reflect.Array:
			return rv.Len() > 0
		default:
			return true
		}
	}
}
