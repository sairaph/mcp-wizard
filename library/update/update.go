package update

import (
	"context"
	"crypto/sha256"
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
// Returns ("", false, nil) on any error (network, rate limit) — never
// returns an error for unreachable servers; that's a doctor warning.
func Check(ctx context.Context, opts Options) (latest string, available bool, err error) {
	url := fmt.Sprintf("https://api.github.com/repos/%s/%s/releases/latest", opts.Owner, opts.Repo)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", false, nil
	}
	req.Header.Set("Accept", "application/vnd.github.v3+json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", false, nil
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", false, nil
	}

	var release struct {
		TagName string `json:"tag_name"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return "", false, nil
	}

	latest = strings.TrimPrefix(release.TagName, "v")
	if latest == "" {
		return "", false, nil
	}

	current, err := Parse(opts.CurrentVersion)
	if err != nil {
		return latest, true, nil // can't parse current — just report latest
	}

	latestV, err := Parse(latest)
	if err != nil {
		return latest, true, nil
	}

	return latest, latestV.Compare(current) > 0, nil
}

// SelfUpdate downloads the latest binary, verifies its SHA256, and swaps it
// into place. On Windows, uses move-aside swap (rename running .exe, move new
// into place) because Windows allows renaming a running executable but not
// deleting it.
func SelfUpdate(ctx context.Context, opts Options) error {
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

	resp, err := http.Get(downloadURL)
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
	_ = f.Close()

	downloadedHash := hex.EncodeToString(hash.Sum(nil))

	// Verify SHA256 if checksum file is available.
	if err := verifyChecksum(checksumURL, assetName, downloadedHash); err != nil {
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
	target := filepath.Join(opts.InstallDir, filepath.Base(opts.AssetName(runtime.GOOS, runtime.GOARCH)))
	if err := swapFile(tempPath, target); err != nil {
		return fmt.Errorf("swap from temp: %w", err)
	}
	return nil
}

func verifyChecksum(checksumURL, assetName, downloadedHash string) error {
	resp, err := http.Get(checksumURL)
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
	const letters = "abcdefghijklmnopqrstuvwxyz0123456789"
	b := make([]byte, n)
	for i := range b {
		b[i] = letters[i%len(letters)]
	}
	return string(b)
}
