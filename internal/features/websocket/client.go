package websocket

import (
	"context"
	"errors"
	"net/http"
	"sync"
	"time"

	gorillaWS "github.com/gorilla/websocket"
)

// Message represents a WebSocket message received from the server.
type Message struct {
	Type int    // gorillaWS.TextMessage or gorillaWS.BinaryMessage
	Data []byte
	Time time.Time
}

// Client is a WebSocket client that wraps gorilla/websocket.
type Client struct {
	conn   *gorillaWS.Conn
	mu     sync.Mutex
	closed bool
}

// NewClient creates a new WebSocket client.
func NewClient() *Client {
	return &Client{}
}

// Connect dials the given WebSocket URL and establishes a connection.
// Custom headers can be passed for the handshake.
func (c *Client) Connect(ctx context.Context, url string, headers map[string]string) error {
	dialer := gorillaWS.Dialer{}

	var reqHeaders http.Header
	if len(headers) > 0 {
		reqHeaders = make(http.Header)
		for k, v := range headers {
			reqHeaders.Set(k, v)
		}
	}

	conn, _, err := dialer.DialContext(ctx, url, reqHeaders)
	if err != nil {
		return err
	}

	c.mu.Lock()
	c.conn = conn
	c.closed = false
	c.mu.Unlock()

	return nil
}

// errClosed is returned when operations are attempted on a closed client.
var errClosed = errors.New("websocket: client is closed")

// Send sends a text message over the WebSocket connection.
func (c *Client) Send(msg string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.closed || c.conn == nil {
		return errClosed
	}
	return c.conn.WriteMessage(gorillaWS.TextMessage, []byte(msg))
}

// SendBinary sends a binary message over the WebSocket connection.
func (c *Client) SendBinary(data []byte) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.closed || c.conn == nil {
		return errClosed
	}
	return c.conn.WriteMessage(gorillaWS.BinaryMessage, data)
}

// Receive reads the next message from the WebSocket connection.
func (c *Client) Receive() (Message, error) {
	c.mu.Lock()
	conn := c.conn
	closed := c.closed
	c.mu.Unlock()

	if closed || conn == nil {
		return Message{}, errClosed
	}

	mt, data, err := conn.ReadMessage()
	if err != nil {
		return Message{}, err
	}

	return Message{
		Type: mt,
		Data: data,
		Time: time.Now(),
	}, nil
}

// Close gracefully closes the WebSocket connection.
func (c *Client) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.closed || c.conn == nil {
		return nil
	}

	c.closed = true
	return c.conn.Close()
}
