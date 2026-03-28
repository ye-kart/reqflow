package exporter

import (
	"fmt"
	"html"
	"strings"

	"github.com/ye-kart/reqflow/internal/domain"
)

const htmlCSS = `
body {
    font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif;
    margin: 0;
    padding: 0;
    display: flex;
    color: #333;
    line-height: 1.6;
}
nav {
    width: 260px;
    min-height: 100vh;
    background: #f5f5f5;
    border-right: 1px solid #ddd;
    padding: 20px;
    position: fixed;
    overflow-y: auto;
}
nav h2 {
    font-size: 1rem;
    margin: 16px 0 8px;
    color: #555;
}
nav a {
    display: block;
    padding: 4px 0;
    color: #0366d6;
    text-decoration: none;
    font-size: 0.9rem;
}
nav a:hover {
    text-decoration: underline;
}
.content {
    margin-left: 300px;
    padding: 32px 48px;
    max-width: 900px;
}
h1 { border-bottom: 2px solid #eee; padding-bottom: 8px; }
h2 { margin-top: 32px; border-bottom: 1px solid #eee; padding-bottom: 6px; }
h3 { margin-top: 24px; }
table {
    border-collapse: collapse;
    margin: 8px 0 16px;
}
th, td {
    border: 1px solid #ddd;
    padding: 6px 12px;
    text-align: left;
}
th { background: #f5f5f5; }
pre {
    background: #f8f8f8;
    border: 1px solid #ddd;
    border-radius: 4px;
    padding: 12px;
    overflow-x: auto;
}
code { font-size: 0.9em; }
.method {
    display: inline-block;
    font-weight: bold;
    padding: 2px 8px;
    border-radius: 3px;
    font-size: 0.85em;
    margin-right: 6px;
}
.method-get { background: #e6f4ea; color: #137333; }
.method-post { background: #fce8e6; color: #c5221f; }
.method-put { background: #fef7e0; color: #ea8600; }
.method-patch { background: #e8f0fe; color: #1a73e8; }
.method-delete { background: #fce8e6; color: #c5221f; }
`

// ExportHTML generates API documentation as a styled HTML page with a
// navigation sidebar. It wraps the markdown content in a full HTML document.
func ExportHTML(c domain.Collection) ([]byte, error) {
	var sb strings.Builder

	sb.WriteString("<!DOCTYPE html>\n<html lang=\"en\">\n<head>\n")
	sb.WriteString(fmt.Sprintf("  <meta charset=\"UTF-8\">\n  <title>%s</title>\n", html.EscapeString(c.Name)))
	sb.WriteString("  <style>")
	sb.WriteString(htmlCSS)
	sb.WriteString("  </style>\n</head>\n<body>\n")

	// Navigation sidebar.
	sb.WriteString("<nav>\n")
	sb.WriteString(fmt.Sprintf("  <h1>%s</h1>\n", html.EscapeString(c.Name)))

	for _, r := range c.Requests {
		anchor := requestAnchor(r)
		sb.WriteString(fmt.Sprintf("  <a href=\"#%s\">%s %s</a>\n",
			anchor, r.Config.Method, html.EscapeString(r.Name)))
	}

	for _, f := range c.Folders {
		writeNavFolder(&sb, f)
	}
	sb.WriteString("</nav>\n")

	// Main content.
	sb.WriteString("<div class=\"content\">\n")
	sb.WriteString(fmt.Sprintf("  <h1>%s</h1>\n", html.EscapeString(c.Name)))
	if c.Description != "" {
		sb.WriteString(fmt.Sprintf("  <p>%s</p>\n", html.EscapeString(c.Description)))
	}

	for _, r := range c.Requests {
		writeHTMLRequest(&sb, r)
	}

	for _, f := range c.Folders {
		writeHTMLFolder(&sb, f)
	}

	sb.WriteString("</div>\n</body>\n</html>\n")

	return []byte(sb.String()), nil
}

func writeNavFolder(sb *strings.Builder, f domain.Folder) {
	sb.WriteString(fmt.Sprintf("  <h2>%s</h2>\n", html.EscapeString(f.Name)))
	for _, r := range f.Requests {
		anchor := requestAnchor(r)
		sb.WriteString(fmt.Sprintf("  <a href=\"#%s\">%s %s</a>\n",
			anchor, r.Config.Method, html.EscapeString(r.Name)))
	}
	for _, sub := range f.Folders {
		writeNavFolder(sb, sub)
	}
}

func writeHTMLFolder(sb *strings.Builder, f domain.Folder) {
	sb.WriteString(fmt.Sprintf("  <h2>Folder: %s</h2>\n", html.EscapeString(f.Name)))
	if f.Description != "" {
		sb.WriteString(fmt.Sprintf("  <p>%s</p>\n", html.EscapeString(f.Description)))
	}
	for _, r := range f.Requests {
		writeHTMLRequest(sb, r)
	}
	for _, sub := range f.Folders {
		writeHTMLFolder(sb, sub)
	}
}

func writeHTMLRequest(sb *strings.Builder, r domain.SavedRequest) {
	anchor := requestAnchor(r)
	path := extractPath(r.Config.URL)
	methodClass := "method-" + strings.ToLower(string(r.Config.Method))

	sb.WriteString(fmt.Sprintf("  <h3 id=\"%s\"><span class=\"method %s\">%s</span> %s</h3>\n",
		anchor, methodClass, r.Config.Method, html.EscapeString(path)))

	if r.Description != "" {
		sb.WriteString(fmt.Sprintf("  <p>%s</p>\n", html.EscapeString(r.Description)))
	}

	if len(r.Config.Headers) > 0 {
		sb.WriteString("  <h4>Headers</h4>\n")
		sb.WriteString("  <table>\n    <tr><th>Key</th><th>Value</th></tr>\n")
		for _, h := range r.Config.Headers {
			sb.WriteString(fmt.Sprintf("    <tr><td>%s</td><td>%s</td></tr>\n",
				html.EscapeString(h.Key), html.EscapeString(h.Value)))
		}
		sb.WriteString("  </table>\n")
	}

	if len(r.Config.Body) > 0 {
		sb.WriteString("  <h4>Body</h4>\n")
		sb.WriteString(fmt.Sprintf("  <pre><code>%s</code></pre>\n",
			html.EscapeString(string(r.Config.Body))))
	}
}

func requestAnchor(r domain.SavedRequest) string {
	name := strings.ToLower(r.Name)
	name = strings.ReplaceAll(name, " ", "-")
	return name
}
