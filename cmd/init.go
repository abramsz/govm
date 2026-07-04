package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/spf13/cobra"

	"github.com/abramsz/govm/pkg/shell"
)

func init() {
	rootCmd.AddCommand(initCmd)
	initCmd.Flags().StringP("shell", "s", "", "target shell: bash, zsh, fish, powershell (auto-detect if empty)")
}

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Add govm to your shell profile (one-time setup)",
	Long: `Add govm to your shell profile (one-time setup).

Detects the current shell and appends the appropriate init command
to the correct profile file. Uses the full binary path so it works
even if govm is not on PATH. Skips if already present.

Supported shells and profiles:
  PowerShell  ~/.config/powershell/Microsoft.PowerShell_profile.ps1 (Unix)
              ~/Documents/PowerShell/Microsoft.PowerShell_profile.ps1 (Windows)
  bash        ~/.bashrc
  zsh         ~/.zshrc
  fish        ~/.config/fish/conf.d/govm.fish

Examples:
  govm init
  govm init --shell bash`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		shellFlag, _ := cmd.Flags().GetString("shell")
		var shellType shell.ShellType
		if shellFlag != "" {
			shellType = shell.ShellType(shellFlag)
		} else {
			shellType = shell.DetectShell()
		}
		line, profile, err := initConfig(shellType)
		if err != nil {
			return err
		}

		// Check if govm init line already exists in profile.
		data, _ := os.ReadFile(profile)
		if strings.Contains(string(data), "govm env --shell") {
			fmt.Printf("govm: already initialized in %s\n", profile)
			return nil
		}

		f, err := os.OpenFile(profile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
		if err != nil {
			return fmt.Errorf("open %s: %w", profile, err)
		}
		defer f.Close()

		if _, err := fmt.Fprintln(f, "\n"+line); err != nil {
			return fmt.Errorf("write %s: %w", profile, err)
		}

		fmt.Printf("govm: added to %s\n", profile)
		fmt.Println("Restart your shell or source the profile to activate.")
		return nil
	},
}

// initConfig returns the init line and profile path for the given shell.
func initConfig(shellType shell.ShellType) (line, profile string, err error) {
	switch shellType {
	case shell.ShellPowerShell:
		profile = psProfilePath()
		line = "govm env --shell powershell | Out-String | Invoke-Expression"

	case shell.ShellBash, shell.ShellZsh:
		profile = filepath.Join(os.Getenv("HOME"), "."+string(shellType)+"rc")
		line = fmt.Sprintf(`eval "$(govm env --shell %s)"`, shellType)

	case shell.ShellFish:
		profile = filepath.Join(os.Getenv("HOME"), ".config", "fish", "conf.d", "govm.fish")
		line = "govm env --shell fish | source"

	default:
		return "", "", fmt.Errorf("unsupported shell: %s", shellType)
	}

	// Ensure parent directory exists for fish.
	dir := filepath.Dir(profile)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", "", fmt.Errorf("create %s: %w", dir, err)
	}

	return line, profile, nil
}

// psProfilePath returns the PowerShell profile path.
// On Windows, always uses USERPROFILE\Documents\PowerShell\.
// On Unix, uses ~/.config/powershell/.
func psProfilePath() string {
	// Windows: $env:USERPROFILE\Documents\PowerShell\Microsoft.PowerShell_profile.ps1
	if runtime.GOOS == "windows" {
		home := os.Getenv("USERPROFILE")
		if home == "" {
			home = os.Getenv("HOME")
		}
		return filepath.Join(home, "Documents", "PowerShell", "Microsoft.PowerShell_profile.ps1")
	}
	// Unix: ~/.config/powershell/Microsoft.PowerShell_profile.ps1
	home := os.Getenv("HOME")
	return filepath.Join(home, ".config", "powershell", "Microsoft.PowerShell_profile.ps1")
}
