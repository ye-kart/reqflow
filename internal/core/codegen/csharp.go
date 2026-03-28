package codegen

import (
	"fmt"
	"strings"

	"github.com/ye-kart/reqflow/internal/domain"
)

// csharpMethodCall maps HTTP methods to C# HttpClient method names.
var csharpMethodCall = map[domain.HTTPMethod]string{
	domain.MethodGet:    "GetAsync",
	domain.MethodPost:   "PostAsync",
	domain.MethodPut:    "PutAsync",
	domain.MethodPatch:  "PatchAsync",
	domain.MethodDelete: "DeleteAsync",
}

// GenerateCSharp generates C# HttpClient code for the given HTTP request.
func GenerateCSharp(req domain.HTTPRequest) (string, error) {
	var sb strings.Builder

	sb.WriteString("using var client = new HttpClient();\n\n")

	// Add headers.
	for _, h := range req.Headers {
		sb.WriteString(fmt.Sprintf("client.DefaultRequestHeaders.Add(%q, %q);\n", h.Key, h.Value))
	}
	if len(req.Headers) > 0 {
		sb.WriteString("\n")
	}

	method := string(req.Method)
	methodCall := csharpMethodCall[req.Method]
	if methodCall == "" {
		methodCall = "SendAsync"
	}

	hasBody := method == "POST" || method == "PUT" || method == "PATCH"

	if hasBody && len(req.Body) > 0 {
		escaped := strings.ReplaceAll(string(req.Body), `\`, `\\`)
		escaped = strings.ReplaceAll(escaped, `"`, `\"`)
		sb.WriteString(fmt.Sprintf("var content = new StringContent(\"%s\", System.Text.Encoding.UTF8, \"application/json\");\n", escaped))
		sb.WriteString(fmt.Sprintf("var response = await client.%s(%q, content);\n", methodCall, req.URL))
	} else {
		sb.WriteString(fmt.Sprintf("var response = await client.%s(%q);\n", methodCall, req.URL))
	}

	sb.WriteString("var body = await response.Content.ReadAsStringAsync();\n")
	sb.WriteString("Console.WriteLine(body);\n")

	return sb.String(), nil
}
