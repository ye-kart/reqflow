package websocket_test

import (
	"bytes"
	"context"
	"strings"
	"testing"
	"time"

	gorillaWS "github.com/gorilla/websocket"
	"github.com/ye-kart/reqflow/internal/features/websocket"
)

func TestInteractive_SendsInputAndPrintsReceived(t *testing.T) {
	srv := newTestServer(t, func(conn *gorillaWS.Conn) {
		// Echo server.
		for {
			mt, data, err := conn.ReadMessage()
			if err != nil {
				return
			}
			conn.WriteMessage(mt, data)
		}
	})
	defer srv.Close()

	client := websocket.NewClient()
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	err := client.Connect(ctx, wsURL(srv), nil)
	if err != nil {
		t.Fatalf("connect error: %v", err)
	}
	defer client.Close()

	// Simulate stdin with two lines.
	input := strings.NewReader("hello\nworld\n")
	var output bytes.Buffer

	err = websocket.RunInteractive(ctx, client, input, &output)
	// RunInteractive exits when input is exhausted (io.EOF), which is not an error.
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	out := output.String()
	if !strings.Contains(out, "> hello") {
		t.Errorf("expected output to contain '> hello', got %q", out)
	}
	if !strings.Contains(out, "< hello") {
		t.Errorf("expected output to contain '< hello', got %q", out)
	}
	if !strings.Contains(out, "> world") {
		t.Errorf("expected output to contain '> world', got %q", out)
	}
	if !strings.Contains(out, "< world") {
		t.Errorf("expected output to contain '< world', got %q", out)
	}
}

func TestInteractive_ExitsOnContextCancel(t *testing.T) {
	srv := newTestServer(t, func(conn *gorillaWS.Conn) {
		for {
			_, _, err := conn.ReadMessage()
			if err != nil {
				return
			}
		}
	})
	defer srv.Close()

	client := websocket.NewClient()
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	err := client.Connect(ctx, wsURL(srv), nil)
	if err != nil {
		t.Fatalf("connect error: %v", err)
	}
	defer client.Close()

	// Use a blocking reader that never returns data; cancel the context to exit.
	blockingReader := &blockReader{ctx: ctx}
	var output bytes.Buffer

	// Cancel after a short delay.
	go func() {
		time.Sleep(50 * time.Millisecond)
		cancel()
	}()

	err = websocket.RunInteractive(ctx, client, blockingReader, &output)
	// Should exit without error (context cancellation is a normal exit).
	if err != nil && err != context.Canceled && err != context.DeadlineExceeded {
		t.Fatalf("unexpected error: %v", err)
	}
}

// blockReader blocks on Read until context is done.
type blockReader struct {
	ctx context.Context
}

func (r *blockReader) Read(p []byte) (int, error) {
	<-r.ctx.Done()
	return 0, r.ctx.Err()
}
