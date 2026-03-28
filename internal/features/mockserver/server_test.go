package mockserver

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/ye-kart/reqflow/internal/domain"
)

func testCollection() domain.Collection {
	return domain.Collection{
		Name: "test-api",
		Requests: []domain.SavedRequest{
			{
				Name: "get-users",
				Config: domain.RequestConfig{
					Method: domain.MethodGet,
					URL:    "https://api.example.com/users",
				},
				Response: &domain.ExampleResponse{
					StatusCode: 200,
					Headers:    []domain.Header{{Key: "Content-Type", Value: "application/json"}},
					Body:       `[{"id":1,"name":"Alice"}]`,
				},
			},
			{
				Name: "create-user",
				Config: domain.RequestConfig{
					Method: domain.MethodPost,
					URL:    "https://api.example.com/users",
				},
				Response: &domain.ExampleResponse{
					StatusCode: 201,
					Body:       `{"id":2,"name":"Bob"}`,
				},
			},
			{
				Name: "get-user-by-id",
				Config: domain.RequestConfig{
					Method: domain.MethodGet,
					URL:    "https://api.example.com/users/{{id}}",
				},
				Response: &domain.ExampleResponse{
					StatusCode: 200,
					Body:       `{"id":"123","name":"Charlie"}`,
				},
			},
		},
	}
}

func TestServer_ExactPathMatch(t *testing.T) {
	srv := New(testCollection(), 0)
	ts := httptest.NewServer(srv.handler())
	defer ts.Close()

	resp, err := http.Get(ts.URL + "/users")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}

	body, _ := io.ReadAll(resp.Body)
	if string(body) != `[{"id":1,"name":"Alice"}]` {
		t.Errorf("unexpected body: %s", body)
	}

	ct := resp.Header.Get("Content-Type")
	if ct != "application/json" {
		t.Errorf("expected Content-Type application/json, got %s", ct)
	}
}

func TestServer_UnmatchedPath_Returns404(t *testing.T) {
	srv := New(testCollection(), 0)
	ts := httptest.NewServer(srv.handler())
	defer ts.Close()

	resp, err := http.Get(ts.URL + "/nonexistent")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 404 {
		t.Errorf("expected 404, got %d", resp.StatusCode)
	}
}

func TestServer_MethodMismatch_Returns404(t *testing.T) {
	srv := New(testCollection(), 0)
	ts := httptest.NewServer(srv.handler())
	defer ts.Close()

	resp, err := http.Post(ts.URL+"/users/123", "application/json", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 404 {
		t.Errorf("expected 404 for method mismatch, got %d", resp.StatusCode)
	}
}

func TestServer_PathParamMatching(t *testing.T) {
	srv := New(testCollection(), 0)
	ts := httptest.NewServer(srv.handler())
	defer ts.Close()

	resp, err := http.Get(ts.URL + "/users/456")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}

	body, _ := io.ReadAll(resp.Body)
	if string(body) != `{"id":"123","name":"Charlie"}` {
		t.Errorf("unexpected body: %s", body)
	}
}

func TestServer_MultipleRoutes(t *testing.T) {
	srv := New(testCollection(), 0)
	ts := httptest.NewServer(srv.handler())
	defer ts.Close()

	// GET /users
	resp, err := http.Get(ts.URL + "/users")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode != 200 {
		t.Errorf("GET /users: expected 200, got %d", resp.StatusCode)
	}

	// POST /users
	resp, err = http.Post(ts.URL+"/users", "application/json", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode != 201 {
		t.Errorf("POST /users: expected 201, got %d", resp.StatusCode)
	}

	// GET /users/99
	resp, err = http.Get(ts.URL + "/users/99")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode != 200 {
		t.Errorf("GET /users/99: expected 200, got %d", resp.StatusCode)
	}
}

func TestServer_ResponseDelay(t *testing.T) {
	srv := New(testCollection(), 0, WithDelay(100*time.Millisecond))
	ts := httptest.NewServer(srv.handler())
	defer ts.Close()

	start := time.Now()
	resp, err := http.Get(ts.URL + "/users")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	resp.Body.Close()
	elapsed := time.Since(start)

	if elapsed < 100*time.Millisecond {
		t.Errorf("expected at least 100ms delay, got %v", elapsed)
	}
}

func TestServer_FolderRequests(t *testing.T) {
	col := domain.Collection{
		Name: "folder-test",
		Folders: []domain.Folder{
			{
				Name: "admin",
				Requests: []domain.SavedRequest{
					{
						Name: "admin-health",
						Config: domain.RequestConfig{
							Method: domain.MethodGet,
							URL:    "https://api.example.com/admin/health",
						},
						Response: &domain.ExampleResponse{
							StatusCode: 200,
							Body:       `{"status":"ok"}`,
						},
					},
				},
			},
		},
	}

	srv := New(col, 0)
	ts := httptest.NewServer(srv.handler())
	defer ts.Close()

	resp, err := http.Get(ts.URL + "/admin/health")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}

	body, _ := io.ReadAll(resp.Body)
	if string(body) != `{"status":"ok"}` {
		t.Errorf("unexpected body: %s", body)
	}
}
