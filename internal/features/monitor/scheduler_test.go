package monitor

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/ye-kart/reqflow/internal/domain"
	"github.com/ye-kart/reqflow/internal/features/runner"
)

// stubHTTPClient implements driven.HTTPClient for testing.
type stubHTTPClient struct {
	response domain.HTTPResponse
	err      error
}

func (s *stubHTTPClient) Do(_ context.Context, _ domain.HTTPRequest) (domain.HTTPResponse, error) {
	return s.response, s.err
}

func TestScheduler_AddAndList(t *testing.T) {
	dir := t.TempDir()
	r := runner.New(&stubHTTPClient{})
	s := NewScheduler(r, dir)

	m := Monitor{
		Name:         "test-monitor",
		WorkflowPath: "/tmp/workflow.yaml",
		Cron:         "*/5 * * * *",
	}

	if err := s.Add(m); err != nil {
		t.Fatalf("Add() error: %v", err)
	}

	monitors := s.List()
	if len(monitors) != 1 {
		t.Fatalf("List() returned %d monitors, want 1", len(monitors))
	}

	if monitors[0].Name != "test-monitor" {
		t.Errorf("monitor name = %q, want %q", monitors[0].Name, "test-monitor")
	}
	if monitors[0].Cron != "*/5 * * * *" {
		t.Errorf("monitor cron = %q, want %q", monitors[0].Cron, "*/5 * * * *")
	}
}

func TestScheduler_AddDuplicate(t *testing.T) {
	dir := t.TempDir()
	r := runner.New(&stubHTTPClient{})
	s := NewScheduler(r, dir)

	m := Monitor{
		Name:         "dup",
		WorkflowPath: "/tmp/workflow.yaml",
		Cron:         "* * * * *",
	}

	if err := s.Add(m); err != nil {
		t.Fatalf("first Add() error: %v", err)
	}

	err := s.Add(m)
	if err == nil {
		t.Fatal("second Add() expected error for duplicate, got nil")
	}
}

func TestScheduler_AddInvalidCron(t *testing.T) {
	dir := t.TempDir()
	r := runner.New(&stubHTTPClient{})
	s := NewScheduler(r, dir)

	m := Monitor{
		Name:         "bad-cron",
		WorkflowPath: "/tmp/workflow.yaml",
		Cron:         "invalid",
	}

	err := s.Add(m)
	if err == nil {
		t.Fatal("Add() with invalid cron expected error, got nil")
	}
}

func TestScheduler_Remove(t *testing.T) {
	dir := t.TempDir()
	r := runner.New(&stubHTTPClient{})
	s := NewScheduler(r, dir)

	m := Monitor{
		Name:         "to-remove",
		WorkflowPath: "/tmp/workflow.yaml",
		Cron:         "*/5 * * * *",
	}

	if err := s.Add(m); err != nil {
		t.Fatalf("Add() error: %v", err)
	}

	if err := s.Remove("to-remove"); err != nil {
		t.Fatalf("Remove() error: %v", err)
	}

	monitors := s.List()
	if len(monitors) != 0 {
		t.Fatalf("List() returned %d monitors after remove, want 0", len(monitors))
	}
}

func TestScheduler_RemoveNotFound(t *testing.T) {
	dir := t.TempDir()
	r := runner.New(&stubHTTPClient{})
	s := NewScheduler(r, dir)

	err := s.Remove("nonexistent")
	if err == nil {
		t.Fatal("Remove() expected error for nonexistent monitor, got nil")
	}
}

func TestScheduler_Persistence(t *testing.T) {
	dir := t.TempDir()
	r := runner.New(&stubHTTPClient{})
	s := NewScheduler(r, dir)

	m := Monitor{
		Name:         "persist-test",
		WorkflowPath: "/tmp/workflow.yaml",
		Cron:         "*/10 * * * *",
		EnvName:      "prod",
		OnFailure: &WebhookNotify{
			URL:     "https://hooks.example.com/notify",
			Headers: map[string]string{"Authorization": "Bearer tok"},
		},
	}

	if err := s.Add(m); err != nil {
		t.Fatalf("Add() error: %v", err)
	}

	// Verify file was created
	configPath := filepath.Join(dir, "persist-test.yaml")
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		t.Fatalf("expected config file at %s", configPath)
	}

	// Create a new scheduler from the same directory to test loading
	s2 := NewScheduler(r, dir)
	if err := s2.Load(); err != nil {
		t.Fatalf("Load() error: %v", err)
	}

	monitors := s2.List()
	if len(monitors) != 1 {
		t.Fatalf("loaded %d monitors, want 1", len(monitors))
	}

	loaded := monitors[0]
	if loaded.Name != "persist-test" {
		t.Errorf("loaded name = %q, want %q", loaded.Name, "persist-test")
	}
	if loaded.Cron != "*/10 * * * *" {
		t.Errorf("loaded cron = %q, want %q", loaded.Cron, "*/10 * * * *")
	}
	if loaded.EnvName != "prod" {
		t.Errorf("loaded env = %q, want %q", loaded.EnvName, "prod")
	}
	if loaded.OnFailure == nil {
		t.Fatal("loaded OnFailure is nil")
	}
	if loaded.OnFailure.URL != "https://hooks.example.com/notify" {
		t.Errorf("loaded webhook URL = %q, want %q", loaded.OnFailure.URL, "https://hooks.example.com/notify")
	}
}

func TestScheduler_RemoveDeletesFile(t *testing.T) {
	dir := t.TempDir()
	r := runner.New(&stubHTTPClient{})
	s := NewScheduler(r, dir)

	m := Monitor{
		Name:         "file-remove",
		WorkflowPath: "/tmp/workflow.yaml",
		Cron:         "* * * * *",
	}

	if err := s.Add(m); err != nil {
		t.Fatalf("Add() error: %v", err)
	}

	configPath := filepath.Join(dir, "file-remove.yaml")
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		t.Fatalf("expected config file at %s", configPath)
	}

	if err := s.Remove("file-remove"); err != nil {
		t.Fatalf("Remove() error: %v", err)
	}

	if _, err := os.Stat(configPath); !os.IsNotExist(err) {
		t.Fatalf("config file should be deleted after Remove")
	}
}

func TestScheduler_StartAndStop(t *testing.T) {
	dir := t.TempDir()
	r := runner.New(&stubHTTPClient{
		response: domain.HTTPResponse{
			StatusCode: 200,
			Body:       []byte(`{"ok":true}`),
		},
	})
	s := NewScheduler(r, dir)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	// Start with no monitors should return immediately when context expires
	errCh := make(chan error, 1)
	go func() {
		errCh <- s.Start(ctx)
	}()

	// Stop explicitly
	s.Stop()

	select {
	case err := <-errCh:
		if err != nil {
			t.Fatalf("Start() returned error: %v", err)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("Start() did not return after Stop()")
	}
}
