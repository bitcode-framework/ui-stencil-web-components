package lang

import (
	"strings"
	"testing"
)

func TestMethod_StepLevelCall(t *testing.T) {
	result := compileAndRun(t, `{
		"structs": {
			"Counter": {
				"fields": {
					"count": "int"
				},
				"methods": {
					"increment": {
						"steps": [
							{"set": "self.count", "expr": "self.count + 1"}
						]
					}
				}
			}
		},
		"steps": [
			{"let": "c", "new": "Counter", "with": {"count": "0"}},
			{"call": "c.increment"},
			{"call": "c.increment"},
			{"return": "c.count"}
		]
	}`, nil)

	if !numEq(result.Value, 2) {
		t.Errorf("expected 2, got %v", result.Value)
	}
}

func TestMethod_LetCallShorthand(t *testing.T) {
	result := compileAndRun(t, `{
		"structs": {
			"Person": {
				"fields": {
					"first": "string",
					"last": "string"
				},
				"methods": {
					"fullName": {
						"returns": "string",
						"steps": [
							{"return": "self.first + ' ' + self.last"}
						]
					}
				}
			}
		},
		"steps": [
			{"let": "p", "new": "Person", "with": {"first": "'Alice'", "last": "'Smith'"}},
			{"let": "name", "call": "p.fullName"},
			{"return": "name"}
		]
	}`, nil)

	if result.Value != "Alice Smith" {
		t.Errorf("expected 'Alice Smith', got %v", result.Value)
	}
}

func TestMethod_ExpressionLevelCall(t *testing.T) {
	result := compileAndRun(t, `{
		"structs": {
			"Person": {
				"fields": {
					"name": "string"
				},
				"methods": {
					"greet": {
						"params": {"greeting": "string"},
						"returns": "string",
						"steps": [
							{"return": "greeting + ', ' + self.name + '!'"}
						]
					}
				}
			}
		},
		"steps": [
			{"let": "p", "new": "Person", "with": {"name": "'Alice'"}},
			{"return": "p.greet('Hello')"}
		]
	}`, nil)

	if result.Value != "Hello, Alice!" {
		t.Errorf("expected 'Hello, Alice!', got %v", result.Value)
	}
}

func TestMethod_SelfMutation(t *testing.T) {
	result := compileAndRun(t, `{
		"structs": {
			"Person": {
				"fields": {
					"name": "string",
					"age": "int"
				},
				"methods": {
					"birthday": {
						"steps": [
							{"set": "self.age", "expr": "self.age + 1"}
						]
					}
				}
			}
		},
		"steps": [
			{"let": "p", "new": "Person", "with": {"name": "'Alice'", "age": "30"}},
			{"call": "p.birthday"},
			{"return": "p.age"}
		]
	}`, nil)

	if !numEq(result.Value, 31) {
		t.Errorf("expected 31, got %v", result.Value)
	}
}

func TestMethod_FrozenStruct_MutationBlocked(t *testing.T) {
	program, err := Parse([]byte(`{
		"structs": {
			"Point": {
				"frozen": true,
				"fields": {
					"x": "int",
					"y": "int"
				},
				"methods": {
					"moveX": {
						"steps": [
							{"set": "self.x", "expr": "self.x + 1"}
						]
					}
				}
			}
		},
		"steps": []
	}`))
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	engine := NewExprLangEngine()
	_, err = Compile(program, engine, DefaultLimits())
	if err == nil {
		t.Fatal("expected compile error for frozen struct mutation")
	}
	if !strings.Contains(err.Error(), "frozen") {
		t.Errorf("expected 'frozen' error, got: %v", err)
	}
}

func TestMethod_FrozenStruct_ReadOnlyAllowed(t *testing.T) {
	result := compileAndRun(t, `{
		"structs": {
			"Point": {
				"frozen": true,
				"fields": {
					"x": "int",
					"y": "int"
				},
				"methods": {
					"sum": {
						"returns": "int",
						"steps": [
							{"return": "self.x + self.y"}
						]
					}
				}
			}
		},
		"steps": [
			{"let": "p", "new": "Point", "with": {"x": "3", "y": "4"}},
			{"return": "p.sum()"}
		]
	}`, nil)

	if !numEq(result.Value, 7) {
		t.Errorf("expected 7, got %v", result.Value)
	}
}

