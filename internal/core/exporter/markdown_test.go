package exporter

import (
	"strings"
	"testing"

	"github.com/ye-kart/reqflow/internal/domain"
)

func testCollection() domain.Collection {
	return domain.Collection{
		Name:        "Pet Store API",
		Description: "API for managing a pet store.",
		Version:     "1.0.0",
		Requests: []domain.SavedRequest{
			{
				Name:        "Health Check",
				Description: "Check service health",
				Config: domain.RequestConfig{
					Method: domain.MethodGet,
					URL:    "https://api.example.com/health",
				},
			},
		},
		Folders: []domain.Folder{
			{
				Name:        "Users",
				Description: "User management endpoints",
				Requests: []domain.SavedRequest{
					{
						Name:        "List Users",
						Description: "Get all users",
						Config: domain.RequestConfig{
							Method: domain.MethodGet,
							URL:    "https://api.example.com/users",
							Headers: []domain.Header{
								{Key: "Accept", Value: "application/json"},
							},
						},
					},
					{
						Name:        "Create User",
						Description: "Create a new user",
						Config: domain.RequestConfig{
							Method: domain.MethodPost,
							URL:    "https://api.example.com/users",
							Headers: []domain.Header{
								{Key: "Content-Type", Value: "application/json"},
							},
							Body: []byte(`{"name":"Alice","email":"alice@example.com"}`),
						},
					},
				},
			},
			{
				Name: "Pets",
				Requests: []domain.SavedRequest{
					{
						Name: "List Pets",
						Config: domain.RequestConfig{
							Method: domain.MethodGet,
							URL:    "https://api.example.com/pets",
						},
					},
				},
			},
		},
	}
}

func TestExportMarkdown_ContainsAllRequestNames(t *testing.T) {
	col := testCollection()

	md, err := ExportMarkdown(col)
	if err != nil {
		t.Fatalf("ExportMarkdown() error = %v", err)
	}

	content := string(md)
	names := []string{"Health Check", "List Users", "Create User", "List Pets"}
	for _, name := range names {
		if !strings.Contains(content, name) {
			t.Errorf("markdown output missing request name %q", name)
		}
	}
}

func TestExportMarkdown_ContainsURLsAndMethods(t *testing.T) {
	col := testCollection()

	md, err := ExportMarkdown(col)
	if err != nil {
		t.Fatalf("ExportMarkdown() error = %v", err)
	}

	content := string(md)

	checks := []string{
		"GET /health",
		"GET /users",
		"POST /users",
		"GET /pets",
	}
	for _, check := range checks {
		if !strings.Contains(content, check) {
			t.Errorf("markdown output missing %q", check)
		}
	}
}

func TestExportMarkdown_HeadersTableGenerated(t *testing.T) {
	col := testCollection()

	md, err := ExportMarkdown(col)
	if err != nil {
		t.Fatalf("ExportMarkdown() error = %v", err)
	}

	content := string(md)

	// Should contain a headers table
	if !strings.Contains(content, "| Key | Value |") {
		t.Error("markdown output missing headers table header")
	}
	if !strings.Contains(content, "| Accept | application/json |") {
		t.Error("markdown output missing Accept header row")
	}
	if !strings.Contains(content, "| Content-Type | application/json |") {
		t.Error("markdown output missing Content-Type header row")
	}
}

func TestExportMarkdown_FoldersCreateSections(t *testing.T) {
	col := testCollection()

	md, err := ExportMarkdown(col)
	if err != nil {
		t.Fatalf("ExportMarkdown() error = %v", err)
	}

	content := string(md)

	if !strings.Contains(content, "## Folder: Users") {
		t.Error("markdown output missing folder section for Users")
	}
	if !strings.Contains(content, "## Folder: Pets") {
		t.Error("markdown output missing folder section for Pets")
	}
}

func TestExportMarkdown_CollectionTitle(t *testing.T) {
	col := testCollection()

	md, err := ExportMarkdown(col)
	if err != nil {
		t.Fatalf("ExportMarkdown() error = %v", err)
	}

	content := string(md)

	if !strings.Contains(content, "# Pet Store API") {
		t.Error("markdown output missing collection title")
	}
	if !strings.Contains(content, "API for managing a pet store.") {
		t.Error("markdown output missing collection description")
	}
}

func TestExportMarkdown_BodyExample(t *testing.T) {
	col := testCollection()

	md, err := ExportMarkdown(col)
	if err != nil {
		t.Fatalf("ExportMarkdown() error = %v", err)
	}

	content := string(md)

	if !strings.Contains(content, `"name":"Alice"`) {
		t.Error("markdown output missing request body example")
	}
}

func TestExportMarkdown_EmptyCollection(t *testing.T) {
	col := domain.Collection{
		Name: "Empty",
	}

	md, err := ExportMarkdown(col)
	if err != nil {
		t.Fatalf("ExportMarkdown() error = %v", err)
	}

	content := string(md)
	if !strings.Contains(content, "# Empty") {
		t.Error("markdown output missing collection title for empty collection")
	}
}
