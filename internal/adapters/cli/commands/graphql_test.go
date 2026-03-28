package commands_test

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ye-kart/reqflow/internal/adapters/cli/commands"
	"github.com/ye-kart/reqflow/internal/app"
	"github.com/ye-kart/reqflow/internal/domain"
)

func TestGraphQLCommand_SendsPostWithQueryBody(t *testing.T) {
	var capturedReq domain.HTTPRequest
	mock := &mockHTTPClient{
		doFunc: func(_ context.Context, req domain.HTTPRequest) (domain.HTTPResponse, error) {
			capturedReq = req
			return domain.HTTPResponse{
				StatusCode: 200,
				Status:     "200 OK",
				Body:       []byte(`{"data": {"users": [{"id": "1"}]}}`),
			}, nil
		},
	}

	a := app.New(mock, nil)
	root := commands.NewRootCommand(a)

	buf := new(bytes.Buffer)
	root.SetOut(buf)
	root.SetErr(buf)
	root.SetArgs([]string{"graphql", "https://api.example.com/graphql", "--query", "{ users { id } }"})

	err := root.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if capturedReq.Method != domain.MethodPost {
		t.Errorf("expected POST method, got %s", capturedReq.Method)
	}

	if capturedReq.URL != "https://api.example.com/graphql" {
		t.Errorf("expected URL https://api.example.com/graphql, got %s", capturedReq.URL)
	}

	if capturedReq.ContentType != "application/json" {
		t.Errorf("expected Content-Type application/json, got %s", capturedReq.ContentType)
	}

	bodyStr := string(capturedReq.Body)
	if !strings.Contains(bodyStr, `"query"`) {
		t.Errorf("expected body to contain query field, got: %s", bodyStr)
	}
	if !strings.Contains(bodyStr, "{ users { id } }") {
		t.Errorf("expected body to contain query string, got: %s", bodyStr)
	}
}

func TestGraphQLCommand_IncludesVariablesInBody(t *testing.T) {
	var capturedReq domain.HTTPRequest
	mock := &mockHTTPClient{
		doFunc: func(_ context.Context, req domain.HTTPRequest) (domain.HTTPResponse, error) {
			capturedReq = req
			return domain.HTTPResponse{
				StatusCode: 200,
				Status:     "200 OK",
				Body:       []byte(`{"data": {"user": {"name": "Alice"}}}`),
			}, nil
		},
	}

	a := app.New(mock, nil)
	root := commands.NewRootCommand(a)

	buf := new(bytes.Buffer)
	root.SetOut(buf)
	root.SetErr(buf)
	root.SetArgs([]string{
		"graphql", "https://api.example.com/graphql",
		"--query", `query GetUser($id: ID!) { user(id: $id) { name } }`,
		"--variables", `{"id": "123"}`,
	})

	err := root.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	bodyStr := string(capturedReq.Body)
	if !strings.Contains(bodyStr, `"variables"`) {
		t.Errorf("expected body to contain variables field, got: %s", bodyStr)
	}
	if !strings.Contains(bodyStr, `"id"`) {
		t.Errorf("expected body to contain variable id, got: %s", bodyStr)
	}
}

func TestGraphQLCommand_QueryFileFlag(t *testing.T) {
	// Create a temporary query file.
	dir := t.TempDir()
	queryFile := filepath.Join(dir, "query.graphql")
	err := os.WriteFile(queryFile, []byte("{ users { id name email } }"), 0644)
	if err != nil {
		t.Fatalf("failed to write query file: %v", err)
	}

	var capturedReq domain.HTTPRequest
	mock := &mockHTTPClient{
		doFunc: func(_ context.Context, req domain.HTTPRequest) (domain.HTTPResponse, error) {
			capturedReq = req
			return domain.HTTPResponse{
				StatusCode: 200,
				Status:     "200 OK",
				Body:       []byte(`{"data": {"users": []}}`),
			}, nil
		},
	}

	a := app.New(mock, nil)
	root := commands.NewRootCommand(a)

	buf := new(bytes.Buffer)
	root.SetOut(buf)
	root.SetErr(buf)
	root.SetArgs([]string{"graphql", "https://api.example.com/graphql", "--query-file", queryFile})

	err = root.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	bodyStr := string(capturedReq.Body)
	if !strings.Contains(bodyStr, "{ users { id name email } }") {
		t.Errorf("expected body to contain query from file, got: %s", bodyStr)
	}
}

