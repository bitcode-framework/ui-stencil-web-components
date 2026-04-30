package server

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/bitcode-framework/go-json/lang"
	"github.com/bitcode-framework/go-json/runtime"
	"github.com/bitcode-framework/go-json/server/adapters"
	"github.com/bitcode-framework/go-json/stdlib"
)

func callMiddleware(mw adapters.MiddlewareFunc, ctx *adapters.RequestContext, handler adapters.HandlerFunc) *adapters.Response {
	return mw(ctx, func() *adapters.Response { return handler(ctx) })
}

func newTestRegistry(cfg *lang.ServerConfig) *MiddlewareRegistry {
	reg := stdlib.DefaultRegistry()
	rt := runtime.NewRuntime(runtime.WithStdlib(reg.All()), runtime.WithStdlibEnv(reg.EnvVars()))
	return NewMiddlewareRegistry(cfg, rt, nil)
}

func newTestContext() *adapters.RequestContext {
	return &adapters.RequestContext{
		Method:  "GET",
		Path:    "/test",
		URL:     "/test",
		Params:  map[string]string{},
		Query:   map[string]string{},
		Headers: map[string]string{},
		Cookies: map[string]string{},
		IP:      "127.0.0.1",
		Store:   map[string]any{},
	}
}

func okHandler(_ *adapters.RequestContext) *adapters.Response {
	return &adapters.Response{Status: 200, Body: map[string]any{"ok": true}}
}

// --- MiddlewareRegistry tests ---

func TestMiddlewareRegistry_ResolveBuiltin(t *testing.T) {
	cfg := &lang.ServerConfig{
		CORS:      &lang.CORSConfig{Origins: []string{"*"}},
		RateLimit: &lang.RateLimitConfig{Requests: 10, Window: "1s", By: "ip"},
	}
	mr := newTestRegistry(cfg)

	names := []string{"logger", "recover", "secure", "request_id", "compress", "cors", "rate_limit"}
	for _, name := range names {
		mw, err := mr.Resolve(name)
		if err != nil {
			t.Errorf("Resolve(%q) returned error: %v", name, err)
		}
		if mw == nil {
			t.Errorf("Resolve(%q) returned nil middleware", name)
		}
	}
}

func TestMiddlewareRegistry_ResolveUnknown(t *testing.T) {
	cfg := &lang.ServerConfig{}
	mr := newTestRegistry(cfg)

	_, err := mr.Resolve("nonexistent_middleware")
	if err == nil {
		t.Error("expected error for unknown middleware, got nil")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("expected 'not found' in error, got: %v", err)
	}
}

func TestMiddlewareRegistry_BuildChain(t *testing.T) {
	cfg := &lang.ServerConfig{
		CORS: &lang.CORSConfig{Origins: []string{"*"}},
	}
	mr := newTestRegistry(cfg)

	chain, err := mr.BuildChain([]string{"logger", "recover", "secure"})
	if err != nil {
		t.Fatalf("BuildChain returned error: %v", err)
	}
	if len(chain) != 3 {
		t.Errorf("expected chain length 3, got %d", len(chain))
	}
}

func TestMiddlewareRegistry_BuildChainError(t *testing.T) {
	cfg := &lang.ServerConfig{}
	mr := newTestRegistry(cfg)

	_, err := mr.BuildChain([]string{"logger", "unknown_mw", "secure"})
	if err == nil {
		t.Error("expected error for unknown middleware in chain, got nil")
	}
	if !strings.Contains(err.Error(), "unknown_mw") {
		t.Errorf("expected error to mention 'unknown_mw', got: %v", err)
	}
}

// --- CORS middleware tests ---

func TestCORSMiddleware_Preflight(t *testing.T) {
	cfg := &lang.ServerConfig{
		CORS: &lang.CORSConfig{Origins: []string{"http://localhost:3000"}},
	}
	mr := newTestRegistry(cfg)
	mw, _ := mr.Resolve("cors")

	ctx := newTestContext()
	ctx.Method = "OPTIONS"

	resp := callMiddleware(mw, ctx, okHandler)
	if resp.Status != 204 {
		t.Errorf("expected status 204, got %d", resp.Status)
	}
	if resp.Headers["Access-Control-Allow-Origin"] != "http://localhost:3000" {
		t.Errorf("expected origin header 'http://localhost:3000', got %q", resp.Headers["Access-Control-Allow-Origin"])
	}
	if resp.Headers["Access-Control-Allow-Methods"] == "" {
		t.Error("expected Access-Control-Allow-Methods header")
	}
	if resp.Headers["Access-Control-Allow-Headers"] == "" {
		t.Error("expected Access-Control-Allow-Headers header")
	}
	if resp.Headers["Access-Control-Max-Age"] == "" {
		t.Error("expected Access-Control-Max-Age header")
	}
}

