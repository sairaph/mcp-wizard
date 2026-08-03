package daemon_test

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/sairaph/mcp-wizard/daemon"
)

func TestIsRunningReturnsFalseForNonExistentLockFile(t *testing.T) {
	if daemon.IsRunning("/nonexistent/path/lock") {
		t.Fatal("IsRunning should return false for non-existent lock file")
	}
}

func TestOpenAndClose(t *testing.T) {
	dir := t.TempDir()
	opts := daemon.Options{
		LockFile: filepath.Join(dir, "lock"),
		PIDFile:  filepath.Join(dir, "daemon.pid"),
	}

	inst, err := daemon.Open(context.Background(), opts)
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}

	if _, err := os.Stat(opts.PIDFile); err != nil {
		t.Fatalf("PID file should exist: %v", err)
	}

	inst.Close()

	if _, err := os.Stat(opts.PIDFile); !os.IsNotExist(err) {
		t.Fatal("PID file should be removed after Close")
	}
}

func TestErrAlreadyRunningWhenOpeningTwice(t *testing.T) {
	dir := t.TempDir()
	opts := daemon.Options{
		LockFile: filepath.Join(dir, "lock"),
	}

	inst, err := daemon.Open(context.Background(), opts)
	if err != nil {
		t.Fatalf("first Open failed: %v", err)
	}
	defer inst.Close()

	_, err = daemon.Open(context.Background(), opts)
	if !errors.Is(err, daemon.ErrAlreadyRunning) {
		t.Fatalf("expected ErrAlreadyRunning, got %v", err)
	}
}

func TestServeRunsFunctionAndReturnsItsError(t *testing.T) {
	dir := t.TempDir()
	opts := daemon.Options{
		LockFile: filepath.Join(dir, "lock"),
	}

	inst, err := daemon.Open(context.Background(), opts)
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	defer inst.Close()

	sentinel := errors.New("sentinel error")
	got := inst.Serve(context.Background(), func(ctx context.Context) error {
		return sentinel
	})
	if !errors.Is(got, sentinel) {
		t.Fatalf("expected sentinel error, got %v", got)
	}
}

func TestDoneChannelClosesAfterServeReturns(t *testing.T) {
	dir := t.TempDir()
	opts := daemon.Options{
		LockFile: filepath.Join(dir, "lock"),
	}

	inst, err := daemon.Open(context.Background(), opts)
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	defer inst.Close()

	done := inst.Done()
	select {
	case <-done:
		t.Fatal("Done channel should not be closed before Serve returns")
	default:
	}

	inst.Serve(context.Background(), func(ctx context.Context) error {
		return nil
	})

	select {
	case <-done:
	default:
		t.Fatal("Done channel should be closed after Serve returns")
	}
}

func TestStaleAfterIsPositive(t *testing.T) {
	if daemon.StaleAfter <= 0 {
		t.Fatal("StaleAfter must be positive")
	}
}

func TestPIDFileWrittenAndCleanedUp(t *testing.T) {
	dir := t.TempDir()
	opts := daemon.Options{
		LockFile: filepath.Join(dir, "lock"),
		PIDFile:  filepath.Join(dir, "daemon.pid"),
	}

	inst, err := daemon.Open(context.Background(), opts)
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}

	data, err := os.ReadFile(opts.PIDFile)
	if err != nil {
		t.Fatalf("PID file should be readable: %v", err)
	}
	if len(data) == 0 {
		t.Fatal("PID file should contain content")
	}

	inst.Close()

	if _, err := os.Stat(opts.PIDFile); !os.IsNotExist(err) {
		t.Fatal("PID file should be removed after Close")
	}
}

func TestIsRunningStaleLock(t *testing.T) {
	dir := t.TempDir()
	lockFile := filepath.Join(dir, "lock")

	if err := os.WriteFile(lockFile, []byte("stale"), 0644); err != nil {
		t.Fatal(err)
	}

	past := time.Now().Add(-(daemon.StaleAfter + time.Minute))
	if err := os.Chtimes(lockFile, past, past); err != nil {
		t.Fatal(err)
	}

	if daemon.IsRunning(lockFile) {
		t.Fatal("IsRunning should return false for a stale lock file")
	}
}
