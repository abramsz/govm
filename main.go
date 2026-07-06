package main

import (
	"runtime/debug"

	"github.com/abramsz/govm/cmd"
)

// version is set at build time via -ldflags, e.g.:
//
//	go build -ldflags="-X 'main.version=v1.0.0'" -o govm .
//
// When installed via `go install @v0.1.0`, Go automatically sets the
// module version in debug.BuildInfo, which is used as a fallback.
// Defaults to "0.1.0-dev" for local development builds.
var version = "0.1.0-dev"

func init() {
	cmd.Version = resolveVersion()
}

func main() {
	cmd.Execute()
}

func resolveVersion() string {
	// If version was set via ldflags, use it.
	if version != "0.1.0-dev" {
		return version
	}
	// Try debug.BuildInfo (set by go install / go build -ldflags).
	if info, ok := debug.ReadBuildInfo(); ok &&
		info.Main.Version != "" &&
		info.Main.Version != "(devel)" {
		return info.Main.Version
	}
	return version
}
