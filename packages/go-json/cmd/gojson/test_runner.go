package gojson

import (
	"encoding/json"
	"flag"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"time"

	"github.com/bitcode-framework/go-json/runtime"
	"github.com/bitcode-framework/go-json/stdlib"
)

type testFile struct {
	Name   string            `json:"name"`
	Test   bool              `json:"test"`
	Import map[string]string `json:"import"`
	Cases  []testCase        `json:"cases"`
	Before []any             `json:"before"`
	After  []any             `json:"after"`
}

type testCase struct {
	Comment     string         `json:"_c"`
	Call        string         `json:"call"`
	With        map[string]any `json:"with"`
	Expect      any            `json:"expect"`
	ExpectError any            `json:"expect_error"`
	Skip        bool           `json:"skip"`
	Only        bool           `json:"only"`
	Timeout     int            `json:"timeout"`
	Table       []tableEntry   `json:"table"`
}

type tableEntry struct {
	With   map[string]any `json:"with"`
	Expect any            `json:"expect"`
}

type testResult struct {
	Name     string
	Comment  string
	Passed   bool
	Duration time.Duration
	Expected any
	Got      any
	Error    string
}

func cmdTest(args []string) {
	fs := flag.NewFlagSet("test", flag.ExitOnError)
	verbose := fs.Bool("verbose", false, "Show input/output for each case")
	filter := fs.String("filter", "", "Run only cases matching pattern")
	fs.Parse(args)

	if fs.NArg() < 1 {
		fmt.Fprintln(os.Stderr, "Usage: go-json test <dir|file> [--verbose] [--filter pattern]")
		os.Exit(1)
	}

	target := fs.Arg(0)

	var testFiles []string
	info, err := os.Stat(target)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %s\n", err.Error())
		os.Exit(1)
	}

	if info.IsDir() {
		testFiles = findTestFiles(target)
	} else {
		testFiles = []string{target}
	}

	if len(testFiles) == 0 {
		fmt.Println("Warning: no test files found")
		return
	}

	var allResults []testResult
	passed, failed := 0, 0

	for _, tf := range testFiles {
		results := runTestFile(tf, *filter, *verbose)
		for _, r := range results {
			allResults = append(allResults, r)
			if r.Passed {
				passed++
			} else {
				failed++
			}
		}
	}

	fmt.Println()
	for _, r := range allResults {
		if r.Passed {
			fmt.Printf("  \u2713 %s: %s (%s)\n", r.Name, r.Comment, r.Duration)
		} else {
			fmt.Printf("  \u2717 %s: %s\n", r.Name, r.Comment)
			if r.Error != "" {
				fmt.Printf("    Error: %s\n", r.Error)
			} else {
				expectedJSON, _ := json.Marshal(r.Expected)
				gotJSON, _ := json.Marshal(r.Got)
				fmt.Printf("    Expected: %s\n", string(expectedJSON))
				fmt.Printf("    Got: %s\n", string(gotJSON))
			}
		}
	}

	fmt.Printf("\n%d passed, %d failed\n", passed, failed)

	if failed > 0 {
		os.Exit(1)
	}
}

func findTestFiles(dir string) []string {
	var files []string
	filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if info.IsDir() {
			return nil
		}
		if !strings.HasSuffix(path, ".json") {
			return nil
		}

		data, err := os.ReadFile(path)
		if err != nil {
			return nil
		}

		var tf testFile
		if err := json.Unmarshal(data, &tf); err != nil {
			return nil
		}
		if tf.Test {
			files = append(files, path)
		}
		return nil
	})
	return files
}

func runTestFile(path, filter string, verbose bool) []testResult {
	data, err := os.ReadFile(path)
	if err != nil {
		return []testResult{{Name: path, Comment: "read error", Error: err.Error()}}
	}

	var tf testFile
	if err := json.Unmarshal(data, &tf); err != nil {
		return []testResult{{Name: path, Comment: "parse error", Error: err.Error()}}
	}

	if len(tf.Cases) == 0 {
		fmt.Printf("Warning: %s has no test cases, skipping\n", path)
		return nil
	}

	reg := stdlib.DefaultRegistry()
	rt := runtime.NewRuntime(
		runtime.WithStdlib(reg.All()),
		runtime.WithStdlibEnv(reg.EnvVars()),
	)

	dir := filepath.Dir(path)
	var results []testResult

	hasOnly := false
	for _, tc := range tf.Cases {
		if tc.Only {
			hasOnly = true
			break
		}
	}

	for _, tc := range tf.Cases {
		if tc.Skip {
			results = append(results, testResult{
				Name: tf.Name, Comment: tc.Comment + " [SKIPPED]", Passed: true,
			})
			continue
		}
		if hasOnly && !tc.Only {
			continue
		}
		if filter != "" && !strings.Contains(tc.Comment, filter) && !strings.Contains(tc.Call, filter) {
			continue
		}

		if tc.Table != nil {
			for i, entry := range tc.Table {
				tableTc := tc
				tableTc.With = entry.With
				tableTc.Expect = entry.Expect
				tableTc.Table = nil
				tableComment := fmt.Sprintf("%s [%d]", tc.Comment, i)
				tr := runSingleCase(rt, dir, tf, tableTc, tableComment, verbose)
				results = append(results, tr)
			}
			continue
		}

		tr := runSingleCase(rt, dir, tf, tc, tc.Comment, verbose)
		results = append(results, tr)
	}

	return results
}

