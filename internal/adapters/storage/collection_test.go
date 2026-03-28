package storage_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/ye-kart/reqflow/internal/adapters/storage"
	"github.com/ye-kart/reqflow/internal/domain"
)

func TestReadCollection_ValidYAML(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "api-tests.yaml")

	content := `name: My API Tests
description: "Tests for the user API"
variables:
  base_url: "https://api.example.com"
auth:
  type: bearer
  bearer:
    token: "{{api_token}}"
headers:
  - key: Accept
    value: application/json
requests:
  - name: Health Check
    method: GET
    url: "{{base_url}}/health"
`
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	fs := storage.NewFilesystem()
	col, err := fs.ReadCollection(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if col.Name != "My API Tests" {
		t.Errorf("expected name 'My API Tests', got %q", col.Name)
	}
	if col.Description != "Tests for the user API" {
		t.Errorf("expected description 'Tests for the user API', got %q", col.Description)
	}

	// Check variables.
	if len(col.Variables) != 1 {
		t.Fatalf("expected 1 variable, got %d", len(col.Variables))
	}
	if col.Variables[0].Key != "base_url" || col.Variables[0].Value != "https://api.example.com" {
		t.Errorf("unexpected variable: %+v", col.Variables[0])
	}

	// Check auth.
	if col.Auth == nil {
		t.Fatal("expected auth to be set")
	}
	if col.Auth.Type != domain.AuthBearer {
		t.Errorf("expected auth type 'bearer', got %q", col.Auth.Type)
	}
	if col.Auth.Bearer == nil || col.Auth.Bearer.Token != "{{api_token}}" {
		t.Errorf("unexpected bearer config: %+v", col.Auth.Bearer)
	}

	// Check headers.
	if len(col.Headers) != 1 {
		t.Fatalf("expected 1 header, got %d", len(col.Headers))
	}
	if col.Headers[0].Key != "Accept" || col.Headers[0].Value != "application/json" {
		t.Errorf("unexpected header: %+v", col.Headers[0])
	}

	// Check requests.
	if len(col.Requests) != 1 {
		t.Fatalf("expected 1 request, got %d", len(col.Requests))
	}
	if col.Requests[0].Name != "Health Check" {
		t.Errorf("expected request name 'Health Check', got %q", col.Requests[0].Name)
	}
	if col.Requests[0].Config.Method != domain.MethodGet {
		t.Errorf("expected method GET, got %q", col.Requests[0].Config.Method)
	}
	if col.Requests[0].Config.URL != "{{base_url}}/health" {
		t.Errorf("expected URL '{{base_url}}/health', got %q", col.Requests[0].Config.URL)
	}
}

func TestWriteCollection_CreatesValidYAML(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test-collection.yaml")

	col := domain.Collection{
		Name:        "Test Collection",
		Description: "A test collection",
		Variables: []domain.Variable{
			{Key: "base_url", Value: "https://api.example.com", Scope: domain.ScopeCollection},
		},
		Auth: &domain.AuthConfig{
			Type:   domain.AuthBearer,
			Bearer: &domain.BearerAuthConfig{Token: "my-token"},
		},
		Headers: []domain.Header{
			{Key: "Accept", Value: "application/json"},
		},
		Requests: []domain.SavedRequest{
			{
				Name: "Get Users",
				Config: domain.RequestConfig{
					Method: domain.MethodGet,
					URL:    "{{base_url}}/users",
				},
			},
		},
	}

	fs := storage.NewFilesystem()
	err := fs.WriteCollection(path, col)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify file was created.
	if _, err := os.Stat(path); os.IsNotExist(err) {
		t.Fatal("expected file to exist")
	}
}

