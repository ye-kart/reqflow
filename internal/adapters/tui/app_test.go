package tui

import (
	"testing"

	"github.com/ye-kart/reqflow/internal/ports/driven"
)

// stubHTTPClient is a minimal stub satisfying driven.HTTPClient.
type stubHTTPClient struct{}

func (s *stubHTTPClient) Do(_ interface{ Context() }, _ interface{}) (interface{}, error) {
	return nil, nil
}

// Compile-time interface verification is done via the real usage.

func TestNewApp(t *testing.T) {
	app := New(nil, nil)
	if app == nil {
		t.Fatal("expected non-nil App")
	}
}

func TestAppFieldsStored(t *testing.T) {
	var hc driven.HTTPClient
	var s driven.Storage
	app := New(hc, s)
	if app.httpClient != hc {
		t.Error("httpClient not stored")
	}
	if app.storage != s {
		t.Error("storage not stored")
	}
}
