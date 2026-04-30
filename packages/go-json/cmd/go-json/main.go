package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	cg "github.com/bitcode-framework/go-json/codegen"
	"github.com/bitcode-framework/go-json/generate"
	goio "github.com/bitcode-framework/go-json/io"
	"github.com/bitcode-framework/go-json/lang"
	"github.com/bitcode-framework/go-json/runtime"
	"github.com/bitcode-framework/go-json/server"
	"github.com/bitcode-framework/go-json/stdlib"
)

const version = "0.1.0"

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	switch os.Args[1] {
	case "run":
		cmdRun(os.Args[2:])
	case "serve":
		cmdServe(os.Args[2:])
	case "check":
		cmdCheck(os.Args[2:])
	case "test":
		cmdTest(os.Args[2:])
	case "ast":
		cmdAST(os.Args[2:])
	case "codegen":
		cmdCodegen(os.Args[2:])
	case "generate":
		cmdGenerate(os.Args[2:])
	case "openapi":
		cmdOpenAPI(os.Args[2:])
	case "migrate":
		cmdMigrate(os.Args[2:])
	case "--version", "-v", "version":
		fmt.Printf("go-json %s\n", version)
	case "--help", "-h", "help":
		printUsage()
	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n\n", os.Args[1])
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println(`go-json — JSON/JSONC programming language engine

Usage: go-json <command> [options]

Commands:
  run       Execute a go-json program
  serve     Start a web server from a server program
  check     Validate a program (compile check, no execution)
  test      Run test files
  ast       Export program AST as JSON
  codegen   Generate Go/JS/Python code from program
  generate  Scaffold CRUD, auth, or project from templates
  openapi   Generate OpenAPI spec from server program
  migrate   Migrate deprecated syntax

Flags:
  --version   Print version
  --help      Print this help`)
}

func cmdRun(args []string) {
	fs := flag.NewFlagSet("run", flag.ExitOnError)
	inputJSON := fs.String("input", "", "Inline JSON input")
	inputFile := fs.String("input-file", "", "Read input from file")
	timeout := fs.String("timeout", "30s", "Execution timeout")
	maxDepth := fs.Int("max-depth", 0, "Override default call depth limit")
	ioModules := fs.String("io", "", "Enable I/O modules (http,fs,sql,exec or 'all')")
	trace := fs.Bool("trace", false, "Print execution trace")
	fs.Parse(args)

	if fs.NArg() < 1 {
		fmt.Fprintln(os.Stderr, "Usage: go-json run <program.json> [options]")
		os.Exit(1)
	}

	programPath := fs.Arg(0)

	if *inputJSON != "" && *inputFile != "" {
		fmt.Fprintln(os.Stderr, "Error: cannot use both --input and --input-file")
		os.Exit(1)
	}

	var input map[string]any
	if *inputJSON != "" {
		if err := json.Unmarshal([]byte(*inputJSON), &input); err != nil {
			fmt.Fprintf(os.Stderr, "Error: invalid JSON input: %s\n", err.Error())
			os.Exit(1)
		}
	} else if *inputFile != "" {
		data, err := os.ReadFile(*inputFile)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: cannot read input file: %s\n", err.Error())
			os.Exit(1)
		}
		if err := json.Unmarshal(data, &input); err != nil {
			fmt.Fprintf(os.Stderr, "Error: invalid JSON in input file: %s\n", err.Error())
			os.Exit(1)
		}
	} else {
		stat, _ := os.Stdin.Stat()
		if stat != nil && (stat.Mode()&os.ModeCharDevice) == 0 {
			data, err := io.ReadAll(os.Stdin)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error reading stdin: %s\n", err.Error())
				os.Exit(1)
			}
			if len(data) > 0 {
				if err := json.Unmarshal(data, &input); err != nil {
					fmt.Fprintf(os.Stderr, "Error: invalid JSON from stdin: %s\n", err.Error())
					os.Exit(1)
				}
			}
		}
	}

	dur, err := time.ParseDuration(*timeout)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: invalid timeout: %s\n", err.Error())
		os.Exit(1)
	}

	reg := stdlib.DefaultRegistry()
	opts := []runtime.Option{
		runtime.WithStdlib(reg.All()),
		runtime.WithStdlibEnv(reg.EnvVars()),
		runtime.WithLimits(runtime.Limits{Timeout: dur}),
		runtime.WithRuntimeTrace(*trace),
	}

	if *maxDepth > 0 {
		opts = append(opts, runtime.WithLimits(runtime.Limits{
			Timeout:  dur,
			MaxDepth: *maxDepth,
		}))
	}

	if *ioModules != "" {
		sec := goio.DefaultSecurityConfig()
		if *ioModules == "all" {
			opts = append(opts, runtime.WithIO(goio.All(sec)...))
		} else {
			for _, mod := range strings.Split(*ioModules, ",") {
				mod = strings.TrimSpace(mod)
				switch mod {
				case "http":
					opts = append(opts, runtime.WithIO(goio.HTTP(sec)))
				case "fs":
					opts = append(opts, runtime.WithIO(goio.FS(sec)))
				case "sql":
					opts = append(opts, runtime.WithIO(goio.SQL(sec)))
				case "exec":
					opts = append(opts, runtime.WithIO(goio.Exec(sec)))
				default:
					fmt.Fprintf(os.Stderr, "Error: unknown I/O module: %s\n", mod)
					os.Exit(1)
				}
			}
		}
	}

	rt := runtime.NewRuntime(opts...)
	defer rt.Close()

	compiled, err := rt.CompileFile(programPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %s\n", err.Error())
		os.Exit(1)
	}

	result, err := rt.Execute(compiled, input)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %s\n", err.Error())
		os.Exit(1)
	}

	if result.Value != nil {
		out, _ := json.MarshalIndent(result.Value, "", "  ")
		fmt.Println(string(out))
	}

	if *trace && result.Trace != nil {
		fmt.Fprintln(os.Stderr, "\n--- Trace ---")
		traceOut, _ := json.MarshalIndent(result.Trace, "", "  ")
		fmt.Fprintln(os.Stderr, string(traceOut))
	}
}

