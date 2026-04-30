package io

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func newTestHTTPModule() *HTTPModule {
	security := DefaultSecurityConfig()
	security.HTTP.AllowedHosts = []string{"127.0.0.1", "localhost"}
	return NewHTTPModule(security)
}

func TestHTTPModule_Get(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			t.Errorf("expected GET, got %s", r.Method)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]any{"message": "success"})
	}))
	defer server.Close()

	module := newTestHTTPModule()

	result, err := module.httpGet(server.URL)
	if err != nil {
		t.Fatalf("httpGet failed: %v", err)
	}

	resp, ok := result.(map[string]any)
	if !ok {
		t.Fatalf("expected map response, got %T", result)
	}

	if resp["status"] != 200 {
		t.Errorf("expected status 200, got %v", resp["status"])
	}

	body, ok := resp["body"].(map[string]any)
	if !ok {
		t.Fatalf("expected body to be map, got %T", resp["body"])
	}

	if body["message"] != "success" {
		t.Errorf("expected message 'success', got %v", body["message"])
	}
}

func TestHTTPModule_Post(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("expected Content-Type application/json, got %s", r.Header.Get("Content-Type"))
		}

		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Errorf("failed to decode body: %v", err)
		}

		if body["name"] != "Alice" {
			t.Errorf("expected name Alice, got %v", body["name"])
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]any{"id": 123, "name": body["name"]})
	}))
	defer server.Close()

	module := newTestHTTPModule()

	result, err := module.httpPost(server.URL, map[string]any{
		"body": map[string]any{"name": "Alice", "age": 30},
	})
	if err != nil {
		t.Fatalf("httpPost failed: %v", err)
	}

	resp, ok := result.(map[string]any)
	if !ok {
		t.Fatalf("expected map response, got %T", result)
	}

	if resp["status"] != 201 {
		t.Errorf("expected status 201, got %v", resp["status"])
	}

	body, ok := resp["body"].(map[string]any)
	if !ok {
		t.Fatalf("expected body to be map, got %T", resp["body"])
	}

	if body["name"] != "Alice" {
		t.Errorf("expected name Alice, got %v", body["name"])
	}
}

func TestHTTPModule_Put(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "PUT" {
			t.Errorf("expected PUT, got %s", r.Method)
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"updated": true}`))
	}))
	defer server.Close()

	security := DefaultSecurityConfig()
	security.HTTP.AllowedHosts = []string{"127.0.0.1", "localhost"}
	module := NewHTTPModule(security)

	result, err := module.httpPut(server.URL, map[string]any{
		"body": map[string]any{"name": "Bob"},
	})
	if err != nil {
		t.Fatalf("httpPut failed: %v", err)
	}

	resp := result.(map[string]any)
	if resp["status"] != 200 {
		t.Errorf("expected status 200, got %v", resp["status"])
	}
}

func TestHTTPModule_Patch(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "PATCH" {
			t.Errorf("expected PATCH, got %s", r.Method)
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"patched": true}`))
	}))
	defer server.Close()

	security := DefaultSecurityConfig()
	security.HTTP.AllowedHosts = []string{"127.0.0.1", "localhost"}
	module := NewHTTPModule(security)

	result, err := module.httpPatch(server.URL, map[string]any{
		"body": map[string]any{"status": "active"},
	})
	if err != nil {
		t.Fatalf("httpPatch failed: %v", err)
	}

	resp := result.(map[string]any)
	if resp["status"] != 200 {
		t.Errorf("expected status 200, got %v", resp["status"])
	}
}

func TestHTTPModule_Delete(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "DELETE" {
			t.Errorf("expected DELETE, got %s", r.Method)
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	security := DefaultSecurityConfig()
	security.HTTP.AllowedHosts = []string{"127.0.0.1", "localhost"}
	module := NewHTTPModule(security)

	result, err := module.httpDelete(server.URL)
	if err != nil {
		t.Fatalf("httpDelete failed: %v", err)
	}

	resp := result.(map[string]any)
	if resp["status"] != 204 {
		t.Errorf("expected status 204, got %v", resp["status"])
	}
}

func TestHTTPModule_AuthBearer(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		auth := r.Header.Get("Authorization")
		if auth != "Bearer test-token-123" {
			t.Errorf("expected Bearer token, got %s", auth)
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"authenticated": true}`))
	}))
	defer server.Close()

	security := DefaultSecurityConfig()
	security.HTTP.AllowedHosts = []string{"127.0.0.1", "localhost"}
	module := NewHTTPModule(security)

	result, err := module.httpGet(server.URL, map[string]any{
		"auth": map[string]any{
			"type":  "bearer",
			"token": "test-token-123",
		},
	})
	if err != nil {
		t.Fatalf("httpGet with bearer auth failed: %v", err)
	}

	resp := result.(map[string]any)
	if resp["status"] != 200 {
		t.Errorf("expected status 200, got %v", resp["status"])
	}
}

func TestHTTPModule_AuthBasic(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		username, password, ok := r.BasicAuth()
		if !ok {
			t.Error("expected basic auth")
		}
		if username != "admin" || password != "secret" {
			t.Errorf("expected admin/secret, got %s/%s", username, password)
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"authenticated": true}`))
	}))
	defer server.Close()

	security := DefaultSecurityConfig()
	security.HTTP.AllowedHosts = []string{"127.0.0.1", "localhost"}
	module := NewHTTPModule(security)

	result, err := module.httpGet(server.URL, map[string]any{
		"auth": map[string]any{
			"type":     "basic",
			"username": "admin",
			"password": "secret",
		},
	})
	if err != nil {
		t.Fatalf("httpGet with basic auth failed: %v", err)
	}

	resp := result.(map[string]any)
	if resp["status"] != 200 {
		t.Errorf("expected status 200, got %v", resp["status"])
	}
}

