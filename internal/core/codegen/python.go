package codegen

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/ye-kart/reqflow/internal/domain"
)

// GeneratePython generates Python requests library code for the given HTTP request.
func GeneratePython(req domain.HTTPRequest) (string, error) {
	var sb strings.Builder

	sb.WriteString("import requests\n\n")

	method := strings.ToLower(string(req.Method))

	sb.WriteString("response = requests.")
	sb.WriteString(method)
	sb.WriteString("(\n")
	sb.WriteString(fmt.Sprintf("    %q,\n", req.URL))

	// Headers.
	if len(req.Headers) > 0 {
		sb.WriteString("    headers={")
		for i, h := range req.Headers {
			if i > 0 {
				sb.WriteString(", ")
			}
			sb.WriteString(fmt.Sprintf("%q: %q", h.Key, h.Value))
		}
		sb.WriteString("},\n")
	}

	// Body.
	if len(req.Body) > 0 {
		// Try to parse as JSON for nicer output.
		var jsonObj interface{}
		if err := json.Unmarshal(req.Body, &jsonObj); err == nil {
			sb.WriteString(fmt.Sprintf("    json=%s,\n", string(req.Body)))
		} else {
			sb.WriteString(fmt.Sprintf("    data=%q,\n", string(req.Body)))
		}
	}

	sb.WriteString(")\n")
	sb.WriteString("print(response.json())\n")

	return sb.String(), nil
}
