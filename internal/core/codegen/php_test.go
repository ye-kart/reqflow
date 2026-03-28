package codegen

import (
	"strings"
	"testing"

	"github.com/ye-kart/reqflow/internal/domain"
)

func TestGeneratePHP_SimpleGET(t *testing.T) {
	req := domain.HTTPRequest{
		Method: domain.MethodGet,
		URL:    "https://api.example.com/users",
	}
	code, err := GeneratePHP(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(code, "curl_init") {
		t.Error("expected 'curl_init'")
	}
	if !strings.Contains(code, "https://api.example.com/users") {
		t.Error("expected URL in output")
	}
	if !strings.Contains(code, "curl_exec") {
		t.Error("expected 'curl_exec'")
	}
}

func TestGeneratePHP_POSTWithJSONBody(t *testing.T) {
	req := domain.HTTPRequest{
		Method: domain.MethodPost,
		URL:    "https://api.example.com/users",
		Body:   []byte(`{"name":"John"}`),
	}
	code, err := GeneratePHP(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(code, "CURLOPT_POST") || !strings.Contains(code, "CURLOPT_CUSTOMREQUEST") {
		// Either POST-specific or custom request
		if !strings.Contains(code, "POST") {
			t.Error("expected POST method indication")
		}
	}
	if !strings.Contains(code, "CURLOPT_POSTFIELDS") {
		t.Error("expected 'CURLOPT_POSTFIELDS' for body")
	}
	if !strings.Contains(code, `{"name":"John"}`) {
		t.Errorf("expected body content, got:\n%s", code)
	}
}

func TestGeneratePHP_HeadersIncluded(t *testing.T) {
	req := domain.HTTPRequest{
		Method: domain.MethodGet,
		URL:    "https://api.example.com/users",
		Headers: []domain.Header{
			{Key: "Accept", Value: "application/json"},
		},
	}
	code, err := GeneratePHP(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(code, "CURLOPT_HTTPHEADER") {
		t.Error("expected 'CURLOPT_HTTPHEADER'")
	}
	if !strings.Contains(code, "Accept: application/json") {
		t.Error("expected header in output")
	}
}

func TestGeneratePHP_AuthHeaderPresent(t *testing.T) {
	req := domain.HTTPRequest{
		Method: domain.MethodGet,
		URL:    "https://api.example.com/users",
		Headers: []domain.Header{
			{Key: "Authorization", Value: "Bearer token123"},
		},
	}
	code, err := GeneratePHP(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(code, "Authorization: Bearer token123") {
		t.Error("expected Authorization header")
	}
}

func TestGeneratePHP_CorrectMethod(t *testing.T) {
	methods := []struct {
		method domain.HTTPMethod
		expect string
	}{
		{domain.MethodPut, `"PUT"`},
		{domain.MethodDelete, `"DELETE"`},
		{domain.MethodPatch, `"PATCH"`},
	}
	for _, m := range methods {
		t.Run(string(m.method), func(t *testing.T) {
			req := domain.HTTPRequest{
				Method: m.method,
				URL:    "https://example.com",
			}
			code, err := GeneratePHP(req)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !strings.Contains(code, m.expect) {
				t.Errorf("expected %q in output, got:\n%s", m.expect, code)
			}
		})
	}
}
