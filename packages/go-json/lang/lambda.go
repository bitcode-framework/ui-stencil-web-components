package lang

import (
	"fmt"
	"strings"
)

// Lambda represents a parsed lambda expression: fn(params) => body.
type Lambda struct {
	Params []string
	Body   string
}

// ParseLambda parses "fn(x, y) => x + y" into a Lambda struct.
// Returns nil if the expression is not a lambda.
func ParseLambda(expr string) *Lambda {
	expr = strings.TrimSpace(expr)
	if !strings.HasPrefix(expr, "fn(") {
		return nil
	}

	// Find closing paren for the parameter list.
	closeIdx := findMatchingParen(expr, 2)
	if closeIdx == -1 {
		return nil
	}

	// Extract params.
	paramStr := strings.TrimSpace(expr[3:closeIdx])
	var params []string
	if paramStr != "" {
		for _, p := range strings.Split(paramStr, ",") {
			trimmed := strings.TrimSpace(p)
			if trimmed == "" {
				return nil // invalid: empty param
			}
			if !isValidIdentifier(trimmed) {
				return nil
			}
			params = append(params, trimmed)
		}
	}

	// Find "=>" after closing paren.
	rest := expr[closeIdx+1:]
	trimmedRest := strings.TrimSpace(rest)
	if !strings.HasPrefix(trimmedRest, "=>") {
		return nil
	}

	body := strings.TrimSpace(trimmedRest[2:])
	if body == "" {
		return nil
	}

	return &Lambda{Params: params, Body: body}
}

// findMatchingParen finds the index of the closing ')' that matches the '(' at startIdx.
// Respects string literals (single and double quotes) and nested parentheses.
func findMatchingParen(s string, startIdx int) int {
	depth := 0
	inSingleQuote := false
	inDoubleQuote := false

	for i := startIdx; i < len(s); i++ {
		ch := s[i]

		// Handle escape sequences inside strings.
		if (inSingleQuote || inDoubleQuote) && ch == '\\' {
			i++ // skip next char
			continue
		}

		if ch == '\'' && !inDoubleQuote {
			inSingleQuote = !inSingleQuote
			continue
		}
		if ch == '"' && !inSingleQuote {
			inDoubleQuote = !inDoubleQuote
			continue
		}

		if inSingleQuote || inDoubleQuote {
			continue
		}

		if ch == '(' {
			depth++
		} else if ch == ')' {
			depth--
			if depth == 0 {
				return i
			}
		}
	}
	return -1
}

// FindInlineLambdas finds all inline lambda expressions in an expression string.
// Returns a slice of [start, end) positions for each lambda found.
// Handles nested function calls, string literals, and complex expressions.
func FindInlineLambdas(expr string) [][2]int {
	var results [][2]int
	i := 0
	for i < len(expr) {
		// Skip string literals.
		if expr[i] == '\'' || expr[i] == '"' {
			quote := expr[i]
			i++
			for i < len(expr) {
				if expr[i] == '\\' {
					i += 2
					continue
				}
				if expr[i] == quote {
					i++
					break
				}
				i++
			}
			continue
		}

		// Look for "fn(" pattern.
		if i+3 <= len(expr) && expr[i:i+3] == "fn(" {
			// Make sure it's not part of a larger identifier (e.g., "xfn(").
			if i > 0 && isIdentChar(expr[i-1]) {
				i++
				continue
			}

			start := i
			end := findLambdaEnd(expr, start)
			if end > start {
				results = append(results, [2]int{start, end})
				i = end
				continue
			}
		}
		i++
	}
	return results
}

// findLambdaEnd finds the end position of a lambda expression starting at pos.
// A lambda is: fn(params) => body
// The body ends at: end of string, or an unbalanced comma/paren that belongs to the outer expression.
func findLambdaEnd(expr string, pos int) int {
	// First, find the closing paren of params.
	closeIdx := findMatchingParen(expr, pos+2)
	if closeIdx == -1 {
		return -1
	}

	// Find "=>" after the closing paren.
	rest := expr[closeIdx+1:]
	trimmed := strings.TrimLeft(rest, " \t")
	if !strings.HasPrefix(trimmed, "=>") {
		return -1
	}

	// Calculate where the body starts.
	arrowOffset := closeIdx + 1 + (len(rest) - len(trimmed)) + 2
	bodyStart := arrowOffset

	if bodyStart >= len(expr) {
		return -1
	}

	// Find where the body ends.
	// The body ends when we encounter an unbalanced ')' or ',' at depth 0
	// that belongs to the outer expression context.
	return findBodyEnd(expr, bodyStart)
}

