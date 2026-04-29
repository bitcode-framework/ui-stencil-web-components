package codegen

import (
	"fmt"
	"strings"

	"github.com/bitcode-framework/go-json/lang"
	"github.com/bitcode-framework/go-json/server"
)

func init() {
	RegisterServerCodegen("python", "fastapi", &PyFastAPICodegen{})
}

type PyFastAPICodegen struct{}

func (g *PyFastAPICodegen) Framework() string { return "fastapi" }
func (g *PyFastAPICodegen) Language() string  { return "python" }

func (g *PyFastAPICodegen) GenerateServer(program *lang.CompiledProgram) (map[string]string, error) {
	files := make(map[string]string)

	cfg := program.AST.Server
	if cfg == nil {
		cfg = &lang.ServerConfig{}
	}
	server.MergeDefaults(cfg)

	flatRoutes := server.FlattenRoutes(program.AST.Routes, "", program.AST.Middleware)

	files["main.py"] = g.generateMain(cfg, flatRoutes)
	files["routes.py"] = g.generateRoutes(flatRoutes)
	files["middleware.py"] = g.generateMiddleware(cfg)

	return files, nil
}

func (g *PyFastAPICodegen) generateMain(cfg *lang.ServerConfig, routes []server.FlatRoute) string {
	var b strings.Builder
	b.WriteString("import uvicorn\n")
	b.WriteString("from fastapi import FastAPI\n")
	if cfg.CORS != nil {
		b.WriteString("from fastapi.middleware.cors import CORSMiddleware\n")
	}
	b.WriteString("from routes import router\n\n")

	b.WriteString("app = FastAPI()\n\n")

	if cfg.CORS != nil {
		b.WriteString("app.add_middleware(\n")
		b.WriteString("    CORSMiddleware,\n")
		origins := make([]string, len(cfg.CORS.Origins))
		for i, o := range cfg.CORS.Origins {
			origins[i] = fmt.Sprintf("    \"%s\"", o)
		}
		b.WriteString(fmt.Sprintf("    allow_origins=[%s],\n", strings.Join(cfg.CORS.Origins, ", ")))
		b.WriteString("    allow_methods=[\"*\"],\n")
		b.WriteString("    allow_headers=[\"*\"],\n")
		b.WriteString(")\n\n")
	}

	b.WriteString("app.include_router(router)\n\n")
	b.WriteString("if __name__ == \"__main__\":\n")
	b.WriteString(fmt.Sprintf("    uvicorn.run(\"main:app\", host=\"%s\", port=%d, reload=True)\n", cfg.Host, cfg.Port))

	return b.String()
}

func (g *PyFastAPICodegen) generateRoutes(routes []server.FlatRoute) string {
	var b strings.Builder
	b.WriteString("from fastapi import APIRouter, Request\n\n")
	b.WriteString("router = APIRouter()\n\n")

	for _, r := range routes {
		method := strings.ToLower(r.Method)
		path := convertPythonPathParams(r.FullPath)
		b.WriteString(fmt.Sprintf("@router.%s(\"%s\")\n", method, path))
		b.WriteString(fmt.Sprintf("async def %s(request: Request):\n", pythonFuncName(r.Handler)))
		b.WriteString("    # TODO: implement handler logic\n")
		b.WriteString("    return {\"status\": \"ok\"}\n\n")
	}

	return b.String()
}

func (g *PyFastAPICodegen) generateMiddleware(cfg *lang.ServerConfig) string {
	var b strings.Builder
	b.WriteString("# Middleware configuration\n\n")

	if cfg.JWT != nil {
		b.WriteString("import os\n")
		b.WriteString("import jwt\n")
		b.WriteString("from fastapi import HTTPException, Depends\n")
		b.WriteString("from fastapi.security import HTTPBearer\n\n")
		b.WriteString("security = HTTPBearer()\n\n")
		b.WriteString("async def verify_token(credentials = Depends(security)):\n")
		b.WriteString(fmt.Sprintf("    secret = os.getenv(\"%s\")\n", cfg.JWT.SecretEnv))
		b.WriteString("    try:\n")
		b.WriteString("        payload = jwt.decode(credentials.credentials, secret, algorithms=[\"HS256\"])\n")
		b.WriteString("        return payload\n")
		b.WriteString("    except jwt.PyJWTError:\n")
		b.WriteString("        raise HTTPException(status_code=401, detail=\"Invalid token\")\n")
	}

	return b.String()
}

func convertPythonPathParams(path string) string {
	parts := strings.Split(path, "/")
	for i, part := range parts {
		if strings.HasPrefix(part, ":") {
			parts[i] = "{" + part[1:] + "}"
		}
	}
	return strings.Join(parts, "/")
}

func pythonFuncName(name string) string {
	result := strings.Builder{}
	for i, c := range name {
		if c >= 'A' && c <= 'Z' {
			if i > 0 {
				result.WriteByte('_')
			}
			result.WriteRune(c + 32)
		} else {
			result.WriteRune(c)
		}
	}
	return result.String()
}
