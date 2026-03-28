package graphql

import (
	"encoding/json"
	"testing"
)

func TestParseGraphQLResponse_DataOnly(t *testing.T) {
	body := []byte(`{"data": {"users": [{"id": "1", "name": "Alice"}]}}`)

	resp, err := ParseGraphQLResponse(body)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resp.Data == nil {
		t.Fatal("expected data to be present")
	}

	var data map[string]interface{}
	if err := json.Unmarshal(resp.Data, &data); err != nil {
		t.Fatalf("invalid data JSON: %v", err)
	}

	users, ok := data["users"].([]interface{})
	if !ok || len(users) != 1 {
		t.Fatalf("expected 1 user, got %v", data["users"])
	}

	if len(resp.Errors) != 0 {
		t.Errorf("expected no errors, got %d", len(resp.Errors))
	}
}

func TestParseGraphQLResponse_ErrorsOnly(t *testing.T) {
	body := []byte(`{"errors": [{"message": "Not found", "locations": [{"line": 1, "column": 3}], "path": ["user"]}]}`)

	resp, err := ParseGraphQLResponse(body)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(resp.Errors) != 1 {
		t.Fatalf("expected 1 error, got %d", len(resp.Errors))
	}

	gqlErr := resp.Errors[0]
	if gqlErr.Message != "Not found" {
		t.Errorf("expected message 'Not found', got %q", gqlErr.Message)
	}

	if len(gqlErr.Locations) != 1 {
		t.Fatalf("expected 1 location, got %d", len(gqlErr.Locations))
	}

	if gqlErr.Locations[0].Line != 1 || gqlErr.Locations[0].Column != 3 {
		t.Errorf("expected location {1,3}, got {%d,%d}", gqlErr.Locations[0].Line, gqlErr.Locations[0].Column)
	}

	if len(gqlErr.Path) != 1 {
		t.Fatalf("expected 1 path element, got %d", len(gqlErr.Path))
	}
}

func TestParseGraphQLResponse_DataAndErrors(t *testing.T) {
	body := []byte(`{
		"data": {"user": null},
		"errors": [{"message": "Unauthorized field"}]
	}`)

	resp, err := ParseGraphQLResponse(body)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resp.Data == nil {
		t.Error("expected data to be present")
	}

	if len(resp.Errors) != 1 {
		t.Errorf("expected 1 error, got %d", len(resp.Errors))
	}
}

func TestParseGraphQLResponse_InvalidJSON(t *testing.T) {
	body := []byte(`not json`)

	_, err := ParseGraphQLResponse(body)
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}

func TestParseGraphQLResponse_EmptyBody(t *testing.T) {
	_, err := ParseGraphQLResponse(nil)
	if err == nil {
		t.Fatal("expected error for empty body")
	}
}

func TestParseGraphQLResponse_HasErrors(t *testing.T) {
	body := []byte(`{"errors": [{"message": "Bad query"}]}`)

	resp, err := ParseGraphQLResponse(body)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !resp.HasErrors() {
		t.Error("expected HasErrors() to return true")
	}
}

func TestParseGraphQLResponse_NoErrors(t *testing.T) {
	body := []byte(`{"data": {"ok": true}}`)

	resp, err := ParseGraphQLResponse(body)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resp.HasErrors() {
		t.Error("expected HasErrors() to return false")
	}
}
