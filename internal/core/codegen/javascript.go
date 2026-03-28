package codegen

import (
	"fmt"
	"strings"

	"github.com/ye-kart/reqflow/internal/domain"
)

// GenerateJavaScript generates JavaScript fetch API code for the given HTTP request.
func GenerateJavaScript(req domain.HTTPRequest) (string, error) {
	var sb strings.Builder

	method := string(req.Method)
	hasOptions := len(req.Headers) > 0 || len(req.Body) > 0 || method != "GET"

	sb.WriteString(fmt.Sprintf("const response = await fetch(%q", req.URL))

	if hasOptions {
		sb.WriteString(", {\n")
		sb.WriteString(fmt.Sprintf("  method: %q,\n", method))

		if len(req.Headers) > 0 {
			sb.WriteString("  headers: {")
			for i, h := range req.Headers {
				if i > 0 {
					sb.WriteString(", ")
				}
				sb.WriteString(fmt.Sprintf("%q: %q", h.Key, h.Value))
			}
			sb.WriteString("},\n")
		}

		if len(req.Body) > 0 {
			sb.WriteString(fmt.Sprintf("  body: JSON.stringify(%s),\n", string(req.Body)))
		}

		sb.WriteString("}")
	}

	sb.WriteString(");\n")
	sb.WriteString("const data = await response.json();\n")
	sb.WriteString("console.log(data);\n")

	return sb.String(), nil
}
