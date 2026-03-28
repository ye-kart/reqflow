package commands_test

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/ye-kart/reqflow/internal/adapters/cli/commands"
	"github.com/ye-kart/reqflow/internal/domain"
)

func TestCodegenCommand_Registered(t *testing.T) {
	mock := &mockHTTPClient{
		doFunc: func(_ context.Context, req domain.HTTPRequest) (domain.HTTPResponse, error) {
			return domain.HTTPResponse{}, nil
		},
	}

	a := newTestApp(mock)
	root := commands.NewRootCommand(a)

	found := false
	for _, cmd := range root.Commands() {
		if cmd.Name() == "codegen" {
			found = true
		}
	}
	if !found {
		t.Error("root command missing 'codegen' subcommand")
	}
}

func TestCodegenCommand_PythonGET(t *testing.T) {
	mock := &mockHTTPClient{
		doFunc: func(_ context.Context, req domain.HTTPRequest) (domain.HTTPResponse, error) {
			t.Fatal("request should not be executed with codegen command")
			return domain.HTTPResponse{}, nil
		},
	}

	a := newTestApp(mock)
	root := commands.NewRootCommand(a)

	buf := new(bytes.Buffer)
	root.SetOut(buf)
	root.SetErr(buf)
	root.SetArgs([]string{"codegen", "--lang", "python", "--method", "GET", "--url", "https://api.example.com/users"})

	err := root.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "import requests") {
		t.Errorf("expected Python requests import, got:\n%s", output)
	}
	if !strings.Contains(output, "https://api.example.com/users") {
		t.Errorf("expected URL in output, got:\n%s", output)
	}
}

func TestCodegenCommand_JavaScriptPOST(t *testing.T) {
	mock := &mockHTTPClient{
		doFunc: func(_ context.Context, req domain.HTTPRequest) (domain.HTTPResponse, error) {
			t.Fatal("request should not be executed")
			return domain.HTTPResponse{}, nil
		},
	}

	a := newTestApp(mock)
	root := commands.NewRootCommand(a)

	buf := new(bytes.Buffer)
	root.SetOut(buf)
	root.SetErr(buf)
	root.SetArgs([]string{
		"codegen", "--lang", "javascript",
		"--method", "POST",
		"--url", "https://api.example.com/users",
		"-d", `{"name":"John"}`,
	})

	err := root.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "fetch(") {
		t.Errorf("expected fetch call, got:\n%s", output)
	}
	if !strings.Contains(output, `"POST"`) {
		t.Errorf("expected POST method, got:\n%s", output)
	}
}

func TestCodegenCommand_UnknownLanguageReturnsError(t *testing.T) {
	mock := &mockHTTPClient{
		doFunc: func(_ context.Context, req domain.HTTPRequest) (domain.HTTPResponse, error) {
			return domain.HTTPResponse{}, nil
		},
	}

	a := newTestApp(mock)
	root := commands.NewRootCommand(a)

	buf := new(bytes.Buffer)
	root.SetOut(buf)
	root.SetErr(buf)
	root.SetArgs([]string{
		"codegen", "--lang", "brainfuck",
		"--method", "GET",
		"--url", "https://example.com",
	})

	err := root.Execute()
	if err == nil {
		t.Fatal("expected error for unknown language, got nil")
	}
	if !strings.Contains(err.Error(), "unsupported language") {
		t.Errorf("expected 'unsupported language' in error, got: %v", err)
	}
}

func TestCodegenCommand_MissingLangReturnsError(t *testing.T) {
	mock := &mockHTTPClient{
		doFunc: func(_ context.Context, req domain.HTTPRequest) (domain.HTTPResponse, error) {
			return domain.HTTPResponse{}, nil
		},
	}

	a := newTestApp(mock)
	root := commands.NewRootCommand(a)

	buf := new(bytes.Buffer)
	root.SetOut(buf)
	root.SetErr(buf)
	root.SetArgs([]string{"codegen", "--method", "GET", "--url", "https://example.com"})

	err := root.Execute()
	if err == nil {
		t.Fatal("expected error for missing --lang, got nil")
	}
}

func TestCodegenCommand_WithHeaders(t *testing.T) {
	mock := &mockHTTPClient{
		doFunc: func(_ context.Context, req domain.HTTPRequest) (domain.HTTPResponse, error) {
			t.Fatal("request should not be executed")
			return domain.HTTPResponse{}, nil
		},
	}

	a := newTestApp(mock)
	root := commands.NewRootCommand(a)

	buf := new(bytes.Buffer)
	root.SetOut(buf)
	root.SetErr(buf)
	root.SetArgs([]string{
		"codegen", "--lang", "curl",
		"--method", "GET",
		"--url", "https://api.example.com/users",
		"-H", "Authorization: Bearer token123",
	})

	err := root.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "Authorization") {
		t.Errorf("expected Authorization header in output, got:\n%s", output)
	}
	if !strings.Contains(output, "Bearer token123") {
		t.Errorf("expected Bearer token in output, got:\n%s", output)
	}
}

func TestCodegenFlag_OnGetCommand(t *testing.T) {
	mock := &mockHTTPClient{
		doFunc: func(_ context.Context, req domain.HTTPRequest) (domain.HTTPResponse, error) {
			t.Fatal("request should not be executed with --codegen flag")
			return domain.HTTPResponse{}, nil
		},
	}

	a := newTestApp(mock)
	root := commands.NewRootCommand(a)

	buf := new(bytes.Buffer)
	root.SetOut(buf)
	root.SetErr(buf)
	root.SetArgs([]string{"get", "https://example.com/api", "--codegen", "python"})

	err := root.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "import requests") {
		t.Errorf("expected Python code output, got:\n%s", output)
	}
	if !strings.Contains(output, "https://example.com/api") {
		t.Errorf("expected URL in output, got:\n%s", output)
	}
}

func TestCodegenFlag_OnPostCommand(t *testing.T) {
	mock := &mockHTTPClient{
		doFunc: func(_ context.Context, req domain.HTTPRequest) (domain.HTTPResponse, error) {
			t.Fatal("request should not be executed with --codegen flag")
			return domain.HTTPResponse{}, nil
		},
	}

	a := newTestApp(mock)
	root := commands.NewRootCommand(a)

	buf := new(bytes.Buffer)
	root.SetOut(buf)
	root.SetErr(buf)
	root.SetArgs([]string{
		"post", "https://example.com/api",
		"-d", `{"key":"val"}`,
		"--codegen", "go",
	})

	err := root.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "http.NewRequest") {
		t.Errorf("expected Go http code, got:\n%s", output)
	}
	if !strings.Contains(output, `"POST"`) {
		t.Errorf("expected POST method, got:\n%s", output)
	}
}
