package exporter

import (
	"encoding/json"
	"testing"

	"github.com/ye-kart/reqflow/internal/core/importer"
	"github.com/ye-kart/reqflow/internal/domain"
)

func TestExportPostman_ValidJSON(t *testing.T) {
	col := testCollection()

	data, err := ExportPostman(col)
	if err != nil {
		t.Fatalf("ExportPostman() error = %v", err)
	}

	var raw map[string]interface{}
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatalf("output is not valid JSON: %v", err)
	}

	info, ok := raw["info"].(map[string]interface{})
	if !ok {
		t.Fatal("missing info section")
	}
	if info["name"] != "Pet Store API" {
		t.Errorf("info.name = %q, want %q", info["name"], "Pet Store API")
	}
}

func TestExportPostman_HasItems(t *testing.T) {
	col := testCollection()

	data, err := ExportPostman(col)
	if err != nil {
		t.Fatalf("ExportPostman() error = %v", err)
	}

	var raw map[string]interface{}
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatalf("output is not valid JSON: %v", err)
	}

	items, ok := raw["item"].([]interface{})
	if !ok {
		t.Fatal("missing item array")
	}
	// testCollection has 1 root request + 2 folders
	if len(items) != 3 {
		t.Errorf("item count = %d, want 3", len(items))
	}
}

func TestExportPostman_RoundTrip(t *testing.T) {
	col := domain.Collection{
		Name: "Round Trip Test",
		Requests: []domain.SavedRequest{
			{
				Name: "Test POST",
				Config: domain.RequestConfig{
					Method: domain.MethodPost,
					URL:    "https://api.example.com/test",
					Headers: []domain.Header{
						{Key: "Content-Type", Value: "application/json"},
					},
					Body: []byte(`{"key":"value"}`),
				},
			},
		},
		Folders: []domain.Folder{
			{
				Name: "Test Folder",
				Requests: []domain.SavedRequest{
					{
						Name: "Folder Request",
						Config: domain.RequestConfig{
							Method: domain.MethodGet,
							URL:    "https://api.example.com/folder",
						},
					},
				},
			},
		},
	}

	data, err := ExportPostman(col)
	if err != nil {
		t.Fatalf("ExportPostman() error = %v", err)
	}

	reimported, err := importer.ParsePostmanCollection(data)
	if err != nil {
		t.Fatalf("ParsePostmanCollection() error = %v", err)
	}

	if reimported.Name != col.Name {
		t.Errorf("Name = %q, want %q", reimported.Name, col.Name)
	}
	if len(reimported.Requests) != 1 {
		t.Fatalf("Requests len = %d, want 1", len(reimported.Requests))
	}
	if reimported.Requests[0].Config.Method != domain.MethodPost {
		t.Errorf("Method = %q, want POST", reimported.Requests[0].Config.Method)
	}
	if len(reimported.Folders) != 1 {
		t.Fatalf("Folders len = %d, want 1", len(reimported.Folders))
	}
	if reimported.Folders[0].Name != "Test Folder" {
		t.Errorf("Folder.Name = %q, want %q", reimported.Folders[0].Name, "Test Folder")
	}
}

func TestExportPostman_WithAuth(t *testing.T) {
	col := domain.Collection{
		Name: "Auth Export",
		Requests: []domain.SavedRequest{
			{
				Name: "Bearer Request",
				Config: domain.RequestConfig{
					Method: domain.MethodGet,
					URL:    "https://api.example.com/secure",
					Auth: &domain.AuthConfig{
						Type: domain.AuthBearer,
						Bearer: &domain.BearerAuthConfig{
							Token: "my-token",
						},
					},
				},
			},
		},
	}

	data, err := ExportPostman(col)
	if err != nil {
		t.Fatalf("ExportPostman() error = %v", err)
	}

	reimported, err := importer.ParsePostmanCollection(data)
	if err != nil {
		t.Fatalf("reimport error: %v", err)
	}

	auth := reimported.Requests[0].Config.Auth
	if auth == nil {
		t.Fatal("Auth is nil after round-trip")
	}
	if auth.Type != domain.AuthBearer {
		t.Errorf("Auth.Type = %q, want bearer", auth.Type)
	}
	if auth.Bearer.Token != "my-token" {
		t.Errorf("Token = %q, want %q", auth.Bearer.Token, "my-token")
	}
}

func TestExportPostman_EmptyCollection(t *testing.T) {
	col := domain.Collection{Name: "Empty"}

	data, err := ExportPostman(col)
	if err != nil {
		t.Fatalf("ExportPostman() error = %v", err)
	}

	var raw map[string]interface{}
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}

	info := raw["info"].(map[string]interface{})
	if info["name"] != "Empty" {
		t.Errorf("info.name = %q, want %q", info["name"], "Empty")
	}
}
