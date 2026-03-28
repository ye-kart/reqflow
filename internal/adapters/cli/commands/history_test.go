package commands_test

import (
	"bytes"
	"context"
	"strings"
	"testing"
	"time"

	"github.com/ye-kart/reqflow/internal/adapters/cli/commands"
	"github.com/ye-kart/reqflow/internal/app"
	"github.com/ye-kart/reqflow/internal/domain"
	featurehttp "github.com/ye-kart/reqflow/internal/features/http"
	"github.com/ye-kart/reqflow/internal/features/history"
)

func newTestAppWithHistory(mock *mockHTTPClient, historyDir string) *app.App {
	store := history.NewStore(historyDir, 100)
	return &app.App{
		HTTPExecutor: featurehttp.NewExecutor(mock),
		HistoryStore: store,
	}
}

func seedHistory(t *testing.T, store *history.Store) {
	t.Helper()
	entries := []history.Entry{
		{
			ID:        "20260101-120000-001",
			Timestamp: time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC),
			Method:    "GET",
			URL:       "https://example.com/api/users",
			Status:    200,
			Duration:  150 * time.Millisecond,
			Request: domain.HTTPRequest{
				Method: domain.MethodGet,
				URL:    "https://example.com/api/users",
			},
			Response: domain.HTTPResponse{
				StatusCode: 200,
				Status:     "200 OK",
				Headers:    []domain.Header{{Key: "Content-Type", Value: "application/json"}},
				Body:       []byte(`{"users":["alice","bob"]}`),
				Duration:   150 * time.Millisecond,
			},
		},
		{
			ID:        "20260101-120100-002",
			Timestamp: time.Date(2026, 1, 1, 12, 1, 0, 0, time.UTC),
			Method:    "POST",
			URL:       "https://example.com/api/posts",
			Status:    201,
			Duration:  200 * time.Millisecond,
			Request: domain.HTTPRequest{
				Method: domain.MethodPost,
				URL:    "https://example.com/api/posts",
				Body:   []byte(`{"title":"hello"}`),
			},
			Response: domain.HTTPResponse{
				StatusCode: 201,
				Status:     "201 Created",
				Headers:    []domain.Header{{Key: "Content-Type", Value: "application/json"}},
				Body:       []byte(`{"id":1}`),
				Duration:   200 * time.Millisecond,
			},
		},
	}

	for _, e := range entries {
		if err := store.Add(e); err != nil {
			t.Fatalf("seeding history: %v", err)
		}
	}
}

func TestHistoryList_ShowsEntries(t *testing.T) {
	histDir := t.TempDir()
	mock := &mockHTTPClient{doFunc: noopDoFunc}
	a := newTestAppWithHistory(mock, histDir)
	seedHistory(t, a.HistoryStore)

	root := commands.NewRootCommand(a)
	buf := new(bytes.Buffer)
	root.SetOut(buf)
	root.SetErr(buf)
	root.SetArgs([]string{"history"})

	if err := root.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "example.com/api/users") {
		t.Errorf("expected output to contain users URL, got:\n%s", output)
	}
	if !strings.Contains(output, "example.com/api/posts") {
		t.Errorf("expected output to contain posts URL, got:\n%s", output)
	}
	if !strings.Contains(output, "GET") {
		t.Errorf("expected output to contain GET method, got:\n%s", output)
	}
	if !strings.Contains(output, "POST") {
		t.Errorf("expected output to contain POST method, got:\n%s", output)
	}
}

func TestHistoryList_Empty(t *testing.T) {
	histDir := t.TempDir()
	mock := &mockHTTPClient{doFunc: noopDoFunc}
	a := newTestAppWithHistory(mock, histDir)

	root := commands.NewRootCommand(a)
	buf := new(bytes.Buffer)
	root.SetOut(buf)
	root.SetErr(buf)
	root.SetArgs([]string{"history"})

	if err := root.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "No history") {
		t.Errorf("expected 'No history' message, got:\n%s", output)
	}
}

func TestHistoryShow_DisplaysEntry(t *testing.T) {
	histDir := t.TempDir()
	mock := &mockHTTPClient{doFunc: noopDoFunc}
	a := newTestAppWithHistory(mock, histDir)
	seedHistory(t, a.HistoryStore)

	root := commands.NewRootCommand(a)
	buf := new(bytes.Buffer)
	root.SetOut(buf)
	root.SetErr(buf)
	root.SetArgs([]string{"history", "show", "20260101-120000-001"})

	if err := root.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "GET") {
		t.Errorf("expected method in output, got:\n%s", output)
	}
	if !strings.Contains(output, "https://example.com/api/users") {
		t.Errorf("expected URL in output, got:\n%s", output)
	}
	if !strings.Contains(output, "200") {
		t.Errorf("expected status in output, got:\n%s", output)
	}
}

