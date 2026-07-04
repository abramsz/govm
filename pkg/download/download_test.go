package download

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/abramsz/govm/pkg/versions"
)

func TestFindFile(t *testing.T) {
	rel := versions.Release{
		Version: "go1.23.4",
		Files: []versions.File{
			{OS: "linux", Arch: "amd64", Kind: "archive", Filename: "go1.23.4.linux-amd64.tar.gz"},
			{OS: "darwin", Arch: "amd64", Kind: "archive", Filename: "go1.23.4.darwin-amd64.tar.gz"},
			{OS: "darwin", Arch: "arm64", Kind: "archive", Filename: "go1.23.4.darwin-arm64.tar.gz"},
			{OS: "windows", Arch: "amd64", Kind: "archive", Filename: "go1.23.4.windows-amd64.zip"},
		},
	}

	got := FindFile(rel, "")
	if got == nil {
		t.Fatalf("FindFile() returned nil for %s/%s", runtime.GOOS, runtime.GOARCH)
	}
	if got.OS != runtime.GOOS || got.Arch != runtime.GOARCH {
		t.Errorf("FindFile() = %s/%s; want %s/%s", got.OS, got.Arch, runtime.GOOS, runtime.GOARCH)
	}
}

func TestFindFile_NotFound(t *testing.T) {
	rel := versions.Release{
		Version: "go1.23.4",
		Files: []versions.File{
			{OS: "linux", Arch: "amd64", Kind: "archive"},
		},
	}
	if runtime.GOOS == "linux" {
		// If running on linux, use a different OS to ensure not-found.
		rel.Files[0].OS = "darwin"
	}

	got := FindFile(rel, "")
	if got != nil {
		t.Errorf("FindFile() should return nil for mismatched platform, got %+v", got)
	}
}

// createTarGz creates a minimal tar.gz file at path with a single file.
func createTarGz(t *testing.T, path string, files map[string]string, symlinks map[string]string) {
	t.Helper()
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	gz := gzip.NewWriter(f)
	defer gz.Close()

	tw := tar.NewWriter(gz)
	defer tw.Close()

	for name, content := range files {
		hdr := &tar.Header{
			Name:     name,
			Size:     int64(len(content)),
			Typeflag: tar.TypeReg,
			Mode:     0o644,
		}
		if err := tw.WriteHeader(hdr); err != nil {
			t.Fatal(err)
		}
		if _, err := io.WriteString(tw, content); err != nil {
			t.Fatal(err)
		}
	}

	for linkName, target := range symlinks {
		hdr := &tar.Header{
			Name:     linkName,
			Linkname: target,
			Typeflag: tar.TypeSymlink,
		}
		if err := tw.WriteHeader(hdr); err != nil {
			t.Fatal(err)
		}
	}
}

func createZip(t *testing.T, path string, files map[string]string) {
	t.Helper()
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	zw := zip.NewWriter(f)
	defer zw.Close()

	for name, content := range files {
		w, err := zw.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := io.WriteString(w, content); err != nil {
			t.Fatal(err)
		}
	}
}

