package codegen

import (
	"strings"
	"testing"

	"github.com/ye-kart/reqflow/internal/domain"
)

func TestGenerateGo_SimpleGET(t *testing.T) {
	req := domain.HTTPRequest{
		Method: domain.MethodGet,
		URL:    "https://api.example.com/users",
	}
	code, err := GenerateGo(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(code, "http.NewRequest") {
		t.Error("expected 'http.NewRequest'")
	}
	if !strings.Contains(code, `"GET"`) {
		t.Error("expected '\"GET\"' method")
	}
	if !strings.Contains(code, `"https://api.example.com/users"`) {
		t.Error("expected URL in output")
	}
	if !strings.Contains(code, "http.DefaultClient.Do") {
		t.Error("expected 'http.DefaultClient.Do'")
	}
	if !strings.Contains(code, "io.ReadAll") {
		t.Error("expected 'io.ReadAll'")
	}
}

func TestGenerateGo_POSTWithJSONBody(t *testing.T) {
	req := domain.HTTPRequest{
		Method: domain.MethodPost,
		URL:    "https://api.example.com/users",
		Body:   []byte(`{"name":"John"}`),
	}
	code, err := GenerateGo(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(code, `"POST"`) {
		t.Error("expected '\"POST\"' method")
	}
	if !strings.Contains(code, "strings.NewReader") {
		t.Error("expected 'strings.NewReader' for body")
	}
	if !strings.Contains(code, `{"name":"John"}`) {
		t.Errorf("expected body content, got:\n%s", code)
	}
}

func TestGenerateGo_HeadersIncluded(t *testing.T) {
	req := domain.HTTPRequest{
		Method: domain.MethodGet,
		URL:    "https://api.example.com/users",
		Headers: []domain.Header{
			{Key: "Accept", Value: "application/json"},
		},
	}
	code, err := GenerateGo(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(code, `req.Header.Set("Accept", "application/json")`) {
		t.Errorf("expected header set call, got:\n%s", code)
	}
}

func TestGenerateGo_AuthHeaderPresent(t *testing.T) {
	req := domain.HTTPRequest{
		Method: domain.MethodGet,
		URL:    "https://api.example.com/users",
		Headers: []domain.Header{
			{Key: "Authorization", Value: "Bearer token123"},
		},
	}
	code, err := GenerateGo(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(code, `req.Header.Set("Authorization", "Bearer token123")`) {
		t.Errorf("expected Authorization header set, got:\n%s", code)
	}
}

func TestGenerateGo_CorrectMethod(t *testing.T) {
	methods := []struct {
		method domain.HTTPMethod
		expect string
	}{
		{domain.MethodGet, `"GET"`},
		{domain.MethodPost, `"POST"`},
		{domain.MethodPut, `"PUT"`},
		{domain.MethodDelete, `"DELETE"`},
	}
	for _, m := range methods {
		t.Run(string(m.method), func(t *testing.T) {
			req := domain.HTTPRequest{
				Method: m.method,
				URL:    "https://example.com",
			}
			code, err := GenerateGo(req)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !strings.Contains(code, m.expect) {
				t.Errorf("expected %q in output, got:\n%s", m.expect, code)
			}
		})
	}
}
