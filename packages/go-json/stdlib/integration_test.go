package stdlib

import (
	"math"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/expr-lang/expr"
)

func TestIntegration_StringFunctions(t *testing.T) {
	tests := []struct {
		expr     string
		expected any
	}{
		{`capitalize("hello")`, "Hello"},
		{`capitalize("")`, ""},
		{`capitalize("A")`, "A"},
		{`title("hello world")`, "Hello World"},
		{`camelCase("hello_world")`, "helloWorld"},
		{`camelCase("hello-world")`, "helloWorld"},
		{`snakeCase("helloWorld")`, "hello_world"},
		{`snakeCase("Hello World")`, "hello_world"},
		{`kebabCase("helloWorld")`, "hello-world"},
		{`pascalCase("hello_world")`, "HelloWorld"},
		{`truncate("hello world", 8)`, "hello..."},
		{`truncate("hi", 10)`, "hi"},
		{`truncate("hello world", 8, "…")`, "hello w…"},
		{`slugify("Hello World!")`, "hello-world"},
		{`slugify("  Multiple   Spaces  ")`, "multiple-spaces"},
		{`strReverse("hello")`, "olleh"},
		{`strReverse("")`, ""},
		{`strCount("hello", "l")`, 2},
		{`strCount("aaa", "a")`, 3},
		{`strCount("hello", "x")`, 0},
		{`replaceFirst("aaa", "a", "b")`, "baa"},
		{`isDigit("123")`, true},
		{`isDigit("12a")`, false},
		{`isDigit("")`, false},
		{`isAlpha("hello")`, true},
		{`isAlpha("hello1")`, false},
		{`isAlpha("")`, false},
		{`isAlphaNum("hello123")`, true},
		{`isAlphaNum("hello 123")`, false},
		{`isEmpty("")`, true},
		{`isEmpty("x")`, false},
		{`isBlank("  ")`, true},
		{`isBlank(" x ")`, false},
		{`isBlank("")`, true},
		{`escapeHTML("<b>hi</b>")`, "&lt;b&gt;hi&lt;/b&gt;"},
		{`unescapeHTML("&lt;b&gt;")`, "<b>"},
	}
	for _, tt := range tests {
		t.Run(tt.expr, func(t *testing.T) {
			got := evalExpr(t, tt.expr, nil)
			if got != tt.expected {
				t.Errorf("%s = %v (%T), want %v (%T)", tt.expr, got, got, tt.expected, tt.expected)
			}
		})
	}
}

func TestIntegration_StringLines(t *testing.T) {
	result := evalExpr(t, `lines("a\nb\nc")`, nil)
	arr := result.([]any)
	if len(arr) != 3 || arr[0] != "a" || arr[1] != "b" || arr[2] != "c" {
		t.Errorf("lines = %v, want [a, b, c]", arr)
	}
}

func TestIntegration_StringWords(t *testing.T) {
	result := evalExpr(t, `words("hello  world  go")`, nil)
	arr := result.([]any)
	if len(arr) != 3 || arr[0] != "hello" || arr[1] != "world" || arr[2] != "go" {
		t.Errorf("words = %v, want [hello, world, go]", arr)
	}
}

func TestIntegration_MathFunctions(t *testing.T) {
	tests := []struct {
		expr     string
		expected float64
		delta    float64
	}{
		{"sin(PI / 2)", 1.0, 1e-10},
		{"cos(0)", 1.0, 1e-10},
		{"tan(0)", 0.0, 1e-10},
		{"asin(1)", math.Pi / 2, 1e-10},
		{"acos(1)", 0.0, 1e-10},
		{"atan(1)", math.Pi / 4, 1e-10},
		{"atan2(1, 1)", math.Pi / 4, 1e-10},
		{"log(E)", 1.0, 1e-10},
		{"log2(8)", 3.0, 1e-10},
		{"log10(1000)", 3.0, 1e-10},
		{"exp(0)", 1.0, 1e-10},
		{"trunc(3.7)", 3.0, 1e-10},
		{"trunc(-3.7)", -3.0, 1e-10},
	}
	for _, tt := range tests {
		t.Run(tt.expr, func(t *testing.T) {
			got := evalExpr(t, tt.expr, nil)
			f, ok := got.(float64)
			if !ok {
				t.Fatalf("%s returned %T, want float64", tt.expr, got)
			}
			if math.Abs(f-tt.expected) > tt.delta {
				t.Errorf("%s = %v, want %v", tt.expr, f, tt.expected)
			}
		})
	}
}

