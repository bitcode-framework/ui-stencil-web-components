package yaegi

import (
	"context"
	"fmt"
	"reflect"
	"time"

	runtimes "github.com/bitcode-framework/go-json-runtimes"
	"github.com/traefik/yaegi/interp"
)

type YaegiVM struct {
	interp    *interp.Interpreter
	opts      runtimes.VMOptions
	config    runtimes.YaegiConfig
	bridge    map[string]any
	params    map[string]any
	cancelled bool
}

func (v *YaegiVM) InjectBridge(bridge map[string]any) error {
	v.bridge = bridge

	symbols := interp.Exports{
		"bitcode/bitcode": {
			"Bridge": reflect.ValueOf(&bridge).Elem(),
		},
	}
	v.interp.Use(symbols)
	return nil
}

func (v *YaegiVM) InjectParams(params map[string]any) error {
	v.params = params

	symbols := interp.Exports{
		"params/params": {
			"Params": reflect.ValueOf(&params).Elem(),
		},
	}
	v.interp.Use(symbols)
	return nil
}

func (v *YaegiVM) Execute(code string, filename string) (any, error) {
	_, err := v.interp.Eval(code)
	if err != nil {
		return nil, fmt.Errorf("eval error in %s: %w", filename, err)
	}

	fnVal, err := v.interp.Eval("main.Execute")
	if err != nil {
		return nil, fmt.Errorf("function main.Execute not found in %s: %w", filename, err)
	}

	if !fnVal.IsValid() || fnVal.Kind() != reflect.Func {
		return nil, fmt.Errorf("main.Execute is not a function in %s", filename)
	}

	timeout := v.opts.Timeout
	if timeout == 0 {
		timeout = 30 * time.Second
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	return v.callExecute(ctx, fnVal)
}

func (v *YaegiVM) callExecute(ctx context.Context, fnVal reflect.Value) (any, error) {
	fnType := fnVal.Type()

	type execResult struct {
		value any
		err   error
	}
	done := make(chan execResult, 1)

	go func() {
		defer func() {
			if r := recover(); r != nil {
				done <- execResult{nil, fmt.Errorf("panic in yaegi script: %v", r)}
			}
		}()

		var results []reflect.Value
		switch fnType.NumIn() {
		case 2:
			results = fnVal.Call([]reflect.Value{
				reflect.ValueOf(ctx),
				reflect.ValueOf(v.params),
			})
		case 1:
			results = fnVal.Call([]reflect.Value{
				reflect.ValueOf(v.params),
			})
		case 0:
			results = fnVal.Call(nil)
		default:
			done <- execResult{nil, fmt.Errorf("unsupported Execute signature: %d params", fnType.NumIn())}
			return
		}

		var result any
		var execErr error
		if len(results) > 0 && results[0].IsValid() {
			result = results[0].Interface()
		}
		if len(results) > 1 && results[1].IsValid() && !results[1].IsNil() {
			execErr = results[1].Interface().(error)
		}
		done <- execResult{result, execErr}
	}()

	select {
	case <-ctx.Done():
		v.cancelled = true
		return nil, fmt.Errorf("yaegi script timeout: %w", ctx.Err())
	case res := <-done:
		return res.value, res.err
	}
}

func (v *YaegiVM) Interrupt(reason string) {
	v.cancelled = true
}

func (v *YaegiVM) Close() {}
