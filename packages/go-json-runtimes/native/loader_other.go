//go:build !linux && !darwin

package native

import "fmt"

func (n *Runtime) loadPlugin(absPath string) (*loadedPlugin, error) {
	return nil, fmt.Errorf("native plugins not supported on this platform (use wasm: instead)")
}
