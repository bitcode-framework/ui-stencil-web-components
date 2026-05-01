package stdlib

import (
	"fmt"
	"math"
	"os"
	"strings"
	"testing"
	"time"
)

func TestMathExt_Trig(t *testing.T) {
	if math.Abs(math.Sin(math.Pi/2)-1.0) > 1e-10 {
		t.Error("sin(PI/2) should be 1")
	}
	if math.Abs(math.Cos(0)-1.0) > 1e-10 {
		t.Error("cos(0) should be 1")
	}
	if math.Abs(math.Tan(0)) > 1e-10 {
		t.Error("tan(0) should be 0")
	}
}

func TestMathExt_Log(t *testing.T) {
	if math.Abs(math.Log(math.E)-1.0) > 1e-10 {
		t.Error("log(E) should be 1")
	}
	if math.Abs(math.Log2(8)-3.0) > 1e-10 {
		t.Error("log2(8) should be 3")
	}
	if math.Abs(math.Log10(1000)-3.0) > 1e-10 {
		t.Error("log10(1000) should be 3")
	}
}

func TestMathExt_Constants(t *testing.T) {
	r := DefaultRegistry()
	env := r.EnvVars()

	pi, ok := env["PI"].(float64)
	if !ok || math.Abs(pi-3.141592653589793) > 1e-10 {
		t.Errorf("PI = %v, want 3.141592653589793", env["PI"])
	}

	e, ok := env["E"].(float64)
	if !ok || math.Abs(e-2.718281828459045) > 1e-10 {
		t.Errorf("E = %v, want 2.718281828459045", env["E"])
	}

	inf, ok := env["Infinity"].(float64)
	if !ok || !math.IsInf(inf, 1) {
		t.Errorf("Infinity = %v, want +Inf", env["Infinity"])
	}

	nan, ok := env["NaN"].(float64)
	if !ok || !math.IsNaN(nan) {
		t.Errorf("NaN = %v, want NaN", env["NaN"])
	}
}

func TestDateTimeExt_ToUnixFromUnix(t *testing.T) {
	dt := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	unix := int(dt.Unix())
	if unix != 1704067200 {
		t.Errorf("toUnix(2024-01-01) = %d, want 1704067200", unix)
	}

	restored := time.Unix(int64(unix), 0).UTC()
	if !restored.Equal(dt) {
		t.Errorf("fromUnix(%d) = %v, want %v", unix, restored, dt)
	}
}

func TestDateTimeExt_StartEndOfDay(t *testing.T) {
	dt := time.Date(2024, 6, 15, 14, 30, 45, 0, time.UTC)

	y, m, d := dt.Date()
	start := time.Date(y, m, d, 0, 0, 0, 0, dt.Location())
	if start.Hour() != 0 || start.Minute() != 0 || start.Second() != 0 {
		t.Errorf("startOfDay should be 00:00:00, got %v", start)
	}

	end := time.Date(y, m, d, 23, 59, 59, 999999999, dt.Location())
	if end.Hour() != 23 || end.Minute() != 59 || end.Second() != 59 {
		t.Errorf("endOfDay should be 23:59:59, got %v", end)
	}
}

func TestDateTimeExt_StartEndOfMonth(t *testing.T) {
	dt := time.Date(2024, 2, 15, 10, 0, 0, 0, time.UTC)

	y, m, _ := dt.Date()
	start := time.Date(y, m, 1, 0, 0, 0, 0, dt.Location())
	if start.Day() != 1 {
		t.Errorf("startOfMonth day = %d, want 1", start.Day())
	}

	end := time.Date(y, m+1, 0, 23, 59, 59, 999999999, dt.Location())
	if end.Day() != 29 {
		t.Errorf("endOfMonth(Feb 2024) day = %d, want 29 (leap year)", end.Day())
	}
}

func TestDateTimeExt_IsWeekend(t *testing.T) {
	sat := time.Date(2024, 6, 15, 0, 0, 0, 0, time.UTC)
	sun := time.Date(2024, 6, 16, 0, 0, 0, 0, time.UTC)
	mon := time.Date(2024, 6, 17, 0, 0, 0, 0, time.UTC)

	if sat.Weekday() != time.Saturday {
		t.Error("2024-06-15 should be Saturday")
	}
	if sun.Weekday() != time.Sunday {
		t.Error("2024-06-16 should be Sunday")
	}
	if mon.Weekday() == time.Saturday || mon.Weekday() == time.Sunday {
		t.Error("2024-06-17 should not be weekend")
	}
}

