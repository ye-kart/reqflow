package codegen

import (
	"fmt"
	"strings"

	"github.com/ye-kart/reqflow/internal/domain"
)

// GeneratePHP generates PHP cURL code for the given HTTP request.
func GeneratePHP(req domain.HTTPRequest) (string, error) {
	var sb strings.Builder

	sb.WriteString("<?php\n\n")
	sb.WriteString("$ch = curl_init();\n\n")
	sb.WriteString(fmt.Sprintf("curl_setopt($ch, CURLOPT_URL, %q);\n", req.URL))
	sb.WriteString("curl_setopt($ch, CURLOPT_RETURNTRANSFER, true);\n")

	method := string(req.Method)
	switch method {
	case "POST":
		sb.WriteString("curl_setopt($ch, CURLOPT_POST, true);\n")
	case "GET":
		// GET is the default, no need to set.
	default:
		sb.WriteString(fmt.Sprintf("curl_setopt($ch, CURLOPT_CUSTOMREQUEST, %q);\n", method))
	}

	if len(req.Headers) > 0 {
		sb.WriteString("curl_setopt($ch, CURLOPT_HTTPHEADER, [\n")
		for _, h := range req.Headers {
			sb.WriteString(fmt.Sprintf("    %q,\n", fmt.Sprintf("%s: %s", h.Key, h.Value)))
		}
		sb.WriteString("]);\n")
	}

	if len(req.Body) > 0 {
		sb.WriteString(fmt.Sprintf("curl_setopt($ch, CURLOPT_POSTFIELDS, '%s');\n", string(req.Body)))
	}

	sb.WriteString("\n$response = curl_exec($ch);\n")
	sb.WriteString("curl_close($ch);\n\n")
	sb.WriteString("echo $response;\n")

	return sb.String(), nil
}
