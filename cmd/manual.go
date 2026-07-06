package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/abramsz/govm/pkg/manual"
)

func init() {
	rootCmd.AddCommand(manualCmd)
}

var manualCmd = &cobra.Command{
	Use:   "manual",
	Short: "Print a comprehensive Markdown user manual (designed for AI / LLM consumption)",
	Long: `Print the full user manual in Markdown format.

This manual is designed for both human readers and large language models (LLMs)
to understand govm's architecture, commands, shell integration, and internals.

Pipe to a file:
  govm manual > docs/govm-manual.md

Or read directly in a terminal with a Markdown renderer.`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		version := rootCmd.Version
		if version == "" {
			version = "0.1.1-dev"
		}
		content := manual.Generate(version)
		fmt.Print(content)
		return nil
	},
}
