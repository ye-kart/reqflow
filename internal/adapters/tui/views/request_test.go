package views

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/ye-kart/reqflow/internal/domain"
)

func TestNewRequestModel(t *testing.T) {
	m := NewRequestModel()
	if m.method != domain.MethodGet {
		t.Errorf("expected default method GET, got %s", m.method)
	}
}

func TestRequestModelMethodCycle(t *testing.T) {
	m := NewRequestModel()
	m.focused = focusMethod

	// Cycle forward through methods
	newModel := m.updateMethodSelector(tea.KeyMsg{Type: tea.KeyDown})
	rm := newModel.(RequestModel)
	if rm.method != domain.MethodPost {
		t.Errorf("expected POST after Down, got %s", rm.method)
	}

	// Cycle backward
	newModel = rm.updateMethodSelector(tea.KeyMsg{Type: tea.KeyUp})
	rm = newModel.(RequestModel)
	if rm.method != domain.MethodGet {
		t.Errorf("expected GET after Up, got %s", rm.method)
	}
}

func TestRequestModelURLInput(t *testing.T) {
	m := NewRequestModel()
	m.focused = focusURL

	// Type a URL character by character
	url := "https://api.example.com"
	for _, ch := range url {
		newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{ch}})
		m = newModel.(RequestModel)
	}

	got := m.URL()
	if got != url {
		t.Errorf("expected URL %q, got %q", url, got)
	}
}

func TestRequestModelViewContainsMethod(t *testing.T) {
	m := NewRequestModel()
	view := m.View()
	if !strings.Contains(view, "GET") {
		t.Error("expected view to contain GET method")
	}
}

func TestRequestModelViewContainsURLLabel(t *testing.T) {
	m := NewRequestModel()
	view := m.View()
	if !strings.Contains(view, "URL") {
		t.Error("expected view to contain URL label")
	}
}

func TestRequestModelBuildRequest(t *testing.T) {
	m := NewRequestModel()

	// Set method and URL
	m.method = domain.MethodPost
	m.urlInput.SetValue("https://api.example.com/data")
	m.bodyInput.SetValue(`{"key": "value"}`)

	req := m.BuildRequest()
	if req.Method != domain.MethodPost {
		t.Errorf("expected POST, got %s", req.Method)
	}
	if req.URL != "https://api.example.com/data" {
		t.Errorf("expected URL, got %q", req.URL)
	}
	if string(req.Body) != `{"key": "value"}` {
		t.Errorf("expected body, got %q", string(req.Body))
	}
}

func TestRequestModelHeaders(t *testing.T) {
	m := NewRequestModel()
	m.AddHeader("Content-Type", "application/json")
	m.AddHeader("Authorization", "Bearer token123")

	req := m.BuildRequest()
	if len(req.Headers) != 2 {
		t.Fatalf("expected 2 headers, got %d", len(req.Headers))
	}
	if req.Headers[0].Key != "Content-Type" || req.Headers[0].Value != "application/json" {
		t.Errorf("unexpected first header: %+v", req.Headers[0])
	}
}

func TestRequestModelRemoveHeader(t *testing.T) {
	m := NewRequestModel()
	m.AddHeader("Content-Type", "application/json")
	m.AddHeader("Authorization", "Bearer token123")
	m.RemoveHeader(0)

	req := m.BuildRequest()
	if len(req.Headers) != 1 {
		t.Fatalf("expected 1 header after remove, got %d", len(req.Headers))
	}
	if req.Headers[0].Key != "Authorization" {
		t.Errorf("expected Authorization header, got %s", req.Headers[0].Key)
	}
}

func TestRequestModelFocusNavigation(t *testing.T) {
	m := NewRequestModel()
	if m.focused != focusMethod {
		t.Fatalf("expected initial focus on method, got %d", m.focused)
	}

	// Tab moves to next field
	newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyTab})
	m = newModel.(RequestModel)
	if m.focused != focusURL {
		t.Errorf("expected focus on URL after Tab, got %d", m.focused)
	}
}

func TestEnterOnSendReturnsSendMsg(t *testing.T) {
	m := NewRequestModel()
	m.focused = focusSend
	m.urlInput.SetValue("https://example.com")

	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil {
		t.Fatal("expected non-nil cmd when pressing Enter on Send")
	}
	msg := cmd()
	if _, ok := msg.(SendRequestMsg); !ok {
		t.Errorf("expected SendRequestMsg, got %T", msg)
	}
}