func TestRoundTrip_WriteReadCollection(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "roundtrip.yaml")

	original := domain.Collection{
		Name:        "Round Trip Test",
		Description: "Testing round-trip serialization",
		Version:     "1.0",
		Variables: []domain.Variable{
			{Key: "base_url", Value: "https://api.example.com", Scope: domain.ScopeCollection},
			{Key: "api_key", Value: "secret-key", Scope: domain.ScopeCollection},
		},
		Auth: &domain.AuthConfig{
			Type:   domain.AuthBearer,
			Bearer: &domain.BearerAuthConfig{Token: "{{api_key}}"},
		},
		Headers: []domain.Header{
			{Key: "Accept", Value: "application/json"},
			{Key: "X-Custom", Value: "custom-value"},
		},
		Requests: []domain.SavedRequest{
			{
				Name:        "Health Check",
				Description: "Check API health",
				Config: domain.RequestConfig{
					Method: domain.MethodGet,
					URL:    "{{base_url}}/health",
				},
			},
		},
		Folders: []domain.Folder{
			{
				Name:        "Users",
				Description: "User endpoints",
				Requests: []domain.SavedRequest{
					{
						Name: "List Users",
						Config: domain.RequestConfig{
							Method: domain.MethodGet,
							URL:    "{{base_url}}/users",
						},
					},
					{
						Name: "Create User",
						Config: domain.RequestConfig{
							Method:      domain.MethodPost,
							URL:         "{{base_url}}/users",
							Body:        []byte(`{"name": "John"}`),
							ContentType: "application/json",
						},
					},
				},
			},
		},
	}

	fs := storage.NewFilesystem()
	if err := fs.WriteCollection(path, original); err != nil {
		t.Fatalf("write error: %v", err)
	}

	loaded, err := fs.ReadCollection(path)
	if err != nil {
		t.Fatalf("read error: %v", err)
	}

	// Verify top-level fields.
	if loaded.Name != original.Name {
		t.Errorf("name: expected %q, got %q", original.Name, loaded.Name)
	}
	if loaded.Description != original.Description {
		t.Errorf("description: expected %q, got %q", original.Description, loaded.Description)
	}
	if loaded.Version != original.Version {
		t.Errorf("version: expected %q, got %q", original.Version, loaded.Version)
	}

	// Verify variables (compare by key, not index, since map ordering is non-deterministic).
	if len(loaded.Variables) != len(original.Variables) {
		t.Fatalf("variables: expected %d, got %d", len(original.Variables), len(loaded.Variables))
	}
	loadedVarMap := make(map[string]string)
	for _, v := range loaded.Variables {
		loadedVarMap[v.Key] = v.Value
	}
	for _, v := range original.Variables {
		if got, ok := loadedVarMap[v.Key]; !ok {
			t.Errorf("variable %q not found in loaded collection", v.Key)
		} else if got != v.Value {
			t.Errorf("variable %q: expected %q, got %q", v.Key, v.Value, got)
		}
	}

	// Verify auth.
	if loaded.Auth == nil {
		t.Fatal("expected auth to be set")
	}
	if loaded.Auth.Type != original.Auth.Type {
		t.Errorf("auth type: expected %q, got %q", original.Auth.Type, loaded.Auth.Type)
	}
	if loaded.Auth.Bearer.Token != original.Auth.Bearer.Token {
		t.Errorf("bearer token: expected %q, got %q", original.Auth.Bearer.Token, loaded.Auth.Bearer.Token)
	}

	// Verify headers.
	if len(loaded.Headers) != len(original.Headers) {
		t.Fatalf("headers: expected %d, got %d", len(original.Headers), len(loaded.Headers))
	}
	for i, h := range original.Headers {
		if loaded.Headers[i].Key != h.Key || loaded.Headers[i].Value != h.Value {
			t.Errorf("header %d: expected %s: %s, got %s: %s",
				i, h.Key, h.Value, loaded.Headers[i].Key, loaded.Headers[i].Value)
		}
	}

	// Verify requests.
	if len(loaded.Requests) != len(original.Requests) {
		t.Fatalf("requests: expected %d, got %d", len(original.Requests), len(loaded.Requests))
	}
	if loaded.Requests[0].Name != original.Requests[0].Name {
		t.Errorf("request name: expected %q, got %q", original.Requests[0].Name, loaded.Requests[0].Name)
	}

	// Verify folders.
	if len(loaded.Folders) != len(original.Folders) {
		t.Fatalf("folders: expected %d, got %d", len(original.Folders), len(loaded.Folders))
	}
	folder := loaded.Folders[0]
	if folder.Name != "Users" {
		t.Errorf("folder name: expected 'Users', got %q", folder.Name)
	}
	if len(folder.Requests) != 2 {
		t.Fatalf("folder requests: expected 2, got %d", len(folder.Requests))
	}
	if folder.Requests[0].Name != "List Users" {
		t.Errorf("folder request 0: expected 'List Users', got %q", folder.Requests[0].Name)
	}
	if folder.Requests[1].Config.ContentType != "application/json" {
		t.Errorf("folder request 1 content_type: expected 'application/json', got %q", folder.Requests[1].Config.ContentType)
	}
	if string(folder.Requests[1].Config.Body) != `{"name": "John"}` {
		t.Errorf("folder request 1 body: expected '{\"name\": \"John\"}', got %q", string(folder.Requests[1].Config.Body))
	}
}

