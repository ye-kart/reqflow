package history_test

import (
	"os"
	"strings"
	"testing"
	"time"

	"github.com/ye-kart/reqflow/internal/domain"
	"github.com/ye-kart/reqflow/internal/features/history"
)

func newTestEntry(id string, url string, status int) history.Entry {
	return history.Entry{
		ID:        id,
		Timestamp: time.Now(),
		Method:    "GET",
		URL:       url,
		Status:    status,
		Duration:  100 * time.Millisecond,
		Request: domain.HTTPRequest{
			Method: domain.MethodGet,
			URL:    url,
		},
		Response: domain.HTTPResponse{
			StatusCode: status,
			Status:     "200 OK",
			Body:       []byte(`{"ok":true}`),
			Headers: []domain.Header{
				{Key: "Content-Type", Value: "application/json"},
			},
		},
	}
}

func TestStore_AddAndList(t *testing.T) {
	dir := t.TempDir()
	store := history.NewStore(dir, 100)

	e1 := newTestEntry("20260101-120000-001", "https://example.com/api/users", 200)
	e1.Timestamp = time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	e2 := newTestEntry("20260101-120001-002", "https://example.com/api/posts", 201)
	e2.Timestamp = time.Date(2026, 1, 1, 12, 1, 0, 0, time.UTC)

	if err := store.Add(e1); err != nil {
		t.Fatalf("Add(e1): %v", err)
	}
	if err := store.Add(e2); err != nil {
		t.Fatalf("Add(e2): %v", err)
	}

	entries, err := store.List(10)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(entries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(entries))
	}

	// Most recent first
	if entries[0].ID != e2.ID {
		t.Errorf("expected most recent entry first, got %s", entries[0].ID)
	}
}

func TestStore_ListLimit(t *testing.T) {
	dir := t.TempDir()
	store := history.NewStore(dir, 100)

	for i := 0; i < 5; i++ {
		e := newTestEntry("entry-"+string(rune('a'+i)), "https://example.com/"+string(rune('a'+i)), 200)
		if err := store.Add(e); err != nil {
			t.Fatalf("Add: %v", err)
		}
	}

	entries, err := store.List(3)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(entries) != 3 {
		t.Errorf("expected 3 entries, got %d", len(entries))
	}
}

func TestStore_GetByID(t *testing.T) {
	dir := t.TempDir()
	store := history.NewStore(dir, 100)

	e := newTestEntry("test-id-123", "https://example.com/api", 200)
	if err := store.Add(e); err != nil {
		t.Fatalf("Add: %v", err)
	}

	got, err := store.Get("test-id-123")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.ID != "test-id-123" {
		t.Errorf("expected ID test-id-123, got %s", got.ID)
	}
	if got.URL != "https://example.com/api" {
		t.Errorf("expected URL https://example.com/api, got %s", got.URL)
	}
	if got.Status != 200 {
		t.Errorf("expected Status 200, got %d", got.Status)
	}
}

func TestStore_GetNotFound(t *testing.T) {
	dir := t.TempDir()
	store := history.NewStore(dir, 100)

	_, err := store.Get("nonexistent")
	if err == nil {
		t.Fatal("expected error for nonexistent ID")
	}
}

func TestStore_SearchByURL(t *testing.T) {
	dir := t.TempDir()
	store := history.NewStore(dir, 100)

	e1 := newTestEntry("search-1", "https://example.com/api/users", 200)
	e2 := newTestEntry("search-2", "https://example.com/api/posts", 201)
	e3 := newTestEntry("search-3", "https://other.com/api/users", 200)

	for _, e := range []history.Entry{e1, e2, e3} {
		if err := store.Add(e); err != nil {
			t.Fatalf("Add: %v", err)
		}
	}

	results, err := store.Search("example.com")
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("expected 2 results for 'example.com', got %d", len(results))
	}

	results, err = store.Search("users")
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("expected 2 results for 'users', got %d", len(results))
	}

	results, err = store.Search("posts")
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result for 'posts', got %d", len(results))
	}
}