func cmdCheck(args []string) {
	fs := flag.NewFlagSet("check", flag.ExitOnError)
	verbose := fs.Bool("verbose", false, "Show program metadata")
	fs.Parse(args)

	if fs.NArg() < 1 {
		fmt.Fprintln(os.Stderr, "Usage: go-json check <program.json>")
		os.Exit(1)
	}

	programPath := fs.Arg(0)

	reg := stdlib.DefaultRegistry()
	rt := runtime.NewRuntime(
		runtime.WithStdlib(reg.All()),
		runtime.WithStdlibEnv(reg.EnvVars()),
	)

	compiled, err := rt.CompileFile(programPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %s\n", err.Error())
		os.Exit(1)
	}

	fmt.Println("OK")

	if *verbose {
		fmt.Printf("Name: %s\n", compiled.Name)
		if len(compiled.Functions) > 0 {
			names := make([]string, 0, len(compiled.Functions))
			for n := range compiled.Functions {
				names = append(names, n)
			}
			fmt.Printf("Functions: %s\n", strings.Join(names, ", "))
		}
		if len(compiled.Structs) > 0 {
			names := make([]string, 0, len(compiled.Structs))
			for n := range compiled.Structs {
				names = append(names, n)
			}
			fmt.Printf("Structs: %s\n", strings.Join(names, ", "))
		}
		if compiled.AST != nil && len(compiled.AST.Imports) > 0 {
			for _, imp := range compiled.AST.Imports {
				fmt.Printf("Import: %s → %s\n", imp.Alias, imp.Path)
			}
		}
	}
}

func cmdAST(args []string) {
	fs := flag.NewFlagSet("ast", flag.ExitOnError)
	output := fs.String("output", "", "Write to file (default: stdout)")
	format := fs.String("format", "json", "Output format (json)")
	fs.Parse(args)

	if fs.NArg() < 1 {
		fmt.Fprintln(os.Stderr, "Usage: go-json ast <program.json> [--output ast.json]")
		os.Exit(1)
	}

	if *format != "json" {
		fmt.Fprintf(os.Stderr, "Error: unsupported format '%s' (only 'json' supported)\n", *format)
		os.Exit(1)
	}

	programPath := fs.Arg(0)
	data, err := os.ReadFile(programPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %s\n", err.Error())
		os.Exit(1)
	}

	program, err := lang.Parse(data)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %s\n", err.Error())
		os.Exit(1)
	}

	astJSON, err := json.MarshalIndent(program, "", "  ")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %s\n", err.Error())
		os.Exit(1)
	}

	if *output != "" {
		if err := os.WriteFile(*output, astJSON, 0644); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %s\n", err.Error())
			os.Exit(1)
		}
		fmt.Printf("AST written to %s\n", *output)
	} else {
		fmt.Println(string(astJSON))
	}
}

