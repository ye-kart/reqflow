package codegen

import (
	"strings"
	"testing"

	"github.com/ye-kart/reqflow/internal/domain"
)

func TestGenerateCSharp_SimpleGET(t *testing.T) {
	req := domain.HTTPRequest{
		Method: domain.MethodGet,
		URL:    "https://api.example.com/users",
	}
	code, err := GenerateCSharp(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(code, "HttpClient") {
		t.Error("expected 'HttpClient'")
	}
	if !strings.Contains(code, "https://api.example.com/users") {
		t.Error("expected URL in output")
	}
	if !strings.Contains(code, "Console.WriteLine") {
		t.Error("expected 'Console.WriteLine'")
	}
}

func TestGenerateCSharp_POSTWithJSONBody(t *testing.T) {
	req := domain.HTTPRequest{
		Method: domain.MethodPost,
		URL:    "https://api.example.com/users",
		Body:   []byte(`{"name":"John"}`),
	}
	code, err := GenerateCSharp(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(code, "StringContent") {
		t.Error("expected 'StringContent' for body")
	}
	if !strings.Contains(code, "name") || !strings.Contains(code, "John") {
		t.Errorf("expected body content with name and John, got:\n%s", code)
	}
}

func TestGenerateCSharp_HeadersIncluded(t *testing.T) {
	req := domain.HTTPRequest{
		Method: domain.MethodGet,
		URL:    "https://api.example.com/users",
		Headers: []domain.Header{
			{Key: "Accept", Value: "application/json"},
		},
	}
	code, err := GenerateCSharp(req)
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

func TestGenerateCSharp_AuthHeaderPresent(t *testing.T) {
	req := domain.HTTPRequest{
		Method: domain.MethodGet,
		URL:    "https://api.example.com/users",
		Headers: []domain.Header{
			{Key: "Authorization", Value: "Bearer token123"},
		},
	}
	code, err := GenerateCSharp(req)
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

func TestGenerateCSharp_CorrectMethod(t *testing.T) {
	methods := []struct {
		method domain.HTTPMethod
		expect string
	}{
		{domain.MethodGet, "GetAsync"},
		{domain.MethodPost, "PostAsync"},
		{domain.MethodPut, "PutAsync"},
		{domain.MethodDelete, "DeleteAsync"},
	}
	for _, m := range methods {
		t.Run(string(m.method), func(t *testing.T) {
			req := domain.HTTPRequest{
				Method: m.method,
				URL:    "https://example.com",
			}
			code, err := GenerateCSharp(req)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !strings.Contains(code, m.expect) {
				t.Errorf("expected %q in output, got:\n%s", m.expect, code)
			}
		})
	}
}
