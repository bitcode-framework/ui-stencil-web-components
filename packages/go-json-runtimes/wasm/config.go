package wasm

import "time"

// Config holds WASM runtime configuration.
type Config struct {
	MaxMemoryMB  int
	MaxExecTime  time.Duration
	CompileCache bool
	WASIEnabled  bool
	PoolSize     int // per-module instance pool size (0 = no pooling, create per-call)
}

// DefaultConfig returns sensible defaults for WASM runtime.
func DefaultConfig() Config {
	return Config{
		MaxMemoryMB:  64,
		MaxExecTime:  30 * time.Second,
		CompileCache: true,
		WASIEnabled:  false,
	}
}
