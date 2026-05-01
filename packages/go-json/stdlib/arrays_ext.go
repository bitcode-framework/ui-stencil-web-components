package stdlib

import (
	"fmt"

	"github.com/expr-lang/expr"
)

// RegisterArraysExt registers extended array functions (Phase 4.5f).
func RegisterArraysExt(r *Registry) {
	r.Register(expr.Function("compact", func(params ...any) (any, error) {
		arr, ok := toAnySlice(params[0])
		if !ok {
			return nil, fmt.Errorf("compact: argument must be an array")
		}
		var result []any
		for _, item := range arr {
			if item != nil {
				result = append(result, item)
			}
		}
		if result == nil {
			result = []any{}
		}
		return result, nil
	}))

	r.Register(expr.Function("includes", func(params ...any) (any, error) {
		arr, ok := toAnySlice(params[0])
		if !ok {
			return nil, fmt.Errorf("includes: first argument must be an array")
		}
		target := params[1]
		for _, item := range arr {
			if equalValues(item, target) {
				return true, nil
			}
		}
		return false, nil
	}))

	r.Register(expr.Function("arrayIndexOf", func(params ...any) (any, error) {
		arr, ok := toAnySlice(params[0])
		if !ok {
			return nil, fmt.Errorf("arrayIndexOf: first argument must be an array")
		}
		target := params[1]
		for i, item := range arr {
			if equalValues(item, target) {
				return i, nil
			}
		}
		return -1, nil
	}))

	r.Register(expr.Function("keyBy", func(params ...any) (any, error) {
		arr, ok := toAnySlice(params[0])
		if !ok {
			return nil, fmt.Errorf("keyBy: first argument must be an array")
		}
		key, ok := params[1].(string)
		if !ok {
			return nil, fmt.Errorf("keyBy: second argument must be a string (field name)")
		}
		result := make(map[string]any)
		for _, item := range arr {
			m, ok := item.(map[string]any)
			if !ok {
				continue
			}
			if v, exists := m[key]; exists {
				result[fmt.Sprintf("%v", v)] = item
			}
		}
		return result, nil
	}))

	r.Register(expr.Function("difference", func(params ...any) (any, error) {
		a, ok1 := toAnySlice(params[0])
		b, ok2 := toAnySlice(params[1])
		if !ok1 || !ok2 {
			return nil, fmt.Errorf("difference: both arguments must be arrays")
		}
		var result []any
		for _, item := range a {
			found := false
			for _, bItem := range b {
				if equalValues(item, bItem) {
					found = true
					break
				}
			}
			if !found {
				result = append(result, item)
			}
		}
		if result == nil {
			result = []any{}
		}
		return result, nil
	}))

	r.Register(expr.Function("intersection", func(params ...any) (any, error) {
		a, ok1 := toAnySlice(params[0])
		b, ok2 := toAnySlice(params[1])
		if !ok1 || !ok2 {
			return nil, fmt.Errorf("intersection: both arguments must be arrays")
		}
		var result []any
		for _, item := range a {
			for _, bItem := range b {
				if equalValues(item, bItem) {
					result = append(result, item)
					break
				}
			}
		}
		if result == nil {
			result = []any{}
		}
		return result, nil
	}))

	r.Register(expr.Function("union", func(params ...any) (any, error) {
		a, ok1 := toAnySlice(params[0])
		b, ok2 := toAnySlice(params[1])
		if !ok1 || !ok2 {
			return nil, fmt.Errorf("union: both arguments must be arrays")
		}
		var result []any
		addUnique := func(items []any) {
			for _, item := range items {
				found := false
				for _, existing := range result {
					if equalValues(item, existing) {
						found = true
						break
					}
				}
				if !found {
					result = append(result, item)
				}
			}
		}
		addUnique(a)
		addUnique(b)
		if result == nil {
			result = []any{}
		}
		return result, nil
	}))

	r.Register(expr.Function("fill", func(params ...any) (any, error) {
		nF, ok := toFloat64(params[0])
		if !ok {
			return nil, fmt.Errorf("fill: first argument must be a number")
		}
		n := int(nF)
		if n < 0 || n > 100000 {
			return nil, fmt.Errorf("fill: size must be between 0 and 100000")
		}
		value := params[1]
		result := make([]any, n)
		for i := range result {
			result[i] = value
		}
		return result, nil
	}))

	r.Register(expr.Function("drop", func(params ...any) (any, error) {
		arr, ok := toAnySlice(params[0])
		if !ok {
			return nil, fmt.Errorf("drop: first argument must be an array")
		}
		nF, ok := toFloat64(params[1])
		if !ok {
			return nil, fmt.Errorf("drop: second argument must be a number")
		}
		n := int(nF)
		if n >= len(arr) {
			return []any{}, nil
		}
		if n < 0 {
			n = 0
		}
		result := make([]any, len(arr)-n)
		copy(result, arr[n:])
		return result, nil
	}))

	r.Register(expr.Function("takeRight", func(params ...any) (any, error) {
		arr, ok := toAnySlice(params[0])
		if !ok {
			return nil, fmt.Errorf("takeRight: first argument must be an array")
		}
		nF, ok := toFloat64(params[1])
		if !ok {
			return nil, fmt.Errorf("takeRight: second argument must be a number")
		}
		n := int(nF)
		if n >= len(arr) {
			result := make([]any, len(arr))
			copy(result, arr)
			return result, nil
		}
		if n <= 0 {
			return []any{}, nil
		}
		result := make([]any, n)
		copy(result, arr[len(arr)-n:])
		return result, nil
	}))

	r.Register(expr.Function("flatMap", func(params ...any) (any, error) {
		arr, ok := toAnySlice(params[0])
		if !ok {
			return nil, fmt.Errorf("flatMap: first argument must be an array")
		}
		field, ok := params[1].(string)
		if !ok {
			return nil, fmt.Errorf("flatMap: second argument must be a field name (string)")
		}
		var result []any
		for _, item := range arr {
			m, ok := item.(map[string]any)
			if !ok {
				continue
			}
			val, exists := m[field]
			if !exists {
				continue
			}
			if nested, ok := toAnySlice(val); ok {
				result = append(result, nested...)
			} else {
				result = append(result, val)
			}
		}
		if result == nil {
			result = []any{}
		}
		return result, nil
	}))

	r.Register(expr.Function("partition", func(params ...any) (any, error) {
		arr, ok := toAnySlice(params[0])
		if !ok {
			return nil, fmt.Errorf("partition: first argument must be an array")
		}
		field, ok := params[1].(string)
		if !ok {
			return nil, fmt.Errorf("partition: second argument must be a field name (string)")
		}
		var matches []any
		var nonMatches []any
		for _, item := range arr {
			m, ok := item.(map[string]any)
			if !ok {
				nonMatches = append(nonMatches, item)
				continue
			}
			val := m[field]
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
}

func toAnySlice(v any) ([]any, bool) {
	if arr, ok := v.([]any); ok {
		return arr, true
	}
	return nil, false
}

func isTruthy(v any) bool {
	if v == nil {
		return false
	}
	switch val := v.(type) {
	case bool:
		return val
	case int:
		return val != 0
	case float64:
		return val != 0
	case string:
		return val != ""
	default:
		return true
	}
}

func equalValues(a, b any) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	aF, aOk := toFloat64(a)
	bF, bOk := toFloat64(b)
	if aOk && bOk {
		return aF == bF
	}
	return fmt.Sprintf("%v", a) == fmt.Sprintf("%v", b)
}
