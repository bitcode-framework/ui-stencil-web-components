package expression

import (
	"fmt"
	"math"
	"strconv"
	"strings"
)

func resolveAggregateOrBuiltin(funcName string, arg any, collections map[string][]map[string]any) float64 {
	if s, ok := arg.(string); ok && strings.Contains(s, ".") {
		parts := strings.SplitN(s, ".", 2)
		collectionName := parts[0]
		fieldName := parts[1]

		records, exists := collections[collectionName]
		if !exists {
			return 0
		}

		var vals []float64
		for _, rec := range records {
			if v, ok := rec[fieldName]; ok {
				vals = append(vals, toFloat64(v))
			}
		}

		return aggregateValues(funcName, vals)
	}

	if arr, ok := arg.([]any); ok {
		var vals []float64
		for _, v := range arr {
			vals = append(vals, toFloat64(v))
		}
		return aggregateValues(funcName, vals)
	}

	return toFloat64(arg)
}

func aggregateValues(funcName string, vals []float64) float64 {
	switch funcName {
	case "sum":
		s := 0.0
		for _, v := range vals {
			s += v
		}
		return s
	case "count":
		return float64(len(vals))
	case "avg":
		if len(vals) == 0 {
			return 0
		}
		s := 0.0
		for _, v := range vals {
			s += v
		}
		return s / float64(len(vals))
	case "min":
		if len(vals) == 0 {
			return 0
		}
		m := vals[0]
		for _, v := range vals[1:] {
			m = math.Min(m, v)
		}
		return m
	case "max":
		if len(vals) == 0 {
			return 0
		}
		m := vals[0]
		for _, v := range vals[1:] {
			m = math.Max(m, v)
		}
		return m
	}
	return 0
}

func toFloat64(v any) float64 {
	switch val := v.(type) {
	case float64:
		return val
	case float32:
		return float64(val)
	case int:
		return float64(val)
	case int64:
		return float64(val)
	case int32:
		return float64(val)
	case string:
		f, err := strconv.ParseFloat(val, 64)
		if err != nil {
			return 0
		}
		return f
	case bool:
		if val {
			return 1
		}
		return 0
	case nil:
		return 0
	default:
		s := fmt.Sprintf("%v", val)
		f, err := strconv.ParseFloat(s, 64)
		if err != nil {
			return 0
		}
		return f
	}
}
