package commands_test

import (
	"bytes"
	"context"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ye-kart/reqflow/internal/adapters/cli/commands"
	"github.com/ye-kart/reqflow/internal/adapters/storage"
	"github.com/ye-kart/reqflow/internal/app"
	"github.com/ye-kart/reqflow/internal/domain"
	featurehttp "github.com/ye-kart/reqflow/internal/features/http"
	"github.com/ye-kart/reqflow/internal/features/runner"
)

func newTestAppWithRunner(mock *mockHTTPClient, store *storage.Filesystem) *app.App {
	return &app.App{
		HTTPExecutor:     featurehttp.NewExecutor(mock),
		Storage:          store,
		Runner:           runner.New(mock),
		CollectionRunner: runner.NewCollectionRunner(mock),
	}
}

func TestCollectionRun_RequiresName(t *testing.T) {
	a := &app.App{}
	root := commands.NewRootCommand(a)
	root.SetArgs([]string{"collection", "run"})

	var out bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&out)

	err := root.Execute()
	if err == nil {
		t.Fatal("expected error for missing collection name, got nil")
	}
}

func TestCollectionRun_ExecutesAndShowsSummary(t *testing.T) {
	dir := t.TempDir()
	store := storage.NewFilesystem()

	col := domain.Collection{
		Name: "my-api",
		Requests: []domain.SavedRequest{
			{
				Name: "Health Check",
				Config: domain.RequestConfig{
					Method: domain.MethodGet,
					URL:    "https://api.example.com/health",
				},
			},
			{
				Name: "List Users",
				Config: domain.RequestConfig{
					Method: domain.MethodGet,
					URL:    "https://api.example.com/users",
				},
			},
		},
	}
	if err := store.WriteCollection(filepath.Join(dir, "my-api.yaml"), col); err != nil {
		t.Fatal(err)
	}

	mock := &mockHTTPClient{
		doFunc: func(_ context.Context, req domain.HTTPRequest) (domain.HTTPResponse, error) {
			return domain.HTTPResponse{StatusCode: 200, Body: []byte(`{"ok":true}`)}, nil
		},
	}
	a := newTestAppWithRunner(mock, store)
	root := commands.NewRootCommand(a)

	buf := new(bytes.Buffer)
	root.SetOut(buf)
	root.SetErr(buf)
	root.SetArgs([]string{"collection", "run", "my-api", "--collection-dir", dir, "--no-color"})

	if err := root.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "Health Check") {
		t.Errorf("expected output to contain 'Health Check', got:\n%s", output)
	}
	if !strings.Contains(output, "List Users") {
		t.Errorf("expected output to contain 'List Users', got:\n%s", output)
	}
	if !strings.Contains(output, "2 passed") {
		t.Errorf("expected output to contain '2 passed', got:\n%s", output)
	}
}

func TestCollectionRun_FolderFilter(t *testing.T) {
	dir := t.TempDir()
	store := storage.NewFilesystem()

	col := domain.Collection{
		Name: "my-api",
		Requests: []domain.SavedRequest{
			{
				Name:   "Root",
				Config: domain.RequestConfig{Method: domain.MethodGet, URL: "https://api.example.com/"},
			},
		},
		Folders: []domain.Folder{
			{
				Name: "Users",
				Requests: []domain.SavedRequest{
					{
						Name:   "List Users",
						Config: domain.RequestConfig{Method: domain.MethodGet, URL: "https://api.example.com/users"},
					},
				},
			},
			{
				Name: "Products",
				Requests: []domain.SavedRequest{
					{
						Name:   "List Products",
						Config: domain.RequestConfig{Method: domain.MethodGet, URL: "https://api.example.com/products"},
					},
				},
			},
		},
	}
	if err := store.WriteCollection(filepath.Join(dir, "my-api.yaml"), col); err != nil {
		t.Fatal(err)
	}

	callCount := 0
	mock := &mockHTTPClient{
		doFunc: func(_ context.Context, req domain.HTTPRequest) (domain.HTTPResponse, error) {
			callCount++
			return domain.HTTPResponse{StatusCode: 200, Body: []byte(`{}`)}, nil
		},
	}
	a := newTestAppWithRunner(mock, store)
	root := commands.NewRootCommand(a)

	buf := new(bytes.Buffer)
	root.SetOut(buf)
	root.SetErr(buf)
	root.SetArgs([]string{"collection", "run", "my-api", "--folder", "Users", "--collection-dir", dir, "--no-color"})

	if err := root.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := buf.String()
	if callCount != 1 {
		t.Errorf("expected 1 HTTP call (only Users folder), got %d", callCount)
	}
	if !strings.Contains(output, "List Users") {
		t.Errorf("expected output to contain 'List Users', got:\n%s", output)
	}
	if !strings.Contains(output, "1 passed") {
		t.Errorf("expected output to contain '1 passed', got:\n%s", output)
	}
}

func TestCollectionRun_CollectionNotFound(t *testing.T) {
	dir := t.TempDir()
	store := storage.NewFilesystem()

	mock := &mockHTTPClient{
		doFunc: noopDoFunc,
	}
	a := newTestAppWithRunner(mock, store)
	root := commands.NewRootCommand(a)

	buf := new(bytes.Buffer)
	root.SetOut(buf)
	root.SetErr(buf)
	root.SetArgs([]string{"collection", "run", "nonexistent", "--collection-dir", dir})

	err := root.Execute()
	if err == nil {
		t.Fatal("expected error for nonexistent collection, got nil")
	}
}

func TestCollectionRun_ShowsFailure(t *testing.T) {
	dir := t.TempDir()
	store := storage.NewFilesystem()

	col := domain.Collection{
		Name: "fail-api",
		Requests: []domain.SavedRequest{
			{
				Name: "Failing Request",
				Config: domain.RequestConfig{
					Method: domain.MethodGet,
					URL:    "https://api.example.com/fail",
				},
			},
		},
	}
	if err := store.WriteCollection(filepath.Join(dir, "fail-api.yaml"), col); err != nil {
		t.Fatal(err)
	}

	mock := &mockHTTPClient{
		doFunc: func(_ context.Context, req domain.HTTPRequest) (domain.HTTPResponse, error) {
			return domain.HTTPResponse{StatusCode: 500, Body: []byte(`{"error":"oops"}`)}, nil
		},
	}
	a := newTestAppWithRunner(mock, store)
	root := commands.NewRootCommand(a)

	buf := new(bytes.Buffer)
	root.SetOut(buf)
	root.SetErr(buf)
	root.SetArgs([]string{"collection", "run", "fail-api", "--collection-dir", dir, "--no-color"})

	if err := root.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "FAIL") {
		t.Errorf("expected output to contain 'FAIL', got:\n%s", output)
	}
	if !strings.Contains(output, "1 failed") {
		t.Errorf("expected output to contain '1 failed', got:\n%s", output)
	}
}
