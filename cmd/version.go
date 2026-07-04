package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

// Version is set by main.init() at startup.
// Override at build time via -ldflags, see main.go.
var Version = "0.0.0-dev"

func init() {
	rootCmd.AddCommand(versionCmd)
	rootCmd.Version = Version
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the govm version",
	Long:  `Print the version number of govm.`,
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Println(Version)
		return nil
	},
}
