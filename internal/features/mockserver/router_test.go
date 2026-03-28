package mockserver

import (
	"testing"

	"github.com/ye-kart/reqflow/internal/domain"
)

func TestMatchRoute_ExactPath(t *testing.T) {
	routes := buildRoutes([]domain.SavedRequest{
		{
			Name: "get-users",
			Config: domain.RequestConfig{
				Method: domain.MethodGet,
				URL:    "https://api.example.com/users",
			},
			Response: &domain.ExampleResponse{
				StatusCode: 200,
				Body:       `[{"id":1}]`,
			},
		},
	})

	resp := matchRoute("GET", "/users", routes)
	if resp == nil {
		t.Fatal("expected a match, got nil")
	}
	if resp.StatusCode != 200 {
		t.Errorf("expected status 200, got %d", resp.StatusCode)
	}
	if resp.Body != `[{"id":1}]` {
		t.Errorf("unexpected body: %s", resp.Body)
	}
}

func TestMatchRoute_MethodMismatch(t *testing.T) {
	routes := buildRoutes([]domain.SavedRequest{
		{
			Name: "get-users",
			Config: domain.RequestConfig{
				Method: domain.MethodGet,
				URL:    "https://api.example.com/users",
			},
			Response: &domain.ExampleResponse{
				StatusCode: 200,
				Body:       `[{"id":1}]`,
			},
		},
	})

	resp := matchRoute("POST", "/users", routes)
	if resp != nil {
		t.Fatal("expected no match for method mismatch")
	}
}

func TestMatchRoute_UnmatchedPath(t *testing.T) {
	routes := buildRoutes([]domain.SavedRequest{
		{
			Name: "get-users",
			Config: domain.RequestConfig{
				Method: domain.MethodGet,
				URL:    "https://api.example.com/users",
			},
			Response: &domain.ExampleResponse{
				StatusCode: 200,
				Body:       `[{"id":1}]`,
			},
		},
	})

	resp := matchRoute("GET", "/posts", routes)
	if resp != nil {
		t.Fatal("expected no match for unmatched path")
	}
}

func TestMatchRoute_PathParams(t *testing.T) {
	routes := buildRoutes([]domain.SavedRequest{
		{
			Name: "get-user-by-id",
			Config: domain.RequestConfig{
				Method: domain.MethodGet,
				URL:    "https://api.example.com/users/{{id}}",
			},
			Response: &domain.ExampleResponse{
				StatusCode: 200,
				Body:       `{"id":"123"}`,
			},
		},
	})

	resp := matchRoute("GET", "/users/123", routes)
	if resp == nil {
		t.Fatal("expected a match for path param, got nil")
	}
	if resp.StatusCode != 200 {
		t.Errorf("expected status 200, got %d", resp.StatusCode)
	}
}

func TestMatchRoute_MultipleRoutes(t *testing.T) {
	routes := buildRoutes([]domain.SavedRequest{
		{
			Name: "get-users",
			Config: domain.RequestConfig{
				Method: domain.MethodGet,
				URL:    "https://api.example.com/users",
			},
			Response: &domain.ExampleResponse{
				StatusCode: 200,
				Body:       `{"route":"users"}`,
			},
		},
		{
			Name: "get-posts",
			Config: domain.RequestConfig{
				Method: domain.MethodGet,
				URL:    "https://api.example.com/posts",
			},
			Response: &domain.ExampleResponse{
				StatusCode: 200,
				Body:       `{"route":"posts"}`,
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
				Body:       `{"created":true}`,
			},
		},
	})

	tests := []struct {
		method     string
		path       string
		wantStatus int
		wantBody   string
	}{
		{"GET", "/users", 200, `{"route":"users"}`},
		{"GET", "/posts", 200, `{"route":"posts"}`},
		{"POST", "/users", 201, `{"created":true}`},
	}

	for _, tt := range tests {
		resp := matchRoute(tt.method, tt.path, routes)
		if resp == nil {
			t.Errorf("matchRoute(%s, %s) = nil, want match", tt.method, tt.path)
			continue
		}
		if resp.StatusCode != tt.wantStatus {
			t.Errorf("matchRoute(%s, %s) status = %d, want %d", tt.method, tt.path, resp.StatusCode, tt.wantStatus)
		}
		if resp.Body != tt.wantBody {
			t.Errorf("matchRoute(%s, %s) body = %s, want %s", tt.method, tt.path, resp.Body, tt.wantBody)
		}
	}
}

func TestMatchRoute_NoResponse(t *testing.T) {
	routes := buildRoutes([]domain.SavedRequest{
		{
			Name: "no-response",
			Config: domain.RequestConfig{
				Method: domain.MethodGet,
				URL:    "https://api.example.com/health",
			},
			// Response is nil
		},
	})

	resp := matchRoute("GET", "/health", routes)
	if resp != nil {
		t.Fatal("expected nil for request with no example response")
	}
}

func TestExtractPath(t *testing.T) {
	tests := []struct {
		url  string
		want string
	}{
		{"https://api.example.com/users", "/users"},
		{"https://api.example.com/users/{{id}}", "/users/{{id}}"},
		{"http://localhost:3000/api/v1/posts", "/api/v1/posts"},
		{"/plain-path", "/plain-path"},
		{"https://api.example.com", "/"},
	}

	for _, tt := range tests {
		got := extractPath(tt.url)
		if got != tt.want {
			t.Errorf("extractPath(%q) = %q, want %q", tt.url, got, tt.want)
		}
	}
}

func TestPathMatches(t *testing.T) {
	tests := []struct {
		pattern string
		path    string
		want    bool
	}{
		{"/users", "/users", true},
		{"/users", "/posts", false},
		{"/users/{{id}}", "/users/123", true},
		{"/users/{{id}}", "/users/abc", true},
		{"/users/{{id}}", "/users/", false},
		{"/users/{{id}}/posts/{{postId}}", "/users/1/posts/42", true},
		{"/users/{{id}}/posts/{{postId}}", "/users/1/posts", false},
		{"/users", "/users/123", false},
	}

	for _, tt := range tests {
		got := pathMatches(tt.pattern, tt.path)
		if got != tt.want {
			t.Errorf("pathMatches(%q, %q) = %v, want %v", tt.pattern, tt.path, got, tt.want)
		}
	}
}
