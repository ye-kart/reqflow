package graphql

import (
	"encoding/json"
	"fmt"
)

// GraphQLResponse represents a parsed GraphQL response with data and errors.
type GraphQLResponse struct {
	Data   json.RawMessage `json:"data"`
	Errors []GraphQLError  `json:"errors"`
}

// GraphQLError represents a single error in a GraphQL response.
type GraphQLError struct {
	Message   string        `json:"message"`
	Locations []Location    `json:"locations"`
	Path      []interface{} `json:"path"`
}

// Location represents a location in a GraphQL query where an error occurred.
type Location struct {
	Line   int `json:"line"`
	Column int `json:"column"`
}

// HasErrors reports whether the response contains any GraphQL errors.
func (r GraphQLResponse) HasErrors() bool {
	return len(r.Errors) > 0
}

// ParseGraphQLResponse parses a raw JSON body into a GraphQLResponse.
func ParseGraphQLResponse(body []byte) (GraphQLResponse, error) {
	if len(body) == 0 {
		return GraphQLResponse{}, fmt.Errorf("graphql: empty response body")
	}

	var resp GraphQLResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return GraphQLResponse{}, fmt.Errorf("graphql: invalid response JSON: %w", err)
	}

	return resp, nil
}
