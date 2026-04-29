package stdlib

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/expr-lang/expr"
)

// RegisterDateTime registers date/time helper functions.
func RegisterDateTime(r *Registry) {
	r.Register(expr.Function("formatDate", func(params ...any) (any, error) {
		dt, err := toTime(params[0])
		if err != nil {
			return nil, fmt.Errorf("formatDate: first argument must be a time value: %s", err)
		}
		format, ok := params[1].(string)
		if !ok {
			return nil, fmt.Errorf("formatDate: second argument must be a format string")
		}
		return dt.Format(translateUniversalDateFormat(format)), nil
	}))

	r.Register(expr.Function("addDuration", func(params ...any) (any, error) {
		dt, err := toTime(params[0])
		if err != nil {
			return nil, fmt.Errorf("addDuration: first argument must be a time value: %s", err)
		}
		durStr, ok := params[1].(string)
		if !ok {
			return nil, fmt.Errorf("addDuration: second argument must be a duration string (e.g. '1h30m')")
		}
		dur, err := parseDurationExtended(durStr)
		if err != nil {
			return nil, fmt.Errorf("addDuration: invalid duration '%s': %s", durStr, err)
		}
		return dt.Add(dur), nil
	}))

	r.Register(expr.Function("diffDates", func(params ...any) (any, error) {
		a, err := toTime(params[0])
		if err != nil {
			return nil, fmt.Errorf("diffDates: first argument must be a time value: %s", err)
		}
		b, err := toTime(params[1])
		if err != nil {
			return nil, fmt.Errorf("diffDates: second argument must be a time value: %s", err)
		}
		return a.Sub(b).String(), nil
	}))
}

// parseDurationExtended extends Go's time.ParseDuration with day/week support.
// "7d" → 7*24h, "2w" → 14*24h, "1d12h" → 36h. Falls back to time.ParseDuration.
func parseDurationExtended(s string) (time.Duration, error) {
	original := s
	var total time.Duration

	for len(s) > 0 {
		i := 0
		for i < len(s) && (s[i] >= '0' && s[i] <= '9' || s[i] == '.') {
			i++
		}
		if i == 0 {
			break
		}
		numStr := s[:i]
		if i >= len(s) {
			break
		}

		unit := s[i]
		s = s[i+1:]

		switch unit {
		case 'd':
			n, err := strconv.ParseFloat(numStr, 64)
			if err != nil {
				return 0, fmt.Errorf("invalid number in duration: %s", numStr)
			}
			total += time.Duration(n * float64(24*time.Hour))
		case 'w':
			n, err := strconv.ParseFloat(numStr, 64)
			if err != nil {
				return 0, fmt.Errorf("invalid number in duration: %s", numStr)
			}
			total += time.Duration(n * float64(7*24*time.Hour))
		default:
			remainder := numStr + string(unit) + s
			d, err := time.ParseDuration(remainder)
			if err != nil {
				return 0, fmt.Errorf("invalid duration: %s", original)
			}
			total += d
			return total, nil
		}
	}

	if total == 0 && len(original) > 0 {
		return time.ParseDuration(original)
	}

	return total, nil
}

func toTime(v any) (time.Time, error) {
	switch t := v.(type) {
	case time.Time:
		return t, nil
	case string:
		formats := []string{
			time.RFC3339,
			"2006-01-02T15:04:05",
			"2006-01-02",
		}
		for _, f := range formats {
			if parsed, err := time.Parse(f, t); err == nil {
				return parsed, nil
			}
		}
		return time.Time{}, fmt.Errorf("cannot parse '%s' as date", t)
	default:
		return time.Time{}, fmt.Errorf("expected time or string, got %T", v)
	}
}

func translateUniversalDateFormat(layout string) string {
	if strings.Contains(layout, "2006") {
		return layout
	}
	if !strings.ContainsAny(layout, "YMDHhms") {
		return layout
	}
	replacer := strings.NewReplacer(
		"YYYY", "2006",
		"YY", "06",
		"MM", "01",
		"DD", "02",
		"HH", "15",
		"hh", "03",
		"mm", "04",
		"ss", "05",
		"SSS", "000",
		"TT", "PM",
	)
	return replacer.Replace(layout)
}
