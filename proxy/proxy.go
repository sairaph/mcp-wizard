// Package proxy bridges a local stdio MCP session to a remote server over
// Streamable HTTP. It is the "bridge" delivery mode for remote servers: the
// AI client spawns the project binary as an ordinary stdio server, and the
// binary forwards every JSON-RPC message to the remote endpoint, adding the
// credentials it holds. The secret therefore lives once, in the project's
// credential store, and never in any client's configuration file, and every
// client works identically, including ones whose config cannot express a
// remote entry at all.
//
// The bridge is transparent: it copies messages in both directions without
// interpreting them, apart from remembering the negotiated protocol version
// so it can be sent on every subsequent request as the transport requires.
package proxy

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/modelcontextprotocol/go-sdk/jsonrpc"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// Config describes the remote endpoint and how to authenticate to it.
type Config struct {
	// URL is the remote Streamable HTTP endpoint.
	URL string

	// Headers are sent on every request, typically
	// {"Authorization": "Bearer <token>"}. Headers the transport manages
	// itself (Accept, Content-Type, Mcp-Session-Id, MCP-Protocol-Version)
	// are rejected.
	Headers map[string]string

	// HeaderFunc, when set, is called once at start to produce additional
	// headers, for example from a credential store. Its result is merged over
	// Headers.
	HeaderFunc func(ctx context.Context) (map[string]string, error)

	// HTTPClient performs the requests. Nil means http.DefaultClient. Its
	// Transport is wrapped to inject the headers.
	HTTPClient *http.Client

	// Local is the transport the AI client talks to. Nil means stdio.
	Local mcp.Transport
}

// ErrUnauthorized is returned (wrapped) when the remote rejects the
// credentials with HTTP 401 or 403.
var ErrUnauthorized = errors.New("remote server rejected the credentials")

// drainTimeout bounds how long in-flight calls may finish after the client
// hung up before the bridge tears the session down anyway.
const drainTimeout = 30 * time.Second

// closeTimeout bounds the session DELETE the transport sends on Close, so a
// stalled remote cannot keep the bridge from exiting.
const closeTimeout = 5 * time.Second

var reservedHeaders = map[string]bool{
	"accept":               true,
	"content-type":         true,
	"mcp-session-id":       true,
	"mcp-protocol-version": true,
}

// Run bridges the local transport to the remote endpoint until the local
// side closes (the normal end of a session, returns nil), the context is
// cancelled (returns ctx.Err()), or the remote fails (returns the failure).
func Run(ctx context.Context, cfg Config) error {
	if cfg.URL == "" {
		return errors.New("proxy: URL must not be empty")
	}
	headers := make(map[string]string, len(cfg.Headers))
	for k, v := range cfg.Headers {
		headers[k] = v
	}
	if cfg.HeaderFunc != nil {
		extra, err := cfg.HeaderFunc(ctx)
		if err != nil {
			return fmt.Errorf("proxy: headers: %w", err)
		}
		for k, v := range extra {
			headers[k] = v
		}
	}
	for k := range headers {
		if reservedHeaders[strings.ToLower(k)] {
			return fmt.Errorf("proxy: header %q is managed by the transport and cannot be overridden", k)
		}
	}

	base := http.DefaultClient
	if cfg.HTTPClient != nil {
		base = cfg.HTTPClient
	}
	rt := &injector{headers: headers, base: base.Transport}
	client := *base
	client.Transport = rt

	local := cfg.Local
	if local == nil {
		local = &mcp.StdioTransport{}
	}

	parent := ctx
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	remoteT := &mcp.StreamableClientTransport{
		Endpoint:   cfg.URL,
		HTTPClient: &client,
		// The bridge cannot participate in the SDK's client session, which
		// is what opens the standalone SSE stream, so server-initiated
		// messages outside a request are not forwarded.
		DisableStandaloneSSE: true,
	}
	remote, err := remoteT.Connect(ctx)
	if err != nil {
		return fmt.Errorf("proxy: connect %s: %w", cfg.URL, err)
	}
	defer remote.Close()

	lconn, err := local.Connect(ctx)
	if err != nil {
		return fmt.Errorf("proxy: local transport: %w", err)
	}
	defer lconn.Close()

	p := &pump{rt: rt, cancel: cancel}
	type outcome struct {
		remoteSide bool
		err        error
	}
	outcomes := make(chan outcome, 2)
	go func() { outcomes <- outcome{false, p.outbound(ctx, lconn, remote)} }()
	go func() { outcomes <- outcome{true, p.inbound(ctx, remote, lconn)} }()

	// The first side to stop ends the session. When the client hung up, let
	// calls still in flight finish first: a client may half-close its side
	// after sending a request and still read the answer. Closing both
	// connections then unblocks the other copy goroutine and any writes
	// still outstanding.
	first := <-outcomes
	if !first.remoteSide && isHangup(first.err) {
		drained := make(chan struct{})
		go func() { p.writes.Wait(); close(drained) }()
		select {
		case <-drained:
		case <-ctx.Done():
		case <-time.After(drainTimeout):
		}
	}
	remote.Close()
	lconn.Close()
	cancel()
	<-outcomes
	p.writes.Wait()

	// A fatal write error (rejected credentials) cancels the pump, so the
	// other side may report the cancellation first; the recorded cause wins.
	fatal := p.fatalError()
	switch {
	case parent.Err() != nil:
		return parent.Err()
	case fatal != nil:
		return fatal
	case first.remoteSide:
		return fmt.Errorf("proxy: remote %s: %w", cfg.URL, first.err)
	case isHangup(first.err):
		return nil // the AI client hung up, which is how sessions end
	default:
		return fmt.Errorf("proxy: local transport: %w", first.err)
	}
}

