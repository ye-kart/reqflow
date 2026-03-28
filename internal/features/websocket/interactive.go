package websocket

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"time"
)

// RunInteractive runs an interactive WebSocket session.
// It reads lines from input, sends them to the server, and prints received
// messages to output. Sent lines are prefixed with "> " and received messages
// with "< ". It returns when input is exhausted (io.EOF) or context is cancelled.
func RunInteractive(ctx context.Context, client *Client, input io.Reader, output io.Writer) error {
	recvErr := make(chan error, 1)
	sendDone := make(chan struct{})

	// Goroutine: receive messages from server and write to output.
	go func() {
		for {
			msg, err := client.Receive()
			if err != nil {
				select {
				case <-ctx.Done():
				default:
					recvErr <- err
				}
				return
			}
			fmt.Fprintf(output, "< %s\n", string(msg.Data))
		}
	}()

	// Goroutine: read lines from input and send to server.
	go func() {
		defer close(sendDone)
		scanner := bufio.NewScanner(input)
		for scanner.Scan() {
			select {
			case <-ctx.Done():
				return
			default:
			}
			line := scanner.Text()
			fmt.Fprintf(output, "> %s\n", line)
			if err := client.Send(line); err != nil {
				return
			}
		}
	}()

	select {
	case <-ctx.Done():
		client.Close()
		return nil
	case err := <-recvErr:
		client.Close()
		return err
	case <-sendDone:
		// Input exhausted. Give a short grace period for final echoed messages.
		time.Sleep(100 * time.Millisecond)
		client.Close()
		return nil
	}
}
