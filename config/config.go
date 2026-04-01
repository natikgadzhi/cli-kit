// Package config provides TOML configuration file loading for CLI tools.
//
// It handles locating config files in ~/.config/{toolName}/config.toml,
// tilde expansion, and integrates with cobra for --config flags.
package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/BurntSushi/toml"
	"github.com/spf13/cobra"
)

// Load reads a TOML config file from path and unmarshals it into dst.
// dst must be a pointer to a struct with `toml` tags.
// Returns a wrapped os.ErrNotExist if the file doesn't exist, so callers
// can check with errors.Is(err, os.ErrNotExist).
func Load(path string, dst any) error {
	expanded, err := ExpandTilde(path)
	if err != nil {
		return fmt.Errorf("expanding config path: %w", err)
	}

	data, err := os.ReadFile(expanded)
	if err != nil {
		return fmt.Errorf("reading config file %s: %w", expanded, err)
	}

	if err := toml.Unmarshal(data, dst); err != nil {
		return fmt.Errorf("parsing config file %s: %w", expanded, err)
	}

	return nil
}

// DefaultPath returns ~/.config/{toolName}/config.toml with the home
// directory expanded. If the home directory cannot be determined, it
// returns the path with a literal ~ prefix.
func DefaultPath(toolName string) string {
	home, err := os.UserHomeDir()
	if err != nil {
		return filepath.Join("~", ".config", toolName, "config.toml")
	}
	return filepath.Join(home, ".config", toolName, "config.toml")
}

// RegisterFlag adds --config as a persistent flag on the given cobra command.
// The default value is DefaultPath(toolName).
func RegisterFlag(cmd *cobra.Command, toolName string) {
	cmd.PersistentFlags().String("config", DefaultPath(toolName), "path to config file")
}

// ExpandTilde replaces a leading "~" or "~/" with the user's home directory.
// Absolute and relative paths are returned unchanged.
func ExpandTilde(path string) (string, error) {
	if path == "~" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("getting home directory: %w", err)
		}
		return home, nil
	}
	if strings.HasPrefix(path, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("getting home directory: %w", err)
		}
		return filepath.Join(home, path[2:]), nil
	}
	return path, nil
}
