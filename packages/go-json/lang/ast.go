package lang

// NodeMeta holds metadata attached to any AST node.
// Comments are preserved for visual editor tooling.
type NodeMeta struct {
	// Comment is a single-line semantic comment (from "_c": "string").
	Comment string
	// Comments is a multi-line semantic comment (from "_c": ["line1", "line2"]).
	Comments []string
	// StepIndex is the 0-based position of this step within its parent steps array.
	StepIndex int
}

// Node is the interface implemented by all AST nodes.
type Node interface {
	nodeType() string
	Meta() *NodeMeta
}

// --------------------------------------------------------------------
// Program — top-level AST node
// --------------------------------------------------------------------

// ElifBlock represents a single elif branch.
type ElifBlock struct {
	Condition string
	Then      []Node
}

// InputField describes a single input parameter's type.
type InputField struct {
	Name string
	Type string
}

// LimitsDef holds program-level resource limit declarations.
type LimitsDef struct {
	MaxDepth          *int    `json:"max_depth,omitempty"`
	MaxSteps          *int    `json:"max_steps,omitempty"`
	MaxLoopIterations *int    `json:"max_loop_iterations,omitempty"`
	MaxNodes          *int    `json:"max_nodes,omitempty"`
	MaxVariables      *int    `json:"max_variables,omitempty"`
	MaxVariableSize   *int    `json:"max_variable_size,omitempty"`
	MaxOutputSize     *int    `json:"max_output_size,omitempty"`
	Timeout           *string `json:"timeout,omitempty"`
}

// FuncParam describes a single function parameter.
// Order matters — params are positional when called from expressions.
type FuncParam struct {
	Name       string
	Type       string
	Default    any
	HasDefault bool
}

// FuncDef defines a user-declared function.
type FuncDef struct {
	Name    string
	Params  []FuncParam // ordered — insertion order from JSON keys
	Returns string
	Steps   []Node
}

// StructDef defines a user-declared struct type.
type StructDef struct {
	NodeMeta
	Name    string
	Frozen  bool                   `json:"frozen,omitempty"`
	Alias   string                 `json:"alias,omitempty"`
	Fields  map[string]*FieldDef   `json:"fields"`
	Methods map[string]*MethodDef  `json:"methods,omitempty"`
}

// FieldDef describes a single struct field.
type FieldDef struct {
	Type       string // "string", "int", "Person", "[]string", "?Address"
	Default    any    // nil if required
	HasDefault bool
}

// MethodDef defines a method on a struct.
// self is implicit — NOT declared in Params.
type MethodDef struct {
	NodeMeta
	Name    string
	Params  []FuncParam    `json:"params,omitempty"`
	Returns string         `json:"returns,omitempty"`
	Steps   []Node         `json:"steps"`
}

// NewConstruction represents a nested struct construction inside a with block.
type NewConstruction struct {
	StructName string
	With       map[string]any
}

// ImportDef describes a single import declaration.
type ImportDef struct {
	Alias    string // namespace alias
	Path     string // raw path string
	PathType string // "relative", "stdlib", "ext", "io", "script", "wasm", "plugin"
}

// ParallelNode represents a parallel execution step.
type ParallelNode struct {
	NodeMeta
	Branches map[string][]Node `json:"parallel"`
	Join     string            `json:"join,omitempty"`     // "all" (default), "any", "settled"
	OnError  string            `json:"on_error,omitempty"` // "cancel_all" (default), "continue", "collect"
	Into     string            `json:"into,omitempty"`
}

func (n *ParallelNode) nodeType() string { return "parallel" }
func (n *ParallelNode) Meta() *NodeMeta  { return &n.NodeMeta }

// ServerConfig holds server-level configuration parsed from the "server" key.
type ServerConfig struct {
	Framework        string            `json:"framework"`
	Port             int               `json:"port"`
	Host             string            `json:"host"`
	Static           any               `json:"static"`           // string or StaticConfig
	Templates        string            `json:"templates"`
	CORS             *CORSConfig       `json:"cors,omitempty"`
	JWT              *JWTConfig        `json:"jwt,omitempty"`
	Auth             *AuthConfig       `json:"auth,omitempty"`
	RateLimit        *RateLimitConfig  `json:"rate_limit,omitempty"`
	GracefulShutdown string            `json:"graceful_shutdown"`
	ReadTimeout      string            `json:"read_timeout"`
	WriteTimeout     string            `json:"write_timeout"`
	MaxBodySize      string            `json:"max_body_size"`
	ErrorTemplates   map[string]string `json:"error_templates,omitempty"`
}

