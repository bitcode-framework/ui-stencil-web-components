package main

import (
	"context"

	"github.com/bitcode-framework/go-json/cmd/cli"
	runtimes "github.com/bitcode-framework/go-json-runtimes"
	gojaRT "github.com/bitcode-framework/go-json-runtimes/goja"
	nodeRT "github.com/bitcode-framework/go-json-runtimes/node"
	pythonRT "github.com/bitcode-framework/go-json-runtimes/python"
	quickjsRT "github.com/bitcode-framework/go-json-runtimes/quickjs"
	wasmRT "github.com/bitcode-framework/go-json-runtimes/wasm"
	yaegiRT "github.com/bitcode-framework/go-json-runtimes/yaegi"
)

func main() {
	cli.Run(
		cli.WithScriptRuntime(newEmbeddedAdapter("goja", []string{".js"}, gojaRT.New())),
		cli.WithScriptRuntime(newEmbeddedAdapter("quickjs", []string{".js"}, quickjsRT.New())),
		cli.WithScriptRuntime(newEmbeddedAdapter("yaegi", []string{".go"}, yaegiRT.New())),
		cli.WithScriptRuntime(wasmRT.New(wasmRT.DefaultConfig())),
		cli.WithScriptRuntime(&externalAdapter{rt: nodeRT.NewAuto(), exts: []string{".ts", ".mjs"}}),
		cli.WithScriptRuntime(&externalAdapter{rt: pythonRT.NewAuto(), exts: []string{".py", ".pyw"}}),
	)
}

type embeddedAdapter struct {
	name    string
	exts    []string
	runtime runtimes.EmbeddedRuntime
}

func newEmbeddedAdapter(name string, exts []string, rt runtimes.EmbeddedRuntime) *embeddedAdapter {
	return &embeddedAdapter{name: name, exts: exts, runtime: rt}
}

func (a *embeddedAdapter) Name() string        { return a.name }
func (a *embeddedAdapter) Extensions() []string { return a.exts }
func (a *embeddedAdapter) CanHandle(ext string) bool {
	for _, e := range a.exts {
		if e == ext {
			return true
		}
	}
	return false
}
func (a *embeddedAdapter) Execute(ctx context.Context, script, function string, params, bridge map[string]any) (any, error) {
	return runtimes.ExecuteEmbedded(ctx, a.runtime, script, function, params, bridge, 0)
}
func (a *embeddedAdapter) Validate() error { return nil }
func (a *embeddedAdapter) Close() error    { return nil }

type externalAdapter struct {
	rt   runtimes.ExternalRuntime
	exts []string
}

func (a *externalAdapter) Name() string        { return a.rt.Name() }
func (a *externalAdapter) Extensions() []string { return a.exts }
func (a *externalAdapter) CanHandle(ext string) bool {
	for _, e := range a.exts {
		if e == ext {
			return true
		}
	}
	return false
}
func (a *externalAdapter) Execute(ctx context.Context, script, function string, params, bridge map[string]any) (any, error) {
	return a.rt.Execute(ctx, script, function, params, bridge)
}
func (a *externalAdapter) Validate() error { return a.rt.Validate() }
func (a *externalAdapter) Close() error    { return a.rt.Close() }


