package codegen

import (
	"fmt"
	"strings"

	"github.com/ye-kart/reqflow/internal/domain"
)

// GenerateJava generates Java HttpClient code for the given HTTP request.
func GenerateJava(req domain.HTTPRequest) (string, error) {
	var sb strings.Builder

	sb.WriteString("HttpClient client = HttpClient.newHttpClient();\n")
	sb.WriteString("HttpRequest request = HttpRequest.newBuilder()\n")
	sb.WriteString(fmt.Sprintf("    .uri(URI.create(%q))\n", req.URL))

	for _, h := range req.Headers {
		sb.WriteString(fmt.Sprintf("    .header(%q, %q)\n", h.Key, h.Value))
	}

	// Method and body.
	method := string(req.Method)
	switch method {
	case "GET":
		sb.WriteString("    .GET()\n")
	case "DELETE":
		sb.WriteString("    .DELETE()\n")
	case "POST", "PUT", "PATCH":
		if len(req.Body) > 0 {
			sb.WriteString(fmt.Sprintf("    .%s(HttpRequest.BodyPublishers.ofString(\"%s\"))\n", method, escapeJavaString(string(req.Body))))
		} else {
			sb.WriteString(fmt.Sprintf("    .%s(HttpRequest.BodyPublishers.noBody())\n", method))
		}
	default:
		sb.WriteString(fmt.Sprintf("    .method(%q, HttpRequest.BodyPublishers.noBody())\n", method))
	}

	sb.WriteString("    .build();\n")
	sb.WriteString("HttpResponse<String> response = client.send(request, HttpResponse.BodyHandlers.ofString());\n")
	sb.WriteString("System.out.println(response.body());\n")

	return sb.String(), nil
}

// escapeJavaString escapes a string for use inside Java double quotes.
// It escapes backslashes and double quotes only.
func escapeJavaString(s string) string {
	s = strings.ReplaceAll(s, `\`, `\\`)
	s = strings.ReplaceAll(s, `"`, `\"`)
	return s
}
