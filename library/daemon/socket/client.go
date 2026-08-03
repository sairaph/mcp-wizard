package socket

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"sync"
	"sync/atomic"

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

	id := atomic.AddInt64(&c.nextID, 1)
	req, err := rpc.NewRequest(id, method, params)
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}

	type callResult struct {
		resp rpc.Response
		err  error
	}
	ch := make(chan callResult, 1)
	go func() {
		if err := c.enc.Encode(req); err != nil {
			ch <- callResult{err: fmt.Errorf("send request: %w", err)}
			return
		}
		var resp rpc.Response
		if err := c.dec.Decode(&resp); err != nil {
			ch <- callResult{err: fmt.Errorf("read response: %w", err)}
			return
		}
		ch <- callResult{resp: resp}
	}()
	select {
	case <-ctx.Done():
		c.conn.Close()
		return ctx.Err()
	case r := <-ch:
		if r.err != nil {
			return r.err
		}
		if r.resp.Error != nil {
			return fmt.Errorf("daemon error: %s", r.resp.Error.Message)
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
