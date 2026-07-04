package cmd

import (
	"github.com/spf13/cobra"

	"github.com/abramsz/govm/pkg/store"
)

func init() {
	rootCmd.AddCommand(whichCmd)
}

var whichCmd = &cobra.Command{
	Use:   "which",
	Short: "Show the path to the active Go binary",
	Long:  `Print the full path to the go binary that is currently active.`,
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		bin, err := store.Which()
		if err != nil {
			return err
		}
		cmd.Println(bin)
		return nil
	},
}
