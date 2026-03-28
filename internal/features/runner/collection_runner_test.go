package runner_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/ye-kart/reqflow/internal/domain"
	"github.com/ye-kart/reqflow/internal/features/runner"
)

func TestCollectionRunner_SingleRequest(t *testing.T) {
	client := &mockHTTPClient{
		responses: []domain.HTTPResponse{
			{StatusCode: 200, Body: []byte(`{"ok":true}`)},
		},
	}

	cr := runner.NewCollectionRunner(client)

	col := domain.Collection{
		Name: "test-api",
		Requests: []domain.SavedRequest{
			{
				Name: "Health Check",
				Config: domain.RequestConfig{
					Method: domain.MethodGet,
					URL:    "https://api.example.com/health",
				},
			},
		},
	}

	opts := domain.CollectionRunOptions{
		Sequential:    true,
		StopOnFailure: true,
	}

	result, err := cr.RunCollection(context.Background(), col, opts)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.CollectionName != "test-api" {
		t.Errorf("collection name = %q, want %q", result.CollectionName, "test-api")
	}
	if result.TotalRequests != 1 {
		t.Errorf("total requests = %d, want 1", result.TotalRequests)
	}
	if result.Passed != 1 {
		t.Errorf("passed = %d, want 1", result.Passed)
	}
	if result.Failed != 0 {
		t.Errorf("failed = %d, want 0", result.Failed)
	}
	if len(result.Results) != 1 {
		t.Fatalf("results count = %d, want 1", len(result.Results))
	}
	if result.Results[0].RequestName != "Health Check" {
		t.Errorf("request name = %q, want %q", result.Results[0].RequestName, "Health Check")
	}
	if !result.Results[0].Passed {
		t.Error("expected request to pass")
	}
}

func TestCollectionRunner_MultiRequest_Sequential(t *testing.T) {
	callOrder := []string{}
	client := &mockHTTPClient{
		responses: []domain.HTTPResponse{
			{StatusCode: 200, Body: []byte(`{"users":[]}`)},
			{StatusCode: 201, Body: []byte(`{"id":1}`)},
			{StatusCode: 200, Body: []byte(`{"id":1,"name":"Alice"}`)},
		},
	}
	// We can verify sequential execution by checking call count after run
	_ = callOrder

	cr := runner.NewCollectionRunner(client)

	col := domain.Collection{
		Name: "user-api",
		Requests: []domain.SavedRequest{
			{
				Name:   "List Users",
				Config: domain.RequestConfig{Method: domain.MethodGet, URL: "https://api.example.com/users"},
			},
			{
				Name:   "Create User",
				Config: domain.RequestConfig{Method: domain.MethodPost, URL: "https://api.example.com/users"},
			},
			{
				Name:   "Get User",
				Config: domain.RequestConfig{Method: domain.MethodGet, URL: "https://api.example.com/users/1"},
			},
		},
	}

	opts := domain.CollectionRunOptions{Sequential: true, StopOnFailure: true}
	result, err := cr.RunCollection(context.Background(), col, opts)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.TotalRequests != 3 {
		t.Errorf("total requests = %d, want 3", result.TotalRequests)
	}
	if result.Passed != 3 {
		t.Errorf("passed = %d, want 3", result.Passed)
	}
	if len(result.Results) != 3 {
		t.Fatalf("results count = %d, want 3", len(result.Results))
	}
	if client.calls != 3 {
		t.Errorf("http calls = %d, want 3", client.calls)
	}
}