func TestMethod_SelfMethodCall(t *testing.T) {
	result := compileAndRun(t, `{
		"structs": {
			"Calc": {
				"fields": {
					"value": "int"
				},
				"methods": {
					"double": {
						"returns": "int",
						"steps": [
							{"return": "self.value * 2"}
						]
					},
					"quadruple": {
						"returns": "int",
						"steps": [
							{"let": "d", "expr": "self.double()"},
							{"return": "d * 2"}
						]
					}
				}
			}
		},
		"steps": [
			{"let": "c", "new": "Calc", "with": {"value": "5"}},
			{"return": "c.quadruple()"}
		]
	}`, nil)

	if !numEq(result.Value, 20) {
		t.Errorf("expected 20, got %v", result.Value)
	}
}

func TestMethod_UndefinedMethod_Error(t *testing.T) {
	program, err := Parse([]byte(`{
		"structs": {
			"Person": {
				"fields": {"name": "string"}
			}
		},
		"steps": [
			{"let": "p", "new": "Person", "with": {"name": "'Alice'"}},
			{"call": "p.nonexistent"}
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
		t.Fatal("expected error for undefined method")
	}
}

func TestMethod_ReturnNewInstance(t *testing.T) {
	result := compileAndRun(t, `{
		"structs": {
			"Point": {
				"frozen": true,
				"fields": {
					"x": "int",
					"y": "int"
				},
				"methods": {
					"withX": {
						"params": {"newX": "int"},
						"returns": "Point",
						"steps": [
							{"return": {"new": "Point", "with": {"x": "newX", "y": "self.y"}}}
						]
					}
				}
			}
		},
		"steps": [
			{"let": "p", "new": "Point", "with": {"x": "1", "y": "2"}},
			{"let": "p2", "call": "p.withX", "with": {"newX": "10"}},
			{"return": {"with": {"old_x": "p.x", "new_x": "p2.x", "y": "p2.y"}}}
		]
	}`, nil)

	m, ok := result.Value.(map[string]any)
	if !ok {
		t.Fatalf("expected map, got %T", result.Value)
	}
	if !numEq(m["old_x"], 1) {
		t.Errorf("expected old_x=1, got %v", m["old_x"])
	}
	if !numEq(m["new_x"], 10) {
		t.Errorf("expected new_x=10, got %v", m["new_x"])
	}
	if !numEq(m["y"], 2) {
		t.Errorf("expected y=2, got %v", m["y"])
	}
}

func TestMethod_Chaining_MultipleCalls(t *testing.T) {
	result := compileAndRun(t, `{
		"structs": {
			"Counter": {
				"fields": {
					"value": "int"
				},
				"methods": {
					"add": {
						"params": {"n": "int"},
						"steps": [
							{"set": "self.value", "expr": "self.value + n"}
						]
					},
					"getResult": {
						"returns": "int",
						"steps": [
							{"return": "self.value"}
						]
					}
				}
			}
		},
		"steps": [
			{"let": "c", "new": "Counter", "with": {"value": "0"}},
			{"call": "c.add", "with": {"n": "10"}},
			{"call": "c.add", "with": {"n": "20"}},
			{"call": "c.add", "with": {"n": "30"}},
			{"return": "c.getResult()"}
		]
	}`, nil)

	if !numEq(result.Value, 60) {
		t.Errorf("expected 60, got %v", result.Value)
	}
}

func TestMethod_SelfReassign_InSwitch_CompileError(t *testing.T) {
	program, err := Parse([]byte(`{
		"structs": {
			"Obj": {
				"fields": {"x": "int"},
				"methods": {
					"bad": {
						"steps": [
							{"switch": "self.x", "cases": {
								"1": [{"set": "self", "value": "replaced"}]
							}}
						]
					}
				}
			}
		},
		"steps": []
	}`))
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	engine := NewExprLangEngine()
	_, err = Compile(program, engine, DefaultLimits())
	if err == nil {
		t.Fatal("expected compile error for self reassign in switch")
	}
	if !strings.Contains(err.Error(), "reassign self") {
		t.Errorf("expected 'reassign self' error, got: %v", err)
	}
}