// StaticConfig holds custom static file serving configuration.
type StaticConfig struct {
	Dir    string `json:"dir"`
	Prefix string `json:"prefix"`
}

// CORSConfig holds CORS middleware configuration.
type CORSConfig struct {
	Origins []string `json:"origins"`
	Methods []string `json:"methods"`
	Headers []string `json:"headers"`
	MaxAge  int      `json:"max_age"`
}

// JWTConfig holds JWT authentication configuration.
type JWTConfig struct {
	SecretEnv string            `json:"secret_env"`
	Algorithm string            `json:"algorithm"`
	Expiry    string            `json:"expiry"`
	Cookie    string            `json:"cookie"`
	Header    string            `json:"header"`
	Prefix    string            `json:"prefix"`
	Claims    map[string]string `json:"claims,omitempty"`
}

// AuthConfig holds the plugable auth system configuration.
type AuthConfig struct {
	Default    string                       `json:"default"`
	Strategies map[string]*StrategyConfig   `json:"strategies,omitempty"`
}

// StrategyConfig holds configuration for a single auth strategy.
type StrategyConfig struct {
	Type       string `json:"type"`
	SecretEnv  string `json:"secret_env,omitempty"`
	Algorithm  string `json:"algorithm,omitempty"`
	Expiry     string `json:"expiry,omitempty"`
	Cookie     string `json:"cookie,omitempty"`
	Header     string `json:"header,omitempty"`
	Prefix     string `json:"prefix,omitempty"`
	QueryParam string `json:"query_param,omitempty"`
	KeysEnv    string `json:"keys_env,omitempty"`
	UsersEnv   string `json:"users_env,omitempty"`
	Realm      string `json:"realm,omitempty"`
	Handler    string `json:"handler,omitempty"`
	Claims     map[string]string `json:"claims,omitempty"`
}

// RateLimitConfig holds rate limiting configuration.
type RateLimitConfig struct {
	Requests int    `json:"requests"`
	Window   string `json:"window"`
	By       string `json:"by"`
}

// RouteConfig represents a single route or route group definition.
type RouteConfig struct {
	Method     string         `json:"method,omitempty"`
	Path       string         `json:"path,omitempty"`
	Handler    string         `json:"handler,omitempty"`
	Middleware []string       `json:"middleware,omitempty"`
	Render     string         `json:"render,omitempty"`
	Prefix     string         `json:"prefix,omitempty"`
	Routes     []RouteConfig  `json:"routes,omitempty"`
	API        *APIAnnotation `json:"api,omitempty"`
}

// APIAnnotation holds OpenAPI annotation for a route.
type APIAnnotation struct {
	Summary     string                       `json:"summary,omitempty"`
	Description string                       `json:"description,omitempty"`
	Tags        []string                     `json:"tags,omitempty"`
	Body        *APIBodyAnnotation           `json:"body,omitempty"`
	Query       map[string]*APIParamAnnotation `json:"query,omitempty"`
	Responses   map[string]*APIResponseAnnotation `json:"responses,omitempty"`
}

// APIBodyAnnotation describes the request body schema for OpenAPI.
type APIBodyAnnotation struct {
	Required bool                          `json:"required,omitempty"`
	Content  map[string]*APIFieldAnnotation `json:"content,omitempty"`
}

// APIFieldAnnotation describes a single field in an API schema.
type APIFieldAnnotation struct {
	Type        string   `json:"type,omitempty"`
	Required    bool     `json:"required,omitempty"`
	Description string   `json:"description,omitempty"`
	Format      string   `json:"format,omitempty"`
	Enum        []string `json:"enum,omitempty"`
	Default     any      `json:"default,omitempty"`
}

// APIParamAnnotation describes a query parameter for OpenAPI.
type APIParamAnnotation struct {
	Type        string `json:"type,omitempty"`
	Description string `json:"description,omitempty"`
	Default     any    `json:"default,omitempty"`
}

// APIResponseAnnotation describes a response for OpenAPI.
type APIResponseAnnotation struct {
	Description string                         `json:"description,omitempty"`
	Content     map[string]*APIFieldAnnotation `json:"content,omitempty"`
}

