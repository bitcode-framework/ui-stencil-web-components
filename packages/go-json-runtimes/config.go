package runtimes

import "time"

type GojaConfig struct {
	Enabled bool
}

type QuickJSConfig struct {
	Enabled bool
}

type YaegiConfig struct {
	Enabled      bool
	BridgesDir   string
	StdlibFilter []string
}

type NodeConfig struct {
	Enabled    string
	Command    string
	MinVersion string
	Pool       *DualPoolRef
	ModulePath string
}

type PythonConfig struct {
	Enabled    string
	Command    string
	MinVersion string
	Pool       *DualPoolRef
	VenvPath   string
}

type DualPoolRef struct {
	Worker     PoolWithTimeoutConfig
	Background PoolWithTimeoutConfig
}

type PoolWithTimeoutConfig struct {
	Pool    PoolConfig
	Timeout TimeoutConfig
}

type PoolConfig struct {
	Size            int
	MaxExecutions   int
	MaxMemoryMB     int
	HardMaxMemoryMB int
	MaxIdleTime     time.Duration
	CrashRecovery   bool
	MaxBackoff      time.Duration
}

// TimeoutConfig holds timeout settings enforced by the CALLER via context.Context.
// go-json-runtimes stores these for introspection but does NOT enforce them.
type TimeoutConfig struct {
	DefaultStepTimeout time.Duration
	MaxStepTimeout     time.Duration
	MaxProcessTimeout  time.Duration
}

type ExecOptions struct {
	MaxMemoryMB int
	Pool        string
}
