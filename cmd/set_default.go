package cmd

import (
	"github.com/spf13/cobra"

	"github.com/abramsz/govm/pkg/config"
	"github.com/abramsz/govm/pkg/store"
)

func init() {
	rootCmd.AddCommand(setDefaultCmd)
}

var setDefaultCmd = &cobra.Command{
	Use:   "default <version>",
	Short: "Set the global default Go version",
	Long: `Set the global default Go version. This updates the symlink at
~/.govm/current so every new shell picks up this version.

Example:
  govm default 1.23.4`,
	Args: cobra.ExactArgs(1),
	ValidArgsFunction: completeInstalledVersions,
	RunE: func(cmd *cobra.Command, args []string) error {
		version := resolveVersion(args[0])
		if err := store.SetDefault(version); err != nil {
			return err
		}
		// Also persist to config file.
		govmHome := store.Home()
		cfg := config.Load(govmHome)
		cfg.DefaultVersion = version
		_ = config.Save(govmHome, cfg) // best-effort

		cmd.Printf("Default Go version set to %s.\n", version)
		cmd.Println("Make sure ~/.govm/current/bin is in your PATH.")
		return nil
	},
}
