package shell

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

// ShellType represents a supported shell.
type ShellType string

const (
	ShellBash       ShellType = "bash"
	ShellZsh        ShellType = "zsh"
	ShellFish       ShellType = "fish"
	ShellPowerShell ShellType = "powershell"
	ShellCmd        ShellType = "cmd"
)

// DetectShell attempts to auto-detect the current shell from environment.
func DetectShell() ShellType {
	if runtime.GOOS == "windows" {
		if os.Getenv("PSModulePath") != "" {
			return ShellPowerShell
		}
		return ShellCmd
	}

	shellPath := os.Getenv("SHELL")
	if shellPath != "" {
		base := filepath.Base(shellPath)
		switch base {
		case "zsh":
			return ShellZsh
		case "fish":
			return ShellFish
		default:
			return ShellBash
		}
	}
	return ShellBash
}

// EnvScript returns shell initialization code for the given shell type.
func EnvScript(shell ShellType, govmHome, govmExe string) string {
	switch shell {
	case ShellBash, ShellZsh:
		return envBashLike(govmHome, govmExe)
	case ShellFish:
		return envFish(govmHome, govmExe)
	case ShellPowerShell:
		return envPowerShell(govmHome, govmExe)
	case ShellCmd:
		return envCmd(govmHome, govmExe)
	default:
		return envBashLike(govmHome, govmExe)
	}
}

func envBashLike(govmHome, govmExe string) string {
	govmHome = filepath.ToSlash(govmHome)
	govmExe = filepath.ToSlash(filepath.Clean(govmExe))
	govmDir := filepath.ToSlash(filepath.Dir(govmExe))
	currentBin := govmHome + "/current/bin"

	var sb strings.Builder
	fmt.Fprintf(&sb, "export PATH=%q:%q:$PATH\n", govmDir, currentBin)
	sb.WriteString("\n")
	sb.WriteString("# govm wrapper function — makes `govm use` work without eval\n")
	fmt.Fprintf(&sb, "_govm_exe=%q\n", govmExe)
	sb.WriteString("govm() {\n")
	sb.WriteString("  case \"$1\" in\n")
	sb.WriteString("    use)\n")
	sb.WriteString("      eval \"$(\"$_govm_exe\" use --shell bash \"$2\")\"\n")
	sb.WriteString("      ;;\n")
	sb.WriteString("    *)\n")
	sb.WriteString("      \"$_govm_exe\" \"$@\"\n")
	sb.WriteString("      ;;\n")
	sb.WriteString("  esac\n")
	sb.WriteString("}\n")
	return sb.String()
}

func envFish(govmHome, govmExe string) string {
	govmHome = filepath.ToSlash(govmHome)
	govmExe = filepath.ToSlash(filepath.Clean(govmExe))
	govmDir := filepath.ToSlash(filepath.Dir(govmExe))
	currentBin := govmHome + "/current/bin"

	var sb strings.Builder
	fmt.Fprintf(&sb, "set -gx PATH %q %q $PATH\n", govmDir, currentBin)
	sb.WriteString("\n")
	sb.WriteString("# govm wrapper function\n")
	fmt.Fprintf(&sb, "set -g _govm_exe %q\n", govmExe)
	sb.WriteString("function govm\n")
	sb.WriteString("  switch $argv[1]\n")
	sb.WriteString("    case use\n")
	sb.WriteString("      eval ($_govm_exe use --shell fish $argv[2])\n")
	sb.WriteString("    case '*'\n")
	sb.WriteString("      $_govm_exe $argv\n")
	sb.WriteString("  end\n")
	sb.WriteString("end\n")
	return sb.String()
}

func envPowerShell(govmHome, govmExe string) string {
	currentBin := filepath.Join(govmHome, "current", "bin")

	var sb strings.Builder
	fmt.Fprintf(&sb, "$govmHome = %q\n", govmHome)
	fmt.Fprintf(&sb, "$env:Path = \"%s;%s;$env:Path\"\n", currentBin, filepath.Dir(govmExe))
	sb.WriteString("\n")
	sb.WriteString("# govm wrapper function — hardcoded path avoids chicken-egg problem\n")
	fmt.Fprintf(&sb, "$govmExe = %q\n", govmExe)
	sb.WriteString("function govm {\n")
	sb.WriteString("  if ($args[0] -eq 'use') {\n")
	sb.WriteString("    & $govmExe use --shell powershell $args[1] | Out-String | Invoke-Expression\n")
	sb.WriteString("  } else {\n")
	sb.WriteString("    & $govmExe $args\n")
	sb.WriteString("  }\n")
	sb.WriteString("}\n")
	return sb.String()
}

func envCmd(govmHome, govmExe string) string {
	govmBinDir := filepath.Dir(govmExe)
	currentBin := filepath.Join(govmHome, "current", "bin")

	var sb strings.Builder
	sb.WriteString("@echo off\n")
	fmt.Fprintf(&sb, "set \"PATH=%s;%s;%%PATH%%\"\n", govmBinDir, currentBin)
	sb.WriteString("\n")
	sb.WriteString("REM govm: add govm.exe dir + ~/.govm/current/bin to PATH\n")
	sb.WriteString("REM Note: cmd.exe cannot intercept 'govm use' — use eval:\n")
	fmt.Fprintf(&sb, "REM   for /f \"tokens=*\" %%%%i in ('%s use --shell cmd 1.23.4') do call %%%%i\n", govmExe)
	sb.WriteString("\n")
	return sb.String()
}