func TestHTTPModule_AuthInvalidType(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	security := DefaultSecurityConfig()
	security.HTTP.AllowedHosts = []string{"127.0.0.1", "localhost"}
	module := NewHTTPModule(security)

	_, err := module.httpGet(server.URL, map[string]any{
		"auth": map[string]any{
			"type": "oauth2",
		},
	})
	if err == nil {
		t.Fatal("expected error for invalid auth type")
	}
	if !strings.Contains(err.Error(), "unsupported auth type") {
		t.Errorf("expected unsupported auth type error, got: %v", err)
	}
}

func TestHTTPModule_AuthPrecedence(t *testing.T) {
	// Test that auth config takes precedence over Authorization header
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		auth := r.Header.Get("Authorization")
		if auth != "Bearer from-auth-config" {
			t.Errorf("expected auth config to take precedence, got %s", auth)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	security := DefaultSecurityConfig()
	security.HTTP.AllowedHosts = []string{"127.0.0.1", "localhost"}
	module := NewHTTPModule(security)

	_, err := module.httpGet(server.URL, map[string]any{
		"headers": map[string]any{
			"Authorization": "Bearer from-header",
		},
		"auth": map[string]any{
			"type":  "bearer",
			"token": "from-auth-config",
		},
	})
	if err != nil {
		t.Fatalf("httpGet failed: %v", err)
	}
}

func TestHTTPModule_SecurityBlockedHost(t *testing.T) {
	security := DefaultSecurityConfig()
	module := NewHTTPModule(security)

	// localhost is blocked by default
	_, err := module.httpGet("http://localhost:8080/api")
	if err == nil {
		t.Fatal("expected error for blocked host")
	}
	if !strings.Contains(err.Error(), "blocked") {
		t.Errorf("expected blocked host error, got: %v", err)
	}
}

func TestHTTPModule_SecurityAllowedHost(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"ok": true}`))
	}))
	defer server.Close()

	security := DefaultSecurityConfig()
	security.HTTP.AllowedHosts = []string{"127.0.0.1", "localhost"}
	module := NewHTTPModule(security)

	result, err := module.httpGet(server.URL)
	if err != nil {
		t.Fatalf("httpGet failed: %v", err)
	}

	resp := result.(map[string]any)
	if resp["status"] != 200 {
		t.Errorf("expected status 200, got %v", resp["status"])
	}
}

func TestHTTPModule_MaxResponseSize(t *testing.T) {
	// Create a server that returns a large response
	largeData := strings.Repeat("x", 1024*1024) // 1MB
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(largeData))
	}))
	defer server.Close()

	security := DefaultSecurityConfig()
	security.HTTP.AllowedHosts = []string{"127.0.0.1", "localhost"}
	security.HTTP.MaxResponseSize = 1024 // 1KB limit
	module := NewHTTPModule(security)

	_, err := module.httpGet(server.URL)
	if err == nil {
		t.Fatal("expected error for response exceeding max size")
	}
	if !strings.Contains(err.Error(), "exceeds max size") {
		t.Errorf("expected max size error, got: %v", err)
	}
}

func TestHTTPModule_Timeout(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(2 * time.Second)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	security := DefaultSecurityConfig()
	security.HTTP.AllowedHosts = []string{"127.0.0.1", "localhost"}
	security.HTTP.Timeout = 1 // 1 second
	module := NewHTTPModule(security)

	_, err := module.httpGet(server.URL)
	if err == nil {
		t.Fatal("expected timeout error")
	}
	if !strings.Contains(err.Error(), "context deadline exceeded") && !strings.Contains(err.Error(), "timeout") {
		t.Errorf("expected timeout error, got: %v", err)
	}
}

func TestHTTPModule_CustomTimeout(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(200 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"ok": true}`))
	}))
	defer server.Close()

	security := DefaultSecurityConfig()
	security.HTTP.AllowedHosts = []string{"127.0.0.1", "localhost"}
	module := NewHTTPModule(security)

	// Custom timeout via options (100ms - should timeout)
	_, err := module.httpGet(server.URL, map[string]any{
		"timeout": 100, // milliseconds
	})
	if err == nil {
		t.Fatal("expected timeout error")
	}

	// Custom timeout via options (500ms - should succeed)
	result, err := module.httpGet(server.URL, map[string]any{
		"timeout": 500, // milliseconds
	})
	if err != nil {
		t.Fatalf("httpGet with custom timeout failed: %v", err)
	}

	resp := result.(map[string]any)
	if resp["status"] != 200 {
		t.Errorf("expected status 200, got %v", resp["status"])
	}
}

