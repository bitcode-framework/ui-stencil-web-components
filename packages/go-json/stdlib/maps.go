package stdlib

import (
	"fmt"
	"strconv"

	"github.com/expr-lang/expr"
)

type pathToken struct {
	key   string
	index int // -1 = map key, >= 0 = array index
}

func parsePath(path string, sep string) []pathToken {
	var tokens []pathToken
	buf := ""
	for i := 0; i < len(path); i++ {
		c := path[i]
		if c == '[' {
			if buf != "" {
				tokens = append(tokens, pathToken{key: buf, index: -1})
				buf = ""
			}
			j := i + 1
			for ; j < len(path) && path[j] != ']'; j++ {
			}
			if idx, err := strconv.Atoi(path[i+1 : j]); err == nil {
				tokens = append(tokens, pathToken{index: idx})
			}
			i = j
		} else if i+len(sep) <= len(path) && path[i:i+len(sep)] == sep {
			if buf != "" {
				tokens = append(tokens, pathToken{key: buf, index: -1})
				buf = ""
			}
			i += len(sep) - 1
		} else {
			buf += string(c)
		}
	}
	if buf != "" {
		tokens = append(tokens, pathToken{key: buf, index: -1})
	}
	return tokens
}

func getIn(data any, tokens []pathToken) any {
	current := data
	for _, t := range tokens {
		if current == nil {
			return nil
		}
		if t.index >= 0 {
			switch arr := current.(type) {
			case []any:
				if t.index >= len(arr) {
					return nil
				}
				current = arr[t.index]
			case []map[string]any:
				if t.index >= len(arr) {
					return nil
				}
				current = arr[t.index]
			default:
				return nil
			}
		} else {
			m, ok := current.(map[string]any)
			if !ok {
				return nil
			}
			current = m[t.key]
		}
	}
	return current
}

func RegisterMaps(r *Registry) {
	r.Register(expr.Function("has", func(params ...any) (any, error) {
		m, ok := params[0].(map[string]any)
		if !ok {
			return nil, fmt.Errorf("has: first argument must be a map")
		}
		key, ok := params[1].(string)
		if !ok {
			return nil, fmt.Errorf("has: second argument must be a string")
		}
		_, exists := m[key]
		return exists, nil
	}))

	r.Register(expr.Function("getIn", func(params ...any) (any, error) {
		if len(params) < 2 {
			return nil, fmt.Errorf("getIn: requires (object, path) or (object, path, separator)")
		}
		path, ok := params[1].(string)
		if !ok {
			return nil, fmt.Errorf("getIn: second argument must be a string")
		}
		sep := "."
		if len(params) > 2 {
			if s, ok := params[2].(string); ok && s != "" {
				sep = s
			}
		}
		tokens := parsePath(path, sep)
		return getIn(params[0], tokens), nil
	}))

	r.Register(expr.Function("merge", func(params ...any) (any, error) {
		a, ok1 := params[0].(map[string]any)
		b, ok2 := params[1].(map[string]any)
		if !ok1 || !ok2 {
			return nil, fmt.Errorf("merge: both arguments must be maps")
		}
		result := make(map[string]any, len(a)+len(b))
		for k, v := range a {
			result[k] = v
		}
		for k, v := range b {
			result[k] = v
		}
		return result, nil
	}))

	r.Register(expr.Function("pick", func(params ...any) (any, error) {
		m, ok := params[0].(map[string]any)
		if !ok {
			return nil, fmt.Errorf("pick: first argument must be a map")
		}
		keys, ok := params[1].([]any)
		if !ok {
			return nil, fmt.Errorf("pick: second argument must be an array of strings")
		}
		result := make(map[string]any)
		for _, k := range keys {
			key, ok := k.(string)
			if !ok {
				continue
			}
			if v, exists := m[key]; exists {
				result[key] = v
			}
		}
		return result, nil
	}))

	r.Register(expr.Function("omit", func(params ...any) (any, error) {
		m, ok := params[0].(map[string]any)
		if !ok {
			return nil, fmt.Errorf("omit: first argument must be a map")
		}
		keys, ok := params[1].([]any)
		if !ok {
			return nil, fmt.Errorf("omit: second argument must be an array of strings")
		}
		exclude := make(map[string]bool, len(keys))
		for _, k := range keys {
			if key, ok := k.(string); ok {
				exclude[key] = true
			}
		}
		result := make(map[string]any)
		for k, v := range m {
			if !exclude[k] {
				result[k] = v
			}
		}
		return result, nil
	}))
}
