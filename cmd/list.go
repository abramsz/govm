package cmd

import (
	"github.com/spf13/cobra"

	"github.com/abramsz/govm/pkg/store"
)

func init() {
	rootCmd.AddCommand(listCmd)
}

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List locally installed Go versions",
	Long:  `List all Go versions that have been installed via govm.`,
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		installed, err := store.ListInstalled()
		if err != nil {
			return err
		}

		if len(installed) == 0 {
			cmd.Println("No versions installed. Run `govm list-remote` to see what's available.")
			return nil
		}

		current, _ := store.Current()

		for _, v := range installed {
			marker := " "
			if v == current {
				marker = "*"
			}
			cmd.Printf(" %s %s\n", marker, v)
		}
		return nil
	},
}
