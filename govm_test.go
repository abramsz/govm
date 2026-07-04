package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// TestCLI builds the govm binary and runs basic commands against it.
// Skips in short mode because it compiles the binary.
func TestCLI(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	dir := t.TempDir()
	binPath := filepath.Join(dir, "govm")
	if os.PathSeparator == '\\' {
		binPath += ".exe"
	}

	// Build the binary.
	cmd := exec.Command("go", "build", "-o", binPath, ".")
	cmd.Dir = filepath.Dir(".") // project root
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("build failed: %v\n%s", err, out)
	}

	// Use a separate govm home dir for isolation.
	govmHome := filepath.Join(dir, "govm_home")
	t.Setenv("GOVM_HOME", govmHome)

	t.Run("help", func(t *testing.T) {
		out, err := exec.Command(binPath, "--help").CombinedOutput()
		if err != nil {
			t.Fatalf("govm --help failed: %v\n%s", err, out)
		}
		if !strings.Contains(string(out), "govm") {
			t.Errorf("help output should contain 'govm':\n%s", out)
		}
	})

	t.Run("list_empty", func(t *testing.T) {
		out, err := exec.Command(binPath, "list").CombinedOutput()
		if err != nil {
			t.Fatalf("govm list failed: %v\n%s", err, out)
		}
		if !strings.Contains(string(out), "No versions installed") {
			t.Errorf("list should show empty message:\n%s", out)
		}
	})

	t.Run("current_not_set", func(t *testing.T) {
		// 'govm current' should exit with non-zero when no default is set.
		out, _ := exec.Command(binPath, "current").CombinedOutput()
		if !strings.Contains(string(out), "no default version set") {
			t.Errorf("current should complain about no default:\n%s", out)
		}
	})

	t.Run("which_not_set", func(t *testing.T) {
		out, _ := exec.Command(binPath, "which").CombinedOutput()
		if !strings.Contains(string(out), "no default version set") {
			t.Errorf("which should complain about no default:\n%s", out)
		}
	})

	t.Run("env_bash", func(t *testing.T) {
		out, err := exec.Command(binPath, "env", "--shell", "bash").CombinedOutput()
		if err != nil {
			t.Fatalf("govm env --shell bash failed: %v\n%s", err, out)
		}
		if !strings.Contains(string(out), "govm() {") {
			t.Errorf("env bash should contain wrapper function:\n%s", out)
		}
	})

	t.Run("env_powershell", func(t *testing.T) {
		out, err := exec.Command(binPath, "env", "--shell", "powershell").CombinedOutput()
		if err != nil {
			t.Fatalf("govm env --shell powershell failed: %v\n%s", err, out)
		}
		if !strings.Contains(string(out), "function govm") {
			t.Errorf("env powershell should contain function:\n%s", out)
		}
	})

	t.Run("list_remote", func(t *testing.T) {
		// This test hits the real API. It's acceptable in non-short mode.
		out, err := exec.Command(binPath, "list-remote").CombinedOutput()
		if err != nil {
			t.Fatalf("govm list-remote failed: %v\n%s", err, out)
		}
		if !strings.Contains(string(out), "stable=") {
			t.Errorf("list-remote output should contain 'stable=':\n%s", out)
		}
	})
}
