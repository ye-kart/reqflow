package views

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/ye-kart/reqflow/internal/adapters/tui/styles"
	"github.com/ye-kart/reqflow/internal/domain"
)

// SendRequestMsg signals that the user wants to send the current request.
type SendRequestMsg struct {
	Request domain.HTTPRequest
}

// Focus positions within the request form.
const (
	focusMethod = iota
	focusURL
	focusHeaders
	focusBody
	focusSend
	focusCount // sentinel
)

// Available methods for the dropdown.
var methods = []domain.HTTPMethod{
	domain.MethodGet,
	domain.MethodPost,
	domain.MethodPut,
	domain.MethodPatch,
	domain.MethodDelete,
}

// textInput is a simple text input model.
type textInput struct {
	value  string
	cursor int
}

func newTextInput() textInput {
	return textInput{}
}

func (t *textInput) SetValue(s string) {
	t.value = s
	t.cursor = len(s)
}

func (t *textInput) Value() string {
	return t.value
}

func (t *textInput) handleKey(msg tea.KeyMsg) {
	switch msg.Type {
	case tea.KeyRunes:
		left := t.value[:t.cursor]
		right := t.value[t.cursor:]
		t.value = left + string(msg.Runes) + right
		t.cursor += len(msg.Runes)
	case tea.KeyBackspace:
		if t.cursor > 0 {
			left := t.value[:t.cursor-1]
			right := t.value[t.cursor:]
			t.value = left + right
			t.cursor--
		}
	case tea.KeyDelete:
		if t.cursor < len(t.value) {
			left := t.value[:t.cursor]
			right := t.value[t.cursor+1:]
			t.value = left + right
		}
	case tea.KeyLeft:
		if t.cursor > 0 {
			t.cursor--
		}
	case tea.KeyRight:
		if t.cursor < len(t.value) {
			t.cursor++
		}
	case tea.KeyHome:
		t.cursor = 0
	case tea.KeyEnd:
		t.cursor = len(t.value)
	}
}

func (t textInput) view(focused bool) string {
	display := t.value
	if display == "" && !focused {
		display = ""
	}
	if focused {
		// Show cursor position.
		left := t.value[:t.cursor]
		right := t.value[t.cursor:]
		cursor := "│"
		display = left + cursor + right
	}
	return display
}

// headerEntry is a key-value pair for headers.
type headerEntry struct {
	key   string
	value string
}

// RequestModel is the request builder view.
type RequestModel struct {
	method    domain.HTTPMethod
	methodIdx int
	urlInput  textInput
	bodyInput textInput
	headers   []headerEntry
	focused   int
}

// NewRequestModel creates a new request builder with defaults.
func NewRequestModel() RequestModel {
	return RequestModel{
		method:    domain.MethodGet,
		methodIdx: 0,
		urlInput:  newTextInput(),
		bodyInput: newTextInput(),
		focused:   focusMethod,
	}
}

// URL returns the current URL value.
func (m RequestModel) URL() string {
	return m.urlInput.Value()
}

// AddHeader adds a header key-value pair.
func (m *RequestModel) AddHeader(key, value string) {
	m.headers = append(m.headers, headerEntry{key: key, value: value})
}

// RemoveHeader removes the header at index i.
func (m *RequestModel) RemoveHeader(i int) {
	if i < 0 || i >= len(m.headers) {
		return
	}
	m.headers = append(m.headers[:i], m.headers[i+1:]...)
}

// BuildRequest constructs an HTTPRequest from the form state.
func (m RequestModel) BuildRequest() domain.HTTPRequest {
	var headers []domain.Header
	for _, h := range m.headers {
		headers = append(headers, domain.Header{Key: h.key, Value: h.value})
	}

	req := domain.HTTPRequest{
		Method:  m.method,
		URL:     m.urlInput.Value(),
		Headers: headers,
	}

	body := m.bodyInput.Value()
	if body != "" {
		req.Body = []byte(body)
		// Default content type for requests with body.
		if m.method == domain.MethodPost || m.method == domain.MethodPut || m.method == domain.MethodPatch {
			req.ContentType = "application/json"
		}
	}

	return req
}

// Init satisfies tea.Model.
func (m RequestModel) Init() tea.Cmd {
	return nil
}

