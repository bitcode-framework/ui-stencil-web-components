package server

import (
	"os"
	"strings"
	"testing"

	"github.com/bitcode-framework/go-json/lang"
	"github.com/bitcode-framework/go-json/server/adapters"
	"github.com/golang-jwt/jwt/v5"
)

const testSecret = "test-secret-key-for-testing"
const testSecretEnv = "TEST_JWT_SECRET"

func setTestSecret(t *testing.T) {
	t.Helper()
	os.Setenv(testSecretEnv, testSecret)
	t.Cleanup(func() { os.Unsetenv(testSecretEnv) })
}

func TestSignToken_Basic(t *testing.T) {
	payload := map[string]any{"sub": "user123", "role": "admin"}
	token, err := signToken(payload, testSecret, "HS256", "")
	if err != nil {
		t.Fatalf("signToken failed: %v", err)
	}
	if token == "" {
		t.Fatal("expected non-empty token string")
	}
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		t.Fatalf("expected 3 JWT parts, got %d", len(parts))
	}
}

func TestSignToken_WithExpiry(t *testing.T) {
	payload := map[string]any{"sub": "user123"}
	token, err := signToken(payload, testSecret, "HS256", "1h")
	if err != nil {
		t.Fatalf("signToken failed: %v", err)
	}

	parsed, err := jwt.Parse(token, func(token *jwt.Token) (any, error) {
		return []byte(testSecret), nil
	})
	if err != nil {
		t.Fatalf("failed to parse token: %v", err)
	}

	claims, ok := parsed.Claims.(jwt.MapClaims)
	if !ok {
		t.Fatal("expected MapClaims")
	}
	if _, exists := claims["exp"]; !exists {
		t.Fatal("expected exp claim to exist")
	}
	if _, exists := claims["iat"]; !exists {
		t.Fatal("expected iat claim to exist")
	}
}

func TestVerifyToken_Valid(t *testing.T) {
	payload := map[string]any{"sub": "user123", "role": "admin"}
	token, err := signToken(payload, testSecret, "HS256", "1h")
	if err != nil {
		t.Fatalf("signToken failed: %v", err)
	}

	result, err := verifyToken(token, testSecret, "HS256")
	if err != nil {
		t.Fatalf("verifyToken failed: %v", err)
	}

	claims, ok := result.(map[string]any)
	if !ok {
		t.Fatal("expected map[string]any result")
	}
	if claims["sub"] != "user123" {
		t.Errorf("expected sub=user123, got %v", claims["sub"])
	}
	if claims["role"] != "admin" {
		t.Errorf("expected role=admin, got %v", claims["role"])
	}
}

func TestVerifyToken_Expired(t *testing.T) {
	payload := map[string]any{"sub": "user123"}
	token, err := signToken(payload, testSecret, "HS256", "-1h")
	if err != nil {
		t.Fatalf("signToken failed: %v", err)
	}

	_, err = verifyToken(token, testSecret, "HS256")
	if err == nil {
		t.Fatal("expected error for expired token")
	}
	if !strings.Contains(err.Error(), "invalid token") {
		t.Errorf("expected 'invalid token' in error, got: %v", err)
	}
}

func TestVerifyToken_WrongSecret(t *testing.T) {
	payload := map[string]any{"sub": "user123"}
	token, err := signToken(payload, testSecret, "HS256", "1h")
	if err != nil {
		t.Fatalf("signToken failed: %v", err)
	}

	_, err = verifyToken(token, "wrong-secret", "HS256")
	if err == nil {
		t.Fatal("expected error for wrong secret")
	}
}

func TestVerifyToken_WrongAlgorithm(t *testing.T) {
	payload := map[string]any{"sub": "user123"}
	token, err := signToken(payload, testSecret, "HS256", "1h")
	if err != nil {
		t.Fatalf("signToken failed: %v", err)
	}

	_, err = verifyToken(token, testSecret, "HS512")
	if err == nil {
		t.Fatal("expected error for wrong algorithm")
	}
	if !strings.Contains(err.Error(), "signing method") {
		t.Errorf("expected 'signing method' in error, got: %v", err)
	}
}

