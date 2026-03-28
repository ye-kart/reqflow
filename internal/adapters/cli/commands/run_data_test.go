package commands_test

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ye-kart/reqflow/internal/adapters/cli/commands"
	"github.com/ye-kart/reqflow/internal/domain"
)

func TestRunCommand_DataFlagAccepted(t *testing.T) {
	a := newTestApp(&mockHTTPClient{
		doFunc: func(_ context.Context, _ domain.HTTPRequest) (domain.HTTPResponse, error) {
			return domain.HTTPResponse{StatusCode: 200, Body: []byte(`{}`)}, nil
		},
	})
	root := commands.NewRootCommand(a)

	tmpDir := t.TempDir()
	wfPath := filepath.Join(tmpDir, "workflow.yaml")
	os.WriteFile(wfPath, []byte(`
name: test
steps:
  - name: get
    method: GET
    url: https://api.example.com/users/{{name}}
`), 0644)

	csvPath := filepath.Join(tmpDir, "data.csv")
	os.WriteFile(csvPath, []byte("name\nJohn\n"), 0644)

	var out bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&out)
	root.SetArgs([]string{"run", wfPath, "--data", csvPath})

	err := root.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRunCommand_DataCSVIteratesCorrectly(t *testing.T) {
	var capturedURLs []string
	a := newTestApp(&mockHTTPClient{
		doFunc: func(_ context.Context, req domain.HTTPRequest) (domain.HTTPResponse, error) {
			capturedURLs = append(capturedURLs, req.URL)
			return domain.HTTPResponse{StatusCode: 200, Body: []byte(`{}`)}, nil
		},
	})
	root := commands.NewRootCommand(a)

	tmpDir := t.TempDir()
	wfPath := filepath.Join(tmpDir, "workflow.yaml")
	os.WriteFile(wfPath, []byte(`
name: iterate-test
steps:
  - name: fetch
    method: GET
    url: https://api.example.com/users/{{name}}
`), 0644)

	csvPath := filepath.Join(tmpDir, "data.csv")
	os.WriteFile(csvPath, []byte("name,role\nJohn,admin\nJane,user\n"), 0644)

	var out bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&out)
	root.SetArgs([]string{"run", wfPath, "--data", csvPath})

	err := root.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(capturedURLs) != 2 {
		t.Fatalf("captured URLs = %d, want 2", len(capturedURLs))
	}
	if capturedURLs[0] != "https://api.example.com/users/John" {
		t.Errorf("URL[0] = %q, want John URL", capturedURLs[0])
	}
	if capturedURLs[1] != "https://api.example.com/users/Jane" {
		t.Errorf("URL[1] = %q, want Jane URL", capturedURLs[1])
	}
}

func TestRunCommand_DataShowsPerIterationSummary(t *testing.T) {
	callCount := 0
	a := newTestApp(&mockHTTPClient{
		doFunc: func(_ context.Context, _ domain.HTTPRequest) (domain.HTTPResponse, error) {
			callCount++
			return domain.HTTPResponse{StatusCode: 200, Body: []byte(`{}`)}, nil
		},
	})
	root := commands.NewRootCommand(a)

	tmpDir := t.TempDir()
	wfPath := filepath.Join(tmpDir, "workflow.yaml")
	os.WriteFile(wfPath, []byte(`
name: summary-test
steps:
  - name: check
    method: GET
    url: https://api.example.com/users/{{name}}
    assert:
      - field: status
        operator: "=="
        expected: 200
`), 0644)

	csvPath := filepath.Join(tmpDir, "data.csv")
	os.WriteFile(csvPath, []byte("name\nJohn\nJane\n"), 0644)

	var out bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&out)
	root.SetArgs([]string{"run", wfPath, "--data", csvPath})

	err := root.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := out.String()
	// Should mention iterations
	if !strings.Contains(output, "Iteration") {
		t.Errorf("output should contain iteration info, got:\n%s", output)
	}
	// Should show summary
	if !strings.Contains(output, "2 iterations") {
		t.Errorf("output should contain '2 iterations', got:\n%s", output)
	}
}

func TestRunCommand_DataJSONFormat(t *testing.T) {
	a := newTestApp(&mockHTTPClient{
		doFunc: func(_ context.Context, _ domain.HTTPRequest) (domain.HTTPResponse, error) {
			return domain.HTTPResponse{StatusCode: 200, Body: []byte(`{}`)}, nil
		},
	})
	root := commands.NewRootCommand(a)

	tmpDir := t.TempDir()
	wfPath := filepath.Join(tmpDir, "workflow.yaml")
	os.WriteFile(wfPath, []byte(`
name: json-output-test
steps:
  - name: check
    method: GET
    url: https://api.example.com/users/{{name}}
    assert:
      - field: status
        operator: "=="
        expected: 200
`), 0644)

	csvPath := filepath.Join(tmpDir, "data.csv")
	os.WriteFile(csvPath, []byte("name\nJohn\n"), 0644)

	var out bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&out)
	root.SetArgs([]string{"run", wfPath, "--data", csvPath, "--output", "json"})

	err := root.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := out.String()
	if !strings.Contains(output, "iterations") {
		t.Errorf("JSON output should contain 'iterations' key, got:\n%s", output)
	}
	if !strings.Contains(output, "total_iterations") {
		t.Errorf("JSON output should contain 'total_iterations' key, got:\n%s", output)
	}
}

func TestRunCommand_DataFileNotFound(t *testing.T) {
	a := newTestApp(&mockHTTPClient{
		doFunc: func(_ context.Context, _ domain.HTTPRequest) (domain.HTTPResponse, error) {
			return domain.HTTPResponse{StatusCode: 200, Body: []byte(`{}`)}, nil
		},
	})
	root := commands.NewRootCommand(a)

	tmpDir := t.TempDir()
	wfPath := filepath.Join(tmpDir, "workflow.yaml")
	os.WriteFile(wfPath, []byte(`
name: test
steps:
  - name: get
    method: GET
    url: https://api.example.com
`), 0644)

	var out bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&out)
	root.SetArgs([]string{"run", wfPath, "--data", "/nonexistent/data.csv"})

	err := root.Execute()
	if err == nil {
		t.Fatal("expected error for nonexistent data file, got nil")
	}
}
