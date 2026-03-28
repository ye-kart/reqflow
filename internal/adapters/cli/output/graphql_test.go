package output

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/ye-kart/reqflow/internal/core/graphql"
)

func TestFormatGraphQLResponse_DataOnly(t *testing.T) {
	resp := graphql.GraphQLResponse{
		Data: json.RawMessage(`{"users": [{"id": "1", "name": "Alice"}]}`),
	}

	var buf bytes.Buffer
	err := FormatGraphQLResponse(&buf, resp, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := buf.String()

	if !strings.Contains(output, "Data:") {
		t.Error("expected output to contain 'Data:'")
	}

	if !strings.Contains(output, `"users"`) {
		t.Error("expected output to contain pretty-printed data")
	}

	if strings.Contains(output, "Errors:") {
		t.Error("expected output not to contain 'Errors:' when there are no errors")
	}
}

func TestFormatGraphQLResponse_ErrorsOnly(t *testing.T) {
	resp := graphql.GraphQLResponse{
		Errors: []graphql.GraphQLError{
			{
				Message:   "Not found",
				Locations: []graphql.Location{{Line: 1, Column: 3}},
				Path:      []interface{}{"user"},
			},
		},
	}

	var buf bytes.Buffer
	err := FormatGraphQLResponse(&buf, resp, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := buf.String()

	if !strings.Contains(output, "Errors:") {
		t.Error("expected output to contain 'Errors:'")
	}

	if !strings.Contains(output, "Not found") {
		t.Error("expected output to contain error message")
	}
}

func TestFormatGraphQLResponse_DataAndErrors(t *testing.T) {
	resp := graphql.GraphQLResponse{
		Data:   json.RawMessage(`{"user": null}`),
		Errors: []graphql.GraphQLError{{Message: "Unauthorized field"}},
	}

	var buf bytes.Buffer
	err := FormatGraphQLResponse(&buf, resp, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := buf.String()

	if !strings.Contains(output, "Data:") {
		t.Error("expected output to contain 'Data:'")
	}

	if !strings.Contains(output, "Errors:") {
		t.Error("expected output to contain 'Errors:'")
	}
}

func TestFormatGraphQLResponse_NoColor(t *testing.T) {
	resp := graphql.GraphQLResponse{
		Errors: []graphql.GraphQLError{{Message: "Bad query"}},
	}

	var buf bytes.Buffer
	err := FormatGraphQLResponse(&buf, resp, true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := buf.String()

	// Should not contain ANSI escape codes when noColor=true
	if strings.Contains(output, "\033[") {
		t.Error("expected no ANSI escape codes when noColor is true")
	}
}

func TestFormatGraphQLResponse_WithColor(t *testing.T) {
	resp := graphql.GraphQLResponse{
		Errors: []graphql.GraphQLError{{Message: "Bad query"}},
	}

	var buf bytes.Buffer
	err := FormatGraphQLResponse(&buf, resp, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := buf.String()

	// Should contain ANSI escape codes for red errors
	if !strings.Contains(output, "\033[") {
		t.Error("expected ANSI escape codes when noColor is false")
	}
}

func TestFormatGraphQLResponse_ErrorWithLocation(t *testing.T) {
	resp := graphql.GraphQLResponse{
		Errors: []graphql.GraphQLError{
			{
				Message:   "Syntax error",
				Locations: []graphql.Location{{Line: 2, Column: 5}},
			},
		},
	}

	var buf bytes.Buffer
	err := FormatGraphQLResponse(&buf, resp, true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := buf.String()

	if !strings.Contains(output, "line 2, column 5") {
		t.Errorf("expected location info in output, got: %s", output)
	}
}