func TestIntegration_MathConstants(t *testing.T) {
	pi := evalExpr(t, "PI", nil)
	if math.Abs(pi.(float64)-math.Pi) > 1e-10 {
		t.Errorf("PI = %v", pi)
	}

	e := evalExpr(t, "E", nil)
	if math.Abs(e.(float64)-math.E) > 1e-10 {
		t.Errorf("E = %v", e)
	}
}

func TestIntegration_MathTypeChecks(t *testing.T) {
	result := evalExpr(t, "isFinite(42)", nil)
	if result != true {
		t.Errorf("isFinite(42) = %v", result)
	}

	result = evalExpr(t, "isFinite(Infinity)", nil)
	if result != false {
		t.Errorf("isFinite(Infinity) = %v", result)
	}
}

func TestIntegration_Random(t *testing.T) {
	r1 := evalExpr(t, "random()", nil).(float64)
	if r1 < 0 || r1 >= 1 {
		t.Errorf("random() = %v, want [0, 1)", r1)
	}
}

func TestIntegration_DateTimeFunctions(t *testing.T) {
	env := map[string]any{
		"dt": time.Date(2024, 6, 15, 14, 30, 0, 0, time.UTC),
	}

	result := evalExpr(t, "toUnix(dt)", env)
	unix, _ := toFloat64(result)
	if int(unix) != 1718461800 {
		t.Errorf("toUnix = %v, want 1718461800", result)
	}

	result = evalExpr(t, "toISO(dt)", env)
	if result != "2024-06-15T14:30:00Z" {
		t.Errorf("toISO = %v", result)
	}

	result = evalExpr(t, "isWeekend(dt)", env)
	if result != true {
		t.Errorf("isWeekend(Saturday) = %v, want true", result)
	}

	result = evalExpr(t, "daysInMonth(dt)", env)
	days, _ := toFloat64(result)
	if int(days) != 30 {
		t.Errorf("daysInMonth(June) = %v, want 30", result)
	}

	result = evalExpr(t, "isLeapYear(dt)", env)
	if result != true {
		t.Errorf("isLeapYear(2024) = %v, want true", result)
	}
}

func TestIntegration_DateTimeFromUnix(t *testing.T) {
	result := evalExpr(t, "fromUnix(1704067200)", nil)
	dt, ok := result.(time.Time)
	if !ok {
		t.Fatalf("fromUnix returned %T, want time.Time", result)
	}
	if dt.Year() != 2024 || dt.Month() != 1 || dt.Day() != 1 {
		t.Errorf("fromUnix(1704067200) = %v, want 2024-01-01", dt)
	}
}

func TestIntegration_DateTimeStartEnd(t *testing.T) {
	env := map[string]any{
		"dt": time.Date(2024, 2, 15, 14, 30, 0, 0, time.UTC),
	}

	result := evalExpr(t, "startOfDay(dt)", env).(time.Time)
	if result.Hour() != 0 || result.Minute() != 0 || result.Second() != 0 {
		t.Errorf("startOfDay = %v", result)
	}

	result = evalExpr(t, "endOfDay(dt)", env).(time.Time)
	if result.Hour() != 23 || result.Minute() != 59 || result.Second() != 59 {
		t.Errorf("endOfDay = %v", result)
	}

	result = evalExpr(t, "startOfMonth(dt)", env).(time.Time)
	if result.Day() != 1 {
		t.Errorf("startOfMonth day = %d", result.Day())
	}

	result = evalExpr(t, "endOfMonth(dt)", env).(time.Time)
	if result.Day() != 29 {
		t.Errorf("endOfMonth(Feb 2024) day = %d, want 29", result.Day())
	}
}

func TestIntegration_DateTimeComparison(t *testing.T) {
	env := map[string]any{
		"a": time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		"b": time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC),
	}

	if evalExpr(t, "isBefore(a, b)", env) != true {
		t.Error("isBefore(jan, jun) should be true")
	}
	if evalExpr(t, "isAfter(a, b)", env) != false {
		t.Error("isAfter(jan, jun) should be false")
	}
}