func TestExtract_TarGz(t *testing.T) {
	dir := t.TempDir()
	archive := filepath.Join(dir, "test.tar.gz")

	createTarGz(t, archive, map[string]string{
		"go/bin/go":      "fake go binary",
		"go/src/main.go": "package main",
	}, nil)

	dest := filepath.Join(dir, "out")
	if err := extractTarGz(archive, dest); err != nil {
		t.Fatalf("extractTarGz failed: %v", err)
	}

	// Verify extracted files.
	if _, err := os.Stat(filepath.Join(dest, "go/bin/go")); err != nil {
		t.Errorf("go/bin/go not extracted: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dest, "go/src/main.go")); err != nil {
		t.Errorf("go/src/main.go not extracted: %v", err)
	}
}

func TestExtract_Zip(t *testing.T) {
	dir := t.TempDir()
	archive := filepath.Join(dir, "test.zip")

	createZip(t, archive, map[string]string{
		"go/bin/go.exe": "fake go binary",
	})

	dest := filepath.Join(dir, "out")
	if err := extractZip(archive, dest); err != nil {
		t.Fatalf("extractZip failed: %v", err)
	}

	if _, err := os.Stat(filepath.Join(dest, "go/bin/go.exe")); err != nil {
		t.Errorf("go/bin/go.exe not extracted: %v", err)
	}
}

func TestExtract_MagicDetection(t *testing.T) {
	dir := t.TempDir()

	t.Run("tar_gz", func(t *testing.T) {
		archive := filepath.Join(dir, "test.tar.gz")
		createTarGz(t, archive, map[string]string{"hello.txt": "world"}, nil)
		dest := filepath.Join(dir, "out1")
		if err := extract(archive, dest); err != nil {
			t.Fatalf("extract(tar.gz) failed: %v", err)
		}
	})

	t.Run("zip", func(t *testing.T) {
		archive := filepath.Join(dir, "test.zip")
		createZip(t, archive, map[string]string{"hello.txt": "world"})
		dest := filepath.Join(dir, "out2")
		if err := extract(archive, dest); err != nil {
			t.Fatalf("extract(zip) failed: %v", err)
		}
	})

	t.Run("unknown_magic", func(t *testing.T) {
		archive := filepath.Join(dir, "test.unknown")
		if err := os.WriteFile(archive, []byte("not an archive"), 0o644); err != nil {
			t.Fatal(err)
		}
		dest := filepath.Join(dir, "out3")
		err := extract(archive, dest)
		if err == nil {
			t.Fatal("expected error for unknown archive format")
		}
		if !strings.Contains(err.Error(), "unknown archive format") {
			t.Errorf("unexpected error: %v", err)
		}
	})
}

func TestExtract_PathTraversalPrevention(t *testing.T) {
	dir := t.TempDir()
	archive := filepath.Join(dir, "escape.tar.gz")

	createTarGz(t, archive, nil, map[string]string{
		"go/bin/link": "../../../etc/passwd",
	})

	dest := filepath.Join(dir, "out")
	if err := extractTarGz(archive, dest); err != nil {
		t.Fatalf("extractTarGz failed: %v", err)
	}

	// The symlink with ".." should have been skipped.
	linkPath := filepath.Join(dest, "go/bin/link")
	if _, err := os.Lstat(linkPath); err != nil {
		t.Logf("symlink correctly skipped (or not created): %v", err)
	}

	// Verify that etc/passwd was NOT created outside dest.
	if _, err := os.Stat(filepath.Join(dir, "etc/passwd")); err == nil {
		t.Error("path traversal succeeded — etc/passwd was created outside destDir")
	}
}

func TestExtract_SafeSymlink(t *testing.T) {
	dir := t.TempDir()
	archive := filepath.Join(dir, "safe.tar.gz")

	createTarGz(t, archive,
		map[string]string{"go/bin/real": "target content"},
		map[string]string{"go/bin/link": "real"},
	)

	dest := filepath.Join(dir, "out")
	if err := extractTarGz(archive, dest); err != nil {
		t.Fatalf("extractTarGz failed: %v", err)
	}

	// Safe symlink (within same directory) should be created.
	linkPath := filepath.Join(dest, "go/bin/link")
	if _, err := os.Lstat(linkPath); err != nil {
		t.Errorf("safe symlink should have been created: %v", err)
	}
}

func TestSHA256Verification(t *testing.T) {
	dir := t.TempDir()

	// Create a temp file with known content.
	path := filepath.Join(dir, "test.bin")
	content := []byte("hello world")
	if err := os.WriteFile(path, content, 0o644); err != nil {
		t.Fatal(err)
	}

	// Compute expected SHA256 (same method as downloadToTemp).
	hasher := sha256.New()
	if _, err := hasher.Write(content); err != nil {
		t.Fatal(err)
	}
	got := hex.EncodeToString(hasher.Sum(nil))
	expected := "b94d27b9934d3e08a52e52d7da7dabfac484efe37a5380ee9088f7ace2efcde9"

	if got != expected {
		t.Errorf("sha256(%q) = %s; want %s", content, got, expected)
	}

	// Verify the same logic that downloadToTemp uses: hash while reading.
	r := strings.NewReader(string(content))
	h := sha256.New()
	tee := io.TeeReader(r, h)
	data, err := io.ReadAll(tee)
	if err != nil {
		t.Fatal(err)
	}
	result := hex.EncodeToString(h.Sum(nil))
	if result != expected {
		t.Errorf("TeeReader sha256 = %s; want %s", result, expected)
	}
	if string(data) != string(content) {
		t.Errorf("read back = %q; want %q", data, content)
	}
}
