package views

import (
	"context"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/ye-kart/reqflow/internal/domain"
	"github.com/ye-kart/reqflow/internal/ports/driven"
)

// mockHTTPClient implements driven.HTTPClient for testing.
type mockHTTPClient struct {
	lastReq  domain.HTTPRequest
	response domain.HTTPResponse
	err      error
}

func (m *mockHTTPClient) Do(_ context.Context, req domain.HTTPRequest) (domain.HTTPResponse, error) {
	m.lastReq = req
	return m.response, m.err
}

// Ensure mockHTTPClient satisfies the interface.
var _ driven.HTTPClient = (*mockHTTPClient)(nil)

func TestNewMainModel(t *testing.T) {
	m := NewMainModel(nil)
	if m.activeTab != tabRequest {
		t.Errorf("expected initial tab to be Request, got %d", m.activeTab)
	}
}

func TestMainModelInitReturnsNil(t *testing.T) {
	m := NewMainModel(nil)
	cmd := m.Init()
	if cmd != nil {
		t.Error("expected Init to return nil cmd")
	}
}

func TestTabSwitchingWithTab(t *testing.T) {
	m := NewMainModel(nil)

	// Start on Request tab (0)
	if m.activeTab != tabRequest {
		t.Fatalf("expected initial tab Request, got %d", m.activeTab)
	}

	// Press Tab to go to Response
	newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyTab})
	m = newModel.(MainModel)
	if m.activeTab != tabResponse {
		t.Errorf("expected Response tab after Tab, got %d", m.activeTab)
	}

	// Press Tab again to go to History
	newModel, _ = m.Update(tea.KeyMsg{Type: tea.KeyTab})
	m = newModel.(MainModel)
	if m.activeTab != tabHistory {
		t.Errorf("expected History tab after second Tab, got %d", m.activeTab)
	}

	// Press Tab again to wrap around to Request
	newModel, _ = m.Update(tea.KeyMsg{Type: tea.KeyTab})
	m = newModel.(MainModel)
	if m.activeTab != tabRequest {
		t.Errorf("expected Request tab after wrap, got %d", m.activeTab)
	}
}

func TestShiftTabSwitchesBackward(t *testing.T) {
	m := NewMainModel(nil)

	// Shift+Tab from Request wraps to History
	newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyShiftTab})
	m = newModel.(MainModel)
	if m.activeTab != tabHistory {
		t.Errorf("expected History tab after Shift+Tab from Request, got %d", m.activeTab)
	}
}

func TestCtrlCQuits(t *testing.T) {
	m := NewMainModel(nil)
	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyCtrlC})
	if cmd == nil {
		t.Fatal("expected non-nil cmd for Ctrl+C")
	}
	// The cmd should produce a tea.QuitMsg.
	msg := cmd()
	if _, ok := msg.(tea.QuitMsg); !ok {
		t.Errorf("expected QuitMsg, got %T", msg)
	}
}

func TestViewContainsTabNames(t *testing.T) {
	m := NewMainModel(nil)
	view := m.View()
	for _, name := range []string{"Request", "Response", "History"} {
		if !strings.Contains(view, name) {
			t.Errorf("expected view to contain %q", name)
		}
	}
}

func TestViewContainsStatusBar(t *testing.T) {
	m := NewMainModel(nil)
	view := m.View()
	if !strings.Contains(view, "Tab") {
		t.Error("expected view to contain key binding hints")
	}
}
