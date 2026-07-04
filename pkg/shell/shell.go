// Package shell generates shell scripts for activating Go versions.
package shell

import (
	"fmt"
	"path/filepath"
	"strings"
)

// Script returns a shell script that activates the given GOROOT
// for the specified shell type.
func Script(shellType ShellType, goroot, version string) string {
	switch shellType {
	case ShellBash, ShellZsh:
		return scriptPOSIX(goroot, version)
	case ShellFish:
		return scriptFish(goroot, version)
	case ShellPowerShell:
		return scriptPowerShell(goroot, version)
	case ShellCmd:
		return scriptCmd(goroot, version)
	default:
		return scriptPOSIX(goroot, version)
	}
}

func scriptPOSIX(goroot, version string) string {
	goroot = strings.ReplaceAll(goroot, "\\", "/")
	var sb strings.Builder
	fmt.Fprintf(&sb, "export GOROOT='%s'\n", goroot)
	fmt.Fprintf(&sb, "export PATH='%s/bin':$PATH\n", goroot)
	fmt.Fprintf(&sb, "export GOVM_VERSION=%s\n", version)
	fmt.Fprintf(&sb, "echo 'govm: now using Go %s'\n", version)
	return sb.String()
}

func scriptFish(goroot, version string) string {
	goroot = strings.ReplaceAll(goroot, "\\", "/")
	var sb strings.Builder
	fmt.Fprintf(&sb, "set -gx GOROOT '%s'\n", goroot)
	fmt.Fprintf(&sb, "set -gx PATH '%s/bin' $PATH\n", goroot)
	fmt.Fprintf(&sb, "set -gx GOVM_VERSION '%s'\n", version)
	fmt.Fprintf(&sb, "echo 'govm: now using Go %s'\n", version)
	return sb.String()
}

func scriptPowerShell(goroot, version string) string {
	goroot = filepath.Clean(goroot)
	var sb strings.Builder
	fmt.Fprintf(&sb, "$env:GOROOT = '%s'\n", goroot)
	fmt.Fprintf(&sb, "$env:Path = '%s\\bin;' + $env:Path\n", goroot)
	fmt.Fprintf(&sb, "$env:GOVM_VERSION = '%s'\n", version)
	fmt.Fprintf(&sb, "Write-Host 'govm: now using Go %s'\n", version)
	return sb.String()
}

func scriptCmd(goroot, version string) string {
	goroot = filepath.Clean(goroot)
	var sb strings.Builder
	sb.WriteString("@echo off\n")
	fmt.Fprintf(&sb, "set GOROOT=%s\n", goroot)
	fmt.Fprintf(&sb, "set PATH=%s\\bin;%%PATH%%\n", goroot)
	fmt.Fprintf(&sb, "set GOVM_VERSION=%s\n", version)
	fmt.Fprintf(&sb, "echo govm: now using Go %s\n", version)
	return sb.String()
}
