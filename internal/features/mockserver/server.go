package mockserver

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"time"

	"github.com/ye-kart/reqflow/internal/domain"
)

// Option is a functional option for configuring a MockServer.
type Option func(*MockServer)

// WithDelay sets a response delay for all mock responses.
func WithDelay(d time.Duration) Option {
	return func(s *MockServer) {
		s.delay = d
	}
}

// MockServer serves canned responses from a collection.
type MockServer struct {
	collection domain.Collection
	port       int
	delay      time.Duration
	routes     []route
	server     *http.Server
}

// New creates a new MockServer from the given collection and port.
func New(collection domain.Collection, port int, opts ...Option) *MockServer {
	s := &MockServer{
		collection: collection,
		port:       port,
	}
	for _, opt := range opts {
		opt(s)
	}
	s.routes = s.collectRoutes()
	return s
}

// collectRoutes gathers all routes from the collection, including nested folders.
func (s *MockServer) collectRoutes() []route {
	var requests []domain.SavedRequest
	requests = append(requests, s.collection.Requests...)
	requests = append(requests, collectFolderRequests(s.collection.Folders)...)
	return buildRoutes(requests)
}

// collectFolderRequests recursively collects requests from nested folders.
func collectFolderRequests(folders []domain.Folder) []domain.SavedRequest {
	var requests []domain.SavedRequest
	for _, f := range folders {
		requests = append(requests, f.Requests...)
		requests = append(requests, collectFolderRequests(f.Folders)...)
	}
	return requests
}

// handler returns the http.Handler for the mock server.
func (s *MockServer) handler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if s.delay > 0 {
			time.Sleep(s.delay)
		}

		resp := matchRoute(r.Method, r.URL.Path, s.routes)
		if resp == nil {
			http.NotFound(w, r)
			return
		}

		for _, h := range resp.Headers {
			w.Header().Set(h.Key, h.Value)
		}

		statusCode := resp.StatusCode
		if statusCode == 0 {
			statusCode = 200
		}
		w.WriteHeader(statusCode)

		if resp.Body != "" {
			fmt.Fprint(w, resp.Body)
		}
	})
}

// Start starts the mock server on the configured port. It blocks until the
// server is shut down.
func (s *MockServer) Start() error {
	addr := fmt.Sprintf(":%d", s.port)
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("listening on %s: %w", addr, err)
	}

	s.server = &http.Server{
		Handler: s.handler(),
	}

	// Update port with the actual bound port (useful when port=0).
	s.port = listener.Addr().(*net.TCPAddr).Port

	return s.server.Serve(listener)
}

// Stop gracefully shuts down the mock server.
func (s *MockServer) Stop() error {
	if s.server == nil {
		return nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return s.server.Shutdown(ctx)
}

// Port returns the port the server is configured to listen on.
func (s *MockServer) Port() int {
	return s.port
}

// RouteCount returns the number of registered routes.
func (s *MockServer) RouteCount() int {
	return len(s.routes)
}
