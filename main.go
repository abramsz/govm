package main

import "github.com/abramsz/govm/cmd"

// version is set at build time via -ldflags, e.g.:
//
//	go build -ldflags="-X 'main.version=v1.0.0'" -o govm .
//
// Defaults to "0.1.0-dev" for development builds.
var version = "0.1.0-dev"

func init() {
	cmd.Version = version
}

func main() {
	cmd.Execute()
}
