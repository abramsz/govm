package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/abramsz/govm/pkg/config"
	"github.com/abramsz/govm/pkg/store"
)

func init() {
	rootCmd.AddCommand(configCmd)
	configCmd.AddCommand(configSetCmd)
	configCmd.AddCommand(configAliasCmd)
}

// resolveVersion resolves aliases in version strings.
// If the input is a defined alias, returns the aliased version.
// Otherwise returns the input unchanged.
func resolveVersion(version string) string {
	cfg := config.Load(store.Home())
	return cfg.ResolveAlias(version)
}

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "View or modify the govm configuration",
	Long: `View or modify the global govm configuration file (~/.govm/config.json).

With no arguments, prints the current configuration as JSON.

Use subcommands to modify specific settings:
  govm config set <key> <value>
  govm config alias <name> <version>

Example:
  govm config
  govm config set mirror https://go-mirror.example.com/dl/
  govm config alias stable 1.26.4
  govm config alias latest 1.26.4`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg := config.Load(store.Home())
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(cfg)
	},
}

var configSetCmd = &cobra.Command{
	Use:   "set <key> <value>",
	Short: "Set a config value",
	Long: `Set a single configuration value.

Supported keys:
  mirror   - download mirror URL (e.g. https://go-mirror.example.com/dl/)
  default  - default Go version`,
	Args: cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		key := args[0]
		value := args[1]

		govmHome := store.Home()
		cfg := config.Load(govmHome)

		switch strings.ToLower(key) {
		case "mirror":
			cfg.Mirror = value
		case "default":
			cfg.DefaultVersion = value
		default:
			return fmt.Errorf("unknown config key %q; supported: mirror, default", key)
		}

		if err := config.Save(govmHome, cfg); err != nil {
			return err
		}
		fmt.Printf("config: %s set to %q\n", key, value)
		return nil
	},
}

var configAliasCmd = &cobra.Command{
	Use:   "alias <name> <version>",
	Short: "Set or remove a version alias",
	Long: `Set or remove a version alias.

Set an alias:
  govm config alias stable 1.26.4
  govm config alias latest 1.26.4

Remove an alias:
  govm config alias stable ""

After setting an alias, you can use it anywhere a version is expected:
  govm install stable
  govm use stable
  govm default latest`,
	Args: cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		version := args[1]

		govmHome := store.Home()
		cfg := config.Load(govmHome)

		if cfg.Aliases == nil {
			cfg.Aliases = make(map[string]string)
		}

		if version == "" {
			delete(cfg.Aliases, name)
			fmt.Printf("config: alias %q removed\n", name)
		} else {
			cfg.Aliases[name] = version
			fmt.Printf("config: alias %q → %s\n", name, version)
		}

		return config.Save(govmHome, cfg)
	},
}
