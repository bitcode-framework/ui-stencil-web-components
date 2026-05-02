package runtimes

type RuntimeOption func(*runtimeSet)

func WithGoja(opts ...GojaConfig) RuntimeOption {
	return func(rs *runtimeSet) {
		cfg := GojaConfig{Enabled: true}
		if len(opts) > 0 {
			cfg = opts[0]
		}
		rs.gojaConfig = &cfg
	}
}

func WithQuickJS(opts ...QuickJSConfig) RuntimeOption {
	return func(rs *runtimeSet) {
		cfg := QuickJSConfig{Enabled: true}
		if len(opts) > 0 {
			cfg = opts[0]
		}
		rs.quickjsConfig = &cfg
	}
}

func WithYaegi(opts ...YaegiConfig) RuntimeOption {
	return func(rs *runtimeSet) {
		cfg := YaegiConfig{Enabled: true}
		if len(opts) > 0 {
			cfg = opts[0]
		}
		rs.yaegiConfig = &cfg
	}
}

func WithNode(opts ...NodeConfig) RuntimeOption {
	return func(rs *runtimeSet) {
		cfg := NodeConfig{Enabled: "auto", Command: "node", MinVersion: "20.0"}
		if len(opts) > 0 {
			cfg = opts[0]
		}
		rs.nodeConfig = &cfg
	}
}

func WithPython(opts ...PythonConfig) RuntimeOption {
	return func(rs *runtimeSet) {
		cfg := PythonConfig{Enabled: "auto", Command: "python3", MinVersion: "3.10.0"}
		if len(opts) > 0 {
			cfg = opts[0]
		}
		rs.pythonConfig = &cfg
	}
}

type runtimeSet struct {
	gojaConfig    *GojaConfig
	quickjsConfig *QuickJSConfig
	yaegiConfig   *YaegiConfig
	nodeConfig    *NodeConfig
	pythonConfig  *PythonConfig
}
