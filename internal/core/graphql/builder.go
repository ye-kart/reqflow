package graphql

import (
	"encoding/json"
	"fmt"
)

// GraphQLRequest represents a GraphQL request with query, variables, and operation name.
type GraphQLRequest struct {
	Query         string
	Variables     map[string]interface{}
	OperationName string
}

// BuildGraphQLBody serializes a GraphQLRequest into a JSON body suitable for
// sending as an HTTP POST to a GraphQL endpoint.
func BuildGraphQLBody(req GraphQLRequest) ([]byte, error) {
	if req.Query == "" {
		return nil, fmt.Errorf("graphql: query must not be empty")
	}

	payload := map[string]interface{}{
		"query": req.Query,
	}

	if req.Variables != nil {
		payload["variables"] = req.Variables
	}

	if req.OperationName != "" {
		payload["operationName"] = req.OperationName
	}

	return json.Marshal(payload)
}