func TestReadCollection_NestedFolders(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "nested.yaml")

	content := `name: Nested Test
folders:
  - name: Level1
    folders:
      - name: Level2
        requests:
          - name: Deep Request
            method: GET
            url: "https://api.example.com/deep"
`
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	fs := storage.NewFilesystem()
	col, err := fs.ReadCollection(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(col.Folders) != 1 {
		t.Fatalf("expected 1 top-level folder, got %d", len(col.Folders))
	}
	if col.Folders[0].Name != "Level1" {
		t.Errorf("expected folder name 'Level1', got %q", col.Folders[0].Name)
	}

	if len(col.Folders[0].Folders) != 1 {
		t.Fatalf("expected 1 nested folder, got %d", len(col.Folders[0].Folders))
	}
	nested := col.Folders[0].Folders[0]
	if nested.Name != "Level2" {
		t.Errorf("expected nested folder name 'Level2', got %q", nested.Name)
	}

	if len(nested.Requests) != 1 {
		t.Fatalf("expected 1 request in nested folder, got %d", len(nested.Requests))
	}
	if nested.Requests[0].Name != "Deep Request" {
		t.Errorf("expected request name 'Deep Request', got %q", nested.Requests[0].Name)
	}
}

func TestReadCollection_AuthInheritance(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "auth-inherit.yaml")

	content := `name: Auth Inherit
auth:
  type: basic
  basic:
    username: admin
    password: secret
folders:
  - name: Admin
    auth:
      type: bearer
      bearer:
        token: "admin-token"
    requests:
      - name: Admin Action
        method: GET
        url: "https://api.example.com/admin"
`
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	fs := storage.NewFilesystem()
	col, err := fs.ReadCollection(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Collection-level auth.
	if col.Auth == nil || col.Auth.Type != domain.AuthBasic {
		t.Fatalf("expected collection auth type 'basic', got %+v", col.Auth)
	}
	if col.Auth.Basic == nil || col.Auth.Basic.Username != "admin" || col.Auth.Basic.Password != "secret" {
		t.Errorf("unexpected basic auth: %+v", col.Auth.Basic)
	}

	// Folder-level auth override.
	if len(col.Folders) != 1 {
		t.Fatalf("expected 1 folder, got %d", len(col.Folders))
	}
	folder := col.Folders[0]
	if folder.Auth == nil || folder.Auth.Type != domain.AuthBearer {
		t.Fatalf("expected folder auth type 'bearer', got %+v", folder.Auth)
	}
	if folder.Auth.Bearer == nil || folder.Auth.Bearer.Token != "admin-token" {
		t.Errorf("unexpected bearer auth: %+v", folder.Auth.Bearer)
	}
}

