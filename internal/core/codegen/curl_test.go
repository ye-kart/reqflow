package codegen

import (
	"strings"
	"testing"

	"github.com/ye-kart/reqflow/internal/domain"
)

func TestGenerateCurl_SimpleGET(t *testing.T) {
	req := domain.HTTPRequest{
		Method: domain.MethodGet,
		URL:    "https://api.example.com/users",
	}
	code, err := GenerateCurl(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(code, "curl") {
		t.Error("expected 'curl' command")
	}
	if !strings.Contains(code, "https://api.example.com/users") {
		t.Error("expected URL in output")
	}
}

func TestGenerateCurl_POSTWithJSONBody(t *testing.T) {
	req := domain.HTTPRequest{
		Method: domain.MethodPost,
		URL:    "https://api.example.com/users",
		Body:   []byte(`{"name":"John"}`),
	}
	code, err := GenerateCurl(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(code, "-X POST") {
		t.Error("expected '-X POST'")
	}
	if !strings.Contains(code, "-d") {
		t.Error("expected '-d' for body")
	}
	if !strings.Contains(code, `{"name":"John"}`) {
		t.Error("expected body content")
	}
}

func TestGenerateCurl_HeadersIncluded(t *testing.T) {
	req := domain.HTTPRequest{
		Method: domain.MethodGet,
		URL:    "https://api.example.com/users",
		Headers: []domain.Header{
			{Key: "Accept", Value: "application/json"},
		},
	}
	code, err := GenerateCurl(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(code, "-H") {
		t.Error("expected '-H' for header")
	}
	if !strings.Contains(code, "Accept: application/json") {
		t.Error("expected header in output")
	}
}

func TestGenerateCurl_AuthHeaderPresent(t *testing.T) {
	req := domain.HTTPRequest{
		Method: domain.MethodGet,
		URL:    "https://api.example.com/users",
		Headers: []domain.Header{
			{Key: "Authorization", Value: "Bearer token123"},
		},
	}
	code, err := GenerateCurl(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(code, "Authorization: Bearer token123") {
		t.Error("expected Authorization header")
	}
}

func TestGenerateCurl_CorrectMethod(t *testing.T) {
	methods := []struct {
		method domain.HTTPMethod
		expect string
	}{
		{domain.MethodPut, "-X PUT"},
		{domain.MethodDelete, "-X DELETE"},
		{domain.MethodPatch, "-X PATCH"},
	}
	for _, m := range methods {
		t.Run(string(m.method), func(t *testing.T) {
			req := domain.HTTPRequest{
				Method: m.method,
				URL:    "https://example.com",
			}
			code, err := GenerateCurl(req)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !strings.Contains(code, m.expect) {
				t.Errorf("expected %q in output, got:\n%s", m.expect, code)
			}
		})
	}
}