func TestDecodeToken_Valid(t *testing.T) {
	payload := map[string]any{"sub": "user123", "name": "Alice"}
	token, err := signToken(payload, testSecret, "HS256", "1h")
	if err != nil {
		t.Fatalf("signToken failed: %v", err)
	}

	result, err := decodeToken(token)
	if err != nil {
		t.Fatalf("decodeToken failed: %v", err)
	}

	claims, ok := result.(map[string]any)
	if !ok {
		t.Fatal("expected map[string]any result")
	}
	if claims["sub"] != "user123" {
		t.Errorf("expected sub=user123, got %v", claims["sub"])
	}
	if claims["name"] != "Alice" {
		t.Errorf("expected name=Alice, got %v", claims["name"])
	}
}

func TestDecodeToken_Expired(t *testing.T) {
	payload := map[string]any{"sub": "user123"}
	token, err := signToken(payload, testSecret, "HS256", "-1h")
	if err != nil {
		t.Fatalf("signToken failed: %v", err)
	}

	result, err := decodeToken(token)
	if err != nil {
		t.Fatalf("decodeToken should succeed for expired token: %v", err)
	}

	claims, ok := result.(map[string]any)
	if !ok {
		t.Fatal("expected map[string]any result")
	}
	if claims["sub"] != "user123" {
		t.Errorf("expected sub=user123, got %v", claims["sub"])
	}
}

func TestExtractToken_FromHeader(t *testing.T) {
	ctx := &adapters.RequestContext{
		Headers: map[string]string{
			"Authorization": "Bearer my-token-value",
		},
		Cookies: map[string]string{},
	}
	cfg := &lang.JWTConfig{}

	token := extractToken(ctx, cfg)
	if token != "my-token-value" {
		t.Errorf("expected 'my-token-value', got %q", token)
	}
}

func TestExtractToken_FromCookie(t *testing.T) {
	ctx := &adapters.RequestContext{
		Headers: map[string]string{},
		Cookies: map[string]string{
			"auth_token": "cookie-token-value",
		},
	}
	cfg := &lang.JWTConfig{
		Cookie: "auth_token",
	}

	token := extractToken(ctx, cfg)
	if token != "cookie-token-value" {
		t.Errorf("expected 'cookie-token-value', got %q", token)
	}
}

func TestExtractToken_CustomHeader(t *testing.T) {
	ctx := &adapters.RequestContext{
		Headers: map[string]string{
			"X-Auth-Token": "Bearer custom-header-token",
		},
		Cookies: map[string]string{},
	}
	cfg := &lang.JWTConfig{
		Header: "X-Auth-Token",
	}

	token := extractToken(ctx, cfg)
	if token != "custom-header-token" {
		t.Errorf("expected 'custom-header-token', got %q", token)
	}
}

func TestExtractToken_CustomPrefix(t *testing.T) {
	ctx := &adapters.RequestContext{
		Headers: map[string]string{
			"Authorization": "Token my-custom-prefix-token",
		},
		Cookies: map[string]string{},
	}
	cfg := &lang.JWTConfig{
		Prefix: "Token ",
	}

	token := extractToken(ctx, cfg)
	if token != "my-custom-prefix-token" {
		t.Errorf("expected 'my-custom-prefix-token', got %q", token)
	}
}

func TestExtractToken_Missing(t *testing.T) {
	ctx := &adapters.RequestContext{
		Headers: map[string]string{},
		Cookies: map[string]string{},
	}
	cfg := &lang.JWTConfig{}

	token := extractToken(ctx, cfg)
	if token != "" {
		t.Errorf("expected empty string, got %q", token)
	}
}

func TestJWTFunctions_Sign(t *testing.T) {
	setTestSecret(t)
	cfg := &lang.JWTConfig{
		SecretEnv: testSecretEnv,
		Algorithm: "HS256",
		Expiry:    "1h",
	}

	fns := JWTFunctions(cfg)
	signFn := fns["sign"].(func(args ...any) (any, error))

	result, err := signFn(map[string]any{"sub": "user123"})
	if err != nil {
		t.Fatalf("sign function failed: %v", err)
	}

	tokenStr, ok := result.(string)
	if !ok {
		t.Fatal("expected string result from sign")
	}
	if tokenStr == "" {
		t.Fatal("expected non-empty token")
	}
}

