package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/abramsz/govm/pkg/config"
	"github.com/abramsz/govm/pkg/download"
	"github.com/abramsz/govm/pkg/store"
	"github.com/abramsz/govm/pkg/versions"
)

var rootCmd = &cobra.Command{
	Use:   "govm",
	Short: "Go Version Manager — install, switch, and manage Go versions",
	Long: `govm is a version manager for the Go programming language,
similar to nvm (Node) and fnm (Node). It lets you install multiple
Go versions side-by-side and switch between them per-shell or globally.`,
	SilenceUsage:  true,
	SilenceErrors: true,
}

// Execute runs the root command.
func Execute() {
	// Set up cache directory at runtime so GOVM_HOME changes take effect.
	versions.SetCacheDir(filepath.Join(store.Home(), "cache"))

	// Apply mirrors from config, falling back to default.
	cfg := config.Load(store.Home())
	mirrors := cfg.EffectiveMirrors()
	versions.SetBaseURLs(mirrors)
	download.SetBaseURLs(mirrors)

	rootCmd.Version = Version
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, "govm:", err)
		os.Exit(1)
	}
}
