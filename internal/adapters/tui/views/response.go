package views

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/ye-kart/reqflow/internal/adapters/tui/styles"
	"github.com/ye-kart/reqflow/internal/domain"
)

// ResponseReceivedMsg is sent when an HTTP response arrives.
type ResponseReceivedMsg struct {
	Response domain.HTTPResponse
	Err      error
}

// ResponseModel displays an HTTP response.
type ResponseModel struct {
	response *domain.HTTPResponse
	loading  bool
	errMsg   string
}

// NewResponseModel creates a new empty response view.
func NewResponseModel() ResponseModel {
	return ResponseModel{}
}

// SetResponse stores the response for display.
func (m *ResponseModel) SetResponse(resp *domain.HTTPResponse) {
	m.response = resp
	m.loading = false
	m.errMsg = ""
}

// SetLoading shows the loading indicator.
func (m *ResponseModel) SetLoading(loading bool) {
	m.loading = loading
	if loading {
		m.errMsg = ""
	}
}

// SetError shows an error message.
func (m *ResponseModel) SetError(msg string) {
	m.errMsg = msg
	m.loading = false
}

// Init satisfies tea.Model.
func (m ResponseModel) Init() tea.Cmd {
	return nil
}

// Update satisfies tea.Model.
func (m ResponseModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	return m, nil
}

// View renders the response view.
func (m ResponseModel) View() string {
	if m.loading {
		return "  Sending request..."
	}
	if m.errMsg != "" {
		return fmt.Sprintf("  Error: %s", m.errMsg)
	}
	if m.response == nil {
		return "  No response yet. Send a request first."
	}

	var b strings.Builder

	// Status line.
	statusStyle := styles.StatusCode(m.response.StatusCode)
	b.WriteString("  ")
	b.WriteString(statusStyle.Render(m.response.Status))
	b.WriteString(fmt.Sprintf("  (%dms)", m.response.Duration.Milliseconds()))
	b.WriteString("\n\n")

	// Headers.
	if len(m.response.Headers) > 0 {
		b.WriteString(styles.Label().Render("  Headers"))
		b.WriteString("\n")
		for _, h := range m.response.Headers {
			b.WriteString(fmt.Sprintf("    %s: %s\n", h.Key, h.Value))
		}
		b.WriteString("\n")
	}

	// Body.
	b.WriteString(styles.Label().Render("  Body"))
	b.WriteString("\n")
	body := string(m.response.Body)
	if prettyJSON, err := prettyPrintJSON(m.response.Body); err == nil {
		body = prettyJSON
	}
	// Indent body lines.
	for _, line := range strings.Split(body, "\n") {
		b.WriteString("    ")
		b.WriteString(line)
		b.WriteString("\n")
	}

	return b.String()
}

// prettyPrintJSON attempts to format JSON with indentation.
func prettyPrintJSON(data []byte) (string, error) {
	var buf bytes.Buffer
	if err := json.Indent(&buf, data, "", "  "); err != nil {
		return "", err
	}
	return buf.String(), nil
}
