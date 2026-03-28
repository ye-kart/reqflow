package commands

import (
	"context"
	"fmt"
	"io"
	"path/filepath"
	"time"

	"github.com/spf13/cobra"
	"github.com/ye-kart/reqflow/internal/app"
	"github.com/ye-kart/reqflow/internal/core/variable"
	"github.com/ye-kart/reqflow/internal/domain"
)

func newCollectionRunCommand(a *app.App) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "run <name>",
		Short: "Run all requests in a collection",
		Long:  "Execute all requests in a collection sequentially and display results.",
		Args:  cobra.ExactArgs(1),
		RunE:  makeCollectionRunE(a),
	}

	cmd.Flags().String("folder", "", "run only requests in the specified folder")
	cmd.Flags().Duration("delay", 0, "delay between requests (e.g. 1s, 500ms)")
	cmd.Flags().Bool("no-stop-on-failure", false, "continue executing on failure")

	return cmd
}

func makeCollectionRunE(a *app.App) func(cmd *cobra.Command, args []string) error {
	return func(cmd *cobra.Command, args []string) error {
		name := args[0]
		dir, _ := cmd.Flags().GetString("collection-dir")
		path := filepath.Join(dir, name+".yaml")

		col, err := a.Storage.ReadCollection(path)
		if err != nil {
			return fmt.Errorf("reading collection %q: %w", name, err)
		}

		if a.CollectionRunner == nil {
			return fmt.Errorf("collection runner not initialized")
		}

		folder, _ := cmd.Flags().GetString("folder")
		delay, _ := cmd.Flags().GetDuration("delay")
		noStopOnFailure, _ := cmd.Flags().GetBool("no-stop-on-failure")

		// Load environment variables if specified.
		vars := make(map[string]string)
		envName, _ := cmd.Flags().GetString("env")
		if envName != "" && a.Storage != nil {
			envDir, _ := cmd.Flags().GetString("env-dir")
			envPath := filepath.Join(envDir, envName+".yaml")
			env, err := a.Storage.ReadEnvironment(envPath)
			if err != nil {
				return fmt.Errorf("loading environment %q: %w", envName, err)
			}
			vars = variable.Resolve(env.Variables)
		}

		opts := domain.CollectionRunOptions{
			Sequential:    true,
			StopOnFailure: !noStopOnFailure,
			FolderName:    folder,
			Delay:         delay,
			Environment:   envName,
			Vars:          vars,
		}

		timeout, _ := cmd.Flags().GetDuration("timeout")
		ctx, cancel := context.WithTimeout(context.Background(), resolveTimeout(timeout))
		defer cancel()

		result, err := a.CollectionRunner.RunCollection(ctx, col, opts)
		if err != nil {
			return fmt.Errorf("running collection: %w", err)
		}

		w := cmd.OutOrStdout()
		noColor, _ := cmd.Flags().GetBool("no-color")
		return formatCollectionRunResult(w, result, noColor)
	}
}

func formatCollectionRunResult(w io.Writer, result domain.CollectionRunResult, noColor bool) error {
	green := "\033[32m"
	red := "\033[31m"
	bold := "\033[1m"
	reset := "\033[0m"
	dim := "\033[2m"
	yellow := "\033[33m"

	if noColor {
		green = ""
		red = ""
		bold = ""
		reset = ""
		dim = ""
		yellow = ""
	}

	fmt.Fprintf(w, "%sCollection: %s%s\n\n", bold, result.CollectionName, reset)

	for i, rr := range result.Results {
		icon := green + "PASS" + reset
		if !rr.Passed {
			if rr.Error != nil {
				icon = red + "ERROR" + reset
			} else {
				icon = red + "FAIL" + reset
			}
		}

		folderInfo := ""
		if rr.FolderPath != "" {
			folderInfo = fmt.Sprintf(" %s(%s)%s", dim, rr.FolderPath, reset)
		}

		fmt.Fprintf(w, "  %d. [%s] %s%s %s(%s)%s\n",
			i+1, icon, rr.RequestName, folderInfo,
			dim, rr.Duration.Round(time.Millisecond), reset)

		if rr.Error != nil {
			fmt.Fprintf(w, "     %sError: %s%s\n", red, rr.Error, reset)
		} else if !rr.Passed {
			fmt.Fprintf(w, "     %sStatus: %d%s\n", red, rr.Response.StatusCode, reset)
		}
	}

	fmt.Fprintln(w)

	// Summary line
	parts := []string{}
	if result.Failed > 0 {
		parts = append(parts, fmt.Sprintf("%s%d failed%s", red, result.Failed, reset))
	}
	if result.Passed > 0 {
		parts = append(parts, fmt.Sprintf("%s%d passed%s", green, result.Passed, reset))
	}
	if result.Skipped > 0 {
		parts = append(parts, fmt.Sprintf("%s%d skipped%s", yellow, result.Skipped, reset))
	}

	summary := ""
	for i, p := range parts {
		if i > 0 {
			summary += ", "
		}
		summary += p
	}
	fmt.Fprintf(w, "%s %s(%s)%s\n", summary, dim, result.Duration.Round(time.Millisecond), reset)

	return nil
}
