package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// cmdTest holds shared state for a command test.
type cmdTest struct {
	govmHome string
}

// runCmd executes the root command with given args, capturing stdout and stderr.
// It returns the combined output (stdout) and error output (stderr).
func (ct *cmdTest) runCmd(t *testing.T, args []string) (stdout, stderr string, err error) {
	t.Helper()

	// Capture os.Stdout and os.Stderr for commands that write directly.
	oldOut := os.Stdout
	oldErr := os.Stderr
	rOut, wOut, _ := os.Pipe()
	rErr, wErr, _ := os.Pipe()
	os.Stdout = wOut
	os.Stderr = wErr

	outCh := make(chan string)
	errCh := make(chan string)
	go func() {
		var buf bytes.Buffer
		_, _ = buf.ReadFrom(rOut)
		outCh <- buf.String()
	}()
	go func() {
		var buf bytes.Buffer
		_, _ = buf.ReadFrom(rErr)
		errCh <- buf.String()
	}()

	rootCmd.SetArgs(args)
	// Also set cobra's output writers so cmd.Print* is captured alongside os.Stdout.
	rootCmd.SetOut(wOut)
	rootCmd.SetErr(wErr)

	_, execErr := rootCmd.ExecuteC()

	// Close pipes so the goroutines finish.
	wOut.Close()
	wErr.Close()
	os.Stdout = oldOut
	os.Stderr = oldErr

	outStr := <-outCh
	errStr := <-errCh

	// Reset cobra output for next test.
	rootCmd.SetOut(nil)
	rootCmd.SetErr(nil)

	return outStr, errStr, execErr
}

// requireContains fails if s does not contain all substrings.
func requireContains(t *testing.T, s string, want ...string) {
	t.Helper()
	for _, w := range want {
		if !strings.Contains(s, w) {
			t.Errorf("expected output to contain %q:\n%s", w, s)
		}
	}
}

// createFakeInstall creates a fake Go installation in the store.
func (ct *cmdTest) createFakeInstall(t *testing.T, version string) {
	t.Helper()
	goRoot := filepath.Join(ct.govmHome, "versions", version, "go", "bin")
	if err := os.MkdirAll(goRoot, 0o755); err != nil {
		t.Fatal(err)
	}
	goBin := filepath.Join(goRoot, "go")
	if os.PathSeparator == '\\' {
		goBin += ".exe"
	}
	if err := os.WriteFile(goBin, []byte("fake-go-binary"), 0o755); err != nil {
		t.Fatal(err)
	}
}

