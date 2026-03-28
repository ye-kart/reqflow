package commands

import (
	"github.com/spf13/cobra"
	"github.com/ye-kart/reqflow/internal/adapters/tui"
	"github.com/ye-kart/reqflow/internal/app"
)

// newTUICommand creates the "tui" subcommand that launches interactive mode.
func newTUICommand(a *app.App) *cobra.Command {
	return &cobra.Command{
		Use:   "tui",
		Short: "Launch interactive TUI mode",
		Long:  "Start an interactive terminal UI for building and sending HTTP requests.",
		RunE: func(cmd *cobra.Command, args []string) error {
			tuiApp := tui.New(a.HTTPClient(), a.Storage)
			return tuiApp.Run()
		},
	}
}
