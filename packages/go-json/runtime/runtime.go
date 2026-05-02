package runtime

import (
	"context"
	"crypto/sha256"
	"fmt"
	"path/filepath"
	"strings"
	"sync"

	goio "github.com/bitcode-framework/go-json/io"
	"github.com/bitcode-framework/go-json/lang"
	"github.com/bitcode-framework/go-json/stdlib"
	"github.com/expr-lang/expr"
)

type Option func(*Runtime)

func WithStdlib(funcs []expr.Option) Option {
	return func(r *Runtime) { r.stdlibOpts = append(r.stdlibOpts, funcs...) }
}

func WithStdlibEnv(envVars map[string]any) Option {
	return func(r *Runtime) {
		for k, v := range envVars {
			r.stdlibEnv[k] = v
		}
	}
}

func WithLimits(limits Limits) Option {
	return func(r *Runtime) { r.limits = limits }
}

func WithRuntimeLogger(l lang.Logger) Option {
	return func(r *Runtime) { r.logger = l }
}

func WithRuntimeDebugger(d lang.Debugger) Option {
	return func(r *Runtime) { r.debugger = d }
}

func WithRuntimeTrace(enabled bool) Option {
	return func(r *Runtime) { r.traceEnabled = enabled }
}

func WithSession(s *Session) Option {
	return func(r *Runtime) { r.session = s }
}

func WithRuntimeContext(ctx context.Context) Option {
	return func(r *Runtime) { r.ctx = ctx }
}

// WithIO registers I/O modules for use in go-json programs.
// Programs must also import the module via "io:name" to access its functions.
func WithIO(modules ...goio.IOModule) Option {
	return func(r *Runtime) {
		for _, mod := range modules {
			r.ioRegistry.RegisterModule(mod.Name(), mod)
		}
	}
}

// WithoutIO explicitly disables all I/O modules.
// This is the default behavior — calling this is a no-op but documents intent.
func WithoutIO() Option {
	return func(r *Runtime) {
		r.ioRegistry = goio.NewIORegistry()
		r.ioDisabled = true
	}
}

// WithIOSecurity sets the security configuration for I/O modules.
func WithIOSecurity(cfg *goio.SecurityConfig) Option {
	return func(r *Runtime) { r.ioSecurity = cfg }
}

// WithEnvResolver overrides the env() function's resolver after registry construction.
// Requires that the registry was created with RegisterEnvFunc (DefaultRegistry does this).
func WithEnvResolver(resolver stdlib.EnvResolver) Option {
	return func(r *Runtime) { r.envResolver = resolver }
}

// WithEnvAccess overrides the env() function's access control after registry construction.
// Requires that the registry was created with RegisterEnvFunc (DefaultRegistry does this).
func WithEnvAccess(config *stdlib.EnvAccessConfig) Option {
	return func(r *Runtime) { r.envAccess = config }
}

// WithEnvHandle provides the EnvHandle from a Registry, enabling WithEnvResolver/WithEnvAccess
// to wire properly. Called automatically when using the standard pattern:
//
//	reg := stdlib.DefaultRegistry()
//	rt := NewRuntime(WithStdlib(reg.All()), WithStdlibEnv(reg.EnvVars()), WithEnvHandle(reg.EnvHandle()))
func WithEnvHandle(h *stdlib.EnvHandle) Option {
	return func(r *Runtime) { r.envHandle = h }
}

// WithScriptRuntime registers a script runtime for use with script: imports.
// Multiple runtimes can be registered; resolution is by file extension (first match wins).
func WithScriptRuntime(rt ScriptRuntime) Option {
	return func(r *Runtime) {
		r.scriptRuntimes.Register(rt)
	}
}

// WithScriptBridge sets the bridge map passed to all script runtimes.
// This is the map[string]any that scripts receive as their host API.
func WithScriptBridge(bridge map[string]any) Option {
	return func(r *Runtime) {
		r.scriptBridge = bridge
	}
}

