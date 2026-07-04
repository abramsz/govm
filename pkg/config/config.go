// Package config manages the govm configuration file (~/.govm/config.json).
//
// The config file stores persistent settings such as version aliases,
// the default Go version mirror, and future configuration options.
package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// FileName is the name of the config file inside GOVM_HOME.
const FileName = "config.json"

// Config holds the govm configuration.
type Config struct {
	// DefaultVersion is the global default Go version.
	// Deprecated: use symlink at GOVM_HOME/current instead; this field may
	// be removed in a future release.
	DefaultVersion string `json:"default_version,omitempty"`

	// Aliases maps human-friendly names to Go versions.
	// Example: "stable" → "1.26.4", "latest" → "1.26.4"
	Aliases map[string]string `json:"aliases,omitempty"`

	// Mirror overrides the default download base URL (go.dev/dl/).
	// Useful for air-gapped environments or regional mirrors.
	Mirror string `json:"mirror,omitempty"`
}

// Default returns a Config with sensible defaults.
func Default() *Config {
	return &Config{
		Aliases: make(map[string]string),
	}
}

// Load reads the config file from govmHome. Returns a default config
// if the file does not exist or cannot be parsed.
func Load(govmHome string) *Config {
	path := filepath.Join(govmHome, FileName)
	data, err := os.ReadFile(path)
	if err != nil {
		return Default()
	}

	cfg := Default()
	if err := json.Unmarshal(data, cfg); err != nil {
		fmt.Fprintf(os.Stderr, "govm: warning: config file corrupted, using defaults: %v\n", err)
		return Default()
	}
	return cfg
}

// Save writes the config to the govm home directory.
func Save(govmHome string, cfg *Config) error {
	path := filepath.Join(govmHome, FileName)
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal config: %w", err)
	}
	if err := os.MkdirAll(govmHome, 0o755); err != nil {
		return fmt.Errorf("create config directory: %w", err)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("write config: %w", err)
	}
	return nil
}

// ResolveAlias follows aliases recursively. Returns the original value
// if it's not an alias. Detects cycles (max depth 10).
func (c *Config) ResolveAlias(name string) string {
	seen := 0
	for seen < 10 {
		resolved, ok := c.Aliases[name]
		if !ok {
			return name
		}
		name = resolved
		seen++
	}
	return name
}
