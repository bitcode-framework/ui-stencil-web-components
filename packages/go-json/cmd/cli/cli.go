package cli

import (
	"github.com/bitcode-framework/go-json/runtime"
)

var extraRuntimeOpts []runtime.Option

// CLIOption configures the go-json CLI.
type CLIOption func()

// WithScriptRuntime registers a script runtime for use with script: imports in CLI commands.
func WithScriptRuntime(rt runtime.ScriptRuntime) CLIOption {
	return func() {
		extraRuntimeOpts = append(extraRuntimeOpts, runtime.WithScriptRuntime(rt))
	}
}

// WithScriptBridge sets the bridge map for script runtimes in CLI commands.
func WithScriptBridge(bridge map[string]any) CLIOption {
	return func() {
		extraRuntimeOpts = append(extraRuntimeOpts, runtime.WithScriptBridge(bridge))
	}
}

// ExtraRuntimeOpts returns the accumulated runtime options from CLI configuration.
// Used by command implementations to include script runtimes in the runtime builder.
func ExtraRuntimeOpts() []runtime.Option {
	return extraRuntimeOpts
}

// ApplyOptions applies CLI options, accumulating runtime options.
func ApplyOptions(opts ...CLIOption) {
	extraRuntimeOpts = nil
	for _, opt := range opts {
		opt()
	}
}