type Runtime struct {
	engine       *lang.ExprLangEngine
	limits       Limits
	logger       lang.Logger
	debugger     lang.Debugger
	traceEnabled bool
	session      *Session
	ctx          context.Context
	stdlibOpts   []expr.Option
	stdlibEnv    map[string]any

	ioRegistry *goio.IORegistry
	ioSecurity *goio.SecurityConfig
	ioDisabled bool

	envHandle   *stdlib.EnvHandle
	envResolver stdlib.EnvResolver
	envAccess   *stdlib.EnvAccessConfig

	extensions *extensionRegistry

	scriptRuntimes *ScriptRuntimeRegistry
	scriptBridge   map[string]any

	cache   map[string]*lang.CompiledProgram
	cacheMu sync.RWMutex
}

func NewRuntime(opts ...Option) *Runtime {
	r := &Runtime{
		engine:         lang.NewExprLangEngine(),
		limits:         DefaultLimits(),
		cache:          make(map[string]*lang.CompiledProgram),
		ctx:            context.Background(),
		stdlibEnv:      make(map[string]any),
		ioRegistry:     goio.NewIORegistry(),
		extensions:     newExtensionRegistry(),
		scriptRuntimes: NewScriptRuntimeRegistry(),
	}

	for _, opt := range opts {
		opt(r)
	}

	if r.envHandle != nil {
		if r.envResolver != nil {
			r.envHandle.SetResolver(r.envResolver)
		}
		if r.envAccess != nil {
			r.envHandle.SetAccess(r.envAccess)
		}
	}

	if len(r.stdlibOpts) > 0 {
		r.engine.AddOptions(r.stdlibOpts...)
	}

	if !r.ioDisabled {
		ioOpts := r.ioRegistry.ExprOptions()
		if len(ioOpts) > 0 {
			r.engine.AddOptions(ioOpts...)
		}
		for k, v := range r.ioRegistry.EnvVars() {
			r.stdlibEnv[k] = v
		}
	}

	// Register extension functions in expression environment.
	for name, ext := range r.extensions.all() {
		if ext.Functions != nil {
			r.stdlibEnv[name] = ext.Functions
		}
	}

	return r
}

