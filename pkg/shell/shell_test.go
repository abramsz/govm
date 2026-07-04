package shell

import (
	"strings"
	"testing"
)

func TestScript_POSIX(t *testing.T) {
	tests := []struct {
		name    string
		shell   ShellType
		goroot  string
		version string
		checks  []string // all must be present
	}{
		{
			name:    "bash",
			shell:   ShellBash,
			goroot:  "/home/user/.govm/versions/1.23.4/go",
			version: "1.23.4",
			checks:  []string{"export GOROOT=", "export PATH=", "GOVM_VERSION=1.23.4"},
		},
		{
			name:    "zsh",
			shell:   ShellZsh,
			goroot:  "/home/user/.govm/versions/1.23.4/go",
			version: "1.23.4",
			checks:  []string{"export GOROOT=", "export PATH=", "GOVM_VERSION=1.23.4"},
		},
		{
			name:    "fish",
			shell:   ShellFish,
			goroot:  "/home/user/.govm/versions/1.23.4/go",
			version: "1.23.4",
			checks:  []string{"set -gx GOROOT", "set -gx PATH", "GOVM_VERSION"},
		},
		{
			name:    "powershell",
			shell:   ShellPowerShell,
			goroot:  `C:\Users\test\.govm\versions\1.23.4\go`,
			version: "1.23.4",
			checks:  []string{"$env:GOROOT =", "$env:Path =", "$env:GOVM_VERSION = '1.23.4'"},
		},
		{
			name:    "cmd",
			shell:   ShellCmd,
			goroot:  `C:\Users\test\.govm\versions\1.23.4\go`,
			version: "1.23.4",
			checks:  []string{"set GOROOT=", "set PATH=", "set GOVM_VERSION=1.23.4"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Script(tt.shell, tt.goroot, tt.version)
			for _, c := range tt.checks {
				if !strings.Contains(got, c) {
					t.Errorf("Script(%q) = %q; want contains %q", tt.shell, got, c)
				}
			}
		})
	}
}

func TestScript_POSIX_NormalizesSlashes(t *testing.T) {
	// POSIX shells should get forward slashes even on Windows input.
	got := Script(ShellBash, `C:\Users\test\.govm\versions\1.23.4\go`, "1.23.4")
	if strings.Contains(got, `\`) {
		t.Errorf("bash script should not contain backslashes: %q", got)
	}
}

func TestScript_PowerShell_KeepsBackslashes(t *testing.T) {
	got := Script(ShellPowerShell, `C:\Users\test\.govm\versions\1.23.4\go`, "1.23.4")
	if !strings.Contains(got, `\`) {
		t.Errorf("PowerShell script should keep backslashes: %q", got)
	}
}

func TestDetectShell(t *testing.T) {
	// We can't easily mock os.Getenv, but we can verify it returns a valid value.
	shell := DetectShell()
	switch shell {
	case ShellBash, ShellZsh, ShellFish, ShellPowerShell, ShellCmd:
		// valid
	default:
		t.Errorf("DetectShell() returned unexpected value: %q", shell)
	}
}

func TestEnvScript_Bash(t *testing.T) {
	got := EnvScript(ShellBash, "/home/user/.govm", "/home/user/.govm/bin/govm")

	checks := []string{
		"export PATH=",
		"_govm_exe=",
		"govm() {",
		"eval",
		"--shell bash",
	}
	for _, c := range checks {
		if !strings.Contains(got, c) {
			t.Errorf("EnvScript(bash) missing %q:\n%s", c, got)
		}
	}
}

func TestEnvScript_Fish(t *testing.T) {
	got := EnvScript(ShellFish, "/home/user/.govm", "/home/user/.govm/bin/govm")

	checks := []string{
		"set -gx PATH",
		"_govm_exe",
		"function govm",
		"--shell fish",
	}
	for _, c := range checks {
		if !strings.Contains(got, c) {
			t.Errorf("EnvScript(fish) missing %q:\n%s", c, got)
		}
	}
}

func TestEnvScript_PowerShell(t *testing.T) {
	got := EnvScript(ShellPowerShell, `C:\Users\test\.govm`, `C:\Users\test\.govm\bin\govm.exe`)

	checks := []string{
		"$govmHome =",
		"$env:Path =",
		"$govmExe =",
		"function govm {",
		"--shell powershell",
	}
	for _, c := range checks {
		if !strings.Contains(got, c) {
			t.Errorf("EnvScript(powershell) missing %q:\n%s", c, got)
		}
	}
}

func TestEnvScript_Cmd(t *testing.T) {
	got := EnvScript(ShellCmd, `C:\Users\test\.govm`, `C:\Users\test\.govm\bin\govm.exe`)

	checks := []string{
		"@echo off",
		"set \"PATH=",
		"--shell cmd",
	}
	for _, c := range checks {
		if !strings.Contains(got, c) {
			t.Errorf("EnvScript(cmd) missing %q:\n%s", c, got)
		}
	}
}

func TestEnvScript_HardcodesBinaryPath(t *testing.T) {
	govmExe := "/custom/path/govm"
	got := EnvScript(ShellBash, "/home/user/.govm", govmExe)
	if !strings.Contains(got, govmExe) {
		t.Errorf("EnvScript should hardcode binary path %q:\n%s", govmExe, got)
	}
}

func TestScript_DefaultShellFallback(t *testing.T) {
	// Empty ShellType should produce POSIX output.
	got := Script("", "/goroot", "1.0.0")
	if !strings.HasPrefix(got, "export") {
		t.Errorf("Script with empty shell type should produce POSIX output: %q", got)
	}
}

func TestEnvScript_DefaultShellFallback(t *testing.T) {
	got := EnvScript("", "/home/user/.govm", "/home/user/.govm/bin/govm")
	if !strings.Contains(got, "export PATH=") {
		t.Errorf("EnvScript with empty shell type should produce bash output: %q", got)
	}
}
