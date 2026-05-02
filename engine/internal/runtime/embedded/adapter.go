package embedded

import (
	runtimes "github.com/bitcode-framework/go-json-runtimes"
	"github.com/bitcode-framework/bitcode/internal/runtime/bridge"
)

// VMAdapter wraps a go-json-runtimes VM to accept *bridge.Context.
// This is the bridge between BitCode's typed world and go-json-runtimes' generic world.
type VMAdapter struct {
	vm runtimes.VM
}

func NewVMAdapter(vm runtimes.VM) *VMAdapter {
	return &VMAdapter{vm: vm}
}

func (a *VMAdapter) InjectBridge(bc *bridge.Context) error {
	bridgeMap := BuildBridgeMap(bc)
	return a.vm.InjectBridge(bridgeMap)
}

func (a *VMAdapter) InjectParams(params map[string]any) error {
	return a.vm.InjectParams(params)
}

func (a *VMAdapter) Execute(code string, filename string) (any, error) {
	return a.vm.Execute(code, filename)
}

func (a *VMAdapter) Interrupt(reason string) {
	a.vm.Interrupt(reason)
}

func (a *VMAdapter) Close() {
	a.vm.Close()
}

// BuildBridgeMap converts *bridge.Context to map[string]any.
// This delegates to the existing gojson_adapter's BuildGoJSONExtension.
func BuildBridgeMap(bc *bridge.Context) map[string]any {
	ext := bridge.BuildGoJSONExtension(bc)
	return ext.Functions
}