func TestHistoryShow_NotFound(t *testing.T) {
	histDir := t.TempDir()
	mock := &mockHTTPClient{doFunc: noopDoFunc}
	a := newTestAppWithHistory(mock, histDir)

	root := commands.NewRootCommand(a)
	buf := new(bytes.Buffer)
	root.SetOut(buf)
	root.SetErr(buf)
	root.SetArgs([]string{"history", "show", "nonexistent"})

	err := root.Execute()
	if err == nil {
		t.Fatal("expected error for nonexistent entry")
	}
}

func TestHistorySearch_FindsMatches(t *testing.T) {
	histDir := t.TempDir()
	mock := &mockHTTPClient{doFunc: noopDoFunc}
	a := newTestAppWithHistory(mock, histDir)
	seedHistory(t, a.HistoryStore)

	root := commands.NewRootCommand(a)
	buf := new(bytes.Buffer)
	root.SetOut(buf)
	root.SetErr(buf)
	root.SetArgs([]string{"history", "search", "users"})

	if err := root.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "users") {
		t.Errorf("expected search results to contain 'users', got:\n%s", output)
	}
	if strings.Contains(output, "posts") {
		t.Errorf("expected search results to NOT contain 'posts', got:\n%s", output)
	}
}

func TestHistoryDiff_ShowsDifferences(t *testing.T) {
	histDir := t.TempDir()
	mock := &mockHTTPClient{doFunc: noopDoFunc}
	a := newTestAppWithHistory(mock, histDir)
	seedHistory(t, a.HistoryStore)

	root := commands.NewRootCommand(a)
	buf := new(bytes.Buffer)
	root.SetOut(buf)
	root.SetErr(buf)
	root.SetArgs([]string{"history", "diff", "20260101-120000-001", "20260101-120100-002"})

	if err := root.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := buf.String()
	// Should show status change (200 -> 201)
	if !strings.Contains(output, "200") || !strings.Contains(output, "201") {
		t.Errorf("expected status diff in output, got:\n%s", output)
	}
}

func TestHistoryClear_RemovesAll(t *testing.T) {
	histDir := t.TempDir()
	mock := &mockHTTPClient{doFunc: noopDoFunc}
	a := newTestAppWithHistory(mock, histDir)
	seedHistory(t, a.HistoryStore)

	root := commands.NewRootCommand(a)
	buf := new(bytes.Buffer)
	root.SetOut(buf)
	root.SetErr(buf)
	root.SetArgs([]string{"history", "clear"})

	if err := root.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "Cleared") {
		t.Errorf("expected 'Cleared' message, got:\n%s", output)
	}

	// Verify empty
	entries, err := a.HistoryStore.List(100)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(entries) != 0 {
		t.Errorf("expected 0 entries after clear, got %d", len(entries))
	}
}

func TestNoHistoryFlag_SkipsSaving(t *testing.T) {
	histDir := t.TempDir()
	mock := &mockHTTPClient{
		doFunc: func(_ context.Context, _ domain.HTTPRequest) (domain.HTTPResponse, error) {
			return domain.HTTPResponse{StatusCode: 200, Body: []byte("ok")}, nil
		},
	}
	a := newTestAppWithHistory(mock, histDir)

	root := commands.NewRootCommand(a)
	buf := new(bytes.Buffer)
	root.SetOut(buf)
	root.SetErr(buf)
	root.SetArgs([]string{"get", "https://example.com/api", "--no-history"})

	if err := root.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	entries, err := a.HistoryStore.List(100)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(entries) != 0 {
		t.Errorf("expected 0 entries with --no-history, got %d", len(entries))
	}
}

func TestRequestSavesToHistory(t *testing.T) {
	histDir := t.TempDir()
	mock := &mockHTTPClient{
		doFunc: func(_ context.Context, _ domain.HTTPRequest) (domain.HTTPResponse, error) {
			return domain.HTTPResponse{
				StatusCode: 200,
				Status:     "200 OK",
				Body:       []byte(`{"ok":true}`),
				Duration:   100 * time.Millisecond,
			}, nil
		},
	}
	a := newTestAppWithHistory(mock, histDir)

	root := commands.NewRootCommand(a)
	buf := new(bytes.Buffer)
	root.SetOut(buf)
	root.SetErr(buf)
	root.SetArgs([]string{"get", "https://example.com/api"})

	if err := root.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	entries, err := a.HistoryStore.List(100)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected 1 history entry, got %d", len(entries))
	}
	if entries[0].URL != "https://example.com/api" {
		t.Errorf("expected URL https://example.com/api, got %s", entries[0].URL)
	}
	if entries[0].Status != 200 {
		t.Errorf("expected status 200, got %d", entries[0].Status)
	}
}
