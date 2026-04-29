package stdlib

import (
	"path/filepath"
	"strings"

	"github.com/expr-lang/expr"
)

// RegisterPath registers path utility stdlib functions.
func RegisterPath(r *Registry) {
	r.Register(expr.Function("basename",
		func(params ...any) (any, error) {
			if len(params) < 1 {
				return "", nil
			}
			p, _ := params[0].(string)
			return filepath.Base(p), nil
		},
		new(func(string) string),
	))

	r.Register(expr.Function("dirname",
		func(params ...any) (any, error) {
			if len(params) < 1 {
				return "", nil
			}
			p, _ := params[0].(string)
			return filepath.ToSlash(filepath.Dir(p)), nil
		},
		new(func(string) string),
	))

	r.Register(expr.Function("extname",
		func(params ...any) (any, error) {
			if len(params) < 1 {
				return "", nil
			}
			p, _ := params[0].(string)
			return filepath.Ext(p), nil
		},
		new(func(string) string),
	))

	r.Register(expr.Function("joinpath",
		func(params ...any) (any, error) {
			parts := make([]string, 0, len(params))
			for _, p := range params {
				if s, ok := p.(string); ok {
					parts = append(parts, s)
				}
			}
			return filepath.ToSlash(filepath.Join(parts...)), nil
		},
		new(func(string, string) string),
		new(func(string, string, string) string),
	))

	r.Register(expr.Function("pathsep",
		func(params ...any) (any, error) {
			return string(filepath.Separator), nil
		},
		new(func() string),
	))

	r.Register(expr.Function("cleanpath",
		func(params ...any) (any, error) {
			if len(params) < 1 {
				return "", nil
			}
			p, _ := params[0].(string)
			cleaned := filepath.Clean(p)
			return filepath.ToSlash(cleaned), nil
		},
		new(func(string) string),
	))

	r.Register(expr.Function("isabs",
		func(params ...any) (any, error) {
			if len(params) < 1 {
				return false, nil
			}
			p, _ := params[0].(string)
			return filepath.IsAbs(p), nil
		},
		new(func(string) bool),
	))

	r.Register(expr.Function("stemname",
		func(params ...any) (any, error) {
			if len(params) < 1 {
				return "", nil
			}
			p, _ := params[0].(string)
			base := filepath.Base(p)
			ext := filepath.Ext(base)
			return strings.TrimSuffix(base, ext), nil
		},
		new(func(string) string),
	))
}
