package stdlib

import (
	"fmt"
	"sort"

	"github.com/expr-lang/expr"
)

// RegisterHigherOrder registers higher-order functions that accept callable lambdas.
func RegisterHigherOrder(r *Registry) {
	r.Register(expr.Function("mapFn", func(params ...any) (any, error) {
		if len(params) < 2 {
			return nil, fmt.Errorf("mapFn: requires 2 arguments (array, function)")
		}
		arr, ok := toAnySlice(params[0])
		if !ok {
			return nil, fmt.Errorf("mapFn: first argument must be an array, got %T", params[0])
		}
		fn, ok := toCallable(params[1])
		if !ok {
			return nil, fmt.Errorf("mapFn: second argument must be a function, got %T", params[1])
		}
		result := make([]any, len(arr))
		for i, item := range arr {
			val, err := fn(item)
			if err != nil {
				return nil, fmt.Errorf("mapFn: error at index %d: %w", i, err)
			}
			result[i] = val
		}
		return result, nil
	}))

	r.Register(expr.Function("filterFn", func(params ...any) (any, error) {
		if len(params) < 2 {
			return nil, fmt.Errorf("filterFn: requires 2 arguments (array, function)")
		}
		arr, ok := toAnySlice(params[0])
		if !ok {
			return nil, fmt.Errorf("filterFn: first argument must be an array, got %T", params[0])
		}
		fn, ok := toCallable(params[1])
		if !ok {
			return nil, fmt.Errorf("filterFn: second argument must be a function, got %T", params[1])
		}
		var result []any
		for _, item := range arr {
			val, err := fn(item)
			if err != nil {
				return nil, fmt.Errorf("filterFn: %w", err)
			}
			if isTruthy(val) {
				result = append(result, item)
			}
		}
		if result == nil {
			result = []any{}
		}
		return result, nil
	}))

	r.Register(expr.Function("reduceFn", func(params ...any) (any, error) {
		if len(params) < 2 {
			return nil, fmt.Errorf("reduceFn: requires at least 2 arguments (array, function[, initial])")
		}
		arr, ok := toAnySlice(params[0])
		if !ok {
			return nil, fmt.Errorf("reduceFn: first argument must be an array, got %T", params[0])
		}
		fn, ok := toCallable2(params[1])
		if !ok {
			return nil, fmt.Errorf("reduceFn: second argument must be a function, got %T", params[1])
		}

		if len(arr) == 0 {
			if len(params) > 2 {
				return params[2], nil
			}
			return nil, nil
		}

		var acc any
		startIdx := 0
		if len(params) > 2 {
			acc = params[2]
		} else {
			acc = arr[0]
			startIdx = 1
		}

		for i := startIdx; i < len(arr); i++ {
			val, err := fn(acc, arr[i])
			if err != nil {
				return nil, fmt.Errorf("reduceFn: error at index %d: %w", i, err)
			}
			acc = val
		}
		return acc, nil
	}))

	r.Register(expr.Function("applyEach", func(params ...any) (any, error) {
		if len(params) < 2 {
			return nil, fmt.Errorf("applyEach: requires 2 arguments (array, function)")
		}
		arr, ok := toAnySlice(params[0])
		if !ok {
			return nil, fmt.Errorf("applyEach: first argument must be an array, got %T", params[0])
		}
		fn, ok := toCallable(params[1])
		if !ok {
			return nil, fmt.Errorf("applyEach: second argument must be a function, got %T", params[1])
		}
		for i, item := range arr {
			if _, err := fn(item); err != nil {
				return nil, fmt.Errorf("applyEach: error at index %d: %w", i, err)
			}
		}
		return nil, nil
	}))

	r.Register(expr.Function("sortFn", func(params ...any) (any, error) {
		if len(params) < 2 {
			return nil, fmt.Errorf("sortFn: requires 2 arguments (array, comparator)")
		}
		arr, ok := toAnySlice(params[0])
		if !ok {
			return nil, fmt.Errorf("sortFn: first argument must be an array, got %T", params[0])
		}
		fn, ok := toCallable2(params[1])
		if !ok {
			return nil, fmt.Errorf("sortFn: second argument must be a function, got %T", params[1])
		}
		result := make([]any, len(arr))
		copy(result, arr)

		var sortErr error
		sort.SliceStable(result, func(i, j int) bool {
			if sortErr != nil {
				return false
			}
			val, err := fn(result[i], result[j])
			if err != nil {
				sortErr = err
				return false
			}
			return isTruthy(val)
		})
		if sortErr != nil {
			return nil, fmt.Errorf("sortFn: %w", sortErr)
		}
		return result, nil
	}))

	r.Register(expr.Function("rejectFn", func(params ...any) (any, error) {
		if len(params) < 2 {
			return nil, fmt.Errorf("rejectFn: requires 2 arguments (array, function)")
		}
		arr, ok := toAnySlice(params[0])
		if !ok {
			return nil, fmt.Errorf("rejectFn: first argument must be an array, got %T", params[0])
		}
		fn, ok := toCallable(params[1])
		if !ok {
			return nil, fmt.Errorf("rejectFn: second argument must be a function, got %T", params[1])
		}
		var result []any
		for _, item := range arr {
			val, err := fn(item)
			if err != nil {
				return nil, fmt.Errorf("rejectFn: %w", err)
			}
			if !isTruthy(val) {
				result = append(result, item)
			}
		}
		if result == nil {
			result = []any{}
		}
		return result, nil
	}))

	r.Register(expr.Function("takeWhileFn", func(params ...any) (any, error) {
		if len(params) < 2 {
			return nil, fmt.Errorf("takeWhileFn: requires 2 arguments (array, function)")
		}
		arr, ok := toAnySlice(params[0])
		if !ok {
			return nil, fmt.Errorf("takeWhileFn: first argument must be an array, got %T", params[0])
		}
		fn, ok := toCallable(params[1])
		if !ok {
			return nil, fmt.Errorf("takeWhileFn: second argument must be a function, got %T", params[1])
		}
		var result []any
		for _, item := range arr {
			val, err := fn(item)
			if err != nil {
				return nil, fmt.Errorf("takeWhileFn: %w", err)
			}
			if !isTruthy(val) {
				break
			}
			result = append(result, item)
		}
		if result == nil {
			result = []any{}
		}
		return result, nil
	}))

	r.Register(expr.Function("dropWhileFn", func(params ...any) (any, error) {
		if len(params) < 2 {
			return nil, fmt.Errorf("dropWhileFn: requires 2 arguments (array, function)")
		}
		arr, ok := toAnySlice(params[0])
		if !ok {
			return nil, fmt.Errorf("dropWhileFn: first argument must be an array, got %T", params[0])
		}
		fn, ok := toCallable(params[1])
		if !ok {
			return nil, fmt.Errorf("dropWhileFn: second argument must be a function, got %T", params[1])
		}
		dropping := true
		var result []any
		for _, item := range arr {
			if dropping {
				val, err := fn(item)
				if err != nil {
					return nil, fmt.Errorf("dropWhileFn: %w", err)
				}
				if isTruthy(val) {
					continue
				}
				dropping = false
			}
			result = append(result, item)
		}
		if result == nil {
			result = []any{}
		}
		return result, nil
	}))

	r.Register(expr.Function("findFn", func(params ...any) (any, error) {
		if len(params) < 2 {
			return nil, fmt.Errorf("findFn: requires 2 arguments (array, function)")
		}
		arr, ok := toAnySlice(params[0])
		if !ok {
			return nil, fmt.Errorf("findFn: first argument must be an array, got %T", params[0])
		}
		fn, ok := toCallable(params[1])
		if !ok {
			return nil, fmt.Errorf("findFn: second argument must be a function, got %T", params[1])
		}
		for _, item := range arr {
			val, err := fn(item)
			if err != nil {
				return nil, fmt.Errorf("findFn: %w", err)
			}
			if isTruthy(val) {
				return item, nil
			}
		}
		return nil, nil
	}))

	r.Register(expr.Function("everyFn", func(params ...any) (any, error) {
		if len(params) < 2 {
			return nil, fmt.Errorf("everyFn: requires 2 arguments (array, function)")
		}
		arr, ok := toAnySlice(params[0])
		if !ok {
			return nil, fmt.Errorf("everyFn: first argument must be an array, got %T", params[0])
		}
		fn, ok := toCallable(params[1])
		if !ok {
			return nil, fmt.Errorf("everyFn: second argument must be a function, got %T", params[1])
		}
		for _, item := range arr {
			val, err := fn(item)
			if err != nil {
				return nil, fmt.Errorf("everyFn: %w", err)
			}
			if !isTruthy(val) {
				return false, nil
			}
		}
		return true, nil
	}))

	r.Register(expr.Function("someFn", func(params ...any) (any, error) {
		if len(params) < 2 {
			return nil, fmt.Errorf("someFn: requires 2 arguments (array, function)")
		}
		arr, ok := toAnySlice(params[0])
		if !ok {
			return nil, fmt.Errorf("someFn: first argument must be an array, got %T", params[0])
		}
		fn, ok := toCallable(params[1])
		if !ok {
			return nil, fmt.Errorf("someFn: second argument must be a function, got %T", params[1])
		}
		for _, item := range arr {
			val, err := fn(item)
			if err != nil {
				return nil, fmt.Errorf("someFn: %w", err)
			}
			if isTruthy(val) {
				return true, nil
			}
		}
		return false, nil
	}))

	r.Register(expr.Function("partitionFn", func(params ...any) (any, error) {
		if len(params) < 2 {
			return nil, fmt.Errorf("partitionFn: requires 2 arguments (array, function)")
		}
		arr, ok := toAnySlice(params[0])
		if !ok {
			return nil, fmt.Errorf("partitionFn: first argument must be an array, got %T", params[0])
		}
		fn, ok := toCallable(params[1])
		if !ok {
			return nil, fmt.Errorf("partitionFn: second argument must be a function, got %T", params[1])
		}
		var matches, nonMatches []any
		for _, item := range arr {
			val, err := fn(item)
			if err != nil {
				return nil, fmt.Errorf("partitionFn: %w", err)
			}
			if isTruthy(val) {
				matches = append(matches, item)
			} else {
				nonMatches = append(nonMatches, item)
			}
		}
		if matches == nil {
			matches = []any{}
		}
		if nonMatches == nil {
			nonMatches = []any{}
		}
		return []any{matches, nonMatches}, nil
	}))

	r.Register(expr.Function("identity", func(params ...any) (any, error) {
		if len(params) < 1 {
			return nil, nil
		}
		return params[0], nil
	}))
}

func toCallable(v any) (func(any) (any, error), bool) {
	switch fn := v.(type) {
	case func(...any) (any, error):
		return func(a any) (any, error) { return fn(a) }, true
	case func(any) (any, error):
		return fn, true
	}
	return nil, false
}

func toCallable2(v any) (func(any, any) (any, error), bool) {
	switch fn := v.(type) {
	case func(...any) (any, error):
		return func(a, b any) (any, error) { return fn(a, b) }, true
	case func(any, any) (any, error):
		return fn, true
	}
	return nil, false
}
