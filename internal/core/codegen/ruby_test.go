package codegen

import (
	"strings"
	"testing"

	"github.com/ye-kart/reqflow/internal/domain"
)

func TestGenerateRuby_SimpleGET(t *testing.T) {
	req := domain.HTTPRequest{
		Method: domain.MethodGet,
		URL:    "https://api.example.com/users",
	}
	code, err := GenerateRuby(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(code, "require") {
		t.Error("expected 'require' statement")
	}
	if !strings.Contains(code, "Net::HTTP") {
		t.Error("expected 'Net::HTTP'")
	}
	if !strings.Contains(code, "https://api.example.com/users") {
		t.Error("expected URL in output")
	}
}

func TestGenerateRuby_POSTWithJSONBody(t *testing.T) {
	req := domain.HTTPRequest{
		Method: domain.MethodPost,
		URL:    "https://api.example.com/users",
		Body:   []byte(`{"name":"John"}`),
	}
	code, err := GenerateRuby(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(code, "Net::HTTP::Post") {
		t.Error("expected 'Net::HTTP::Post'")
	}
	if !strings.Contains(code, "body") {
		t.Error("expected body assignment")
	}
	if !strings.Contains(code, `{"name":"John"}`) {
		t.Errorf("expected body content, got:\n%s", code)
	}
}

func TestGenerateRuby_HeadersIncluded(t *testing.T) {
	req := domain.HTTPRequest{
		Method: domain.MethodGet,
		URL:    "https://api.example.com/users",
		Headers: []domain.Header{
			{Key: "Accept", Value: "application/json"},
		},
	}
	code, err := GenerateRuby(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(code, `"Accept"`) {
		t.Error("expected Accept header key")
	}
	if !strings.Contains(code, `"application/json"`) {
		t.Error("expected Accept header value")
	}
}

func TestGenerateRuby_AuthHeaderPresent(t *testing.T) {
	req := domain.HTTPRequest{
		Method: domain.MethodGet,
		URL:    "https://api.example.com/users",
		Headers: []domain.Header{
			{Key: "Authorization", Value: "Bearer token123"},
		},
	}
	code, err := GenerateRuby(req)
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

func TestGenerateRuby_CorrectMethod(t *testing.T) {
	methods := []struct {
		method domain.HTTPMethod
		class  string
	}{
		{domain.MethodGet, "Net::HTTP::Get"},
		{domain.MethodPost, "Net::HTTP::Post"},
		{domain.MethodPut, "Net::HTTP::Put"},
		{domain.MethodDelete, "Net::HTTP::Delete"},
	}
	for _, m := range methods {
		t.Run(string(m.method), func(t *testing.T) {
			req := domain.HTTPRequest{
				Method: m.method,
				URL:    "https://example.com",
			}
			code, err := GenerateRuby(req)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !strings.Contains(code, m.class) {
				t.Errorf("expected %q in output, got:\n%s", m.class, code)
			}
		})
	}
}
