package commands

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/ye-kart/reqflow/internal/platform/plugins"
)

func newPluginCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "plugin",
		Short: "Manage reqflow plugins",
	}

	cmd.AddCommand(newPluginListCommand())
	cmd.AddCommand(newPluginInfoCommand())

	return cmd
}

func newPluginListCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List installed/registered plugins",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			infos := plugins.ListRegistered()
			if len(infos) == 0 {
				fmt.Fprintln(cmd.OutOrStdout(), "No plugins registered.")
				return nil
			}

			for _, info := range infos {
				types := strings.Join(info.Types, ", ")
				fmt.Fprintf(cmd.OutOrStdout(), "  %s  v%s  [%s]  (%s)\n",
					info.Name, info.Version, types, info.Source)
			}
			return nil
		},
	}
}

func newPluginInfoCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "info <name>",
		Short: "Show details about a specific plugin",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]
			info, ok := plugins.GetPluginInfo(name)
			if !ok {
				return fmt.Errorf("plugin %q not found", name)
			}

			w := cmd.OutOrStdout()
			fmt.Fprintf(w, "Name:    %s\n", info.Name)
			fmt.Fprintf(w, "Version: %s\n", info.Version)
			fmt.Fprintf(w, "Source:  %s\n", info.Source)
			fmt.Fprintf(w, "Types:   %s\n", strings.Join(info.Types, ", "))
			return nil
		},
	}
}
