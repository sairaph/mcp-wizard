// Package socket provides a Unix-socket daemon with JSON-RPC IPC: a Server
// that owns the socket and a file lock, and a Client that calls it.
package socket

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"

	"github.com/gofrs/flock"
	"github.com/sairaph/mcp-wizard/daemon/rpc"
)

// Handler serves one RPC method. It receives the raw params and returns a
// result that is JSON-encoded into the response.
type Handler func(ctx context.Context, params json.RawMessage) (any, error)

// Server listens on a Unix socket and dispatches JSON-RPC requests to
// registered handlers.
//
// Lifecycle: Open acquires the lock and socket, Serve accepts connections
// until the context is cancelled or Close is called, and Serve returns only
// after every in-flight handler has finished. Close is safe to call from
// inside a handler (for example a "shutdown" RPC): it signals shutdown and
// returns immediately; the goroutine blocked in Serve does the waiting.
type Server struct {
	socketPath  string
	lockPath    string
	listener    net.Listener
	handlers    map[string]Handler
	flock       *flock.Flock
	mu          sync.RWMutex
	wg          sync.WaitGroup
	cleanupOnce sync.Once
	serving     atomic.Bool
	ctx         context.Context
	cancel      context.CancelFunc
}

// New creates a Server for name inside socketDir. The socket is
// socketDir/name.sock and the lock is socketDir/lock.
func New(socketDir, name string) *Server {
	return &Server{
		socketPath: filepath.Join(socketDir, name+".sock"),
		lockPath:   filepath.Join(socketDir, "lock"),
		handlers:   make(map[string]Handler),
	}
}

// Handle registers handler for method. It panics on a nil handler.
func (s *Server) Handle(method string, handler Handler) {
	if handler == nil {
		panic("socket: Handle requires a non-nil handler")
	}
	s.mu.Lock()
	s.handlers[method] = handler
	s.mu.Unlock()
}

// Open acquires the daemon lock and starts listening on the socket. A server
// may be reopened after Close, but only once Serve has returned.
func (s *Server) Open() error {
	if s.serving.Load() {
		return fmt.Errorf("socket: Open called while Serve is running")
	}
	if err := os.MkdirAll(filepath.Dir(s.socketPath), 0700); err != nil {
		return fmt.Errorf("create socket dir: %w", err)
	}
	s.flock = flock.New(s.lockPath)
	locked, err := s.flock.TryLock()
	if err != nil {
		return fmt.Errorf("acquire lock: %w", err)
	}
	if !locked {
		return fmt.Errorf("daemon is already running")
	}
	if err := os.Remove(s.socketPath); err != nil && !os.IsNotExist(err) {
		s.flock.Unlock()
		s.flock = nil
		return fmt.Errorf("remove stale socket: %w", err)
	}
	listener, err := net.Listen("unix", s.socketPath)
	if err != nil {
		s.flock.Unlock()
		s.flock = nil
		return fmt.Errorf("listen: %w", err)
	}
	s.listener = listener
	s.cleanupOnce = sync.Once{}
	s.ctx, s.cancel = context.WithCancel(context.Background())
	return nil
}

// Serve accepts connections until ctx is cancelled or Close is called. It
// returns once the listener is closed and all connection handlers have
// finished. A nil error means an orderly shutdown.
func (s *Server) Serve(ctx context.Context) error {
	if s.listener == nil {
		return fmt.Errorf("socket: Serve called before Open")
	}
	if !s.serving.CompareAndSwap(false, true) {
		return fmt.Errorf("socket: Serve is already running")
	}
	defer s.serving.Store(false)
	// Close connections when the caller's context ends, not only on Close.
	stop := context.AfterFunc(ctx, s.cleanup)
	defer stop()
	defer s.wg.Wait()
	defer s.cleanup()
	for {
		conn, err := s.listener.Accept()
		if err != nil {
			select {
			case <-ctx.Done():
				return nil
			case <-s.ctx.Done():
				return nil
			default:
				return fmt.Errorf("accept: %w", err)
			}
		}
		s.wg.Add(1)
		go s.handleConn(ctx, conn)
	}
}

func (s *Server) handleConn(ctx context.Context, conn net.Conn) {
	defer s.wg.Done()
	defer conn.Close()

	// Handlers get a context that ends with the connection, so a handler
	// blocked on ctx.Done() is released by Close as well.
	hctx, hcancel := context.WithCancel(ctx)
	defer hcancel()

	// Watch for cancellation - close the connection to unblock Decode.
	done := make(chan struct{})
	defer close(done)
	go func() {
		select {
		case <-ctx.Done():
		case <-s.ctx.Done():
		case <-done:
			return
		}
		hcancel()
		conn.Close()
	}()

	dec := json.NewDecoder(conn)
	enc := json.NewEncoder(conn)
	for {
		var req rpc.Request
		if err := dec.Decode(&req); err != nil {
			return // connection closed or error
		}
		resp := s.dispatch(hctx, req)
		if err := enc.Encode(resp); err != nil {
			return
		}
	}
}

func (s *Server) dispatch(ctx context.Context, req rpc.Request) rpc.Response {
	s.mu.RLock()
	handler, ok := s.handlers[req.Method]
	s.mu.RUnlock()
	if !ok {
		return rpc.NewErrorResponse(req.ID, rpc.CodeMethod, "unknown method: "+req.Method)
	}
	result, err := safeCall(ctx, handler, req.Params)
	if err != nil {
		return rpc.NewErrorResponse(req.ID, rpc.CodeInternal, err.Error())
	}
	resp, rpcErr := rpc.NewResponse(req.ID, result)
	if rpcErr != nil {
		return rpc.NewErrorResponse(req.ID, rpc.CodeInternal, rpcErr.Error())
	}
	return resp
}

func safeCall(ctx context.Context, handler Handler, params json.RawMessage) (result any, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("handler panic: %v", r)
		}
	}()
	return handler(ctx, params)
}

// cleanup cancels in-flight connections, closes the listener, removes the
// socket file and releases the lock. It runs at most once per Open.
func (s *Server) cleanup() {
	s.cleanupOnce.Do(func() {
		if s.cancel != nil {
			s.cancel()
		}
		if s.listener != nil {
			s.listener.Close()
		}
		os.Remove(s.socketPath)
		if s.flock != nil {
			s.flock.Unlock()
		}
	})
}

// Close stops the server: it closes the listener, cancels every open
// connection, removes the socket file and releases the lock. It does not wait
// for handlers to finish; Serve returns once they have. It is safe to call
// more than once and from inside a handler.
func (s *Server) Close() {
	s.cleanup()
}

// SocketPath returns the path of the Unix socket.
func (s *Server) SocketPath() string { return s.socketPath }
