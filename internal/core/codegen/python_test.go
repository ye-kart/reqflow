package codegen

import (
	"strings"
	"testing"

	"github.com/ye-kart/reqflow/internal/domain"
)

func TestGeneratePython_SimpleGET(t *testing.T) {
	req := domain.HTTPRequest{
		Method: domain.MethodGet,
		URL:    "https://api.example.com/users",
	}
	code, err := GeneratePython(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(code, "import requests") {
		t.Error("expected 'import requests'")
	}
	if !strings.Contains(code, "requests.get(") {
		t.Error("expected 'requests.get('")
	}
	if !strings.Contains(code, `"https://api.example.com/users"`) {
		t.Error("expected URL in output")
	}
}

func TestGeneratePython_POSTWithJSONBody(t *testing.T) {
	req := domain.HTTPRequest{
		Method: domain.MethodPost,
		URL:    "https://api.example.com/users",
		Body:   []byte(`{"name":"John","age":30}`),
	}
	code, err := GeneratePython(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(code, "requests.post(") {
		t.Error("expected 'requests.post('")
	}
	if !strings.Contains(code, `json=`) {
		t.Error("expected json= keyword argument for JSON body")
	}
	if !strings.Contains(code, `"name"`) {
		t.Error("expected body content in output")
	}
}

func TestGeneratePython_HeadersIncluded(t *testing.T) {
	req := domain.HTTPRequest{
		Method: domain.MethodGet,
		URL:    "https://api.example.com/users",
		Headers: []domain.Header{
			{Key: "Accept", Value: "application/json"},
			{Key: "X-Custom", Value: "value"},
		},
	}
	code, err := GeneratePython(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(code, "headers=") {
		t.Error("expected 'headers='")
	}
	if !strings.Contains(code, `"Accept"`) {
		t.Error("expected Accept header key")
	}
	if !strings.Contains(code, `"application/json"`) {
		t.Error("expected Accept header value")
	}
	if !strings.Contains(code, `"X-Custom"`) {
		t.Error("expected X-Custom header key")
	}
}

func TestGeneratePython_AuthHeaderPresent(t *testing.T) {
	req := domain.HTTPRequest{
		Method: domain.MethodGet,
		URL:    "https://api.example.com/users",
		Headers: []domain.Header{
			{Key: "Authorization", Value: "Bearer token123"},
		},
	}
	code, err := GeneratePython(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(code, `"Authorization"`) {
		t.Error("expected Authorization header key")
	}
	if !strings.Contains(code, `"Bearer token123"`) {
		t.Error("expected Bearer token value")
	}
}

func TestGeneratePython_CorrectMethod(t *testing.T) {
	methods := []struct {
		method domain.HTTPMethod
		call   string
	}{
		{domain.MethodGet, "requests.get("},
		{domain.MethodPost, "requests.post("},
		{domain.MethodPut, "requests.put("},
		{domain.MethodPatch, "requests.patch("},
		{domain.MethodDelete, "requests.delete("},
	}
	for _, m := range methods {
		t.Run(string(m.method), func(t *testing.T) {
			req := domain.HTTPRequest{
				Method: m.method,
				URL:    "https://example.com",
			}
			code, err := GeneratePython(req)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !strings.Contains(code, m.call) {
				t.Errorf("expected %q in output, got:\n%s", m.call, code)
			}
		})
	}
}
