package lock_test

import (
	"errors"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"
	"strings"
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

func TestIsRunningUnlockedFileIsNotRunning(t *testing.T) {
	dir := t.TempDir()
	lockFile := filepath.Join(dir, "lock")

	// A leftover lock file from a crashed daemon holds no flock, so it must
	// not be mistaken for a running daemon.
	if err := os.WriteFile(lockFile, []byte("stale"), 0644); err != nil {
		t.Fatal(err)
	}
	past := time.Now().Add(-time.Hour)
	if err := os.Chtimes(lockFile, past, past); err != nil {
		t.Fatal(err)
	}

	if lock.IsRunning(lockFile) {
		t.Fatal("IsRunning should return false for an unlocked leftover lock file")
	}
}

// TestMain lets the test binary double as a daemon: when LOCK_TEST_CHILD is
// set it opens the lock named by the environment, prints a marker, and waits
// for SIGINT (or the parent to kill it).
func TestMain(m *testing.M) {
	if os.Getenv("LOCK_TEST_CHILD") == "" {
		os.Exit(m.Run())
	}
	// Register the handler before taking the lock: the parent treats the
	// held lock as "daemon is up" and may send SIGINT immediately.
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt)
	inst, err := lock.Open(lock.Options{
		LockFile: os.Getenv("LOCK_TEST_LOCK"),
		PIDFile:  os.Getenv("LOCK_TEST_PID"),
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, "child open:", err)
		os.Exit(1)
	}
	fmt.Println("child running")
	select {
	case <-sig:
	case <-time.After(30 * time.Second):
	}
	inst.Close()
	fmt.Println("child stopped")
	os.Exit(0)
}

func TestEnsureRunningStartsDaemonAndStopStopsIt(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("detached child + SIGINT test is Unix only")
	}
	dir := t.TempDir()
	opts := lock.Options{
		LockFile: filepath.Join(dir, "lock"),
		PIDFile:  filepath.Join(dir, "daemon.pid"),
		LogFile:  filepath.Join(dir, "logs", "daemon.log"),
	}
	exe, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("LOCK_TEST_CHILD", "1")
	t.Setenv("LOCK_TEST_LOCK", opts.LockFile)
	t.Setenv("LOCK_TEST_PID", opts.PIDFile)

	// The child exits inside TestMain before flags are parsed, so it needs
	// no arguments.
	if !lock.EnsureRunning(exe, nil, opts) {
		t.Fatal("EnsureRunning should report the daemon running after starting it")
	}
	t.Cleanup(func() { _ = lock.Stop(opts.PIDFile, opts.LockFile) })
	if !lock.IsRunning(opts.LockFile) {
		t.Fatal("child should hold the lock")
	}
	// Second call finds it already running and does not spawn another.
	if !lock.EnsureRunning(exe, nil, opts) {
		t.Fatal("EnsureRunning should be idempotent")
	}

	if err := lock.Stop(opts.PIDFile, opts.LockFile); err != nil {
		t.Fatalf("Stop: %v", err)
	}
	deadline := time.Now().Add(5 * time.Second)
	for lock.IsRunning(opts.LockFile) && time.Now().Before(deadline) {
		time.Sleep(50 * time.Millisecond)
	}
	if lock.IsRunning(opts.LockFile) {
		t.Fatal("daemon still holds the lock after Stop")
	}

	deadline = time.Now().Add(2 * time.Second)
	var log []byte
	for time.Now().Before(deadline) {
		log, _ = os.ReadFile(opts.LogFile)
		if strings.Contains(string(log), "child stopped") {
			break
		}
		time.Sleep(50 * time.Millisecond)
	}
	if !strings.Contains(string(log), "child running") || !strings.Contains(string(log), "child stopped") {
		t.Fatalf("LogFile should capture the daemon's stdout, got %q", log)
	}
}

func TestStopRejectsBadPIDFile(t *testing.T) {
	dir := t.TempDir()
	if err := lock.Stop("", filepath.Join(dir, "lock")); err == nil {
		t.Fatal("empty pidFile must be an error")
	}

	// A PID file without a held lock is stale: it is removed and nothing is
	// signalled, even though the PID (ours) is alive.
	stale := filepath.Join(dir, "stale.pid")
	if err := os.WriteFile(stale, []byte(fmt.Sprint(os.Getpid())), 0600); err != nil {
		t.Fatal(err)
	}
	if err := lock.Stop(stale, filepath.Join(dir, "unheld-lock")); !errors.Is(err, lock.ErrNotRunning) {
		t.Fatalf("expected ErrNotRunning for a stale PID file, got %v", err)
	}
	if _, err := os.Stat(stale); !os.IsNotExist(err) {
		t.Fatal("stale PID file should be removed")
	}
	pidFile := filepath.Join(dir, "pid")
	if err := os.WriteFile(pidFile, []byte("not-a-pid"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := lock.Stop(pidFile, ""); err == nil {
		t.Fatal("garbage pid must be an error")
	}
}
