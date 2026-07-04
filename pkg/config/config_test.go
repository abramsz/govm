package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDefault(t *testing.T) {
	cfg := Default()
	if cfg.Aliases == nil {
		t.Error("Default().Aliases should be non-nil")
	}
	if cfg.Mirror != "" {
		t.Errorf("Default().Mirror = %q; want empty", cfg.Mirror)
	}
}

func TestSaveAndLoad(t *testing.T) {
	dir := t.TempDir()

	cfg := &Config{
		DefaultVersion: "1.23.4",
		Aliases:        map[string]string{"stable": "1.26.4"},
		Mirror:         "https://mirror.example.com",
	}

	if err := Save(dir, cfg); err != nil {
		t.Fatalf("Save() failed: %v", err)
	}

	loaded := Load(dir)
	if loaded.DefaultVersion != "1.23.4" {
		t.Errorf("DefaultVersion = %q; want 1.23.4", loaded.DefaultVersion)
	}
	if loaded.Aliases["stable"] != "1.26.4" {
		t.Errorf("Aliases[stable] = %q; want 1.26.4", loaded.Aliases["stable"])
	}
	if loaded.Mirror != "https://mirror.example.com" {
		t.Errorf("Mirror = %q; want https://mirror.example.com", loaded.Mirror)
	}
}

func TestLoadMissingFile(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "nonexistent")
	cfg := Load(dir)
	// Should return defaults.
	if cfg.DefaultVersion != "" {
		t.Errorf("expected empty DefaultVersion, got %q", cfg.DefaultVersion)
	}
}

func TestLoadCorruptedFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, FileName)
	if err := os.WriteFile(path, []byte("{invalid json}"), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg := Load(dir)
	// Should return defaults without error.
	if cfg.Aliases == nil {
		t.Error("expected non-nil Aliases map from defaults")
	}
}

func TestResolveAlias(t *testing.T) {
	cfg := &Config{
		Aliases: map[string]string{
			"stable": "1.26.4",
			"latest": "stable",
		},
	}

	tests := []struct {
		input    string
		expected string
	}{
		{"stable", "1.26.4"},
		{"1.23.4", "1.23.4"},     // no alias → unchanged
		{"latest", "1.26.4"},     // recursive alias
		{"unknown", "unknown"},   // not an alias → unchanged
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := cfg.ResolveAlias(tt.input)
			if got != tt.expected {
				t.Errorf("ResolveAlias(%q) = %q; want %q", tt.input, got, tt.expected)
			}
		})
	}
}

func TestResolveAliasCycleProtection(t *testing.T) {
	cfg := &Config{
		Aliases: map[string]string{
			"a": "b",
			"b": "c",
			"c": "a", // cycle
		},
	}

	got := cfg.ResolveAlias("a")
	// The cycle is broken after 10 iterations; the return value is whatever
	// the last alias resolved to before hitting the limit.
	if got != "b" {
		t.Errorf("expected cycle to halt at 'b', got %q", got)
	}
}

func TestSaveCreatesDirectory(t *testing.T) {
	// Save should create the govm home directory if it doesn't exist.
	dir := filepath.Join(t.TempDir(), "deep", "nested", "path")
	cfg := Default()
	cfg.DefaultVersion = "1.23.4"

	if err := Save(dir, cfg); err != nil {
		t.Fatalf("Save() with non-existent dir failed: %v", err)
	}

	if _, err := os.Stat(filepath.Join(dir, FileName)); err != nil {
		t.Errorf("config file was not created: %v", err)
	}
}