func TestCollectionRunner_FolderFilter(t *testing.T) {
	client := &mockHTTPClient{
		responses: []domain.HTTPResponse{
			{StatusCode: 200, Body: []byte(`{}`)},
			{StatusCode: 201, Body: []byte(`{}`)},
		},
	}

	cr := runner.NewCollectionRunner(client)

	col := domain.Collection{
		Name: "api",
		Requests: []domain.SavedRequest{
			{Name: "Root Request", Config: domain.RequestConfig{Method: domain.MethodGet, URL: "https://api.example.com/"}},
		},
		Folders: []domain.Folder{
			{
				Name: "Users",
				Requests: []domain.SavedRequest{
					{Name: "List Users", Config: domain.RequestConfig{Method: domain.MethodGet, URL: "https://api.example.com/users"}},
					{Name: "Create User", Config: domain.RequestConfig{Method: domain.MethodPost, URL: "https://api.example.com/users"}},
				},
			},
			{
				Name: "Products",
				Requests: []domain.SavedRequest{
					{Name: "List Products", Config: domain.RequestConfig{Method: domain.MethodGet, URL: "https://api.example.com/products"}},
				},
			},
		},
	}

	opts := domain.CollectionRunOptions{
		Sequential:    true,
		StopOnFailure: true,
		FolderName:    "Users",
	}

	result, err := cr.RunCollection(context.Background(), col, opts)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.TotalRequests != 2 {
		t.Errorf("total requests = %d, want 2", result.TotalRequests)
	}
	if client.calls != 2 {
		t.Errorf("http calls = %d, want 2 (only Users folder)", client.calls)
	}
	for _, r := range result.Results {
		if r.FolderPath != "Users" {
			t.Errorf("folder path = %q, want %q", r.FolderPath, "Users")
		}
	}
}

func TestCollectionRunner_CollectionAuth_Applied(t *testing.T) {
	var capturedReqs []domain.HTTPRequest
	client := &capturingHTTPClient{
		response: domain.HTTPResponse{StatusCode: 200, Body: []byte(`{}`)},
		captured: &capturedReqs,
	}

	cr := runner.NewCollectionRunner(client)

	col := domain.Collection{
		Name: "auth-api",
		Auth: &domain.AuthConfig{
			Type:   domain.AuthBearer,
			Bearer: &domain.BearerAuthConfig{Token: "test-token"},
		},
		Requests: []domain.SavedRequest{
			{Name: "Get Resource", Config: domain.RequestConfig{Method: domain.MethodGet, URL: "https://api.example.com/resource"}},
		},
	}

	opts := domain.CollectionRunOptions{Sequential: true, StopOnFailure: true}
	_, err := cr.RunCollection(context.Background(), col, opts)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(capturedReqs) != 1 {
		t.Fatalf("expected 1 captured request, got %d", len(capturedReqs))
	}

	found := false
	for _, h := range capturedReqs[0].Headers {
		if h.Key == "Authorization" && h.Value == "Bearer test-token" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected Authorization: Bearer test-token header, got headers: %v", capturedReqs[0].Headers)
	}
}

func TestCollectionRunner_CollectionHeaders_Applied(t *testing.T) {
	var capturedReqs []domain.HTTPRequest
	client := &capturingHTTPClient{
		response: domain.HTTPResponse{StatusCode: 200, Body: []byte(`{}`)},
		captured: &capturedReqs,
	}

	cr := runner.NewCollectionRunner(client)

	col := domain.Collection{
		Name: "header-api",
		Headers: []domain.Header{
			{Key: "X-API-Key", Value: "abc123"},
			{Key: "Accept", Value: "application/json"},
		},
		Requests: []domain.SavedRequest{
			{Name: "Get Resource", Config: domain.RequestConfig{Method: domain.MethodGet, URL: "https://api.example.com/resource"}},
		},
	}

	opts := domain.CollectionRunOptions{Sequential: true, StopOnFailure: true}
	_, err := cr.RunCollection(context.Background(), col, opts)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(capturedReqs) != 1 {
		t.Fatalf("expected 1 captured request, got %d", len(capturedReqs))
	}

	headerMap := make(map[string]string)
	for _, h := range capturedReqs[0].Headers {
		headerMap[h.Key] = h.Value
	}
	if headerMap["X-API-Key"] != "abc123" {
		t.Errorf("expected X-API-Key: abc123, got %q", headerMap["X-API-Key"])
	}
	if headerMap["Accept"] != "application/json" {
		t.Errorf("expected Accept: application/json, got %q", headerMap["Accept"])
	}
}