func TestJWTFunctions_Verify(t *testing.T) {
	setTestSecret(t)
	cfg := &lang.JWTConfig{
		SecretEnv: testSecretEnv,
		Algorithm: "HS256",
		Expiry:    "1h",
	}

	fns := JWTFunctions(cfg)
	signFn := fns["sign"].(func(args ...any) (any, error))
	verifyFn := fns["verify"].(func(args ...any) (any, error))

	tokenResult, err := signFn(map[string]any{"sub": "user456", "role": "editor"})
	if err != nil {
		t.Fatalf("sign failed: %v", err)
	}

	claims, err := verifyFn(tokenResult.(string))
	if err != nil {
		t.Fatalf("verify failed: %v", err)
	}

	claimsMap, ok := claims.(map[string]any)
	if !ok {
		t.Fatal("expected map[string]any from verify")
	}
	if claimsMap["sub"] != "user456" {
		t.Errorf("expected sub=user456, got %v", claimsMap["sub"])
	}
	if claimsMap["role"] != "editor" {
		t.Errorf("expected role=editor, got %v", claimsMap["role"])
	}
}

func TestJWTFunctions_Decode(t *testing.T) {
	setTestSecret(t)
	cfg := &lang.JWTConfig{
		SecretEnv: testSecretEnv,
		Algorithm: "HS256",
		Expiry:    "1h",
	}

	fns := JWTFunctions(cfg)
	signFn := fns["sign"].(func(args ...any) (any, error))
	decodeFn := fns["decode"].(func(args ...any) (any, error))

	tokenResult, err := signFn(map[string]any{"sub": "user789"})
	if err != nil {
		t.Fatalf("sign failed: %v", err)
	}

	claims, err := decodeFn(tokenResult.(string))
	if err != nil {
		t.Fatalf("decode failed: %v", err)
	}

	claimsMap, ok := claims.(map[string]any)
	if !ok {
		t.Fatal("expected map[string]any from decode")
	}
	if claimsMap["sub"] != "user789" {
		t.Errorf("expected sub=user789, got %v", claimsMap["sub"])
	}
}

func TestJWTFunctions_Refresh(t *testing.T) {
	setTestSecret(t)
	cfg := &lang.JWTConfig{
		SecretEnv: testSecretEnv,
		Algorithm: "HS256",
		Expiry:    "2h",
	}

	fns := JWTFunctions(cfg)
	signFn := fns["sign"].(func(args ...any) (any, error))
	refreshFn := fns["refresh"].(func(args ...any) (any, error))
	verifyFn := fns["verify"].(func(args ...any) (any, error))

	tokenResult, err := signFn(map[string]any{"sub": "user999", "role": "admin"})
	if err != nil {
		t.Fatalf("sign failed: %v", err)
	}

	newToken, err := refreshFn(tokenResult.(string))
	if err != nil {
		t.Fatalf("refresh failed: %v", err)
	}

	newTokenStr, ok := newToken.(string)
	if !ok {
		t.Fatal("expected string from refresh")
	}
	if newTokenStr == "" {
		t.Fatal("expected non-empty refreshed token")
	}
	claims, err := verifyFn(newTokenStr)
	if err != nil {
		t.Fatalf("verify refreshed token failed: %v", err)
	}

	claimsMap := claims.(map[string]any)
	if claimsMap["sub"] != "user999" {
		t.Errorf("expected sub=user999, got %v", claimsMap["sub"])
	}
	if claimsMap["role"] != "admin" {
		t.Errorf("expected role=admin, got %v", claimsMap["role"])
	}
}

