package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/abramsz/govm/pkg/shell"
	"github.com/abramsz/govm/pkg/store"
)

func init() {
	rootCmd.AddCommand(envCmd)
	envCmd.Flags().StringP("shell", "s", "", "target shell: bash, zsh, fish, powershell, cmd (auto-detect if empty)")
	envCmd.Flags().Bool("use-on-cd", false, "auto-switch version when cd into a directory with .go-version (not yet implemented)")
}

var envCmd = &cobra.Command{
	Use:   "env",
	Short: "Print shell initialization script (eval this in your shell config)",
	Long: `Print shell initialization code that sets up PATH and a govm wrapper function.

The generated script bakes in the full path to the govm binary, so it works
even when govm is not on PATH yet (important on Windows).

Add one of these to your shell config:

  Bash / Zsh:
    eval "$(govm env --shell bash)"

  Fish:
    govm env --shell fish | source

  PowerShell:
    govm env --shell powershell | Out-String | Invoke-Expression

  cmd.exe (add to AutoRun):
    for /f "tokens=*" %i in ('govm env --shell cmd') do call %%i

After setup, 'govm use <version>' works directly without manual eval.`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		shellFlag, _ := cmd.Flags().GetString("shell")

		var shellType shell.ShellType
		if shellFlag != "" {
			shellType = shell.ShellType(shellFlag)
		} else {
			shellType = shell.DetectShell()
		}

		govmHome := store.Home()

		// Get the current binary's path so the init script can hardcode it.
		govmExe, err := os.Executable()
		if err != nil {
			return fmt.Errorf("get executable path: %w", err)
		}
		// Clean resolves . and .. in the path.
		govmExe = filepath.Clean(govmExe)

		script := shell.EnvScript(shellType, govmHome, govmExe)
		fmt.Print(script)
		return nil
	},
}
