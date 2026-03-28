package importer

import (
	"encoding/json"
	"testing"

	"github.com/ye-kart/reqflow/internal/domain"
)

func TestParsePostmanCollection_ValidCollection(t *testing.T) {
	data := []byte(`{
		"info": {
			"name": "My API",
			"schema": "https://schema.getpostman.com/json/collection/v2.1.0/collection.json"
		},
		"item": [
			{
				"name": "Get Users",
				"request": {
					"method": "GET",
					"url": {
						"raw": "https://api.example.com/users"
					},
					"header": [
						{"key": "Accept", "value": "application/json"}
					]
				}
			}
		]
	}`)

	col, err := ParsePostmanCollection(data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if col.Name != "My API" {
		t.Errorf("Name = %q, want %q", col.Name, "My API")
	}
	if len(col.Requests) != 1 {
		t.Fatalf("Requests len = %d, want 1", len(col.Requests))
	}

	req := col.Requests[0]
	if req.Name != "Get Users" {
		t.Errorf("Request.Name = %q, want %q", req.Name, "Get Users")
	}
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

func TestParsePostmanCollection_NestedFolders(t *testing.T) {
	data := []byte(`{
		"info": {"name": "Nested API"},
		"item": [
			{
				"name": "Users Folder",
				"item": [
					{
						"name": "List Users",
						"request": {
							"method": "GET",
							"url": {"raw": "https://api.example.com/users"}
						}
					},
					{
						"name": "Admin Subfolder",
						"item": [
							{
								"name": "List Admins",
								"request": {
									"method": "GET",
									"url": {"raw": "https://api.example.com/admins"}
								}
							}
						]
					}
				]
			}
		]
	}`)

	col, err := ParsePostmanCollection(data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(col.Folders) != 1 {
		t.Fatalf("Folders len = %d, want 1", len(col.Folders))
	}

	folder := col.Folders[0]
	if folder.Name != "Users Folder" {
		t.Errorf("Folder.Name = %q, want %q", folder.Name, "Users Folder")
	}
	if len(folder.Requests) != 1 {
		t.Errorf("Folder.Requests len = %d, want 1", len(folder.Requests))
	}
	if len(folder.Folders) != 1 {
		t.Fatalf("Folder.Folders len = %d, want 1", len(folder.Folders))
	}

	subfolder := folder.Folders[0]
	if subfolder.Name != "Admin Subfolder" {
		t.Errorf("Subfolder.Name = %q, want %q", subfolder.Name, "Admin Subfolder")
	}
	if len(subfolder.Requests) != 1 {
		t.Fatalf("Subfolder.Requests len = %d, want 1", len(subfolder.Requests))
	}
	if subfolder.Requests[0].Name != "List Admins" {
		t.Errorf("Subfolder request name = %q, want %q", subfolder.Requests[0].Name, "List Admins")
	}
}

func TestParsePostmanCollection_Auth(t *testing.T) {
	tests := []struct {
		name     string
		authJSON string
		wantType domain.AuthType
		check    func(*testing.T, *domain.AuthConfig)
	}{
		{
			name: "basic auth",
			authJSON: `{
				"type": "basic",
				"basic": [
					{"key": "username", "value": "admin"},
					{"key": "password", "value": "secret"}
				]
			}`,
			wantType: domain.AuthBasic,
			check: func(t *testing.T, auth *domain.AuthConfig) {
				if auth.Basic == nil {
					t.Fatal("Basic auth config is nil")
				}
				if auth.Basic.Username != "admin" {
					t.Errorf("Username = %q, want %q", auth.Basic.Username, "admin")
				}
				if auth.Basic.Password != "secret" {
					t.Errorf("Password = %q, want %q", auth.Basic.Password, "secret")
				}
			},
		},
		{
			name: "bearer auth",
			authJSON: `{
				"type": "bearer",
				"bearer": [
					{"key": "token", "value": "my-token-123"}
				]
			}`,
			wantType: domain.AuthBearer,
			check: func(t *testing.T, auth *domain.AuthConfig) {
				if auth.Bearer == nil {
					t.Fatal("Bearer auth config is nil")
				}
				if auth.Bearer.Token != "my-token-123" {
					t.Errorf("Token = %q, want %q", auth.Bearer.Token, "my-token-123")
				}
			},
		},
		{
			name: "apikey auth",
			authJSON: `{
				"type": "apikey",
				"apikey": [
					{"key": "key", "value": "X-API-Key"},
					{"key": "value", "value": "abc123"},
					{"key": "in", "value": "header"}
				]
			}`,
			wantType: domain.AuthAPIKey,
			check: func(t *testing.T, auth *domain.AuthConfig) {
				if auth.APIKey == nil {
					t.Fatal("APIKey auth config is nil")
				}
				if auth.APIKey.Key != "X-API-Key" {
					t.Errorf("Key = %q, want %q", auth.APIKey.Key, "X-API-Key")
				}
				if auth.APIKey.Value != "abc123" {
					t.Errorf("Value = %q, want %q", auth.APIKey.Value, "abc123")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data := []byte(`{
				"info": {"name": "Auth Test"},
				"item": [
					{
						"name": "Auth Request",
						"request": {
							"method": "GET",
							"url": {"raw": "https://api.example.com/secure"},
							"auth": ` + tt.authJSON + `
						}
					}
				]
			}`)

			col, err := ParsePostmanCollection(data)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if len(col.Requests) != 1 {
				t.Fatalf("Requests len = %d, want 1", len(col.Requests))
			}
			auth := col.Requests[0].Config.Auth
			if auth == nil {
				t.Fatal("Auth is nil")
			}
			if auth.Type != tt.wantType {
				t.Errorf("Auth.Type = %q, want %q", auth.Type, tt.wantType)
			}
			tt.check(t, auth)
		})
	}
}

func TestParsePostmanCollection_Variables(t *testing.T) {
	data := []byte(`{
		"info": {"name": "Var Test"},
		"variable": [
			{"key": "base_url", "value": "https://api.example.com"},
			{"key": "token", "value": "abc123"}
		],
		"item": []
	}`)

	col, err := ParsePostmanCollection(data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(col.Variables) != 2 {
		t.Fatalf("Variables len = %d, want 2", len(col.Variables))
	}

	varMap := make(map[string]string)
	for _, v := range col.Variables {
		varMap[v.Key] = v.Value
	}
	if varMap["base_url"] != "https://api.example.com" {
		t.Errorf("base_url = %q, want %q", varMap["base_url"], "https://api.example.com")
	}
	if varMap["token"] != "abc123" {
		t.Errorf("token = %q, want %q", varMap["token"], "abc123")
	}
}

func TestParsePostmanCollection_RequestBody(t *testing.T) {
	data := []byte(`{
		"info": {"name": "Body Test"},
		"item": [
			{
				"name": "Create Item",
				"request": {
					"method": "POST",
					"url": {"raw": "https://api.example.com/items"},
					"body": {
						"mode": "raw",
						"raw": "{\"name\":\"test\"}"
					}
				}
			}
		]
	}`)

	col, err := ParsePostmanCollection(data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(col.Requests) != 1 {
		t.Fatalf("Requests len = %d, want 1", len(col.Requests))
	}
	if string(col.Requests[0].Config.Body) != `{"name":"test"}` {
		t.Errorf("Body = %q, want %q", string(col.Requests[0].Config.Body), `{"name":"test"}`)
	}
}

func TestParsePostmanCollection_InvalidJSON(t *testing.T) {
	_, err := ParsePostmanCollection([]byte(`{invalid`))
	if err == nil {
		t.Fatal("expected error for invalid JSON, got nil")
	}
}

func TestParsePostmanCollection_URLAsString(t *testing.T) {
	data := []byte(`{
		"info": {"name": "URL String Test"},
		"item": [
			{
				"name": "Simple Request",
				"request": {
					"method": "GET",
					"url": "https://api.example.com/simple"
				}
			}
		]
	}`)

	col, err := ParsePostmanCollection(data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if col.Requests[0].Config.URL != "https://api.example.com/simple" {
		t.Errorf("URL = %q, want %q", col.Requests[0].Config.URL, "https://api.example.com/simple")
	}
}

func TestParsePostmanCollection_RoundTrip(t *testing.T) {
	// This test verifies Postman import -> export -> import produces equivalent results.
	original := []byte(`{
		"info": {"name": "Round Trip"},
		"item": [
			{
				"name": "Test Request",
				"request": {
					"method": "POST",
					"url": {"raw": "https://api.example.com/test"},
					"header": [
						{"key": "Content-Type", "value": "application/json"}
					],
					"body": {
						"mode": "raw",
						"raw": "{\"key\":\"value\"}"
					}
				}
			}
		]
	}`)

	col1, err := ParsePostmanCollection(original)
	if err != nil {
		t.Fatalf("first parse error: %v", err)
	}

	// Re-export and re-import to verify round-trip.
	exported, err := json.Marshal(PostmanCollectionFromDomain(col1))
	if err != nil {
		t.Fatalf("export error: %v", err)
	}

	col2, err := ParsePostmanCollection(exported)
	if err != nil {
		t.Fatalf("second parse error: %v", err)
	}

	if col1.Name != col2.Name {
		t.Errorf("Name mismatch: %q vs %q", col1.Name, col2.Name)
	}
	if len(col1.Requests) != len(col2.Requests) {
		t.Fatalf("Requests len mismatch: %d vs %d", len(col1.Requests), len(col2.Requests))
	}
	if col1.Requests[0].Config.Method != col2.Requests[0].Config.Method {
		t.Errorf("Method mismatch: %q vs %q", col1.Requests[0].Config.Method, col2.Requests[0].Config.Method)
	}
	if col1.Requests[0].Config.URL != col2.Requests[0].Config.URL {
		t.Errorf("URL mismatch: %q vs %q", col1.Requests[0].Config.URL, col2.Requests[0].Config.URL)
	}
	if string(col1.Requests[0].Config.Body) != string(col2.Requests[0].Config.Body) {
		t.Errorf("Body mismatch: %q vs %q", string(col1.Requests[0].Config.Body), string(col2.Requests[0].Config.Body))
	}
}
