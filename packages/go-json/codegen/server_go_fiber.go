package codegen

import (
	"fmt"
	"strings"

	"github.com/bitcode-framework/go-json/lang"
	"github.com/bitcode-framework/go-json/server"
)

func init() {
	RegisterServerCodegen("go", "fiber", &GoFiberCodegen{})
}

type GoFiberCodegen struct{}

func (g *GoFiberCodegen) Framework() string { return "fiber" }
func (g *GoFiberCodegen) Language() string  { return "go" }

func (g *GoFiberCodegen) GenerateServer(program *lang.CompiledProgram) (map[string]string, error) {
	files := make(map[string]string)

	cfg := program.AST.Server
	if cfg == nil {
		cfg = &lang.ServerConfig{}
	}
	server.MergeDefaults(cfg)

	flatRoutes := server.FlattenRoutes(program.AST.Routes, "", program.AST.Middleware)

	files["main.go"] = g.generateMain(cfg, flatRoutes)
	files["handlers.go"] = g.generateHandlers(program, flatRoutes)
	files["types.go"] = g.generateTypes()

	return files, nil
}

func (g *GoFiberCodegen) generateMain(cfg *lang.ServerConfig, routes []server.FlatRoute) string {
	var b strings.Builder
	b.WriteString("package main\n\n")
	b.WriteString("import (\n")
	b.WriteString("\t\"log\"\n")
	b.WriteString("\t\"os\"\n")
	b.WriteString("\t\"os/signal\"\n")
	b.WriteString("\t\"syscall\"\n\n")
	b.WriteString("\t\"github.com/gofiber/fiber/v2\"\n")
	if cfg.CORS != nil {
		b.WriteString("\t\"github.com/gofiber/fiber/v2/middleware/cors\"\n")
	}
	b.WriteString(")\n\n")

	b.WriteString("func main() {\n")
	b.WriteString("\tapp := fiber.New()\n\n")

	if cfg.CORS != nil {
		b.WriteString("\tapp.Use(cors.New(cors.Config{\n")
		b.WriteString(fmt.Sprintf("\t\tAllowOrigins: %q,\n", strings.Join(cfg.CORS.Origins, ",")))
		b.WriteString("\t}))\n\n")
	}

	for _, r := range routes {
		method := strings.Title(strings.ToLower(r.Method))
		b.WriteString(fmt.Sprintf("\tapp.%s(%q, %s)\n", method, r.FullPath, handlerFuncName(r.Handler)))
	}

	b.WriteString(fmt.Sprintf("\n\tgo func() {\n\t\tif err := app.Listen(\":%d\"); err != nil {\n\t\t\tlog.Fatal(err)\n\t\t}\n\t}()\n\n", cfg.Port))
	b.WriteString("\tquit := make(chan os.Signal, 1)\n")
	b.WriteString("\tsignal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)\n")
	b.WriteString("\t<-quit\n")
	b.WriteString("\tapp.Shutdown()\n")
	b.WriteString("}\n")

	return b.String()
}

func (g *GoFiberCodegen) generateHandlers(program *lang.CompiledProgram, routes []server.FlatRoute) string {
	var b strings.Builder
	b.WriteString("package main\n\n")
	b.WriteString("import \"github.com/gofiber/fiber/v2\"\n\n")

	for _, r := range routes {
		fn, ok := program.Functions[r.Handler]
		if !ok {
			continue
		}
		b.WriteString(fmt.Sprintf("func %s(c *fiber.Ctx) error {\n", handlerFuncName(r.Handler)))
		b.WriteString("\t// TODO: implement handler logic\n")
		_ = fn
		b.WriteString("\treturn c.JSON(fiber.Map{\"status\": \"ok\"})\n")
		b.WriteString("}\n\n")
	}

	return b.String()
}

func (g *GoFiberCodegen) generateTypes() string {
	var b strings.Builder
	b.WriteString("package main\n\n")
	b.WriteString("// Request types and response types go here.\n")
	return b.String()
}

func handlerFuncName(name string) string {
	if len(name) == 0 {
		return "handler"
	}
	return "handle" + strings.ToUpper(name[:1]) + name[1:]
}