func TestCollectionRunner_CollectionVariables_Substituted(t *testing.T) {
	var capturedReqs []domain.HTTPRequest
	client := &capturingHTTPClient{
		response: domain.HTTPResponse{StatusCode: 200, Body: []byte(`{}`)},
		captured: &capturedReqs,
	}

	cr := runner.NewCollectionRunner(client)

	col := domain.Collection{
		Name: "var-api",
		Variables: []domain.Variable{
			{Key: "base_url", Value: "https://api.example.com"},
		},
		Requests: []domain.SavedRequest{
			{Name: "Get Users", Config: domain.RequestConfig{Method: domain.MethodGet, URL: "{{base_url}}/users"}},
		},
	}

	opts := domain.CollectionRunOptions{Sequential: true, StopOnFailure: true}
	_, err := cr.RunCollection(context.Background(), col, opts)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(capturedReqs) != 1 {
		t.Fatalf("expected 1 captured request, got %d", len(capturedReqs))
	}

	if capturedReqs[0].URL != "https://api.example.com/users" {
		t.Errorf("URL = %q, want %q", capturedReqs[0].URL, "https://api.example.com/users")
	}
}

func TestCollectionRunner_FolderOverridesCollection(t *testing.T) {
	var capturedReqs []domain.HTTPRequest
	client := &capturingHTTPClient{
		response: domain.HTTPResponse{StatusCode: 200, Body: []byte(`{}`)},
		captured: &capturedReqs,
	}

	cr := runner.NewCollectionRunner(client)

	col := domain.Collection{
		Name: "override-api",
		Auth: &domain.AuthConfig{
			Type:   domain.AuthBearer,
			Bearer: &domain.BearerAuthConfig{Token: "collection-token"},
		},
		Headers: []domain.Header{
			{Key: "X-Collection", Value: "yes"},
			{Key: "X-Override", Value: "collection"},
		},
		Variables: []domain.Variable{
			{Key: "base_url", Value: "https://collection.example.com"},
			{Key: "shared_var", Value: "from-collection"},
		},
		Folders: []domain.Folder{
			{
				Name: "Admin",
				Auth: &domain.AuthConfig{
					Type:   domain.AuthBearer,
					Bearer: &domain.BearerAuthConfig{Token: "folder-token"},
				},
				Headers: []domain.Header{
					{Key: "X-Override", Value: "folder"},
				},
				Variables: []domain.Variable{
					{Key: "shared_var", Value: "from-folder"},
				},
				Requests: []domain.SavedRequest{
					{Name: "Admin Action", Config: domain.RequestConfig{Method: domain.MethodGet, URL: "{{base_url}}/admin"}},
				},
			},
		},
	}

	opts := domain.CollectionRunOptions{
		Sequential:    true,
		StopOnFailure: true,
		FolderName:    "Admin",
	}
	_, err := cr.RunCollection(context.Background(), col, opts)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(capturedReqs) != 1 {
		t.Fatalf("expected 1 captured request, got %d", len(capturedReqs))
	}

	req := capturedReqs[0]

	// Folder auth should override collection auth
	authFound := false
	for _, h := range req.Headers {
		if h.Key == "Authorization" && h.Value == "Bearer folder-token" {
			authFound = true
		}
	}
	if !authFound {
		t.Errorf("expected folder auth (Bearer folder-token), got headers: %v", req.Headers)
	}

	// Folder header should override same-key collection header
	headerMap := make(map[string]string)
	for _, h := range req.Headers {
		if h.Key != "Authorization" {
			headerMap[h.Key] = h.Value
		}
	}
	if headerMap["X-Override"] != "folder" {
		t.Errorf("expected X-Override: folder, got %q", headerMap["X-Override"])
	}
	// Collection-only header should still be present
	if headerMap["X-Collection"] != "yes" {
		t.Errorf("expected X-Collection: yes, got %q", headerMap["X-Collection"])
	}

	// Collection variable should still be used for base_url
	if req.URL != "https://collection.example.com/admin" {
		t.Errorf("URL = %q, want %q", req.URL, "https://collection.example.com/admin")
	}
}

