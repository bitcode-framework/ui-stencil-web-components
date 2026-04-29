package server

import (
	"testing"

	"github.com/bitcode-framework/go-json/server/adapters"
)

func TestBuildRequestMap(t *testing.T) {
	ctx := &adapters.RequestContext{
		Method:  "POST",
		Path:    "/users",
		URL:     "/users?page=1",
		Params:  map[string]string{"id": "42"},
		Query:   map[string]string{"page": "1"},
		Headers: map[string]string{"Content-Type": "application/json"},
		Body:    map[string]any{"name": "test"},
		Cookies: map[string]string{"session": "abc"},
		IP:      "127.0.0.1",
		Store:   map[string]any{},
	}

	m := BuildRequestMap(ctx)
	if m["method"] != "POST" {
		t.Errorf("expected POST, got %v", m["method"])
	}
	if m["ip"] != "127.0.0.1" {
		t.Errorf("expected 127.0.0.1, got %v", m["ip"])
	}
	body, ok := m["body"].(map[string]any)
	if !ok || body["name"] != "test" {
		t.Errorf("unexpected body: %v", m["body"])
	}
}

func TestConvertToResponse_Nil(t *testing.T) {
	resp := ConvertToResponse(nil, FlatRoute{})
	if resp.Status != 204 {
		t.Errorf("expected 204, got %d", resp.Status)
	}
}

func TestConvertToResponse_JSON(t *testing.T) {
	result := map[string]any{
		"status": float64(200),
		"body":   map[string]any{"users": []any{}},
	}
	resp := ConvertToResponse(result, FlatRoute{})
	if resp.Status != 200 {
		t.Errorf("expected 200, got %d", resp.Status)
	}
	if resp.Body == nil {
		t.Error("expected body")
	}
}

func TestConvertToResponse_Redirect(t *testing.T) {
	result := map[string]any{
		"redirect": "/login",
	}
	resp := ConvertToResponse(result, FlatRoute{})
	if resp.Redirect != "/login" {
		t.Errorf("expected /login, got %s", resp.Redirect)
	}
	if resp.Status != 302 {
		t.Errorf("expected 302, got %d", resp.Status)
	}
}

func TestConvertToResponse_Template(t *testing.T) {
	result := map[string]any{
		"data": map[string]any{"title": "Home"},
	}
	resp := ConvertToResponse(result, FlatRoute{Render: "index.html"})
	if resp.Template != "index.html" {
		t.Errorf("expected index.html, got %s", resp.Template)
	}
}

func TestConvertToResponse_Error(t *testing.T) {
	result := map[string]any{
		"error": "something went wrong",
	}
	resp := ConvertToResponse(result, FlatRoute{})
	if resp.Status != 500 {
		t.Errorf("expected 500, got %d", resp.Status)
	}
}

func TestConvertToResponse_ErrorWithStatus(t *testing.T) {
	result := map[string]any{
		"status": float64(403),
		"error":  "forbidden",
	}
	resp := ConvertToResponse(result, FlatRoute{})
	if resp.Status != 403 {
		t.Errorf("expected 403, got %d", resp.Status)
	}
}

func TestConvertToResponse_Cookies(t *testing.T) {
	result := map[string]any{
		"status": float64(200),
		"body":   "ok",
		"cookies": map[string]any{
			"name":      "token",
			"value":     "abc123",
			"http_only": true,
			"max_age":   float64(3600),
		},
	}
	resp := ConvertToResponse(result, FlatRoute{})
	if len(resp.Cookies) != 1 {
		t.Fatalf("expected 1 cookie, got %d", len(resp.Cookies))
	}
	if resp.Cookies[0].Name != "token" || !resp.Cookies[0].HTTPOnly {
		t.Errorf("unexpected cookie: %+v", resp.Cookies[0])
	}
}

func TestConvertToResponse_NoBody204(t *testing.T) {
	result := map[string]any{}
	resp := ConvertToResponse(result, FlatRoute{})
	if resp.Status != 204 {
		t.Errorf("expected 204, got %d", resp.Status)
	}
}
