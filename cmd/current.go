package cmd

import (
	"github.com/spf13/cobra"

	"github.com/abramsz/govm/pkg/store"
)

func init() {
	rootCmd.AddCommand(currentCmd)
}

var currentCmd = &cobra.Command{
	Use:   "current",
	Short: "Show the currently active Go version",
	Long:  `Print the version number of the Go installation currently in use.`,
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		v, err := store.Current()
		if err != nil {
			return err
		}
		cmd.Println(v)
		return nil
	},
}
