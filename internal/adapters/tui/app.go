package tui

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/ye-kart/reqflow/internal/adapters/tui/views"
	"github.com/ye-kart/reqflow/internal/ports/driven"
)

// App is the TUI driving adapter. It owns the Bubble Tea program
// and bridges user interaction to the application core.
type App struct {
	httpClient driven.HTTPClient
	storage    driven.Storage
}

// New creates a new TUI App wired to the given driven adapters.
func New(hc driven.HTTPClient, s driven.Storage) *App {
	return &App{
		httpClient: hc,
		storage:    s,
	}
}

// Run starts the interactive TUI and blocks until the user quits.
func (a *App) Run() error {
	m := views.NewMainModel(a.httpClient)
	p := tea.NewProgram(m, tea.WithAltScreen())
	_, err := p.Run()
	return err
}
