package commands_test

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/ye-kart/reqflow/internal/adapters/cli/commands"
	"github.com/ye-kart/reqflow/internal/adapters/storage"
	"github.com/ye-kart/reqflow/internal/domain"
)

func TestImportFileCommand_AutoDetect(t *testing.T) {
	tmpDir := t.TempDir()
	postmanFile := filepath.Join(tmpDir, "collection.json")
	data := []byte(`{
		"info": {"name": "CLI Import Test","schema":"https://schema.getpostman.com/json/collection/v2.1.0/collection.json"},
		"item": [
			{
				"name": "Test Request",
				"request": {
					"method": "GET",
					"url": {"raw": "https://example.com/api"}
				}
			}
		]
	}`)
	if err := os.WriteFile(postmanFile, data, 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	mock := &mockHTTPClient{doFunc: noopDoFunc}
	store := storage.NewFilesystem()
	a := newTestAppWithStorage(mock, store)
	root := commands.NewRootCommand(a)

	buf := new(bytes.Buffer)
	root.SetOut(buf)
	root.SetErr(buf)
	root.SetArgs([]string{"import", "file", postmanFile})

	err := root.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := buf.String()
	if !containsAll(output, "CLI Import Test", "1 request") {
		t.Errorf("output = %q, expected collection name and request count", output)
	}
}

func TestImportFileCommand_ExplicitFormat(t *testing.T) {
	tmpDir := t.TempDir()
	harFile := filepath.Join(tmpDir, "archive.har")
	data := []byte(`{
		"log": {
			"entries": [
				{
					"request": {
						"method": "GET",
						"url": "https://example.com/api",
						"headers": []
					}
				}
			]
		}
	}`)
	if err := os.WriteFile(harFile, data, 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	mock := &mockHTTPClient{doFunc: noopDoFunc}
	store := storage.NewFilesystem()
	a := newTestAppWithStorage(mock, store)
	root := commands.NewRootCommand(a)

	buf := new(bytes.Buffer)
	root.SetOut(buf)
	root.SetErr(buf)
	root.SetArgs([]string{"import", "file", harFile, "--format", "har"})

	err := root.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := buf.String()
	if !containsAll(output, "HAR Import", "1 request") {
		t.Errorf("output = %q, expected collection info", output)
	}
}

func TestImportFileCommand_SaveToCollection(t *testing.T) {
	tmpDir := t.TempDir()
	colDir := filepath.Join(tmpDir, "collections")
	if err := os.MkdirAll(colDir, 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	postmanFile := filepath.Join(tmpDir, "input.json")
	data := []byte(`{
		"info": {"name": "Save Test","schema":"https://schema.getpostman.com/json/collection/v2.1.0/collection.json"},
		"item": [
			{
				"name": "Request 1",
				"request": {
					"method": "GET",
					"url": {"raw": "https://example.com"}
				}
			}
		]
	}`)
	if err := os.WriteFile(postmanFile, data, 0644); err != nil {
		t.Fatalf("write: %v", err)
	}

	mock := &mockHTTPClient{doFunc: noopDoFunc}
	store := storage.NewFilesystem()
	a := newTestAppWithStorage(mock, store)
	root := commands.NewRootCommand(a)

	buf := new(bytes.Buffer)
	root.SetOut(buf)
	root.SetErr(buf)
	root.SetArgs([]string{"import", "file", postmanFile, "--save", "myimport", "--collection-dir", colDir})

	err := root.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify the collection file was created.
	savedPath := filepath.Join(colDir, "myimport.yaml")
	if _, err := os.Stat(savedPath); os.IsNotExist(err) {
		t.Errorf("expected collection file at %s, not found", savedPath)
	}
}

func TestExportCollectionCommand_Postman(t *testing.T) {
	tmpDir := t.TempDir()
	colDir := filepath.Join(tmpDir, "collections")
	store := storage.NewFilesystem()

	// Write a collection using the storage layer.
	col := domain.Collection{
		Name: "Export Test",
		Requests: []domain.SavedRequest{
			{
				Name: "Test GET",
				Config: domain.RequestConfig{
					Method: domain.MethodGet,
					URL:    "https://example.com/api",
				},
			},
		},
	}
	if err := os.MkdirAll(colDir, 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := store.WriteCollection(filepath.Join(colDir, "myapi.yaml"), col); err != nil {
		t.Fatalf("write collection: %v", err)
	}

	mock := &mockHTTPClient{doFunc: noopDoFunc}
	a := newTestAppWithStorage(mock, store)
	root := commands.NewRootCommand(a)

	buf := new(bytes.Buffer)
	root.SetOut(buf)
	root.SetErr(buf)
	root.SetArgs([]string{"export", "collection", "myapi", "--format", "postman", "--collection-dir", colDir})

	err := root.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify the output is valid Postman JSON.
	output := buf.Bytes()
	var raw map[string]interface{}
	if err := json.Unmarshal(output, &raw); err != nil {
		t.Fatalf("output is not valid JSON: %v\noutput: %s", err, string(output))
	}

	info, ok := raw["info"].(map[string]interface{})
	if !ok {
		t.Fatal("missing info section in Postman output")
	}
	if info["name"] != "Export Test" {
		t.Errorf("info.name = %q, want %q", info["name"], "Export Test")
	}
}

func TestExportCollectionCommand_OpenAPI(t *testing.T) {
	tmpDir := t.TempDir()
	colDir := filepath.Join(tmpDir, "collections")
	store := storage.NewFilesystem()

	col := domain.Collection{
		Name: "OpenAPI Export",
		Requests: []domain.SavedRequest{
			{
				Name: "List",
				Config: domain.RequestConfig{
					Method: domain.MethodGet,
					URL:    "https://example.com/items",
				},
			},
		},
	}
	if err := os.MkdirAll(colDir, 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := store.WriteCollection(filepath.Join(colDir, "openapi-test.yaml"), col); err != nil {
		t.Fatalf("write collection: %v", err)
	}

	mock := &mockHTTPClient{doFunc: noopDoFunc}
	a := newTestAppWithStorage(mock, store)
	root := commands.NewRootCommand(a)

	buf := new(bytes.Buffer)
	root.SetOut(buf)
	root.SetErr(buf)
	root.SetArgs([]string{"export", "collection", "openapi-test", "--format", "openapi", "--collection-dir", colDir})

	err := root.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := buf.String()
	if !containsAll(output, "openapi:", "paths:") {
		t.Errorf("output missing OpenAPI markers, got: %s", output)
	}
}

func TestExportCollectionCommand_Curl(t *testing.T) {
	tmpDir := t.TempDir()
	colDir := filepath.Join(tmpDir, "collections")
	store := storage.NewFilesystem()

	col := domain.Collection{
		Name: "Curl Export",
		Requests: []domain.SavedRequest{
			{
				Name: "Test",
				Config: domain.RequestConfig{
					Method: domain.MethodGet,
					URL:    "https://example.com/api",
				},
			},
		},
	}
	if err := os.MkdirAll(colDir, 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := store.WriteCollection(filepath.Join(colDir, "curl-test.yaml"), col); err != nil {
		t.Fatalf("write collection: %v", err)
	}

	mock := &mockHTTPClient{doFunc: noopDoFunc}
	a := newTestAppWithStorage(mock, store)
	root := commands.NewRootCommand(a)

	buf := new(bytes.Buffer)
	root.SetOut(buf)
	root.SetErr(buf)
	root.SetArgs([]string{"export", "collection", "curl-test", "--format", "curl", "--collection-dir", colDir})

	err := root.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := buf.String()
	if !containsAll(output, "curl", "https://example.com/api") {
		t.Errorf("output missing curl command, got: %s", output)
	}
}

func containsAll(s string, substrs ...string) bool {
	for _, sub := range substrs {
		if !bytes.Contains([]byte(s), []byte(sub)) {
			return false
		}
	}
	return true
}
