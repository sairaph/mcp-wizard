package update

import (
	"context"
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

	"github.com/sairaph/mcp-wizard/internal/tmpname"
)

// Options controls the self-update behaviour.
type Options struct {
	Owner          string                       // GitHub owner
	Repo           string                       // GitHub repo
	CurrentVersion string                       // current version string
	AssetName      func(os, arch string) string // maps "linux"/"amd64" to asset name
	DaemonStop     func() error                 // optional: stop daemon before swap
	InstallDir     string                       // where the binary lives (for swap)
	BinaryName     string                       // installed binary filename
	TempDir        string                       // temp directory for downloads (empty = os.TempDir)

	// AllowMissingChecksum lets SelfUpdate proceed when SHA256SUMS.txt cannot
	// be fetched or does not list the asset. By default a missing or
	// unreadable checksum file is an error, so a corrupted or mismatched
	// download is never installed silently. A checksum that is present but
	// does not match is always an error.
	AllowMissingChecksum bool

	// HTTPClient performs the requests. Nil means http.DefaultClient.
	HTTPClient *http.Client

	// ReleaseBaseURL is the host serving release assets, without a trailing
	// slash. Empty means https://github.com. APIBaseURL is the host serving
	// the releases API. Empty means https://api.github.com. Both exist so
	// tests and mirrors can redirect traffic.
	ReleaseBaseURL string
	APIBaseURL     string
}

func (o Options) client() *http.Client {
	if o.HTTPClient != nil {
		return o.HTTPClient
	}
	return http.DefaultClient
}

func (o Options) releaseURL(asset string) string {
	base := o.ReleaseBaseURL
	if base == "" {
		base = "https://github.com"
	}
	return fmt.Sprintf("%s/%s/%s/releases/latest/download/%s", base, o.Owner, o.Repo, asset)
}

func (o Options) latestReleaseAPIURL() string {
	base := o.APIBaseURL
	if base == "" {
		base = "https://api.github.com"
	}
	return fmt.Sprintf("%s/repos/%s/%s/releases/latest", base, o.Owner, o.Repo)
}

// ChecksumFile is the name of the SHA256 manifest published with each release.
const ChecksumFile = "SHA256SUMS.txt"

