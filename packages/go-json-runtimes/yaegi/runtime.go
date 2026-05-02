package yaegi

import (
	runtimes "github.com/bitcode-framework/go-json-runtimes"
	"github.com/traefik/yaegi/interp"
	"github.com/traefik/yaegi/stdlib"
)

type YaegiRuntime struct {
	config runtimes.YaegiConfig
}

func New(opts ...runtimes.YaegiConfig) *YaegiRuntime {
	cfg := runtimes.YaegiConfig{Enabled: true}
	if len(opts) > 0 {
		cfg = opts[0]
	}
	return &YaegiRuntime{config: cfg}
}

func (r *YaegiRuntime) Name() string { return "yaegi" }

func (r *YaegiRuntime) NewVM(opts runtimes.VMOptions) (runtimes.VM, error) {
	i := interp.New(interp.Options{})
	i.Use(stdlib.Symbols)
	return &YaegiVM{
		interp: i,
		opts:   opts,
		config: r.config,
	}, nil
}
