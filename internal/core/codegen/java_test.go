package codegen

import (
	"strings"
	"testing"

	"github.com/ye-kart/reqflow/internal/domain"
)

func TestGenerateJava_SimpleGET(t *testing.T) {
	req := domain.HTTPRequest{
		Method: domain.MethodGet,
		URL:    "https://api.example.com/users",
	}
	code, err := GenerateJava(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(code, "HttpClient") {
		t.Error("expected 'HttpClient'")
	}
	if !strings.Contains(code, "HttpRequest") {
		t.Error("expected 'HttpRequest'")
	}
	if !strings.Contains(code, `URI.create("https://api.example.com/users")`) {
		t.Error("expected URI.create with URL")
	}
	if !strings.Contains(code, "System.out.println") {
		t.Error("expected 'System.out.println'")
	}
}

func TestGenerateJava_POSTWithJSONBody(t *testing.T) {
	req := domain.HTTPRequest{
		Method: domain.MethodPost,
		URL:    "https://api.example.com/users",
		Body:   []byte(`{"name":"John"}`),
	}
	code, err := GenerateJava(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(code, "POST") {
		t.Error("expected POST method")
	}
	if !strings.Contains(code, "BodyPublishers.ofString") {
		t.Error("expected 'BodyPublishers.ofString' for body")
	}
	if !strings.Contains(code, `name`) || !strings.Contains(code, `John`) {
		t.Errorf("expected body content with name and John, got:\n%s", code)
	}
}

func TestGenerateJava_HeadersIncluded(t *testing.T) {
	req := domain.HTTPRequest{
		Method: domain.MethodGet,
		URL:    "https://api.example.com/users",
		Headers: []domain.Header{
			{Key: "Accept", Value: "application/json"},
		},
	}
	code, err := GenerateJava(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(code, `.header("Accept", "application/json")`) {
		t.Errorf("expected header call, got:\n%s", code)
	}
}

func TestGenerateJava_AuthHeaderPresent(t *testing.T) {
	req := domain.HTTPRequest{
		Method: domain.MethodGet,
		URL:    "https://api.example.com/users",
		Headers: []domain.Header{
			{Key: "Authorization", Value: "Bearer token123"},
		},
	}
	code, err := GenerateJava(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(code, `.header("Authorization", "Bearer token123")`) {
		t.Errorf("expected Authorization header, got:\n%s", code)
	}
}

func TestGenerateJava_CorrectMethod(t *testing.T) {
	methods := []struct {
		method domain.HTTPMethod
		expect string
	}{
		{domain.MethodGet, ".GET()"},
		{domain.MethodPost, ".POST("},
		{domain.MethodPut, ".PUT("},
		{domain.MethodDelete, ".DELETE()"},
	}
	for _, m := range methods {
		t.Run(string(m.method), func(t *testing.T) {
			req := domain.HTTPRequest{
				Method: m.method,
				URL:    "https://example.com",
			}
			code, err := GenerateJava(req)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !strings.Contains(code, m.expect) {
				t.Errorf("expected %q in output, got:\n%s", m.expect, code)
			}
		})
	}
}
