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

func TestTabSwitchingWithNumberKeys(t *testing.T) {
	m := NewMainModel(nil)

	// Start on Request tab (0)
	if m.activeTab != tabRequest {
		t.Fatalf("expected initial tab Request, got %d", m.activeTab)
	}

	// Press '2' to go to Response
	newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'2'}})
	m = newModel.(MainModel)
	if m.activeTab != tabResponse {
		t.Errorf("expected Response tab after '2', got %d", m.activeTab)
	}

	// Press '3' to go to History
	newModel, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'3'}})
	m = newModel.(MainModel)
	if m.activeTab != tabHistory {
		t.Errorf("expected History tab after '3', got %d", m.activeTab)
	}

	// Press '1' to go to Request
	newModel, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'1'}})
	m = newModel.(MainModel)
	if m.activeTab != tabRequest {
		t.Errorf("expected Request tab after '1', got %d", m.activeTab)
	}
}

func TestTabSwitchingWithTabFromNonRequestTab(t *testing.T) {
	m := NewMainModel(nil)

	// Go to Response tab first
	newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'2'}})
	m = newModel.(MainModel)

	// Tab from Response tab should switch to History
	newModel, _ = m.Update(tea.KeyMsg{Type: tea.KeyTab})
	m = newModel.(MainModel)
	if m.activeTab != tabHistory {
		t.Errorf("expected History tab after Tab from Response, got %d", m.activeTab)
	}

	// Tab from History wraps to Request
	newModel, _ = m.Update(tea.KeyMsg{Type: tea.KeyTab})
	m = newModel.(MainModel)
	if m.activeTab != tabRequest {
		t.Errorf("expected Request tab after Tab from History, got %d", m.activeTab)
	}
}

func TestShiftTabSwitchesBackwardFromNonRequestTab(t *testing.T) {
	m := NewMainModel(nil)

	// Go to History tab
	newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'3'}})
	m = newModel.(MainModel)

	// Shift+Tab from History goes to Response
	newModel, _ = m.Update(tea.KeyMsg{Type: tea.KeyShiftTab})
	m = newModel.(MainModel)
	if m.activeTab != tabResponse {
		t.Errorf("expected Response tab after Shift+Tab from History, got %d", m.activeTab)
	}
}

func TestTabOnRequestTabNavigatesFormFields(t *testing.T) {
	m := NewMainModel(nil)
	// On Request tab, Tab should navigate form fields not switch tabs
	newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyTab})
	m = newModel.(MainModel)
	if m.activeTab != tabRequest {
		t.Errorf("expected to stay on Request tab when Tab navigates fields, got %d", m.activeTab)
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