func TestIntegration_ArrayFunctions(t *testing.T) {
	env := map[string]any{
		"arr":  []any{1, nil, 2, nil, 3},
		"nums": []any{1, 2, 3, 4, 5},
		"a":    []any{1, 2, 3, 4},
		"b":    []any{3, 4, 5, 6},
	}

	result := evalExpr(t, "compact(arr)", env).([]any)
	if len(result) != 3 {
		t.Errorf("compact = %v, want 3 items", result)
	}

	result2 := evalExpr(t, "includes(nums, 3)", env)
	if result2 != true {
		t.Error("includes([1..5], 3) should be true")
	}

	result2 = evalExpr(t, "includes(nums, 99)", env)
	if result2 != false {
		t.Error("includes([1..5], 99) should be false")
	}

	idx := evalExpr(t, "arrayIndexOf(nums, 3)", env)
	if idx != 2 {
		t.Errorf("arrayIndexOf = %v, want 2", idx)
	}

	idx = evalExpr(t, "arrayIndexOf(nums, 99)", env)
	if idx != -1 {
		t.Errorf("arrayIndexOf(missing) = %v, want -1", idx)
	}

	diff := evalExpr(t, "difference(a, b)", env).([]any)
	if len(diff) != 2 {
		t.Errorf("difference = %v, want [1,2]", diff)
	}

	inter := evalExpr(t, "intersection(a, b)", env).([]any)
	if len(inter) != 2 {
		t.Errorf("intersection = %v, want [3,4]", inter)
	}

	uni := evalExpr(t, "union(a, b)", env).([]any)
	if len(uni) != 6 {
		t.Errorf("union = %v, want 6 items", uni)
	}

	filled := evalExpr(t, "fill(3, 0)", env).([]any)
	if len(filled) != 3 {
		t.Errorf("fill(3, 0) = %v", filled)
	}

	dropped := evalExpr(t, "drop(nums, 2)", env).([]any)
	if len(dropped) != 3 {
		t.Errorf("drop(5, 2) = %v, want 3 items", dropped)
	}

	right := evalExpr(t, "takeRight(nums, 2)", env).([]any)
	if len(right) != 2 {
		t.Errorf("takeRight(5, 2) = %v, want 2 items", right)
	}
}

func TestIntegration_ArrayFlatMap(t *testing.T) {
	env := map[string]any{
		"users": []any{
			map[string]any{"name": "Alice", "tags": []any{"admin", "user"}},
			map[string]any{"name": "Bob", "tags": []any{"user"}},
			map[string]any{"name": "Charlie", "tags": []any{"moderator", "user"}},
		},
	}

	result := evalExpr(t, `flatMap(users, "tags")`, env).([]any)
	if len(result) != 5 {
		t.Errorf("flatMap = %v, want 5 tags", result)
	}
}

func TestIntegration_ArrayPartition(t *testing.T) {
	env := map[string]any{
		"users": []any{
			map[string]any{"name": "Alice", "active": true},
			map[string]any{"name": "Bob", "active": false},
			map[string]any{"name": "Charlie", "active": true},
		},
	}

	result := evalExpr(t, `partition(users, "active")`, env).([]any)
	matches := result[0].([]any)
	nonMatches := result[1].([]any)
	if len(matches) != 2 {
		t.Errorf("partition matches = %d, want 2", len(matches))
	}
	if len(nonMatches) != 1 {
		t.Errorf("partition nonMatches = %d, want 1", len(nonMatches))
	}
}

func TestIntegration_ArrayKeyBy(t *testing.T) {
	env := map[string]any{
		"users": []any{
			map[string]any{"id": "u1", "name": "Alice"},
			map[string]any{"id": "u2", "name": "Bob"},
		},
	}

	result := evalExpr(t, `keyBy(users, "id")`, env).(map[string]any)
	if len(result) != 2 {
		t.Errorf("keyBy = %v, want 2 entries", result)
	}
	u1 := result["u1"].(map[string]any)
	if u1["name"] != "Alice" {
		t.Errorf("keyBy[u1].name = %v, want Alice", u1["name"])
	}
}

