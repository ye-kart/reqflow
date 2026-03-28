package graphql

import (
	"encoding/json"
	"testing"
)

func TestBuildGraphQLBody_QueryOnly(t *testing.T) {
	req := GraphQLRequest{
		Query: "{ users { id name } }",
	}

	body, err := BuildGraphQLBody(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}

	if result["query"] != "{ users { id name } }" {
		t.Errorf("expected query '{ users { id name } }', got %v", result["query"])
	}

	if _, ok := result["variables"]; ok {
		t.Error("expected no variables key when Variables is nil")
	}

	if _, ok := result["operationName"]; ok {
		t.Error("expected no operationName key when OperationName is empty")
	}
}

func TestBuildGraphQLBody_WithVariables(t *testing.T) {
	req := GraphQLRequest{
		Query:     "query GetUser($id: ID!) { user(id: $id) { name } }",
		Variables: map[string]interface{}{"id": "123"},
	}

	body, err := BuildGraphQLBody(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}

	vars, ok := result["variables"].(map[string]interface{})
	if !ok {
		t.Fatal("expected variables to be an object")
	}

	if vars["id"] != "123" {
		t.Errorf("expected variable id=123, got %v", vars["id"])
	}
}

func TestBuildGraphQLBody_WithOperationName(t *testing.T) {
	req := GraphQLRequest{
		Query:         "query GetUser { user { name } }",
		OperationName: "GetUser",
	}

	body, err := BuildGraphQLBody(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}

	if result["operationName"] != "GetUser" {
		t.Errorf("expected operationName 'GetUser', got %v", result["operationName"])
	}
}

func TestBuildGraphQLBody_FullRequest(t *testing.T) {
	req := GraphQLRequest{
		Query:         "query GetUsers($limit: Int) { users(limit: $limit) { id name } }",
		Variables:     map[string]interface{}{"limit": float64(10)},
		OperationName: "GetUsers",
	}

	body, err := BuildGraphQLBody(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}

	if result["query"] != req.Query {
		t.Errorf("query mismatch")
	}
	if result["operationName"] != "GetUsers" {
		t.Errorf("operationName mismatch")
	}

	vars := result["variables"].(map[string]interface{})
	if vars["limit"] != float64(10) {
		t.Errorf("expected limit=10, got %v", vars["limit"])
	}
}

func TestBuildGraphQLBody_EmptyQuery(t *testing.T) {
	req := GraphQLRequest{
		Query: "",
	}

	_, err := BuildGraphQLBody(req)
	if err == nil {
		t.Fatal("expected error for empty query")
	}
}