func cmdCodegen(args []string) {
	fs := flag.NewFlagSet("codegen", flag.ExitOnError)
	target := fs.String("target", "", "Target language: go, js, python (required)")
	output := fs.String("output", "", "Write to file (default: stdout)")
	pkg := fs.String("package", "main", "Go package name (only for --target go)")
	fs.Parse(args)

	if fs.NArg() < 1 || *target == "" {
		fmt.Fprintln(os.Stderr, "Usage: go-json codegen <program.json> --target go|js|python [--output file]")
		os.Exit(1)
	}

	programPath := fs.Arg(0)

	reg := stdlib.DefaultRegistry()
	rt := runtime.NewRuntime(
		runtime.WithStdlib(reg.All()),
		runtime.WithStdlibEnv(reg.EnvVars()),
	)
	defer rt.Close()

	compiled, err := rt.CompileFile(programPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %s\n", err.Error())
		os.Exit(1)
	}

	var gen cg.CodeGenerator
	switch *target {
	case "go":
		gen = &cg.GoGenerator{PackageName: *pkg}
	case "js", "javascript":
		gen = &cg.JSGenerator{}
	case "python", "py":
		gen = &cg.PythonGenerator{}
	default:
		fmt.Fprintf(os.Stderr, "Error: unsupported target '%s' (use go, js, or python)\n", *target)
		os.Exit(1)
	}

	code, err := gen.Generate(compiled)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %s\n", err.Error())
		os.Exit(1)
	}

	if *output != "" {
		if err := os.WriteFile(*output, []byte(code), 0644); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %s\n", err.Error())
			os.Exit(1)
		}
		fmt.Printf("Generated %s code written to %s\n", gen.Language(), *output)
	} else {
		fmt.Print(code)
	}
}

func cmdMigrate(args []string) {
	fs := flag.NewFlagSet("migrate", flag.ExitOnError)
	from := fs.String("from", "", "Source version (auto-detect if omitted)")
	to := fs.String("to", "v2", "Target version")
	output := fs.String("output", "", "Write to file (default: stdout)")
	dryRun := fs.Bool("dry-run", false, "Show changes without applying")
	fs.Parse(args)

	if fs.NArg() < 1 {
		fmt.Fprintln(os.Stderr, "Usage: go-json migrate <program.json> [--from v1] [--to v2] [--dry-run]")
		os.Exit(1)
	}

	programPath := fs.Arg(0)
	data, err := os.ReadFile(programPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %s\n", err.Error())
		os.Exit(1)
	}

	migrated, changes := migrateProgram(string(data), *from, *to)

	if len(changes) == 0 {
		fmt.Println("Program is already current — no changes needed")
		return
	}

	if *dryRun {
		fmt.Println("Changes that would be applied:")
		for _, c := range changes {
			fmt.Printf("  - %s\n", c)
		}
		return
	}

	if *output != "" {
		if err := os.WriteFile(*output, []byte(migrated), 0644); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %s\n", err.Error())
			os.Exit(1)
		}
		fmt.Printf("Migrated program written to %s\n", *output)
	} else {
		fmt.Println(migrated)
	}

	fmt.Fprintf(os.Stderr, "Applied %d changes\n", len(changes))
}

func migrateProgram(source, from, to string) (string, []string) {
	var raw any
	if err := json.Unmarshal([]byte(source), &raw); err != nil {
		return source, nil
	}

	renames := map[string]string{
		"unique":     "uniq",
		"startsWith": "hasPrefix",
		"endsWith":   "hasSuffix",
	}

	var changes []string
	migrated := migrateValue(raw, renames, &changes)

	out, err := json.MarshalIndent(migrated, "", "  ")
	if err != nil {
		return source, nil
	}

	return string(out), changes
}