func TestIntegration_MapFunctions(t *testing.T) {
	env := map[string]any{
		"obj": map[string]any{
			"a": map[string]any{"x": 1},
			"b": 2,
		},
		"base": map[string]any{"port": 8080, "host": "localhost"},
	}

	result := evalExpr(t, `deepClone(obj)`, env)
	cloned := result.(map[string]any)
	if cloned["b"] != float64(2) {
		t.Errorf("deepClone.b = %v", cloned["b"])
	}

	result = evalExpr(t, `deepEqual(obj, obj)`, env)
	if result != true {
		t.Error("deepEqual(obj, obj) should be true")
	}

	result = evalExpr(t, `setIn(base, "port", 9090)`, env)
	m := result.(map[string]any)
	if m["port"] != 9090 {
		t.Errorf("setIn port = %v, want 9090", m["port"])
	}

	result = evalExpr(t, `deleteIn(base, "host")`, env)
	m = result.(map[string]any)
	if _, exists := m["host"]; exists {
		t.Error("deleteIn should remove host")
	}

	result = evalExpr(t, `defaults(base, {"debug": true})`, env)
	m = result.(map[string]any)
	if m["debug"] != true {
		t.Error("defaults should add debug")
	}
	if m["port"] != 8080 {
		t.Errorf("defaults should keep port = 8080, got %v", m["port"])
	}
}

func TestIntegration_MapDeepMerge(t *testing.T) {
	env := map[string]any{
		"a": map[string]any{
			"db": map[string]any{"host": "localhost", "port": 5432},
		},
		"b": map[string]any{
			"db": map[string]any{"port": 3306, "name": "mydb"},
		},
	}

	result := evalExpr(t, "deepMerge(a, b)", env).(map[string]any)
	db := result["db"].(map[string]any)
	if db["host"] != "localhost" {
		t.Errorf("deepMerge db.host = %v", db["host"])
	}
	if db["port"] != 3306 {
		t.Errorf("deepMerge db.port = %v, want 3306 (b overrides)", db["port"])
	}
	if db["name"] != "mydb" {
		t.Errorf("deepMerge db.name = %v", db["name"])
	}
}

func TestIntegration_MapKeys(t *testing.T) {
	env := map[string]any{
		"obj": map[string]any{"hello_world": 1, "foo_bar": 2},
	}

	result := evalExpr(t, `mapKeys(obj, "camelCase")`, env).(map[string]any)
	if _, ok := result["helloWorld"]; !ok {
		t.Errorf("mapKeys camelCase: %v", result)
	}
	if _, ok := result["fooBar"]; !ok {
		t.Errorf("mapKeys camelCase: %v", result)
	}

	result = evalExpr(t, `mapKeys(obj, "upper")`, env).(map[string]any)
	if _, ok := result["HELLO_WORLD"]; !ok {
		t.Errorf("mapKeys upper: %v", result)
	}
}

func TestIntegration_MapValues(t *testing.T) {
	env := map[string]any{
		"obj": map[string]any{"a": "  hello  ", "b": "  world  "},
	}

	result := evalExpr(t, `mapValues(obj, "trim")`, env).(map[string]any)
	if result["a"] != "hello" || result["b"] != "world" {
		t.Errorf("mapValues trim: %v", result)
	}

	result = evalExpr(t, `mapValues(obj, "upper")`, env).(map[string]any)
	if result["a"] != "  HELLO  " || result["b"] != "  WORLD  " {
		t.Errorf("mapValues upper: %v", result)
	}
}

func TestIntegration_ValidateNamespace(t *testing.T) {
	tests := []struct {
		expr     string
		expected bool
	}{
		{`validate.isEmail("user@example.com")`, true},
		{`validate.isEmail("invalid")`, false},
		{`validate.isURL("https://example.com")`, true},
		{`validate.isURL("not-a-url")`, false},
		{`validate.isIP("192.168.1.1")`, true},
		{`validate.isIP("999.999.999.999")`, false},
		{`validate.isUUID("550e8400-e29b-41d4-a716-446655440000")`, true},
		{`validate.isUUID("not-uuid")`, false},
		{`validate.isJSON("{\"a\":1}")`, true},
		{`validate.isJSON("not json")`, false},
		{`validate.isNumeric("3.14")`, true},
		{`validate.isNumeric("abc")`, false},
		{`validate.isHexColor("#FF0000")`, true},
		{`validate.isHexColor("red")`, false},
		{`validate.isCreditCard("4111111111111111")`, true},
		{`validate.isCreditCard("1234567890")`, false},
	}
	for _, tt := range tests {
		t.Run(tt.expr, func(t *testing.T) {
			got := evalExpr(t, tt.expr, nil)
			if got != tt.expected {
				t.Errorf("%s = %v, want %v", tt.expr, got, tt.expected)
			}
		})
	}
}