// Check queries the GitHub releases API for the latest version.
// Returns the latest version string and whether an update is available.
// Returns an error on network failures, non-200 status, or decode errors.
func Check(ctx context.Context, opts Options) (latest string, available bool, err error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, opts.latestReleaseAPIURL(), nil)
	if err != nil {
		return "", false, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Accept", "application/vnd.github.v3+json")

	resp, err := opts.client().Do(req)
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

// SelfUpdate downloads the latest binary, verifies its SHA256 against the
// release's SHA256SUMS.txt, and swaps it into place. On Windows, uses
// move-aside swap (rename running .exe, move new into place) because Windows
// allows renaming a running executable but not deleting it.
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
	fail := func(err error) error {
		_ = tempF.Close()
		_ = os.Remove(tempFile)
		return err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, opts.releaseURL(assetName), nil)
	if err != nil {
		return fail(fmt.Errorf("create download request: %w", err))
	}
	resp, err := opts.client().Do(req)
	if err != nil {
		return fail(fmt.Errorf("download: %w", err))
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fail(fmt.Errorf("download failed: HTTP %d", resp.StatusCode))
	}

	hash := sha256.New()
	if _, err := io.Copy(io.MultiWriter(tempF, hash), resp.Body); err != nil {
		return fail(fmt.Errorf("download write: %w", err))
	}
	if err := tempF.Close(); err != nil {
		_ = os.Remove(tempFile)
		return fmt.Errorf("close downloaded file: %w", err)
	}

	downloadedHash := hex.EncodeToString(hash.Sum(nil))

	if err := verifyChecksum(ctx, opts, assetName, downloadedHash); err != nil {
		_ = os.Remove(tempFile)
		return err
	}

	// Make executable.
	if err := os.Chmod(tempFile, 0755); err != nil {
		_ = os.Remove(tempFile)
		return fmt.Errorf("chmod: %w", err)
	}

	// Nothing irreversible happens after a cancelled context.
	if err := ctx.Err(); err != nil {
		_ = os.Remove(tempFile)
		return err
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
	return nil
}

// SwapFrom performs the swap when the install script has already downloaded
// the binary to a temp file. Used for the `<bin> update --from <tempfile>` path.
func SwapFrom(ctx context.Context, tempPath string, opts Options) error {
	if opts.BinaryName == "" {
		return fmt.Errorf("BinaryName must not be empty")
	}
	if opts.InstallDir == "" {
		return fmt.Errorf("InstallDir must not be empty")
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

// ErrChecksumUnavailable is returned (wrapped) when the release has no usable
// checksum for the asset and AllowMissingChecksum is false.
var ErrChecksumUnavailable = errors.New("checksum unavailable")

// verifyChecksum fetches SHA256SUMS.txt for the release and compares the
// asset's entry with downloadedHash.
func verifyChecksum(ctx context.Context, opts Options, assetName, downloadedHash string) error {
	unavailable := func(reason string) error {
		if opts.AllowMissingChecksum {
			return nil
		}
		return fmt.Errorf("%w: %s (set AllowMissingChecksum to skip verification)", ErrChecksumUnavailable, reason)
	}

	if err := ctx.Err(); err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, opts.releaseURL(ChecksumFile), nil)
	if err != nil {
		return unavailable(err.Error())
	}
	resp, err := opts.client().Do(req)
	if err != nil {
		// A cancelled update is an abort, never a reason to skip verification.
		if ctxErr := ctx.Err(); ctxErr != nil {
			return ctxErr
		}
		return unavailable(err.Error())
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return unavailable(fmt.Sprintf("%s: HTTP %d", ChecksumFile, resp.StatusCode))
	}

	data, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return unavailable("read " + ChecksumFile + ": " + err.Error())
	}

	expected, ok := lookupChecksum(string(data), assetName)
	if !ok {
		return unavailable(fmt.Sprintf("%s has no entry for %s", ChecksumFile, assetName))
	}
	if !strings.EqualFold(expected, downloadedHash) {
		return fmt.Errorf("SHA256 mismatch for %s: expected %s, got %s", assetName, expected, downloadedHash)
	}
	return nil
}

// lookupChecksum finds the hash for assetName in a sha256sum-style manifest
// ("<hash>  <file>" or "<hash> *<file>" per line).
func lookupChecksum(manifest, assetName string) (string, bool) {
	for _, line := range strings.Split(manifest, "\n") {
		parts := strings.Fields(strings.TrimSpace(line))
		if len(parts) != 2 {
			continue
		}
		filename := strings.TrimPrefix(parts[1], "*")
		if filename == assetName {
			return parts[0], true
		}
	}
	return "", false
}

// swapFile moves source into place at target. On the same filesystem this is
// a single atomic rename. Across filesystems the source is first copied to a
// staging file beside target and then renamed, so target is still replaced
// atomically. On Windows the existing target is moved aside first, because a
// running executable can be renamed but not overwritten; it is restored if
// the swap fails. Removing it afterwards fails while that executable is
// still running (the usual self-update case), so a "<target>.old-xxxxxxxx"
// file can remain until RemoveStaleBinaries is called on a later start.
// rename is os.Rename, replaceable in tests to simulate cross-device moves.
var rename = os.Rename

func swapFile(source, target string) error {
	dir := filepath.Dir(target)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("create directory: %w", err)
	}

	var oldTarget string
	if runtime.GOOS == "windows" {
		if _, err := os.Stat(target); err == nil {
			oldTarget = target + ".old-" + tmpname.Suffix(8)
			if err := rename(target, oldTarget); err != nil {
				return fmt.Errorf("move aside old binary: %w", err)
			}
		}
	}
	restore := func() {
		if oldTarget != "" {
			_ = os.Rename(oldTarget, target)
		}
	}

	err := rename(source, target)
	if err != nil && isCrossDevice(err) {
		staged := target + ".staging-" + tmpname.Suffix(8)
		if copyErr := copyFile(source, staged); copyErr != nil {
			restore()
			return fmt.Errorf("copy binary across devices: %w", copyErr)
		}
		if err = rename(staged, target); err != nil {
			_ = os.Remove(staged)
			restore()
			return fmt.Errorf("rename staged binary: %w", err)
		}
		_ = os.Remove(source)
	} else if err != nil {
		restore()
		return fmt.Errorf("rename: %w", err)
	}

	if oldTarget != "" {
		os.Remove(oldTarget)
	}
	return nil
}

// RemoveStaleBinaries deletes "<binary>.old-*" files left in installDir by a
// previous Windows self-update. Call it early at startup; errors are
// ignored because the files are harmless.
func RemoveStaleBinaries(installDir, binaryName string) {
	matches, _ := filepath.Glob(filepath.Join(installDir, binaryName+".old-*"))
	for _, m := range matches {
		_ = os.Remove(m)
	}
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.OpenFile(dst, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0755)
	if err != nil {
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