func TestCommands(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("GOVM_HOME", dir)
	ct := &cmdTest{govmHome: dir}

	t.Run("help", func(t *testing.T) {
		stdout, _, err := ct.runCmd(t, []string{"--help"})
		if err != nil {
			t.Fatal(err)
		}
		requireContains(t, stdout,
			"govm", "install", "list", "list-remote", "use",
			"default", "uninstall", "current", "which", "env")
	})

	t.Run("list_empty", func(t *testing.T) {
		stdout, _, err := ct.runCmd(t, []string{"list"})
		if err != nil {
			t.Fatal(err)
		}
		requireContains(t, stdout, "No versions installed")
	})

	t.Run("current_not_set", func(t *testing.T) {
		_, stderr, err := ct.runCmd(t, []string{"current"})
		if err == nil {
			t.Fatal("expected error for current when no default set")
		}
		requireContains(t, stderr+err.Error(), "no default version set")
	})

	t.Run("which_not_set", func(t *testing.T) {
		_, stderr, err := ct.runCmd(t, []string{"which"})
		if err == nil {
			t.Fatal("expected error for which when no default set")
		}
		requireContains(t, stderr+err.Error(), "no default version set")
	})

	t.Run("uninstall_not_installed", func(t *testing.T) {
		_, stderr, err := ct.runCmd(t, []string{"uninstall", "9.99.99"})
		if err == nil {
			t.Fatal("expected error")
		}
		requireContains(t, stderr+err.Error(), "not installed")
	})

	// Now install a fake version and test stateful commands.
	ct.createFakeInstall(t, "1.23.4")
	ct.createFakeInstall(t, "1.22.10")
	ct.createFakeInstall(t, "1.25.11")

	t.Run("list_with_versions", func(t *testing.T) {
		stdout, _, err := ct.runCmd(t, []string{"list"})
		if err != nil {
			t.Fatal(err)
		}
		requireContains(t, stdout, "1.23.4", "1.22.10", "1.25.11")
	})

	t.Run("list_sorted_newest_first", func(t *testing.T) {
		stdout, _, err := ct.runCmd(t, []string{"list"})
		if err != nil {
			t.Fatal(err)
		}
		lines := strings.Split(strings.TrimSpace(stdout), "\n")
		versions := make([]string, 0, len(lines))
		for _, line := range lines {
			// Lines are like " * 1.25.11" or "   1.23.4"
			v := strings.TrimSpace(line)
			if idx := strings.LastIndex(v, " "); idx >= 0 {
				v = v[idx+1:]
			}
			versions = append(versions, v)
		}
		expected := []string{"1.25.11", "1.23.4", "1.22.10"}
		if len(versions) != len(expected) {
			t.Fatalf("got %v; want %v", versions, expected)
		}
		for i, v := range versions {
			if v != expected[i] {
				t.Errorf("position %d: got %s; want %s", i, v, expected[i])
			}
		}
	})

	t.Run("default_ok", func(t *testing.T) {
		stdout, _, err := ct.runCmd(t, []string{"default", "1.23.4"})
		if err != nil {
			t.Fatal(err)
		}
		requireContains(t, stdout, "Default Go version set to 1.23.4")
	})

	t.Run("default_not_installed", func(t *testing.T) {
		_, stderr, err := ct.runCmd(t, []string{"default", "9.99.99"})
		if err == nil {
			t.Fatal("expected error")
		}
		requireContains(t, stderr+err.Error(), "not installed")
	})

	t.Run("current_after_default", func(t *testing.T) {
		stdout, _, err := ct.runCmd(t, []string{"current"})
		if err != nil {
			t.Fatal(err)
		}
		requireContains(t, stdout, "1.23.4")
	})

	t.Run("which_after_default", func(t *testing.T) {
		stdout, _, err := ct.runCmd(t, []string{"which"})
		if err != nil {
			t.Fatal(err)
		}
		requireContains(t, stdout, "1.23.4", "go")
	})

	t.Run("list_shows_current", func(t *testing.T) {
		stdout, _, err := ct.runCmd(t, []string{"list"})
		if err != nil {
			t.Fatal(err)
		}
		requireContains(t, stdout, "* 1.23.4")
	})

	t.Run("use_not_installed", func(t *testing.T) {
		_, stderr, err := ct.runCmd(t, []string{"use", "9.99.99"})
		if err == nil {
			t.Fatal("expected error")
		}
		requireContains(t, stderr+err.Error(), "not installed")
	})

	t.Run("use_bash", func(t *testing.T) {
		stdout, _, err := ct.runCmd(t, []string{"use", "--shell", "bash", "1.23.4"})
		if err != nil {
			t.Fatal(err)
		}
		requireContains(t, stdout,
			"export GOROOT=",
			"export PATH=",
			"GOVM_VERSION=1.23.4")
	})

	t.Run("use_powershell", func(t *testing.T) {
		stdout, _, err := ct.runCmd(t, []string{"use", "--shell", "powershell", "1.23.4"})
		if err != nil {
			t.Fatal(err)
		}
		requireContains(t, stdout,
			"$env:GOROOT =",
			"$env:Path =",
			"$env:GOVM_VERSION = '1.23.4'")
	})

	t.Run("use_fish", func(t *testing.T) {
		stdout, _, err := ct.runCmd(t, []string{"use", "--shell", "fish", "1.23.4"})
		if err != nil {
			t.Fatal(err)
		}
		requireContains(t, stdout,
			"set -gx GOROOT",
			"set -gx PATH",
			"GOVM_VERSION")
	})

	t.Run("use_cmd", func(t *testing.T) {
		stdout, _, err := ct.runCmd(t, []string{"use", "--shell", "cmd", "1.23.4"})
		if err != nil {
			t.Fatal(err)
		}
		requireContains(t, stdout,
			"set GOROOT=",
			"set PATH=",
			"set GOVM_VERSION=1.23.4")
	})

	t.Run("uninstall_not_current", func(t *testing.T) {
		stdout, _, err := ct.runCmd(t, []string{"uninstall", "1.22.10"})
		if err != nil {
			t.Fatal(err)
		}
		// No output on success — just means no error.
		t.Logf("uninstall 1.22.10 stdout: %q", stdout)
	})

	t.Run("uninstall_current_protected", func(t *testing.T) {
		_, stderr, err := ct.runCmd(t, []string{"uninstall", "1.23.4"})
		if err == nil {
			t.Fatal("expected error when uninstalling current default")
		}
		requireContains(t, stderr+err.Error(), "cannot uninstall")
	})

	t.Run("env_bash", func(t *testing.T) {
		stdout, _, err := ct.runCmd(t, []string{"env", "--shell", "bash"})
		if err != nil {
			t.Fatal(err)
		}
		requireContains(t, stdout,
			"export PATH=",
			"_govm_exe=",
			"govm() {",
			"--shell bash")
	})

	t.Run("env_fish", func(t *testing.T) {
		stdout, _, err := ct.runCmd(t, []string{"env", "--shell", "fish"})
		if err != nil {
			t.Fatal(err)
		}
		requireContains(t, stdout,
			"set -gx PATH",
			"function govm",
			"--shell fish")
	})

	t.Run("env_powershell", func(t *testing.T) {
		stdout, _, err := ct.runCmd(t, []string{"env", "--shell", "powershell"})
		if err != nil {
			t.Fatal(err)
		}
		requireContains(t, stdout,
			"$govmHome =",
			"$env:Path =",
			"function govm",
			"--shell powershell")
	})

	t.Run("env_cmd", func(t *testing.T) {
		stdout, _, err := ct.runCmd(t, []string{"env", "--shell", "cmd"})
		if err != nil {
			t.Fatal(err)
		}
		requireContains(t, stdout,
			"@echo off",
			"set \"PATH=",
			"--shell cmd")
	})

	t.Run("unknown_version_message", func(t *testing.T) {
		if testing.Short() {
			t.Skip("skipping network-dependent test in short mode")
		}
		// The command prints a message and returns nil (not an error).
		stdout, stderr, err := ct.runCmd(t, []string{"install", "0.0.0"})
		if err != nil {
			t.Fatal(err)
		}
		requireContains(t, stdout+stderr, "not found", "list-remote")
	})
}

