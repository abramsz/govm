// Package store manages the local Go version store under GOVM_HOME.
//
// Directory layout:
//
//	$GOVM_HOME/
//	  versions/
//	    <version>/
//	      go/          ← extracted GOROOT
//	  current → versions/<version>/go   ← symlink for global default
package store

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sort"

	"golang.org/x/mod/semver"

	"github.com/abramsz/govm/pkg/download"
	"github.com/abramsz/govm/pkg/versions"
)

// Home returns the govm data directory.
// Defaults to ~/.govm; override with GOVM_HOME.
func Home() string {
	if h := os.Getenv("GOVM_HOME"); h != "" {
		return h
	}
	home, err := os.UserHomeDir()
	if err != nil {
		// Fall back to temp dir when $HOME is unset (e.g. minimal containers).
		return filepath.Join(os.TempDir(), ".govm")
	}
	return filepath.Join(home, ".govm")
}

// VersionsDir returns the directory where Go distributions are stored.
func VersionsDir() string {
	return filepath.Join(Home(), "versions")
}

// CurrentLink returns the path to the "current" symlink.
func CurrentLink() string {
	return filepath.Join(Home(), "current")
}

// Install downloads and extracts a Go release into the store.
// When overwrite is true, a pre-existing installation is removed first.
// Returns the GOROOT path of the installed version.
func Install(ctx context.Context, release versions.Release, overwrite bool, archOverride string) (string, error) {
	if err := Lock(); err != nil {
		return "", err
	}
	defer Unlock()
	short := versions.ShortVersion(release.Version)
	destDir := filepath.Join(VersionsDir(), short)

	if _, err := os.Stat(filepath.Join(destDir, "go")); err == nil {
		if overwrite {
			if err := os.RemoveAll(destDir); err != nil {
				return "", fmt.Errorf("remove existing install %s: %w", short, err)
			}
		} else {
			return "", fmt.Errorf("version %s is already installed at %s", short, destDir)
		}
	} else {
		// Clean up any partial install (missing go/ subdirectory).
		_ = os.RemoveAll(destDir)
	}

	goroot, err := download.Download(ctx, release, destDir, archOverride)
	if err != nil {
		os.RemoveAll(destDir)
		return "", fmt.Errorf("install %s: %w", short, err)
	}

	fmt.Fprintf(os.Stderr, "Installed Go %s to %s\n", short, goroot)
	return goroot, nil
}

// IsInstalled checks whether a version is installed.
func IsInstalled(version string) bool {
	goroot := Goroot(version)
	info, err := os.Stat(goroot)
	return err == nil && info.IsDir()
}

// ListInstalled returns all installed version strings, sorted newest first.
func ListInstalled() ([]string, error) {
	dir := VersionsDir()
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("read versions dir: %w", err)
	}

	var result []string
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		// Only count directories that contain a 'go' subdirectory.
		if info, err := os.Stat(filepath.Join(dir, e.Name(), "go")); err == nil && info.IsDir() {
			result = append(result, e.Name())
		}
	}

	// Sort newest first (descending by version).
	sort.Slice(result, func(i, j int) bool {
		// semver.Compare expects "v"-prefixed versions.
		return semver.Compare("v"+result[i], "v"+result[j]) > 0
	})

	return result, nil
}

// Uninstall removes an installed version from the store.
func Uninstall(version string) error {
	if err := Lock(); err != nil {
		return err
	}
	defer Unlock()

	if !IsInstalled(version) {
		return fmt.Errorf("version %s is not installed", version)
	}

	// Don't allow uninstalling the active default.
	if current, err := Current(); err == nil && current == version {
		return fmt.Errorf("cannot uninstall the currently active default version %s — switch to another version first", version)
	}

	dir := filepath.Join(VersionsDir(), version)
	return os.RemoveAll(dir)
}

// SetDefault sets the global default Go version via symlink.
func SetDefault(version string) error {
	if err := Lock(); err != nil {
		return err
	}
	defer Unlock()

	if !IsInstalled(version) {
		return fmt.Errorf("version %s is not installed — run `govm install %s` first", version, version)
	}

	goroot := Goroot(version)
	link := CurrentLink()

	// Remove existing symlink (or regular file/dir).
	_ = os.Remove(link)

	if err := os.Symlink(goroot, link); err != nil {
		// Symlink may not be supported (e.g. some Windows configs).
		// Fall back to a file containing the path.
		return os.WriteFile(link, []byte(goroot), 0o644)
	}
	return nil
}

// Current returns the version string of the global default.
func Current() (string, error) {
	link := CurrentLink()

	target, err := os.Readlink(link)
	if err != nil {
		// Try reading as a fallback file.
		data, ferr := os.ReadFile(link)
		if ferr != nil {
			return "", fmt.Errorf("no default version set — run `govm default <version>`")
		}
		target = string(data)
	}

	// Parse version from GOROOT path: .../versions/<ver>/go → <ver>
	versionsDir := VersionsDir()
	if !filepath.IsAbs(target) {
		target = filepath.Join(filepath.Dir(link), target)
	}
	target = filepath.Clean(target)

	// Walk up from target: .../versions/1.23.4/go → 1.23.4
	rel, err := filepath.Rel(versionsDir, target)
	if err != nil {
		return "", fmt.Errorf("resolve current version: %w", err)
	}

	// rel should be like "1.23.4/go" (or "1.23.4\go" on Windows)
	return filepath.Dir(rel), nil
}

// Which returns the full path to the go binary for the current default.
func Which() (string, error) {
	_, err := Current()
	if err != nil {
		return "", err
	}

	link := CurrentLink()
	target, err := os.Readlink(link)
	if err != nil {
		data, ferr := os.ReadFile(link)
		if ferr != nil {
			return "", fmt.Errorf("no default version set")
		}
		target = string(data)
	}
	if !filepath.IsAbs(target) {
		target = filepath.Join(filepath.Dir(link), target)
	}

	goBin := filepath.Join(target, "bin", "go")
	if runtime.GOOS == "windows" {
		goBin += ".exe"
	}
	return goBin, nil
}

// Goroot returns the GOROOT path for an installed version.
func Goroot(version string) string {
	return filepath.Join(VersionsDir(), version, "go")
}


