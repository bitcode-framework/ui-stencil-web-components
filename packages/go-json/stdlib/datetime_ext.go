package stdlib

import (
	"fmt"
	"time"

	"github.com/expr-lang/expr"
)

// RegisterDateTimeExt registers extended date/time functions (Phase 4.5f).
func RegisterDateTimeExt(r *Registry) {
	r.Register(expr.Function("toUnix", func(params ...any) (any, error) {
		dt, err := toTime(params[0])
		if err != nil {
			return nil, fmt.Errorf("toUnix: %s", err)
		}
		return int(dt.Unix()), nil
	}))

	r.Register(expr.Function("fromUnix", func(params ...any) (any, error) {
		ts, ok := toFloat64(params[0])
		if !ok {
			return nil, fmt.Errorf("fromUnix: argument must be a number")
		}
		return time.Unix(int64(ts), 0).UTC(), nil
	}))

	r.Register(expr.Function("toISO", func(params ...any) (any, error) {
		dt, err := toTime(params[0])
		if err != nil {
			return nil, fmt.Errorf("toISO: %s", err)
		}
		return dt.Format(time.RFC3339), nil
	}))

	r.Register(expr.Function("startOfDay", func(params ...any) (any, error) {
		dt, err := toTime(params[0])
		if err != nil {
			return nil, fmt.Errorf("startOfDay: %s", err)
		}
		y, m, d := dt.Date()
		return time.Date(y, m, d, 0, 0, 0, 0, dt.Location()), nil
	}))

	r.Register(expr.Function("endOfDay", func(params ...any) (any, error) {
		dt, err := toTime(params[0])
		if err != nil {
			return nil, fmt.Errorf("endOfDay: %s", err)
		}
		y, m, d := dt.Date()
		return time.Date(y, m, d, 23, 59, 59, 999999999, dt.Location()), nil
	}))

	r.Register(expr.Function("startOfMonth", func(params ...any) (any, error) {
		dt, err := toTime(params[0])
		if err != nil {
			return nil, fmt.Errorf("startOfMonth: %s", err)
		}
		y, m, _ := dt.Date()
		return time.Date(y, m, 1, 0, 0, 0, 0, dt.Location()), nil
	}))

	r.Register(expr.Function("endOfMonth", func(params ...any) (any, error) {
		dt, err := toTime(params[0])
		if err != nil {
			return nil, fmt.Errorf("endOfMonth: %s", err)
		}
		y, m, _ := dt.Date()
		return time.Date(y, m+1, 0, 23, 59, 59, 999999999, dt.Location()), nil
	}))

	r.Register(expr.Function("isWeekend", func(params ...any) (any, error) {
		dt, err := toTime(params[0])
		if err != nil {
			return nil, fmt.Errorf("isWeekend: %s", err)
		}
		wd := dt.Weekday()
		return wd == time.Saturday || wd == time.Sunday, nil
	}))

	r.Register(expr.Function("isBefore", func(params ...any) (any, error) {
		a, err := toTime(params[0])
		if err != nil {
			return nil, fmt.Errorf("isBefore: first argument: %s", err)
		}
		b, err := toTime(params[1])
		if err != nil {
			return nil, fmt.Errorf("isBefore: second argument: %s", err)
		}
		return a.Before(b), nil
	}))

	r.Register(expr.Function("isAfter", func(params ...any) (any, error) {
		a, err := toTime(params[0])
		if err != nil {
			return nil, fmt.Errorf("isAfter: first argument: %s", err)
		}
		b, err := toTime(params[1])
		if err != nil {
			return nil, fmt.Errorf("isAfter: second argument: %s", err)
		}
		return a.After(b), nil
	}))

	r.Register(expr.Function("daysInMonth", func(params ...any) (any, error) {
		dt, err := toTime(params[0])
		if err != nil {
			return nil, fmt.Errorf("daysInMonth: %s", err)
		}
		y, m, _ := dt.Date()
		return time.Date(y, m+1, 0, 0, 0, 0, 0, dt.Location()).Day(), nil
	}))

	r.Register(expr.Function("isLeapYear", func(params ...any) (any, error) {
		dt, err := toTime(params[0])
		if err != nil {
			return nil, fmt.Errorf("isLeapYear: %s", err)
		}
		y := dt.Year()
		return y%4 == 0 && (y%100 != 0 || y%400 == 0), nil
	}))
}
