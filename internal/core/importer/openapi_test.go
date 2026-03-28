package importer

import (
	"testing"

	"github.com/ye-kart/reqflow/internal/domain"
)

func TestParseOpenAPI_ValidSpec(t *testing.T) {
	data := []byte(`{
		"openapi": "3.0.3",
		"info": {"title": "Pet Store", "version": "1.0.0"},
		"servers": [{"url": "https://api.example.com"}],
		"paths": {
			"/pets": {
				"get": {
					"summary": "List pets",
					"responses": {"200": {"description": "OK"}}
				}
			}
		}
	}`)

	col, err := ParseOpenAPI(data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if col.Name != "Pet Store" {
		t.Errorf("Name = %q, want %q", col.Name, "Pet Store")
	}

	if len(col.Requests) != 1 {
		t.Fatalf("Requests len = %d, want 1", len(col.Requests))
	}

	req := col.Requests[0]
	if req.Config.Method != domain.MethodGet {
		t.Errorf("Method = %q, want GET", req.Config.Method)
	}
	if req.Config.URL != "https://api.example.com/pets" {
		t.Errorf("URL = %q, want %q", req.Config.URL, "https://api.example.com/pets")
	}
}

func TestParseOpenAPI_MultiplePaths(t *testing.T) {
	data := []byte(`{
		"openapi": "3.0.3",
		"info": {"title": "Multi Path", "version": "1.0.0"},
		"servers": [{"url": "https://api.example.com"}],
		"paths": {
			"/users": {
				"get": {
					"summary": "List users",
					"responses": {"200": {"description": "OK"}}
				},
				"post": {
					"summary": "Create user",
					"responses": {"201": {"description": "Created"}}
				}
			},
			"/items": {
				"get": {
					"summary": "List items",
					"responses": {"200": {"description": "OK"}}
				}
			}
		}
	}`)

	col, err := ParseOpenAPI(data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(col.Requests) != 3 {
		t.Fatalf("Requests len = %d, want 3", len(col.Requests))
	}

	methods := make(map[string]bool)
	for _, r := range col.Requests {
		methods[string(r.Config.Method)] = true
	}
	if !methods["GET"] {
		t.Error("missing GET method")
	}
	if !methods["POST"] {
		t.Error("missing POST method")
	}
}

func TestParseOpenAPI_ServersAsBaseURL(t *testing.T) {
	data := []byte(`{
		"openapi": "3.0.3",
		"info": {"title": "Server Test", "version": "1.0.0"},
		"servers": [
			{"url": "https://staging.example.com"},
			{"url": "https://prod.example.com"}
		],
		"paths": {
			"/health": {
				"get": {
					"summary": "Health check",
					"responses": {"200": {"description": "OK"}}
				}
			}
		}
	}`)

	col, err := ParseOpenAPI(data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// First server should be used as base URL variable.
	found := false
	for _, v := range col.Variables {
		if v.Key == "base_url" && v.Value == "https://staging.example.com" {
			found = true
		}
	}
	if !found {
		t.Error("missing base_url variable from servers[0]")
	}

	// URL should use the first server.
	if col.Requests[0].Config.URL != "https://staging.example.com/health" {
		t.Errorf("URL = %q, want %q", col.Requests[0].Config.URL, "https://staging.example.com/health")
	}
}

func TestParseOpenAPI_SecuritySchemes(t *testing.T) {
	data := []byte(`{
		"openapi": "3.0.3",
		"info": {"title": "Secure API", "version": "1.0.0"},
		"paths": {
			"/secure": {
				"get": {
					"summary": "Secure endpoint",
					"responses": {"200": {"description": "OK"}}
				}
			}
		},
		"components": {
			"securitySchemes": {
				"bearerAuth": {
					"type": "http",
					"scheme": "bearer"
				}
			}
		},
		"security": [{"bearerAuth": []}]
	}`)

	col, err := ParseOpenAPI(data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if col.Auth == nil {
		t.Fatal("expected collection-level Auth from security schemes")
	}
	if col.Auth.Type != domain.AuthBearer {
		t.Errorf("Auth.Type = %q, want bearer", col.Auth.Type)
	}
}

func TestParseOpenAPI_SecuritySchemes_APIKey(t *testing.T) {
	data := []byte(`{
		"openapi": "3.0.3",
		"info": {"title": "API Key API", "version": "1.0.0"},
		"paths": {
			"/data": {
				"get": {
					"summary": "Get data",
					"responses": {"200": {"description": "OK"}}
				}
			}
		},
		"components": {
			"securitySchemes": {
				"apiKeyAuth": {
					"type": "apiKey",
					"name": "X-API-Key",
					"in": "header"
				}
			}
		},
		"security": [{"apiKeyAuth": []}]
	}`)

	col, err := ParseOpenAPI(data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if col.Auth == nil {
		t.Fatal("expected collection-level Auth")
	}
	if col.Auth.Type != domain.AuthAPIKey {
		t.Errorf("Auth.Type = %q, want apikey", col.Auth.Type)
	}
	if col.Auth.APIKey.Key != "X-API-Key" {
		t.Errorf("APIKey.Key = %q, want %q", col.Auth.APIKey.Key, "X-API-Key")
	}
}

func TestParseOpenAPI_SecuritySchemes_BasicAuth(t *testing.T) {
	data := []byte(`{
		"openapi": "3.0.3",
		"info": {"title": "Basic Auth API", "version": "1.0.0"},
		"paths": {
			"/data": {
				"get": {
					"summary": "Get data",
					"responses": {"200": {"description": "OK"}}
				}
			}
		},
		"components": {
			"securitySchemes": {
				"basicAuth": {
					"type": "http",
					"scheme": "basic"
				}
			}
		},
		"security": [{"basicAuth": []}]
	}`)

	col, err := ParseOpenAPI(data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if col.Auth == nil {
		t.Fatal("expected collection-level Auth")
	}
	if col.Auth.Type != domain.AuthBasic {
		t.Errorf("Auth.Type = %q, want basic", col.Auth.Type)
	}
}

func TestParseOpenAPI_YAMLFormat(t *testing.T) {
	data := []byte(`openapi: "3.0.3"
info:
  title: YAML API
  version: "1.0.0"
servers:
  - url: https://api.example.com
paths:
  /items:
    get:
      summary: List items
      responses:
        "200":
          description: OK
`)

	col, err := ParseOpenAPI(data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if col.Name != "YAML API" {
		t.Errorf("Name = %q, want %q", col.Name, "YAML API")
	}
	if len(col.Requests) != 1 {
		t.Fatalf("Requests len = %d, want 1", len(col.Requests))
	}
}

func TestParseOpenAPI_NoServers(t *testing.T) {
	data := []byte(`{
		"openapi": "3.0.3",
		"info": {"title": "No Server", "version": "1.0.0"},
		"paths": {
			"/test": {
				"get": {
					"summary": "Test",
					"responses": {"200": {"description": "OK"}}
				}
			}
		}
	}`)

	col, err := ParseOpenAPI(data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Without servers, URL should just be the path.
	if col.Requests[0].Config.URL != "/test" {
		t.Errorf("URL = %q, want %q", col.Requests[0].Config.URL, "/test")
	}
}

func TestParseOpenAPI_InvalidInput(t *testing.T) {
	_, err := ParseOpenAPI([]byte(`{invalid`))
	if err == nil {
		t.Fatal("expected error for invalid input, got nil")
	}
}
