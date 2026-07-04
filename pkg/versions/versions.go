// Package versions fetches and parses the Go version list from go.dev.
package versions

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Release represents a single Go release from the download API.
type Release struct {
	Version string `json:"version"` // e.g. "go1.23.4"
	Stable  bool   `json:"stable"`
	Files   []File `json:"files"`
}

// File describes a downloadable archive for a specific OS/arch.
type File struct {
	Filename string `json:"filename"`
	OS       string `json:"os"`
	Arch     string `json:"arch"`
	Version  string `json:"version"`
	SHA256   string `json:"sha256"`
	Size     int64  `json:"size"`
	Kind     string `json:"kind"` // "archive", "installer", "source"
}

// cacheDir is the directory for caching version lists (e.g. ~/.govm/cache).
// Set via SetCacheDir; empty means no caching.
var cacheDir string

// SetCacheDir sets the cache directory. Should be called once at startup.
func SetCacheDir(dir string) {
	cacheDir = dir
}

// cacheTTL controls how often the remote API is polled.
const cacheTTL = 1 * time.Hour

// defaultHTTPClient is shared by FetchAll / FetchStable with a 30s timeout.
var defaultHTTPClient = &http.Client{Timeout: 30 * time.Second}

// FetchAll retrieves every release from go.dev/dl/?mode=json.
// Results are cached in CacheDir for cacheTTL.
func FetchAll(ctx context.Context) ([]Release, error) {
	// Try cache first.
	if cached, ok := loadCache(); ok {
		return cached, nil
	}

	releases, err := fetchRemote(ctx)
	if err != nil {
		return nil, err
	}

	// Save to cache (best-effort).
	if cacheDir != "" {
		_ = saveCache(releases)
	}

	return releases, nil
}

// FetchStable returns only stable releases.
func FetchStable(ctx context.Context) ([]Release, error) {
	all, err := FetchAll(ctx)
	if err != nil {
		return nil, err
	}
	var stable []Release
	for _, r := range all {
		if r.Stable {
			stable = append(stable, r)
		}
	}
	return stable, nil
}

// fetchRemote performs the actual HTTP request.
func fetchRemote(ctx context.Context) ([]Release, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", "https://go.dev/dl/?mode=json", nil)
	if err != nil {
		return nil, fmt.Errorf("fetch version list: %w", err)
	}

	resp, err := defaultHTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch version list: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("fetch version list: HTTP %d", resp.StatusCode)
	}

	var releases []Release
	if err := json.NewDecoder(resp.Body).Decode(&releases); err != nil {
		return nil, fmt.Errorf("decode version list: %w", err)
	}
	return releases, nil
}

// loadCache returns cached releases if fresh.
func loadCache() ([]Release, bool) {
	if cacheDir == "" {
		return nil, false
	}
	path := filepath.Join(cacheDir, "versions.json")
	info, err := os.Stat(path)
	if err != nil {
		return nil, false
	}
	if time.Since(info.ModTime()) > cacheTTL {
		return nil, false
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, false
	}
	var releases []Release
	if err := json.Unmarshal(data, &releases); err != nil {
		return nil, false
	}
	return releases, true
}

// saveCache writes releases to the cache file.
func saveCache(releases []Release) error {
	if cacheDir == "" {
		return nil
	}
	if err := os.MkdirAll(cacheDir, 0o755); err != nil {
		return err
	}
	path := filepath.Join(cacheDir, "versions.json")
	data, err := json.Marshal(releases)
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}

// ShortVersion strips the "go" prefix from a version string.
// "go1.23.4" → "1.23.4"
func ShortVersion(v string) string {
	return strings.TrimPrefix(v, "go")
}