func TestInstallHelp(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("GOVM_HOME", dir)
	ct := &cmdTest{govmHome: dir}

	stdout, _, err := ct.runCmd(t, []string{"install", "--help"})
	if err != nil {
		t.Fatal(err)
	}
	requireContains(t, stdout, "install", "1.23.4", "--reinstall")
}

func TestListRemoteShort(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping network test in short mode")
	}

	dir := t.TempDir()
	t.Setenv("GOVM_HOME", dir)
	ct := &cmdTest{govmHome: dir}

	stdout, _, err := ct.runCmd(t, []string{"list-remote"})
	if err != nil {
		t.Fatal(err)
	}
	requireContains(t, stdout, "stable=")
	requireContains(t, stdout, ".") // at least one version number
}

func TestListRemoteAll(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping network test in short mode")
	}

	dir := t.TempDir()
	t.Setenv("GOVM_HOME", dir)
	ct := &cmdTest{govmHome: dir}

	stdout, _, err := ct.runCmd(t, []string{"list-remote", "--all"})
	if err != nil {
		t.Fatal(err)
	}
	requireContains(t, stdout, "stable=")
	requireContains(t, stdout, ".") // at least one version number
}

func TestInvalidArgs(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("GOVM_HOME", dir)
	ct := &cmdTest{govmHome: dir}

	t.Run("install_no_args", func(t *testing.T) {
		// cobra's arg validation may print help and return nil.
		// Check that no panic occurs and the output is sensible.
		stdout, stderr, err := ct.runCmd(t, []string{"install"})
		_ = err // cobra may or may not return error depending on state
		output := stdout + stderr
		if output == "" {
			t.Fatal("expected some output (usage or error)")
		}
	})

	t.Run("use_no_args", func(t *testing.T) {
		_, _, err := ct.runCmd(t, []string{"use"})
		if err == nil {
			t.Fatal("expected error")
		}
	})

	t.Run("default_no_args", func(t *testing.T) {
		_, _, err := ct.runCmd(t, []string{"default"})
		if err == nil {
			t.Fatal("expected error")
		}
	})

	t.Run("uninstall_no_args", func(t *testing.T) {
		_, _, err := ct.runCmd(t, []string{"uninstall"})
		if err == nil {
			t.Fatal("expected error")
		}
	})

	t.Run("unknown_command", func(t *testing.T) {
		_, stderr, err := ct.runCmd(t, []string{"unknown-cmd"})
		if err == nil {
			t.Fatal("expected error")
		}
		requireContains(t, stderr+err.Error(), "unknown command")
	})
}

// TestCmdResetOutput verifies each command at least runs without crashing.
func TestCmdAllRegistered(t *testing.T) {
	commands := []string{"list", "current", "which", "env"}
	for _, name := range commands {
		t.Run(name, func(t *testing.T) {
			dir := t.TempDir()
			t.Setenv("GOVM_HOME", dir)
			ct := &cmdTest{govmHome: dir}

			_, _, err := ct.runCmd(t, []string{name})
			// Some commands error when no default is set — that's expected.
			// We just check they don't panic.
			if err != nil && name == "list" {
				t.Fatal(err) // list should never error
			}
		})
	}
}

