package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/abramsz/govm/pkg/shell"
	"github.com/abramsz/govm/pkg/store"
)

func init() {
	rootCmd.AddCommand(useCmd)
	useCmd.Flags().StringP("shell", "s", "", "shell syntax: bash, zsh, fish, powershell, cmd (auto-detect if empty)")
}

var useCmd = &cobra.Command{
	Use:   "use <version>",
	Short: "Switch Go version in the current shell",
	Long: `Activate the specified Go version for the current shell session.

Outputs a shell-syntax activation script. Pipe through eval or Invoke-Expression:

  Bash / Zsh:   eval "$(govm use 1.23.4)"
  Fish:         govm use 1.23.4 | source
  PowerShell:   govm use 1.23.4 | Out-String | Invoke-Expression
  cmd.exe:      for /f "tokens=*" %i in ('govm use 1.23.4') do call %i

Use --shell to override auto-detection (the wrapper from 'govm env' passes this automatically).`,
	Args:              cobra.ExactArgs(1),
	ValidArgsFunction: completeInstalledVersions,
	RunE: func(cmd *cobra.Command, args []string) error {
		version := resolveVersion(args[0])

		if !store.IsInstalled(version) {
			return fmt.Errorf("version %s is not installed — run `govm install %s` first", version, version)
		}

		// Determine target shell.
		shellFlag, _ := cmd.Flags().GetString("shell")
		var shellType shell.ShellType
		if shellFlag != "" {
			shellType = shell.ShellType(shellFlag)
		} else {
			shellType = shell.DetectShell()
		}

		goroot := store.Goroot(version)
		script := shell.Script(shellType, goroot, version)

		// If stdout is a terminal, show usage hints.
		if isTerminal() {
			fmt.Fprintln(os.Stderr, "=== govm use", version, "===")
			fmt.Fprintln(os.Stderr)
			fmt.Fprintf(os.Stderr, "To activate Go %s, run one of:\n", version)
			fmt.Fprintln(os.Stderr)
			fmt.Fprintln(os.Stderr, "  # Temporary (current shell only):")
			switch shellType {
			case shell.ShellPowerShell:
				fmt.Fprintln(os.Stderr, "  govm use", version, "| Out-String | Invoke-Expression")
			case shell.ShellCmd:
				fmt.Fprintf(os.Stderr, `  for /f "tokens=*" %%%%i in ('govm use %s') do call %%%%i`, version)
				fmt.Fprintln(os.Stderr)
			default:
				fmt.Fprintln(os.Stderr, "  eval \"$(govm use", version, ")\"")
			}
			fmt.Fprintln(os.Stderr)
			fmt.Fprintln(os.Stderr, "  # Permanent (add to shell config):")
			fmt.Fprintln(os.Stderr, "  eval \"$(govm env)\"")
			fmt.Fprintln(os.Stderr, "  govm default", version)
			fmt.Fprintln(os.Stderr)
		}

		fmt.Print(script)
		return nil
	},
}
