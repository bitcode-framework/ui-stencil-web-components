package stdlib

import (
	"encoding/json"
	"fmt"

	"github.com/expr-lang/expr"
)

// RegisterJSON registers toJSON and fromJSON stdlib functions.
func RegisterJSON(r *Registry) {
	r.Register(expr.Function("toJSON",
		func(params ...any) (any, error) {
			if len(params) < 1 {
				return "null", nil
			}
			b, err := json.Marshal(params[0])
			if err != nil {
				return nil, fmt.Errorf("toJSON: %w", err)
			}
			return string(b), nil
		},
		new(func(any) string),
	))

	r.Register(expr.Function("fromJSON",
		func(params ...any) (any, error) {
			if len(params) < 1 {
				return nil, fmt.Errorf("fromJSON: string argument required")
			}
			s, ok := params[0].(string)
			if !ok {
				return nil, fmt.Errorf("fromJSON: argument must be a string")
			}
			var result any
			if err := json.Unmarshal([]byte(s), &result); err != nil {
				return nil, fmt.Errorf("fromJSON: %w", err)
			}
			return result, nil
		},
		new(func(string) any),
	))
}
