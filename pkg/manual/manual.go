// Package manual generates a comprehensive Markdown user manual for govm,
// designed for consumption by AI language models and human readers alike.
//
// The manual content is maintained as a Markdown file (manual.md) and embedded
// at compile time using Go's //go:embed directive.
package manual

import (
	_ "embed"
	"strings"
)

//go:embed manual.md
var manualContent string

// Generate returns the user manual with {{VERSION}} replaced by the given
// version string.
func Generate(govmVersion string) string {
	return strings.Replace(manualContent, "{{VERSION}}", govmVersion, 1)
}
