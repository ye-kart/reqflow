package views

import (
	"context"
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/ye-kart/reqflow/internal/adapters/tui/styles"
	"github.com/ye-kart/reqflow/internal/domain"
	"github.com/ye-kart/reqflow/internal/ports/driven"
)

// Tab indices.
const (
	tabRequest  = 0
	tabResponse = 1
	tabHistory  = 2
	tabCount    = 3
)

// historyEntry records a past request and its response.
type historyEntry struct {
	method string
	url    string
	status int
}

// MainModel is the top-level Bubble Tea model with tab navigation.
type MainModel struct {
	activeTab   int
	request     RequestModel
	response    ResponseModel
	history     []historyEntry
	httpClient  driven.HTTPClient
	lastSentReq *domain.HTTPRequest
	width       int
	height      int
}

// NewMainModel creates the root TUI model.
func NewMainModel(hc driven.HTTPClient) MainModel {
	return MainModel{
		activeTab:  tabRequest,
		request:    NewRequestModel(),
		response:   NewResponseModel(),
		httpClient: hc,
	}
}

// Init satisfies tea.Model.
func (m MainModel) Init() tea.Cmd {
	return nil
}

// Update handles all input messages.
func (m MainModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyCtrlC:
			return m, tea.Quit
		case tea.KeyTab:
			// When on Request tab, delegate Tab to form navigation.
			// Otherwise, switch tabs.
			if m.activeTab == tabRequest {
				newReq, cmd := m.request.Update(msg)
				m.request = newReq.(RequestModel)
				return m, cmd
			}
			m.activeTab = (m.activeTab + 1) % tabCount
			return m, nil
		case tea.KeyShiftTab:
			if m.activeTab == tabRequest {
				newReq, cmd := m.request.Update(msg)
				m.request = newReq.(RequestModel)
				return m, cmd
			}
			m.activeTab = (m.activeTab - 1 + tabCount) % tabCount
			return m, nil
		case tea.KeyRight:
			if msg.Alt {
				m.activeTab = (m.activeTab + 1) % tabCount
				return m, nil
			}
		case tea.KeyLeft:
			if msg.Alt {
				m.activeTab = (m.activeTab - 1 + tabCount) % tabCount
				return m, nil
			}
		}

		// Also handle number keys for quick tab switching.
		if msg.Type == tea.KeyRunes && len(msg.Runes) == 1 {
			switch msg.Runes[0] {
			case '1':
				if m.activeTab != tabRequest {
					m.activeTab = tabRequest
					return m, nil
				}
			case '2':
				if m.activeTab != tabResponse {
					m.activeTab = tabResponse
					return m, nil
				}
			case '3':
				if m.activeTab != tabHistory {
					m.activeTab = tabHistory
					return m, nil
				}
			}
		}

		// Delegate to active tab's update only for the request tab.
		if m.activeTab == tabRequest {
			newReq, cmd := m.request.Update(msg)
			m.request = newReq.(RequestModel)
			return m, cmd
		}

	case SendRequestMsg:
		m.response.SetLoading(true)
		m.activeTab = tabResponse
		req := msg.Request
		m.lastSentReq = &req
		return m, m.sendRequest(msg.Request)

	case ResponseReceivedMsg:
		if msg.Err != nil {
			m.response.SetError(msg.Err.Error())
		} else {
			m.response.SetResponse(&msg.Response)
			entry := historyEntry{
				status: msg.Response.StatusCode,
			}
			if m.lastSentReq != nil {
				entry.method = string(m.lastSentReq.Method)
				entry.url = m.lastSentReq.URL
			}
			m.history = append(m.history, entry)
		}
		return m, nil
	}

	return m, nil
}

// sendRequest performs the HTTP request asynchronously.
func (m MainModel) sendRequest(req domain.HTTPRequest) tea.Cmd {
	hc := m.httpClient
	return func() tea.Msg {
		if hc == nil {
			return ResponseReceivedMsg{Err: fmt.Errorf("no HTTP client configured")}
		}
		resp, err := hc.Do(context.Background(), req)
		return ResponseReceivedMsg{Response: resp, Err: err}
	}
}

// View renders the entire TUI.
func (m MainModel) View() string {
	var b strings.Builder

	// Tab bar.
	b.WriteString(m.renderTabBar())
	b.WriteString("\n\n")

	// Active tab content.
	switch m.activeTab {
	case tabRequest:
		b.WriteString(m.request.View())
	case tabResponse:
		b.WriteString(m.response.View())
	case tabHistory:
		b.WriteString(m.renderHistory())
	}

	b.WriteString("\n")

	// Status bar.
	b.WriteString(m.renderStatusBar())

	return b.String()
}

// renderTabBar renders the tab navigation.
func (m MainModel) renderTabBar() string {
	tabs := []string{"Request", "Response", "History"}
	var rendered []string

	for i, name := range tabs {
		if i == m.activeTab {
			rendered = append(rendered, styles.ActiveTab().Render(name))
		} else {
			rendered = append(rendered, styles.InactiveTab().Render(name))
		}
	}

	return lipgloss.JoinHorizontal(lipgloss.Top, rendered...)
}

// renderHistory renders the history tab.
func (m MainModel) renderHistory() string {
	if len(m.history) == 0 {
		return "  No requests sent yet."
	}

	var b strings.Builder
	for i, entry := range m.history {
		b.WriteString(fmt.Sprintf("  %d. %s %s → %d\n", i+1, entry.method, entry.url, entry.status))
	}
	return b.String()
}

// renderStatusBar renders the bottom status bar with key hints.
func (m MainModel) renderStatusBar() string {
	hints := "Tab: navigate • 1/2/3: switch tabs • Enter: send • Ctrl+C: quit"
	return styles.StatusBar().Render(hints)
}
