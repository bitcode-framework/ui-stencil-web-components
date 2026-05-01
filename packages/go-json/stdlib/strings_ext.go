package stdlib

import (
	"fmt"
	"html"
	"strings"
	"unicode"

	"github.com/expr-lang/expr"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

// RegisterStringsExt registers extended string functions (Phase 4.5f).
func RegisterStringsExt(r *Registry) {
	r.Register(expr.Function("capitalize", func(params ...any) (any, error) {
		s, ok := params[0].(string)
		if !ok {
			return nil, fmt.Errorf("capitalize: argument must be a string")
		}
		if s == "" {
			return s, nil
		}
		runes := []rune(s)
		runes[0] = unicode.ToUpper(runes[0])
		return string(runes), nil
	}))

	r.Register(expr.Function("title", func(params ...any) (any, error) {
		s, ok := params[0].(string)
		if !ok {
			return nil, fmt.Errorf("title: argument must be a string")
		}
		caser := cases.Title(language.Und)
		return caser.String(s), nil
	}))

	r.Register(expr.Function("camelCase", func(params ...any) (any, error) {
		s, ok := params[0].(string)
		if !ok {
			return nil, fmt.Errorf("camelCase: argument must be a string")
		}
		return toCamelCase(s, false), nil
	}))

	r.Register(expr.Function("snakeCase", func(params ...any) (any, error) {
		s, ok := params[0].(string)
		if !ok {
			return nil, fmt.Errorf("snakeCase: argument must be a string")
		}
		return toSnakeCase(s), nil
	}))

	r.Register(expr.Function("kebabCase", func(params ...any) (any, error) {
		s, ok := params[0].(string)
		if !ok {
			return nil, fmt.Errorf("kebabCase: argument must be a string")
		}
		return toKebabCase(s), nil
	}))

	r.Register(expr.Function("pascalCase", func(params ...any) (any, error) {
		s, ok := params[0].(string)
		if !ok {
			return nil, fmt.Errorf("pascalCase: argument must be a string")
		}
		return toCamelCase(s, true), nil
	}))

	r.Register(expr.Function("truncate", func(params ...any) (any, error) {
		s, ok := params[0].(string)
		if !ok {
			return nil, fmt.Errorf("truncate: first argument must be a string")
		}
		maxF, ok := toFloat64(params[1])
		if !ok {
			return nil, fmt.Errorf("truncate: second argument must be a number")
		}
		max := int(maxF)
		suffix := "..."
		if len(params) > 2 {
			if sf, ok := params[2].(string); ok {
				suffix = sf
			}
		}
		runes := []rune(s)
		if len(runes) <= max {
			return s, nil
		}
		suffixLen := len([]rune(suffix))
		if max <= suffixLen {
			return string(runes[:max]), nil
		}
		return string(runes[:max-suffixLen]) + suffix, nil
	}))

	r.Register(expr.Function("slugify", func(params ...any) (any, error) {
		s, ok := params[0].(string)
		if !ok {
			return nil, fmt.Errorf("slugify: argument must be a string")
		}
		return slugify(s), nil
	}))

	r.Register(expr.Function("reverse", func(params ...any) (any, error) {
		s, ok := params[0].(string)
		if !ok {
			return nil, fmt.Errorf("reverse: argument must be a string")
		}
		runes := []rune(s)
		for i, j := 0, len(runes)-1; i < j; i, j = i+1, j-1 {
			runes[i], runes[j] = runes[j], runes[i]
		}
		return string(runes), nil
	}))

	r.Register(expr.Function("count", func(params ...any) (any, error) {
		s, ok := params[0].(string)
		if !ok {
			return nil, fmt.Errorf("count: first argument must be a string")
		}
		sub, ok := params[1].(string)
		if !ok {
			return nil, fmt.Errorf("count: second argument must be a string")
		}
		return strings.Count(s, sub), nil
	}))

	r.Register(expr.Function("replaceFirst", func(params ...any) (any, error) {
		s, ok := params[0].(string)
		if !ok {
			return nil, fmt.Errorf("replaceFirst: first argument must be a string")
		}
		old, ok := params[1].(string)
		if !ok {
			return nil, fmt.Errorf("replaceFirst: second argument must be a string")
		}
		newStr, ok := params[2].(string)
		if !ok {
			return nil, fmt.Errorf("replaceFirst: third argument must be a string")
		}
		return strings.Replace(s, old, newStr, 1), nil
	}))

	r.Register(expr.Function("lines", func(params ...any) (any, error) {
		s, ok := params[0].(string)
		if !ok {
			return nil, fmt.Errorf("lines: argument must be a string")
		}
		parts := strings.Split(s, "\n")
		result := make([]any, len(parts))
		for i, p := range parts {
			result[i] = p
		}
		return result, nil
	}))

	r.Register(expr.Function("words", func(params ...any) (any, error) {
		s, ok := params[0].(string)
		if !ok {
			return nil, fmt.Errorf("words: argument must be a string")
		}
		parts := strings.Fields(s)
		result := make([]any, len(parts))
		for i, p := range parts {
			result[i] = p
		}
		return result, nil
	}))

	r.Register(expr.Function("isDigit", func(params ...any) (any, error) {
		s, ok := params[0].(string)
		if !ok {
			return nil, fmt.Errorf("isDigit: argument must be a string")
		}
		if s == "" {
			return false, nil
		}
		for _, r := range s {
			if !unicode.IsDigit(r) {
				return false, nil
			}
		}
		return true, nil
	}))

	r.Register(expr.Function("isAlpha", func(params ...any) (any, error) {
		s, ok := params[0].(string)
		if !ok {
			return nil, fmt.Errorf("isAlpha: argument must be a string")
		}
		if s == "" {
			return false, nil
		}
		for _, r := range s {
			if !unicode.IsLetter(r) {
				return false, nil
			}
		}
		return true, nil
	}))

	r.Register(expr.Function("isAlphaNum", func(params ...any) (any, error) {
		s, ok := params[0].(string)
		if !ok {
			return nil, fmt.Errorf("isAlphaNum: argument must be a string")
		}
		if s == "" {
			return false, nil
		}
		for _, r := range s {
			if !unicode.IsLetter(r) && !unicode.IsDigit(r) {
				return false, nil
			}
		}
		return true, nil
	}))

	r.Register(expr.Function("isEmpty", func(params ...any) (any, error) {
		s, ok := params[0].(string)
		if !ok {
			return nil, fmt.Errorf("isEmpty: argument must be a string")
		}
		return s == "", nil
	}))

	r.Register(expr.Function("isBlank", func(params ...any) (any, error) {
		s, ok := params[0].(string)
		if !ok {
			return nil, fmt.Errorf("isBlank: argument must be a string")
		}
		return strings.TrimSpace(s) == "", nil
	}))

	r.Register(expr.Function("escapeHTML", func(params ...any) (any, error) {
		s, ok := params[0].(string)
		if !ok {
			return nil, fmt.Errorf("escapeHTML: argument must be a string")
		}
		return html.EscapeString(s), nil
	}))

	r.Register(expr.Function("unescapeHTML", func(params ...any) (any, error) {
		s, ok := params[0].(string)
		if !ok {
			return nil, fmt.Errorf("unescapeHTML: argument must be a string")
		}
		return html.UnescapeString(s), nil
	}))
}

func toCamelCase(s string, pascal bool) string {
	w := splitWords(s)
	if len(w) == 0 {
		return ""
	}
	for i, word := range w {
		if i == 0 && !pascal {
			w[i] = strings.ToLower(word)
		} else {
			if len(word) > 0 {
				w[i] = strings.ToUpper(word[:1]) + strings.ToLower(word[1:])
			}
		}
	}
	return strings.Join(w, "")
}

func toSnakeCase(s string) string {
	w := splitWords(s)
	for i, word := range w {
		w[i] = strings.ToLower(word)
	}
	return strings.Join(w, "_")
}

func toKebabCase(s string) string {
	w := splitWords(s)
	for i, word := range w {
		w[i] = strings.ToLower(word)
	}
	return strings.Join(w, "-")
}

func splitWords(s string) []string {
	var words []string
	var current []rune
	runes := []rune(s)
	for i, r := range runes {
		if r == '_' || r == '-' || r == ' ' {
			if len(current) > 0 {
				words = append(words, string(current))
				current = nil
			}
		} else if unicode.IsUpper(r) && i > 0 && len(current) > 0 && !unicode.IsUpper(runes[i-1]) {
			words = append(words, string(current))
			current = []rune{r}
		} else if unicode.IsUpper(r) && i > 0 && len(current) > 1 && unicode.IsUpper(runes[i-1]) && i+1 < len(runes) && unicode.IsLower(runes[i+1]) {
			// Handle "XMLParser" → ["XML", "Parser"]
			words = append(words, string(current))
			current = []rune{r}
		} else {
			current = append(current, r)
		}
	}
	if len(current) > 0 {
		words = append(words, string(current))
	}
	return words
}

func slugify(s string) string {
	s = strings.ToLower(s)
	var result []rune
	prevDash := false
	for _, r := range s {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			result = append(result, r)
			prevDash = false
		} else if !prevDash && len(result) > 0 {
			result = append(result, '-')
			prevDash = true
		}
	}
	return strings.TrimRight(string(result), "-")
}