func TestListCollections_ReturnsYAMLFiles(t *testing.T) {
	dir := t.TempDir()

	// Create some .yaml files and a non-yaml file.
	for _, name := range []string{"api-tests.yaml", "integration.yaml", "notes.txt"} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(""), 0644); err != nil {
			t.Fatal(err)
		}
	}

	fs := storage.NewFilesystem()
	names, err := fs.ListCollections(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(names) != 2 {
		t.Fatalf("expected 2 collections, got %d: %v", len(names), names)
	}

	nameSet := make(map[string]bool)
	for _, n := range names {
		nameSet[n] = true
	}
	if !nameSet["api-tests"] {
		t.Error("expected 'api-tests' in list")
	}
	if !nameSet["integration"] {
		t.Error("expected 'integration' in list")
	}
}

func TestListCollections_EmptyDir(t *testing.T) {
	dir := t.TempDir()

	fs := storage.NewFilesystem()
	names, err := fs.ListCollections(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(names) != 0 {
		t.Errorf("expected 0 collections, got %d", len(names))
	}
}

func TestRoundTrip_ExampleResponse(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "mock-collection.yaml")

	original := domain.Collection{
		Name: "Mock API",
		Requests: []domain.SavedRequest{
			{
				Name: "Get Users",
				Config: domain.RequestConfig{
					Method: domain.MethodGet,
					URL:    "https://api.example.com/users",
				},
				Response: &domain.ExampleResponse{
					StatusCode: 200,
					Headers:    []domain.Header{{Key: "Content-Type", Value: "application/json"}},
					Body:       `[{"id":1}]`,
				},
			},
			{
				Name: "Health",
				Config: domain.RequestConfig{
					Method: domain.MethodGet,
					URL:    "https://api.example.com/health",
				},
				// No response defined.
			},
		},
	}

	fs := storage.NewFilesystem()
	if err := fs.WriteCollection(path, original); err != nil {
		t.Fatalf("write error: %v", err)
	}

	loaded, err := fs.ReadCollection(path)
	if err != nil {
		t.Fatalf("read error: %v", err)
	}

	if len(loaded.Requests) != 2 {
		t.Fatalf("expected 2 requests, got %d", len(loaded.Requests))
	}

	// First request should have response.
	r := loaded.Requests[0]
	if r.Response == nil {
		t.Fatal("expected response to be set for first request")
	}
	if r.Response.StatusCode != 200 {
		t.Errorf("status code: expected 200, got %d", r.Response.StatusCode)
	}
	if len(r.Response.Headers) != 1 {
		t.Fatalf("headers: expected 1, got %d", len(r.Response.Headers))
	}
	if r.Response.Headers[0].Key != "Content-Type" || r.Response.Headers[0].Value != "application/json" {
		t.Errorf("unexpected header: %+v", r.Response.Headers[0])
	}
	if r.Response.Body != `[{"id":1}]` {
		t.Errorf("body: expected '[{\"id\":1}]', got %q", r.Response.Body)
	}

	// Second request should have nil response.
	if loaded.Requests[1].Response != nil {
		t.Errorf("expected nil response for second request, got %+v", loaded.Requests[1].Response)
	}
}

func TestReadCollection_WithResponseYAML(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "mock.yaml")

	content := `name: Mock Test
requests:
  - name: Get Users
    method: GET
    url: "https://api.example.com/users"
    response:
      status_code: 200
      headers:
        - key: Content-Type
          value: application/json
      body: '[{"id":1}]'
`
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	fs := storage.NewFilesystem()
	col, err := fs.ReadCollection(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(col.Requests) != 1 {
		t.Fatalf("expected 1 request, got %d", len(col.Requests))
	}
	r := col.Requests[0]
	if r.Response == nil {
		t.Fatal("expected response to be set")
	}
	if r.Response.StatusCode != 200 {
		t.Errorf("expected status 200, got %d", r.Response.StatusCode)
	}
	if r.Response.Body != `[{"id":1}]` {
		t.Errorf("expected body '[{\"id\":1}]', got %q", r.Response.Body)
	}
}

func TestReadCollection_NonexistentFile(t *testing.T) {
	fs := storage.NewFilesystem()
	_, err := fs.ReadCollection("/nonexistent/path/collection.yaml")
	if err == nil {
		t.Fatal("expected error for nonexistent file, got nil")
	}
}
