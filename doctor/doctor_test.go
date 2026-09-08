package doctor_test

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/sairaph/mcp-wizard/doctor"
)

func TestOKFormatting(t *testing.T) {
	r := doctor.New(staticCheck{name: "test", status: doctor.OK})
	var buf bytes.Buffer
	code := r.Run(context.Background(), &buf)
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d", code)
	}
	if !bytes.Contains(buf.Bytes(), []byte("[ok]")) {
		t.Fatalf("expected [ok] in output:\n%s", buf.String())
	}
}

func TestFailFormatting(t *testing.T) {
	r := doctor.New(staticCheck{name: "test", status: doctor.Fail, detail: "something broke"})
	var buf bytes.Buffer
	code := r.Run(context.Background(), &buf)
	if code != 1 {
		t.Fatalf("expected exit code 1, got %d", code)
	}
	if !bytes.Contains(buf.Bytes(), []byte("[fail]")) {
		t.Fatalf("expected [fail] in output:\n%s", buf.String())
	}
	if !bytes.Contains(buf.Bytes(), []byte("something broke")) {
		t.Fatalf("expected detail in output:\n%s", buf.String())
	}
}

func TestWarnFormatting(t *testing.T) {
	r := doctor.New(staticCheck{name: "test", status: doctor.Warn, detail: "warning"})
	var buf bytes.Buffer
	code := r.Run(context.Background(), &buf)
	if code != 0 {
		t.Fatalf("expected exit code 0 for warn, got %d", code)
	}
	if !bytes.Contains(buf.Bytes(), []byte("[warn]")) {
		t.Fatalf("expected [warn] in output:\n%s", buf.String())
	}
}

func TestMixedResultsReturnsExitCode1(t *testing.T) {
	r := doctor.New(
		staticCheck{name: "ok", status: doctor.OK},
		staticCheck{name: "fail", status: doctor.Fail, detail: "error"},
		staticCheck{name: "warn", status: doctor.Warn},
	)
	var buf bytes.Buffer
	code := r.Run(context.Background(), &buf)
	if code != 1 {
		t.Fatalf("expected exit code 1, got %d", code)
	}
}

func TestAllOKReturnsExitCode0(t *testing.T) {
	r := doctor.New(
		staticCheck{name: "a", status: doctor.OK},
		staticCheck{name: "b", status: doctor.OK},
	)
	var buf bytes.Buffer
	code := r.Run(context.Background(), &buf)
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d", code)
	}
}

func TestExecutableCheckOnTestBinary(t *testing.T) {
	exe, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	c := doctor.ExecutableCheck{Executable: exe}
	result := c.Run(context.Background())
	if result.Status != doctor.OK {
		t.Fatalf("expected OK, got %s: %s", result.Status, result.Detail)
	}
}

func TestExecutableCheckOnNonExistent(t *testing.T) {
	c := doctor.ExecutableCheck{Executable: "/nonexistent/binary"}
	result := c.Run(context.Background())
	if result.Status != doctor.Fail {
		t.Fatalf("expected Fail, got %s", result.Status)
	}
}

func TestPathCheckOnNonExistentDir(t *testing.T) {
	c := doctor.PathCheck{Dir: "/nonexistent-dir-12345"}
	result := c.Run(context.Background())
	if result.Status != doctor.Warn {
		t.Fatalf("expected Warn, got %s", result.Status)
	}
}

func TestConfigExistsCheckOnExistingFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")
	if err := os.WriteFile(path, []byte("{}"), 0644); err != nil {
		t.Fatal(err)
	}
	c := doctor.ConfigExistsCheck{Path: path}
	result := c.Run(context.Background())
	if result.Status != doctor.OK {
		t.Fatalf("expected OK, got %s: %s", result.Status, result.Detail)
	}
}

func TestConfigExistsCheckOnNonExistentFile(t *testing.T) {
	c := doctor.ConfigExistsCheck{Path: "/nonexistent/config.json"}
	result := c.Run(context.Background())
	if result.Status != doctor.Fail {
		t.Fatalf("expected Fail, got %s", result.Status)
	}
}

func TestUpdateCheckNoConfig(t *testing.T) {
	c := doctor.UpdateCheck{}
	result := c.Run(context.Background())
	if result.Status != doctor.Warn {
		t.Fatalf("expected Warn, got %s", result.Status)
	}
}

func TestAddAppendsChecks(t *testing.T) {
	r := doctor.New()
	r.Add(
		staticCheck{name: "a", status: doctor.OK},
		staticCheck{name: "b", status: doctor.OK},
	)
	var buf bytes.Buffer
	code := r.Run(context.Background(), &buf)
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d", code)
	}
}

// staticCheck is a helper that returns a Check with constant results.
type staticCheck struct {
	name   string
	status doctor.Status
	detail string
}

func (s staticCheck) Name() string { return s.name }
func (s staticCheck) Run(_ context.Context) doctor.Result {
	return doctor.Result{Name: s.name, Status: s.status, Detail: s.detail}
}
