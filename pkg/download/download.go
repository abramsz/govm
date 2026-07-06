// Package download handles fetching and extracting Go distribution archives.
package download

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/abramsz/govm/pkg/config"
	"github.com/abramsz/govm/pkg/versions"
)

const archivePerm os.FileMode = 0o755

// defaultHTTPClient is used for downloads with a 30m timeout (large archives).
var defaultHTTPClient = &http.Client{Timeout: 30 * time.Minute}

// baseURLs is the ordered list of download base URLs to try.
// Set via SetBaseURLs.
var baseURLs []string

// SetBaseURLs sets the ordered list of mirror URLs to try.
func SetBaseURLs(urls []string) {
	baseURLs = urls
}

// FindFile returns the download File for the given OS/arch from a release.
// If archOverride is empty, runtime.GOARCH is used.
// Returns nil if no matching file exists (unsupported platform).
func FindFile(release versions.Release, archOverride string) *versions.File {
	arch := archOverride
	if arch == "" {
		arch = runtime.GOARCH
	}
	for _, f := range release.Files {
		if f.OS == runtime.GOOS && f.Arch == arch && f.Kind == "archive" {
			return &f
		}
	}
	return nil
}

// Download fetches a Go distribution archive from go.dev and saves it to destDir.
// destDir is created if needed. The archive is saved to a temp file, then
// extracted and cleaned up. Returns the path to the extracted Go root.
// The SHA256 checksum from the release metadata is verified after download.
func Download(ctx context.Context, release versions.Release, destDir string, archOverride string) (string, error) {
	f := FindFile(release, archOverride)
	if f == nil {
		return "", fmt.Errorf("no archive for %s/%s in release %s", runtime.GOOS, runtime.GOARCH, release.Version)
	}

	urls := baseURLs
	if len(urls) == 0 {
		urls = []string{config.DefaultMirror}
	}

	var lastErr error
	var tmpFile string
	for _, base := range urls {
		url := base + f.Filename
		fmt.Fprintf(os.Stderr, "Downloading from %s\n", base)

		tmpFile, lastErr = downloadToTemp(ctx, url, f.Size, f.SHA256)
		if lastErr == nil {
			break
		}
		fmt.Fprintf(os.Stderr, "  failed: %v, trying next mirror...\n", lastErr)
		lastErr = fmt.Errorf("%s: %w", base, lastErr)
	}
	if lastErr != nil {
		return "", fmt.Errorf("all mirrors failed: %v", lastErr)
	}
	defer os.Remove(tmpFile)

	if err := os.MkdirAll(destDir, archivePerm); err != nil {
		return "", fmt.Errorf("create destination %s: %w", destDir, err)
	}

	fmt.Fprintf(os.Stderr, "Extracting to %s\n", destDir)
	if err := extract(tmpFile, destDir); err != nil {
		return "", fmt.Errorf("extract: %w", err)
	}

	// The Go archive contains a top-level "go/" directory.
	// destDir/go is the resulting GOROOT.
	goroot := filepath.Join(destDir, "go")
	if _, err := os.Stat(goroot); err != nil {
		return "", fmt.Errorf("go root not found after extraction (expected %s): %w", goroot, err)
	}

	return goroot, nil
}

// downloadToTemp fetches url into a temporary file and returns its path.
// If expectedSHA256 is non-empty, the downloaded content is verified against it.
func downloadToTemp(ctx context.Context, url string, expectedSize int64, expectedSHA256 string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return "", fmt.Errorf("create request: %w", err)
	}

	resp, err := defaultHTTPClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("download %s: %w", url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("download %s: HTTP %d", url, resp.StatusCode)
	}

	tmp, err := os.CreateTemp("", "govm-download-*")
	if err != nil {
		return "", fmt.Errorf("create temp file: %w", err)
	}
	defer tmp.Close()

	// Chain readers: progress → hash (if needed) → file.
	var reader io.Reader = newProgressReader(resp.Body, expectedSize, os.Stderr)
	if expectedSHA256 != "" {
		hasher := sha256.New()
		reader = io.TeeReader(reader, hasher)
		if _, err := io.Copy(tmp, reader); err != nil {
			os.Remove(tmp.Name())
			return "", fmt.Errorf("write download: %w", err)
		}
		fmt.Fprintln(os.Stderr) // newline after progress

		got := hex.EncodeToString(hasher.Sum(nil))
		if !strings.EqualFold(got, expectedSHA256) {
			os.Remove(tmp.Name())
			return "", fmt.Errorf("sha256 mismatch: got %s, expected %s", got, expectedSHA256)
		}
	} else {
		if _, err := io.Copy(tmp, reader); err != nil {
			os.Remove(tmp.Name())
			return "", fmt.Errorf("write download: %w", err)
		}
		fmt.Fprintln(os.Stderr) // newline after progress
	}

	return tmp.Name(), nil
}

