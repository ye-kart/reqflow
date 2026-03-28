package codegen

import (
	"fmt"
	"strings"

	"github.com/ye-kart/reqflow/internal/domain"
)

// GenerateCurl generates a curl command string from an HTTPRequest.
func GenerateCurl(req domain.HTTPRequest) (string, error) {
	var parts []string

	// Method (omit for GET since it's the default).
	if req.Method != domain.MethodGet {
		parts = append(parts, fmt.Sprintf("-X %s", req.Method))
	}

	// Headers.
	for _, h := range req.Headers {
		parts = append(parts, fmt.Sprintf("-H '%s: %s'", h.Key, h.Value))
	}

	// Body.
	if len(req.Body) > 0 {
		parts = append(parts, fmt.Sprintf("-d '%s'", string(req.Body)))
	}

	// Build the final command.
	if len(parts) == 0 {
		return fmt.Sprintf("curl '%s'", req.URL), nil
	}

	var sb strings.Builder
	sb.WriteString("curl")

	for _, p := range parts {
		sb.WriteString(" \\\n  ")
		sb.WriteString(p)
	}
	sb.WriteString(" \\\n  '")
	sb.WriteString(req.URL)
	sb.WriteString("'")

	return sb.String(), nil
}
