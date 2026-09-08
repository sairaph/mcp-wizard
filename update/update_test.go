package update_test

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/sairaph/mcp-wizard/update"
)

// fakeRelease serves a GitHub-shaped releases API and asset download tree.
type fakeRelease struct {
	tag      string
	assets   map[string][]byte // asset name -> bytes
	manifest string            // SHA256SUMS.txt body, "" means 404
}

func (f *fakeRelease) handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/repos/o/r/releases/latest", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"tag_name":"` + f.tag + `"}`))
	})
	mux.HandleFunc("/o/r/releases/latest/download/", func(w http.ResponseWriter, r *http.Request) {
		name := strings.TrimPrefix(r.URL.Path, "/o/r/releases/latest/download/")
		if name == update.ChecksumFile {
			if f.manifest == "" {
				http.NotFound(w, r)
				return
			}
			w.Write([]byte(f.manifest))
			return
		}
		data, ok := f.assets[name]
		if !ok {
			http.NotFound(w, r)
			return
		}
		w.Write(data)
	})
	return mux
}

func sum(b []byte) string {
	h := sha256.Sum256(b)
	return hex.EncodeToString(h[:])
}

func assetName(string, string) string { return "tool-" + runtime.GOOS + "-" + runtime.GOARCH }

func newOpts(t *testing.T, srv *httptest.Server) update.Options {
	t.Helper()
	return update.Options{
		Owner:          "o",
		Repo:           "r",
		CurrentVersion: "1.0.0",
		AssetName:      assetName,
		InstallDir:     filepath.Join(t.TempDir(), "bin"),
		BinaryName:     "tool",
		TempDir:        t.TempDir(),
		ReleaseBaseURL: srv.URL,
		APIBaseURL:     srv.URL,
		HTTPClient:     srv.Client(),
	}
}

func TestCheckReportsNewerRelease(t *testing.T) {
	srv := httptest.NewServer((&fakeRelease{tag: "v1.2.0"}).handler())
	defer srv.Close()
	latest, available, err := update.Check(context.Background(), newOpts(t, srv))
	if err != nil {
		t.Fatal(err)
	}
	if latest != "1.2.0" || !available {
		t.Fatalf("latest=%q available=%v", latest, available)
	}
}

func TestCheckSameVersionIsNotAvailable(t *testing.T) {
	srv := httptest.NewServer((&fakeRelease{tag: "v1.0.0"}).handler())
	defer srv.Close()
	_, available, err := update.Check(context.Background(), newOpts(t, srv))
	if err != nil || available {
		t.Fatalf("available=%v err=%v", available, err)
	}
}

func TestCheckNon200IsError(t *testing.T) {
	srv := httptest.NewServer(http.NotFoundHandler())
	defer srv.Close()
	if _, _, err := update.Check(context.Background(), newOpts(t, srv)); err == nil {
		t.Fatal("expected error on 404")
	}
}

func TestSelfUpdateVerifiesAndSwaps(t *testing.T) {
	binary := []byte("#!/bin/sh\necho new\n")
	name := assetName("", "")
	f := &fakeRelease{tag: "v2.0.0", assets: map[string][]byte{name: binary}}
	f.manifest = sum(binary) + "  " + name + "\n"
	srv := httptest.NewServer(f.handler())
	defer srv.Close()

	opts := newOpts(t, srv)
	target := filepath.Join(opts.InstallDir, opts.BinaryName)
	os.MkdirAll(opts.InstallDir, 0755)
	os.WriteFile(target, []byte("old"), 0755)

	stopped := false
	opts.DaemonStop = func() error { stopped = true; return nil }

	if err := update.SelfUpdate(context.Background(), opts); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(target)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != string(binary) {
		t.Fatalf("target not replaced: %q", got)
	}
	if !stopped {
		t.Fatal("DaemonStop should run before the swap")
	}
	if runtime.GOOS != "windows" {
		info, _ := os.Stat(target)
		if info.Mode()&0111 == 0 {
			t.Fatal("installed binary must be executable")
		}
	}
	leftovers, _ := filepath.Glob(filepath.Join(opts.TempDir, "*"))
	if len(leftovers) != 0 {
		t.Fatalf("temp download should be gone, found %v", leftovers)
	}
}

func TestSelfUpdateRejectsMismatch(t *testing.T) {
	binary := []byte("payload")
	name := assetName("", "")
	f := &fakeRelease{tag: "v2.0.0", assets: map[string][]byte{name: binary}}
	f.manifest = strings.Repeat("0", 64) + " *" + name + "\n"
	srv := httptest.NewServer(f.handler())
	defer srv.Close()

	opts := newOpts(t, srv)
	err := update.SelfUpdate(context.Background(), opts)
	if err == nil || !strings.Contains(err.Error(), "SHA256 mismatch") {
		t.Fatalf("expected mismatch error, got %v", err)
	}
	if _, statErr := os.Stat(filepath.Join(opts.InstallDir, opts.BinaryName)); !os.IsNotExist(statErr) {
		t.Fatal("nothing must be installed on mismatch")
	}
	leftovers, _ := filepath.Glob(filepath.Join(opts.TempDir, "*"))
	if len(leftovers) != 0 {
		t.Fatalf("temp download should be removed on failure, found %v", leftovers)
	}
}

