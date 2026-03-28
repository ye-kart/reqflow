package importer

import (
	"testing"

	"github.com/ye-kart/reqflow/internal/domain"
)

func TestParseHAR_ValidArchive(t *testing.T) {
	data := []byte(`{
		"log": {
			"version": "1.2",
			"entries": [
				{
					"request": {
						"method": "GET",
						"url": "https://api.example.com/users",
						"headers": [
							{"name": "Accept", "value": "application/json"}
						]
					}
				}
			]
		}
	}`)

	col, err := ParseHAR(data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if col.Name != "HAR Import" {
		t.Errorf("Name = %q, want %q", col.Name, "HAR Import")
	}
	if len(col.Requests) != 1 {
		t.Fatalf("Requests len = %d, want 1", len(col.Requests))
	}

	req := col.Requests[0]
	if req.Config.Method != domain.MethodGet {
		t.Errorf("Method = %q, want GET", req.Config.Method)
	}
	if req.Config.URL != "https://api.example.com/users" {
		t.Errorf("URL = %q, want %q", req.Config.URL, "https://api.example.com/users")
	}
	if len(req.Config.Headers) != 1 {
		t.Fatalf("Headers len = %d, want 1", len(req.Config.Headers))
	}
	if req.Config.Headers[0].Key != "Accept" || req.Config.Headers[0].Value != "application/json" {
		t.Errorf("Header = %+v, want Accept: application/json", req.Config.Headers[0])
	}
}

func TestParseHAR_MultipleEntries(t *testing.T) {
	data := []byte(`{
		"log": {
			"entries": [
				{
					"request": {
						"method": "GET",
						"url": "https://api.example.com/users",
						"headers": []
					}
				},
				{
					"request": {
						"method": "POST",
						"url": "https://api.example.com/items",
						"headers": []
					}
				},
				{
					"request": {
						"method": "DELETE",
						"url": "https://api.example.com/items/1",
						"headers": []
					}
				}
			]
		}
	}`)

	col, err := ParseHAR(data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(col.Requests) != 3 {
		t.Fatalf("Requests len = %d, want 3", len(col.Requests))
	}

	expected := []struct {
		method domain.HTTPMethod
		url    string
	}{
		{domain.MethodGet, "https://api.example.com/users"},
		{domain.MethodPost, "https://api.example.com/items"},
		{domain.MethodDelete, "https://api.example.com/items/1"},
	}

	for i, want := range expected {
		if col.Requests[i].Config.Method != want.method {
			t.Errorf("Request[%d].Method = %q, want %q", i, col.Requests[i].Config.Method, want.method)
		}
		if col.Requests[i].Config.URL != want.url {
			t.Errorf("Request[%d].URL = %q, want %q", i, col.Requests[i].Config.URL, want.url)
		}
	}
}

func TestParseHAR_PostWithBody(t *testing.T) {
	data := []byte(`{
		"log": {
			"entries": [
				{
					"request": {
						"method": "POST",
						"url": "https://api.example.com/items",
						"headers": [
							{"name": "Content-Type", "value": "application/json"}
						],
						"postData": {
							"mimeType": "application/json",
							"text": "{\"name\":\"test\"}"
						}
					}
				}
			]
		}
	}`)

	col, err := ParseHAR(data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(col.Requests) != 1 {
		t.Fatalf("Requests len = %d, want 1", len(col.Requests))
	}

	req := col.Requests[0]
	if string(req.Config.Body) != `{"name":"test"}` {
		t.Errorf("Body = %q, want %q", string(req.Config.Body), `{"name":"test"}`)
	}
	if req.Config.ContentType != "application/json" {
		t.Errorf("ContentType = %q, want %q", req.Config.ContentType, "application/json")
	}
}

func TestParseHAR_RequestNames(t *testing.T) {
	data := []byte(`{
		"log": {
			"entries": [
				{
					"request": {
						"method": "GET",
						"url": "https://api.example.com/users?page=1",
						"headers": []
					}
				}
			]
		}
	}`)

	col, err := ParseHAR(data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Name should be derived from method + URL path.
	if col.Requests[0].Name != "GET /users" {
		t.Errorf("Name = %q, want %q", col.Requests[0].Name, "GET /users")
	}
}

func TestParseHAR_InvalidJSON(t *testing.T) {
	_, err := ParseHAR([]byte(`{invalid`))
	if err == nil {
		t.Fatal("expected error for invalid JSON, got nil")
	}
}

func TestParseHAR_EmptyEntries(t *testing.T) {
	data := []byte(`{"log": {"entries": []}}`)

	col, err := ParseHAR(data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(col.Requests) != 0 {
		t.Errorf("Requests len = %d, want 0", len(col.Requests))
	}
}
