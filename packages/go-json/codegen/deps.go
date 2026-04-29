package codegen

import (
	"fmt"
	"strings"

	"github.com/bitcode-framework/go-json/lang"
)

// DetectedFeatures holds which features a program uses.
type DetectedFeatures struct {
	HasServer    bool
	HasJWT       bool
	HasSQL       bool
	HasHTTP      bool
	HasFS        bool
	HasRedis     bool
	HasMongoDB   bool
	HasTemplates bool
	HasCORS      bool
	Framework    string
}

// DetectFeatures scans a compiled program for feature usage.
func DetectFeatures(program *lang.CompiledProgram) DetectedFeatures {
	f := DetectedFeatures{}

	if program.AST.Server != nil {
		f.HasServer = true
		f.Framework = program.AST.Server.Framework
		if program.AST.Server.JWT != nil {
			f.HasJWT = true
		}
		if program.AST.Server.CORS != nil {
			f.HasCORS = true
		}
		if program.AST.Server.Templates != "" {
			f.HasTemplates = true
		}
	}

	if program.AST.RequestedModules != nil {
		for _, imp := range program.AST.RequestedModules {
			switch imp.Path {
			case "io:sql":
				f.HasSQL = true
			case "io:http":
				f.HasHTTP = true
			case "io:fs":
				f.HasFS = true
			case "io:redis":
				f.HasRedis = true
			case "io:mongo":
				f.HasMongoDB = true
			}
		}
	}

	return f
}

// GenerateGoMod generates go.mod content based on detected features.
func GenerateGoMod(moduleName string, features DetectedFeatures) string {
	var b strings.Builder
	b.WriteString(fmt.Sprintf("module %s\n\ngo 1.22\n\nrequire (\n", moduleName))

	switch features.Framework {
	case "fiber":
		b.WriteString("\tgithub.com/gofiber/fiber/v2 v2.52.0\n")
	case "echo":
		b.WriteString("\tgithub.com/labstack/echo/v4 v4.12.0\n")
	case "gin":
		b.WriteString("\tgithub.com/gin-gonic/gin v1.10.0\n")
	case "chi":
		b.WriteString("\tgithub.com/go-chi/chi/v5 v5.1.0\n")
	}

	if features.HasJWT {
		b.WriteString("\tgithub.com/golang-jwt/jwt/v5 v5.2.0\n")
	}
	if features.HasSQL {
		b.WriteString("\tgithub.com/mattn/go-sqlite3 v1.14.22\n")
	}
	if features.HasRedis {
		b.WriteString("\tgithub.com/redis/go-redis/v9 v9.5.0\n")
	}

	b.WriteString(")\n")
	return b.String()
}

// GeneratePackageJSON generates package.json content for JS projects.
func GeneratePackageJSON(name string, features DetectedFeatures) string {
	var b strings.Builder
	b.WriteString("{\n")
	b.WriteString(fmt.Sprintf("  \"name\": %q,\n", name))
	b.WriteString("  \"version\": \"1.0.0\",\n")
	b.WriteString("  \"main\": \"index.js\",\n")
	b.WriteString("  \"scripts\": {\n")
	b.WriteString("    \"start\": \"node index.js\",\n")
	b.WriteString("    \"dev\": \"nodemon index.js\"\n")
	b.WriteString("  },\n")
	b.WriteString("  \"dependencies\": {\n")

	deps := []string{`    "express": "^4.18.0"`}
	if features.HasJWT {
		deps = append(deps, `    "jsonwebtoken": "^9.0.0"`)
	}
	if features.HasCORS {
		deps = append(deps, `    "cors": "^2.8.0"`)
	}
	if features.HasSQL {
		deps = append(deps, `    "better-sqlite3": "^11.0.0"`)
	}

	b.WriteString(strings.Join(deps, ",\n"))
	b.WriteString("\n  }\n}\n")
	return b.String()
}

// GenerateRequirementsTxt generates requirements.txt for Python projects.
func GenerateRequirementsTxt(features DetectedFeatures) string {
	var lines []string
	lines = append(lines, "fastapi>=0.110.0")
	lines = append(lines, "uvicorn[standard]>=0.29.0")

	if features.HasJWT {
		lines = append(lines, "PyJWT>=2.8.0")
	}
	if features.HasSQL {
		lines = append(lines, "sqlalchemy>=2.0.0")
	}
	if features.HasRedis {
		lines = append(lines, "redis>=5.0.0")
	}

	return strings.Join(lines, "\n") + "\n"
}

// GenerateEnvExample generates .env.example from program config.
func GenerateEnvExample(program *lang.CompiledProgram) string {
	var lines []string

	if program.AST.Server != nil {
		lines = append(lines, fmt.Sprintf("PORT=%d", program.AST.Server.Port))

		if program.AST.Server.JWT != nil && program.AST.Server.JWT.SecretEnv != "" {
			lines = append(lines, fmt.Sprintf("%s=your-secret-key-here", program.AST.Server.JWT.SecretEnv))
		}

		if program.AST.Server.Auth != nil {
			for _, sc := range program.AST.Server.Auth.Strategies {
				if sc.SecretEnv != "" {
					lines = append(lines, fmt.Sprintf("%s=your-secret-here", sc.SecretEnv))
				}
				if sc.KeysEnv != "" {
					lines = append(lines, fmt.Sprintf("%s=key1:name1,key2:name2", sc.KeysEnv))
				}
				if sc.UsersEnv != "" {
					lines = append(lines, fmt.Sprintf("%s=user1:pass1,user2:pass2", sc.UsersEnv))
				}
			}
		}
	}

	if program.AST.RequestedModules != nil {
		for _, imp := range program.AST.RequestedModules {
			if imp.Path == "io:sql" {
				lines = append(lines, "DATABASE_URL=sqlite:./app.db")
			}
			if imp.Path == "io:redis" {
				lines = append(lines, "REDIS_URL=redis://localhost:6379")
			}
		}
	}

	if len(lines) == 0 {
		return ""
	}

	return strings.Join(lines, "\n") + "\n"
}
