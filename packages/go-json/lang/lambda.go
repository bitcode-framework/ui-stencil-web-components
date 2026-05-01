package lang

import (
	"fmt"
	"strings"
	"sync/atomic"
)

// Lambda represents a parsed lambda expression.
// Anonymous: fn(params) => body (Name is empty)
// Named:    fn name(params) => body (Name is set, enables recursion)
type Lambda struct {
	Name   string
	Params []string
	Body   string
}

// ParseLambda parses lambda expressions into a Lambda struct.
// Supports two forms:
//   - Anonymous: "fn(x, y) => x + y"
//   - Named:    "fn factorial(n) => n <= 1 ? 1 : n * factorial(n - 1)"
//
// Returns nil if the expression is not a lambda.
func ParseLambda(expr string) *Lambda {
	expr = strings.TrimSpace(expr)
	if !strings.HasPrefix(expr, "fn") {
		return nil
	}

	rest := expr[2:]
	if len(rest) == 0 {
		return nil
	}

	var name string
	var parenStart int

	if rest[0] == '(' {
		// Anonymous: fn(params) => body
		parenStart = 2
	} else if rest[0] == ' ' || isIdentStart(rest[0]) {
		// Named: fn name(params) => body
		// Find the opening paren
		trimmedRest := strings.TrimLeft(rest, " ")
		if len(trimmedRest) == 0 || !isIdentStart(trimmedRest[0]) {
			return nil
		}
		nameEnd := 0
		for nameEnd < len(trimmedRest) && isIdentChar(trimmedRest[nameEnd]) {
			nameEnd++
		}
		name = trimmedRest[:nameEnd]
		afterName := strings.TrimLeft(trimmedRest[nameEnd:], " ")
		if len(afterName) == 0 || afterName[0] != '(' {
			return nil
		}
		// Calculate parenStart relative to original expr
		parenStart = len(expr) - len(afterName)
	} else {
		return nil
	}

	closeIdx := findMatchingParen(expr, parenStart)
	if closeIdx == -1 {
		return nil
	}

	paramStr := strings.TrimSpace(expr[parenStart+1 : closeIdx])
	var params []string
	if paramStr != "" {
		for _, p := range strings.Split(paramStr, ",") {
			trimmed := strings.TrimSpace(p)
			if trimmed == "" {
				return nil
			}
			if !isValidIdentifier(trimmed) {
				return nil
			}
			params = append(params, trimmed)
		}
	}

	afterParen := expr[closeIdx+1:]
	trimmedAfter := strings.TrimSpace(afterParen)
	if !strings.HasPrefix(trimmedAfter, "=>") {
		return nil
	}

	body := strings.TrimSpace(trimmedAfter[2:])
	if body == "" {
		return nil
	}

	return &Lambda{Name: name, Params: params, Body: body}
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
// Detects both anonymous (fn(...)) and named (fn name(...)) forms.
func FindInlineLambdas(expr string) [][2]int {
	var results [][2]int
	i := 0
	for i < len(expr) {
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

		// Detect "fn(" or "fn " followed by identifier then "("
		if i+2 <= len(expr) && expr[i:i+2] == "fn" {
			if i > 0 && isIdentChar(expr[i-1]) {
				i++
				continue
			}

			if i+2 < len(expr) && (expr[i+2] == '(' || expr[i+2] == ' ' || isIdentStart(expr[i+2])) {
				start := i
				end := findLambdaEnd(expr, start)
				if end > start {
					results = append(results, [2]int{start, end})
					i = end
					continue
				}
			}
		}
		i++
	}
	return results
}

// findLambdaEnd finds the end position of a lambda expression starting at pos.
// Handles both fn(params) => body and fn name(params) => body.
func findLambdaEnd(expr string, pos int) int {
	// Find the opening paren — skip "fn" + optional name
	parenIdx := -1
	i := pos + 2 // skip "fn"
	for i < len(expr) {
		if expr[i] == '(' {
			parenIdx = i
			break
		}
		if expr[i] != ' ' && !isIdentChar(expr[i]) {
			return -1
		}
		i++
	}
	if parenIdx == -1 {
		return -1
	}

	closeIdx := findMatchingParen(expr, parenIdx)
	if closeIdx == -1 {
		return -1
	}

	rest := expr[closeIdx+1:]
	trimmed := strings.TrimLeft(rest, " \t")
	if !strings.HasPrefix(trimmed, "=>") {
		return -1
	}

	arrowOffset := closeIdx + 1 + (len(rest) - len(trimmed)) + 2
	if arrowOffset >= len(expr) {
		return -1
	}

	return findBodyEnd(expr, arrowOffset)
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
func PreprocessLambdas(expr string, engine ExprEngine, env map[string]any, maxDepth int) (string, map[string]func(...any) (any, error)) {
	trimmed := strings.TrimSpace(expr)

	if lambda := ParseLambda(trimmed); lambda != nil {
		funcs := map[string]func(...any) (any, error){
			"__direct_lambda": CompileLambda(lambda, engine, env, maxDepth),
		}
		return "", funcs
	}

	positions := FindInlineLambdas(trimmed)
	if len(positions) == 0 {
		return expr, nil
	}

	funcs := make(map[string]func(...any) (any, error), len(positions))
	processed := trimmed
	for idx := len(positions) - 1; idx >= 0; idx-- {
		pos := positions[idx]
		lambdaStr := processed[pos[0]:pos[1]]
		lambda := ParseLambda(lambdaStr)
		if lambda == nil {
			continue
		}
		name := fmt.Sprintf("__lambda_%d", idx)
		funcs[name] = CompileLambda(lambda, engine, env, maxDepth)
		processed = processed[:pos[0]] + name + processed[pos[1]:]
	}

	return processed, funcs
}

// CompileLambda creates a Go function from a Lambda definition.
// capturedEnv is a snapshot of the current scope at definition time.
// maxDepth limits recursion depth for named lambdas (0 = use default 1000).
func CompileLambda(lambda *Lambda, engine ExprEngine, capturedEnv map[string]any, maxDepth int) func(...any) (any, error) {
	if maxDepth <= 0 {
		maxDepth = 1000
	}

	snapshot := make(map[string]any, len(capturedEnv))
	for k, v := range capturedEnv {
		snapshot[k] = v
	}

	if lambda.Name == "" {
		return func(args ...any) (any, error) {
			evalEnv := make(map[string]any, len(snapshot)+len(lambda.Params))
			for k, v := range snapshot {
				evalEnv[k] = v
			}
			for i, param := range lambda.Params {
				if i < len(args) {
					evalEnv[param] = args[i]
				} else {
					evalEnv[param] = nil
				}
			}
			return engine.Eval(lambda.Body, evalEnv)
		}
	}

	// Named lambda: supports self-recursion via forward-declared closure.
	var self func(...any) (any, error)
	var depth int64

	self = func(args ...any) (any, error) {
		current := atomic.AddInt64(&depth, 1)
		defer atomic.AddInt64(&depth, -1)
		if current > int64(maxDepth) {
			return nil, fmt.Errorf("lambda '%s': recursion depth limit (%d) exceeded", lambda.Name, maxDepth)
		}

		evalEnv := make(map[string]any, len(snapshot)+len(lambda.Params)+1)
		for k, v := range snapshot {
			evalEnv[k] = v
		}
		for i, param := range lambda.Params {
			if i < len(args) {
				evalEnv[param] = args[i]
			} else {
				evalEnv[param] = nil
			}
		}
		evalEnv[lambda.Name] = self

		return engine.Eval(lambda.Body, evalEnv)
	}

	return self
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
