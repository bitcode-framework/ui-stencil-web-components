package io

import (
	"fmt"
	"strings"
)

// TranslateQuery converts universal placeholder syntax (? and :name) to driver-specific syntax.
// Positional mode: args is []any, placeholders are ?
// Named mode: args is map[string]any, placeholders are :name
// Escape: ?? becomes literal ?
func TranslateQuery(query, driver string, args any) (string, []any, error) {
	if query == "" {
		return "", nil, nil
	}

	hasPositional := false
	hasNamed := false

	for i := 0; i < len(query); i++ {
		if query[i] == '\'' {
			i++
			for i < len(query) && query[i] != '\'' {
				i++
			}
			continue
		}
		if query[i] == '?' {
			if i+1 < len(query) && query[i+1] == '?' {
				i++
				continue
			}
			hasPositional = true
		}
		if query[i] == ':' && i+1 < len(query) && isIdentStart(query[i+1]) {
			hasNamed = true
		}
	}

	if hasPositional && hasNamed {
		return "", nil, fmt.Errorf("cannot mix positional (?) and named (:name) placeholders in the same query")
	}

	if hasPositional {
		return translatePositional(query, driver, args)
	}
	if hasNamed {
		return translateNamed(query, driver, args)
	}

	if strings.Contains(query, "??") {
		return strings.ReplaceAll(query, "??", "?"), nil, nil
	}

	return query, nil, nil
}

func translatePositional(query, driver string, args any) (string, []any, error) {
	argSlice, ok := args.([]any)
	if !ok {
		if args == nil {
			argSlice = nil
		} else {
			return "", nil, fmt.Errorf("positional placeholders require []any args, got %T", args)
		}
	}

	var result strings.Builder
	paramIndex := 0

	for i := 0; i < len(query); i++ {
		if query[i] == '?' && i+1 < len(query) && query[i+1] == '?' {
			result.WriteByte('?')
			i++
			continue
		}
		if query[i] == '\'' {
			result.WriteByte('\'')
			i++
			for i < len(query) && query[i] != '\'' {
				result.WriteByte(query[i])
				i++
			}
			if i < len(query) {
				result.WriteByte('\'')
			}
			continue
		}
		if query[i] == '?' {
			paramIndex++
			result.WriteString(driverPlaceholder(driver, paramIndex))
			continue
		}
		result.WriteByte(query[i])
	}

	if argSlice != nil && paramIndex != len(argSlice) {
		return "", nil, fmt.Errorf("placeholder count (%d) does not match arg count (%d)", paramIndex, len(argSlice))
	}

	return result.String(), argSlice, nil
}

func translateNamed(query, driver string, args any) (string, []any, error) {
	argsMap, ok := args.(map[string]any)
	if !ok {
		if args == nil {
			argsMap = map[string]any{}
		} else {
			return "", nil, fmt.Errorf("named placeholders require map[string]any args, got %T", args)
		}
	}

	var result strings.Builder
	var orderedArgs []any
	paramIndex := 0

	for i := 0; i < len(query); i++ {
		if query[i] == '\'' {
			result.WriteByte('\'')
			i++
			for i < len(query) && query[i] != '\'' {
				result.WriteByte(query[i])
				i++
			}
			if i < len(query) {
				result.WriteByte('\'')
			}
			continue
		}
		if query[i] == ':' && i+1 < len(query) && isIdentStart(query[i+1]) {
			name := extractName(query, i+1)
			i += len(name)

			val, ok := argsMap[name]
			if !ok {
				return "", nil, fmt.Errorf("named parameter :%s not found in args", name)
			}

			paramIndex++
			orderedArgs = append(orderedArgs, val)
			result.WriteString(driverPlaceholder(driver, paramIndex))
			continue
		}
		result.WriteByte(query[i])
	}

	return result.String(), orderedArgs, nil
}

func driverPlaceholder(driver string, index int) string {
	switch driver {
	case "postgres":
		return fmt.Sprintf("$%d", index)
	case "sqlserver":
		return fmt.Sprintf("@p%d", index)
	case "oracle":
		return fmt.Sprintf(":%d", index)
	default:
		return "?"
	}
}

func isIdentStart(b byte) bool {
	return (b >= 'a' && b <= 'z') || (b >= 'A' && b <= 'Z') || b == '_'
}

func isIdentChar(b byte) bool {
	return isIdentStart(b) || (b >= '0' && b <= '9')
}

func extractName(query string, start int) string {
	end := start
	for end < len(query) && isIdentChar(query[end]) {
		end++
	}
	return query[start:end]
}
