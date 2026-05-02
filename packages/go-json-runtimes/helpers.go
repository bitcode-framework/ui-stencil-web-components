package runtimes

import "fmt"

func ToInt(v any) int {
	switch val := v.(type) {
	case int:
		return val
	case int64:
		return int(val)
	case float64:
		return int(val)
	case float32:
		return int(val)
	default:
		return 0
	}
}

func ToStringSlice(v any) []string {
	switch val := v.(type) {
	case []string:
		return val
	case []any:
		result := make([]string, len(val))
		for i, item := range val {
			result[i] = fmt.Sprintf("%v", item)
		}
		return result
	default:
		return nil
	}
}

func ToAnySliceSlice(v any) [][]any {
	switch val := v.(type) {
	case [][]any:
		return val
	case []any:
		result := make([][]any, len(val))
		for i, item := range val {
			if inner, ok := item.([]any); ok {
				result[i] = inner
			}
		}
		return result
	default:
		return nil
	}
}
