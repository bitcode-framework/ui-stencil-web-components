package server

import (
	"encoding/base64"
	"os"
	"testing"

	"github.com/bitcode-framework/go-json/lang"
	"github.com/bitcode-framework/go-json/server/adapters"
)

func TestAPIKeyStrategy_HeaderValid(t *testing.T) {
	const envKey = "TEST_API_KEYS_HEADER_VALID"
	os.Setenv(envKey, "abc123:myapp,def456:otherapp")
	t.Cleanup(func() { os.Unsetenv(envKey) })

	s := &APIKeyStrategy{
		name:    "apikey",
		header:  "X-API-Key",
		keysEnv: envKey,
	}

	ctx := &adapters.RequestContext{
		Headers: map[string]string{"X-API-Key": "abc123"},
		Query:   map[string]string{},
	}

	result, err := s.Authenticate(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	m, ok := result.(map[string]any)
	if !ok {
		t.Fatalf("expected map result, got %T", result)
	}
	if m["key"] != "abc123" {
		t.Errorf("expected key=abc123, got %v", m["key"])
	}
	if m["name"] != "myapp" {
		t.Errorf("expected name=myapp, got %v", m["name"])
	}
}

func TestAPIKeyStrategy_QueryParamValid(t *testing.T) {
	const envKey = "TEST_API_KEYS_QUERY_VALID"
	os.Setenv(envKey, "qkey1:service1")
	t.Cleanup(func() { os.Unsetenv(envKey) })

	s := &APIKeyStrategy{
		name:       "apikey",
		header:     "X-API-Key",
		queryParam: "api_key",
		keysEnv:    envKey,
	}

	ctx := &adapters.RequestContext{
		Headers: map[string]string{},
		Query:   map[string]string{"api_key": "qkey1"},
	}

	result, err := s.Authenticate(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	m, ok := result.(map[string]any)
	if !ok {
		t.Fatalf("expected map result, got %T", result)
	}
	if m["key"] != "qkey1" {
		t.Errorf("expected key=qkey1, got %v", m["key"])
	}
	if m["name"] != "service1" {
		t.Errorf("expected name=service1, got %v", m["name"])
	}
}

func TestAPIKeyStrategy_MissingKey(t *testing.T) {
	const envKey = "TEST_API_KEYS_MISSING"
	os.Setenv(envKey, "abc:name")
	t.Cleanup(func() { os.Unsetenv(envKey) })

	s := &APIKeyStrategy{
		name:    "apikey",
		header:  "X-API-Key",
		keysEnv: envKey,
	}

	ctx := &adapters.RequestContext{
		Headers: map[string]string{},
		Query:   map[string]string{},
	}

	_, err := s.Authenticate(ctx)
	if err == nil {
		t.Fatal("expected error for missing key")
	}
	if err.Error() != "missing API key" {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestAPIKeyStrategy_InvalidKey(t *testing.T) {
	const envKey = "TEST_API_KEYS_INVALID"
	os.Setenv(envKey, "validkey:myapp")
	t.Cleanup(func() { os.Unsetenv(envKey) })

	s := &APIKeyStrategy{
		name:    "apikey",
		header:  "X-API-Key",
		keysEnv: envKey,
	}

	ctx := &adapters.RequestContext{
		Headers: map[string]string{"X-API-Key": "wrongkey"},
		Query:   map[string]string{},
	}

	_, err := s.Authenticate(ctx)
	if err == nil {
		t.Fatal("expected error for invalid key")
	}
	if err.Error() != "invalid API key" {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestAPIKeyStrategy_KeyWithoutName(t *testing.T) {
	const envKey = "TEST_API_KEYS_NO_NAME"
	os.Setenv(envKey, "simplekey")
	t.Cleanup(func() { os.Unsetenv(envKey) })

	s := &APIKeyStrategy{
		name:    "apikey",
		header:  "X-API-Key",
		keysEnv: envKey,
	}

	ctx := &adapters.RequestContext{
		Headers: map[string]string{"X-API-Key": "simplekey"},
		Query:   map[string]string{},
	}

	result, err := s.Authenticate(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	m, ok := result.(map[string]any)
	if !ok {
		t.Fatalf("expected map result, got %T", result)
	}
	if m["key"] != "simplekey" {
		t.Errorf("expected key=simplekey, got %v", m["key"])
	}
	if _, hasName := m["name"]; hasName {
		t.Errorf("expected no name field, got %v", m["name"])
	}
}

func TestBasicStrategy_Valid(t *testing.T) {
	const envKey = "TEST_BASIC_USERS_VALID"
	os.Setenv(envKey, "admin:secret123,reader:pass456")
	t.Cleanup(func() { os.Unsetenv(envKey) })

	s := &BasicStrategy{
		name:     "basic",
		usersEnv: envKey,
	}

	creds := base64.StdEncoding.EncodeToString([]byte("admin:secret123"))
	ctx := &adapters.RequestContext{
		Headers: map[string]string{"Authorization": "Basic " + creds},
	}

	result, err := s.Authenticate(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	m, ok := result.(map[string]any)
	if !ok {
		t.Fatalf("expected map result, got %T", result)
	}
	if m["username"] != "admin" {
		t.Errorf("expected username=admin, got %v", m["username"])
	}
}

func TestBasicStrategy_MissingHeader(t *testing.T) {
	const envKey = "TEST_BASIC_USERS_MISSING"
	os.Setenv(envKey, "admin:pass")
	t.Cleanup(func() { os.Unsetenv(envKey) })

	s := &BasicStrategy{
		name:     "basic",
		usersEnv: envKey,
	}

	ctx := &adapters.RequestContext{
		Headers: map[string]string{},
	}

	_, err := s.Authenticate(ctx)
	if err == nil {
		t.Fatal("expected error for missing header")
	}
	if err.Error() != "missing basic auth credentials" {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestBasicStrategy_InvalidEncoding(t *testing.T) {
	const envKey = "TEST_BASIC_USERS_BADENC"
	os.Setenv(envKey, "admin:pass")
	t.Cleanup(func() { os.Unsetenv(envKey) })

	s := &BasicStrategy{
		name:     "basic",
		usersEnv: envKey,
	}

	ctx := &adapters.RequestContext{
		Headers: map[string]string{"Authorization": "Basic !!!not-valid-base64!!!"},
	}

	_, err := s.Authenticate(ctx)
	if err == nil {
		t.Fatal("expected error for invalid encoding")
	}
	if err.Error() != "invalid basic auth encoding" {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestBasicStrategy_WrongCredentials(t *testing.T) {
	const envKey = "TEST_BASIC_USERS_WRONG"
	os.Setenv(envKey, "admin:correctpass")
	t.Cleanup(func() { os.Unsetenv(envKey) })

	s := &BasicStrategy{
		name:     "basic",
		usersEnv: envKey,
	}

	creds := base64.StdEncoding.EncodeToString([]byte("admin:wrongpass"))
	ctx := &adapters.RequestContext{
		Headers: map[string]string{"Authorization": "Basic " + creds},
	}

	_, err := s.Authenticate(ctx)
	if err == nil {
		t.Fatal("expected error for wrong credentials")
	}
	if err.Error() != "invalid credentials" {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestAuthRegistry_DefaultStrategy(t *testing.T) {
	const envKey = "TEST_REGISTRY_DEFAULT_KEYS"
	os.Setenv(envKey, "testkey:testname")
	t.Cleanup(func() { os.Unsetenv(envKey) })

	cfg := &lang.AuthConfig{
		Default: "myapikey",
		Strategies: map[string]*lang.StrategyConfig{
			"myapikey": {
				Type:    "apikey",
				Header:  "X-API-Key",
				KeysEnv: envKey,
			},
		},
	}

	ar := NewAuthRegistry(cfg, nil, nil, nil)

	mw, err := ar.Middleware("auth")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	ctx := &adapters.RequestContext{
		Headers: map[string]string{"X-API-Key": "testkey"},
		Query:   map[string]string{},
	}

	resp := mw(ctx, func() *adapters.Response {
		return &adapters.Response{Status: 200, Body: "ok"}
	})

	if resp.Status != 200 {
		t.Errorf("expected 200, got %d", resp.Status)
	}
	if ctx.User == nil {
		t.Error("expected user to be set on context")
	}
}

func TestAuthRegistry_NamedStrategy(t *testing.T) {
	const envKey = "TEST_REGISTRY_NAMED_KEYS"
	os.Setenv(envKey, "namedkey:svc")
	t.Cleanup(func() { os.Unsetenv(envKey) })

	cfg := &lang.AuthConfig{
		Default: "other",
		Strategies: map[string]*lang.StrategyConfig{
			"other": {
				Type:    "basic",
				UsersEnv: "NONEXISTENT_ENV",
			},
			"apikey": {
				Type:       "apikey",
				Header:     "X-API-Key",
				QueryParam: "key",
				KeysEnv:    envKey,
			},
		},
	}

	ar := NewAuthRegistry(cfg, nil, nil, nil)

	mw, err := ar.Middleware("auth:apikey")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	ctx := &adapters.RequestContext{
		Headers: map[string]string{"X-API-Key": "namedkey"},
		Query:   map[string]string{},
	}

	resp := mw(ctx, func() *adapters.Response {
		return &adapters.Response{Status: 200, Body: "ok"}
	})

	if resp.Status != 200 {
		t.Errorf("expected 200, got %d", resp.Status)
	}

	m, ok := ctx.User.(map[string]any)
	if !ok {
		t.Fatalf("expected map user, got %T", ctx.User)
	}
	if m["name"] != "svc" {
		t.Errorf("expected name=svc, got %v", m["name"])
	}
}

func TestAuthRegistry_UnknownStrategy(t *testing.T) {
	cfg := &lang.AuthConfig{
		Default: "myapikey",
		Strategies: map[string]*lang.StrategyConfig{
			"myapikey": {
				Type:    "apikey",
				Header:  "X-API-Key",
				KeysEnv: "SOME_ENV",
			},
		},
	}

	ar := NewAuthRegistry(cfg, nil, nil, nil)

	_, err := ar.Middleware("auth:nonexistent")
	if err == nil {
		t.Fatal("expected error for unknown strategy")
	}
	expected := `auth strategy "nonexistent" not found`
	if err.Error() != expected {
		t.Errorf("expected %q, got %q", expected, err.Error())
	}
}

func TestAuthRegistry_JWTFallback(t *testing.T) {
	jwtCfg := &lang.JWTConfig{
		SecretEnv: "TEST_JWT_SECRET_FALLBACK",
		Algorithm: "HS256",
	}

	ar := NewAuthRegistry(nil, jwtCfg, nil, nil)

	if ar.defaultStrategy != "jwt" {
		t.Errorf("expected default strategy 'jwt', got %q", ar.defaultStrategy)
	}

	_, ok := ar.strategies["jwt"]
	if !ok {
		t.Fatal("expected jwt strategy to be registered")
	}

	strategy := ar.strategies["jwt"]
	if strategy.Type() != "bearer" {
		t.Errorf("expected type 'bearer', got %q", strategy.Type())
	}
}
