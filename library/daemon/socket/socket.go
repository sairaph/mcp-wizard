package socket

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/gofrs/flock"
	"github.com/sairaph/mcp-wizard/daemon/rpc"
)

type Handler func(ctx context.Context, params json.RawMessage) (any, error)

type Server struct {
	socketPath string
	lockPath   string
	listener   net.Listener
	handlers   map[string]Handler
	flock        *flock.Flock
	mu           sync.RWMutex
	wg           sync.WaitGroup
	cleanupOnce  sync.Once
	ctx          context.Context
	cancel       context.CancelFunc
}

func New(socketDir, name string) *Server {
	return &Server{
		socketPath: filepath.Join(socketDir, name+".sock"),
		lockPath:   filepath.Join(socketDir, "lock"),
		handlers:   make(map[string]Handler),
	}
}

func (s *Server) Handle(method string, handler Handler) {
	if handler == nil {
		panic("socket: Handle requires a non-nil handler")
	}
	s.mu.Lock()
	s.handlers[method] = handler
	s.mu.Unlock()
}

func (s *Server) Open() error {
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
	os.Remove(s.socketPath)
	listener, err := net.Listen("unix", s.socketPath)
	if err != nil {
		s.flock.Unlock()
		return fmt.Errorf("listen: %w", err)
	}
	s.listener = listener
	s.ctx, s.cancel = context.WithCancel(context.Background())
	return nil
}

func (s *Server) Serve(ctx context.Context) error {
	if s.listener == nil {
		return fmt.Errorf("socket: Serve called before Open")
	}
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
	dec := json.NewDecoder(conn)
	enc := json.NewEncoder(conn)
	for {
		select {
		case <-ctx.Done():
			return
		case <-s.ctx.Done():
			return
		default:
		}
		conn.SetReadDeadline(time.Now().Add(1 * time.Second))
		var req rpc.Request
		if err := dec.Decode(&req); err != nil {
			if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
				continue
			}
			// Send error response before closing so the client doesn't hang.
			errResp := rpc.NewErrorResponse(0, rpc.CodeParse, "invalid request: "+err.Error())
			enc.Encode(errResp)
			return
		}
		resp := s.dispatch(ctx, req)
		conn.SetWriteDeadline(time.Now().Add(5 * time.Second))
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
	result, err := handler(ctx, req.Params)
	if err != nil {
		return rpc.NewErrorResponse(req.ID, rpc.CodeInternal, err.Error())
	}
	resp, rpcErr := rpc.NewResponse(req.ID, result)
	if rpcErr != nil {
		return rpc.NewErrorResponse(req.ID, rpc.CodeInternal, rpcErr.Error())
	}
	return resp
}

func (s *Server) cleanup() {
	s.cleanupOnce.Do(func() {
		if s.listener != nil {
			s.listener.Close()
		}
		os.Remove(s.socketPath)
		if s.flock != nil {
			s.flock.Unlock()
		}
	})
}

func (s *Server) Close() {
	if s.cancel != nil {
		s.cancel()
	}
	s.cleanup()
	s.wg.Wait()
}

func (s *Server) SocketPath() string { return s.socketPath }
