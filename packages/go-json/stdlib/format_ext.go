package stdlib

import (
	"fmt"
	"math"
	"strings"

	"github.com/expr-lang/expr"
)

// RegisterFormatExt registers number formatting functions (Phase 4.5f).
func RegisterFormatExt(r *Registry) {
	r.Register(expr.Function("toFixed", func(params ...any) (any, error) {
		x, ok := toFloat64(params[0])
		if !ok {
			return nil, fmt.Errorf("toFixed: first argument must be a number")
		}
		decimals := 2
		if len(params) > 1 {
			if d, ok := toFloat64(params[1]); ok {
				decimals = int(d)
			}
		}
		return fmt.Sprintf("%.*f", decimals, x), nil
	}))

	r.Register(expr.Function("formatNumber", func(params ...any) (any, error) {
		x, ok := toFloat64(params[0])
		if !ok {
			return nil, fmt.Errorf("formatNumber: first argument must be a number")
		}
		decimals := 0
		sep := ","
		decSep := "."
		if len(params) > 1 {
			if d, ok := toFloat64(params[1]); ok {
				decimals = int(d)
			}
		}
		if len(params) > 2 {
			if s, ok := params[2].(string); ok {
				sep = s
			}
		}
		if len(params) > 3 {
			if s, ok := params[3].(string); ok {
				decSep = s
			}
		}

		negative := x < 0
		formatted := fmt.Sprintf("%.*f", decimals, math.Abs(x))
		parts := strings.Split(formatted, ".")
		intPart := parts[0]

		var result []byte
		for i, c := range intPart {
			if i > 0 && (len(intPart)-i)%3 == 0 {
				result = append(result, []byte(sep)...)
			}
			result = append(result, byte(c))
		}

		final := string(result)
		if len(parts) > 1 {
			final += decSep + parts[1]
		}
		if negative {
			final = "-" + final
		}
		return final, nil
	}))

	r.Register(expr.Function("formatBytes", func(params ...any) (any, error) {
		n, ok := toFloat64(params[0])
		if !ok {
			return nil, fmt.Errorf("formatBytes: argument must be a number")
		}
		units := []string{"B", "KB", "MB", "GB", "TB", "PB"}
		size := math.Abs(n)
		i := 0
		for size >= 1024 && i < len(units)-1 {
			size /= 1024
			i++
		}
		if n < 0 {
			return fmt.Sprintf("-%.2f %s", size, units[i]), nil
		}
		return fmt.Sprintf("%.2f %s", size, units[i]), nil
	}))

	r.Register(expr.Function("formatPercent", func(params ...any) (any, error) {
		x, ok := toFloat64(params[0])
		if !ok {
			return nil, fmt.Errorf("formatPercent: first argument must be a number")
		}
		decimals := 1
		if len(params) > 1 {
			if d, ok := toFloat64(params[1]); ok {
				decimals = int(d)
			}
		}
		return fmt.Sprintf("%.*f%%", decimals, x*100), nil
	}))
}
