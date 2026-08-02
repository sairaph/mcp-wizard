package update

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"time"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

// Options controls the self-update behaviour.
type Options struct {
	Owner          string // GitHub owner
	Repo           string // GitHub repo
	CurrentVersion string // current version string
	AssetName      func(os, arch string) string // maps "linux"/"amd64" to asset name
	DaemonStop     func() error                 // optional: stop daemon before swap
	InstallDir     string // where the binary lives (for swap)
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
		return "", false, nil
	}

	current, err := Parse(opts.CurrentVersion)
	if err != nil {
		return latest, false, nil
	}

	latestV, err := Parse(latest)
	if err != nil {
		return latest, false, nil
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
	tempFile := filepath.Join(tempDir, assetName+".download")
	f, err := os.Create(tempFile)
	if err != nil {
		return fmt.Errorf("create temp file: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, downloadURL, nil)
	if err != nil {
		_ = f.Close()
		_ = os.Remove(tempFile)
		return fmt.Errorf("create download request: %w", err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		_ = f.Close()
		_ = os.Remove(tempFile)
		return fmt.Errorf("download: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		_ = f.Close()
		_ = os.Remove(tempFile)
		return fmt.Errorf("download failed: HTTP %d", resp.StatusCode)
	}

	hash := sha256.New()
	if _, err := io.Copy(io.MultiWriter(f, hash), resp.Body); err != nil {
		_ = f.Close()
		_ = os.Remove(tempFile)
		return fmt.Errorf("download write: %w", err)
	}
	if err := f.Close(); err != nil {
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
	target := filepath.Join(opts.InstallDir, filepath.Base(assetName))
	if err := swapFile(tempFile, target); err != nil {
		_ = os.Remove(tempFile)
		return fmt.Errorf("swap binary: %w", err)
	}

	_ = os.Remove(tempFile)
	return nil
}

// SwapFrom performs the swap when the install script has already downloaded
// the binary to a temp file. Used for the `<bin> update --from <tempfile>` path.
func SwapFrom(ctx context.Context, tempPath string, opts Options) error {
	defer os.Remove(tempPath)
	if err := os.Chmod(tempPath, 0755); err != nil {
		return fmt.Errorf("chmod temp binary: %w", err)
	}
	target := filepath.Join(opts.InstallDir, filepath.Base(opts.AssetName(runtime.GOOS, runtime.GOARCH)))
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
		if parts[1] == assetName {
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
	if runtime.GOOS == "windows" {
		// Move old binary aside with a unique name.
		if _, err := os.Stat(target); err == nil {
			oldTarget := target + ".old-" + randString(8)
			if err := os.Rename(target, oldTarget); err != nil {
				return fmt.Errorf("move aside old binary: %w", err)
			}
			// Best-effort cleanup of old files.
			defer os.Remove(oldTarget)
		}
	}

	if err := os.Rename(source, target); err != nil {
		return fmt.Errorf("rename: %w", err)
	}
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
