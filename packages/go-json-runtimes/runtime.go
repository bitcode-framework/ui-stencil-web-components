package runtimes

import (
	"context"
	"time"
)

// EmbeddedRuntime represents an embeddable script engine (goja, quickjs, yaegi).
type EmbeddedRuntime interface {
	Name() string
	NewVM(opts VMOptions) (VM, error)
}

// VM represents a single execution instance of an embedded runtime.
type VM interface {
	InjectBridge(bridge map[string]any) error
	InjectParams(params map[string]any) error
	Execute(code string, filename string) (any, error)
	Interrupt(reason string)
	Close()
}

// VMOptions configures a new VM instance.
type VMOptions struct {
	Timeout      time.Duration
	MaxMemoryMB  int
	HardMaxMemMB int
}

// ExternalRuntime represents a child-process script engine (Node.js, Python).
type ExternalRuntime interface {
	Name() string
	Execute(ctx context.Context, script string, function string, params map[string]any, bridge map[string]any) (any, error)
	Validate() error
	Close() error
}