// Program is the root AST node representing an entire go-json program.
type Program struct {
	NodeMeta
	Name      string
	GoJSON    string // language version (e.g. "1")
	Input     []InputField
	Imports   []*ImportDef
	Structs   map[string]*StructDef
	Functions map[string]*FuncDef
	Steps     []Node
	Limits    *LimitsDef
	Constants map[string]any
	Enums     map[string]any

	// Server mode fields
	Server     *ServerConfig  `json:"server,omitempty"`
	Routes     []RouteConfig  `json:"routes,omitempty"`
	Middleware []string       `json:"middleware,omitempty"` // global middleware names

	RequestedModules map[string]ImportDef // io:, ext:, and script: imports for runtime validation

	SourcePath string // absolute path of the source file (set by CompileFile for relative import resolution)
}

func (n *Program) nodeType() string { return "program" }
func (n *Program) Meta() *NodeMeta  { return &n.NodeMeta }

// --------------------------------------------------------------------
// Step nodes — one struct per step type
// --------------------------------------------------------------------

// LetNode declares a new variable.
// Exactly one of Value, Expr, With must be set (enforced by parser).
// Call/New are shorthands: {"let": "x", "call": "fn", "with": {...}}
type LetNode struct {
	NodeMeta
	Name     string
	Type     string // optional explicit type annotation
	Value    any    // literal value mode
	Expr     string // expression mode
	With     map[string]string // computed object mode
	HasValue bool   // distinguishes nil literal from "not set"
	HasExpr  bool
	HasWith  bool
	Call         string
	CallWith     map[string]string
	CallWithArgs []string
	CallArgs     []any
	New     string
	NewWith map[string]any
}

func (n *LetNode) nodeType() string { return "let" }
func (n *LetNode) Meta() *NodeMeta  { return &n.NodeMeta }

// SetNode updates an existing variable.
// Target supports dot-path notation: "person.address.city", "items[0].name"
type SetNode struct {
	NodeMeta
	Target   string // variable name or dot-path
	Value    any
	Expr     string
	With     map[string]string
	HasValue bool
	HasExpr  bool
	HasWith  bool
}

func (n *SetNode) nodeType() string { return "set" }
func (n *SetNode) Meta() *NodeMeta  { return &n.NodeMeta }

// IfNode represents an if/elif/else control flow step.
type IfNode struct {
	NodeMeta
	Condition string
	Then      []Node
	Elif      []ElifBlock
	Else      []Node
}

func (n *IfNode) nodeType() string { return "if" }
func (n *IfNode) Meta() *NodeMeta  { return &n.NodeMeta }

// SwitchNode represents a switch/case control flow step.
// Cases maps string keys to step arrays. "default" is the fallback case.
type SwitchNode struct {
	NodeMeta
	Expr  string
	Cases map[string][]Node // key → steps; "default" is special
}

func (n *SwitchNode) nodeType() string { return "switch" }
func (n *SwitchNode) Meta() *NodeMeta  { return &n.NodeMeta }

// ForNode represents a for-each or range loop.
// For-each: Variable + In + optional Index
// Range:    Variable + Range ([start, end] or [start, end, step])
type ForNode struct {
	NodeMeta
	Variable string
	In       string  // expression evaluating to array (for-each mode)
	Range    []any   // [start, end] or [start, end, step] (range mode)
	Index    string  // optional index variable name
	Steps    []Node
}

func (n *ForNode) nodeType() string { return "for" }
func (n *ForNode) Meta() *NodeMeta  { return &n.NodeMeta }

// WhileNode represents a while loop.
type WhileNode struct {
	NodeMeta
	Condition string
	Steps     []Node
}

func (n *WhileNode) nodeType() string { return "while" }
func (n *WhileNode) Meta() *NodeMeta  { return &n.NodeMeta }

// BreakNode exits the innermost loop.
type BreakNode struct {
	NodeMeta
}

func (n *BreakNode) nodeType() string { return "break" }
func (n *BreakNode) Meta() *NodeMeta  { return &n.NodeMeta }

// ContinueNode skips to the next iteration of the innermost loop.
type ContinueNode struct {
	NodeMeta
}

func (n *ContinueNode) nodeType() string { return "continue" }
func (n *ContinueNode) Meta() *NodeMeta  { return &n.NodeMeta }

