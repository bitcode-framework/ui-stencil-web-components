package lang

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/bitcode-framework/go-json/stdlib"
)

// defaultInputs maps testdata filenames (without extension) to the input
// they require. Programs not listed here run with nil input.
var defaultInputs = map[string]map[string]any{
	"control_flow": {"score": 85},
}

func TestTestdata_AllFixtures(t *testing.T) {
	entries, err := os.ReadDir("../testdata")
	if err != nil {
		t.Fatalf("failed to read testdata dir: %v", err)
	}

	reg := stdlib.DefaultRegistry()

	var found int
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		ext := filepath.Ext(name)
		if ext != ".json" && ext != ".jsonc" {
			continue
		}
		found++

		baseName := strings.TrimSuffix(name, ext)
		t.Run(baseName, func(t *testing.T) {
			data, err := os.ReadFile(filepath.Join("../testdata", name))
			if err != nil {
				t.Fatalf("failed to read %s: %v", name, err)
			}

			program, err := Parse(data)
			if err != nil {
				t.Fatalf("parse error for %s: %v", name, err)
			}

			engine := NewExprLangEngine(reg.All()...)
			compiled, err := Compile(program, engine, DefaultLimits())
			if err != nil {
				t.Fatalf("compile error for %s: %v", name, err)
			}

			input := defaultInputs[baseName]
			for k, v := range reg.EnvVars() {
				if input == nil {
					input = make(map[string]any)
				}
				input[k] = v
			}

			vm := NewVM(compiled, engine)
			result, err := vm.Execute(input)
			if err != nil {
				t.Fatalf("execution error for %s: %v", name, err)
			}

			if result.Value != nil {
				t.Logf("%s: returned %v (type: %T)", name, result.Value, result.Value)
			}
		})
	}

	if found == 0 {
		t.Fatal("no .json or .jsonc fixture files found in ../testdata")
	}
	t.Logf("ran %d fixture files", found)
}
