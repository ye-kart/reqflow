package commands

import (
	"os"

	"github.com/spf13/cobra"
)

// newCompletionCommand creates the completion subcommand for shell completions.
func newCompletionCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "completion [bash|zsh|fish|powershell]",
		Short: "Generate shell completion script",
		Long: `Generate shell completion script for reqflow.

Install completions by adding the output to your shell profile:

  reqflow completion bash > /etc/bash_completion.d/reqflow
  reqflow completion zsh > "${fpath[1]}/_reqflow"
  reqflow completion fish > ~/.config/fish/completions/reqflow.fish
  reqflow completion powershell > reqflow.ps1`,
		DisableFlagsInUseLine: true,
		ValidArgs:             []string{"bash", "zsh", "fish", "powershell"},
		Args:                  cobra.MatchAll(cobra.ExactArgs(1), cobra.OnlyValidArgs),
		RunE: func(cmd *cobra.Command, args []string) error {
			switch args[0] {
			case "bash":
				return cmd.Root().GenBashCompletion(os.Stdout)
			case "zsh":
				return cmd.Root().GenZshCompletion(os.Stdout)
			case "fish":
				return cmd.Root().GenFishCompletion(os.Stdout, true)
			case "powershell":
				return cmd.Root().GenPowerShellCompletionWithDesc(os.Stdout)
			}
			return nil
		},
	}
	return cmd
}