// findBodyEnd finds where a lambda body ends.
// The body ends at: end of string, or an unbalanced ')' or ','
// at paren depth 0 that belongs to the outer context.
func findBodyEnd(expr string, start int) int {
	parenDepth := 0
	bracketDepth := 0
	inSingleQuote := false
	inDoubleQuote := false

	i := start
	for i < len(expr) {
		ch := expr[i]

		// Handle escape sequences inside strings.
		if (inSingleQuote || inDoubleQuote) && ch == '\\' {
			i += 2
			continue
		}

		if ch == '\'' && !inDoubleQuote {
			inSingleQuote = !inSingleQuote
			i++
			continue
		}
		if ch == '"' && !inSingleQuote {
			inDoubleQuote = !inDoubleQuote
			i++
			continue
		}

		if inSingleQuote || inDoubleQuote {
			i++
			continue
		}

		switch ch {
		case '(':
			parenDepth++
		case ')':
			if parenDepth == 0 {
				// Unbalanced — belongs to outer context.
				return i
			}
			parenDepth--
		case '[':
			bracketDepth++
		case ']':
			if bracketDepth == 0 {
				return i
			}
			bracketDepth--
		case ',':
			if parenDepth == 0 && bracketDepth == 0 {
				// Comma at top level — belongs to outer context.
				return i
			}
		}
		i++
	}

	return len(expr)
}

// PreprocessLambdas finds lambda expressions in an expression string,
// compiles each one, and returns the modified expression with lambda
// references replaced by placeholder names, plus the compiled functions.
//
// For standalone lambda (entire expression is a lambda):
//   - Returns empty string and a single entry "__direct_lambda" in the map.
//
// For inline lambdas (e.g., mapFn(items, fn(x) => x * 2)):
//   - Replaces each lambda with "__lambda_N" and returns compiled funcs.
func PreprocessLambdas(expr string, engine ExprEngine, env map[string]any) (string, map[string]func(...any) (any, error)) {
	trimmed := strings.TrimSpace(expr)

	// Case 1: The entire expression is a standalone lambda.
	if lambda := ParseLambda(trimmed); lambda != nil {
		funcs := map[string]func(...any) (any, error){
			"__direct_lambda": CompileLambda(lambda, engine, env),
		}
		return "", funcs
	}

	// Case 2: Find inline lambdas within the expression.
	positions := FindInlineLambdas(trimmed)
	if len(positions) == 0 {
		return expr, nil
	}

	funcs := make(map[string]func(...any) (any, error), len(positions))
	// Process from end to start so positions remain valid.
	processed := trimmed
	for idx := len(positions) - 1; idx >= 0; idx-- {
		pos := positions[idx]
		lambdaStr := processed[pos[0]:pos[1]]
		lambda := ParseLambda(lambdaStr)
		if lambda == nil {
			continue
		}
		name := fmt.Sprintf("__lambda_%d", idx)
		funcs[name] = CompileLambda(lambda, engine, env)
		processed = processed[:pos[0]] + name + processed[pos[1]:]
	}

	return processed, funcs
}

// CompileLambda creates a Go function from a Lambda definition.
// capturedEnv is a snapshot of the current scope at definition time.
func CompileLambda(lambda *Lambda, engine ExprEngine, capturedEnv map[string]any) func(...any) (any, error) {
	// Snapshot the environment at definition time.
	snapshot := make(map[string]any, len(capturedEnv))
	for k, v := range capturedEnv {
		snapshot[k] = v
	}

	return func(args ...any) (any, error) {
		// Create evaluation environment: captured scope + param bindings.
		evalEnv := make(map[string]any, len(snapshot)+len(lambda.Params))
		for k, v := range snapshot {
			evalEnv[k] = v
		}

		// Bind positional args to param names.
		for i, param := range lambda.Params {
			if i < len(args) {
				evalEnv[param] = args[i]
			} else {
				evalEnv[param] = nil
			}
		}

		// Evaluate body as expr-lang expression.
		return engine.Eval(lambda.Body, evalEnv)
	}
}

// isValidIdentifier checks if a string is a valid go-json identifier.
func isValidIdentifier(s string) bool {
	if s == "" {
		return false
	}
	for i, ch := range s {
		if i == 0 {
			if !isIdentStart(byte(ch)) {
				return false
			}
		} else {
			if !isIdentChar(byte(ch)) {
				return false
			}
		}
	}
	return true
}

// isIdentStart checks if a byte can start an identifier.
func isIdentStart(ch byte) bool {
	return (ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') || ch == '_'
}

// isIdentChar checks if a byte can be part of an identifier.
func isIdentChar(ch byte) bool {
	return isIdentStart(ch) || (ch >= '0' && ch <= '9')
}
