package commands

import (
	"context"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/ye-kart/reqflow/internal/app"
	"github.com/ye-kart/reqflow/internal/features/websocket"
)

func newWSCommand(_ *app.App) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "ws",
		Short: "WebSocket operations",
		Long:  "Connect to WebSocket servers, send and receive messages interactively or in batch mode.",
	}

	cmd.AddCommand(newWSConnectCommand())
	cmd.AddCommand(newWSSendCommand())
	cmd.AddCommand(newWSListenCommand())

	return cmd
}

func newWSConnectCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "connect <url>",
		Short: "Connect to a WebSocket server and enter interactive mode",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			url := args[0]
			headers := parseWSHeaders(cmd)

			client := websocket.NewClient()
			ctx, cancel := context.WithCancel(cmd.Context())
			defer cancel()

			if err := client.Connect(ctx, url, headers); err != nil {
				return fmt.Errorf("connecting to %s: %w", url, err)
			}
			defer client.Close()

			input := cmd.InOrStdin()
			output := cmd.OutOrStdout()

			return websocket.RunInteractive(ctx, client, input, output)
		},
	}

	cmd.Flags().StringSliceP("header", "H", nil, `add headers (format "Key: Value")`)

	return cmd
}

func newWSSendCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "send <url>",
		Short: "Send a single message from stdin to a WebSocket server",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			url := args[0]
			headers := parseWSHeaders(cmd)

			client := websocket.NewClient()
			ctx, cancel := context.WithCancel(cmd.Context())
			defer cancel()

			if err := client.Connect(ctx, url, headers); err != nil {
				return fmt.Errorf("connecting to %s: %w", url, err)
			}
			defer client.Close()

			// Read all of stdin as the message.
			data, err := io.ReadAll(cmd.InOrStdin())
			if err != nil {
				return fmt.Errorf("reading stdin: %w", err)
			}

			msg := strings.TrimRight(string(data), "\n")
			if err := client.Send(msg); err != nil {
				return fmt.Errorf("sending message: %w", err)
			}

			return nil
		},
	}

	cmd.Flags().StringSliceP("header", "H", nil, `add headers (format "Key: Value")`)

	return cmd
}

func newWSListenCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "listen <url>",
		Short: "Listen for messages from a WebSocket server",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			url := args[0]
			headers := parseWSHeaders(cmd)
			timeout, _ := cmd.Flags().GetDuration("timeout")

			client := websocket.NewClient()
			ctx, cancel := context.WithCancel(cmd.Context())
			defer cancel()

			if timeout > 0 {
				ctx, cancel = context.WithTimeout(ctx, timeout)
				defer cancel()
			}

			if err := client.Connect(ctx, url, headers); err != nil {
				return fmt.Errorf("connecting to %s: %w", url, err)
			}
			defer client.Close()

			w := cmd.OutOrStdout()
			for {
				msg, err := client.Receive()
				if err != nil {
					// Context timeout or connection closed by server is a normal exit.
					select {
					case <-ctx.Done():
						return nil
					default:
					}
					// Check if it's a normal close.
					if websocket.IsCloseError(err) {
						return nil
					}
					return fmt.Errorf("receiving message: %w", err)
				}
				fmt.Fprintf(w, "< %s\n", string(msg.Data))
			}
		},
	}

	cmd.Flags().StringSliceP("header", "H", nil, `add headers (format "Key: Value")`)
	cmd.Flags().Duration("timeout", 0, "listen timeout (e.g. 10s)")

	return cmd
}

// parseWSHeaders extracts header key-value pairs from the -H flag.
func parseWSHeaders(cmd *cobra.Command) map[string]string {
	headerSlice, _ := cmd.Flags().GetStringSlice("header")
	if len(headerSlice) == 0 {
		return nil
	}

	headers := make(map[string]string, len(headerSlice))
	for _, h := range headerSlice {
		key, value, ok := parseHeader(h)
		if ok {
			headers[key] = value
		}
	}
	return headers
}

// defaultEnvDir is needed by the ws command to avoid import cycle warnings.
// It is already defined in request.go but is package-private; we reuse
// the parseHeader function defined there.

// Ensure the ws subcommand has a reasonable default timeout for listen.
func init() {
	_ = time.Second // ensure time is imported
}
