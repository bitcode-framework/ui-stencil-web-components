package generate

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/template"
)

// PatternMeta describes a codegen pattern template.
type PatternMeta struct {
	Name        string       `json:"name"`
	Description string       `json:"description"`
	Language    string       `json:"language"`
	Files       []PatternFile `json:"files"`
}

// PatternFile describes a single file in a pattern template.
type PatternFile struct {
	Template string `json:"template"`
	Output   string `json:"output"`
	Once     bool   `json:"once,omitempty"`
	PerModel bool   `json:"per_model,omitempty"`
}

// PatternData holds the data passed to pattern templates.
type PatternData struct {
	ProjectName string
	Model       *TableInfo
	Models      []*TableInfo
}

// BuiltinPatterns returns the names of all built-in patterns.
func BuiltinPatterns() []string {
	return []string{"simple", "service-layer", "ddd", "hexagonal"}
}

// GenerateFromPattern generates files using a pattern template.
func GenerateFromPattern(patternDir string, models []*TableInfo, projectName string) (map[string]string, error) {
	metaPath := filepath.Join(patternDir, "template.json")
	metaData, err := os.ReadFile(metaPath)
	if err != nil {
		return nil, fmt.Errorf("read pattern metadata: %w", err)
	}

	var meta PatternMeta
	if err := json.Unmarshal(metaData, &meta); err != nil {
		return nil, fmt.Errorf("parse pattern metadata: %w", err)
	}

	files := make(map[string]string)

	for _, pf := range meta.Files {
		tmplPath := filepath.Join(patternDir, pf.Template)
		tmplContent, err := os.ReadFile(tmplPath)
		if err != nil {
			return nil, fmt.Errorf("read template %s: %w", pf.Template, err)
		}

		tmpl, err := template.New(pf.Template).Funcs(patternFuncMap()).Parse(string(tmplContent))
		if err != nil {
			return nil, fmt.Errorf("parse template %s: %w", pf.Template, err)
		}

		if pf.Once {
			data := PatternData{
				ProjectName: projectName,
				Models:      models,
			}
			var buf strings.Builder
			if err := tmpl.Execute(&buf, data); err != nil {
				return nil, fmt.Errorf("execute template %s: %w", pf.Template, err)
			}
			files[pf.Output] = buf.String()
		} else if pf.PerModel {
			for _, model := range models {
				data := PatternData{
					ProjectName: projectName,
					Model:       model,
					Models:      models,
				}
				var buf strings.Builder
				if err := tmpl.Execute(&buf, data); err != nil {
					return nil, fmt.Errorf("execute template %s for %s: %w", pf.Template, model.Name, err)
				}
				outputPath := strings.ReplaceAll(pf.Output, "{{.Model.Name}}", model.Name)
				outputPath = strings.ReplaceAll(outputPath, "{{.ModelLower}}", strings.ToLower(model.Name))
				files[outputPath] = buf.String()
			}
		}
	}

	return files, nil
}

// ExportPattern copies a built-in pattern to a user directory.
func ExportPattern(patternName, outputDir string) error {
	builtinDir := getBuiltinPatternDir(patternName)
	if builtinDir == "" {
		return fmt.Errorf("unknown built-in pattern: %s", patternName)
	}

	return filepath.Walk(builtinDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(builtinDir, path)
		target := filepath.Join(outputDir, rel)

		if info.IsDir() {
			return os.MkdirAll(target, 0755)
		}

		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		return os.WriteFile(target, data, 0644)
	})
}

func getBuiltinPatternDir(name string) string {
	exe, err := os.Executable()
	if err != nil {
		return ""
	}
	dir := filepath.Dir(exe)
	candidate := filepath.Join(dir, "templates", name)
	if _, err := os.Stat(candidate); err == nil {
		return candidate
	}
	return ""
}

func patternFuncMap() template.FuncMap {
	return template.FuncMap{
		"lower":      strings.ToLower,
		"upper":      strings.ToUpper,
		"title":      strings.Title,
		"capitalize": capitalize,
		"singular":   singularize,
		"plural": func(s string) string {
			if strings.HasSuffix(s, "y") {
				return s[:len(s)-1] + "ies"
			}
			return s + "s"
		},
		"snake": func(s string) string {
			var result strings.Builder
			for i, c := range s {
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
		},
		"camel": func(s string) string {
			parts := strings.Split(s, "_")
			for i := range parts {
				if i > 0 {
					parts[i] = capitalize(parts[i])
				}
			}
			return strings.Join(parts, "")
		},
		"pascal": func(s string) string {
			parts := strings.Split(s, "_")
			for i := range parts {
				parts[i] = capitalize(parts[i])
			}
			return strings.Join(parts, "")
		},
	}
}
