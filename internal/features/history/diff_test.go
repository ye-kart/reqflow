package history_test

import (
	"strings"
	"testing"
	"time"

	"github.com/ye-kart/reqflow/internal/domain"
	"github.com/ye-kart/reqflow/internal/features/history"
)

func TestCompare_SameResponse_NoDiff(t *testing.T) {
	e := newTestEntry("same-1", "https://example.com/api", 200)
	diff := history.Compare(e, e)

	if diff.StatusChanged {
		t.Error("expected StatusChanged=false for identical entries")
	}
	if len(diff.HeaderDiffs) != 0 {
		t.Errorf("expected 0 header diffs, got %d", len(diff.HeaderDiffs))
	}
	if diff.BodyDiff != "" {
		t.Errorf("expected empty body diff, got %q", diff.BodyDiff)
	}
}

func TestCompare_DifferentStatus(t *testing.T) {
	a := newTestEntry("diff-status-a", "https://example.com/api", 200)
	b := newTestEntry("diff-status-b", "https://example.com/api", 404)

	diff := history.Compare(a, b)

	if !diff.StatusChanged {
		t.Error("expected StatusChanged=true")
	}
	if diff.OldStatus != 200 {
		t.Errorf("expected OldStatus=200, got %d", diff.OldStatus)
	}
	if diff.NewStatus != 404 {
		t.Errorf("expected NewStatus=404, got %d", diff.NewStatus)
	}
}

func TestCompare_DifferentHeaders(t *testing.T) {
	a := history.Entry{
		ID:        "hdr-a",
		Timestamp: time.Now(),
		Method:    "GET",
		URL:       "https://example.com/api",
		Status:    200,
		Response: domain.HTTPResponse{
			StatusCode: 200,
			Headers: []domain.Header{
				{Key: "Content-Type", Value: "application/json"},
				{Key: "X-Request-Id", Value: "abc123"},
			},
			Body: []byte("{}"),
		},
	}
	b := history.Entry{
		ID:        "hdr-b",
		Timestamp: time.Now(),
		Method:    "GET",
		URL:       "https://example.com/api",
		Status:    200,
		Response: domain.HTTPResponse{
			StatusCode: 200,
			Headers: []domain.Header{
				{Key: "Content-Type", Value: "text/html"},
				{Key: "X-New-Header", Value: "new"},
			},
			Body: []byte("{}"),
		},
	}

	diff := history.Compare(a, b)

	if diff.StatusChanged {
		t.Error("expected StatusChanged=false")
	}

	// Should detect: Content-Type changed, X-Request-Id removed, X-New-Header added
	if len(diff.HeaderDiffs) < 3 {
		t.Fatalf("expected at least 3 header diffs, got %d: %+v", len(diff.HeaderDiffs), diff.HeaderDiffs)
	}

	found := map[string]bool{}
	for _, hd := range diff.HeaderDiffs {
		switch hd.Key {
		case "Content-Type":
			found["changed"] = true
			if hd.OldValue != "application/json" || hd.NewValue != "text/html" {
				t.Errorf("Content-Type diff: expected json->html, got %s->%s", hd.OldValue, hd.NewValue)
			}
		case "X-Request-Id":
			found["removed"] = true
			if !hd.Removed {
				t.Error("expected X-Request-Id to be marked as Removed")
			}
		case "X-New-Header":
			found["added"] = true
			if !hd.Added {
				t.Error("expected X-New-Header to be marked as Added")
			}
		}
	}

	if !found["changed"] {
		t.Error("missing Content-Type change in diffs")
	}
	if !found["removed"] {
		t.Error("missing X-Request-Id removal in diffs")
	}
	if !found["added"] {
		t.Error("missing X-New-Header addition in diffs")
	}
}

func TestCompare_DifferentBody(t *testing.T) {
	a := history.Entry{
		ID:        "body-a",
		Timestamp: time.Now(),
		Method:    "GET",
		URL:       "https://example.com/api",
		Status:    200,
		Response: domain.HTTPResponse{
			StatusCode: 200,
			Body:       []byte(`{"name":"Alice","age":30}`),
		},
	}
	b := history.Entry{
		ID:        "body-b",
		Timestamp: time.Now(),
		Method:    "GET",
		URL:       "https://example.com/api",
		Status:    200,
		Response: domain.HTTPResponse{
			StatusCode: 200,
			Body:       []byte(`{"name":"Bob","age":25}`),
		},
	}

	diff := history.Compare(a, b)

	if diff.BodyDiff == "" {
		t.Error("expected non-empty body diff")
	}
	if !strings.Contains(diff.BodyDiff, "Alice") || !strings.Contains(diff.BodyDiff, "Bob") {
		t.Errorf("body diff should reference both old and new values, got:\n%s", diff.BodyDiff)
	}
}

func TestCompare_EmptyBodiesNoDiff(t *testing.T) {
	a := history.Entry{
		ID:     "empty-a",
		Status: 204,
		Response: domain.HTTPResponse{
			StatusCode: 204,
			Body:       nil,
		},
	}
	b := history.Entry{
		ID:     "empty-b",
		Status: 204,
		Response: domain.HTTPResponse{
			StatusCode: 204,
			Body:       nil,
		},
	}

	diff := history.Compare(a, b)
	if diff.BodyDiff != "" {
		t.Errorf("expected empty body diff for nil bodies, got %q", diff.BodyDiff)
	}
}
