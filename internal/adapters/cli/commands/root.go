package commands

import (
	"time"

	"github.com/spf13/cobra"
	"github.com/ye-kart/reqflow/internal/app"
)

// NewRootCommand creates the root cobra command with all subcommands.
func NewRootCommand(a *app.App) *cobra.Command {
	root := &cobra.Command{
		Use:   "reqflow",
		Short: "A developer-friendly HTTP client for the terminal",
		Long:  "reqflow is a CLI tool for sending HTTP requests with variable substitution, authentication, and environment management.",
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	// Global persistent flags.
	root.PersistentFlags().StringP("output", "o", "pretty", "output format (pretty, json, raw, minimal)")
	root.PersistentFlags().Bool("no-color", false, "disable colors")
	root.PersistentFlags().DurationP("timeout", "t", 30*time.Second, "request timeout")
	root.PersistentFlags().BoolP("verbose", "v", false, "show request details")
	root.PersistentFlags().StringSliceP("header", "H", nil, `add headers (format "Key: Value")`)
	root.PersistentFlags().StringSliceP("query", "q", nil, `add query params (format "key=value")`)

	// Register HTTP method subcommands.
	root.AddCommand(newGetCommand(a))
	root.AddCommand(newPostCommand(a))
	root.AddCommand(newPutCommand(a))
	root.AddCommand(newPatchCommand(a))
	root.AddCommand(newDeleteCommand(a))

	// Shell completion.
	root.AddCommand(newCompletionCommand())

	return root
}