func TestCORSMiddleware_NormalRequest(t *testing.T) {
	cfg := &lang.ServerConfig{
		CORS: &lang.CORSConfig{Origins: []string{"http://localhost:3000"}},
	}
	mr := newTestRegistry(cfg)
	mw, _ := mr.Resolve("cors")

	ctx := newTestContext()
	ctx.Method = "GET"

	resp := callMiddleware(mw, ctx, okHandler)
	if resp.Status != 200 {
		t.Errorf("expected status 200, got %d", resp.Status)
	}
	if resp.Headers["Access-Control-Allow-Origin"] != "http://localhost:3000" {
		t.Errorf("expected origin header 'http://localhost:3000', got %q", resp.Headers["Access-Control-Allow-Origin"])
	}
}

// --- Secure middleware tests ---

func TestSecureMiddleware_Headers(t *testing.T) {
	cfg := &lang.ServerConfig{}
	mr := newTestRegistry(cfg)
	mw, _ := mr.Resolve("secure")

	ctx := newTestContext()
	resp := callMiddleware(mw, ctx, okHandler)

	expected := map[string]string{
		"X-Content-Type-Options": "nosniff",
		"X-Frame-Options":       "SAMEORIGIN",
		"X-XSS-Protection":      "1; mode=block",
		"Referrer-Policy":        "strict-origin-when-cross-origin",
	}
	for header, want := range expected {
		got := resp.Headers[header]
		if got != want {
			t.Errorf("header %q: expected %q, got %q", header, want, got)
		}
	}
}

// --- Request ID middleware tests ---

func TestRequestIDMiddleware_Generate(t *testing.T) {
	cfg := &lang.ServerConfig{}
	mr := newTestRegistry(cfg)
	mw, _ := mr.Resolve("request_id")

	ctx := newTestContext()
	resp := callMiddleware(mw, ctx, okHandler)

	reqID := resp.Headers["X-Request-ID"]
	if reqID == "" {
		t.Error("expected X-Request-ID header to be generated")
	}
	if len(reqID) < 32 {
		t.Errorf("expected UUID-like request ID, got %q", reqID)
	}
	if ctx.Store["request_id"] != reqID {
		t.Errorf("expected store request_id to match header, got %v", ctx.Store["request_id"])
	}
}

func TestRequestIDMiddleware_Passthrough(t *testing.T) {
	cfg := &lang.ServerConfig{}
	mr := newTestRegistry(cfg)
	mw, _ := mr.Resolve("request_id")

	ctx := newTestContext()
	ctx.Headers["X-Request-ID"] = "my-custom-id-123"

	resp := callMiddleware(mw, ctx, okHandler)

	if resp.Headers["X-Request-ID"] != "my-custom-id-123" {
		t.Errorf("expected passthrough of X-Request-ID, got %q", resp.Headers["X-Request-ID"])
	}
	if ctx.Store["request_id"] != "my-custom-id-123" {
		t.Errorf("expected store request_id to be 'my-custom-id-123', got %v", ctx.Store["request_id"])
	}
}

// --- Recover middleware tests ---

func TestRecoverMiddleware_NoPanic(t *testing.T) {
	cfg := &lang.ServerConfig{}
	mr := newTestRegistry(cfg)
	mw, _ := mr.Resolve("recover")

	ctx := newTestContext()
	resp := callMiddleware(mw, ctx, okHandler)

	if resp.Status != 200 {
		t.Errorf("expected status 200, got %d", resp.Status)
	}
}

func TestRecoverMiddleware_WithPanic(t *testing.T) {
	cfg := &lang.ServerConfig{}
	mr := newTestRegistry(cfg)
	mw, _ := mr.Resolve("recover")

	ctx := newTestContext()
	panicHandler := func(_ *adapters.RequestContext) *adapters.Response {
		panic("something went wrong")
	}

	resp := callMiddleware(mw, ctx, panicHandler)

	if resp.Status != 500 {
		t.Errorf("expected status 500, got %d", resp.Status)
	}
	body, ok := resp.Body.(map[string]any)
	if !ok {
		t.Fatalf("expected map body, got %T", resp.Body)
	}
	errObj, ok := body["error"].(map[string]any)
	if !ok {
		t.Fatalf("expected error object in body, got %v", body)
	}
	if errObj["code"] != "INTERNAL_ERROR" {
		t.Errorf("expected error code INTERNAL_ERROR, got %v", errObj["code"])
	}
}

// --- Rate limit middleware tests ---

func TestRateLimitMiddleware_UnderLimit(t *testing.T) {
	cfg := &lang.ServerConfig{
		RateLimit: &lang.RateLimitConfig{Requests: 5, Window: "1s", By: "ip"},
	}
	mr := newTestRegistry(cfg)
	mw, _ := mr.Resolve("rate_limit")

	ctx := newTestContext()
	ctx.IP = "10.0.0.1"

	for i := 0; i < 5; i++ {
		resp := callMiddleware(mw, ctx, okHandler)
		if resp.Status != 200 {
			t.Errorf("request %d: expected status 200, got %d", i+1, resp.Status)
		}
	}
}