// Close releases all resources held by I/O modules that implement Close() error.
func (r *Runtime) Close() error {
	var firstErr error
	for _, mod := range r.ioRegistry.AllModules() {
		if closer, ok := mod.(interface{ Close() error }); ok {
			if err := closer.Close(); err != nil && firstErr == nil {
				firstErr = err
			}
		}
	}
	if r.scriptRuntimes != nil {
		if err := r.scriptRuntimes.Close(); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	return firstErr
}

// IORegistry returns the runtime's I/O module registry.
func (r *Runtime) IORegistry() *goio.IORegistry {
	return r.ioRegistry
}

// IODisabled returns whether I/O is explicitly disabled.
func (r *Runtime) IODisabled() bool {
	return r.ioDisabled
}

// AddStdlibEnv adds environment variables to the runtime after construction.
func (r *Runtime) AddStdlibEnv(vars map[string]any) {
	for k, v := range vars {
		r.stdlibEnv[k] = v
	}
}

// Compile parses and compiles a go-json program (no import resolution).
func (r *Runtime) Compile(input []byte) (*lang.CompiledProgram, error) {
	return r.compileWithBasePath(input, "")
}

// CompileFile parses and compiles a go-json program from a file path,
// enabling import resolution relative to the file's directory.
func (r *Runtime) CompileFile(path string) (*lang.CompiledProgram, error) {
	program, err := lang.ParseFile(path)
	if err != nil {
		return nil, err
	}

	absPath, absErr := filepath.Abs(path)
	if absErr == nil {
		program.SourcePath = absPath
	} else {
		program.SourcePath = path
	}

	dir := filepath.Dir(path)
	resolver := lang.NewImportResolver()
	if err := resolver.ResolveImports(program, dir, []string{path}); err != nil {
		return nil, err
	}

	compiled, err := lang.Compile(program, r.engine, r.limits.ToResolved())
	if err != nil {
		return nil, err
	}

	key := path
	r.cacheMu.Lock()
	r.cache[key] = compiled
	r.cacheMu.Unlock()

	return compiled, nil
}

func (r *Runtime) compileWithBasePath(input []byte, basePath string) (*lang.CompiledProgram, error) {
	key := contentHash(input)

	r.cacheMu.RLock()
	if cached, ok := r.cache[key]; ok {
		r.cacheMu.RUnlock()
		return cached, nil
	}
	r.cacheMu.RUnlock()

	program, err := lang.Parse(input)
	if err != nil {
		return nil, err
	}

	if basePath != "" && len(program.Imports) > 0 {
		resolver := lang.NewImportResolver()
		if err := resolver.ResolveImports(program, basePath, nil); err != nil {
			return nil, err
		}
	}

	compiled, err := lang.Compile(program, r.engine, r.limits.ToResolved())
	if err != nil {
		return nil, err
	}

	r.cacheMu.Lock()
	r.cache[key] = compiled
	r.cacheMu.Unlock()

	return compiled, nil
}

// Execute runs a compiled program with the given input.
func (r *Runtime) Execute(program *lang.CompiledProgram, input map[string]any) (*lang.ExecutionResult, error) {
	var vmOpts []lang.VMOption

	ctx := r.ctx
	vmOpts = append(vmOpts, lang.WithContext(ctx))

	if r.logger != nil {
		vmOpts = append(vmOpts, lang.WithLogger(r.logger))
	}
	if r.debugger != nil {
		vmOpts = append(vmOpts, lang.WithDebugger(r.debugger))
	}
	if r.traceEnabled {
		vmOpts = append(vmOpts, lang.WithTrace(true))
	}

	if r.session != nil && input != nil {
		input["session"] = r.session.ToMap()
	}

	if program.AST != nil && program.AST.RequestedModules != nil {
		for alias, imp := range program.AST.RequestedModules {
			switch imp.PathType {
			case "io":
				moduleName := strings.TrimPrefix(imp.Path, "io:")
				if r.ioDisabled {
					return nil, fmt.Errorf("I/O module '%s' requested but I/O is disabled", moduleName)
				}
				mod := r.ioRegistry.GetModule(moduleName)
				if mod == nil {
					return nil, fmt.Errorf("I/O module '%s' not registered (imported as '%s')", moduleName, alias)
				}
				if input == nil {
					input = make(map[string]any)
				}
				input[alias] = mod.Functions()

			case "ext":
				extName := strings.TrimPrefix(imp.Path, "ext:")
				ext := r.extensions.get(extName)
				if ext == nil {
					return nil, fmt.Errorf("extension '%s' not registered (imported as '%s')", extName, alias)
				}
				if input == nil {
					input = make(map[string]any)
				}
				input[alias] = ext.Functions

			case "script":
				proxy, err := r.resolveScriptImport(alias, imp.Path, program)
				if err != nil {
					return nil, err
				}
				if input == nil {
					input = make(map[string]any)
				}
				input[alias] = proxy
			}
		}
	}

	for k, v := range r.stdlibEnv {
		if input == nil {
			input = make(map[string]any)
		}
		if r.ioRegistry.HasModule(k) || r.extensions.get(k) != nil {
			continue
		}
		input[k] = v
	}

	vm := lang.NewVM(program, r.engine, vmOpts...)
	return vm.Execute(input)
}

// ExecuteFunction calls a specific function from a compiled program with named arguments.
// Used by the server handler bridge to invoke route handler functions.
func (r *Runtime) ExecuteFunction(program *lang.CompiledProgram, funcName string, args map[string]any) (any, error) {
	fn, ok := program.Functions[funcName]
	if !ok {
		return nil, fmt.Errorf("function %q not found in program", funcName)
	}

	var vmOpts []lang.VMOption
	vmOpts = append(vmOpts, lang.WithContext(r.ctx))
	if r.logger != nil {
		vmOpts = append(vmOpts, lang.WithLogger(r.logger))
	}
	if r.debugger != nil {
		vmOpts = append(vmOpts, lang.WithDebugger(r.debugger))
	}
	if r.traceEnabled {
		vmOpts = append(vmOpts, lang.WithTrace(true))
	}

	input := make(map[string]any)
	if r.session != nil {
		input["session"] = r.session.ToMap()
	}

	if program.AST != nil && program.AST.RequestedModules != nil {
		for alias, imp := range program.AST.RequestedModules {
			switch imp.PathType {
			case "io":
				moduleName := strings.TrimPrefix(imp.Path, "io:")
				if r.ioDisabled {
					return nil, fmt.Errorf("I/O module '%s' requested but I/O is disabled", moduleName)
				}
				mod := r.ioRegistry.GetModule(moduleName)
				if mod == nil {
					return nil, fmt.Errorf("I/O module '%s' not registered (imported as '%s')", moduleName, alias)
				}
				input[alias] = mod.Functions()
			case "ext":
				extName := strings.TrimPrefix(imp.Path, "ext:")
				ext := r.extensions.get(extName)
				if ext == nil {
					return nil, fmt.Errorf("extension '%s' not registered (imported as '%s')", extName, alias)
				}
				input[alias] = ext.Functions

			case "script":
				proxy, err := r.resolveScriptImport(alias, imp.Path, program)
				if err != nil {
					return nil, err
				}
				input[alias] = proxy
			}
		}
	}

	for k, v := range r.stdlibEnv {
		if r.ioRegistry.HasModule(k) || r.extensions.get(k) != nil {
			continue
		}
		input[k] = v
	}

	vm := lang.NewVM(program, r.engine, vmOpts...)
	return vm.ExecuteFunction(fn, args, input)
}

// ExecuteJSON compiles and executes a program in one call (with caching).
func (r *Runtime) ExecuteJSON(programJSON []byte, input map[string]any) (*lang.ExecutionResult, error) {
	compiled, err := r.Compile(programJSON)
	if err != nil {
		return nil, err
	}
	return r.Execute(compiled, input)
}

func (r *Runtime) resolveScriptImport(alias, path string, program *lang.CompiledProgram) (map[string]any, error) {
	scriptPath := strings.TrimPrefix(path, "script:")
	if filepath.IsAbs(scriptPath) {
		return nil, fmt.Errorf("script: import path must be relative, got: %s (import '%s')", scriptPath, alias)
	}

	programDir := ""
	if program.AST != nil && program.AST.SourcePath != "" {
		programDir = filepath.Dir(program.AST.SourcePath)
	}

	if programDir != "" {
		absPath := filepath.Join(programDir, scriptPath)
		absPath = filepath.Clean(absPath)
		cleanBase := filepath.Clean(programDir) + string(filepath.Separator)
		if absPath != filepath.Clean(programDir) && !strings.HasPrefix(absPath, cleanBase) {
			return nil, fmt.Errorf("script: import path escapes base directory: %s (import '%s')", scriptPath, alias)
		}
		scriptPath = absPath
	}

	ext := filepath.Ext(scriptPath)
	rt := r.scriptRuntimes.Resolve(ext)
	if rt == nil {
		return nil, fmt.Errorf("no script runtime registered for extension '%s' (import '%s')", ext, alias)
	}
	if err := rt.Validate(); err != nil {
		return nil, fmt.Errorf("script runtime '%s' not available: %w", rt.Name(), err)
	}

	return r.buildScriptProxy(rt, scriptPath), nil
}

func (r *Runtime) buildScriptProxy(rt ScriptRuntime, scriptPath string) map[string]any {
	ctx := r.ctx
	bridge := r.scriptBridge
	return map[string]any{
		"call": func(args ...any) (any, error) {
			if len(args) == 0 {
				return nil, fmt.Errorf("script.call requires at least a function name argument")
			}
			function, ok := args[0].(string)
			if !ok {
				return nil, fmt.Errorf("script.call first argument must be a function name (string), got %T", args[0])
			}
			params := map[string]any{"args": args[1:]}
			return rt.Execute(ctx, scriptPath, function, params, bridge)
		},
		"exec": func(args ...any) (any, error) {
			params := map[string]any{"args": args}
			return rt.Execute(ctx, scriptPath, "", params, bridge)
		},
	}
}

func contentHash(data []byte) string {
	h := sha256.Sum256(data)
	return fmt.Sprintf("%x", h[:8])
}
