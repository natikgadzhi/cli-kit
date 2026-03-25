// Package derived provides consistent derived data directory management for CLI tools.
//
// All tools store cached/derived data as Markdown files with YAML frontmatter
// in a shared directory structure: ~/.local/share/lambdal/derived/{tool-name}/
package derived

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/spf13/cobra"
)

const (
	defaultBase = ".local/share/lambdal/derived"
	envBase     = "LAMBDAL_DERIVED_DIR"
)

// RegisterFlag adds the -d/--derived flag to a Cobra command's persistent flags.
func RegisterFlag(cmd *cobra.Command, toolName string) {
	defaultPath := DefaultPath(toolName)
	cmd.PersistentFlags().StringP("derived", "d", defaultPath,
		fmt.Sprintf("Derived data directory (default: %s)", defaultPath))
}

// Resolve returns the derived directory path for the given command and tool.
// Priority: -d flag > {TOOL}_DERIVED_DIR env > LAMBDAL_DERIVED_DIR env > default.
func Resolve(cmd *cobra.Command, toolName string) string {
	f := cmd.Flags().Lookup("derived")
	if f == nil {
		f = cmd.PersistentFlags().Lookup("derived")
	}
	if f != nil && f.Changed {
		return f.Value.String()
	}

	toolEnvKey := toolEnvVar(toolName)
	if v := os.Getenv(toolEnvKey); v != "" {
		return v
	}

	if v := os.Getenv(envBase); v != "" {
		return filepath.Join(v, toolName)
	}

	return DefaultPath(toolName)
}

// DefaultPath returns the default derived directory for a tool.
func DefaultPath(toolName string) string {
	home, err := os.UserHomeDir()
	if err != nil {
		home = "~"
	}
	return filepath.Join(home, defaultBase, toolName)
}

// EnsureDir creates the derived directory if it doesn't exist.
func EnsureDir(path string) error {
	return os.MkdirAll(path, 0755)
}

func toolEnvVar(toolName string) string {
	// Convert tool name to uppercase, replace hyphens with underscores
	upper := ""
	for _, c := range toolName {
		if c == '-' {
			upper += "_"
		} else if c >= 'a' && c <= 'z' {
			upper += string(c - 32)
		} else {
			upper += string(c)
		}
	}
	return upper + "_DERIVED_DIR"
}

// Frontmatter represents the YAML frontmatter for a derived Markdown file.
type Frontmatter struct {
	Tool       string `yaml:"tool"`
	ObjectType string `yaml:"object_type"`
	Slug       string `yaml:"slug"`
	SourceURL  string `yaml:"source_url,omitempty"`
	CreatedAt  string `yaml:"created_at"`
	UpdatedAt  string `yaml:"updated_at"`
	Command    string `yaml:"command"`
}

// NewFrontmatter creates a Frontmatter with timestamps set to now.
func NewFrontmatter(tool, objectType, slug, sourceURL, command string) Frontmatter {
	now := time.Now().UTC().Format(time.RFC3339)
	return Frontmatter{
		Tool:       tool,
		ObjectType: objectType,
		Slug:       slug,
		SourceURL:  sourceURL,
		CreatedAt:  now,
		UpdatedAt:  now,
		Command:    command,
	}
}

// FormatFile produces a complete Markdown file with YAML frontmatter and body.
func FormatFile(fm Frontmatter, body string) string {
	return fmt.Sprintf(`---
tool: %s
object_type: %s
slug: %s
source_url: %s
created_at: %s
updated_at: %s
command: %q
---

%s
`, fm.Tool, fm.ObjectType, fm.Slug, fm.SourceURL, fm.CreatedAt, fm.UpdatedAt, fm.Command, body)
}
