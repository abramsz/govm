package cmd

import (
	"github.com/spf13/cobra"

	"github.com/abramsz/govm/pkg/store"
)

func init() {
	rootCmd.AddCommand(uninstallCmd)
}

var uninstallCmd = &cobra.Command{
	Use:   "uninstall <version>",
	Short: "Remove an installed Go version",
	Long: `Uninstall the specified Go version from the local store.

Example:
  govm uninstall 1.22.10`,
	Args: cobra.ExactArgs(1),
	ValidArgsFunction: completeInstalledVersions,
	RunE: func(cmd *cobra.Command, args []string) error {
		return store.Uninstall(resolveVersion(args[0]))
	},
}