func TestRateLimitMiddleware_OverLimit(t *testing.T) {
	cfg := &lang.ServerConfig{
		RateLimit: &lang.RateLimitConfig{Requests: 2, Window: "1s", By: "ip"},
	}
	mr := newTestRegistry(cfg)
	mw, _ := mr.Resolve("rate_limit")

	ctx := newTestContext()
	ctx.IP = "10.0.0.2"

	// First 2 should pass
	for i := 0; i < 2; i++ {
		resp := callMiddleware(mw, ctx, okHandler)
		if resp.Status != 200 {
			t.Errorf("request %d: expected status 200, got %d", i+1, resp.Status)
		}
	}

	// Third should be rate limited
	resp := callMiddleware(mw, ctx, okHandler)
	if resp.Status != 429 {
		t.Errorf("expected status 429, got %d", resp.Status)
	}
	body, ok := resp.Body.(map[string]any)
	if !ok {
		t.Fatalf("expected map body, got %T", resp.Body)
	}
	errObj, ok := body["error"].(map[string]any)
	if !ok {
		t.Fatalf("expected error object in body, got %v", body)
	}
	if errObj["code"] != "RATE_LIMIT_EXCEEDED" {
		t.Errorf("expected RATE_LIMIT_EXCEEDED, got %v", errObj["code"])
	}
	if resp.Headers["Retry-After"] == "" {
		t.Error("expected Retry-After header")
	}
}

func TestRateLimitMiddleware_WindowReset(t *testing.T) {
	cfg := &lang.ServerConfig{
		RateLimit: &lang.RateLimitConfig{Requests: 1, Window: "100ms", By: "ip"},
	}
	mr := newTestRegistry(cfg)
	mw, _ := mr.Resolve("rate_limit")

	ctx := newTestContext()
	ctx.IP = "10.0.0.3"

	// First request passes
	resp := callMiddleware(mw, ctx, okHandler)
	if resp.Status != 200 {
		t.Errorf("first request: expected status 200, got %d", resp.Status)
	}

	// Second request should be rate limited
	resp = callMiddleware(mw, ctx, okHandler)
	if resp.Status != 429 {
		t.Errorf("second request: expected status 429, got %d", resp.Status)
	}

	// Wait for window to expire
	time.Sleep(150 * time.Millisecond)

	// After window reset, should pass again
	resp = callMiddleware(mw, ctx, okHandler)
	if resp.Status != 200 {
		t.Errorf("after reset: expected status 200, got %d", resp.Status)
	}
}

// --- Compress middleware tests ---

func TestCompressMiddleware_SmallBody(t *testing.T) {
	cfg := &lang.ServerConfig{}
	mr := newTestRegistry(cfg)
	mw, _ := mr.Resolve("compress")

	ctx := newTestContext()
	ctx.Headers["Accept-Encoding"] = "gzip"

	smallHandler := func(_ *adapters.RequestContext) *adapters.Response {
		return &adapters.Response{
			Status: 200,
			Body:   map[string]any{"msg": "small"},
		}
	}

	resp := callMiddleware(mw, ctx, smallHandler)
	if resp.Headers != nil && resp.Headers["Content-Encoding"] == "gzip" {
		t.Error("expected no gzip for small body")
	}
}

func makeLargeBody() map[string]any {
	data := make(map[string]any)
	for i := 0; i < 200; i++ {
		data[fmt.Sprintf("field_%04d", i)] = strings.Repeat("x", 20)
	}
	return data
}

func TestCompressMiddleware_NoAcceptEncoding(t *testing.T) {
	cfg := &lang.ServerConfig{}
	mr := newTestRegistry(cfg)
	mw, _ := mr.Resolve("compress")

	ctx := newTestContext()

	largeHandler := func(_ *adapters.RequestContext) *adapters.Response {
		return &adapters.Response{Status: 200, Body: makeLargeBody()}
	}

	resp := callMiddleware(mw, ctx, largeHandler)
	if resp.Headers != nil && resp.Headers["Content-Encoding"] == "gzip" {
		t.Error("expected no gzip without Accept-Encoding header")
	}
}

func TestCompressMiddleware_LargeBody(t *testing.T) {
	cfg := &lang.ServerConfig{}
	mr := newTestRegistry(cfg)
	mw, _ := mr.Resolve("compress")

	ctx := newTestContext()
	ctx.Headers["Accept-Encoding"] = "gzip"

	largeHandler := func(_ *adapters.RequestContext) *adapters.Response {
		return &adapters.Response{Status: 200, Body: makeLargeBody(), Headers: map[string]string{}}
	}

	resp := callMiddleware(mw, ctx, largeHandler)
	if resp.Headers["Content-Encoding"] != "gzip" {
		t.Error("expected Content-Encoding: gzip for large body with Accept-Encoding")
	}
	if resp.Headers["Vary"] != "Accept-Encoding" {
		t.Error("expected Vary: Accept-Encoding header")
	}
}