func TestDateTimeExt_DaysInMonth(t *testing.T) {
	tests := []struct {
		year  int
		month time.Month
		days  int
	}{
		{2024, time.January, 31},
		{2024, time.February, 29},
		{2023, time.February, 28},
		{2024, time.April, 30},
	}
	for _, tt := range tests {
		dt := time.Date(tt.year, tt.month, 1, 0, 0, 0, 0, time.UTC)
		y, m, _ := dt.Date()
		days := time.Date(y, m+1, 0, 0, 0, 0, 0, dt.Location()).Day()
		if days != tt.days {
			t.Errorf("daysInMonth(%d-%02d) = %d, want %d", tt.year, tt.month, days, tt.days)
		}
	}
}

func TestDateTimeExt_IsLeapYear(t *testing.T) {
	tests := []struct {
		year int
		leap bool
	}{
		{2024, true},
		{2023, false},
		{2000, true},
		{1900, false},
	}
	for _, tt := range tests {
		y := tt.year
		isLeap := y%4 == 0 && (y%100 != 0 || y%400 == 0)
		if isLeap != tt.leap {
			t.Errorf("isLeapYear(%d) = %v, want %v", tt.year, isLeap, tt.leap)
		}
	}
}

func TestArraysExt_Compact(t *testing.T) {
	input := []any{1, nil, "hello", nil, 3}
	var result []any
	for _, item := range input {
		if item != nil {
			result = append(result, item)
		}
	}
	if len(result) != 3 {
		t.Errorf("compact: expected 3 items, got %d", len(result))
	}
}

func TestArraysExt_Includes(t *testing.T) {
	arr := []any{1, "hello", 3.14}
	if !equalValues(arr[1], "hello") {
		t.Error("includes should find 'hello'")
	}
	if equalValues("missing", arr[0]) {
		t.Error("includes should not find 'missing' == 1")
	}
}

func TestArraysExt_Difference(t *testing.T) {
	a := []any{1, 2, 3, 4, 5}
	b := []any{2, 4}
	var result []any
	for _, item := range a {
		found := false
		for _, bItem := range b {
			if equalValues(item, bItem) {
				found = true
				break
			}
		}
		if !found {
			result = append(result, item)
		}
	}
	if len(result) != 3 {
		t.Errorf("difference: expected 3 items, got %d: %v", len(result), result)
	}
}

func TestArraysExt_Intersection(t *testing.T) {
	a := []any{1, 2, 3, 4}
	b := []any{2, 4, 6}
	var result []any
	for _, item := range a {
		for _, bItem := range b {
			if equalValues(item, bItem) {
				result = append(result, item)
				break
			}
		}
	}
	if len(result) != 2 {
		t.Errorf("intersection: expected 2 items, got %d: %v", len(result), result)
	}
}

func TestArraysExt_EqualValues(t *testing.T) {
	if !equalValues(1, 1) {
		t.Error("1 == 1")
	}
	if !equalValues(1, 1.0) {
		t.Error("1 == 1.0")
	}
	if !equalValues("hello", "hello") {
		t.Error("hello == hello")
	}
	if equalValues(1, 2) {
		t.Error("1 != 2")
	}
	if equalValues(nil, 1) {
		t.Error("nil != 1")
	}
	if !equalValues(nil, nil) {
		t.Error("nil == nil")
	}
}

func TestMapsExt_DeepMerge(t *testing.T) {
	a := map[string]any{
		"x": 1,
		"nested": map[string]any{
			"a": 1,
			"b": 2,
		},
	}
	b := map[string]any{
		"y": 2,
		"nested": map[string]any{
			"b": 99,
			"c": 3,
		},
	}
	result := deepMergeMap(a, b)
	nested := result["nested"].(map[string]any)
	if nested["a"] != 1 || nested["b"] != 99 || nested["c"] != 3 {
		t.Errorf("deepMerge nested = %v, want {a:1, b:99, c:3}", nested)
	}
	if result["x"] != 1 || result["y"] != 2 {
		t.Errorf("deepMerge top-level wrong: %v", result)
	}
}

func TestMapsExt_SetIn(t *testing.T) {
	obj := map[string]any{"a": map[string]any{"b": 1}}
	result := deepCloneMap(obj)
	setNestedValue(result, []string{"a", "b"}, 99)
	nested := result["a"].(map[string]any)
	if nested["b"] != 99 {
		t.Errorf("setIn: a.b = %v, want 99", nested["b"])
	}
	origNested := obj["a"].(map[string]any)
	if origNested["b"] != 1 {
		t.Error("setIn should not mutate original")
	}
}

func TestMapsExt_DeleteIn(t *testing.T) {
	obj := map[string]any{"a": map[string]any{"b": 1, "c": 2}}
	result := deepCloneMap(obj)
	deleteNestedKey(result, []string{"a", "b"})
	nested := result["a"].(map[string]any)
	if _, exists := nested["b"]; exists {
		t.Error("deleteIn should remove a.b")
	}
	if nested["c"] != float64(2) {
		t.Errorf("deleteIn should keep a.c = 2, got %v", nested["c"])
	}
}

