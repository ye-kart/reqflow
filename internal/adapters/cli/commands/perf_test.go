package commands_test

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/ye-kart/reqflow/internal/adapters/cli/commands"
	"github.com/ye-kart/reqflow/internal/app"
	"github.com/ye-kart/reqflow/internal/domain"
)

type stubPerfHTTPClient struct{}

func (s *stubPerfHTTPClient) Do(_ context.Context, _ domain.HTTPRequest) (domain.HTTPResponse, error) {
	return domain.HTTPResponse{StatusCode: 200, Status: "200 OK"}, nil
}

func newPerfTestApp() *app.App {
	return app.New(&stubPerfHTTPClient{}, nil)
}

func TestPerfCommand_requires_url(t *testing.T) {
	a := newPerfTestApp()
	root := commands.NewRootCommand(a)
	root.SetArgs([]string{"perf"})

	err := root.Execute()
	if err == nil {
		t.Fatal("expected error for missing URL")
	}
}

func TestPerfCommand_runs_with_defaults(t *testing.T) {
	a := newPerfTestApp()
	root := commands.NewRootCommand(a)

	var buf bytes.Buffer
	root.SetOut(&buf)
	root.SetErr(&buf)
	root.SetArgs([]string{"perf", "http://localhost/test", "--duration", "100ms", "--vus", "1"})

	err := root.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "Total Requests") {
		t.Fatalf("output missing 'Total Requests', got: %s", out)
	}
	if !strings.Contains(out, "Requests/sec") {
		t.Fatalf("output missing 'Requests/sec', got: %s", out)
	}
}

func TestPerfCommand_accepts_method_flag(t *testing.T) {
	a := newPerfTestApp()
	root := commands.NewRootCommand(a)

	var buf bytes.Buffer
	root.SetOut(&buf)
	root.SetErr(&buf)
	root.SetArgs([]string{"perf", "http://localhost/test", "--method", "POST", "--duration", "100ms", "--vus", "1"})

	err := root.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestPerfCommand_accepts_header_flag(t *testing.T) {
	a := newPerfTestApp()
	root := commands.NewRootCommand(a)

	var buf bytes.Buffer
	root.SetOut(&buf)
	root.SetErr(&buf)
	root.SetArgs([]string{"perf", "http://localhost/test",
		"-H", "X-Test: value",
		"--duration", "100ms", "--vus", "1"})

	err := root.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestPerfCommand_accepts_body_flag(t *testing.T) {
	a := newPerfTestApp()
	root := commands.NewRootCommand(a)

	var buf bytes.Buffer
	root.SetOut(&buf)
	root.SetErr(&buf)
	root.SetArgs([]string{"perf", "http://localhost/test",
		"--method", "POST",
		"-d", `{"key":"value"}`,
		"--duration", "100ms", "--vus", "1"})

	err := root.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestPerfCommand_shows_percentiles_in_output(t *testing.T) {
	a := newPerfTestApp()
	root := commands.NewRootCommand(a)

	var buf bytes.Buffer
	root.SetOut(&buf)
	root.SetErr(&buf)
	root.SetArgs([]string{"perf", "http://localhost/test", "--duration", "100ms", "--vus", "1"})

	err := root.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	out := buf.String()
	for _, label := range []string{"P50", "P90", "P95", "P99"} {
		if !strings.Contains(out, label) {
			t.Fatalf("output missing %q, got: %s", label, out)
		}
	}
}
