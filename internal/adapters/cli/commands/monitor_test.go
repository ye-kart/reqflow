package commands_test

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ye-kart/reqflow/internal/adapters/cli/commands"
	"github.com/ye-kart/reqflow/internal/adapters/storage"
	"github.com/ye-kart/reqflow/internal/app"
	"github.com/ye-kart/reqflow/internal/domain"
	featurehttp "github.com/ye-kart/reqflow/internal/features/http"
	"github.com/ye-kart/reqflow/internal/features/monitor"
	"github.com/ye-kart/reqflow/internal/features/runner"
)

func newTestAppWithScheduler(mock *mockHTTPClient, store *storage.Filesystem, monitorDir string) *app.App {
	r := runner.New(mock)
	sched := monitor.NewScheduler(r, monitorDir)
	return &app.App{
		HTTPExecutor: featurehttp.NewExecutor(mock),
		Runner:       r,
		Storage:      store,
		Scheduler:    sched,
	}
}

func TestMonitorCreate_CreatesMonitor(t *testing.T) {
	monitorDir := t.TempDir()
	wfDir := t.TempDir()

	// Create a workflow file for the monitor to reference
	wfPath := filepath.Join(wfDir, "test.yaml")
	wfContent := []byte("name: test-workflow\nsteps:\n  - name: health\n    method: GET\n    url: https://api.example.com/health\n")
	if err := os.WriteFile(wfPath, wfContent, 0644); err != nil {
		t.Fatal(err)
	}

	mock := &mockHTTPClient{doFunc: noopDoFunc}
	store := storage.NewFilesystem()
	a := newTestAppWithScheduler(mock, store, monitorDir)
	root := commands.NewRootCommand(a)

	buf := new(bytes.Buffer)
	root.SetOut(buf)
	root.SetErr(buf)
	root.SetArgs([]string{
		"monitor", "create", "health-check",
		"--workflow", wfPath,
		"--cron", "*/5 * * * *",
	})

	if err := root.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "health-check") {
		t.Errorf("expected output to contain 'health-check', got: %s", output)
	}

	// Verify config file was persisted
	configPath := filepath.Join(monitorDir, "health-check.yaml")
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		t.Fatal("expected monitor config file to exist")
	}
}

func TestMonitorCreate_WithEnv(t *testing.T) {
	monitorDir := t.TempDir()
	wfDir := t.TempDir()

	wfPath := filepath.Join(wfDir, "test.yaml")
	wfContent := []byte("name: test-workflow\nsteps:\n  - name: health\n    method: GET\n    url: https://api.example.com/health\n")
	if err := os.WriteFile(wfPath, wfContent, 0644); err != nil {
		t.Fatal(err)
	}

	mock := &mockHTTPClient{doFunc: noopDoFunc}
	store := storage.NewFilesystem()
	a := newTestAppWithScheduler(mock, store, monitorDir)
	root := commands.NewRootCommand(a)

	buf := new(bytes.Buffer)
	root.SetOut(buf)
	root.SetErr(buf)
	root.SetArgs([]string{
		"monitor", "create", "health-check-prod",
		"--workflow", wfPath,
		"--cron", "*/10 * * * *",
		"-e", "prod",
	})

	if err := root.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Load the persisted monitor and verify env
	sched := monitor.NewScheduler(runner.New(mock), monitorDir)
	if err := sched.Load(); err != nil {
		t.Fatal(err)
	}

	monitors := sched.List()
	if len(monitors) != 1 {
		t.Fatalf("expected 1 monitor, got %d", len(monitors))
	}
	if monitors[0].EnvName != "prod" {
		t.Errorf("expected env 'prod', got %q", monitors[0].EnvName)
	}
}

