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

// Client is a JSON-RPC client over a Unix socket. Calls are serialised; a
// cancelled or failed call closes the connection and the client must be
// re-dialed.
type Client struct {
	socketPath string
	conn       net.Conn
	enc        *json.Encoder
	dec        *json.Decoder
	nextID     int64
	mu         sync.Mutex
}

// Dial connects to the daemon socket at socketPath.
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

// Call sends method with params and decodes the result into result (which
// may be nil). A daemon-side error is returned as an error.
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
			// The stream is now out of step (a late reply could arrive for
			// this ID); close so the caller re-dials rather than mis-pairing.
			c.conn.Close()
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

// Close closes the connection.
func (c *Client) Close() error {
	return c.conn.Close()
}
