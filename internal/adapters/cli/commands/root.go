package commands

import (
	"time"

	"github.com/spf13/cobra"
	"github.com/ye-kart/reqflow/internal/app"
	"github.com/ye-kart/reqflow/internal/platform/config"
)

// RootOption is a functional option for configuring the root command.
type RootOption func(*rootOptions)

type rootOptions struct {
	configOpts []config.LoadOption
}

// WithGlobalConfigDir sets the global config directory for testing.
func WithGlobalConfigDir(dir string) RootOption {
	return func(o *rootOptions) {
		o.configOpts = append(o.configOpts, config.WithGlobalDir(dir))
	}
}

// WithProjectConfigDir sets the project config directory for testing.
func WithProjectConfigDir(dir string) RootOption {
	return func(o *rootOptions) {
		o.configOpts = append(o.configOpts, config.WithProjectDir(dir))
	}
}

// NewRootCommand creates the root cobra command with all subcommands.
func NewRootCommand(a *app.App, opts ...RootOption) *cobra.Command {
	o := &rootOptions{}
	for _, opt := range opts {
		opt(o)
	}

	root := &cobra.Command{
		Use:   "reqflow",
		Short: "A developer-friendly HTTP client for the terminal",
		Long:  "reqflow is a CLI tool for sending HTTP requests with variable substitution, authentication, and environment management.",
		SilenceUsage:  true,
		SilenceErrors: true,
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load(o.configOpts...)
			if err != nil {
				return err
			}

			// Apply config defaults where flags were not explicitly set.
			if !cmd.Flags().Changed("output") {
				cmd.Flags().Set("output", string(cfg.Output))
			}
			if !cmd.Flags().Changed("no-color") {
				if cfg.NoColor {
					cmd.Flags().Set("no-color", "true")
				}
			}
			if !cmd.Flags().Changed("timeout") {
				cmd.Flags().Set("timeout", cfg.Timeout.String())
			}

			// Store config and config options in the command context for subcommands.
			cmd.SetContext(withConfig(cmd.Context(), cfg))
			cmd.SetContext(withConfigOpts(cmd.Context(), o.configOpts))

			return nil
		},
	}

	// Global persistent flags.
	root.PersistentFlags().StringP("output", "o", "pretty", "output format (pretty, json, raw, minimal)")
	root.PersistentFlags().Bool("no-color", false, "disable colors")
	root.PersistentFlags().DurationP("timeout", "t", 30*time.Second, "request timeout")
	root.PersistentFlags().BoolP("verbose", "v", false, "show request details")
	root.PersistentFlags().StringSliceP("header", "H", nil, `add headers (format "Key: Value")`)
	root.PersistentFlags().StringSliceP("query", "q", nil, `add query params (format "key=value")`)
	root.PersistentFlags().StringP("env", "e", "", "environment to load variables from")
	root.PersistentFlags().String("env-dir", defaultEnvDir(), "directory containing environment files")
	root.PersistentFlags().Bool("dry-run", false, "show request that would be sent without executing it")
	root.PersistentFlags().Bool("trace", false, "show detailed timing breakdown")

	// Register HTTP method subcommands.
	root.AddCommand(newGetCommand(a))
	root.AddCommand(newPostCommand(a))
	root.AddCommand(newPutCommand(a))
	root.AddCommand(newPatchCommand(a))
	root.AddCommand(newDeleteCommand(a))

	// Register completion command and flag completions.
	root.AddCommand(newCompletionCommand())
	registerFlagCompletions(root)

	// Register import/export subcommands.
	root.AddCommand(newImportCommand(a))
	root.AddCommand(newExportCommand(a))

	// Register environment management subcommand.
	root.AddCommand(newEnvCommand(a))

	// Register config subcommands.
	root.AddCommand(newConfigCommand())

	// Register workflow run subcommand.
	root.AddCommand(newRunCommand(a))

	// Register collection management subcommand.
	root.AddCommand(newCollectionCommand(a))

	// Register cookie management subcommand.
	root.AddCommand(newCookieCommand(a))

	// Register mock server subcommand.
	root.AddCommand(newMockCommand(a))

	// Register monitor management subcommand.
	root.AddCommand(newMonitorCommand(a))

	return root
}
