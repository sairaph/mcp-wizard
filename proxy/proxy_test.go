package proxy_test

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/sairaph/mcp-wizard/proxy"
)

type echoInput struct {
	Text string `json:"text"`
}

type sleepInput struct {
	Millis int `json:"millis"`
}

// remoteServer is a Streamable HTTP MCP server that requires a bearer token
// and records the headers it saw on tools/call requests.
type remoteServer struct {
	srv   *httptest.Server
	mu    sync.Mutex
	calls []http.Header
}

func newRemoteServer(t *testing.T, token string) *remoteServer {
	t.Helper()
	server := mcp.NewServer(&mcp.Implementation{Name: "remote", Version: "1"}, nil)
	mcp.AddTool(server, &mcp.Tool{Name: "echo", Description: "echo text"},
		func(ctx context.Context, req *mcp.CallToolRequest, in echoInput) (*mcp.CallToolResult, any, error) {
			return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: "echo: " + in.Text}}}, nil, nil
		})
	mcp.AddTool(server, &mcp.Tool{Name: "sleep", Description: "sleep"},
		func(ctx context.Context, req *mcp.CallToolRequest, in sleepInput) (*mcp.CallToolResult, any, error) {
			select {
			case <-time.After(time.Duration(in.Millis) * time.Millisecond):
			case <-ctx.Done():
				return nil, nil, ctx.Err()
			}
			return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: "slept"}}}, nil, nil
		})
	handler := mcp.NewStreamableHTTPHandler(func(*http.Request) *mcp.Server { return server }, nil)

	rs := &remoteServer{}
	var flakyOnce sync.Once
	rs.srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer "+token {
			w.Header().Set("WWW-Authenticate", `Bearer realm="test"`)
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		if r.Method == http.MethodPost {
			body, _ := io.ReadAll(r.Body)
			r.Body = io.NopCloser(bytes.NewReader(body))
			if r.Header.Get("X-Record") != "" {
				rs.mu.Lock()
				rs.calls = append(rs.calls, r.Header.Clone())
				rs.mu.Unlock()
			}
			if bytes.Contains(body, []byte(`"name":"flaky"`)) {
				rejected := false
				flakyOnce.Do(func() { rejected = true })
				if rejected {
					http.Error(w, "slow down", http.StatusTooManyRequests)
					return
				}
			}
			if bytes.Contains(body, []byte(`"name":"drop"`)) {
				// Send headers, then kill the connection mid-response.
				w.WriteHeader(http.StatusOK)
				w.(http.Flusher).Flush()
				if hj, ok := w.(http.Hijacker); ok {
					conn, _, _ := hj.Hijack()
					conn.Close()
				}
				return
			}
		}
		handler.ServeHTTP(w, r)
	}))
	t.Cleanup(rs.srv.Close)
	return rs
}

// startBridge runs the proxy between an in-memory client transport and the
// remote server, returning the client-side transport and a done channel.
func startBridge(t *testing.T, cfg proxy.Config) (*mcp.InMemoryTransport, <-chan error, context.CancelFunc) {
	t.Helper()
	clientT, bridgeT := mcp.NewInMemoryTransports()
	cfg.Local = bridgeT
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- proxy.Run(ctx, cfg) }()
	return clientT, done, cancel
}

