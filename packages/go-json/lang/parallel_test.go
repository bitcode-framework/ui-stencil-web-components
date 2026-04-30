package lang

import (
	"strings"
	"testing"
)

func TestParallel_BasicExecution(t *testing.T) {
	result := compileAndRun(t, `{
		"steps": [
			{
				"parallel": {
					"a": [{"return": 1}],
					"b": [{"return": 2}]
				},
				"into": "results"
			},
			{"return": "results"}
		]
	}`, nil)

	m, ok := result.Value.(map[string]any)
	if !ok {
		t.Fatalf("expected map, got %T", result.Value)
	}
	if !numEq(m["a"], 1) {
		t.Errorf("expected a=1, got %v", m["a"])
	}
	if !numEq(m["b"], 2) {
		t.Errorf("expected b=2, got %v", m["b"])
	}
}

func TestParallel_EmptyBranches(t *testing.T) {
	result := compileAndRun(t, `{
		"steps": [
			{
				"parallel": {},
				"into": "results"
			},
			{"return": "results"}
		]
	}`, nil)

	m, ok := result.Value.(map[string]any)
	if !ok {
		t.Fatalf("expected map, got %T", result.Value)
	}
	if len(m) != 0 {
		t.Errorf("expected empty map, got %v", m)
	}
}

func TestParallel_ReadsParentScope(t *testing.T) {
	result := compileAndRun(t, `{
		"steps": [
			{"let": "x", "value": 10},
			{
				"parallel": {
					"doubled": [{"return": "x * 2"}]
				},
				"into": "results"
			},
			{"return": "results.doubled"}
		]
	}`, nil)

	if !numEq(result.Value, 20) {
		t.Errorf("expected 20, got %v", result.Value)
	}
}

func TestParallel_CancelAll_OnError(t *testing.T) {
	program, err := Parse([]byte(`{
		"steps": [
			{
				"parallel": {
					"fail": [{"error": "'branch failed'"}],
					"ok": [{"return": 42}]
				},
				"on_error": "cancel_all",
				"into": "results"
			}
		]
	}`))
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	engine := NewExprLangEngine()
	compiled, err := Compile(program, engine, DefaultLimits())
	if err != nil {
		t.Fatalf("compile error: %v", err)
	}

	vm := NewVM(compiled, engine)
	_, err = vm.Execute(nil)
	if err == nil {
		t.Fatal("expected error from cancel_all mode")
	}
	if !strings.Contains(err.Error(), "branch failed") {
		t.Errorf("expected 'branch failed' error, got: %v", err)
	}
}

func TestParallel_Continue_OnError(t *testing.T) {
	result := compileAndRun(t, `{
		"steps": [
			{
				"parallel": {
					"fail": [{"error": "'oops'"}],
					"ok": [{"return": 42}]
				},
				"on_error": "continue",
				"into": "results"
			},
			{"return": "results"}
		]
	}`, nil)

	m, ok := result.Value.(map[string]any)
	if !ok {
		t.Fatalf("expected map, got %T", result.Value)
	}
	if m["fail"] != nil {
		t.Errorf("expected fail=nil, got %v", m["fail"])
	}
	if !numEq(m["ok"], 42) {
		t.Errorf("expected ok=42, got %v", m["ok"])
	}
}

func TestParallel_Collect_OnError(t *testing.T) {
	result := compileAndRun(t, `{
		"steps": [
			{
				"parallel": {
					"fail": [{"error": "'oops'"}],
					"ok": [{"return": 42}]
				},
				"on_error": "collect",
				"into": "results"
			},
			{"return": "results"}
		]
	}`, nil)

	m, ok := result.Value.(map[string]any)
	if !ok {
		t.Fatalf("expected map, got %T", result.Value)
	}
	failResult, ok := m["fail"].(map[string]any)
	if !ok {
		t.Fatalf("expected fail to be error map, got %T", m["fail"])
	}
	if failResult["error"] != true {
		t.Errorf("expected error=true, got %v", failResult["error"])
	}
	if !numEq(m["ok"], 42) {
		t.Errorf("expected ok=42, got %v", m["ok"])
	}
}

func TestParallel_JoinAny(t *testing.T) {
	result := compileAndRun(t, `{
		"steps": [
			{
				"parallel": {
					"fast": [{"return": "'first'"}],
					"slow": [{"return": "'second'"}]
				},
				"join": "any",
				"into": "results"
			},
			{"return": "results"}
		]
	}`, nil)

	m, ok := result.Value.(map[string]any)
	if !ok {
		t.Fatalf("expected map, got %T", result.Value)
	}
	hasResult := false
	for _, v := range m {
		if v != nil {
			hasResult = true
			break
		}
	}
	if !hasResult {
		t.Error("expected at least one result from join:any")
	}
}

func TestParallel_JoinSettled(t *testing.T) {
	result := compileAndRun(t, `{
		"steps": [
			{
				"parallel": {
					"fail": [{"error": "'branch error'"}],
					"ok": [{"return": 42}]
				},
				"join": "settled",
				"on_error": "cancel_all",
				"into": "results"
			},
			{"return": "results"}
		]
	}`, nil)

	m, ok := result.Value.(map[string]any)
	if !ok {
		t.Fatalf("expected map, got %T", result.Value)
	}
	if !numEq(m["ok"], 42) {
		t.Errorf("expected ok=42, got %v", m["ok"])
	}
	failResult, ok := m["fail"].(map[string]any)
	if !ok {
		t.Fatalf("expected fail to be error map, got %T (%v)", m["fail"], m["fail"])
	}
	if failResult["error"] != true {
		t.Errorf("expected error=true in settled fail result, got %v", failResult["error"])
	}
}

func TestParallel_JoinSettled_IgnoresOnError(t *testing.T) {
	result := compileAndRun(t, `{
		"steps": [
			{
				"parallel": {
					"fail": [{"error": "'oops'"}],
					"ok": [{"return": 99}]
				},
				"join": "settled",
				"on_error": "continue",
				"into": "results"
			},
			{"return": "results"}
		]
	}`, nil)

	m, ok := result.Value.(map[string]any)
	if !ok {
		t.Fatalf("expected map, got %T", result.Value)
	}
	failResult, ok := m["fail"].(map[string]any)
	if !ok {
		t.Fatalf("settled should collect error info even with on_error:continue, got %T (%v)", m["fail"], m["fail"])
	}
	if failResult["error"] != true {
		t.Errorf("expected error=true, got %v", failResult["error"])
	}
	if !numEq(m["ok"], 99) {
		t.Errorf("expected ok=99, got %v", m["ok"])
	}
}

func TestParallel_JoinSettled_AllFail(t *testing.T) {
	result := compileAndRun(t, `{
		"steps": [
			{
				"parallel": {
					"a": [{"error": "'err a'"}],
					"b": [{"error": "'err b'"}]
				},
				"join": "settled",
				"on_error": "cancel_all",
				"into": "results"
			},
			{"return": "results"}
		]
	}`, nil)

	m, ok := result.Value.(map[string]any)
	if !ok {
		t.Fatalf("expected map, got %T", result.Value)
	}
	for name, v := range m {
		errMap, ok := v.(map[string]any)
		if !ok {
			t.Errorf("branch '%s': expected error map, got %T", name, v)
			continue
		}
		if errMap["error"] != true {
			t.Errorf("branch '%s': expected error=true, got %v", name, errMap["error"])
		}
	}
}
