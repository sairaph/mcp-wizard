package lock_test

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/sairaph/mcp-wizard/daemon/lock"
)

func TestIsRunningReturnsFalseForNonExistentLockFile(t *testing.T) {
	if lock.IsRunning("/nonexistent/path/lock") {
		t.Fatal("IsRunning should return false for non-existent lock file")
	}
}

func TestOpenAndClose(t *testing.T) {
	dir := t.TempDir()
	opts := lock.Options{
		LockFile: filepath.Join(dir, "lock"),
		PIDFile:  filepath.Join(dir, "lock.pid"),
	}

	inst, err := lock.Open(opts)
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
	opts := lock.Options{
		LockFile: filepath.Join(dir, "lock"),
	}

	inst, err := lock.Open(opts)
	if err != nil {
		t.Fatalf("first Open failed: %v", err)
	}
	defer inst.Close()

	_, err = lock.Open(opts)
	if !errors.Is(err, lock.ErrAlreadyRunning) {
		t.Fatalf("expected ErrAlreadyRunning, got %v", err)
	}
}

func TestStaleAfterIsPositive(t *testing.T) {
	if lock.StaleAfter <= 0 {
		t.Fatal("StaleAfter must be positive")
	}
}

func TestPIDFileWrittenAndCleanedUp(t *testing.T) {
	dir := t.TempDir()
	opts := lock.Options{
		LockFile: filepath.Join(dir, "lock"),
		PIDFile:  filepath.Join(dir, "lock.pid"),
	}

	inst, err := lock.Open(opts)
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

	past := time.Now().Add(-(lock.StaleAfter + time.Minute))
	if err := os.Chtimes(lockFile, past, past); err != nil {
		t.Fatal(err)
	}

	if lock.IsRunning(lockFile) {
		t.Fatal("IsRunning should return false for a stale lock file")
	}
}
