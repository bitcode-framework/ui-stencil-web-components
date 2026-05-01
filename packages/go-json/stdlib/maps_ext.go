package stdlib

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strings"

	"github.com/expr-lang/expr"
)

// RegisterMapsExt registers extended map/object functions (Phase 4.5f).
func RegisterMapsExt(r *Registry) {
	r.Register(expr.Function("deepMerge", func(params ...any) (any, error) {
		a, ok1 := params[0].(map[string]any)
		b, ok2 := params[1].(map[string]any)
		if !ok1 || !ok2 {
			return nil, fmt.Errorf("deepMerge: both arguments must be maps")
		}
		return deepMergeMap(a, b), nil
	}))

	r.Register(expr.Function("deepClone", func(params ...any) (any, error) {
		data, err := json.Marshal(params[0])
		if err != nil {
			return nil, fmt.Errorf("deepClone: cannot serialize: %s", err)
		}
		var result any
		if err := json.Unmarshal(data, &result); err != nil {
			return nil, fmt.Errorf("deepClone: cannot deserialize: %s", err)
		}
		return result, nil
	}))

	r.Register(expr.Function("deepEqual", func(params ...any) (any, error) {
		return reflect.DeepEqual(params[0], params[1]), nil
	}))

	r.Register(expr.Function("setIn", func(params ...any) (any, error) {
		obj, ok := params[0].(map[string]any)
		if !ok {
			return nil, fmt.Errorf("setIn: first argument must be a map")
		}
		path, ok := params[1].(string)
		if !ok {
			return nil, fmt.Errorf("setIn: second argument must be a string path")
		}
		value := params[2]
		result := deepCloneMap(obj)
		setNestedValue(result, strings.Split(path, "."), value)
		return result, nil
	}))

	r.Register(expr.Function("deleteIn", func(params ...any) (any, error) {
		obj, ok := params[0].(map[string]any)
		if !ok {
			return nil, fmt.Errorf("deleteIn: first argument must be a map")
		}
		path, ok := params[1].(string)
		if !ok {
			return nil, fmt.Errorf("deleteIn: second argument must be a string path")
		}
		result := deepCloneMap(obj)
		deleteNestedKey(result, strings.Split(path, "."))
		return result, nil
	}))

	r.Register(expr.Function("defaults", func(params ...any) (any, error) {
		obj, ok1 := params[0].(map[string]any)
		defs, ok2 := params[1].(map[string]any)
		if !ok1 || !ok2 {
			return nil, fmt.Errorf("defaults: both arguments must be maps")
		}
		result := make(map[string]any, len(obj)+len(defs))
		for k, v := range defs {
			result[k] = v
		}
		for k, v := range obj {
			result[k] = v
		}
		return result, nil
	}))
}

func deepMergeMap(a, b map[string]any) map[string]any {
	result := make(map[string]any, len(a)+len(b))
	for k, v := range a {
		result[k] = v
	}
	for k, v := range b {
		if bMap, ok := v.(map[string]any); ok {
			if aMap, ok := result[k].(map[string]any); ok {
				result[k] = deepMergeMap(aMap, bMap)
				continue
			}
		}
		result[k] = v
	}
	return result
}

func deepCloneMap(m map[string]any) map[string]any {
	data, _ := json.Marshal(m)
	var result map[string]any
	json.Unmarshal(data, &result)
	if result == nil {
		result = make(map[string]any)
	}
	return result
}

func setNestedValue(m map[string]any, parts []string, value any) {
	for i := 0; i < len(parts)-1; i++ {
		next, ok := m[parts[i]].(map[string]any)
		if !ok {
			next = make(map[string]any)
			m[parts[i]] = next
		}
		m = next
	}
	m[parts[len(parts)-1]] = value
}

func deleteNestedKey(m map[string]any, parts []string) {
	for i := 0; i < len(parts)-1; i++ {
		next, ok := m[parts[i]].(map[string]any)
		if !ok {
			return
		}
		m = next
	}
	delete(m, parts[len(parts)-1])
}
