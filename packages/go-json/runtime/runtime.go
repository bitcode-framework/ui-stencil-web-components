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

	extensions *extensionRegistry

	cache   map[string]*lang.CompiledProgram
	cacheMu sync.RWMutex
}

func NewRuntime(opts ...Option) *Runtime {
	r := &Runtime{
		engine:     lang.NewExprLangEngine(),
		limits:     DefaultLimits(),
		cache:      make(map[string]*lang.CompiledProgram),
		ctx:        context.Background(),
		stdlibEnv:  make(map[string]any),
		ioRegistry: goio.NewIORegistry(),
		extensions: newExtensionRegistry(),
	}

	for _, opt := range opts {
		opt(r)
	}

	if len(r.stdlibOpts) > 0 {
		r.engine.AddOptions(r.stdlibOpts...)
	}

	// Register I/O module functions in expr-lang engine.
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

// IORegistry returns the runtime's I/O module registry.
func (r *Runtime) IORegistry() *goio.IORegistry {
	return r.ioRegistry
}

// IODisabled returns whether I/O is explicitly disabled.
func (r *Runtime) IODisabled() bool {
	return r.ioDisabled
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

// ExecuteJSON compiles and executes a program in one call (with caching).
func (r *Runtime) ExecuteJSON(programJSON []byte, input map[string]any) (*lang.ExecutionResult, error) {
	compiled, err := r.Compile(programJSON)
	if err != nil {
		return nil, err
	}
	return r.Execute(compiled, input)
}

func contentHash(data []byte) string {
	h := sha256.Sum256(data)
	return fmt.Sprintf("%x", h[:8])
}
