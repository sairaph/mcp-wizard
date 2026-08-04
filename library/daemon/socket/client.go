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

type callResult struct {
	resp rpc.Response
	err  error
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

	ch := make(chan callResult, 1)
	go func() {
		if err := c.enc.Encode(req); err != nil {
			ch <- callResult{err: fmt.Errorf("send request: %w", err)}
			return
		}
		c.conn.SetReadDeadline(time.Now().Add(30 * time.Second))
		var resp rpc.Response
		if err := c.dec.Decode(&resp); err != nil {
			ch <- callResult{err: fmt.Errorf("read response: %w", err)}
			return
		}
		ch <- callResult{resp: resp}
	}()

	select {
	case <-ctx.Done():
		// Close the connection to unblock the goroutine immediately.
		// The client must be re-dialed after a cancelled call.
		c.conn.Close()
		return ctx.Err()
	case r := <-ch:
		if r.err != nil {
			return r.err
		}
		if r.resp.Error != nil {
			return fmt.Errorf("daemon error: %s", r.resp.Error.Message)
		}
		if r.resp.ID != id {
			return fmt.Errorf("daemon: response ID %d does not match request ID %d", r.resp.ID, id)
		}
		if result != nil && r.resp.Result != nil {
			if err := json.Unmarshal(r.resp.Result, result); err != nil {
				return fmt.Errorf("decode result: %w", err)
			}
		}
		return nil
	}
}

func (c *Client) Close() error {
	return c.conn.Close()
}
