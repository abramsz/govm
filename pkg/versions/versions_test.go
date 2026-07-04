package versions

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestShortVersion(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"go1.23.4", "1.23.4"},
		{"go1.22.10", "1.22.10"},
		{"go1.24.0-rc1", "1.24.0-rc1"},
		{"go1.21", "1.21"},
		{"1.23.4", "1.23.4"},
		{"", ""},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := ShortVersion(tt.input)
			if got != tt.expected {
				t.Errorf("ShortVersion(%q) = %q; want %q", tt.input, got, tt.expected)
			}
		})
	}
}

func TestFetchStable(t *testing.T) {
	releases := []Release{
		{Version: "go1.23.4", Stable: true},
		{Version: "go1.24.0-rc1", Stable: false},
		{Version: "go1.22.10", Stable: true},
		{Version: "go1.25.0-beta1", Stable: false},
	}

	var stable []Release
	for _, r := range releases {
		if r.Stable {
			stable = append(stable, r)
		}
	}

	if len(stable) != 2 {
		t.Errorf("expected 2 stable releases, got %d", len(stable))
	}
	for _, r := range stable {
		if !r.Stable {
			t.Errorf("unstable release slipped through: %s", r.Version)
		}
	}
}

func TestCache_SaveAndLoad(t *testing.T) {
	dir := t.TempDir()
	prev := cacheDir
	SetCacheDir(dir)
	defer SetCacheDir(prev)

	r := []Release{
		{Version: "go1.23.4", Stable: true},
		{Version: "go1.22.10", Stable: false},
	}

	if err := saveCache(r); err != nil {
		t.Fatalf("saveCache failed: %v", err)
	}

	cacheFile := filepath.Join(dir, "versions.json")
	if _, err := os.Stat(cacheFile); err != nil {
		t.Fatalf("cache file not created: %v", err)
	}

	loaded, ok := loadCache()
	if !ok {
		t.Fatal("loadCache returned false for fresh cache")
	}
	if len(loaded) != 2 {
		t.Fatalf("loaded %d releases, want 2", len(loaded))
	}
	if loaded[0].Version != "go1.23.4" {
		t.Errorf("loaded[0].Version = %q; want go1.23.4", loaded[0].Version)
	}
}

func TestCache_MissingCache(t *testing.T) {
	dir := t.TempDir()
	prev := cacheDir
	SetCacheDir(dir)
	defer SetCacheDir(prev)

	_, ok := loadCache()
	if ok {
		t.Error("loadCache should return false for missing cache file")
	}
}

func TestCache_StaleCache(t *testing.T) {
	dir := t.TempDir()
	prev := cacheDir
	SetCacheDir(dir)
	defer SetCacheDir(prev)

	r := []Release{{Version: "go1.23.4", Stable: true}}
	if err := saveCache(r); err != nil {
		t.Fatalf("saveCache: %v", err)
	}

	cacheFile := filepath.Join(dir, "versions.json")
	oldTime := time.Now().Add(-2 * cacheTTL)
	if err := os.Chtimes(cacheFile, oldTime, oldTime); err != nil {
		t.Skipf("cannot set mtime (unsupported on this platform): %v", err)
	}

	_, ok := loadCache()
	if ok {
		t.Error("loadCache should return false for stale cache (>1h old)")
	}
}

func TestCache_DisabledWhenDirEmpty(t *testing.T) {
	prev := cacheDir
	SetCacheDir("")
	defer SetCacheDir(prev)

	_, ok := loadCache()
	if ok {
		t.Error("loadCache should return false when cacheDir is empty")
	}
}
