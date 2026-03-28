package commands_test

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	gorillaWS "github.com/gorilla/websocket"
	"github.com/ye-kart/reqflow/internal/adapters/cli/commands"
	"github.com/ye-kart/reqflow/internal/domain"
)

// newWSTestServer creates an httptest server that upgrades to WebSocket.
func newWSTestServer(t *testing.T, handler func(conn *gorillaWS.Conn)) *httptest.Server {
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

func wsTestURL(srv *httptest.Server) string {
	return "ws" + srv.URL[len("http"):]
}

func TestWSConnectCommand_InteractiveMode(t *testing.T) {
	srv := newWSTestServer(t, func(conn *gorillaWS.Conn) {
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

	mock := &mockHTTPClient{
		doFunc: func(_ context.Context, _ domain.HTTPRequest) (domain.HTTPResponse, error) {
			return domain.HTTPResponse{StatusCode: 200}, nil
		},
	}
	a := newTestApp(mock)
	root := commands.NewRootCommand(a)

	buf := new(bytes.Buffer)
	root.SetOut(buf)
	root.SetErr(buf)
	root.SetIn(strings.NewReader("hello\n"))
	root.SetArgs([]string{"ws", "connect", wsTestURL(srv)})

	err := root.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "> hello") {
		t.Errorf("expected output to contain '> hello', got:\n%s", output)
	}
	if !strings.Contains(output, "< hello") {
		t.Errorf("expected output to contain '< hello', got:\n%s", output)
	}
}

func TestWSSendCommand_SendsSingleMessage(t *testing.T) {
	receivedCh := make(chan string, 1)
	srv := newWSTestServer(t, func(conn *gorillaWS.Conn) {
		_, data, err := conn.ReadMessage()
		if err != nil {
			receivedCh <- ""
			return
		}
		receivedCh <- string(data)
	})
	defer srv.Close()

	mock := &mockHTTPClient{
		doFunc: func(_ context.Context, _ domain.HTTPRequest) (domain.HTTPResponse, error) {
			return domain.HTTPResponse{StatusCode: 200}, nil
		},
	}
	a := newTestApp(mock)
	root := commands.NewRootCommand(a)

	buf := new(bytes.Buffer)
	root.SetOut(buf)
	root.SetErr(buf)
	root.SetIn(strings.NewReader("hello from pipe"))
	root.SetArgs([]string{"ws", "send", wsTestURL(srv)})

	err := root.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	received := <-receivedCh
	if received != "hello from pipe" {
		t.Errorf("expected server to receive 'hello from pipe', got %q", received)
	}
}

func TestWSListenCommand_ReceivesMessages(t *testing.T) {
	srv := newWSTestServer(t, func(conn *gorillaWS.Conn) {
		conn.WriteMessage(gorillaWS.TextMessage, []byte("msg1"))
		conn.WriteMessage(gorillaWS.TextMessage, []byte("msg2"))
		conn.Close()
	})
	defer srv.Close()

	mock := &mockHTTPClient{
		doFunc: func(_ context.Context, _ domain.HTTPRequest) (domain.HTTPResponse, error) {
			return domain.HTTPResponse{StatusCode: 200}, nil
		},
	}
	a := newTestApp(mock)
	root := commands.NewRootCommand(a)

	buf := new(bytes.Buffer)
	root.SetOut(buf)
	root.SetErr(buf)
	root.SetArgs([]string{"ws", "listen", wsTestURL(srv), "--timeout", "2s"})

	err := root.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "msg1") {
		t.Errorf("expected output to contain 'msg1', got:\n%s", output)
	}
	if !strings.Contains(output, "msg2") {
		t.Errorf("expected output to contain 'msg2', got:\n%s", output)
	}
}

func TestWSConnectCommand_CustomHeaders(t *testing.T) {
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

	mock := &mockHTTPClient{
		doFunc: func(_ context.Context, _ domain.HTTPRequest) (domain.HTTPResponse, error) {
			return domain.HTTPResponse{StatusCode: 200}, nil
		},
	}
	a := newTestApp(mock)
	root := commands.NewRootCommand(a)

	buf := new(bytes.Buffer)
	root.SetOut(buf)
	root.SetErr(buf)
	root.SetIn(strings.NewReader(""))
	root.SetArgs([]string{
		"ws", "connect",
		"ws" + srv.URL[len("http"):],
		"-H", "X-Custom: myvalue",
	})

	err := root.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if receivedHeaders.Get("X-Custom") != "myvalue" {
		t.Errorf("expected X-Custom header 'myvalue', got %q", receivedHeaders.Get("X-Custom"))
	}
}
