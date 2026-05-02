package quickjs

import (
	"fmt"

	runtimes "github.com/bitcode-framework/go-json-runtimes"
	"github.com/fastschema/qjs"
)

type QuickJSVM struct {
	rt   *qjs.Runtime
	opts runtimes.VMOptions
}

func (v *QuickJSVM) InjectBridge(bridge map[string]any) error {
	ctx := v.rt.Context()

	bridgeVal, err := qjs.ToJsValue(ctx, bridge)
	if err != nil {
		return fmt.Errorf("failed to convert bridge to JS value: %w", err)
	}
	ctx.Global().SetPropertyStr("bitcode", bridgeVal)

	ctx.SetFunc("__bc_log", func(this *qjs.This) (*qjs.Value, error) {
		args := this.Args()
		level := args[0].String()
		msg := args[1].String()
		if logFn, ok := bridge["log"]; ok {
			if callable, ok := logFn.(func(...any) (any, error)); ok {
				callable(level, msg)
			}
		}
		return ctx.NewUndefined(), nil
	})

	_, err = ctx.Eval("__console_init__", qjs.Code(consoleInitJS))
	if err != nil {
		return fmt.Errorf("failed to init console: %w", err)
	}

	return nil
}

func (v *QuickJSVM) InjectParams(params map[string]any) error {
	ctx := v.rt.Context()
	paramsVal, err := qjs.ToJsValue(ctx, params)
	if err != nil {
		return fmt.Errorf("failed to inject params: %w", err)
	}
	ctx.Global().SetPropertyStr("params", paramsVal)
	return nil
}

func (v *QuickJSVM) Execute(code string, filename string) (any, error) {
	result, err := v.rt.Eval(filename, qjs.Code(code))
	if err != nil {
		return nil, fmt.Errorf("script error: %w", err)
	}

	if result == nil || result.IsUndefined() || result.IsNull() {
		return nil, nil
	}

	if result.IsObject() {
		execProp := result.GetPropertyStr("execute")
		if execProp != nil && !execProp.IsUndefined() && execProp.IsFunction() {
			ctx := v.rt.Context()
			bitcodeVal := ctx.Global().GetPropertyStr("bitcode")
			paramsVal := ctx.Global().GetPropertyStr("params")
			callResult := result.Call("execute", bitcodeVal.Raw(), paramsVal.Raw())
			if callResult == nil {
				return nil, nil
			}
			return v.exportValue(callResult)
		}
	}

	return v.exportValue(result)
}

func (v *QuickJSVM) Interrupt(reason string) {
	v.rt.Close()
}

func (v *QuickJSVM) Close() {
	defer func() { recover() }()
	v.rt.Close()
}

func (v *QuickJSVM) exportValue(val *qjs.Value) (any, error) {
	if val == nil || val.IsUndefined() || val.IsNull() {
		return nil, nil
	}
	goVal, err := qjs.ToGoValue[any](val)
	if err != nil {
		return nil, fmt.Errorf("failed to convert JS value: %w", err)
	}
	return goVal, nil
}

const consoleInitJS = `
const console = {
  log: (...args) => typeof __bc_log !== 'undefined' ? __bc_log('info', args.join(' ')) : undefined,
  warn: (...args) => typeof __bc_log !== 'undefined' ? __bc_log('warn', args.join(' ')) : undefined,
  error: (...args) => typeof __bc_log !== 'undefined' ? __bc_log('error', args.join(' ')) : undefined,
  debug: (...args) => typeof __bc_log !== 'undefined' ? __bc_log('debug', args.join(' ')) : undefined,
};
`