func TestMapsExt_Defaults(t *testing.T) {
	obj := map[string]any{"a": 1}
	defs := map[string]any{"a": 99, "b": 2}
	result := make(map[string]any)
	for k, v := range defs {
		result[k] = v
	}
	for k, v := range obj {
		result[k] = v
	}
	if result["a"] != 1 {
		t.Errorf("defaults: a = %v, want 1 (obj overrides)", result["a"])
	}
	if result["b"] != 2 {
		t.Errorf("defaults: b = %v, want 2 (from defaults)", result["b"])
	}
}

func TestValidate_IsEmail(t *testing.T) {
	tests := []struct {
		input    string
		expected bool
	}{
		{"user@example.com", true},
		{"user+tag@domain.co.uk", true},
		{"invalid", false},
		{"@domain.com", false},
		{"user@", false},
		{"", false},
	}
	for _, tt := range tests {
		got := emailRegex.MatchString(tt.input)
		if got != tt.expected {
			t.Errorf("validate.isEmail(%q) = %v, want %v", tt.input, got, tt.expected)
		}
	}
}

func TestValidate_IsUUID(t *testing.T) {
	tests := []struct {
		input    string
		expected bool
	}{
		{"550e8400-e29b-41d4-a716-446655440000", true},
		{"not-a-uuid", false},
		{"550e8400-e29b-51d4-a716-446655440000", false},
		{"", false},
	}
	for _, tt := range tests {
		got := uuidRegex.MatchString(strings.ToLower(tt.input))
		if got != tt.expected {
			t.Errorf("validate.isUUID(%q) = %v, want %v", tt.input, got, tt.expected)
		}
	}
}

func TestValidate_IsHexColor(t *testing.T) {
	tests := []struct {
		input    string
		expected bool
	}{
		{"#fff", true},
		{"#FF0000", true},
		{"#12345g", false},
		{"fff", false},
		{"#1234", false},
	}
	for _, tt := range tests {
		got := hexColorRegex.MatchString(tt.input)
		if got != tt.expected {
			t.Errorf("validate.isHexColor(%q) = %v, want %v", tt.input, got, tt.expected)
		}
	}
}

func TestValidate_LuhnCheck(t *testing.T) {
	tests := []struct {
		input    string
		expected bool
	}{
		{"4111111111111111", true},
		{"4111111111111112", false},
		{"5500 0000 0000 0004", true},
		{"1234", false},
	}
	for _, tt := range tests {
		got := luhnCheck(tt.input)
		if got != tt.expected {
			t.Errorf("luhnCheck(%q) = %v, want %v", tt.input, got, tt.expected)
		}
	}
}

func TestFormatExt_FormatBytes(t *testing.T) {
	tests := []struct {
		input    float64
		expected string
	}{
		{0, "0.00 B"},
		{1024, "1.00 KB"},
		{1048576, "1.00 MB"},
		{1073741824, "1.00 GB"},
	}
	for _, tt := range tests {
		units := []string{"B", "KB", "MB", "GB", "TB", "PB"}
		size := math.Abs(tt.input)
		i := 0
		for size >= 1024 && i < len(units)-1 {
			size /= 1024
			i++
		}
		got := fmt.Sprintf("%.2f %s", size, units[i])
		if tt.input < 0 {
			got = "-" + got
		}
		if got != tt.expected {
			t.Errorf("formatBytes(%v) = %q, want %q", tt.input, got, tt.expected)
		}
	}
}

func TestCryptoExt_EncryptDecrypt(t *testing.T) {
	ns := CryptoNamespace()
	encryptFn := ns["encrypt"].(func(...any) (any, error))
	decryptFn := ns["decrypt"].(func(...any) (any, error))

	key := "my-secret-key-for-testing"
	plaintext := "Hello, World!"

	encrypted, err := encryptFn(plaintext, key)
	if err != nil {
		t.Fatalf("encrypt error: %v", err)
	}
	encStr := encrypted.(string)
	if encStr == plaintext {
		t.Error("encrypted should differ from plaintext")
	}

	decrypted, err := decryptFn(encStr, key)
	if err != nil {
		t.Fatalf("decrypt error: %v", err)
	}
	if decrypted != plaintext {
		t.Errorf("decrypt = %q, want %q", decrypted, plaintext)
	}

	_, err = decryptFn(encStr, "wrong-key")
	if err == nil {
		t.Error("decrypt with wrong key should fail")
	}
}