func TestSelfUpdateMissingChecksumIsErrorByDefault(t *testing.T) {
	name := assetName("", "")
	f := &fakeRelease{tag: "v2.0.0", assets: map[string][]byte{name: []byte("x")}}
	srv := httptest.NewServer(f.handler())
	defer srv.Close()

	opts := newOpts(t, srv)
	err := update.SelfUpdate(context.Background(), opts)
	if !errors.Is(err, update.ErrChecksumUnavailable) {
		t.Fatalf("expected ErrChecksumUnavailable, got %v", err)
	}

	// Manifest present but without our asset is also unavailable.
	f.manifest = strings.Repeat("a", 64) + "  other-file\n"
	err = update.SelfUpdate(context.Background(), opts)
	if !errors.Is(err, update.ErrChecksumUnavailable) {
		t.Fatalf("expected ErrChecksumUnavailable for missing entry, got %v", err)
	}

	opts.AllowMissingChecksum = true
	if err := update.SelfUpdate(context.Background(), opts); err != nil {
		t.Fatalf("AllowMissingChecksum should permit the install, got %v", err)
	}
}

func TestSelfUpdateCancelledBeforeChecksumIsAbort(t *testing.T) {
	name := assetName("", "")
	binary := []byte("payload")
	f := &fakeRelease{tag: "v2.0.0", assets: map[string][]byte{name: binary}}
	f.manifest = sum(binary) + "  " + name + "\n"
	srv := httptest.NewServer(f.handler())
	defer srv.Close()

	opts := newOpts(t, srv)
	opts.AllowMissingChecksum = true
	ctx, cancel := context.WithCancel(context.Background())
	// Cancel once the asset has been served, before the checksum request.
	calls := 0
	opts.HTTPClient = &http.Client{Transport: cancelAfterFirst{rt: srv.Client().Transport, cancel: cancel, calls: &calls}}

	err := update.SelfUpdate(ctx, opts)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context.Canceled, got %v", err)
	}
	if _, statErr := os.Stat(filepath.Join(opts.InstallDir, opts.BinaryName)); !os.IsNotExist(statErr) {
		t.Fatal("a cancelled update must not install anything")
	}
	leftovers, _ := filepath.Glob(filepath.Join(opts.TempDir, "*"))
	if len(leftovers) != 0 {
		t.Fatalf("temp download should be removed on cancel, found %v", leftovers)
	}
}

// cancelAfterFirst lets the first request through and cancels the context
// before every later one.
type cancelAfterFirst struct {
	rt     http.RoundTripper
	cancel context.CancelFunc
	calls  *int
}

func (c cancelAfterFirst) RoundTrip(r *http.Request) (*http.Response, error) {
	*c.calls++
	if *c.calls > 1 {
		c.cancel()
		return nil, context.Canceled
	}
	return c.rt.RoundTrip(r)
}

func TestRemoveStaleBinaries(t *testing.T) {
	dir := t.TempDir()
	for _, n := range []string{"tool.old-abc12345", "tool.old-zzz", "tool", "other.old-x"} {
		os.WriteFile(filepath.Join(dir, n), []byte("x"), 0644)
	}
	update.RemoveStaleBinaries(dir, "tool")
	left, _ := filepath.Glob(filepath.Join(dir, "*"))
	var names []string
	for _, l := range left {
		names = append(names, filepath.Base(l))
	}
	if len(names) != 2 || names[0] != "other.old-x" || names[1] != "tool" {
		t.Fatalf("unexpected leftovers: %v", names)
	}
}

func TestSelfUpdateValidatesOptions(t *testing.T) {
	ctx := context.Background()
	if err := update.SelfUpdate(ctx, update.Options{}); err == nil {
		t.Fatal("empty options must fail")
	}
	if err := update.SelfUpdate(ctx, update.Options{InstallDir: "x", BinaryName: "y"}); err == nil {
		t.Fatal("nil AssetName must fail")
	}
	err := update.SelfUpdate(ctx, update.Options{InstallDir: "x", BinaryName: "y", AssetName: func(string, string) string { return "" }})
	if err == nil {
		t.Fatal("empty asset name must fail")
	}
}

func TestSwapFromReplacesTarget(t *testing.T) {
	dir := t.TempDir()
	opts := update.Options{InstallDir: filepath.Join(dir, "bin"), BinaryName: "tool"}
	staged := filepath.Join(dir, "download")
	if err := os.WriteFile(staged, []byte("fresh"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := update.SwapFrom(context.Background(), staged, opts); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(filepath.Join(opts.InstallDir, "tool"))
	if err != nil || string(got) != "fresh" {
		t.Fatalf("got %q err %v", got, err)
	}
	if _, err := os.Stat(staged); !os.IsNotExist(err) {
		t.Fatal("staged file should be consumed")
	}
}