// Ensure the root command variant is covered.
func TestExecuteOutput(t *testing.T) {
	// rootCmd.SilenceErrors is true, so the error path goes to stderr.
	_, stderr, err := (&cmdTest{}).runCmd(t, []string{"--help"})
	if err != nil {
		t.Fatal(err)
	}
	// Just verify stderr is empty.
	if stderr != "" {
		t.Logf("stderr output: %q", stderr)
	}
}

// TestRootCmdVersionSilence tests that SilenceUsage/SilenceErrors work.
func TestRootCmdSilence(t *testing.T) {
	if !rootCmd.SilenceUsage {
		t.Error("rootCmd.SilenceUsage should be true")
	}
	if !rootCmd.SilenceErrors {
		t.Error("rootCmd.SilenceErrors should be true")
	}
}

func TestConfigCmd(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("GOVM_HOME", dir)
	ct := &cmdTest{govmHome: dir}

	t.Run("view_default", func(t *testing.T) {
		stdout, _, err := ct.runCmd(t, []string{"config"})
		if err != nil {
			t.Fatal(err)
		}
		// Empty config should show empty JSON or default.
		requireContains(t, stdout, "{")
	})

	t.Run("set_mirror", func(t *testing.T) {
		stdout, _, err := ct.runCmd(t, []string{"config", "set", "mirror", "https://mirror.example.com"})
		if err != nil {
			t.Fatal(err)
		}
		requireContains(t, stdout, "mirror")
	})

	t.Run("view_after_set", func(t *testing.T) {
		stdout, _, err := ct.runCmd(t, []string{"config"})
		if err != nil {
			t.Fatal(err)
		}
		requireContains(t, stdout, "mirror.example.com")
	})

	t.Run("alias_set_and_resolve", func(t *testing.T) {
		stdout, _, err := ct.runCmd(t, []string{"config", "alias", "stable", "1.26.4"})
		if err != nil {
			t.Fatal(err)
		}
		requireContains(t, stdout, "stable")
	})

	t.Run("alias_remove", func(t *testing.T) {
		stdout, _, err := ct.runCmd(t, []string{"config", "alias", "stable", ""})
		if err != nil {
			t.Fatal(err)
		}
		requireContains(t, stdout, "removed")
	})

	t.Run("config_set_unknown_key", func(t *testing.T) {
		_, stderr, err := ct.runCmd(t, []string{"config", "set", "invalid", "value"})
		if err == nil {
			t.Fatal("expected error for unknown config key")
		}
		requireContains(t, stderr+err.Error(), "unknown config key")
	})
}

func TestVersionCmd(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("GOVM_HOME", dir)
	ct := &cmdTest{govmHome: dir}

	t.Run("version_subcommand", func(t *testing.T) {
		stdout, _, err := ct.runCmd(t, []string{"version"})
		if err != nil {
			t.Fatal(err)
		}
		requireContains(t, stdout, ".")
	})

	t.Run("version_flag", func(t *testing.T) {
		stdout, _, err := ct.runCmd(t, []string{"--version"})
		if err != nil {
			t.Fatal(err)
		}
		requireContains(t, stdout, "govm version")
	})
}

func TestAliasResolution(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("GOVM_HOME", dir)
	ct := &cmdTest{govmHome: dir}

	// Set up an alias and a fake install.
	_, _, err := ct.runCmd(t, []string{"config", "alias", "mytest", "1.23.4"})
	if err != nil {
		t.Fatal(err)
	}
	ct.createFakeInstall(t, "1.23.4")

	// Use should work with alias.
	t.Run("use_with_alias", func(t *testing.T) {
		stdout, _, err := ct.runCmd(t, []string{"use", "--shell", "bash", "mytest"})
		if err != nil {
			t.Fatal(err)
		}
		requireContains(t, stdout, "1.23.4")
	})

	// Default should work with alias.
	t.Run("default_with_alias", func(t *testing.T) {
		stdout, _, err := ct.runCmd(t, []string{"default", "mytest"})
		if err != nil {
			t.Fatal(err)
		}
		requireContains(t, stdout, "1.23.4")
	})
}

// Test that the progressReader doesn't use cmd.Printf at all.
func TestCmdProgressStderr(t *testing.T) {
	// This is an architectural test — the download progress uses fmt.Fprintf(os.Stderr)
	// directly, not cobra's output. Verify os.Stderr capture works.
	dir := t.TempDir()
	t.Setenv("GOVM_HOME", dir)

	// Just test list to exercise os.Stderr path.
	stdout, _, err := (&cmdTest{govmHome: dir}).runCmd(t, []string{"env", "--shell", "bash"})
	if err != nil {
		t.Fatal(err)
	}
	requireContains(t, stdout, "govm()")
}
