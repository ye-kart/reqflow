package websocket_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	gorillaWS "github.com/gorilla/websocket"
	"github.com/ye-kart/reqflow/internal/features/websocket"
)

// newTestServer creates an httptest server that upgrades HTTP to WebSocket.
func newTestServer(t *testing.T, handler func(conn *gorillaWS.Conn)) *httptest.Server {
	t.Helper()
	upgrader := gorillaWS.Upgrader{
		CheckOrigin: func(r *http.Request) bool { return true },
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			t.Logf("upgrade error: %v", err)
			return
		}
		defer conn.Close()
		handler(conn)
	}))
	return srv
}

// wsURL converts an httptest server URL to ws:// scheme.
func wsURL(srv *httptest.Server) string {
	return "ws" + srv.URL[len("http"):]
}

func TestClient_Connect(t *testing.T) {
	srv := newTestServer(t, func(conn *gorillaWS.Conn) {
		// Just accept the connection and wait for close.
		for {
			_, _, err := conn.ReadMessage()
			if err != nil {
				return
			}
		}
	})
	defer srv.Close()

	client := websocket.NewClient()
	err := client.Connect(context.Background(), wsURL(srv), nil)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	defer client.Close()
}

func TestClient_SendAndReceiveText(t *testing.T) {
	srv := newTestServer(t, func(conn *gorillaWS.Conn) {
		// Echo server: read a message and send it back.
		mt, data, err := conn.ReadMessage()
		if err != nil {
			return
		}
		conn.WriteMessage(mt, data)
	})
	defer srv.Close()

	client := websocket.NewClient()
	err := client.Connect(context.Background(), wsURL(srv), nil)
	if err != nil {
		t.Fatalf("connect error: %v", err)
	}
	defer client.Close()

	err = client.Send("hello")
	if err != nil {
		t.Fatalf("send error: %v", err)
	}

	msg, err := client.Receive()
	if err != nil {
		t.Fatalf("receive error: %v", err)
	}

	if msg.Type != gorillaWS.TextMessage {
		t.Errorf("expected text message type %d, got %d", gorillaWS.TextMessage, msg.Type)
	}
	if string(msg.Data) != "hello" {
		t.Errorf("expected data 'hello', got %q", string(msg.Data))
	}
	if msg.Time.IsZero() {
		t.Error("expected non-zero timestamp")
	}
}

func TestClient_SendAndReceiveBinary(t *testing.T) {
	srv := newTestServer(t, func(conn *gorillaWS.Conn) {
		mt, data, err := conn.ReadMessage()
		if err != nil {
			return
		}
		conn.WriteMessage(mt, data)
	})
	defer srv.Close()

	client := websocket.NewClient()
	err := client.Connect(context.Background(), wsURL(srv), nil)
	if err != nil {
		t.Fatalf("connect error: %v", err)
	}
	defer client.Close()

	payload := []byte{0x01, 0x02, 0x03, 0xFF}
	err = client.SendBinary(payload)
	if err != nil {
		t.Fatalf("send binary error: %v", err)
	}

	msg, err := client.Receive()
	if err != nil {
		t.Fatalf("receive error: %v", err)
	}

	if msg.Type != gorillaWS.BinaryMessage {
		t.Errorf("expected binary message type %d, got %d", gorillaWS.BinaryMessage, msg.Type)
	}
	if len(msg.Data) != 4 || msg.Data[3] != 0xFF {
		t.Errorf("expected binary payload, got %v", msg.Data)
	}
}

func TestClient_Close(t *testing.T) {
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
	err := client.Connect(context.Background(), wsURL(srv), nil)
	if err != nil {
		t.Fatalf("connect error: %v", err)
	}

	err = client.Close()
	if err != nil {
		t.Fatalf("close error: %v", err)
	}

	// After close, Send should return an error.
	err = client.Send("should fail")
	if err == nil {
		t.Error("expected error after close, got nil")
	}
}

func TestClient_CustomHeaders(t *testing.T) {
	var receivedHeaders http.Header
	upgrader := gorillaWS.Upgrader{
		CheckOrigin: func(r *http.Request) bool { return true },
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedHeaders = r.Header.Clone()
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer conn.Close()
		for {
			_, _, err := conn.ReadMessage()
			if err != nil {
				return
			}
		}
	}))
	defer srv.Close()

	client := websocket.NewClient()
	headers := map[string]string{
		"X-Custom-Header": "custom-value",
		"Authorization":   "Bearer token123",
	}
	err := client.Connect(context.Background(), "ws"+srv.URL[len("http"):], headers)
	if err != nil {
		t.Fatalf("connect error: %v", err)
	}
	defer client.Close()

	if receivedHeaders.Get("X-Custom-Header") != "custom-value" {
		t.Errorf("expected X-Custom-Header 'custom-value', got %q", receivedHeaders.Get("X-Custom-Header"))
	}
	if receivedHeaders.Get("Authorization") != "Bearer token123" {
		t.Errorf("expected Authorization 'Bearer token123', got %q", receivedHeaders.Get("Authorization"))
	}
}

func TestClient_ConnectTimeout(t *testing.T) {
	// Use a context with very short timeout to a non-routable address.
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	client := websocket.NewClient()
	// 10.255.255.1 is non-routable, so connect should timeout.
	err := client.Connect(ctx, "ws://10.255.255.1:9999/ws", nil)
	if err == nil {
		client.Close()
		t.Fatal("expected timeout error, got nil")
	}
}
