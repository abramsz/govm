package cmd

import (
	"fmt"
	"runtime"

	"github.com/spf13/cobra"

	"github.com/abramsz/govm/pkg/download"
	"github.com/abramsz/govm/pkg/store"
	"github.com/abramsz/govm/pkg/versions"
)

// archOrRuntime returns arch if non-empty, else runtime.GOARCH.
func archOrRuntime(arch string) string {
	if arch != "" {
		return arch
	}
	return runtime.GOARCH
}

func init() {
	rootCmd.AddCommand(installCmd)
	installCmd.Flags().Bool("reinstall", false, "reinstall even if already installed")
	installCmd.Flags().StringP("arch", "a", "", "target architecture: amd64, arm64, 386, etc. (default: host arch)")
}

var installCmd = &cobra.Command{
	Use:   "install <version>",
	Short: "Download and install a Go version",
	Long: `Download and install the specified Go version from go.dev.
If no other version is set as default, the first install is
automatically set as the global default.

Examples:
  govm install 1.23.4
  govm install 1.22.10
  govm install --reinstall 1.23.4
  govm install 1.23.4 --arch arm64`,
	Args:              cobra.ExactArgs(1),
	ValidArgsFunction: completeRemoteVersions,
	RunE: func(cmd *cobra.Command, args []string) error {
		wantVersion := resolveVersion(args[0])
		reinstall, _ := cmd.Flags().GetBool("reinstall")

		// Already installed? Skip unless --reinstall.
		if store.IsInstalled(wantVersion) && !reinstall {
			cmd.Printf("Go %s is already installed.\n", wantVersion)
			return nil
		}

		// Find the release in the remote list.
		all, err := versions.FetchAll(cmd.Context())
		if err != nil {
			return err
		}

		var found *versions.Release
		for _, r := range all {
			if versions.ShortVersion(r.Version) == wantVersion {
				found = &r
				break
			}
		}
		if found == nil {
			cmd.Println("Version not found. Run `govm list-remote` to see available versions.")
			cmd.SilenceUsage = true
			return nil
		}

		archOverride, _ := cmd.Flags().GetString("arch")

		// Check whether the release has an archive for the target platform.
		if download.FindFile(*found, archOverride) == nil {
			return fmt.Errorf("go %s has no archive for %s/%s — try a different version or platform", wantVersion, runtime.GOOS, archOrRuntime(archOverride))
		}

		if _, err := store.Install(cmd.Context(), *found, reinstall, archOverride); err != nil {
			return err
		}

		// Auto-set as default if nothing is set yet.
		if _, err := store.Current(); err != nil {
			_ = store.SetDefault(wantVersion)
		}

		return nil
	},
}
