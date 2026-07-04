package cmd

import (
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/spf13/cobra"

	"github.com/abramsz/govm/pkg/versions"
)

func init() {
	rootCmd.AddCommand(listRemoteCmd)

	listRemoteCmd.Flags().BoolP("all", "a", false, "include unstable and release-candidate versions")
}

var listRemoteCmd = &cobra.Command{
	Use:   "list-remote",
	Short: "List all available Go versions from go.dev",
	Long: `Fetch and display every Go version available for download from go.dev.

By default only stable releases are shown. Use --all to include
unstable and release-candidate versions.`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		showAll, _ := cmd.Flags().GetBool("all")

		var releases []versions.Release
		var err error
		if showAll {
			releases, err = versions.FetchAll(cmd.Context())
		} else {
			releases, err = versions.FetchStable(cmd.Context())
		}
		if err != nil {
			return err
		}

		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		for _, r := range releases {
			fmt.Fprintf(w, "%s\tstable=%v\n", versions.ShortVersion(r.Version), r.Stable)
		}
		w.Flush()
		return nil
	},
}