func TestCollectionRunner_StopOnFailure(t *testing.T) {
	client := &mockHTTPClient{
		responses: []domain.HTTPResponse{
			{StatusCode: 200, Body: []byte(`{}`)},
			{StatusCode: 500, Body: []byte(`{"error":"server error"}`)},
			{StatusCode: 200, Body: []byte(`{}`)},
		},
	}

	cr := runner.NewCollectionRunner(client)

	col := domain.Collection{
		Name: "fail-api",
		Requests: []domain.SavedRequest{
			{Name: "OK Request", Config: domain.RequestConfig{Method: domain.MethodGet, URL: "https://api.example.com/ok"}},
			{Name: "Failing Request", Config: domain.RequestConfig{Method: domain.MethodGet, URL: "https://api.example.com/fail"}},
			{Name: "Skipped Request", Config: domain.RequestConfig{Method: domain.MethodGet, URL: "https://api.example.com/skip"}},
		},
	}

	opts := domain.CollectionRunOptions{Sequential: true, StopOnFailure: true}
	result, err := cr.RunCollection(context.Background(), col, opts)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Passed != 1 {
		t.Errorf("passed = %d, want 1", result.Passed)
	}
	if result.Failed != 1 {
		t.Errorf("failed = %d, want 1", result.Failed)
	}
	if result.Skipped != 1 {
		t.Errorf("skipped = %d, want 1", result.Skipped)
	}
	// Only 2 HTTP calls should have been made (3rd was skipped)
	if client.calls != 2 {
		t.Errorf("http calls = %d, want 2", client.calls)
	}
}

func TestCollectionRunner_Delay_BetweenRequests(t *testing.T) {
	client := &mockHTTPClient{
		responses: []domain.HTTPResponse{
			{StatusCode: 200, Body: []byte(`{}`)},
			{StatusCode: 200, Body: []byte(`{}`)},
			{StatusCode: 200, Body: []byte(`{}`)},
		},
	}

	cr := runner.NewCollectionRunner(client)

	col := domain.Collection{
		Name: "delay-api",
		Requests: []domain.SavedRequest{
			{Name: "R1", Config: domain.RequestConfig{Method: domain.MethodGet, URL: "https://api.example.com/1"}},
			{Name: "R2", Config: domain.RequestConfig{Method: domain.MethodGet, URL: "https://api.example.com/2"}},
			{Name: "R3", Config: domain.RequestConfig{Method: domain.MethodGet, URL: "https://api.example.com/3"}},
		},
	}

	delay := 50 * time.Millisecond
	opts := domain.CollectionRunOptions{
		Sequential:    true,
		StopOnFailure: true,
		Delay:         delay,
	}

	start := time.Now()
	result, err := cr.RunCollection(context.Background(), col, opts)
	elapsed := time.Since(start)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.TotalRequests != 3 {
		t.Errorf("total requests = %d, want 3", result.TotalRequests)
	}

	// Delay is applied between requests (2 gaps for 3 requests)
	expectedMin := 2 * delay
	if elapsed < expectedMin {
		t.Errorf("elapsed = %v, want >= %v (delay between requests)", elapsed, expectedMin)
	}
}

func TestCollectionRunner_NestedFolders_Flattened(t *testing.T) {
	client := &mockHTTPClient{
		responses: []domain.HTTPResponse{
			{StatusCode: 200, Body: []byte(`{}`)},
			{StatusCode: 200, Body: []byte(`{}`)},
			{StatusCode: 200, Body: []byte(`{}`)},
		},
	}

	cr := runner.NewCollectionRunner(client)

	col := domain.Collection{
		Name: "nested-api",
		Folders: []domain.Folder{
			{
				Name: "Users",
				Requests: []domain.SavedRequest{
					{Name: "List Users", Config: domain.RequestConfig{Method: domain.MethodGet, URL: "https://api.example.com/users"}},
				},
				Folders: []domain.Folder{
					{
						Name: "Admin",
						Requests: []domain.SavedRequest{
							{Name: "List Admins", Config: domain.RequestConfig{Method: domain.MethodGet, URL: "https://api.example.com/users/admins"}},
						},
					},
				},
			},
		},
		Requests: []domain.SavedRequest{
			{Name: "Root", Config: domain.RequestConfig{Method: domain.MethodGet, URL: "https://api.example.com/"}},
		},
	}

	opts := domain.CollectionRunOptions{Sequential: true, StopOnFailure: true}
	result, err := cr.RunCollection(context.Background(), col, opts)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.TotalRequests != 3 {
		t.Errorf("total requests = %d, want 3", result.TotalRequests)
	}

	// Check folder paths
	paths := make(map[string]string)
	for _, r := range result.Results {
		paths[r.RequestName] = r.FolderPath
	}

	if paths["Root"] != "" {
		t.Errorf("Root folder path = %q, want empty", paths["Root"])
	}
	if paths["List Users"] != "Users" {
		t.Errorf("List Users folder path = %q, want %q", paths["List Users"], "Users")
	}
	if paths["List Admins"] != "Users/Admin" {
		t.Errorf("List Admins folder path = %q, want %q", paths["List Admins"], "Users/Admin")
	}
}