// ReturnNode returns a value from a function or program.
// Overloaded: string → expression, map → value/expr/with modes.
type ReturnNode struct {
	NodeMeta
	Expr     string
	Value    any
	With     map[string]string
	New      string
	NewWith  map[string]any
	HasExpr  bool
	HasValue bool
	HasWith  bool
	HasNew   bool
}

func (n *ReturnNode) nodeType() string { return "return" }
func (n *ReturnNode) Meta() *NodeMeta  { return &n.NodeMeta }

// CallNode calls a function at step level (without storing result).
// Supports three argument modes (mutually exclusive):
//   - With: named expression args (object with) — {"call": "fn", "with": {"a": "expr"}}
//   - WithArgs: positional expression args (array with) — {"call": "fn", "with": ["expr1", "expr2"]}
//   - Args: positional literal args (args key) — {"call": "fn", "args": [1, "hello", true]}
type CallNode struct {
	NodeMeta
	Function string
	With     map[string]string // named expression args (object with)
	WithArgs []string          // positional expression args (array with)
	Args     []any             // positional literal args (args key)
}

func (n *CallNode) nodeType() string { return "call" }
func (n *CallNode) Meta() *NodeMeta  { return &n.NodeMeta }

// CatchBlock defines the catch clause of a try/catch.
type CatchBlock struct {
	As    string // variable name for the error object
	Steps []Node
}

// TryNode represents try/catch/finally error handling.
type TryNode struct {
	NodeMeta
	Try     []Node
	Catch   *CatchBlock
	Finally []Node
}

func (n *TryNode) nodeType() string { return "try" }
func (n *TryNode) Meta() *NodeMeta  { return &n.NodeMeta }

// ErrorNode throws an error.
// Overloaded: string → message expression, map → structured error.
type ErrorNode struct {
	NodeMeta
	// Simple mode: message is an expression string
	Message string
	// Structured mode: code, message, details are all expression strings
	Code    string
	Details string
	IsStructured bool
}

func (n *ErrorNode) nodeType() string { return "error" }
func (n *ErrorNode) Meta() *NodeMeta  { return &n.NodeMeta }

// LogNode emits a log entry.
// Overloaded: string → simple message, map → structured log.
type LogNode struct {
	NodeMeta
	// Simple mode: message expression
	Message string
	// Structured mode
	Level string            // expression for log level
	Data  map[string]string // each value is an expression
	IsStructured bool
}

func (n *LogNode) nodeType() string { return "log" }
func (n *LogNode) Meta() *NodeMeta  { return &n.NodeMeta }

// MatchNode performs structural pattern matching with variable binding and guards.
type MatchNode struct {
	NodeMeta
	Subject string
	Cases   []MatchCase
}

// MatchCase is a single case in a match step.
type MatchCase struct {
	Pattern any
	Guard   string
	Steps   []Node
}

func (n *MatchNode) nodeType() string { return "match" }
func (n *MatchNode) Meta() *NodeMeta  { return &n.NodeMeta }

// SleepNode pauses execution for a specified duration.
type SleepNode struct {
	NodeMeta
	Duration any // int (literal ms) or string (expression)
}

func (n *SleepNode) nodeType() string { return "sleep" }
func (n *SleepNode) Meta() *NodeMeta  { return &n.NodeMeta }

// RetryNode retries a block of steps with configurable backoff.
type RetryNode struct {
	NodeMeta
	Steps   []Node
	Max     int
	Delay   int
	Backoff string // "fixed", "linear", "exponential"
}

func (n *RetryNode) nodeType() string { return "retry" }
func (n *RetryNode) Meta() *NodeMeta  { return &n.NodeMeta }

// AssertNode validates a condition and throws ASSERTION_FAILED if false.
type AssertNode struct {
	NodeMeta
	Condition string
	Message   string
}

func (n *AssertNode) nodeType() string { return "assert" }
func (n *AssertNode) Meta() *NodeMeta  { return &n.NodeMeta }

// CommentNode is a standalone comment step (only "_c", no other keys).
// It is preserved in the AST but not executed.
type CommentNode struct {
	NodeMeta
}

func (n *CommentNode) nodeType() string { return "comment" }
func (n *CommentNode) Meta() *NodeMeta  { return &n.NodeMeta }
