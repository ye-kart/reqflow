package importer

import (
	"testing"

	"github.com/ye-kart/reqflow/internal/domain"
)

func TestParseInsomnia_ValidExport(t *testing.T) {
	data := []byte(`{
		"_type": "export",
		"__export_format": 4,
		"resources": [
			{
				"_id": "wrk_1",
				"_type": "workspace",
				"name": "My Workspace"
			},
			{
				"_id": "req_1",
				"_type": "request",
				"parentId": "wrk_1",
				"name": "Get Users",
				"method": "GET",
				"url": "https://api.example.com/users",
				"headers": [
					{"name": "Accept", "value": "application/json"}
				]
			}
		]
	}`)

	col, err := ParseInsomnia(data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if col.Name != "My Workspace" {
		t.Errorf("Name = %q, want %q", col.Name, "My Workspace")
	}
	if len(col.Requests) != 1 {
		t.Fatalf("Requests len = %d, want 1", len(col.Requests))
	}

	req := col.Requests[0]
	if req.Name != "Get Users" {
		t.Errorf("Name = %q, want %q", req.Name, "Get Users")
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
}

func TestParseInsomnia_RequestGroupsAsFolders(t *testing.T) {
	data := []byte(`{
		"_type": "export",
		"__export_format": 4,
		"resources": [
			{
				"_id": "wrk_1",
				"_type": "workspace",
				"name": "API"
			},
			{
				"_id": "fld_1",
				"_type": "request_group",
				"parentId": "wrk_1",
				"name": "Users"
			},
			{
				"_id": "req_1",
				"_type": "request",
				"parentId": "fld_1",
				"name": "List Users",
				"method": "GET",
				"url": "https://api.example.com/users",
				"headers": []
			},
			{
				"_id": "req_2",
				"_type": "request",
				"parentId": "fld_1",
				"name": "Create User",
				"method": "POST",
				"url": "https://api.example.com/users",
				"headers": [],
				"body": {
					"mimeType": "application/json",
					"text": "{\"name\":\"Alice\"}"
				}
			},
			{
				"_id": "fld_2",
				"_type": "request_group",
				"parentId": "fld_1",
				"name": "Admin"
			},
			{
				"_id": "req_3",
				"_type": "request",
				"parentId": "fld_2",
				"name": "List Admins",
				"method": "GET",
				"url": "https://api.example.com/admins",
				"headers": []
			}
		]
	}`)

	col, err := ParseInsomnia(data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(col.Folders) != 1 {
		t.Fatalf("Folders len = %d, want 1", len(col.Folders))
	}

	folder := col.Folders[0]
	if folder.Name != "Users" {
		t.Errorf("Folder.Name = %q, want %q", folder.Name, "Users")
	}
	if len(folder.Requests) != 2 {
		t.Errorf("Folder.Requests len = %d, want 2", len(folder.Requests))
	}
	if len(folder.Folders) != 1 {
		t.Fatalf("Folder.Folders len = %d, want 1", len(folder.Folders))
	}

	subfolder := folder.Folders[0]
	if subfolder.Name != "Admin" {
		t.Errorf("Subfolder.Name = %q, want %q", subfolder.Name, "Admin")
	}
	if len(subfolder.Requests) != 1 {
		t.Errorf("Subfolder.Requests len = %d, want 1", len(subfolder.Requests))
	}
}

func TestParseInsomnia_RequestWithBody(t *testing.T) {
	data := []byte(`{
		"_type": "export",
		"__export_format": 4,
		"resources": [
			{
				"_id": "wrk_1",
				"_type": "workspace",
				"name": "Test"
			},
			{
				"_id": "req_1",
				"_type": "request",
				"parentId": "wrk_1",
				"name": "Create Item",
				"method": "POST",
				"url": "https://api.example.com/items",
				"headers": [
					{"name": "Content-Type", "value": "application/json"}
				],
				"body": {
					"mimeType": "application/json",
					"text": "{\"name\":\"item1\"}"
				}
			}
		]
	}`)

	col, err := ParseInsomnia(data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	req := col.Requests[0]
	if string(req.Config.Body) != `{"name":"item1"}` {
		t.Errorf("Body = %q, want %q", string(req.Config.Body), `{"name":"item1"}`)
	}
}

func TestParseInsomnia_RequestWithAuth(t *testing.T) {
	data := []byte(`{
		"_type": "export",
		"__export_format": 4,
		"resources": [
			{
				"_id": "wrk_1",
				"_type": "workspace",
				"name": "Auth Test"
			},
			{
				"_id": "req_1",
				"_type": "request",
				"parentId": "wrk_1",
				"name": "Secure",
				"method": "GET",
				"url": "https://api.example.com/secure",
				"headers": [],
				"authentication": {
					"type": "bearer",
					"token": "my-token"
				}
			}
		]
	}`)

	col, err := ParseInsomnia(data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	auth := col.Requests[0].Config.Auth
	if auth == nil {
		t.Fatal("Auth is nil")
	}
	if auth.Type != domain.AuthBearer {
		t.Errorf("Auth.Type = %q, want bearer", auth.Type)
	}
	if auth.Bearer.Token != "my-token" {
		t.Errorf("Token = %q, want %q", auth.Bearer.Token, "my-token")
	}
}

func TestParseInsomnia_InvalidJSON(t *testing.T) {
	_, err := ParseInsomnia([]byte(`{invalid`))
	if err == nil {
		t.Fatal("expected error for invalid JSON, got nil")
	}
}

func TestParseInsomnia_NoWorkspace(t *testing.T) {
	data := []byte(`{
		"_type": "export",
		"__export_format": 4,
		"resources": [
			{
				"_id": "req_1",
				"_type": "request",
				"parentId": "wrk_1",
				"name": "Orphan Request",
				"method": "GET",
				"url": "https://api.example.com/orphan",
				"headers": []
			}
		]
	}`)

	col, err := ParseInsomnia(data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if col.Name != "Insomnia Import" {
		t.Errorf("Name = %q, want %q", col.Name, "Insomnia Import")
	}
	// Request should still be captured at root level.
	if len(col.Requests) != 1 {
		t.Fatalf("Requests len = %d, want 1", len(col.Requests))
	}
}
