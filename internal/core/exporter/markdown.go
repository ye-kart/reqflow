package exporter

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/ye-kart/reqflow/internal/domain"
)

// ExportMarkdown generates API documentation in Markdown format from a
// collection. Each folder becomes a section and each request becomes a
// subsection with method, path, headers table, and body example.
func ExportMarkdown(c domain.Collection) ([]byte, error) {
	var sb strings.Builder

	// Title.
	sb.WriteString(fmt.Sprintf("# %s\n\n", c.Name))

	if c.Description != "" {
		sb.WriteString(c.Description)
		sb.WriteString("\n\n")
	}

	// Top-level requests (not in any folder).
	for _, r := range c.Requests {
		writeRequest(&sb, r, 3)
	}

	// Folders.
	for _, f := range c.Folders {
		writeFolder(&sb, f, 2)
	}

	return []byte(sb.String()), nil
}

// writeFolder writes a folder section. depth controls the heading level.
func writeFolder(sb *strings.Builder, f domain.Folder, depth int) {
	heading := strings.Repeat("#", depth)
	sb.WriteString(fmt.Sprintf("%s Folder: %s\n\n", heading, f.Name))

	if f.Description != "" {
		sb.WriteString(f.Description)
		sb.WriteString("\n\n")
	}

	for _, r := range f.Requests {
		writeRequest(sb, r, depth+1)
	}

	// Nested folders.
	for _, sub := range f.Folders {
		writeFolder(sb, sub, depth+1)
	}
}

// writeRequest writes a single request section at the given heading depth.
func writeRequest(sb *strings.Builder, r domain.SavedRequest, depth int) {
	heading := strings.Repeat("#", depth)
	path := extractPath(r.Config.URL)

	sb.WriteString(fmt.Sprintf("%s %s %s\n\n", heading, r.Config.Method, path))

	if r.Description != "" {
		sb.WriteString(r.Description)
		sb.WriteString("\n\n")
	}

	sb.WriteString(fmt.Sprintf("**Name:** %s\n\n", r.Name))

	// Headers table.
	if len(r.Config.Headers) > 0 {
		sb.WriteString("**Headers:**\n\n")
		sb.WriteString("| Key | Value |\n")
		sb.WriteString("|-----|-------|\n")
		for _, h := range r.Config.Headers {
			sb.WriteString(fmt.Sprintf("| %s | %s |\n", h.Key, h.Value))
		}
		sb.WriteString("\n")
	}

	// Request body example.
	if len(r.Config.Body) > 0 {
		sb.WriteString("**Body:**\n\n")
		sb.WriteString("```json\n")
		sb.WriteString(string(r.Config.Body))
		sb.WriteString("\n```\n\n")
	}
}

// extractPath returns the path portion of a URL, or the full URL if parsing fails.
func extractPath(rawURL string) string {
	u, err := url.Parse(rawURL)
	if err != nil {
		return rawURL
	}
	return u.Path
}
