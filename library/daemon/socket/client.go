package socket

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/sairaph/mcp-wizard/daemon/rpc"
)

type Client struct {
	socketPath string
	conn       net.Conn
	enc        *json.Encoder
	dec        *json.Decoder
	nextID     int64
	mu         sync.Mutex
}

func Dial(socketPath string) (*Client, error) {
	conn, err := net.Dial("unix", socketPath)
	if err != nil {
		return nil, fmt.Errorf("dial daemon: %w", err)
	}
	return &Client{
		socketPath: socketPath,
		conn:       conn,
		enc:        json.NewEncoder(conn),
		dec:        json.NewDecoder(conn),
	}, nil
}

func (c *Client) Call(ctx context.Context, method string, params, result any) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	id := c.nextID
	c.nextID++

	req, err := rpc.NewRequest(id, method, params)
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}

	if err := c.enc.Encode(req); err != nil {
		return fmt.Errorf("send request: %w", err)
	}

	done := make(chan struct{})
	go func() {
		select {
		case <-done:
		case <-ctx.Done():
			c.conn.SetReadDeadline(time.Now())
		}
	}()
	defer close(done)

	c.conn.SetReadDeadline(time.Now().Add(30 * time.Second))
	var resp rpc.Response
	if err := c.dec.Decode(&resp); err != nil {
		return fmt.Errorf("read response: %w", err)
	}
	if resp.Error != nil {
		return fmt.Errorf("daemon error: %s", resp.Error.Message)
	}
	if resp.ID != id {
		return fmt.Errorf("daemon: response ID %d does not match request ID %d", resp.ID, id)
	}
	if result != nil && resp.Result != nil {
		if err := json.Unmarshal(resp.Result, result); err != nil {
			return fmt.Errorf("decode result: %w", err)
		}
	}
	return nil
}

func (c *Client) Close() error {
	return c.conn.Close()
}