func TestBridgeForwardsToolsWithInjectedHeaders(t *testing.T) {
	rs := newRemoteServer(t, "s3cret")
	clientT, done, cancel := startBridge(t, proxy.Config{
		URL:     rs.srv.URL,
		Headers: map[string]string{"X-Record": "1"},
		HeaderFunc: func(context.Context) (map[string]string, error) {
			return map[string]string{"Authorization": "Bearer s3cret"}, nil
		},
	})
	defer cancel()

	client := mcp.NewClient(&mcp.Implementation{Name: "harness", Version: "1"}, nil)
	session, err := client.Connect(context.Background(), clientT, nil)
	if err != nil {
		t.Fatalf("connect through bridge: %v", err)
	}

	tools, err := session.ListTools(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	var names []string
	for _, tool := range tools.Tools {
		names = append(names, tool.Name)
	}
	if !strings.Contains(strings.Join(names, ","), "echo") {
		t.Fatalf("tools through bridge = %v", names)
	}

	res, err := session.CallTool(context.Background(), &mcp.CallToolParams{Name: "echo", Arguments: map[string]any{"text": "hi"}})
	if err != nil {
		t.Fatal(err)
	}
	if got := res.Content[0].(*mcp.TextContent).Text; got != "echo: hi" {
		t.Fatalf("tool result = %q", got)
	}

	// Closing the client ends the session; the bridge must return nil.
	session.Close()
	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("bridge returned %v after client hangup, want nil", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("bridge did not stop after the client closed")
	}

	// Requests after initialize carry the negotiated protocol version.
	rs.mu.Lock()
	defer rs.mu.Unlock()
	if len(rs.calls) < 2 {
		t.Fatalf("expected initialize plus later calls to be recorded, got %d", len(rs.calls))
	}
	last := rs.calls[len(rs.calls)-1]
	if last.Get("MCP-Protocol-Version") == "" {
		t.Fatalf("later requests must carry MCP-Protocol-Version, headers: %v", last)
	}
	if last.Get("Authorization") != "Bearer s3cret" {
		t.Fatalf("Authorization not injected: %v", last)
	}
}

func TestBridgeSurfacesRejectedCredentials(t *testing.T) {
	rs := newRemoteServer(t, "right")
	clientT, done, cancel := startBridge(t, proxy.Config{
		URL:     rs.srv.URL,
		Headers: map[string]string{"Authorization": "Bearer wrong"},
	})
	defer cancel()

	client := mcp.NewClient(&mcp.Implementation{Name: "harness", Version: "1"}, nil)
	ctx, ctxCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer ctxCancel()
	if _, err := client.Connect(ctx, clientT, nil); err == nil {
		t.Fatal("connect must fail when the remote rejects the token")
	}
	select {
	case err := <-done:
		if !errors.Is(err, proxy.ErrUnauthorized) {
			t.Fatalf("bridge error = %v, want ErrUnauthorized", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("bridge did not stop after a rejected request")
	}
}

func TestBridgeStopsOnContextCancel(t *testing.T) {
	rs := newRemoteServer(t, "tok")
	_, done, cancel := startBridge(t, proxy.Config{
		URL:     rs.srv.URL,
		Headers: map[string]string{"Authorization": "Bearer tok"},
	})
	cancel()
	select {
	case err := <-done:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("bridge error = %v, want context.Canceled", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("bridge did not stop on cancel")
	}
}

func TestRunValidatesConfig(t *testing.T) {
	if err := proxy.Run(context.Background(), proxy.Config{}); err == nil {
		t.Fatal("empty URL must be rejected")
	}
	err := proxy.Run(context.Background(), proxy.Config{
		URL:        "http://127.0.0.1:1",
		HeaderFunc: func(context.Context) (map[string]string, error) { return nil, errors.New("no token") },
	})
	if err == nil || !strings.Contains(err.Error(), "no token") {
		t.Fatalf("HeaderFunc error must be surfaced, got %v", err)
	}
}

func connect(t *testing.T, clientT *mcp.InMemoryTransport) *mcp.ClientSession {
	t.Helper()
	client := mcp.NewClient(&mcp.Implementation{Name: "harness", Version: "1"}, nil)
	session, err := client.Connect(context.Background(), clientT, nil)
	if err != nil {
		t.Fatalf("connect through bridge: %v", err)
	}
	return session
}

func TestBridgeRunsCallsConcurrently(t *testing.T) {
	rs := newRemoteServer(t, "tok")
	clientT, _, cancel := startBridge(t, proxy.Config{URL: rs.srv.URL, Headers: map[string]string{"Authorization": "Bearer tok"}})
	defer cancel()
	session := connect(t, clientT)
	defer session.Close()

	start := time.Now()
	var wg sync.WaitGroup
	for i := 0; i < 3; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if _, err := session.CallTool(context.Background(), &mcp.CallToolParams{Name: "sleep", Arguments: map[string]any{"millis": 300}}); err != nil {
				t.Error(err)
			}
		}()
	}
	wg.Wait()
	if elapsed := time.Since(start); elapsed > 700*time.Millisecond {
		t.Fatalf("three 300ms calls took %v; calls must not be serialised by the bridge", elapsed)
	}
}

func TestBridgeCancellationOvertakesInFlightCall(t *testing.T) {
	rs := newRemoteServer(t, "tok")
	clientT, _, cancel := startBridge(t, proxy.Config{URL: rs.srv.URL, Headers: map[string]string{"Authorization": "Bearer tok"}})
	defer cancel()
	session := connect(t, clientT)
	defer session.Close()

	ctx, cancelCall := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() {
		_, err := session.CallTool(ctx, &mcp.CallToolParams{Name: "sleep", Arguments: map[string]any{"millis": 5000}})
		done <- err
	}()
	time.Sleep(100 * time.Millisecond)
	cancelCall()
	select {
	case err := <-done:
		if err == nil {
			t.Fatal("cancelled call should fail")
		}
	case <-time.After(3 * time.Second):
		t.Fatal("cancelled call did not return")
	}
	// A new call must go through promptly even though the sleeping call is
	// still owned by the remote.
	start := time.Now()
	if _, err := session.CallTool(context.Background(), &mcp.CallToolParams{Name: "echo", Arguments: map[string]any{"text": "after"}}); err != nil {
		t.Fatal(err)
	}
	if elapsed := time.Since(start); elapsed > 2*time.Second {
		t.Fatalf("follow-up call waited %v behind the cancelled call", elapsed)
	}
}

func TestBridgeTransientRejectionFailsOnlyThatCall(t *testing.T) {
	rs := newRemoteServer(t, "tok")
	clientT, done, cancel := startBridge(t, proxy.Config{URL: rs.srv.URL, Headers: map[string]string{"Authorization": "Bearer tok"}})
	defer cancel()
	session := connect(t, clientT)

	// First call to "flaky" is answered 429 by the remote; the client must
	// get an error for that call and the session must survive.
	if _, err := session.CallTool(context.Background(), &mcp.CallToolParams{Name: "flaky"}); err == nil {
		t.Fatal("expected the rejected call to fail")
	}
	res, err := session.CallTool(context.Background(), &mcp.CallToolParams{Name: "echo", Arguments: map[string]any{"text": "still up"}})
	if err != nil {
		t.Fatalf("session should survive a transient rejection: %v", err)
	}
	if res.Content[0].(*mcp.TextContent).Text != "echo: still up" {
		t.Fatalf("unexpected result %+v", res)
	}
	session.Close()
	if err := <-done; err != nil {
		t.Fatalf("bridge returned %v after clean hangup", err)
	}
}

func TestBridgeRemoteFailureIsAnError(t *testing.T) {
	rs := newRemoteServer(t, "tok")
	clientT, done, cancel := startBridge(t, proxy.Config{URL: rs.srv.URL, Headers: map[string]string{"Authorization": "Bearer tok"}})
	defer cancel()
	session := connect(t, clientT)
	defer session.Close()

	// The remote answers the call with 200 headers and then kills the
	// connection. Two outcomes are acceptable, and both must be honest:
	// the SDK gives up on the remote connection, in which case the bridge
	// exits with an error; or it reports a per-call failure and keeps the
	// session, in which case the next call must still work.
	_, callErr := session.CallTool(context.Background(), &mcp.CallToolParams{Name: "drop"})
	if callErr == nil {
		t.Fatal("a dropped response must fail the call")
	}
	select {
	case err := <-done:
		if err == nil {
			t.Fatal("bridge exited with nil although the remote connection failed")
		}
		if !strings.Contains(err.Error(), "remote") {
			t.Fatalf("remote failure must be attributed to the remote, got %v", err)
		}
	case <-time.After(1500 * time.Millisecond):
		res, err := session.CallTool(context.Background(), &mcp.CallToolParams{Name: "echo", Arguments: map[string]any{"text": "alive"}})
		if err != nil {
			t.Fatalf("session was kept but is unusable: %v", err)
		}
		if res.Content[0].(*mcp.TextContent).Text != "echo: alive" {
			t.Fatalf("unexpected result %+v", res)
		}
	}
}

func TestBridgeRemoteGoneBeforeCallIsPerCallError(t *testing.T) {
	rs := newRemoteServer(t, "tok")
	clientT, done, cancel := startBridge(t, proxy.Config{URL: rs.srv.URL, Headers: map[string]string{"Authorization": "Bearer tok"}})
	defer cancel()
	session := connect(t, clientT)

	rs.srv.CloseClientConnections()
	rs.srv.Close()
	if _, err := session.CallTool(context.Background(), &mcp.CallToolParams{Name: "echo", Arguments: map[string]any{"text": "x"}}); err == nil {
		t.Fatal("call to a dead remote must fail")
	}
	// The client decides to hang up; that is a clean local exit.
	session.Close()
	select {
	case err := <-done:
		if err != nil && !strings.Contains(err.Error(), "remote") {
			t.Fatalf("unexpected error class: %v", err)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("bridge did not stop")
	}
}

func TestBridgeMidSessionCancel(t *testing.T) {
	rs := newRemoteServer(t, "tok")
	clientT, done, cancel := startBridge(t, proxy.Config{URL: rs.srv.URL, Headers: map[string]string{"Authorization": "Bearer tok"}})
	session := connect(t, clientT)
	defer session.Close()
	go session.CallTool(context.Background(), &mcp.CallToolParams{Name: "sleep", Arguments: map[string]any{"millis": 5000}})
	time.Sleep(100 * time.Millisecond)
	cancel()
	select {
	case err := <-done:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("bridge error = %v, want context.Canceled", err)
		}
	case <-time.After(8 * time.Second):
		t.Fatal("bridge did not stop on mid-session cancel")
	}
}

func TestReservedHeadersRejected(t *testing.T) {
	err := proxy.Run(context.Background(), proxy.Config{URL: "http://127.0.0.1:1", Headers: map[string]string{"Mcp-Session-Id": "x"}})
	if err == nil || !strings.Contains(err.Error(), "Mcp-Session-Id") {
		t.Fatalf("reserved header must be rejected, got %v", err)
	}
}

func TestUnauthorizedCarriesChallenge(t *testing.T) {
	rs := newRemoteServer(t, "right")
	clientT, done, cancel := startBridge(t, proxy.Config{URL: rs.srv.URL, Headers: map[string]string{"Authorization": "Bearer wrong"}})
	defer cancel()
	client := mcp.NewClient(&mcp.Implementation{Name: "harness", Version: "1"}, nil)
	ctx, ctxCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer ctxCancel()
	_, _ = client.Connect(ctx, clientT, nil)
	err := <-done
	if !errors.Is(err, proxy.ErrUnauthorized) || !strings.Contains(err.Error(), `realm="test"`) {
		t.Fatalf("want ErrUnauthorized with the WWW-Authenticate challenge, got %v", err)
	}
}

// TestBridgeDeliversResponseAfterClientHalfClose models a client that writes
// one request, closes its output, and waits for the answer (echo | bridge).
func TestBridgeDeliversResponseAfterClientHalfClose(t *testing.T) {
	rs := newRemoteServer(t, "tok")
	toBridgeR, toBridgeW := io.Pipe()
	fromBridgeR, fromBridgeW := io.Pipe()
	done := make(chan error, 1)
	go func() {
		done <- proxy.Run(context.Background(), proxy.Config{
			URL:     rs.srv.URL,
			Headers: map[string]string{"Authorization": "Bearer tok"},
			Local:   &mcp.IOTransport{Reader: toBridgeR, Writer: fromBridgeW},
		})
	}()

	init := `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-06-18","capabilities":{},"clientInfo":{"name":"t","version":"0"}}}` + "\n"
	if _, err := io.WriteString(toBridgeW, init); err != nil {
		t.Fatal(err)
	}
	toBridgeW.Close() // half-close: no more requests, still reading

	line := make(chan string, 1)
	go func() {
		buf := make([]byte, 64<<10)
		n, _ := fromBridgeR.Read(buf)
		line <- string(buf[:n])
	}()
	select {
	case got := <-line:
		if !strings.Contains(got, `"protocolVersion"`) {
			t.Fatalf("expected the initialize result after half-close, got %q", got)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("response was not delivered after the client half-closed")
	}
	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("bridge returned %v, want nil", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("bridge did not exit after draining")
	}
}
