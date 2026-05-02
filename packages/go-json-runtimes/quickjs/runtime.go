package quickjs

import (
	runtimes "github.com/bitcode-framework/go-json-runtimes"
	"github.com/fastschema/qjs"
)

type QuickJSRuntime struct{}

func New() *QuickJSRuntime {
	return &QuickJSRuntime{}
}

func (r *QuickJSRuntime) Name() string { return "quickjs" }

func (r *QuickJSRuntime) NewVM(opts runtimes.VMOptions) (runtimes.VM, error) {
	rt, err := qjs.New()
	if err != nil {
		return nil, err
	}
	return &QuickJSVM{rt: rt, opts: opts}, nil
}
