package codegen

import (
	"fmt"
	"strings"

	"github.com/bitcode-framework/go-json/lang"
	"github.com/bitcode-framework/go-json/server"
)

func init() {
	RegisterServerCodegen("javascript", "express", &JSExpressCodegen{})
}

type JSExpressCodegen struct{}

func (g *JSExpressCodegen) Framework() string { return "express" }
func (g *JSExpressCodegen) Language() string  { return "javascript" }

func (g *JSExpressCodegen) GenerateServer(program *lang.CompiledProgram) (map[string]string, error) {
	files := make(map[string]string)

	cfg := program.AST.Server
	if cfg == nil {
		cfg = &lang.ServerConfig{}
	}
	server.MergeDefaults(cfg)

	flatRoutes := server.FlattenRoutes(program.AST.Routes, "", program.AST.Middleware)

	files["index.js"] = g.generateIndex(cfg, flatRoutes)
	files["routes.js"] = g.generateRoutes(flatRoutes)
	files["middleware.js"] = g.generateMiddleware(cfg)

	return files, nil
}

func (g *JSExpressCodegen) generateIndex(cfg *lang.ServerConfig, routes []server.FlatRoute) string {
	var b strings.Builder
	b.WriteString("const express = require('express');\n")
	if cfg.CORS != nil {
		b.WriteString("const cors = require('cors');\n")
	}
	b.WriteString("const { setupRoutes } = require('./routes');\n\n")

	b.WriteString("const app = express();\n\n")
	b.WriteString("app.use(express.json());\n")

	if cfg.CORS != nil {
		b.WriteString(fmt.Sprintf("app.use(cors({ origin: %s }));\n",
			formatJSArray(cfg.CORS.Origins)))
	}

	b.WriteString("\nsetupRoutes(app);\n\n")
	b.WriteString(fmt.Sprintf("const PORT = process.env.PORT || %d;\n", cfg.Port))
	b.WriteString("app.listen(PORT, () => {\n")
	b.WriteString("  console.log(`Server running on port ${PORT}`);\n")
	b.WriteString("});\n")

	return b.String()
}

func (g *JSExpressCodegen) generateRoutes(routes []server.FlatRoute) string {
	var b strings.Builder
	b.WriteString("function setupRoutes(app) {\n")

	for _, r := range routes {
		method := strings.ToLower(r.Method)
		path := convertPathParams(r.FullPath)
		b.WriteString(fmt.Sprintf("  app.%s('%s', (req, res) => {\n", method, path))
		b.WriteString("    // TODO: implement handler logic\n")
		b.WriteString("    res.json({ status: 'ok' });\n")
		b.WriteString("  });\n\n")
	}

	b.WriteString("}\n\n")
	b.WriteString("module.exports = { setupRoutes };\n")

	return b.String()
}

func (g *JSExpressCodegen) generateMiddleware(cfg *lang.ServerConfig) string {
	var b strings.Builder
	b.WriteString("// Middleware configuration\n\n")

	if cfg.JWT != nil {
		b.WriteString("const jwt = require('jsonwebtoken');\n\n")
		b.WriteString("function authMiddleware(req, res, next) {\n")
		b.WriteString("  const token = req.headers.authorization?.replace('Bearer ', '');\n")
		b.WriteString("  if (!token) return res.status(401).json({ error: 'Unauthorized' });\n")
		b.WriteString(fmt.Sprintf("  try {\n    req.user = jwt.verify(token, process.env.%s);\n    next();\n", cfg.JWT.SecretEnv))
		b.WriteString("  } catch (err) {\n    res.status(401).json({ error: 'Invalid token' });\n  }\n}\n\n")
		b.WriteString("module.exports = { authMiddleware };\n")
	} else {
		b.WriteString("module.exports = {};\n")
	}

	return b.String()
}

func convertPathParams(path string) string {
	parts := strings.Split(path, "/")
	for i, part := range parts {
		if strings.HasPrefix(part, ":") {
			parts[i] = ":" + part[1:]
		}
	}
	return strings.Join(parts, "/")
}

func formatJSArray(items []string) string {
	if len(items) == 1 {
		return fmt.Sprintf("'%s'", items[0])
	}
	quoted := make([]string, len(items))
	for i, item := range items {
		quoted[i] = fmt.Sprintf("'%s'", item)
	}
	return "[" + strings.Join(quoted, ", ") + "]"
}