// isHangup reports whether a local read error means the client went away
// normally.
func isHangup(err error) bool {
	return err == nil || errors.Is(err, io.EOF) || errors.Is(err, net.ErrClosed)
}

// pump copies messages between the two connections and tracks the
// initialize handshake so the negotiated protocol version reaches the
// remote transport's headers.
type pump struct {
	rt     *injector
	cancel context.CancelFunc
	writes sync.WaitGroup

	mu     sync.Mutex
	initID jsonrpc.ID
	seen   bool
	fatal  error
}

// outbound forwards client-to-remote traffic. Calls are written from their
// own goroutines because the remote transport's Write blocks until the
// remote answers, and a client must be able to send further calls and
// cancellations while one is in flight. Notifications and responses are
// written inline to keep their order relative to each other.
//
// A write that fails for one call becomes a JSON-RPC error response to the
// client for that call only, mirroring what the SDK's own client does for
// transient HTTP failures; rejected credentials end the session.
func (p *pump) outbound(ctx context.Context, from, to mcp.Connection) error {
	for {
		msg, err := from.Read(ctx)
		if err != nil {
			if fatal := p.fatalError(); fatal != nil {
				return fatal
			}
			return err
		}
		p.observeOutbound(msg)

		req, isCall := msg.(*jsonrpc.Request)
		if !isCall || !req.IsCall() {
			if err := to.Write(ctx, msg); err != nil {
				if p.failed(err) {
					return err
				}
				// A notification the remote would not take is not fatal.
			}
			continue
		}

		p.writes.Add(1)
		go func(req *jsonrpc.Request) {
			defer p.writes.Done()
			err := to.Write(ctx, req)
			if err == nil {
				return
			}
			if p.failed(err) {
				return
			}
			if ctx.Err() != nil {
				return
			}
			_ = from.Write(ctx, &jsonrpc.Response{ID: req.ID, Error: err})
		}(req)
	}
}

// failed records a session-ending write error and stops the pump. It
// reports whether err was fatal.
func (p *pump) failed(err error) bool {
	if !errors.Is(err, ErrUnauthorized) {
		return false
	}
	p.mu.Lock()
	if p.fatal == nil {
		p.fatal = err
	}
	p.mu.Unlock()
	p.cancel()
	return true
}

func (p *pump) fatalError() error {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.fatal
}

// inbound forwards remote-to-client traffic.
func (p *pump) inbound(ctx context.Context, from, to mcp.Connection) error {
	for {
		msg, err := from.Read(ctx)
		if err != nil {
			return err
		}
		p.observeInbound(msg)
		if err := to.Write(ctx, msg); err != nil {
			return err
		}
	}
}

// observeOutbound watches client-to-remote traffic for the initialize call.
func (p *pump) observeOutbound(msg jsonrpc.Message) {
	req, ok := msg.(*jsonrpc.Request)
	if !ok || req.Method != "initialize" || !req.IsCall() {
		return
	}
	p.mu.Lock()
	p.initID = req.ID
	p.seen = true
	p.mu.Unlock()
}

// observeInbound watches remote-to-client traffic for the initialize result
// and records the protocol version the server chose.
func (p *pump) observeInbound(msg jsonrpc.Message) {
	resp, ok := msg.(*jsonrpc.Response)
	if !ok || resp.Error != nil {
		return
	}
	p.mu.Lock()
	match := p.seen && resp.ID == p.initID
	p.mu.Unlock()
	if !match {
		return
	}
	var result struct {
		ProtocolVersion string `json:"protocolVersion"`
	}
	if json.Unmarshal(resp.Result, &result) == nil && result.ProtocolVersion != "" {
		p.rt.setProtocolVersion(result.ProtocolVersion)
	}
}

// injector adds the configured headers and the negotiated protocol version
// to every outgoing request.
type injector struct {
	headers map[string]string
	base    http.RoundTripper

	mu      sync.RWMutex
	version string
}

func (i *injector) setProtocolVersion(v string) {
	i.mu.Lock()
	i.version = v
	i.mu.Unlock()
}

func (i *injector) RoundTrip(req *http.Request) (*http.Response, error) {
	req = req.Clone(req.Context())
	for k, v := range i.headers {
		req.Header.Set(k, v)
	}
	i.mu.RLock()
	// Protocol versions from 2026-06-30 on also require Mcp-Method and
	// Mcp-Name headers derived from the body; the SDK sets those only when
	// the version header is present before its own header pass, which a
	// bridge cannot arrange. go-sdk v1.6.1 negotiates at most 2025-11-25.
	if i.version != "" && req.Header.Get("MCP-Protocol-Version") == "" {
		req.Header.Set("MCP-Protocol-Version", i.version)
	}
	i.mu.RUnlock()

	if req.Method == http.MethodDelete {
		// The transport's Close sends the session DELETE on a detached
		// context; bound it so a stalled remote cannot hang shutdown.
		ctx, cancel := context.WithCancel(req.Context())
		time.AfterFunc(closeTimeout, cancel)
		req = req.WithContext(ctx)
	}

	base := i.base
	if base == nil {
		base = http.DefaultTransport
	}
	resp, err := base.RoundTrip(req)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		challenge := resp.Header.Get("WWW-Authenticate")
		resp.Body.Close()
		if challenge != "" {
			return nil, fmt.Errorf("%w (HTTP %d, WWW-Authenticate: %s)", ErrUnauthorized, resp.StatusCode, challenge)
		}
		return nil, fmt.Errorf("%w (HTTP %d)", ErrUnauthorized, resp.StatusCode)
	}
	return resp, nil
}
