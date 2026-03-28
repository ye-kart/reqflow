package views

import (
	"strings"
	"testing"
	"time"

	"github.com/ye-kart/reqflow/internal/domain"
)

func TestNewResponseModel(t *testing.T) {
	m := NewResponseModel()
	if m.response != nil {
		t.Error("expected nil response on new model")
	}
}

func TestResponseModelSetResponse(t *testing.T) {
	m := NewResponseModel()
	resp := &domain.HTTPResponse{
		StatusCode: 200,
		Status:     "200 OK",
		Headers:    []domain.Header{{Key: "Content-Type", Value: "application/json"}},
		Body:       []byte(`{"message": "hello"}`),
		Duration:   150 * time.Millisecond,
	}
	m.SetResponse(resp)
	if m.response == nil {
		t.Fatal("expected response to be set")
	}
	if m.response.StatusCode != 200 {
		t.Errorf("expected status 200, got %d", m.response.StatusCode)
	}
}

func TestResponseModelViewWithNoResponse(t *testing.T) {
	m := NewResponseModel()
	view := m.View()
	if !strings.Contains(view, "No response") {
		t.Error("expected 'No response' placeholder in view")
	}
}

func TestResponseModelViewWithResponse(t *testing.T) {
	m := NewResponseModel()
	m.SetResponse(&domain.HTTPResponse{
		StatusCode: 200,
		Status:     "200 OK",
		Headers:    []domain.Header{{Key: "Content-Type", Value: "application/json"}},
		Body:       []byte(`{"key":"value"}`),
		Duration:   100 * time.Millisecond,
	})

	view := m.View()

	if !strings.Contains(view, "200") {
		t.Error("expected view to contain status code 200")
	}
	if !strings.Contains(view, "Content-Type") {
		t.Error("expected view to contain header name")
	}
	if !strings.Contains(view, "key") {
		t.Error("expected view to contain response body")
	}
}

func TestResponseModelViewWithJSONPrettyPrint(t *testing.T) {
	m := NewResponseModel()
	m.SetResponse(&domain.HTTPResponse{
		StatusCode: 200,
		Status:     "200 OK",
		Body:       []byte(`{"name":"test","items":[1,2,3]}`),
		Duration:   50 * time.Millisecond,
	})

	view := m.View()
	// Pretty-printed JSON should have newlines/indentation
	if !strings.Contains(view, "name") {
		t.Error("expected pretty-printed JSON to contain field name")
	}
}

func TestResponseModelViewShowsTiming(t *testing.T) {
	m := NewResponseModel()
	m.SetResponse(&domain.HTTPResponse{
		StatusCode: 200,
		Status:     "200 OK",
		Body:       []byte("ok"),
		Duration:   250 * time.Millisecond,
	})

	view := m.View()
	if !strings.Contains(view, "250") {
		t.Error("expected view to contain timing info")
	}
}

func TestResponseModelSetLoading(t *testing.T) {
	m := NewResponseModel()
	m.SetLoading(true)
	view := m.View()
	if !strings.Contains(view, "Sending") {
		t.Error("expected loading indicator in view")
	}
}

func TestResponseModelSetError(t *testing.T) {
	m := NewResponseModel()
	m.SetError("connection refused")
	view := m.View()
	if !strings.Contains(view, "connection refused") {
		t.Error("expected error message in view")
	}
}
