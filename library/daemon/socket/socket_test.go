package socket_test

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/sairaph/mcp-wizard/daemon/socket"
)

func TestNewServerAndSocketPath(t *testing.T) {
	dir := t.TempDir()
	s := socket.New(dir, "test")
	if s.SocketPath() != filepath.Join(dir, "test.sock") {
		t.Fatalf("unexpected socket path: %s", s.SocketPath())
	}
}

func TestOpenAndClose(t *testing.T) {
	dir := t.TempDir()
	s := socket.New(dir, "test")
	if err := s.Open(); err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	s.Close()
	if _, err := os.Stat(s.SocketPath()); !os.IsNotExist(err) {
		t.Fatal("socket file should be removed after Close")
	}
}

func TestErrAlreadyRunning(t *testing.T) {
	dir := t.TempDir()
	s1 := socket.New(dir, "test")
	if err := s1.Open(); err != nil {
		t.Fatalf("first Open failed: %v", err)
	}
	defer s1.Close()

	s2 := socket.New(dir, "test")
	if err := s2.Open(); err == nil {
		t.Fatal("expected error for second Open, got nil")
	}
}

func TestHandlerRegistration(t *testing.T) {
	dir := t.TempDir()
	s := socket.New(dir, "test")
	s.Handle("ping", func(ctx context.Context, params json.RawMessage) (any, error) {
		return "pong", nil
	})
	// No error expected; registration just stores the handler
}

func TestClientServerRoundTrip(t *testing.T) {
	dir := t.TempDir()
	s := socket.New(dir, "rpc")

	s.Handle("ping", func(ctx context.Context, params json.RawMessage) (any, error) {
		return "pong", nil
	})
	s.Handle("echo", func(ctx context.Context, params json.RawMessage) (any, error) {
		var msg string
		if err := json.Unmarshal(params, &msg); err != nil {
			return nil, err
		}
		return msg, nil
	})

	if err := s.Open(); err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	defer s.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go s.Serve(ctx)

	c, err := socket.Dial(s.SocketPath())
	if err != nil {
		t.Fatalf("Dial failed: %v", err)
	}
	defer c.Close()

	var result string
	if err := c.Call(ctx, "ping", nil, &result); err != nil {
		t.Fatalf("Call ping failed: %v", err)
	}
	if result != "pong" {
		t.Fatalf("expected pong, got %s", result)
	}

	var echoResult string
	if err := c.Call(ctx, "echo", "hello world", &echoResult); err != nil {
		t.Fatalf("Call echo failed: %v", err)
	}
	if echoResult != "hello world" {
		t.Fatalf("expected 'hello world', got %s", echoResult)
	}
}

func TestUnknownMethod(t *testing.T) {
	dir := t.TempDir()
	s := socket.New(dir, "unknown")

	if err := s.Open(); err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	defer s.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go s.Serve(ctx)

	c, err := socket.Dial(s.SocketPath())
	if err != nil {
		t.Fatalf("Dial failed: %v", err)
	}
	defer c.Close()

	var result string
	err = c.Call(ctx, "nonexistent", nil, &result)
	if err == nil {
		t.Fatal("expected error for unknown method, got nil")
	}
}

func TestHandlerError(t *testing.T) {
	dir := t.TempDir()
	s := socket.New(dir, "err")
	s.Handle("fail", func(ctx context.Context, params json.RawMessage) (any, error) {
		return nil, errors.New("handler failed")
	})

	if err := s.Open(); err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	defer s.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go s.Serve(ctx)

	c, err := socket.Dial(s.SocketPath())
	if err != nil {
		t.Fatalf("Dial failed: %v", err)
	}
	defer c.Close()

	var result string
	err = c.Call(ctx, "fail", nil, &result)
	if err == nil {
		t.Fatal("expected error for failing handler, got nil")
	}
}
