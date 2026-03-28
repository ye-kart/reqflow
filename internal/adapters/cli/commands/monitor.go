package commands

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/ye-kart/reqflow/internal/app"
	"github.com/ye-kart/reqflow/internal/features/monitor"
)

// defaultMonitorDir returns the default monitor config directory (~/.reqflow/monitors/).
func defaultMonitorDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ".reqflow/monitors"
	}
	return filepath.Join(home, ".reqflow", "monitors")
}

func newMonitorCommand(a *app.App) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "monitor",
		Short: "Manage scheduled workflow monitors",
		Long:  "Create, list, delete, and run scheduled workflow monitors with cron expressions.",
	}

	cmd.AddCommand(newMonitorCreateCommand(a))
	cmd.AddCommand(newMonitorListCommand(a))
	cmd.AddCommand(newMonitorDeleteCommand(a))
	cmd.AddCommand(newMonitorRunCommand(a))
	cmd.AddCommand(newMonitorStartCommand(a))

	return cmd
}

func newMonitorCreateCommand(a *app.App) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "create <name>",
		Short: "Create a new scheduled monitor",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]

			workflowPath, _ := cmd.Flags().GetString("workflow")
			cronExpr, _ := cmd.Flags().GetString("cron")
			envName, _ := cmd.Flags().GetString("env")

			if a.Scheduler == nil {
				return fmt.Errorf("scheduler not initialized")
			}

			m := monitor.Monitor{
				Name:         name,
				WorkflowPath: workflowPath,
				Cron:         cronExpr,
				EnvName:      envName,
			}

			if err := a.Scheduler.Add(m); err != nil {
				return fmt.Errorf("creating monitor: %w", err)
			}

			cmd.Printf("Created monitor %q (cron: %s)\n", name, cronExpr)
			return nil
		},
	}

	cmd.Flags().String("workflow", "", "path to workflow YAML file")
	cmd.Flags().String("cron", "", "cron expression (e.g. \"*/5 * * * *\")")
	_ = cmd.MarkFlagRequired("workflow")
	_ = cmd.MarkFlagRequired("cron")

	return cmd
}

func newMonitorListCommand(a *app.App) *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List all monitors",
		RunE: func(cmd *cobra.Command, _ []string) error {
			if a.Scheduler == nil {
				return fmt.Errorf("scheduler not initialized")
			}

			monitors := a.Scheduler.List()
			if len(monitors) == 0 {
				cmd.Println("No monitors configured.")
				return nil
			}

			for _, m := range monitors {
				envStr := ""
				if m.EnvName != "" {
					envStr = fmt.Sprintf(" [env: %s]", m.EnvName)
				}
				cmd.Printf("  %-20s  %-20s  %s%s\n", m.Name, m.Cron, m.WorkflowPath, envStr)
			}
			return nil
		},
	}
}

func newMonitorDeleteCommand(a *app.App) *cobra.Command {
	return &cobra.Command{
		Use:   "delete <name>",
		Short: "Delete a monitor",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]

			if a.Scheduler == nil {
				return fmt.Errorf("scheduler not initialized")
			}

			if err := a.Scheduler.Remove(name); err != nil {
				return fmt.Errorf("deleting monitor: %w", err)
			}

			cmd.Printf("Deleted monitor %q\n", name)
			return nil
		},
	}
}

func newMonitorRunCommand(a *app.App) *cobra.Command {
	return &cobra.Command{
		Use:   "run <name>",
		Short: "Run a monitor's workflow once",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]

			if a.Scheduler == nil {
				return fmt.Errorf("scheduler not initialized")
			}

			timeout, _ := cmd.Flags().GetDuration("timeout")
			ctx, cancel := context.WithTimeout(context.Background(), resolveTimeout(timeout))
			defer cancel()

			// Find the monitor to display its details
			monitors := a.Scheduler.List()
			for _, m := range monitors {
				if m.Name == name {
					cmd.Printf("Running monitor %q (workflow: %s)...\n", name, m.WorkflowPath)
					break
				}
			}

			result, err := a.Scheduler.RunOnceWithResult(ctx, name, nil)
			if err != nil {
				return fmt.Errorf("running monitor: %w", err)
			}

			cmd.Printf("Workflow: %s\n", result.Name)
			cmd.Printf("Steps: %d passed, %d failed (%s)\n",
				result.TotalPassed, result.TotalFailed, result.Duration)

			return nil
		},
	}
}

func newMonitorStartCommand(a *app.App) *cobra.Command {
	return &cobra.Command{
		Use:   "start",
		Short: "Start the scheduler (foreground)",
		Long:  "Start the monitor scheduler in the foreground. It will run configured monitors on their cron schedules.",
		RunE: func(cmd *cobra.Command, _ []string) error {
			if a.Scheduler == nil {
				return fmt.Errorf("scheduler not initialized")
			}

			cmd.Println("Starting monitor scheduler...")
			monitors := a.Scheduler.List()
			if len(monitors) == 0 {
				cmd.Println("No monitors configured. Add monitors with 'reqflow monitor create'.")
				return nil
			}

			for _, m := range monitors {
				cmd.Printf("  Scheduling %q (%s)\n", m.Name, m.Cron)
			}
			cmd.Println("Press Ctrl+C to stop.")

			return a.Scheduler.Start(cmd.Context())
		},
	}
}