func TestHTTPModule_NonJSONResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("plain text response"))
	}))
	defer server.Close()

	security := DefaultSecurityConfig()
	security.HTTP.AllowedHosts = []string{"127.0.0.1", "localhost"}
	module := NewHTTPModule(security)

	result, err := module.httpGet(server.URL)
	if err != nil {
		t.Fatalf("httpGet failed: %v", err)
	}

	resp := result.(map[string]any)
	body, ok := resp["body"].(string)
	if !ok {
		t.Fatalf("expected body to be string, got %T", resp["body"])
	}

	if body != "plain text response" {
		t.Errorf("expected 'plain text response', got %s", body)
	}
}

func TestHTTPModule_JSONResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]any{
			"users": []map[string]any{
				{"id": 1, "name": "Alice"},
				{"id": 2, "name": "Bob"},
			},
		})
	}))
	defer server.Close()

	module := newTestHTTPModule()

	result, err := module.httpGet(server.URL)
	if err != nil {
		t.Fatalf("httpGet failed: %v", err)
	}

	resp := result.(map[string]any)
	body, ok := resp["body"].(map[string]any)
	if !ok {
		t.Fatalf("expected body to be map, got %T", resp["body"])
	}

	users, ok := body["users"].([]any)
	if !ok {
		t.Fatalf("expected users to be array, got %T", body["users"])
	}

	if len(users) != 2 {
		t.Errorf("expected 2 users, got %d", len(users))
	}
}

func TestHTTPModule_CustomHeaders(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-Custom-Header") != "custom-value" {
			t.Errorf("expected custom header, got %s", r.Header.Get("X-Custom-Header"))
		}
		if r.Header.Get("User-Agent") != "go-json/1.0" {
			t.Errorf("expected custom user agent, got %s", r.Header.Get("User-Agent"))
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	module := newTestHTTPModule()

	_, err := module.httpGet(server.URL, map[string]any{
		"headers": map[string]any{
			"X-Custom-Header": "custom-value",
			"User-Agent":      "go-json/1.0",
		},
	})
	if err != nil {
		t.Fatalf("httpGet with custom headers failed: %v", err)
	}
}

func TestHTTPModule_ResponseHeaders(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Request-ID", "abc-123")
		w.Header().Set("X-Rate-Limit", "100")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"ok": true}`))
	}))
	defer server.Close()

	module := newTestHTTPModule()

	result, err := module.httpGet(server.URL)
	if err != nil {
		t.Fatalf("httpGet failed: %v", err)
	}

	resp := result.(map[string]any)
	headers, ok := resp["headers"].(map[string]any)
	if !ok {
		t.Fatalf("expected headers to be map, got %T", resp["headers"])
	}

	if headers["X-Request-Id"] != "abc-123" {
		t.Errorf("expected X-Request-ID header, got %v", headers["X-Request-Id"])
	}

	if headers["X-Rate-Limit"] != "100" {
		t.Errorf("expected X-Rate-Limit header, got %v", headers["X-Rate-Limit"])
	}
}

func TestHTTPModule_MissingURL(t *testing.T) {
	module := NewHTTPModule(DefaultSecurityConfig())

	_, err := module.httpGet()
	if err == nil {
		t.Fatal("expected error for missing URL")
	}
	if !strings.Contains(err.Error(), "url is required") {
		t.Errorf("expected 'url is required' error, got: %v", err)
	}
}

func TestHTTPModule_InvalidURL(t *testing.T) {
	module := NewHTTPModule(DefaultSecurityConfig())

	_, err := module.httpGet(123) // not a string
	if err == nil {
		t.Fatal("expected error for invalid URL type")
	}
	if !strings.Contains(err.Error(), "url must be a string") {
		t.Errorf("expected 'url must be a string' error, got: %v", err)
	}
}
