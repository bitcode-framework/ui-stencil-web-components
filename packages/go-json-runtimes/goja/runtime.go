package goja

import (
	runtimes "github.com/bitcode-framework/go-json-runtimes"
	"github.com/dop251/goja"
)

type GojaRuntime struct{}

func New() *GojaRuntime {
	return &GojaRuntime{}
}

func (r *GojaRuntime) Name() string { return "goja" }

func (r *GojaRuntime) NewVM(opts runtimes.VMOptions) (runtimes.VM, error) {
	rt := goja.New()
	rt.SetFieldNameMapper(goja.TagFieldNameMapper("json", true))
	return &GojaVM{rt: rt, opts: opts}, nil
}