func TestJWTMiddleware_ValidToken(t *testing.T) {
	setTestSecret(t)
	cfg := &lang.JWTConfig{
		SecretEnv: testSecretEnv,
		Algorithm: "HS256",
	}

	payload := map[string]any{"sub": "user123", "role": "admin"}
	token, err := signToken(payload, testSecret, "HS256", "1h")
	if err != nil {
		t.Fatalf("signToken failed: %v", err)
	}

	middleware := JWTMiddleware(cfg)
	ctx := &adapters.RequestContext{
		Headers: map[string]string{
			"Authorization": "Bearer " + token,
		},
		Cookies: map[string]string{},
	}

	next := func() *adapters.Response {
		return &adapters.Response{Status: 200, Body: map[string]any{"ok": true}}
	}

	resp := middleware(ctx, next)
	if resp.Status != 200 {
		t.Errorf("expected status 200, got %d", resp.Status)
	}
	if ctx.User == nil {
		t.Fatal("expected ctx.User to be set")
	}
	userClaims, ok := ctx.User.(map[string]any)
	if !ok {
		t.Fatal("expected ctx.User to be map[string]any")
	}
	if userClaims["sub"] != "user123" {
		t.Errorf("expected sub=user123, got %v", userClaims["sub"])
	}
}

func TestJWTMiddleware_MissingToken(t *testing.T) {
	setTestSecret(t)
	cfg := &lang.JWTConfig{
		SecretEnv: testSecretEnv,
		Algorithm: "HS256",
	}

	middleware := JWTMiddleware(cfg)
	ctx := &adapters.RequestContext{
		Headers: map[string]string{},
		Cookies: map[string]string{},
	}

	next := func() *adapters.Response {
		return &adapters.Response{Status: 200}
	}

	resp := middleware(ctx, next)
	if resp.Status != 401 {
		t.Errorf("expected status 401, got %d", resp.Status)
	}

	body, ok := resp.Body.(map[string]any)
	if !ok {
		t.Fatal("expected body to be map[string]any")
	}
	errObj, ok := body["error"].(map[string]any)
	if !ok {
		t.Fatal("expected error field in body")
	}
	if errObj["code"] != "UNAUTHORIZED" {
		t.Errorf("expected code=UNAUTHORIZED, got %v", errObj["code"])
	}
}

func TestJWTMiddleware_InvalidToken(t *testing.T) {
	setTestSecret(t)
	cfg := &lang.JWTConfig{
		SecretEnv: testSecretEnv,
		Algorithm: "HS256",
	}

	middleware := JWTMiddleware(cfg)
	ctx := &adapters.RequestContext{
		Headers: map[string]string{
			"Authorization": "Bearer invalid.token.here",
		},
		Cookies: map[string]string{},
	}

	next := func() *adapters.Response {
		return &adapters.Response{Status: 200}
	}

	resp := middleware(ctx, next)
	if resp.Status != 401 {
		t.Errorf("expected status 401, got %d", resp.Status)
	}

	body, ok := resp.Body.(map[string]any)
	if !ok {
		t.Fatal("expected body to be map[string]any")
	}
	errObj, ok := body["error"].(map[string]any)
	if !ok {
		t.Fatal("expected error field in body")
	}
	if errObj["code"] != "UNAUTHORIZED" {
		t.Errorf("expected code=UNAUTHORIZED, got %v", errObj["code"])
	}
}

func TestJWTMiddleware_MissingSecret(t *testing.T) {
	os.Unsetenv(testSecretEnv)
	cfg := &lang.JWTConfig{
		SecretEnv: testSecretEnv,
		Algorithm: "HS256",
	}

	payload := map[string]any{"sub": "user123"}
	token, err := signToken(payload, testSecret, "HS256", "1h")
	if err != nil {
		t.Fatalf("signToken failed: %v", err)
	}

	middleware := JWTMiddleware(cfg)
	ctx := &adapters.RequestContext{
		Headers: map[string]string{
			"Authorization": "Bearer " + token,
		},
		Cookies: map[string]string{},
	}

	next := func() *adapters.Response {
		return &adapters.Response{Status: 200}
	}

	resp := middleware(ctx, next)
	if resp.Status != 500 {
		t.Errorf("expected status 500, got %d", resp.Status)
	}

	body, ok := resp.Body.(map[string]any)
	if !ok {
		t.Fatal("expected body to be map[string]any")
	}
	errObj, ok := body["error"].(map[string]any)
	if !ok {
		t.Fatal("expected error field in body")
	}
	if errObj["code"] != "JWT_CONFIG_ERROR" {
		t.Errorf("expected code=JWT_CONFIG_ERROR, got %v", errObj["code"])
	}
}
