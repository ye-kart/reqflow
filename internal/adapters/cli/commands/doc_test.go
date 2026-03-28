package commands_test

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ye-kart/reqflow/internal/adapters/cli/commands"
	"github.com/ye-kart/reqflow/internal/adapters/storage"
	"github.com/ye-kart/reqflow/internal/domain"
)

func writeTestCollection(t *testing.T, dir string, store *storage.Filesystem) {
	t.Helper()
	col := domain.Collection{
		Name:        "my-api",
		Description: "My test API",
		Requests: []domain.SavedRequest{
			{
				Name: "Health Check",
				Config: domain.RequestConfig{
					Method: domain.MethodGet,
					URL:    "https://api.example.com/health",
				},
			},
		},
		Folders: []domain.Folder{
			{
				Name: "Users",
				Requests: []domain.SavedRequest{
					{
						Name: "List Users",
						Config: domain.RequestConfig{
							Method: domain.MethodGet,
							URL:    "https://api.example.com/users",
							Headers: []domain.Header{
								{Key: "Accept", Value: "application/json"},
							},
						},
					},
				},
			},
		},
	}
	if err := store.WriteCollection(filepath.Join(dir, "my-api.yaml"), col); err != nil {
		t.Fatal(err)
	}
}

func TestDocCommand_Markdown(t *testing.T) {
	dir := t.TempDir()
	store := storage.NewFilesystem()
	writeTestCollection(t, dir, store)

	mock := &mockHTTPClient{doFunc: noopDoFunc}
	a := newTestAppWithStorage(mock, store)
	root := commands.NewRootCommand(a)

	buf := new(bytes.Buffer)
	root.SetOut(buf)
	root.SetErr(buf)
	root.SetArgs([]string{"doc", "my-api", "--format", "markdown", "--collection-dir", dir})

	if err := root.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "# my-api") {
		t.Error("expected markdown title")
	}
	if !strings.Contains(output, "Health Check") {
		t.Error("expected request name in markdown")
	}
	if !strings.Contains(output, "Folder: Users") {
		t.Error("expected folder section in markdown")
	}
}

func TestDocCommand_HTML(t *testing.T) {
	dir := t.TempDir()
	store := storage.NewFilesystem()
	writeTestCollection(t, dir, store)

	mock := &mockHTTPClient{doFunc: noopDoFunc}
	a := newTestAppWithStorage(mock, store)
	root := commands.NewRootCommand(a)

	buf := new(bytes.Buffer)
	root.SetOut(buf)
	root.SetErr(buf)
	root.SetArgs([]string{"doc", "my-api", "--format", "html", "--collection-dir", dir})

	if err := root.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "<html") {
		t.Error("expected HTML output")
	}
	if !strings.Contains(output, "my-api") {
		t.Error("expected collection name in HTML")
	}
}

func TestDocCommand_OpenAPI(t *testing.T) {
	dir := t.TempDir()
	store := storage.NewFilesystem()
	writeTestCollection(t, dir, store)

	mock := &mockHTTPClient{doFunc: noopDoFunc}
	a := newTestAppWithStorage(mock, store)
	root := commands.NewRootCommand(a)

	buf := new(bytes.Buffer)
	root.SetOut(buf)
	root.SetErr(buf)
	root.SetArgs([]string{"doc", "my-api", "--format", "openapi", "--collection-dir", dir})

	if err := root.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "openapi:") {
		t.Error("expected OpenAPI output")
	}
	if !strings.Contains(output, "paths:") {
		t.Error("expected paths in OpenAPI output")
	}
}

func TestDocCommand_HTMLOutputToFile(t *testing.T) {
	dir := t.TempDir()
	store := storage.NewFilesystem()
	writeTestCollection(t, dir, store)

	outputFile := filepath.Join(dir, "docs.html")

	mock := &mockHTTPClient{doFunc: noopDoFunc}
	a := newTestAppWithStorage(mock, store)
	root := commands.NewRootCommand(a)

	buf := new(bytes.Buffer)
	root.SetOut(buf)
	root.SetErr(buf)
	root.SetArgs([]string{"doc", "my-api", "--format", "html", "--doc-output", outputFile, "--collection-dir", dir})

	if err := root.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	data, err := os.ReadFile(outputFile)
	if err != nil {
		t.Fatalf("failed to read output file: %v", err)
	}

	content := string(data)
	if !strings.Contains(content, "<html") {
		t.Error("expected HTML in output file")
	}
}

func TestDocCommand_DefaultFormatIsMarkdown(t *testing.T) {
	dir := t.TempDir()
	store := storage.NewFilesystem()
	writeTestCollection(t, dir, store)

	mock := &mockHTTPClient{doFunc: noopDoFunc}
	a := newTestAppWithStorage(mock, store)
	root := commands.NewRootCommand(a)

	buf := new(bytes.Buffer)
	root.SetOut(buf)
	root.SetErr(buf)
	root.SetArgs([]string{"doc", "my-api", "--collection-dir", dir})

	if err := root.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "# my-api") {
		t.Error("expected markdown output as default format")
	}
}

func TestDocCommand_InvalidFormat(t *testing.T) {
	dir := t.TempDir()
	store := storage.NewFilesystem()
	writeTestCollection(t, dir, store)

	mock := &mockHTTPClient{doFunc: noopDoFunc}
	a := newTestAppWithStorage(mock, store)
	root := commands.NewRootCommand(a)

	buf := new(bytes.Buffer)
	root.SetOut(buf)
	root.SetErr(buf)
	root.SetArgs([]string{"doc", "my-api", "--format", "csv", "--collection-dir", dir})

	err := root.Execute()
	if err == nil {
		t.Fatal("expected error for invalid format")
	}
	if !strings.Contains(err.Error(), "unsupported") {
		t.Errorf("expected 'unsupported' in error, got: %v", err)
	}
}

func TestDocCommand_Registered(t *testing.T) {
	mock := &mockHTTPClient{doFunc: noopDoFunc}
	a := newTestAppWithStorage(mock, storage.NewFilesystem())
	root := commands.NewRootCommand(a)

	found := false
	for _, cmd := range root.Commands() {
		if cmd.Name() == "doc" {
			found = true
		}
	}
	if !found {
		t.Error("root command missing 'doc' subcommand")
	}
}
