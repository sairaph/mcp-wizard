package secret_test

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/sairaph/mcp-wizard/secret"
)

func TestFileStoreRoundTrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "creds.json")
	store := secret.NewFileStore(path)

	sess := secret.NewSession()
	sess.Set("email", "user@example.com")
	sess.Set("token", "abc123")
	sess.Set("count", 42)

	ctx := context.Background()
	if err := store.Save(ctx, sess); err != nil {
		t.Fatalf("Save: %v", err)
	}

	got, ok, err := store.Load(ctx)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if !ok {
		t.Fatal("Load returned ok=false, want true")
	}
	if got.GetString("email") != "user@example.com" {
		t.Errorf("email = %q, want %q", got.GetString("email"), "user@example.com")
	}
	if got.GetString("token") != "abc123" {
		t.Errorf("token = %q, want %q", got.GetString("token"), "abc123")
	}
	if got.Get("count") != float64(42) {
		t.Errorf("count = %v, want %v", got.Get("count"), float64(42))
	}
}

func TestFileStoreNotFound(t *testing.T) {
	dir := t.TempDir()
	store := secret.NewFileStore(filepath.Join(dir, "nonexistent.json"))

	_, ok, err := store.Load(context.Background())
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if ok {
		t.Fatal("Load returned ok=true, want false")
	}
}

func TestFileStoreDelete(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "creds.json")
	store := secret.NewFileStore(path)

	sess := secret.NewSession()
	sess.Set("x", "y")
	if err := store.Save(context.Background(), sess); err != nil {
		t.Fatalf("Save: %v", err)
	}
	if err := store.Delete(context.Background()); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatal("file still exists after Delete")
	}
}

func TestFileStoreDeleteNonexistent(t *testing.T) {
	dir := t.TempDir()
	store := secret.NewFileStore(filepath.Join(dir, "nope.json"))

	if err := store.Delete(context.Background()); err != nil {
		t.Fatalf("Delete on nonexistent file: %v", err)
	}
}

func TestFileStorePermissions(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "sub", "creds.json")
	store := secret.NewFileStore(path)

	sess := secret.NewSession()
	sess.Set("key", "val")
	if err := store.Save(context.Background(), sess); err != nil {
		t.Fatalf("Save: %v", err)
	}

	// Check directory permission is 0700.
	info, err := os.Stat(filepath.Join(dir, "sub"))
	if err != nil {
		t.Fatalf("Stat dir: %v", err)
	}
	if perm := info.Mode().Perm(); perm != 0700 {
		t.Errorf("directory permissions = %o, want 0700", perm)
	}

	// Check file permission is 0600.
	info, err = os.Stat(path)
	if err != nil {
		t.Fatalf("Stat file: %v", err)
	}
	if perm := info.Mode().Perm(); perm != 0600 {
		t.Errorf("file permissions = %o, want 0600", perm)
	}
}

func TestEnvStore(t *testing.T) {
	t.Setenv("MCPW_EMAIL", "env@example.com")
	t.Setenv("MCPW_TOKEN", "envtoken")

	mapping := map[string]string{
		"email": "MCPW_EMAIL",
		"token": "MCPW_TOKEN",
		"other": "MCPW_OTHER",
	}
	store := secret.NewEnvStore(mapping)

	sess, ok, err := store.Load(context.Background())
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if !ok {
		t.Fatal("Load returned ok=false, want true")
	}
	if sess.GetString("email") != "env@example.com" {
		t.Errorf("email = %q, want %q", sess.GetString("email"), "env@example.com")
	}
	if sess.GetString("token") != "envtoken" {
		t.Errorf("token = %q, want %q", sess.GetString("token"), "envtoken")
	}
	if sess.GetString("other") != "" {
		t.Errorf("other = %q, want empty", sess.GetString("other"))
	}
}

func TestEnvStoreSaveDeleteNoop(t *testing.T) {
	store := secret.NewEnvStore(map[string]string{"k": "V"})

	if err := store.Save(context.Background(), secret.NewSession()); err != nil {
		t.Fatalf("Save: %v", err)
	}
	if err := store.Delete(context.Background()); err != nil {
		t.Fatalf("Delete: %v", err)
	}
}

func TestSessionGetSet(t *testing.T) {
	s := secret.NewSession()

	if v := s.Get("missing"); v != nil {
		t.Errorf("Get missing = %v, want nil", v)
	}
	if v := s.GetString("missing"); v != "" {
		t.Errorf("GetString missing = %q, want empty", v)
	}

	s.Set("email", "a@b.com")
	if v := s.GetString("email"); v != "a@b.com" {
		t.Errorf("GetString email = %q, want %q", v, "a@b.com")
	}

	s.Set("count", 7)
	if v := s.Get("count"); v != 7 {
		t.Errorf("Get count = %v, want 7", v)
	}

	// GetString on non-string key returns empty.
	if v := s.GetString("count"); v != "" {
		t.Errorf("GetString count = %q, want empty", v)
	}
}

func TestSessionClone(t *testing.T) {
	s := secret.NewSession()
	s.Set("a", "1")
	s.Set("b", "2")

	c := s.Clone()
	c.Set("a", "changed")
	c.Set("c", "3")

	if s.GetString("a") != "1" {
		t.Errorf("original a = %q, want %q", s.GetString("a"), "1")
	}
	if s.GetString("c") != "" {
		t.Errorf("original c = %q, want empty", s.GetString("c"))
	}
}

func TestRetryableError(t *testing.T) {
	err := secret.RetryableError{Message: "login failed"}
	if err.Error() != "login failed" {
		t.Errorf("Error() = %q, want %q", err.Error(), "login failed")
	}

	var target secret.RetryableError
	if !errors.As(err, &target) {
		t.Fatal("errors.As failed for RetryableError")
	}
}
