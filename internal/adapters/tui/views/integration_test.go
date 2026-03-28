package views

import (
	"context"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/ye-kart/reqflow/internal/domain"
)

func TestSendRequestTriggersHTTPCall(t *testing.T) {
	mock := &mockHTTPClient{
		response: domain.HTTPResponse{
			StatusCode: 200,
			Status:     "200 OK",
			Body:       []byte(`{"result":"ok"}`),
			Duration:   50 * time.Millisecond,
		},
	}

	m := NewMainModel(mock)

	// Build a request via the request model
	m.request.method = domain.MethodPost
	m.request.urlInput.SetValue("https://api.test.com/data")
	m.request.bodyInput.SetValue(`{"key":"value"}`)

	req := m.request.BuildRequest()

	// Simulate sending
	sendMsg := SendRequestMsg{Request: req}
	newModel, cmd := m.Update(sendMsg)
	m = newModel.(MainModel)

	// Should switch to response tab and show loading
	if m.activeTab != tabResponse {
		t.Errorf("expected Response tab after send, got %d", m.activeTab)
	}

	// Execute the command to get the response
	if cmd == nil {
		t.Fatal("expected non-nil cmd after send")
	}
	msg := cmd()
	respMsg, ok := msg.(ResponseReceivedMsg)
	if !ok {
		t.Fatalf("expected ResponseReceivedMsg, got %T", msg)
	}
	if respMsg.Err != nil {
		t.Fatalf("unexpected error: %v", respMsg.Err)
	}
	if respMsg.Response.StatusCode != 200 {
		t.Errorf("expected status 200, got %d", respMsg.Response.StatusCode)
	}

	// Verify the mock received the request
	if mock.lastReq.Method != domain.MethodPost {
		t.Errorf("expected POST method in HTTP call, got %s", mock.lastReq.Method)
	}
	if mock.lastReq.URL != "https://api.test.com/data" {
		t.Errorf("expected URL, got %q", mock.lastReq.URL)
	}
}

func TestSendWithNilHTTPClientReturnsError(t *testing.T) {
	m := NewMainModel(nil)

	req := domain.HTTPRequest{
		Method: domain.MethodGet,
		URL:    "https://example.com",
	}

	sendMsg := SendRequestMsg{Request: req}
	_, cmd := m.Update(sendMsg)

	msg := cmd()
	respMsg := msg.(ResponseReceivedMsg)
	if respMsg.Err == nil {
		t.Error("expected error for nil HTTP client")
	}
}

func TestResponseReceivedUpdatesView(t *testing.T) {
	m := NewMainModel(nil)
	m.activeTab = tabResponse

	resp := domain.HTTPResponse{
		StatusCode: 201,
		Status:     "201 Created",
		Headers:    []domain.Header{{Key: "Location", Value: "/items/42"}},
		Body:       []byte(`{"id":42}`),
		Duration:   75 * time.Millisecond,
	}

	newModel, _ := m.Update(ResponseReceivedMsg{Response: resp})
	m = newModel.(MainModel)

	view := m.response.View()
	if !strings.Contains(view, "201") {
		t.Error("expected view to contain status code 201")
	}
	if !strings.Contains(view, "Location") {
		t.Error("expected view to contain Location header")
	}
}

func TestResponseReceivedWithErrorUpdatesView(t *testing.T) {
	m := NewMainModel(nil)
	m.activeTab = tabResponse

	newModel, _ := m.Update(ResponseReceivedMsg{
		Err: context.DeadlineExceeded,
	})
	m = newModel.(MainModel)

	view := m.response.View()
	if !strings.Contains(view, "deadline") {
		t.Error("expected error message in view")
	}
}

func TestHistoryAccumulatesEntries(t *testing.T) {
	mock := &mockHTTPClient{
		response: domain.HTTPResponse{StatusCode: 200, Status: "200 OK", Duration: 10 * time.Millisecond},
	}
	m := NewMainModel(mock)

	// Send first request
	req1 := domain.HTTPRequest{Method: domain.MethodGet, URL: "https://a.com"}
	newModel, _ := m.Update(SendRequestMsg{Request: req1})
	m = newModel.(MainModel)
	cmd := m.sendRequest(req1)
	respMsg := cmd()
	newModel, _ = m.Update(respMsg)
	m = newModel.(MainModel)

	// Send second request
	req2 := domain.HTTPRequest{Method: domain.MethodPost, URL: "https://b.com"}
	m.lastSentReq = &req2
	newModel, _ = m.Update(ResponseReceivedMsg{
		Response: domain.HTTPResponse{StatusCode: 201, Status: "201 Created", Duration: 20 * time.Millisecond},
	})
	m = newModel.(MainModel)

	if len(m.history) != 2 {
		t.Fatalf("expected 2 history entries, got %d", len(m.history))
	}
}

func TestWindowSizeMsgUpdatesSize(t *testing.T) {
	m := NewMainModel(nil)
	newModel, _ := m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	m = newModel.(MainModel)
	if m.width != 120 || m.height != 40 {
		t.Errorf("expected 120x40, got %dx%d", m.width, m.height)
	}
}
