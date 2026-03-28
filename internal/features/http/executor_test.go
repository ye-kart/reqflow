package http_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/ye-kart/reqflow/internal/domain"
	featurehttp "github.com/ye-kart/reqflow/internal/features/http"
	"github.com/ye-kart/reqflow/internal/ports/driven"
)

// mockHTTPClient implements driven.HTTPClient for testing.
type mockHTTPClient struct {
	doFunc func(ctx context.Context, req domain.HTTPRequest) (domain.HTTPResponse, error)
}

func (m *mockHTTPClient) Do(ctx context.Context, req domain.HTTPRequest) (domain.HTTPResponse, error) {
	return m.doFunc(ctx, req)
}

func TestExecute_SendsCorrectRequestToHTTPClient(t *testing.T) {
	var capturedReq domain.HTTPRequest
	mock := &mockHTTPClient{
		doFunc: func(_ context.Context, req domain.HTTPRequest) (domain.HTTPResponse, error) {
			capturedReq = req
			return domain.HTTPResponse{StatusCode: 200, Status: "200 OK"}, nil
		},
	}

	executor := featurehttp.NewExecutor(mock)
	config := domain.RequestConfig{
		Method: domain.MethodGet,
		URL:    "https://example.com/api",
		Headers: []domain.Header{
			{Key: "Accept", Value: "application/json"},
		},
	}

	_, err := executor.Execute(context.Background(), config, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if capturedReq.Method != domain.MethodGet {
		t.Errorf("expected method GET, got %s", capturedReq.Method)
	}
	if capturedReq.URL != "https://example.com/api" {
		t.Errorf("expected URL https://example.com/api, got %s", capturedReq.URL)
	}
	if len(capturedReq.Headers) != 1 || capturedReq.Headers[0].Key != "Accept" {
		t.Errorf("expected Accept header, got %v", capturedReq.Headers)
	}
}

func TestExecute_AppliesVariableSubstitution(t *testing.T) {
	var capturedReq domain.HTTPRequest
	mock := &mockHTTPClient{
		doFunc: func(_ context.Context, req domain.HTTPRequest) (domain.HTTPResponse, error) {
			capturedReq = req
			return domain.HTTPResponse{StatusCode: 200}, nil
		},
	}

	executor := featurehttp.NewExecutor(mock)
	config := domain.RequestConfig{
		Method: domain.MethodGet,
		URL:    "https://{{host}}/api/{{version}}",
	}
	vars := map[string]string{
		"host":    "example.com",
		"version": "v2",
	}

	_, err := executor.Execute(context.Background(), config, vars)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := "https://example.com/api/v2"
	if capturedReq.URL != expected {
		t.Errorf("expected URL %s, got %s", expected, capturedReq.URL)
	}
}

func TestExecute_AppliesAuth(t *testing.T) {
	var capturedReq domain.HTTPRequest
	mock := &mockHTTPClient{
		doFunc: func(_ context.Context, req domain.HTTPRequest) (domain.HTTPResponse, error) {
			capturedReq = req
			return domain.HTTPResponse{StatusCode: 200}, nil
		},
	}

	executor := featurehttp.NewExecutor(mock)
	config := domain.RequestConfig{
		Method: domain.MethodGet,
		URL:    "https://example.com/api",
		Auth: &domain.AuthConfig{
			Type:   domain.AuthBearer,
			Bearer: &domain.BearerAuthConfig{Token: "my-token"},
		},
	}

	_, err := executor.Execute(context.Background(), config, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	found := false
	for _, h := range capturedReq.Headers {
		if h.Key == "Authorization" && h.Value == "Bearer my-token" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected Authorization header with Bearer token, got headers: %v", capturedReq.Headers)
	}
}

func TestExecute_ReturnsResponseFromHTTPClient(t *testing.T) {
	expectedResp := domain.HTTPResponse{
		StatusCode: 201,
		Status:     "201 Created",
		Body:       []byte(`{"id": 1}`),
		Headers:    []domain.Header{{Key: "Content-Type", Value: "application/json"}},
		Duration:   100 * time.Millisecond,
		Size:       9,
	}

	mock := &mockHTTPClient{
		doFunc: func(_ context.Context, _ domain.HTTPRequest) (domain.HTTPResponse, error) {
			return expectedResp, nil
		},
	}

	executor := featurehttp.NewExecutor(mock)
	config := domain.RequestConfig{
		Method: domain.MethodPost,
		URL:    "https://example.com/api",
		Body:   []byte(`{"name": "test"}`),
	}

	result, err := executor.Execute(context.Background(), config, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Response.StatusCode != 201 {
		t.Errorf("expected status 201, got %d", result.Response.StatusCode)
	}
	if string(result.Response.Body) != `{"id": 1}` {
		t.Errorf("expected body {\"id\": 1}, got %s", string(result.Response.Body))
	}
}

func TestExecute_ReturnsErrorWhenHTTPClientFails(t *testing.T) {
	clientErr := errors.New("connection refused")
	mock := &mockHTTPClient{
		doFunc: func(_ context.Context, _ domain.HTTPRequest) (domain.HTTPResponse, error) {
			return domain.HTTPResponse{}, clientErr
		},
	}

	executor := featurehttp.NewExecutor(mock)
	config := domain.RequestConfig{
		Method: domain.MethodGet,
		URL:    "https://example.com/api",
	}

	_, err := executor.Execute(context.Background(), config, nil)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, clientErr) {
		t.Errorf("expected error to wrap %v, got %v", clientErr, err)
	}
}

func TestBuildRequest_ReturnsRequestWithoutSending(t *testing.T) {
	callCount := 0
	mock := &mockHTTPClient{
		doFunc: func(_ context.Context, _ domain.HTTPRequest) (domain.HTTPResponse, error) {
			callCount++
			return domain.HTTPResponse{}, nil
		},
	}

	executor := featurehttp.NewExecutor(mock)
	config := domain.RequestConfig{
		Method: domain.MethodGet,
		URL:    "https://example.com/api",
		Headers: []domain.Header{
			{Key: "Accept", Value: "application/json"},
		},
		Auth: &domain.AuthConfig{
			Type:   domain.AuthBearer,
			Bearer: &domain.BearerAuthConfig{Token: "my-token"},
		},
	}

	req, err := executor.BuildRequest(config, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if callCount != 0 {
		t.Errorf("expected no HTTP calls, got %d", callCount)
	}
	if req.Method != domain.MethodGet {
		t.Errorf("expected method GET, got %s", req.Method)
	}
	if req.URL != "https://example.com/api" {
		t.Errorf("expected URL https://example.com/api, got %s", req.URL)
	}

	// Auth should be applied
	found := false
	for _, h := range req.Headers {
		if h.Key == "Authorization" && h.Value == "Bearer my-token" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected Authorization header, got headers: %v", req.Headers)
	}
}

// mockScriptEngine implements driven.ScriptEngine for testing.
type mockScriptEngine struct {
	executeFunc func(script string, ctx driven.ScriptContext) (driven.ScriptResult, error)
}

func (m *mockScriptEngine) Execute(script string, ctx driven.ScriptContext) (driven.ScriptResult, error) {
	return m.executeFunc(script, ctx)
}

func TestExecute_PreRequestScriptModifiesRequest(t *testing.T) {
	var capturedReq domain.HTTPRequest
	httpMock := &mockHTTPClient{
		doFunc: func(_ context.Context, req domain.HTTPRequest) (domain.HTTPResponse, error) {
			capturedReq = req
			return domain.HTTPResponse{StatusCode: 200}, nil
		},
	}

	scriptMock := &mockScriptEngine{
		executeFunc: func(script string, ctx driven.ScriptContext) (driven.ScriptResult, error) {
			// Simulate pre-request script modifying the URL.
			updated := ctx.Request
			updated.URL = "https://modified.example.com/api"
			return driven.ScriptResult{
				UpdatedVariables: ctx.Variables,
				UpdatedRequest:   &updated,
			}, nil
		},
	}

	executor := featurehttp.NewExecutor(httpMock, featurehttp.WithScriptEngine(scriptMock))
	config := domain.RequestConfig{
		Method:    domain.MethodGet,
		URL:       "https://example.com/api",
		PreScript: `pm.request.url = "https://modified.example.com/api";`,
	}

	_, err := executor.Execute(context.Background(), config, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if capturedReq.URL != "https://modified.example.com/api" {
		t.Errorf("expected modified URL, got %s", capturedReq.URL)
	}
}

func TestExecute_PostResponseScriptExtractsData(t *testing.T) {
	httpMock := &mockHTTPClient{
		doFunc: func(_ context.Context, _ domain.HTTPRequest) (domain.HTTPResponse, error) {
			return domain.HTTPResponse{
				StatusCode: 200,
				Body:       []byte(`{"token":"abc123"}`),
			}, nil
		},
	}

	scriptMock := &mockScriptEngine{
		executeFunc: func(script string, ctx driven.ScriptContext) (driven.ScriptResult, error) {
			vars := make(map[string]string)
			for k, v := range ctx.Variables {
				vars[k] = v
			}
			vars["token"] = "abc123"
			return driven.ScriptResult{
				UpdatedVariables: vars,
				Console:          []string{"extracted token"},
			}, nil
		},
	}

	executor := featurehttp.NewExecutor(httpMock, featurehttp.WithScriptEngine(scriptMock))
	config := domain.RequestConfig{
		Method:     domain.MethodGet,
		URL:        "https://example.com/api",
		PostScript: `pm.variables.set("token", pm.response.json().token);`,
	}

	result, err := executor.Execute(context.Background(), config, map[string]string{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(result.ScriptConsole) != 1 || result.ScriptConsole[0] != "extracted token" {
		t.Errorf("expected console [extracted token], got %v", result.ScriptConsole)
	}
}

func TestExecute_TestResultsCaptured(t *testing.T) {
	httpMock := &mockHTTPClient{
		doFunc: func(_ context.Context, _ domain.HTTPRequest) (domain.HTTPResponse, error) {
			return domain.HTTPResponse{StatusCode: 200}, nil
		},
	}

	scriptMock := &mockScriptEngine{
		executeFunc: func(script string, ctx driven.ScriptContext) (driven.ScriptResult, error) {
			return driven.ScriptResult{
				UpdatedVariables: ctx.Variables,
				TestResults: []domain.TestResult{
					{Name: "status is 200", Passed: true},
					{Name: "body has data", Passed: false, Error: "missing data"},
				},
			}, nil
		},
	}

	executor := featurehttp.NewExecutor(httpMock, featurehttp.WithScriptEngine(scriptMock))
	config := domain.RequestConfig{
		Method:     domain.MethodGet,
		URL:        "https://example.com/api",
		PostScript: `pm.test("status is 200", function() { pm.expect(pm.response.code).to.eql(200); });`,
	}

	result, err := executor.Execute(context.Background(), config, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(result.TestResults) != 2 {
		t.Fatalf("expected 2 test results, got %d", len(result.TestResults))
	}
	if !result.TestResults[0].Passed {
		t.Error("expected first test to pass")
	}
	if result.TestResults[1].Passed {
		t.Error("expected second test to fail")
	}
}

func TestExecute_NoScriptEngineSkipsScripts(t *testing.T) {
	httpMock := &mockHTTPClient{
		doFunc: func(_ context.Context, _ domain.HTTPRequest) (domain.HTTPResponse, error) {
			return domain.HTTPResponse{StatusCode: 200}, nil
		},
	}

	// No script engine provided.
	executor := featurehttp.NewExecutor(httpMock)
	config := domain.RequestConfig{
		Method:     domain.MethodGet,
		URL:        "https://example.com/api",
		PreScript:  `pm.request.url = "modified";`,
		PostScript: `pm.test("test", function() {});`,
	}

	result, err := executor.Execute(context.Background(), config, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Request.URL != "https://example.com/api" {
		t.Errorf("expected original URL without script engine, got %s", result.Request.URL)
	}
}

func TestExecute_ReturnsErrorWhenAuthConfigInvalid(t *testing.T) {
	mock := &mockHTTPClient{
		doFunc: func(_ context.Context, _ domain.HTTPRequest) (domain.HTTPResponse, error) {
			return domain.HTTPResponse{}, nil
		},
	}

	executor := featurehttp.NewExecutor(mock)
	config := domain.RequestConfig{
		Method: domain.MethodGet,
		URL:    "https://example.com/api",
		Auth: &domain.AuthConfig{
			Type:  domain.AuthBasic,
			Basic: nil, // nil sub-config should cause error
		},
	}

	_, err := executor.Execute(context.Background(), config, nil)
	if err == nil {
		t.Fatal("expected error for nil basic auth config, got nil")
	}
	if !errors.Is(err, domain.ErrInvalidAuth) {
		t.Errorf("expected ErrInvalidAuth, got %v", err)
	}
}
