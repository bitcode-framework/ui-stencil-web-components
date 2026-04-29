package stdlib

import (
	"encoding/json"
	"fmt"

	"github.com/expr-lang/expr"
)

// RegisterJSON registers JSON stdlib functions.
//
// NOTE: toJSON and fromJSON are already provided by expr-lang as built-in
// functions (since v1.16+). The built-in toJSON uses json.MarshalIndent
// (pretty-printed), while toCompactJSON uses json.Marshal (compact).
// fromJSON is NOT re-registered here — use the expr-lang built-in.
func RegisterJSON(r *Registry) {
	r.Register(expr.Function("toCompactJSON",
		func(params ...any) (any, error) {
			if len(params) < 1 {
				return "null", nil
			}
			b, err := json.Marshal(params[0])
			if err != nil {
				return nil, fmt.Errorf("toCompactJSON: %w", err)
			}
			return string(b), nil
		},
		new(func(any) string),
	))
}