func TestGraphQLCommand_OperationNameFlag(t *testing.T) {
	var capturedReq domain.HTTPRequest
	mock := &mockHTTPClient{
		doFunc: func(_ context.Context, req domain.HTTPRequest) (domain.HTTPResponse, error) {
			capturedReq = req
			return domain.HTTPResponse{
				StatusCode: 200,
				Status:     "200 OK",
				Body:       []byte(`{"data": {"users": []}}`),
			}, nil
		},
	}

	a := app.New(mock, nil)
	root := commands.NewRootCommand(a)

	buf := new(bytes.Buffer)
	root.SetOut(buf)
	root.SetErr(buf)
	root.SetArgs([]string{
		"graphql", "https://api.example.com/graphql",
		"--query", "query GetUsers { users { id } }",
		"--operation-name", "GetUsers",
	})

	err := root.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	bodyStr := string(capturedReq.Body)
	if !strings.Contains(bodyStr, `"operationName"`) {
		t.Errorf("expected body to contain operationName, got: %s", bodyStr)
	}
	if !strings.Contains(bodyStr, `"GetUsers"`) {
		t.Errorf("expected body to contain GetUsers, got: %s", bodyStr)
	}
}

func TestGraphQLCommand_DisplaysGraphQLErrors(t *testing.T) {
	mock := &mockHTTPClient{
		doFunc: func(_ context.Context, req domain.HTTPRequest) (domain.HTTPResponse, error) {
			return domain.HTTPResponse{
				StatusCode: 200,
				Status:     "200 OK",
				Body:       []byte(`{"errors": [{"message": "Cannot query field 'foo'"}]}`),
			}, nil
		},
	}

	a := app.New(mock, nil)
	root := commands.NewRootCommand(a)

	buf := new(bytes.Buffer)
	root.SetOut(buf)
	root.SetErr(buf)
	root.SetArgs([]string{"graphql", "https://api.example.com/graphql", "--query", "{ foo }", "--no-color"})

	err := root.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "Errors:") {
		t.Errorf("expected output to show 'Errors:', got: %s", output)
	}
	if !strings.Contains(output, "Cannot query field 'foo'") {
		t.Errorf("expected output to show error message, got: %s", output)
	}
}

func TestGraphQLCommand_DisplaysDataAsPrettyJSON(t *testing.T) {
	mock := &mockHTTPClient{
		doFunc: func(_ context.Context, req domain.HTTPRequest) (domain.HTTPResponse, error) {
			return domain.HTTPResponse{
				StatusCode: 200,
				Status:     "200 OK",
				Body:       []byte(`{"data": {"users": [{"id": "1", "name": "Alice"}]}}`),
			}, nil
		},
	}

	a := app.New(mock, nil)
	root := commands.NewRootCommand(a)

	buf := new(bytes.Buffer)
	root.SetOut(buf)
	root.SetErr(buf)
	root.SetArgs([]string{"graphql", "https://api.example.com/graphql", "--query", "{ users { id name } }", "--no-color"})

	err := root.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "Data:") {
		t.Errorf("expected output to contain 'Data:', got: %s", output)
	}
	// Pretty-printed JSON should have indentation
	if !strings.Contains(output, "  ") {
		t.Errorf("expected pretty-printed output with indentation, got: %s", output)
	}
}

func TestGraphQLCommand_RequiresQueryOrQueryFile(t *testing.T) {
	mock := &mockHTTPClient{
		doFunc: func(_ context.Context, req domain.HTTPRequest) (domain.HTTPResponse, error) {
			return domain.HTTPResponse{StatusCode: 200, Body: []byte(`{"data": {}}`)}, nil
		},
	}

	a := app.New(mock, nil)
	root := commands.NewRootCommand(a)

	buf := new(bytes.Buffer)
	root.SetOut(buf)
	root.SetErr(buf)
	root.SetArgs([]string{"graphql", "https://api.example.com/graphql"})

	err := root.Execute()
	if err == nil {
		t.Fatal("expected error when neither --query nor --query-file is provided")
	}
}

func TestGraphQLCommand_SupportsHeaderFlags(t *testing.T) {
	var capturedReq domain.HTTPRequest
	mock := &mockHTTPClient{
		doFunc: func(_ context.Context, req domain.HTTPRequest) (domain.HTTPResponse, error) {
			capturedReq = req
			return domain.HTTPResponse{
				StatusCode: 200,
				Status:     "200 OK",
				Body:       []byte(`{"data": {}}`),
			}, nil
		},
	}

	a := app.New(mock, nil)
	root := commands.NewRootCommand(a)

	buf := new(bytes.Buffer)
	root.SetOut(buf)
	root.SetErr(buf)
	root.SetArgs([]string{
		"graphql", "https://api.example.com/graphql",
		"--query", "{ me { id } }",
		"-H", "Authorization: Bearer token123",
	})

	err := root.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	found := false
	for _, h := range capturedReq.Headers {
		if h.Key == "Authorization" && h.Value == "Bearer token123" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected Authorization header, got headers: %v", capturedReq.Headers)
	}
}
