package stdlib

import (
	"fmt"
	"math"
	"math/rand"

	"github.com/expr-lang/expr"
)

// RegisterMathExt registers extended math functions and constants (Phase 4.5f).
func RegisterMathExt(r *Registry) {
	mathUnary := func(name string, fn func(float64) float64) {
		r.Register(expr.Function(name, func(params ...any) (any, error) {
			x, ok := toFloat64(params[0])
			if !ok {
				return nil, fmt.Errorf("%s: argument must be a number", name)
			}
			return fn(x), nil
		}))
	}

	mathUnary("sin", math.Sin)
	mathUnary("cos", math.Cos)
	mathUnary("tan", math.Tan)
	mathUnary("asin", math.Asin)
	mathUnary("acos", math.Acos)
	mathUnary("atan", math.Atan)
	mathUnary("log", math.Log)
	mathUnary("log2", math.Log2)
	mathUnary("log10", math.Log10)
	mathUnary("exp", math.Exp)
	mathUnary("trunc", math.Trunc)

	r.Register(expr.Function("atan2", func(params ...any) (any, error) {
		y, ok1 := toFloat64(params[0])
		x, ok2 := toFloat64(params[1])
		if !ok1 || !ok2 {
			return nil, fmt.Errorf("atan2: arguments must be numbers")
		}
		return math.Atan2(y, x), nil
	}))

	r.Register(expr.Function("random", func(params ...any) (any, error) {
		return rand.Float64(), nil
	}))

	r.Register(expr.Function("isNaN", func(params ...any) (any, error) {
		x, ok := toFloat64(params[0])
		if !ok {
			return false, nil
		}
		return math.IsNaN(x), nil
	}))

	r.Register(expr.Function("isInfinite", func(params ...any) (any, error) {
		x, ok := toFloat64(params[0])
		if !ok {
			return false, nil
		}
		return math.IsInf(x, 0), nil
	}))

	r.Register(expr.Function("isFinite", func(params ...any) (any, error) {
		x, ok := toFloat64(params[0])
		if !ok {
			return false, nil
		}
		return !math.IsNaN(x) && !math.IsInf(x, 0), nil
	}))

	r.RegisterEnv("PI", math.Pi)
	r.RegisterEnv("E", math.E)
	r.RegisterEnv("Infinity", math.Inf(1))
	r.RegisterEnv("NaN", math.NaN())
}