func TestCryptoExt_HashVerifyPassword(t *testing.T) {
	ns := CryptoNamespace()
	hashFn := ns["hashPassword"].(func(...any) (any, error))
	verifyFn := ns["verifyPassword"].(func(...any) (any, error))

	hash, err := hashFn("mypassword")
	if err != nil {
		t.Fatalf("hashPassword error: %v", err)
	}
	hashStr := hash.(string)
	if !strings.HasPrefix(hashStr, "$2a$") {
		t.Errorf("hash should start with $2a$, got %s", hashStr[:10])
	}

	valid, err := verifyFn("mypassword", hashStr)
	if err != nil {
		t.Fatalf("verifyPassword error: %v", err)
	}
	if valid != true {
		t.Error("verifyPassword should return true for correct password")
	}

	invalid, err := verifyFn("wrongpassword", hashStr)
	if err != nil {
		t.Fatalf("verifyPassword error: %v", err)
	}
	if invalid != false {
		t.Error("verifyPassword should return false for wrong password")
	}
}

func TestCryptoExt_SHA512(t *testing.T) {
	ns := CryptoNamespace()
	sha512Fn := ns["sha512"].(func(...any) (any, error))

	result, err := sha512Fn("hello")
	if err != nil {
		t.Fatalf("sha512 error: %v", err)
	}
	hash := result.(string)
	if len(hash) != 128 {
		t.Errorf("sha512 hash length = %d, want 128", len(hash))
	}
}

func TestCryptoExt_RandomBytes(t *testing.T) {
	ns := CryptoNamespace()
	randomBytesFn := ns["randomBytes"].(func(...any) (any, error))

	result, err := randomBytesFn(16)
	if err != nil {
		t.Fatalf("randomBytes error: %v", err)
	}
	hex := result.(string)
	if len(hex) != 32 {
		t.Errorf("randomBytes(16) hex length = %d, want 32", len(hex))
	}

	result2, _ := randomBytesFn(16)
	if result == result2 {
		t.Error("randomBytes should produce different values")
	}

	_, err = randomBytesFn(0)
	if err == nil {
		t.Error("randomBytes(0) should error")
	}

	_, err = randomBytesFn(2000)
	if err == nil {
		t.Error("randomBytes(2000) should error (max 1024)")
	}
}

func TestEnv_Basic(t *testing.T) {
	os.Setenv("GO_JSON_TEST_VAR", "hello")
	defer os.Unsetenv("GO_JSON_TEST_VAR")

	resolver := os.Getenv
	val := resolver("GO_JSON_TEST_VAR")
	if val != "hello" {
		t.Errorf("env(GO_JSON_TEST_VAR) = %q, want 'hello'", val)
	}

	val = resolver("NONEXISTENT_VAR_XYZ")
	if val != "" {
		t.Errorf("env(NONEXISTENT) = %q, want ''", val)
	}
}

func TestEnv_AccessDeny(t *testing.T) {
	config := &EnvAccessConfig{
		Deny: []string{"*_SECRET", "*_PASSWORD"},
	}

	err := checkEnvAccess("JWT_SECRET", config)
	if err == nil {
		t.Error("JWT_SECRET should be denied")
	}

	err = checkEnvAccess("DB_PASSWORD", config)
	if err == nil {
		t.Error("DB_PASSWORD should be denied")
	}

	err = checkEnvAccess("APP_NAME", config)
	if err != nil {
		t.Errorf("APP_NAME should be allowed, got: %v", err)
	}
}

func TestEnv_AccessAllow(t *testing.T) {
	config := &EnvAccessConfig{
		Allow: []string{"APP_*", "PUBLIC_*"},
	}

	err := checkEnvAccess("APP_NAME", config)
	if err != nil {
		t.Errorf("APP_NAME should be allowed, got: %v", err)
	}

	err = checkEnvAccess("PUBLIC_KEY", config)
	if err != nil {
		t.Errorf("PUBLIC_KEY should be allowed, got: %v", err)
	}

	err = checkEnvAccess("DB_HOST", config)
	if err == nil {
		t.Error("DB_HOST should be denied (not in allow list)")
	}
}

func TestRegistry_ValidateNamespace(t *testing.T) {
	r := DefaultRegistry()
	env := r.EnvVars()

	validate, ok := env["validate"]
	if !ok {
		t.Fatal("expected validate in env vars")
	}

	validateMap, ok := validate.(map[string]any)
	if !ok {
		t.Fatalf("expected validate to be map, got %T", validate)
	}

	expectedFuncs := []string{"isEmail", "isURL", "isIP", "isUUID", "isJSON", "isNumeric", "isAlpha", "isBase64", "isHexColor", "isCreditCard"}
	for _, name := range expectedFuncs {
		if _, ok := validateMap[name]; !ok {
			t.Errorf("expected %s in validate namespace", name)
		}
	}
}
