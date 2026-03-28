package commands

import (
	"fmt"
	"text/tabwriter"

	"github.com/spf13/cobra"
	"github.com/ye-kart/reqflow/internal/app"
	"github.com/ye-kart/reqflow/internal/features/history"
)

func newHistoryCommand(a *app.App) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "history",
		Short: "View request history",
		Long:  "List, search, compare, and manage saved request/response history.",
		RunE: func(cmd *cobra.Command, args []string) error {
			if a.HistoryStore == nil {
				return fmt.Errorf("history store not initialized")
			}

			limit, _ := cmd.Flags().GetInt("limit")

			entries, err := a.HistoryStore.List(limit)
			if err != nil {
				return fmt.Errorf("listing history: %w", err)
			}

			if len(entries) == 0 {
				cmd.Println("No history entries.")
				return nil
			}

			return printHistoryTable(cmd, entries)
		},
	}

	cmd.Flags().Int("limit", 20, "max number of entries to show")

	cmd.AddCommand(newHistoryShowCommand(a))
	cmd.AddCommand(newHistorySearchCommand(a))
	cmd.AddCommand(newHistoryDiffCommand(a))
	cmd.AddCommand(newHistoryClearCommand(a))

	return cmd
}

func newHistoryShowCommand(a *app.App) *cobra.Command {
	return &cobra.Command{
		Use:   "show <id>",
		Short: "Show full request/response details",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if a.HistoryStore == nil {
				return fmt.Errorf("history store not initialized")
			}

			entry, err := a.HistoryStore.Get(args[0])
			if err != nil {
				return err
			}

			w := cmd.OutOrStdout()
			fmt.Fprintf(w, "ID:       %s\n", entry.ID)
			fmt.Fprintf(w, "Time:     %s\n", entry.Timestamp.Format("2006-01-02 15:04:05"))
			fmt.Fprintf(w, "Method:   %s\n", entry.Method)
			fmt.Fprintf(w, "URL:      %s\n", entry.URL)
			fmt.Fprintf(w, "Status:   %d\n", entry.Status)
			fmt.Fprintf(w, "Duration: %s\n", entry.Duration)

			fmt.Fprintln(w)
			fmt.Fprintln(w, "--- Request Headers ---")
			for _, h := range entry.Request.Headers {
				fmt.Fprintf(w, "  %s: %s\n", h.Key, h.Value)
			}
			if len(entry.Request.Body) > 0 {
				fmt.Fprintln(w)
				fmt.Fprintln(w, "--- Request Body ---")
				fmt.Fprintln(w, string(entry.Request.Body))
			}

			fmt.Fprintln(w)
			fmt.Fprintln(w, "--- Response Headers ---")
			for _, h := range entry.Response.Headers {
				fmt.Fprintf(w, "  %s: %s\n", h.Key, h.Value)
			}
			if len(entry.Response.Body) > 0 {
				fmt.Fprintln(w)
				fmt.Fprintln(w, "--- Response Body ---")
				fmt.Fprintln(w, string(entry.Response.Body))
			}

			return nil
		},
	}
}

func newHistorySearchCommand(a *app.App) *cobra.Command {
	return &cobra.Command{
		Use:   "search <query>",
		Short: "Search history by URL",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if a.HistoryStore == nil {
				return fmt.Errorf("history store not initialized")
			}

			entries, err := a.HistoryStore.Search(args[0])
			if err != nil {
				return fmt.Errorf("searching history: %w", err)
			}

			if len(entries) == 0 {
				cmd.Printf("No history entries matching %q.\n", args[0])
				return nil
			}

			return printHistoryTable(cmd, entries)
		},
	}
}

func newHistoryDiffCommand(a *app.App) *cobra.Command {
	return &cobra.Command{
		Use:   "diff <id1> <id2>",
		Short: "Compare two history entries",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			if a.HistoryStore == nil {
				return fmt.Errorf("history store not initialized")
			}

			entryA, err := a.HistoryStore.Get(args[0])
			if err != nil {
				return fmt.Errorf("getting entry %q: %w", args[0], err)
			}

			entryB, err := a.HistoryStore.Get(args[1])
			if err != nil {
				return fmt.Errorf("getting entry %q: %w", args[1], err)
			}

			diff := history.Compare(entryA, entryB)
			w := cmd.OutOrStdout()

			if diff.StatusChanged {
				fmt.Fprintf(w, "Status: %d -> %d\n", diff.OldStatus, diff.NewStatus)
			} else {
				fmt.Fprintf(w, "Status: %d (unchanged)\n", entryA.Response.StatusCode)
			}

			if len(diff.HeaderDiffs) > 0 {
				fmt.Fprintln(w)
				fmt.Fprintln(w, "Header changes:")
				for _, hd := range diff.HeaderDiffs {
					switch {
					case hd.Added:
						fmt.Fprintf(w, "  + %s: %s\n", hd.Key, hd.NewValue)
					case hd.Removed:
						fmt.Fprintf(w, "  - %s: %s\n", hd.Key, hd.OldValue)
					default:
						fmt.Fprintf(w, "  ~ %s: %s -> %s\n", hd.Key, hd.OldValue, hd.NewValue)
					}
				}
			}

			if diff.BodyDiff != "" {
				fmt.Fprintln(w)
				fmt.Fprintln(w, "Body diff:")
				fmt.Fprintln(w, diff.BodyDiff)
			}

			if !diff.StatusChanged && len(diff.HeaderDiffs) == 0 && diff.BodyDiff == "" {
				fmt.Fprintln(w, "No differences found.")
			}

			return nil
		},
	}
}

func newHistoryClearCommand(a *app.App) *cobra.Command {
	return &cobra.Command{
		Use:   "clear",
		Short: "Clear all history entries",
		RunE: func(cmd *cobra.Command, args []string) error {
			if a.HistoryStore == nil {
				return fmt.Errorf("history store not initialized")
			}

			if err := a.HistoryStore.Clear(); err != nil {
				return fmt.Errorf("clearing history: %w", err)
			}

			cmd.Println("Cleared all history entries.")
			return nil
		},
	}
}

func printHistoryTable(cmd *cobra.Command, entries []history.Entry) error {
	w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 0, 2, ' ', 0)
	fmt.Fprintf(w, "ID\tTIME\tMETHOD\tURL\tSTATUS\tDURATION\n")
	for _, e := range entries {
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%d\t%s\n",
			e.ID,
			e.Timestamp.Format("2006-01-02 15:04:05"),
			e.Method,
			e.URL,
			e.Status,
			e.Duration,
		)
	}
	return w.Flush()
}
