package store

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestHome_Default(t *testing.T) {
	// Unset GOVM_HOME to test default.
	t.Setenv("GOVM_HOME", "")
	home := Home()
	if home == "" {
		t.Fatal("Home() returned empty")
	}
	if !filepath.IsAbs(home) {
		t.Errorf("Home() should be absolute: %s", home)
	}
	if filepath.Base(home) != ".govm" {
		t.Errorf("Home() base should be .govm: %s", home)
	}
}

func TestHome_GOVM_HOME(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("GOVM_HOME", dir)

	got := Home()
	if got != dir {
		t.Errorf("Home() = %s; want %s", got, dir)
	}
}

func TestVersionsDir(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("GOVM_HOME", dir)

	got := VersionsDir()
	want := filepath.Join(dir, "versions")
	if got != want {
		t.Errorf("VersionsDir() = %s; want %s", got, want)
	}
}

func TestCurrentLink(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("GOVM_HOME", dir)

	got := CurrentLink()
	want := filepath.Join(dir, "current")
	if got != want {
		t.Errorf("CurrentLink() = %s; want %s", got, want)
	}
}

func TestGoroot(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("GOVM_HOME", dir)

	got := Goroot("1.23.4")
	want := filepath.Join(dir, "versions", "1.23.4", "go")
	if got != want {
		t.Errorf("Goroot() = %s; want %s", got, want)
	}
}

func TestLock_AcquireAndRelease(t *testing.T) {
	// Speed up lock retry for testing.
	oldInterval := retryInterval
	oldMax := retryMax
	retryInterval = 10 * time.Millisecond
	retryMax = 10
	defer func() {
		retryInterval = oldInterval
		retryMax = oldMax
	}()

	dir := t.TempDir()
	t.Setenv("GOVM_HOME", dir)

	if err := Lock(); err != nil {
		t.Fatalf("Lock() failed: %v", err)
	}

	// Lock should fail while held.
	if err := Lock(); err == nil {
		t.Error("Lock() should fail when already held")
	}

	Unlock()

	// Lock should succeed after release.
	if err := Lock(); err != nil {
		t.Errorf("Lock() after Unlock failed: %v", err)
	}
	Unlock()
}

func TestLock_Twice(t *testing.T) {
	oldInterval := retryInterval
	oldMax := retryMax
	retryInterval = 10 * time.Millisecond
	retryMax = 10
	defer func() {
		retryInterval = oldInterval
		retryMax = oldMax
	}()

	dir := t.TempDir()
	t.Setenv("GOVM_HOME", dir)

	if err := Lock(); err != nil {
		t.Fatal(err)
	}
	defer Unlock()

	// Second Lock should time out.
	if err := Lock(); err == nil {
		t.Error("expected lock failure")
	}
}

func TestLock_CleanupOnUnlock(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("GOVM_HOME", dir)

	_ = Lock()
	Unlock()

	lockPath := filepath.Join(Home(), lockFileName)
	if _, err := os.Stat(lockPath); !os.IsNotExist(err) {
		t.Errorf("lock file should be removed after Unlock: %v", err)
	}
}

func TestIsInstalled_NotFound(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("GOVM_HOME", dir)

	if IsInstalled("9.99.99") {
		t.Error("IsInstalled should return false for non-existent version")
	}
}

func TestListInstalled_Sorting(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("GOVM_HOME", dir)

	// Create fake version directories with go/ subdirectory.
	versions := []string{"1.23.4", "1.25.11", "1.24.0", "1.22.10"}
	for _, v := range versions {
		goDir := filepath.Join(VersionsDir(), v, "go")
		if err := os.MkdirAll(goDir, 0o755); err != nil {
			t.Fatal(err)
		}
	}

	got, err := ListInstalled()
	if err != nil {
		t.Fatalf("ListInstalled() failed: %v", err)
	}

	// Expected: newest first.
	expected := []string{"1.25.11", "1.24.0", "1.23.4", "1.22.10"}
	if len(got) != len(expected) {
		t.Fatalf("ListInstalled() = %v; want %v", got, expected)
	}
	for i := range got {
		if got[i] != expected[i] {
			t.Errorf("ListInstalled()[%d] = %s; want %s", i, got[i], expected[i])
		}
	}
}

func TestListInstalled_Empty(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("GOVM_HOME", dir)

	got, err := ListInstalled()
	if err != nil {
		t.Fatalf("ListInstalled() failed: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("ListInstalled() should be empty, got %v", got)
	}
}

func TestUninstall_Guard(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("GOVM_HOME", dir)

	version := "1.23.4"
	// Create a fake installation.
	goDir := filepath.Join(VersionsDir(), version, "go")
	if err := os.MkdirAll(goDir, 0o755); err != nil {
		t.Fatal(err)
	}

	// Set it as default.
	if err := SetDefault(version); err != nil {
		t.Fatalf("SetDefault failed: %v", err)
	}

	// Uninstall should fail.
	err := Uninstall(version)
	if err == nil {
		t.Fatal("Uninstall should fail for active default version")
	}
}

func TestUninstall_NotInstalled(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("GOVM_HOME", dir)

	err := Uninstall("9.99.99")
	if err == nil {
		t.Fatal("Uninstall should fail for non-installed version")
	}
}

func TestSetDefault_NotInstalled(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("GOVM_HOME", dir)

	err := SetDefault("9.99.99")
	if err == nil {
		t.Fatal("SetDefault should fail for non-installed version")
	}
}

func TestCurrent_NotSet(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("GOVM_HOME", dir)

	_, err := Current()
	if err == nil {
		t.Fatal("Current() should fail when no default set")
	}
}

func TestWhich_NotSet(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("GOVM_HOME", dir)

	_, err := Which()
	if err == nil {
		t.Fatal("Which() should fail when no default set")
	}
}
