package server

import (
	"bytes"
	"encoding/json"
	"fmt"
	"html/template"
	"math"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// TemplateEngine manages Go html/template rendering with layouts and partials.
type TemplateEngine struct {
	dir      string
	devMode  bool
	mu       sync.RWMutex
	compiled *template.Template
	funcMap  template.FuncMap
}

// NewTemplateEngine creates a template engine that loads templates from dir.
func NewTemplateEngine(dir string, devMode bool) (*TemplateEngine, error) {
	te := &TemplateEngine{
		dir:     dir,
		devMode: devMode,
		funcMap: builtinTemplateFuncs(),
	}

	if err := te.load(); err != nil {
		return nil, err
	}

	return te, nil
}

// Render executes a named template with the given data and returns HTML.
func (te *TemplateEngine) Render(name string, data map[string]any) (string, error) {
	if te.devMode {
		if err := te.load(); err != nil {
			return "", fmt.Errorf("template reload error: %w", err)
		}
	}

	te.mu.RLock()
	tmpl := te.compiled
	te.mu.RUnlock()

	if tmpl == nil {
		return "", fmt.Errorf("no templates loaded")
	}

	var buf bytes.Buffer
	if err := tmpl.ExecuteTemplate(&buf, name, data); err != nil {
		return "", fmt.Errorf("template %q: %w", name, err)
	}

	return buf.String(), nil
}

func (te *TemplateEngine) load() error {
	te.mu.Lock()
	defer te.mu.Unlock()

	tmpl := template.New("").Funcs(te.funcMap)

	err := filepath.Walk(te.dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}

		ext := filepath.Ext(path)
		if ext != ".html" && ext != ".tmpl" && ext != ".gohtml" {
			return nil
		}

		rel, err := filepath.Rel(te.dir, path)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)

		content, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("read template %s: %w", rel, err)
		}

		_, err = tmpl.New(rel).Parse(string(content))
		if err != nil {
			return fmt.Errorf("parse template %s: %w", rel, err)
		}

		return nil
	})

	if err != nil {
		return fmt.Errorf("load templates from %s: %w", te.dir, err)
	}

	te.compiled = tmpl
	return nil
}

func builtinTemplateFuncs() template.FuncMap {
	return template.FuncMap{
		"json": func(v any) string {
			b, err := json.Marshal(v)
			if err != nil {
				return ""
			}
			return string(b)
		},
		"formatDate": func(t any, layout string) string {
			switch v := t.(type) {
			case time.Time:
				return v.Format(translateDateFormat(layout))
			case string:
				parsed, err := time.Parse(time.RFC3339, v)
				if err != nil {
					return v
				}
				return parsed.Format(translateDateFormat(layout))
			}
			return fmt.Sprintf("%v", t)
		},
		"upper":   strings.ToUpper,
		"lower":   strings.ToLower,
		"title":   strings.Title,
		"replace": strings.ReplaceAll,
		"truncate": func(s string, n int) string {
			if len(s) <= n {
				return s
			}
			if n <= 3 {
				return s[:n]
			}
			return s[:n-3] + "..."
		},
		"default": func(def, val any) any {
			if val == nil || val == "" || val == 0 || val == false {
				return def
			}
			return val
		},
		"safeHTML": func(s string) template.HTML {
			return template.HTML(s)
		},
		"urlEncode": url.QueryEscape,
		"add": func(a, b int) int {
			return a + b
		},
		"sub": func(a, b int) int {
			return a - b
		},
		"mul": func(a, b int) int {
			return a * b
		},
		"div": func(a, b int) int {
			if b == 0 {
				return 0
			}
			return a / b
		},
		"mod": func(a, b int) int {
			if b == 0 {
				return 0
			}
			return a % b
		},
		"seq": func(args ...int) []int {
			start, end, step := 0, 0, 1
			switch len(args) {
			case 1:
				end = args[0]
			case 2:
				start, end = args[0], args[1]
			case 3:
				start, end, step = args[0], args[1], args[2]
			default:
				return nil
			}
			if step == 0 {
				return nil
			}
			var result []int
			maxIter := int(math.Abs(float64(end-start)/float64(step))) + 1
			if maxIter > 10000 {
				return nil
			}
			if step > 0 {
				for i := start; i < end; i += step {
					result = append(result, i)
				}
			} else {
				for i := start; i > end; i += step {
					result = append(result, i)
				}
			}
			return result
		},
		"contains": strings.Contains,
		"hasPrefix": strings.HasPrefix,
		"hasSuffix": strings.HasSuffix,
		"split":     strings.Split,
		"join":      strings.Join,
		"trim":      strings.TrimSpace,
	}
}

func translateDateFormat(layout string) string {
	if strings.Contains(layout, "2006") {
		return layout
	}
	replacer := strings.NewReplacer(
		"YYYY", "2006",
		"YY", "06",
		"MM", "01",
		"DD", "02",
		"HH", "15",
		"hh", "03",
		"mm", "04",
		"ss", "05",
		"SSS", "000",
		"TT", "PM",
	)
	return replacer.Replace(layout)
}
