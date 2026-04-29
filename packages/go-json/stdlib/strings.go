package stdlib

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/expr-lang/expr"
)

func RegisterStrings(r *Registry) {
	r.Register(expr.Function("padLeft", func(params ...any) (any, error) {
		s, ok := params[0].(string)
		if !ok {
			return nil, fmt.Errorf("padLeft: first argument must be a string")
		}
		length, ok := toFloat64(params[1])
		if !ok {
			return nil, fmt.Errorf("padLeft: second argument must be a number")
		}
		pad := " "
		if len(params) > 2 {
			if p, ok := params[2].(string); ok && len(p) > 0 {
				pad = p
			}
		}
		n := int(length)
		runes := []rune(s)
		if len(runes) >= n {
			return s, nil
		}
		needed := n - len(runes)
		padRunes := []rune(pad)
		var prefix []rune
		for len(prefix) < needed {
			prefix = append(prefix, padRunes...)
		}
		return string(prefix[:needed]) + s, nil
	}))

	r.Register(expr.Function("padRight", func(params ...any) (any, error) {
		s, ok := params[0].(string)
		if !ok {
			return nil, fmt.Errorf("padRight: first argument must be a string")
		}
		length, ok := toFloat64(params[1])
		if !ok {
			return nil, fmt.Errorf("padRight: second argument must be a number")
		}
		pad := " "
		if len(params) > 2 {
			if p, ok := params[2].(string); ok && len(p) > 0 {
				pad = p
			}
		}
		n := int(length)
		runes := []rune(s)
		if len(runes) >= n {
			return s, nil
		}
		needed := n - len(runes)
		padRunes := []rune(pad)
		var suffix []rune
		for len(suffix) < needed {
			suffix = append(suffix, padRunes...)
		}
		return s + string(suffix[:needed]), nil
	}))

	r.Register(expr.Function("substring", func(params ...any) (any, error) {
		s, ok := params[0].(string)
		if !ok {
			return nil, fmt.Errorf("substring: first argument must be a string")
		}
		startF, ok := toFloat64(params[1])
		if !ok {
			return nil, fmt.Errorf("substring: second argument must be a number")
		}
		runes := []rune(s)
		start := int(startF)
		if start < 0 {
			start = 0
		}
		if start >= len(runes) {
			return "", nil
		}
		end := len(runes)
		if len(params) > 2 {
			if endF, ok := toFloat64(params[2]); ok {
				end = int(endF)
			}
		}
		if end < start {
			return "", nil
		}
		if end > len(runes) {
			end = len(runes)
		}
		return string(runes[start:end]), nil
	}))

	r.Register(expr.Function("format", func(params ...any) (any, error) {
		if len(params) < 1 {
			return nil, fmt.Errorf("format: requires at least a template string")
		}
		template, ok := params[0].(string)
		if !ok {
			return nil, fmt.Errorf("format: first argument must be a string")
		}
		args := params[1:]
		return fmt.Sprintf(template, args...), nil
	}))

	r.Register(expr.Function("strMatches", func(params ...any) (any, error) {
		s, ok := params[0].(string)
		if !ok {
			return nil, fmt.Errorf("strMatches: first argument must be a string")
		}
		pattern, ok := params[1].(string)
		if !ok {
			return nil, fmt.Errorf("strMatches: second argument must be a string")
		}
		re, err := regexp.Compile(pattern)
		if err != nil {
			return nil, fmt.Errorf("strMatches: invalid regex pattern: %s", err.Error())
		}
		return re.MatchString(s), nil
	}))

	r.Register(expr.Function("strContains", func(params ...any) (any, error) {
		s, ok := params[0].(string)
		if !ok {
			return nil, fmt.Errorf("strContains: first argument must be a string")
		}
		substr, ok := params[1].(string)
		if !ok {
			return nil, fmt.Errorf("strContains: second argument must be a string")
		}
		return strings.Contains(s, substr), nil
	}, new(func(string, string) bool)))

	r.Register(expr.Function("strStartsWith", func(params ...any) (any, error) {
		s, ok := params[0].(string)
		if !ok {
			return nil, fmt.Errorf("strStartsWith: first argument must be a string")
		}
		prefix, ok := params[1].(string)
		if !ok {
			return nil, fmt.Errorf("strStartsWith: second argument must be a string")
		}
		return strings.HasPrefix(s, prefix), nil
	}, new(func(string, string) bool)))

	r.Register(expr.Function("strEndsWith", func(params ...any) (any, error) {
		s, ok := params[0].(string)
		if !ok {
			return nil, fmt.Errorf("strEndsWith: first argument must be a string")
		}
		suffix, ok := params[1].(string)
		if !ok {
			return nil, fmt.Errorf("strEndsWith: second argument must be a string")
		}
		return strings.HasSuffix(s, suffix), nil
	}, new(func(string, string) bool)))
}