// updateMethodSelector handles key events for the method dropdown.
func (m RequestModel) updateMethodSelector(msg tea.KeyMsg) tea.Model {
	switch msg.Type {
	case tea.KeyDown:
		m.methodIdx = (m.methodIdx + 1) % len(methods)
		m.method = methods[m.methodIdx]
	case tea.KeyUp:
		m.methodIdx = (m.methodIdx - 1 + len(methods)) % len(methods)
		m.method = methods[m.methodIdx]
	}
	return m
}

// Update processes key events.
func (m RequestModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	keyMsg, ok := msg.(tea.KeyMsg)
	if !ok {
		return m, nil
	}

	// Tab navigates between fields.
	if keyMsg.Type == tea.KeyTab {
		m.focused = (m.focused + 1) % focusCount
		return m, nil
	}
	if keyMsg.Type == tea.KeyShiftTab {
		m.focused = (m.focused - 1 + focusCount) % focusCount
		return m, nil
	}

	switch m.focused {
	case focusMethod:
		return m.updateMethodSelector(keyMsg), nil
	case focusURL:
		m.urlInput.handleKey(keyMsg)
		return m, nil
	case focusBody:
		m.bodyInput.handleKey(keyMsg)
		return m, nil
	case focusSend:
		if keyMsg.Type == tea.KeyEnter {
			req := m.BuildRequest()
			return m, func() tea.Msg {
				return SendRequestMsg{Request: req}
			}
		}
	}

	return m, nil
}

// View renders the request builder form.
func (m RequestModel) View() string {
	var b strings.Builder

	// Method selector.
	methodLabel := styles.Label().Render("  Method")
	methodStyle := styles.MethodColor(string(m.method))
	methodDisplay := methodStyle.Render(string(m.method))
	if m.focused == focusMethod {
		methodDisplay += "  ↑↓"
	}
	if m.focused == focusMethod {
		b.WriteString(styles.Focused().Render(methodLabel + "  " + methodDisplay))
	} else {
		b.WriteString(styles.Blurred().Render(methodLabel + "  " + methodDisplay))
	}
	b.WriteString("\n\n")

	// URL input.
	urlLabel := styles.Label().Render("  URL")
	urlValue := m.urlInput.view(m.focused == focusURL)
	if urlValue == "" {
		urlValue = "https://..."
	}
	urlLine := urlLabel + "    " + urlValue
	if m.focused == focusURL {
		b.WriteString(styles.Focused().Render(urlLine))
	} else {
		b.WriteString(styles.Blurred().Render(urlLine))
	}
	b.WriteString("\n\n")

	// Headers.
	headersLabel := styles.Label().Render("  Headers")
	var headersDisplay string
	if len(m.headers) == 0 {
		headersDisplay = "  (none)"
	} else {
		var lines []string
		for _, h := range m.headers {
			lines = append(lines, fmt.Sprintf("    %s: %s", h.key, h.value))
		}
		headersDisplay = strings.Join(lines, "\n")
	}
	headerBlock := headersLabel + "\n" + headersDisplay
	if m.focused == focusHeaders {
		b.WriteString(styles.Focused().Render(headerBlock))
	} else {
		b.WriteString(styles.Blurred().Render(headerBlock))
	}
	b.WriteString("\n\n")

	// Body input (visible for methods that support body).
	if m.method == domain.MethodPost || m.method == domain.MethodPut || m.method == domain.MethodPatch {
		bodyLabel := styles.Label().Render("  Body")
		bodyValue := m.bodyInput.view(m.focused == focusBody)
		if bodyValue == "" {
			bodyValue = "{}"
		}
		bodyLine := bodyLabel + "   " + bodyValue
		if m.focused == focusBody {
			b.WriteString(styles.Focused().Render(bodyLine))
		} else {
			b.WriteString(styles.Blurred().Render(bodyLine))
		}
		b.WriteString("\n\n")
	}

	// Send button.
	sendLabel := "[ Send Request ]"
	if m.focused == focusSend {
		b.WriteString(styles.Focused().Render("  " + styles.MethodColor("POST").Render(sendLabel)))
	} else {
		b.WriteString(styles.Blurred().Render("  " + sendLabel))
	}
	b.WriteString("\n")

	return b.String()
}