func migrateValue(v any, renames map[string]string, changes *[]string) any {
	switch val := v.(type) {
	case map[string]any:
		result := make(map[string]any, len(val))
		for k, child := range val {
			newKey := k
			if renamed, ok := renames[k]; ok {
				newKey = renamed
				*changes = append(*changes, fmt.Sprintf("renamed key '%s' → '%s'", k, renamed))
			}
			result[newKey] = migrateValue(child, renames, changes)
		}
		return result
	case []any:
		result := make([]any, len(val))
		for i, item := range val {
			result[i] = migrateValue(item, renames, changes)
		}
		return result
	default:
		return v
	}
}

func cmdServe(args []string) {
	fs := flag.NewFlagSet("serve", flag.ExitOnError)
	dev := fs.Bool("dev", false, "Enable dev mode (template reload, verbose logging)")
	port := fs.Int("port", 0, "Override server port")
	host := fs.String("host", "", "Override server host")
	docs := fs.Bool("docs", false, "Enable /docs Swagger UI endpoint")
	ioSpec := fs.String("io", "all", "Enable I/O modules (http,fs,sql,exec or 'all')")
	fs.Parse(args)

	if fs.NArg() < 1 {
		fmt.Fprintln(os.Stderr, "Usage: go-json serve <program.json> [options]")
		os.Exit(1)
	}

	programPath := fs.Arg(0)

	reg := stdlib.DefaultRegistry()
	rtOpts := []runtime.Option{
		runtime.WithStdlib(reg.All()),
		runtime.WithStdlibEnv(reg.EnvVars()),
	}

	sec := goio.DefaultSecurityConfig()
	if *ioSpec == "all" {
		rtOpts = append(rtOpts, runtime.WithIO(goio.All(sec)...))
	} else if *ioSpec != "" {
		for _, mod := range strings.Split(*ioSpec, ",") {
			mod = strings.TrimSpace(mod)
			switch mod {
			case "http":
				rtOpts = append(rtOpts, runtime.WithIO(goio.HTTP(sec)))
			case "fs":
				rtOpts = append(rtOpts, runtime.WithIO(goio.FS(sec)))
			case "sql":
				rtOpts = append(rtOpts, runtime.WithIO(goio.SQL(sec)))
			case "exec":
				rtOpts = append(rtOpts, runtime.WithIO(goio.Exec(sec)))
			default:
				fmt.Fprintf(os.Stderr, "Warning: unknown I/O module: %s\n", mod)
			}
		}
	}

	rt := runtime.NewRuntime(rtOpts...)
	defer rt.Close()

	var serverOpts []server.ServerOption
	if *dev {
		serverOpts = append(serverOpts, server.WithDevMode(true))
	}
	if *docs {
		serverOpts = append(serverOpts, server.WithDocs(true))
	}
	if *port > 0 {
		serverOpts = append(serverOpts, server.WithPort(*port))
	}
	if *host != "" {
		serverOpts = append(serverOpts, server.WithHost(*host))
	}

	srv, err := server.NewServer(programPath, rt, serverOpts...)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	if err := srv.Start(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func cmdOpenAPI(args []string) {
	fs := flag.NewFlagSet("openapi", flag.ExitOnError)
	output := fs.String("output", "", "Output file (default: stdout)")
	fs.Parse(args)

	if fs.NArg() < 1 {
		fmt.Fprintln(os.Stderr, "Usage: go-json openapi <program.json> [--output file.json]")
		os.Exit(1)
	}

	programPath := fs.Arg(0)

	reg := stdlib.DefaultRegistry()
	rt := runtime.NewRuntime(
		runtime.WithStdlib(reg.All()),
		runtime.WithStdlibEnv(reg.EnvVars()),
	)

	compiled, err := rt.CompileFile(programPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	spec := server.GenerateOpenAPISpec(compiled)

	data, err := json.MarshalIndent(spec, "", "  ")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	if *output != "" {
		if err := os.WriteFile(*output, data, 0644); err != nil {
			fmt.Fprintf(os.Stderr, "Error writing file: %v\n", err)
			os.Exit(1)
		}
		fmt.Fprintf(os.Stderr, "OpenAPI spec written to %s\n", *output)
	} else {
		fmt.Println(string(data))
	}
}

func cmdGenerate(args []string) {
	if len(args) < 1 {
		fmt.Fprintln(os.Stderr, "Usage: go-json generate <crud|auth|project> [options]")
		os.Exit(1)
	}

	switch args[0] {
	case "crud":
		cmdGenerateCRUD(args[1:])
	case "auth":
		cmdGenerateAuth(args[1:])
	case "project":
		cmdGenerateProject(args[1:])
	default:
		fmt.Fprintf(os.Stderr, "Unknown generate subcommand: %s\n", args[0])
		os.Exit(1)
	}
}

func cmdGenerateCRUD(args []string) {
	fs := flag.NewFlagSet("generate crud", flag.ExitOnError)
	table := fs.String("table", "", "Table name")
	fields := fs.String("fields", "", "Manual fields (name:type,name:type)")
	dsn := fs.String("dsn", "", "Database DSN for introspection")
	auth := fs.Bool("auth", false, "Add JWT auth middleware to write routes")
	output := fs.String("output", "", "Output file (default: stdout)")
	dryRun := fs.Bool("dry-run", false, "Print without writing")
	fs.Parse(args)

	if *table == "" {
		fmt.Fprintln(os.Stderr, "Error: --table is required")
		os.Exit(1)
	}

	var info *generate.TableInfo
	if *fields != "" {
		info = generate.ParseManualFields(*table, *fields)
	} else if *dsn != "" {
		tables, err := generate.IntrospectDB(*dsn, []string{*table})
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		if len(tables) == 0 {
			fmt.Fprintf(os.Stderr, "Error: table %q not found\n", *table)
			os.Exit(1)
		}
		info = tables[0]
	} else {
		fmt.Fprintln(os.Stderr, "Error: --fields or --dsn is required")
		os.Exit(1)
	}

	result, err := generate.GenerateCRUDJSON(info, generate.CRUDOptions{
		TableName: *table,
		Auth:      *auth,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	if *dryRun || *output == "" {
		fmt.Println(result)
	} else {
		if err := os.WriteFile(*output, []byte(result), 0644); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		fmt.Fprintf(os.Stderr, "Generated CRUD for %s → %s\n", *table, *output)
	}
}

func cmdGenerateAuth(args []string) {
	fs := flag.NewFlagSet("generate auth", flag.ExitOnError)
	output := fs.String("output", "", "Output file (default: stdout)")
	fs.Parse(args)

	result, err := generate.GenerateAuth()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	if *output == "" {
		fmt.Println(result)
		fmt.Fprintln(os.Stderr, "\n-- SQL Migration --")
		fmt.Fprintln(os.Stderr, generate.GenerateAuthSQL())
	} else {
		if err := os.WriteFile(*output, []byte(result), 0644); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		fmt.Fprintf(os.Stderr, "Generated auth scaffold → %s\n", *output)
	}
}

func cmdGenerateProject(args []string) {
	fs := flag.NewFlagSet("generate project", flag.ExitOnError)
	name := fs.String("name", "my-app", "Project name")
	auth := fs.Bool("auth", false, "Include auth scaffold")
	output := fs.String("output", "", "Output directory (default: ./<name>)")
	fs.Parse(args)

	if fs.NArg() > 0 {
		*name = fs.Arg(0)
	}

	dir := *output
	if dir == "" {
		dir = *name
	}

	files := generate.GenerateProject(*name, *auth)

	for path, content := range files {
		fullPath := dir + "/" + path
		dirPath := fullPath[:strings.LastIndex(fullPath, "/")]
		if err := os.MkdirAll(dirPath, 0755); err != nil {
			fmt.Fprintf(os.Stderr, "Error creating directory: %v\n", err)
			os.Exit(1)
		}
		if err := os.WriteFile(fullPath, []byte(content), 0644); err != nil {
			fmt.Fprintf(os.Stderr, "Error writing %s: %v\n", path, err)
			os.Exit(1)
		}
		fmt.Fprintf(os.Stderr, "  created %s\n", path)
	}
	fmt.Fprintf(os.Stderr, "\nProject %q created in %s/\n", *name, dir)
}
