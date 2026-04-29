package codegen

import (
	"fmt"
	"strings"

	"github.com/bitcode-framework/go-json/lang"
	"github.com/bitcode-framework/go-json/server"
)

func init() {
	RegisterServerCodegen("go", "nethttp", &GoNetHTTPCodegen{})
}

type GoNetHTTPCodegen struct{}

func (g *GoNetHTTPCodegen) Framework() string { return "nethttp" }
func (g *GoNetHTTPCodegen) Language() string  { return "go" }

func (g *GoNetHTTPCodegen) GenerateServer(program *lang.CompiledProgram) (map[string]string, error) {
	files := make(map[string]string)

	cfg := program.AST.Server
	if cfg == nil {
		cfg = &lang.ServerConfig{}
	}
	server.MergeDefaults(cfg)

	flatRoutes := server.FlattenRoutes(program.AST.Routes, "", program.AST.Middleware)

	files["main.go"] = g.generateMain(cfg, flatRoutes)
	files["handlers.go"] = g.generateHandlers(program, flatRoutes)

	return files, nil
}

func (g *GoNetHTTPCodegen) generateMain(cfg *lang.ServerConfig, routes []server.FlatRoute) string {
	var b strings.Builder
	b.WriteString("package main\n\n")
	b.WriteString("import (\n")
	b.WriteString("\t\"log\"\n")
	b.WriteString("\t\"net/http\"\n")
	b.WriteString(")\n\n")

	b.WriteString("func main() {\n")
	b.WriteString("\tmux := http.NewServeMux()\n\n")

	for _, r := range routes {
		pattern := fmt.Sprintf("%s %s", r.Method, r.FullPath)
		b.WriteString(fmt.Sprintf("\tmux.HandleFunc(%q, %s)\n", pattern, handlerFuncName(r.Handler)))
	}

	b.WriteString(fmt.Sprintf("\n\tlog.Printf(\"Server starting on :%d\")\n", cfg.Port))
	b.WriteString(fmt.Sprintf("\tlog.Fatal(http.ListenAndServe(\":%d\", mux))\n", cfg.Port))
	b.WriteString("}\n")

	return b.String()
}

func (g *GoNetHTTPCodegen) generateHandlers(program *lang.CompiledProgram, routes []server.FlatRoute) string {
	var b strings.Builder
	b.WriteString("package main\n\n")
	b.WriteString("import (\n")
	b.WriteString("\t\"encoding/json\"\n")
	b.WriteString("\t\"net/http\"\n")
	b.WriteString(")\n\n")

	for _, r := range routes {
		if _, ok := program.Functions[r.Handler]; !ok {
			continue
		}
		b.WriteString(fmt.Sprintf("func %s(w http.ResponseWriter, r *http.Request) {\n", handlerFuncName(r.Handler)))
		b.WriteString("\tw.Header().Set(\"Content-Type\", \"application/json\")\n")
		b.WriteString("\tjson.NewEncoder(w).Encode(map[string]string{\"status\": \"ok\"})\n")
		b.WriteString("}\n\n")
	}

	return b.String()
}