func TestStore_MaxEntriesEviction(t *testing.T) {
	dir := t.TempDir()
	store := history.NewStore(dir, 3)

	for i := 0; i < 5; i++ {
		e := newTestEntry("evict-"+string(rune('a'+i)), "https://example.com/"+string(rune('a'+i)), 200)
		e.Timestamp = time.Now().Add(time.Duration(i) * time.Second)
		if err := store.Add(e); err != nil {
			t.Fatalf("Add: %v", err)
		}
	}

	entries, err := store.List(100)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(entries) != 3 {
		t.Errorf("expected 3 entries after eviction, got %d", len(entries))
	}

	// Oldest entries should have been evicted
	for _, e := range entries {
		if e.ID == "evict-a" || e.ID == "evict-b" {
			t.Errorf("expected evicted entry %s to be gone", e.ID)
		}
	}
}

func TestStore_Clear(t *testing.T) {
	dir := t.TempDir()
	store := history.NewStore(dir, 100)

	for i := 0; i < 3; i++ {
		e := newTestEntry("clear-"+string(rune('a'+i)), "https://example.com/"+string(rune('a'+i)), 200)
		if err := store.Add(e); err != nil {
			t.Fatalf("Add: %v", err)
		}
	}

	if err := store.Clear(); err != nil {
		t.Fatalf("Clear: %v", err)
	}

	entries, err := store.List(100)
	if err != nil {
		t.Fatalf("List after clear: %v", err)
	}
	if len(entries) != 0 {
		t.Errorf("expected 0 entries after clear, got %d", len(entries))
	}

	// Directory should still exist
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		t.Error("expected directory to still exist after clear")
	}
}

func TestStore_PreservesRequestResponse(t *testing.T) {
	dir := t.TempDir()
	store := history.NewStore(dir, 100)

	e := history.Entry{
		ID:        "preserve-test",
		Timestamp: time.Now(),
		Method:    "POST",
		URL:       "https://example.com/api",
		Status:    201,
		Duration:  250 * time.Millisecond,
		Request: domain.HTTPRequest{
			Method:      domain.MethodPost,
			URL:         "https://example.com/api",
			Headers:     []domain.Header{{Key: "Content-Type", Value: "application/json"}},
			Body:        []byte(`{"name":"test"}`),
			ContentType: "application/json",
		},
		Response: domain.HTTPResponse{
			StatusCode: 201,
			Status:     "201 Created",
			Headers:    []domain.Header{{Key: "Location", Value: "/api/1"}},
			Body:       []byte(`{"id":1}`),
			Duration:   250 * time.Millisecond,
		},
	}

	if err := store.Add(e); err != nil {
		t.Fatalf("Add: %v", err)
	}

	got, err := store.Get("preserve-test")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}

	if got.Method != "POST" {
		t.Errorf("Method: expected POST, got %s", got.Method)
	}
	if string(got.Request.Body) != `{"name":"test"}` {
		t.Errorf("Request.Body: expected {\"name\":\"test\"}, got %s", string(got.Request.Body))
	}
	if string(got.Response.Body) != `{"id":1}` {
		t.Errorf("Response.Body: expected {\"id\":1}, got %s", string(got.Response.Body))
	}
	if got.Response.StatusCode != 201 {
		t.Errorf("Response.StatusCode: expected 201, got %d", got.Response.StatusCode)
	}
	if len(got.Response.Headers) != 1 || got.Response.Headers[0].Key != "Location" {
		t.Errorf("Response.Headers: expected Location header, got %v", got.Response.Headers)
	}
}

func TestStore_SearchCaseInsensitive(t *testing.T) {
	dir := t.TempDir()
	store := history.NewStore(dir, 100)

	e := newTestEntry("case-test", "https://Example.COM/API/Users", 200)
	if err := store.Add(e); err != nil {
		t.Fatalf("Add: %v", err)
	}

	results, err := store.Search("example.com")
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if len(results) != 1 {
		t.Errorf("expected 1 result for case-insensitive search, got %d", len(results))
	}
}

func TestStore_GenerateID(t *testing.T) {
	id := history.GenerateID()
	if id == "" {
		t.Error("expected non-empty ID")
	}
	// ID should contain a timestamp-like prefix
	if !strings.Contains(id, "-") {
		t.Errorf("expected ID to contain dashes, got %s", id)
	}
}