func TestCollectionRunner_ContinueOnFailure(t *testing.T) {
	client := &mockHTTPClient{
		responses: []domain.HTTPResponse{
			{StatusCode: 200, Body: []byte(`{}`)},
			{StatusCode: 500, Body: []byte(`{}`)},
			{StatusCode: 200, Body: []byte(`{}`)},
		},
	}

	cr := runner.NewCollectionRunner(client)

	col := domain.Collection{
		Name: "continue-api",
		Requests: []domain.SavedRequest{
			{Name: "R1", Config: domain.RequestConfig{Method: domain.MethodGet, URL: "https://api.example.com/1"}},
			{Name: "R2", Config: domain.RequestConfig{Method: domain.MethodGet, URL: "https://api.example.com/2"}},
			{Name: "R3", Config: domain.RequestConfig{Method: domain.MethodGet, URL: "https://api.example.com/3"}},
		},
	}

	opts := domain.CollectionRunOptions{
		Sequential:    true,
		StopOnFailure: false, // continue on failure
	}

	result, err := cr.RunCollection(context.Background(), col, opts)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Passed != 2 {
		t.Errorf("passed = %d, want 2", result.Passed)
	}
	if result.Failed != 1 {
		t.Errorf("failed = %d, want 1", result.Failed)
	}
	if result.Skipped != 0 {
		t.Errorf("skipped = %d, want 0", result.Skipped)
	}
	if client.calls != 3 {
		t.Errorf("http calls = %d, want 3 (all executed)", client.calls)
	}
}

func TestCollectionRunner_NetworkError_Fails(t *testing.T) {
	client := &mockHTTPClient{
		errors: []error{fmt.Errorf("connection refused")},
	}

	cr := runner.NewCollectionRunner(client)

	col := domain.Collection{
		Name: "error-api",
		Requests: []domain.SavedRequest{
			{Name: "Bad Request", Config: domain.RequestConfig{Method: domain.MethodGet, URL: "https://api.example.com/down"}},
		},
	}

	opts := domain.CollectionRunOptions{Sequential: true, StopOnFailure: true}
	result, err := cr.RunCollection(context.Background(), col, opts)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Failed != 1 {
		t.Errorf("failed = %d, want 1", result.Failed)
	}
	if result.Results[0].Error == nil {
		t.Error("expected error in result")
	}
	if result.Results[0].Passed {
		t.Error("expected request to not pass")
	}
}

func TestCollectionRunner_AdditionalVars_Override(t *testing.T) {
	var capturedReqs []domain.HTTPRequest
	client := &capturingHTTPClient{
		response: domain.HTTPResponse{StatusCode: 200, Body: []byte(`{}`)},
		captured: &capturedReqs,
	}

	cr := runner.NewCollectionRunner(client)

	col := domain.Collection{
		Name: "vars-api",
		Variables: []domain.Variable{
			{Key: "base_url", Value: "https://collection.example.com"},
		},
		Requests: []domain.SavedRequest{
			{Name: "Get", Config: domain.RequestConfig{Method: domain.MethodGet, URL: "{{base_url}}/resource"}},
		},
	}

	opts := domain.CollectionRunOptions{
		Sequential:    true,
		StopOnFailure: true,
		Vars: map[string]string{
			"base_url": "https://override.example.com",
		},
	}

	_, err := cr.RunCollection(context.Background(), col, opts)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(capturedReqs) != 1 {
		t.Fatalf("expected 1 request, got %d", len(capturedReqs))
	}
	if capturedReqs[0].URL != "https://override.example.com/resource" {
		t.Errorf("URL = %q, want %q", capturedReqs[0].URL, "https://override.example.com/resource")
	}
}

// capturingHTTPClient records all requests sent through it.
type capturingHTTPClient struct {
	response domain.HTTPResponse
	captured *[]domain.HTTPRequest
}

func (c *capturingHTTPClient) Do(_ context.Context, req domain.HTTPRequest) (domain.HTTPResponse, error) {
	*c.captured = append(*c.captured, req)
	return c.response, nil
}
