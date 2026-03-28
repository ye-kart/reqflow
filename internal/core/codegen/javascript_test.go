package codegen

import (
	"strings"
	"testing"

	"github.com/ye-kart/reqflow/internal/domain"
)

func TestGenerateJavaScript_SimpleGET(t *testing.T) {
	req := domain.HTTPRequest{
		Method: domain.MethodGet,
		URL:    "https://api.example.com/users",
	}
	code, err := GenerateJavaScript(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(code, "fetch(") {
		t.Error("expected 'fetch('")
	}
	if !strings.Contains(code, `"https://api.example.com/users"`) {
		t.Error("expected URL in output")
	}
	if !strings.Contains(code, "console.log(") {
		t.Error("expected 'console.log('")
	}
}

func TestGenerateJavaScript_POSTWithJSONBody(t *testing.T) {
	req := domain.HTTPRequest{
		Method: domain.MethodPost,
		URL:    "https://api.example.com/users",
		Body:   []byte(`{"name":"John"}`),
	}
	code, err := GenerateJavaScript(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(code, `"POST"`) {
		t.Error("expected '\"POST\"' method")
	}
	if !strings.Contains(code, "body:") {
		t.Error("expected 'body:' in output")
	}
	if !strings.Contains(code, `"name"`) {
		t.Error("expected body content")
	}
}

func TestGenerateJavaScript_HeadersIncluded(t *testing.T) {
	req := domain.HTTPRequest{
		Method: domain.MethodGet,
		URL:    "https://api.example.com/users",
		Headers: []domain.Header{
			{Key: "Accept", Value: "application/json"},
		},
	}
	code, err := GenerateJavaScript(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(code, "headers:") {
		t.Error("expected 'headers:' in output")
	}
	if !strings.Contains(code, `"Accept"`) {
		t.Error("expected Accept header key")
	}
}

func TestGenerateJavaScript_AuthHeaderPresent(t *testing.T) {
	req := domain.HTTPRequest{
		Method: domain.MethodGet,
		URL:    "https://api.example.com/users",
		Headers: []domain.Header{
			{Key: "Authorization", Value: "Bearer token123"},
		},
	}
	code, err := GenerateJavaScript(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(code, `"Authorization"`) {
		t.Error("expected Authorization header")
	}
	if !strings.Contains(code, `"Bearer token123"`) {
		t.Error("expected Bearer token value")
	}
}

func TestGenerateJavaScript_CorrectMethod(t *testing.T) {
	methods := []struct {
		method domain.HTTPMethod
		expect string
	}{
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
			code, err := GenerateJavaScript(req)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !strings.Contains(code, m.expect) {
				t.Errorf("expected %q in output, got:\n%s", m.expect, code)
			}
		})
	}
}
