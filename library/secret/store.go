package secret

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

// Store is the credential persistence interface.
type Store interface {
	// Save persists the session. The store decides the format and location.
	Save(ctx context.Context, s *Session) error

	// Load retrieves the stored session. The bool reports whether data existed.
	Load(ctx context.Context) (*Session, bool, error)

	// Delete removes stored credentials.
	Delete(ctx context.Context) error

	// Path returns the storage location for display (doctor, install summary).
	Path() string
}

// FileStore writes credentials as JSON at 0600 in a 0700 directory,
// using atomic temp-file + rename to prevent partial writes.
type FileStore struct {
	path string
}

// NewFileStore creates a FileStore at the given path.
// The parent directory is created with 0700 permissions if it doesn't exist.
func NewFileStore(path string) *FileStore {
	return &FileStore{path: path}
}

func (s *FileStore) Save(ctx context.Context, sess *Session) error {
	dir := filepath.Dir(s.path)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("create credential directory: %w", err)
	}

	data, err := json.MarshalIndent(sess.Values, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal credentials: %w", err)
	}

	// Atomic write: write to temp file, rename into place.
	tmp, err := os.CreateTemp(dir, ".credentials-*")
	if err != nil {
		return fmt.Errorf("create temp credential file: %w", err)
	}
	tmpPath := tmp.Name()

	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		_ = os.Remove(tmpPath)
		return fmt.Errorf("write credentials: %w", err)
	}
	if err := tmp.Chmod(0600); err != nil {
		_ = tmp.Close()
		_ = os.Remove(tmpPath)
		return fmt.Errorf("set credential file permissions: %w", err)
	}
	if err := tmp.Sync(); err != nil {
		_ = tmp.Close()
		_ = os.Remove(tmpPath)
		return fmt.Errorf("sync credentials: %w", err)
	}
	if err := tmp.Close(); err != nil {
		_ = os.Remove(tmpPath)
		return fmt.Errorf("close credential file: %w", err)
	}
	if err := os.Rename(tmpPath, s.path); err != nil {
		_ = os.Remove(tmpPath)
		return fmt.Errorf("publish credentials: %w", err)
	}
	return nil
}

func (s *FileStore) Load(ctx context.Context) (*Session, bool, error) {
	data, err := os.ReadFile(s.path)
	if errors.Is(err, os.ErrNotExist) {
		return NewSession(), false, nil
	}
	if err != nil {
		return nil, false, fmt.Errorf("read credentials: %w", err)
	}

	sess := NewSession()
	if len(data) == 0 {
		return sess, true, nil
	}
	if err := json.Unmarshal(data, &sess.Values); err != nil {
		return nil, false, fmt.Errorf("parse credentials: %w", err)
	}
	return sess, true, nil
}

func (s *FileStore) Delete(ctx context.Context) error {
	err := os.Remove(s.path)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	return err
}

func (s *FileStore) Path() string { return s.path }

// EnvStore reads credentials from environment variables.
// It is read-only (Save and Delete are no-ops) and intended for
// unattended/server-runtime use where credentials come from env vars.
type EnvStore struct {
	mapping map[string]string // map[credentialKey] = envVarName
	path    string            // display path (e.g. "environment variables")
}

// NewEnvStore creates an EnvStore that reads the given credential keys
// from their corresponding environment variables.
func NewEnvStore(mapping map[string]string) *EnvStore {
	return &EnvStore{mapping: mapping, path: "environment variables"}
}

func (s *EnvStore) Save(_ context.Context, _ *Session) error { return nil }

func (s *EnvStore) Load(_ context.Context) (*Session, bool, error) {
	sess := NewSession()
	found := false
	for key, envName := range s.mapping {
		if val := os.Getenv(envName); val != "" {
			sess.Set(key, val)
			found = true
		}
	}
	return sess, found, nil
}

func (s *EnvStore) Delete(_ context.Context) error { return nil }
func (s *EnvStore) Path() string                   { return s.path }

// RetryableError signals a login stage failure that should re-prompt
// rather than aborting the login flow.
type RetryableError struct {
	Message string
}

func (e RetryableError) Error() string { return e.Message }
