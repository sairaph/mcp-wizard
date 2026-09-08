//go:build !windows

package update

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"syscall"
	"testing"
)

func TestSwapFileCrossDeviceStagesBesideTarget(t *testing.T) {
	dir := t.TempDir()
	source := filepath.Join(dir, "download")
	target := filepath.Join(dir, "bin", "tool")
	os.MkdirAll(filepath.Dir(target), 0755)
	os.WriteFile(source, []byte("new"), 0600)
	os.WriteFile(target, []byte("old"), 0755)

	orig := rename
	defer func() { rename = orig }()
	calls := 0
	rename = func(from, to string) error {
		calls++
		if from == source {
			return &os.LinkError{Op: "rename", Old: from, New: to, Err: syscall.EXDEV}
		}
		return os.Rename(from, to)
	}

	if err := swapFile(source, target); err != nil {
		t.Fatal(err)
	}
	got, _ := os.ReadFile(target)
	if string(got) != "new" {
		t.Fatalf("target = %q", got)
	}
	if _, err := os.Stat(source); !os.IsNotExist(err) {
		t.Fatal("source should be removed after a cross-device copy")
	}
	leftovers, _ := filepath.Glob(filepath.Join(filepath.Dir(target), "tool.staging-*"))
	if len(leftovers) != 0 {
		t.Fatalf("staging file left behind: %v", leftovers)
	}
	if calls < 2 {
		t.Fatalf("expected the staged rename after the EXDEV, got %d rename calls", calls)
	}
}

func TestSwapFileCopyFailureLeavesTargetIntact(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "tool")
	os.WriteFile(target, []byte("old"), 0755)

	orig := rename
	defer func() { rename = orig }()
	rename = func(from, to string) error {
		return &os.LinkError{Op: "rename", Old: from, New: to, Err: syscall.EXDEV}
	}
	// The source does not exist, so the staged copy fails.
	err := swapFile(filepath.Join(dir, "missing"), target)
	if err == nil {
		t.Fatal("expected copy failure")
	}
	got, _ := os.ReadFile(target)
	if string(got) != "old" {
		t.Fatalf("target must be untouched on failure, got %q", got)
	}
}

func TestSwapFileOtherRenameErrorIsReturned(t *testing.T) {
	dir := t.TempDir()
	source := filepath.Join(dir, "src")
	target := filepath.Join(dir, "dst")
	os.WriteFile(source, []byte("x"), 0600)
	orig := rename
	defer func() { rename = orig }()
	rename = func(from, to string) error { return errors.New("boom") }
	if err := swapFile(source, target); err == nil || err.Error() != "rename: boom" {
		t.Fatalf("err = %v", err)
	}
}

func TestSwapFromMissingTempFile(t *testing.T) {
	dir := t.TempDir()
	err := SwapFrom(context.Background(), filepath.Join(dir, "nope"), Options{InstallDir: dir, BinaryName: "tool"})
	if err == nil {
		t.Fatal("missing temp file must be an error")
	}
	if _, statErr := os.Stat(filepath.Join(dir, "tool")); !os.IsNotExist(statErr) {
		t.Fatal("nothing must be installed")
	}
}
