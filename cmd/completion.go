package cmd

import (
	"context"

	"github.com/spf13/cobra"

	"github.com/abramsz/govm/pkg/store"
	"github.com/abramsz/govm/pkg/versions"
)

// completeInstalledVersions provides shell completion for installed Go versions.
func completeInstalledVersions(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	if len(args) > 0 {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
	installed, err := store.ListInstalled()
	if err != nil {
		return nil, cobra.ShellCompDirectiveError
	}
	return installed, cobra.ShellCompDirectiveNoFileComp
}

// completeRemoteVersions provides shell completion for remote Go versions.
func completeRemoteVersions(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	if len(args) > 0 {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
	// Use the command's context for cancellation; fall back to Background.
	ctx := cmd.Context()
	if ctx == nil {
		ctx = context.Background()
	}
	releases, err := versions.FetchStable(ctx)
	if err != nil {
		return nil, cobra.ShellCompDirectiveError
	}
	names := make([]string, 0, len(releases))
	for _, r := range releases {
		names = append(names, versions.ShortVersion(r.Version))
	}
	return names, cobra.ShellCompDirectiveNoFileComp
}
