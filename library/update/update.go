package update

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
	"time"
)

// Options controls the self-update behaviour.
type Options struct {
	Owner          string // GitHub owner
	Repo           string // GitHub repo
	CurrentVersion string // current version string
	AssetName      func(os, arch string) string // maps "linux"/"amd64" to asset name
	DaemonStop     func() error                 // optional: stop daemon before swap
	InstallDir     string // where the binary lives (for swap)
	BinaryName     string // installed binary filename
	TempDir        string // temp directory for downloads (empty = os.TempDir)
}

// Check queries the GitHub releases API for the latest version.
// Returns the latest version string and whether an update is available.
// Returns an error on network failures, non-200 status, or decode errors.
func Check(ctx context.Context, opts Options) (latest string, available bool, err error) {
	url := fmt.Sprintf("https://api.github.com/repos/%s/%s/releases/latest", opts.Owner, opts.Repo)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", false, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Accept", "application/vnd.github.v3+json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", false, fmt.Errorf("http request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", false, fmt.Errorf("GitHub API: HTTP %d", resp.StatusCode)
	}

	var release struct {
		TagName string `json:"tag_name"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return "", false, fmt.Errorf("decode response: %w", err)
	}

	latest = strings.TrimPrefix(release.TagName, "v")
	if latest == "" {
		return "", false, fmt.Errorf("release tag is empty")
	}

	current, err := Parse(opts.CurrentVersion)
	if err != nil {
		return "", false, fmt.Errorf("unparseable current version %q: %w", opts.CurrentVersion, err)
	}

	latestV, err := Parse(latest)
	if err != nil {
		return latest, false, fmt.Errorf("unparseable latest version %q: %w", latest, err)
	}

	return latest, latestV.Compare(current) > 0, nil
}

// SelfUpdate downloads the latest binary, verifies its SHA256, and swaps it
// into place. On Windows, uses move-aside swap (rename running .exe, move new
// into place) because Windows allows renaming a running executable but not
// deleting it.
func SelfUpdate(ctx context.Context, opts Options) error {
	if opts.InstallDir == "" {
		return fmt.Errorf("InstallDir must not be empty")
	}
	if opts.BinaryName == "" {
		return fmt.Errorf("BinaryName must not be empty")
	}
	if opts.AssetName == nil {
		return fmt.Errorf("AssetName function must not be nil")
	}
	osName := runtime.GOOS
	arch := runtime.GOARCH
	assetName := opts.AssetName(osName, arch)
	if assetName == "" {
		return fmt.Errorf("no asset name for %s/%s", osName, arch)
	}

	downloadURL := fmt.Sprintf("https://github.com/%s/%s/releases/latest/download/%s", opts.Owner, opts.Repo, assetName)
	checksumURL := fmt.Sprintf("https://github.com/%s/%s/releases/latest/download/SHA256SUMS.txt", opts.Owner, opts.Repo)

	tempDir := opts.TempDir
	if tempDir == "" {
		tempDir = os.TempDir()
	}

	// Download binary.
	tempF, err := os.CreateTemp(tempDir, assetName+".*.download")
	if err != nil {
		return fmt.Errorf("create temp file: %w", err)
	}
	tempFile := tempF.Name()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, downloadURL, nil)
	if err != nil {
		_ = tempF.Close()
		_ = os.Remove(tempFile)
		return fmt.Errorf("create download request: %w", err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		_ = tempF.Close()
		_ = os.Remove(tempFile)
		return fmt.Errorf("download: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		_ = tempF.Close()
		_ = os.Remove(tempFile)
		return fmt.Errorf("download failed: HTTP %d", resp.StatusCode)
	}

	hash := sha256.New()
	if _, err := io.Copy(io.MultiWriter(tempF, hash), resp.Body); err != nil {
		_ = tempF.Close()
		_ = os.Remove(tempFile)
		return fmt.Errorf("download write: %w", err)
	}
	if err := tempF.Close(); err != nil {
		_ = os.Remove(tempFile)
		return fmt.Errorf("close downloaded file: %w", err)
	}

	downloadedHash := hex.EncodeToString(hash.Sum(nil))

	// Verify SHA256 if checksum file is available.
	if err := verifyChecksum(ctx, checksumURL, assetName, downloadedHash); err != nil {
		_ = os.Remove(tempFile)
		return err
	}

	// Make executable.
	if err := os.Chmod(tempFile, 0755); err != nil {
		_ = os.Remove(tempFile)
		return fmt.Errorf("chmod: %w", err)
	}

	// Stop daemon before swap.
	if opts.DaemonStop != nil {
		if err := opts.DaemonStop(); err != nil {
			_ = os.Remove(tempFile)
			return fmt.Errorf("stop daemon: %w", err)
		}
	}

	// Swap into place.
	target := filepath.Join(opts.InstallDir, opts.BinaryName)
	if err := swapFile(tempFile, target); err != nil {
		_ = os.Remove(tempFile)
		return fmt.Errorf("swap binary: %w", err)
	}

	// After swapFile succeeds, the temp file no longer exists.
	// No cleanup needed.
	return nil
}

// SwapFrom performs the swap when the install script has already downloaded
// the binary to a temp file. Used for the `<bin> update --from <tempfile>` path.
func SwapFrom(ctx context.Context, tempPath string, opts Options) error {
	if opts.AssetName == nil {
		return fmt.Errorf("AssetName function must not be nil")
	}
	if opts.BinaryName == "" {
		return fmt.Errorf("BinaryName must not be empty")
	}
	if opts.InstallDir == "" {
		return fmt.Errorf("InstallDir must not be empty")
	}
	assetName := opts.AssetName(runtime.GOOS, runtime.GOARCH)
	if assetName == "" {
		return fmt.Errorf("no asset name for %s/%s", runtime.GOOS, runtime.GOARCH)
	}
	defer os.Remove(tempPath)
	if err := os.Chmod(tempPath, 0755); err != nil {
		return fmt.Errorf("chmod temp binary: %w", err)
	}
	target := filepath.Join(opts.InstallDir, opts.BinaryName)
	if err := swapFile(tempPath, target); err != nil {
		return fmt.Errorf("swap from temp: %w", err)
	}
	return nil
}

func verifyChecksum(ctx context.Context, checksumURL, assetName, downloadedHash string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, checksumURL, nil)
	if err != nil {
		return nil // checksum unavailable — non-fatal
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil // checksum unavailable — non-fatal
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil // checksum unavailable — non-fatal
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil
	}

	// Find the line for our asset.
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		parts := strings.Fields(line)
		if len(parts) != 2 {
			continue
		}
		filename := strings.TrimPrefix(parts[1], "*")
		if filename == assetName {
			if parts[0] != downloadedHash {
				return fmt.Errorf("SHA256 mismatch: expected %s, got %s", parts[0], downloadedHash)
			}
			return nil
		}
	}
	return nil // asset not found in checksum file — non-fatal
}

func swapFile(source, target string) error {
	dir := filepath.Dir(target)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("create directory: %w", err)
	}

	// On Unix, os.Rename is atomic when source and target are on the same FS.
	// On Windows, we use move-aside: rename old binary, move new into place.
	var oldTarget string
	if runtime.GOOS == "windows" {
		if _, err := os.Stat(target); err == nil {
			oldTarget = target + ".old-" + randString(8)
			if err := os.Rename(target, oldTarget); err != nil {
				return fmt.Errorf("move aside old binary: %w", err)
			}
		}
	}
	if oldTarget != "" {
		defer os.Remove(oldTarget)
	}

	err := os.Rename(source, target)
	if err != nil {
		// If the rename failed and we moved the old binary aside, restore it.
		if oldTarget != "" {
			os.Rename(oldTarget, target) // best-effort restore
		}
		if errors.Is(err, syscall.EXDEV) {
			if err := copyFile(source, target); err != nil {
				return fmt.Errorf("copy binary across devices: %w", err)
			}
			os.Remove(source)
			return nil
		}
		return fmt.Errorf("rename: %w", err)
	}
	return nil
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.Create(dst)
	if err != nil {
		in.Close()
		return err
	}

	// Clean up partial output on failure.
	cleanup := true
	defer func() {
		out.Close()
		if cleanup {
			os.Remove(dst)
		}
	}()

	if _, err := io.Copy(out, in); err != nil {
		return err
	}
	if err := out.Chmod(0755); err != nil {
		return err
	}
	if err := out.Sync(); err != nil {
		return err
	}
	cleanup = false
	return nil
}

func randString(n int) string {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		for i := range b {
			b[i] = byte(time.Now().UnixNano() >> uint(i*8%64))
		}
	}
	const letters = "abcdefghijklmnopqrstuvwxyz0123456789"
	for i := range b {
		b[i] = letters[int(b[i])%len(letters)]
	}
	return string(b)
}
