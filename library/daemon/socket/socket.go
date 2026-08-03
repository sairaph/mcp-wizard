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
	mu           sync.Mutex
	wg           sync.WaitGroup
	cleanupOnce  sync.Once
}

func New(socketDir, name string) *Server {
	return &Server{
		socketPath: filepath.Join(socketDir, name+".sock"),
		lockPath:   filepath.Join(socketDir, "lock"),
		handlers:   make(map[string]Handler),
	}
}

func (s *Server) Handle(method string, handler Handler) {
	s.handlers[method] = handler
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
		conn.SetReadDeadline(time.Now().Add(5 * time.Minute))
		var req rpc.Request
		if err := dec.Decode(&req); err != nil {
			return
		}
		resp := s.dispatch(ctx, req)
		enc.Encode(resp)
	}
}

func (s *Server) dispatch(ctx context.Context, req rpc.Request) rpc.Response {
	handler, ok := s.handlers[req.Method]
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
	if s.listener != nil {
		s.listener.Close()
	}
	s.wg.Wait()
	s.cleanupOnce.Do(func() {
		os.Remove(s.socketPath)
		if s.flock != nil {
			s.flock.Unlock()
		}
	})
}

func (s *Server) SocketPath() string { return s.socketPath }