func runSingleCase(rt *runtime.Runtime, dir string, tf testFile, tc testCase, comment string, verbose bool) testResult {
	start := time.Now()
	result, err := executeTestCase(rt, dir, tf, tc)
	duration := time.Since(start)

	tr := testResult{
		Name:     tf.Name,
		Comment:  comment,
		Duration: duration,
		Expected: tc.Expect,
		Got:      result,
	}

	if tc.ExpectError != nil {
		if err == nil {
			tr.Passed = false
			tr.Error = "expected error but got none"
		} else {
			tr.Passed = matchExpectError(err, tc.ExpectError)
			if !tr.Passed {
				tr.Error = fmt.Sprintf("error mismatch: got '%s'", err.Error())
			}
		}
	} else if err != nil {
		tr.Error = err.Error()
		tr.Passed = false
	} else {
		tr.Passed = deepEqual(result, tc.Expect)
	}

	if verbose {
		fmt.Printf("  [%s] %s\n", tf.Name, comment)
		fmt.Printf("    Call: %s\n", tc.Call)
		withJSON, _ := json.Marshal(tc.With)
		fmt.Printf("    With: %s\n", string(withJSON))
		resultJSON, _ := json.Marshal(result)
		fmt.Printf("    Result: %s\n", string(resultJSON))
	}

	return tr
}

func matchExpectError(err error, expectError any) bool {
	errMsg := err.Error()
	switch v := expectError.(type) {
	case string:
		return strings.Contains(errMsg, v)
	case map[string]any:
		if code, ok := v["code"].(string); ok {
			return strings.Contains(errMsg, code)
		}
		if msg, ok := v["message"].(string); ok {
			return strings.Contains(errMsg, msg)
		}
	case bool:
		return v
	}
	return true
}

func executeTestCase(rt *runtime.Runtime, dir string, tf testFile, tc testCase) (any, error) {
	parts := strings.SplitN(tc.Call, ".", 2)
	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid call format: %s (expected 'module.function')", tc.Call)
	}

	alias := parts[0]
	funcName := parts[1]

	importPath, ok := tf.Import[alias]
	if !ok {
		return nil, fmt.Errorf("import alias '%s' not found", alias)
	}

	wrapperJSON := buildTestWrapper(importPath, funcName, tc.With)

	tmpFile := filepath.Join(dir, fmt.Sprintf("_test_wrapper_%d.json", time.Now().UnixNano()))
	if err := os.WriteFile(tmpFile, []byte(wrapperJSON), 0644); err != nil {
		return nil, fmt.Errorf("cannot write test wrapper: %s", err.Error())
	}
	defer os.Remove(tmpFile)

	compiled, err := rt.CompileFile(tmpFile)
	if err != nil {
		return nil, err
	}

	result, err := rt.Execute(compiled, nil)
	if err != nil {
		return nil, err
	}

	return result.Value, nil
}

func buildTestWrapper(importPath, funcName string, with map[string]any) string {
	withExprs := make(map[string]string)
	for k, v := range with {
		switch val := v.(type) {
		case string:
			withExprs[k] = val
		default:
			j, _ := json.Marshal(val)
			withExprs[k] = string(j)
		}
	}

	withJSON, _ := json.Marshal(withExprs)

	return fmt.Sprintf(`{
		"import": {"_m": "%s"},
		"steps": [
			{"let": "_result", "call": "_m.%s", "with": %s},
			{"return": "_result"}
		]
	}`, importPath, funcName, string(withJSON))
}

func deepEqual(a, b any) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}

	af, aIsFloat := toFloat(a)
	bf, bIsFloat := toFloat(b)
	if aIsFloat && bIsFloat {
		return math.Abs(af-bf) < 1e-9
	}

	if reflect.DeepEqual(a, b) {
		return true
	}

	aJSON, _ := json.Marshal(a)
	bJSON, _ := json.Marshal(b)
	return string(aJSON) == string(bJSON)
}

func toFloat(v any) (float64, bool) {
	switch n := v.(type) {
	case float64:
		return n, true
	case int:
		return float64(n), true
	case int64:
		return float64(n), true
	case json.Number:
		f, err := n.Float64()
		return f, err == nil
	}
	return 0, false
}
