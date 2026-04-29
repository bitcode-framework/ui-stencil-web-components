package executor

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/bitcode-framework/go-json/runtime"
)

var conditionVarRe = regexp.MustCompile(`\{\{(\w+(?:\.\w+)*)\}\}`)

func normalizeCondition(condition string) string {
	trimmed := strings.TrimSpace(condition)
	if strings.HasPrefix(trimmed, "{{") && strings.HasSuffix(trimmed, "}}") {
		inner := trimmed[2 : len(trimmed)-2]
		if !strings.Contains(inner, "{{") {
			return strings.TrimSpace(inner)
		}
	}
	return conditionVarRe.ReplaceAllString(condition, "$1")
}

func EvaluateCondition(condition string, execCtx *Context) bool {
	condition = InterpolateTranslations(condition, execCtx)
	condition = normalizeCondition(condition)

	env := make(map[string]any, len(execCtx.Input)+len(execCtx.Variables)+1)
	env["input"] = execCtx.Input
	for k, v := range execCtx.Variables {
		env[k] = v
	}

	result, err := runtime.EvalExprBool(condition, env)
	if err != nil {
		return false
	}
	return result
}

func InterpolateTranslations(s string, execCtx *Context) string {
	for {
		start := strings.Index(s, "{{t('")
		if start == -1 {
			break
		}
		end := strings.Index(s[start:], "')}}")
		if end == -1 {
			break
		}
		end += start + len("')}}")
		key := s[start+len("{{t('") : end-len("')}}")] 
		translated := execCtx.T(key)
		s = s[:start] + translated + s[end:]
	}
	return s
}

func InterpolateString(template string, execCtx *Context) string {
	result := InterpolateTranslations(template, execCtx)
	for key, val := range execCtx.Input {
		result = strings.ReplaceAll(result, "{{input."+key+"}}", formatValue(val))
	}
	for key, val := range execCtx.Variables {
		result = strings.ReplaceAll(result, "{{"+key+"}}", formatValue(val))
	}
	return result
}

func formatValue(v any) string {
	if v == nil {
		return ""
	}
	return fmt.Sprintf("%v", v)
}
