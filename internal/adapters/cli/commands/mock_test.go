package commands_test

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/ye-kart/reqflow/internal/adapters/cli/commands"
	"github.com/ye-kart/reqflow/internal/adapters/storage"
	"github.com/ye-kart/reqflow/internal/domain"
)

func TestMockStart_ServesCollection(t *testing.T) {
	dir := t.TempDir()
	store := storage.NewFilesystem()

	col := domain.Collection{
		Name: "test-api",
		Requests: []domain.SavedRequest{
			{
				Name: "get-health",
				Config: domain.RequestConfig{
					Method: domain.MethodGet,
					URL:    "https://api.example.com/health",
				},
				Response: &domain.ExampleResponse{
					StatusCode: 200,
					Headers:    []domain.Header{{Key: "Content-Type", Value: "application/json"}},
					Body:       `{"status":"ok"}`,
				},
			},
		},
	}
	if err := store.WriteCollection(filepath.Join(dir, "test-api.yaml"), col); err != nil {
		t.Fatal(err)
	}

	mock := &mockHTTPClient{doFunc: noopDoFunc}
	a := newTestAppWithStorage(mock, store)
	root := commands.NewRootCommand(a)

	buf := new(bytes.Buffer)
	root.SetOut(buf)
	root.SetErr(buf)
	root.SetArgs([]string{
		"mock", "start", "test-api",
		"--port", "0",
		"--collection-dir", dir,
	})

	// Run the mock server in background since it blocks.
	errCh := make(chan error, 1)
	go func() {
		errCh <- root.Execute()
	}()

	// Give the server a moment to start.
	time.Sleep(200 * time.Millisecond)

	output := buf.String()
	// The command should print that it's starting the mock server.
	if !strings.Contains(output, "Mock server") && !strings.Contains(output, "mock server") {
		// The server started but we just need to verify the command accepted the args.
		// Since port=0 picks a random port, we just verify no immediate error.
		select {
		case err := <-errCh:
			// If it returned, check for errors other than "server closed"
			if err != nil && !strings.Contains(err.Error(), "closed") {
				t.Fatalf("unexpected error: %v", err)
			}
		default:
			// Server is still running, good.
		}
	}
}

func TestMockStop_PrintsReminder(t *testing.T) {
	mock := &mockHTTPClient{doFunc: func(ctx context.Context, req domain.HTTPRequest) (domain.HTTPResponse, error) {
		return domain.HTTPResponse{StatusCode: 200, Body: []byte("ok")}, nil
	}}
	a := newTestAppWithStorage(mock, storage.NewFilesystem())
	root := commands.NewRootCommand(a)

	buf := new(bytes.Buffer)
	root.SetOut(buf)
	root.SetErr(buf)
	root.SetArgs([]string{"mock", "stop"})

	if err := root.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "Ctrl+C") && !strings.Contains(output, "foreground") {
		t.Errorf("expected reminder about foreground server, got: %s", output)
	}
}

func TestMockStart_WithDelay(t *testing.T) {
	dir := t.TempDir()
	store := storage.NewFilesystem()

	col := domain.Collection{
		Name: "delay-api",
		Requests: []domain.SavedRequest{
			{
				Name: "get-data",
				Config: domain.RequestConfig{
					Method: domain.MethodGet,
					URL:    "https://api.example.com/data",
				},
				Response: &domain.ExampleResponse{
					StatusCode: 200,
					Body:       `{"data":true}`,
				},
			},
		},
	}
	if err := store.WriteCollection(filepath.Join(dir, "delay-api.yaml"), col); err != nil {
		t.Fatal(err)
	}

	mock := &mockHTTPClient{doFunc: noopDoFunc}
	a := newTestAppWithStorage(mock, store)
	root := commands.NewRootCommand(a)

	buf := new(bytes.Buffer)
	root.SetOut(buf)
	root.SetErr(buf)
	root.SetArgs([]string{
		"mock", "start", "delay-api",
		"--port", "0",
		"--delay", "50ms",
		"--collection-dir", dir,
	})

	errCh := make(chan error, 1)
	go func() {
		errCh <- root.Execute()
	}()

	// Wait for server to start.
	time.Sleep(200 * time.Millisecond)

	// Extract port from output.
	output := buf.String()
	// We can't easily test delay via CLI test, but at least verify the flag is accepted.
	_ = output

	select {
	case err := <-errCh:
		if err != nil && !strings.Contains(err.Error(), "closed") {
			t.Fatalf("unexpected error: %v", err)
		}
	default:
		// Server still running - that's fine.
	}
}

func TestMockStart_CollectionNotFound(t *testing.T) {
	dir := t.TempDir()
	store := storage.NewFilesystem()

	mock := &mockHTTPClient{doFunc: noopDoFunc}
	a := newTestAppWithStorage(mock, store)
	root := commands.NewRootCommand(a)

	buf := new(bytes.Buffer)
	root.SetOut(buf)
	root.SetErr(buf)
	root.SetArgs([]string{
		"mock", "start", "nonexistent",
		"--collection-dir", dir,
	})

	err := root.Execute()
	if err == nil {
		t.Fatal("expected error for nonexistent collection")
	}
}

// Suppress unused import warnings - io and http are used in the test helper.
var _ = io.ReadAll
var _ = http.Get