func TestIntegration_FormatFunctions(t *testing.T) {
	tests := []struct {
		expr     string
		expected string
	}{
		{`toFixed(3.14159, 2)`, "3.14"},
		{`toFixed(3.14159, 0)`, "3"},
		{`toFixed(3.14159, 4)`, "3.1416"},
		{`formatNumber(1234567.89, 2)`, "1,234,567.89"},
		{`formatNumber(1000, 0)`, "1,000"},
		{`formatBytes(0)`, "0.00 B"},
		{`formatBytes(1024)`, "1.00 KB"},
		{`formatBytes(1048576)`, "1.00 MB"},
		{`formatPercent(0.156, 1)`, "15.6%"},
		{`formatPercent(1, 0)`, "100%"},
	}
	for _, tt := range tests {
		t.Run(tt.expr, func(t *testing.T) {
			got := evalExpr(t, tt.expr, nil)
			if got != tt.expected {
				t.Errorf("%s = %q, want %q", tt.expr, got, tt.expected)
			}
		})
	}
}

func TestIntegration_CryptoNamespace(t *testing.T) {
	hash := evalExpr(t, `crypto.sha256("hello")`, nil).(string)
	if len(hash) != 64 {
		t.Errorf("sha256 len = %d", len(hash))
	}

	hash = evalExpr(t, `crypto.sha512("hello")`, nil).(string)
	if len(hash) != 128 {
		t.Errorf("sha512 len = %d", len(hash))
	}

	uid := evalExpr(t, `crypto.uuid()`, nil).(string)
	if len(uid) != 36 {
		t.Errorf("uuid len = %d", len(uid))
	}

	randHex := evalExpr(t, `crypto.randomBytes(16)`, nil).(string)
	if len(randHex) != 32 {
		t.Errorf("randomBytes(16) hex len = %d, want 32", len(randHex))
	}
}

func TestIntegration_CryptoEncryptDecrypt(t *testing.T) {
	env := map[string]any{}
	encrypted := evalExpr(t, `crypto.encrypt("secret data", "my-key-123")`, env).(string)
	if encrypted == "secret data" {
		t.Error("encrypted should differ from plaintext")
	}

	env["encrypted"] = encrypted
	decrypted := evalExpr(t, `crypto.decrypt(encrypted, "my-key-123")`, env).(string)
	if decrypted != "secret data" {
		t.Errorf("decrypt = %q, want 'secret data'", decrypted)
	}
}

func TestIntegration_CryptoPassword(t *testing.T) {
	hash := evalExpr(t, `crypto.hashPassword("mypass")`, nil).(string)
	if !strings.HasPrefix(hash, "$2a$") {
		t.Errorf("hash prefix wrong: %s", hash[:10])
	}

	env := map[string]any{"hash": hash}
	valid := evalExpr(t, `crypto.verifyPassword("mypass", hash)`, env)
	if valid != true {
		t.Error("verifyPassword correct should be true")
	}

	invalid := evalExpr(t, `crypto.verifyPassword("wrong", hash)`, env)
	if invalid != false {
		t.Error("verifyPassword wrong should be false")
	}
}

func TestIntegration_EnvFunction(t *testing.T) {
	os.Setenv("GO_JSON_INTEGRATION_TEST", "integration_value")
	defer os.Unsetenv("GO_JSON_INTEGRATION_TEST")

	result := evalExpr(t, `env("GO_JSON_INTEGRATION_TEST")`, nil)
	if result != "integration_value" {
		t.Errorf("env = %v, want 'integration_value'", result)
	}

	result = evalExpr(t, `env("NONEXISTENT_XYZ_123", "fallback")`, nil)
	if result != "fallback" {
		t.Errorf("env with default = %v, want 'fallback'", result)
	}
}

func TestIntegration_EnvWithAccessControl(t *testing.T) {
	reg := DefaultRegistryWithEnv(nil, &EnvAccessConfig{
		Deny: []string{"*_SECRET"},
	})
	opts := reg.All()
	env := make(map[string]any)
	for k, v := range reg.EnvVars() {
		env[k] = v
	}
	opts = append(opts, expr.Env(env))

	program, err := expr.Compile(`env("JWT_SECRET")`, opts...)
	if err != nil {
		t.Fatalf("compile error: %v", err)
	}
	_, err = expr.Run(program, env)
	if err == nil {
		t.Fatal("expected access denied error")
	}
	if !strings.Contains(err.Error(), "access denied") {
		t.Errorf("error = %v, want 'access denied'", err)
	}
}
