package commands

import (
	"fmt"
	"path/filepath"
	"time"

	"github.com/spf13/cobra"
	"github.com/ye-kart/reqflow/internal/app"
	"github.com/ye-kart/reqflow/internal/features/mockserver"
)

func newMockCommand(a *app.App) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "mock",
		Short: "Mock server from collections",
		Long:  "Start a local mock server that serves canned responses from a collection.",
	}

	cmd.PersistentFlags().String("collection-dir", defaultCollectionDir(), "directory containing collection files")

	cmd.AddCommand(newMockStartCommand(a))
	cmd.AddCommand(newMockStopCommand())

	return cmd
}

func newMockStartCommand(a *app.App) *cobra.Command {
	var port int
	var delay time.Duration

	cmd := &cobra.Command{
		Use:   "start <collection>",
		Short: "Start a mock server from a collection",
		Long:  "Start a local HTTP server that returns example responses defined in a collection.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			dir, _ := cmd.Flags().GetString("collection-dir")
			name := args[0]
			path := filepath.Join(dir, name+".yaml")

			col, err := a.Storage.ReadCollection(path)
			if err != nil {
				return fmt.Errorf("reading collection %q: %w", name, err)
			}

			var opts []mockserver.Option
			if delay > 0 {
				opts = append(opts, mockserver.WithDelay(delay))
			}

			srv := mockserver.New(col, port, opts...)

			cmd.Printf("Mock server starting for collection %q on port %d\n", name, port)
			cmd.Printf("Routes: %d\n", srv.RouteCount())
			cmd.Printf("Press Ctrl+C to stop.\n")

			if err := srv.Start(); err != nil {
				return fmt.Errorf("starting mock server: %w", err)
			}

			return nil
		},
	}

	cmd.Flags().IntVar(&port, "port", 8080, "port to listen on")
	cmd.Flags().DurationVar(&delay, "delay", 0, "response delay (e.g. 100ms)")

	return cmd
}

func newMockStopCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "stop",
		Short: "Stop the mock server",
		Long:  "The mock server runs in the foreground. Use Ctrl+C to stop it.",
		RunE: func(cmd *cobra.Command, _ []string) error {
			cmd.Println("The mock server runs in the foreground.")
			cmd.Println("Use Ctrl+C to stop a running mock server.")
			return nil
		},
	}
}