// extract unpacks an archive (tar.gz or zip) into destDir.
// File type is detected from magic bytes, not the filename extension.
func extract(archivePath, destDir string) error {
	f, err := os.Open(archivePath)
	if err != nil {
		return err
	}

	// Read first 4 bytes to detect archive type.
	var magic [4]byte
	n, _ := f.Read(magic[:])
	f.Close()
	if n < 2 {
		return fmt.Errorf("cannot detect archive type: file too small")
	}

	// ZIP files start with "PK" (0x50 0x4B).
	if magic[0] == 0x50 && magic[1] == 0x4B {
		return extractZip(archivePath, destDir)
	}
	// Gzip files start with 0x1F 0x8B.
	if magic[0] == 0x1F && magic[1] == 0x8B {
		return extractTarGz(archivePath, destDir)
	}

	return fmt.Errorf("unknown archive format (magic: %x %x)", magic[0], magic[1])
}

func extractTarGz(path, destDir string) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()

	gzr, err := gzip.NewReader(f)
	if err != nil {
		return err
	}
	defer gzr.Close()

	tr := tar.NewReader(gzr)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		clean := filepath.Clean(hdr.Name)
		if strings.Contains(clean, "..") {
			continue
		}

		target := filepath.Join(destDir, clean)

		switch hdr.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(target, archivePerm); err != nil {
				return err
			}
		case tar.TypeReg:
			if err := os.MkdirAll(filepath.Dir(target), archivePerm); err != nil {
				return err
			}
			out, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, os.FileMode(hdr.Mode))
			if err != nil {
				return err
			}
			if _, err := io.Copy(out, tr); err != nil {
				out.Close()
				return err
			}
			out.Close()
		case tar.TypeSymlink:
			if err := os.MkdirAll(filepath.Dir(target), archivePerm); err != nil {
				return err
			}
			// Validate symlink target path — prevent path traversal.
			cleanLink := filepath.Clean(hdr.Linkname)
			if strings.Contains(cleanLink, "..") {
				continue
			}
			// Resolve the symlink target relative to destDir and ensure it stays inside.
			resolved := filepath.Join(destDir, cleanLink)
			if !strings.HasPrefix(filepath.Clean(resolved), filepath.Clean(destDir)+string(filepath.Separator)) &&
				resolved != filepath.Clean(destDir) {
				continue
			}
			if err := os.Symlink(cleanLink, target); err != nil {
				fmt.Fprintf(os.Stderr, "govm: warning: failed to create symlink %s → %s: %v\n", target, cleanLink, err)
			}
		}
	}
	return nil
}

func extractZip(path, destDir string) error {
	r, err := zip.OpenReader(path)
	if err != nil {
		return err
	}
	defer r.Close()

	for _, f := range r.File {
		clean := filepath.Clean(f.Name)
		if strings.Contains(clean, "..") {
			continue
		}

		target := filepath.Join(destDir, clean)

		if f.FileInfo().IsDir() {
			if err := os.MkdirAll(target, archivePerm); err != nil {
				return err
			}
			continue
		}

		if err := os.MkdirAll(filepath.Dir(target), archivePerm); err != nil {
			return err
		}

		out, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, f.Mode())
		if err != nil {
			return err
		}

		rc, err := f.Open()
		if err != nil {
			out.Close()
			return err
		}

		_, err = io.Copy(out, rc)
		rc.Close()
		out.Close()
		if err != nil {
			return err
		}
	}
	return nil
}