func TestMonitorList_ShowsMonitors(t *testing.T) {
	monitorDir := t.TempDir()

	mock := &mockHTTPClient{doFunc: noopDoFunc}
	store := storage.NewFilesystem()
	a := newTestAppWithScheduler(mock, store, monitorDir)

	// Add some monitors via the scheduler directly
	_ = a.Scheduler.Add(monitor.Monitor{
		Name:         "api-health",
		WorkflowPath: "/tmp/wf.yaml",
		Cron:         "*/5 * * * *",
	})
	_ = a.Scheduler.Add(monitor.Monitor{
		Name:         "smoke-test",
		WorkflowPath: "/tmp/smoke.yaml",
		Cron:         "0 * * * *",
	})

	root := commands.NewRootCommand(a)

	buf := new(bytes.Buffer)
	root.SetOut(buf)
	root.SetErr(buf)
	root.SetArgs([]string{"monitor", "list"})

	if err := root.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "api-health") {
		t.Errorf("expected output to contain 'api-health', got: %s", output)
	}
	if !strings.Contains(output, "smoke-test") {
		t.Errorf("expected output to contain 'smoke-test', got: %s", output)
	}
	if !strings.Contains(output, "*/5 * * * *") {
		t.Errorf("expected output to contain cron expression, got: %s", output)
	}
}

func TestMonitorList_Empty(t *testing.T) {
	monitorDir := t.TempDir()

	mock := &mockHTTPClient{doFunc: noopDoFunc}
	store := storage.NewFilesystem()
	a := newTestAppWithScheduler(mock, store, monitorDir)

	root := commands.NewRootCommand(a)

	buf := new(bytes.Buffer)
	root.SetOut(buf)
	root.SetErr(buf)
	root.SetArgs([]string{"monitor", "list"})

	if err := root.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "No monitors") {
		t.Errorf("expected 'No monitors' message, got: %s", output)
	}
}

func TestMonitorDelete_RemovesMonitor(t *testing.T) {
	monitorDir := t.TempDir()

	mock := &mockHTTPClient{doFunc: noopDoFunc}
	store := storage.NewFilesystem()
	a := newTestAppWithScheduler(mock, store, monitorDir)

	_ = a.Scheduler.Add(monitor.Monitor{
		Name:         "to-delete",
		WorkflowPath: "/tmp/wf.yaml",
		Cron:         "*/5 * * * *",
	})

	root := commands.NewRootCommand(a)

	buf := new(bytes.Buffer)
	root.SetOut(buf)
	root.SetErr(buf)
	root.SetArgs([]string{"monitor", "delete", "to-delete"})

	if err := root.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "Deleted") {
		t.Errorf("expected 'Deleted' message, got: %s", output)
	}

	monitors := a.Scheduler.List()
	if len(monitors) != 0 {
		t.Errorf("expected 0 monitors after delete, got %d", len(monitors))
	}
}

func TestMonitorRun_ExecutesWorkflow(t *testing.T) {
	monitorDir := t.TempDir()
	wfDir := t.TempDir()

	wfPath := filepath.Join(wfDir, "test.yaml")
	wfContent := []byte("name: test-workflow\nsteps:\n  - name: health\n    method: GET\n    url: https://api.example.com/health\n    assert:\n      - field: status\n        operator: \"==\"\n        expected: 200\n")
	if err := os.WriteFile(wfPath, wfContent, 0644); err != nil {
		t.Fatal(err)
	}

	mock := &mockHTTPClient{
		doFunc: func(_ context.Context, _ domain.HTTPRequest) (domain.HTTPResponse, error) {
			return domain.HTTPResponse{
				StatusCode: 200,
				Body:       []byte(`{"status":"ok"}`),
			}, nil
		},
	}
	store := storage.NewFilesystem()
	a := newTestAppWithScheduler(mock, store, monitorDir)

	_ = a.Scheduler.Add(monitor.Monitor{
		Name:         "run-once",
		WorkflowPath: wfPath,
		Cron:         "*/5 * * * *",
	})

	root := commands.NewRootCommand(a)

	buf := new(bytes.Buffer)
	root.SetOut(buf)
	root.SetErr(buf)
	root.SetArgs([]string{"monitor", "run", "run-once"})

	if err := root.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "test-workflow") {
		t.Errorf("expected output to contain workflow name, got: %s", output)
	}
}
