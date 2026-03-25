package derived

import (
	"os"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

func TestDefaultPath(t *testing.T) {
	path := DefaultPath("fm")
	if !strings.HasSuffix(path, ".local/share/lambdal/derived/fm") {
		t.Errorf("DefaultPath = %q, expected to end with .local/share/lambdal/derived/fm", path)
	}
}

func TestToolEnvVar(t *testing.T) {
	tests := []struct {
		tool string
		want string
	}{
		{"fm", "FM_DERIVED_DIR"},
		{"slack-cli", "SLACK_CLI_DERIVED_DIR"},
		{"gdrive-cli", "GDRIVE_CLI_DERIVED_DIR"},
	}
	for _, tt := range tests {
		got := toolEnvVar(tt.tool)
		if got != tt.want {
			t.Errorf("toolEnvVar(%q) = %q, want %q", tt.tool, got, tt.want)
		}
	}
}

func TestResolveFlag(t *testing.T) {
	cmd := &cobra.Command{}
	RegisterFlag(cmd, "fm")
	cmd.PersistentFlags().Set("derived", "/custom/path")

	got := Resolve(cmd, "fm")
	if got != "/custom/path" {
		t.Errorf("Resolve with flag = %q, want /custom/path", got)
	}
}

func TestResolveToolEnv(t *testing.T) {
	os.Setenv("FM_DERIVED_DIR", "/env/fm/path")
	defer os.Unsetenv("FM_DERIVED_DIR")

	cmd := &cobra.Command{}
	RegisterFlag(cmd, "fm")

	got := Resolve(cmd, "fm")
	if got != "/env/fm/path" {
		t.Errorf("Resolve with tool env = %q, want /env/fm/path", got)
	}
}

func TestResolveBaseEnv(t *testing.T) {
	os.Setenv("LAMBDAL_DERIVED_DIR", "/env/base")
	defer os.Unsetenv("LAMBDAL_DERIVED_DIR")

	cmd := &cobra.Command{}
	RegisterFlag(cmd, "fm")

	got := Resolve(cmd, "fm")
	if got != "/env/base/fm" {
		t.Errorf("Resolve with base env = %q, want /env/base/fm", got)
	}
}

func TestNewFrontmatter(t *testing.T) {
	fm := NewFrontmatter("fm", "email", "M123", "https://app.fastmail.com/M123", "fm fetch M123")
	if fm.Tool != "fm" {
		t.Errorf("Tool = %q", fm.Tool)
	}
	if fm.CreatedAt == "" {
		t.Error("CreatedAt should be set")
	}
	if fm.CreatedAt != fm.UpdatedAt {
		t.Error("CreatedAt and UpdatedAt should match on new frontmatter")
	}
}

func TestFormatFile(t *testing.T) {
	fm := Frontmatter{
		Tool:       "fm",
		ObjectType: "email",
		Slug:       "M123",
		SourceURL:  "https://example.com",
		CreatedAt:  "2026-03-15T10:00:00Z",
		UpdatedAt:  "2026-03-15T10:00:00Z",
		Command:    "fm fetch M123",
	}
	got := FormatFile(fm, "# Hello\n\nBody here.")
	if !strings.Contains(got, "---") {
		t.Error("expected YAML frontmatter delimiters")
	}
	if !strings.Contains(got, "tool: fm") {
		t.Error("expected tool field")
	}
	if !strings.Contains(got, "# Hello") {
		t.Error("expected body content")
	}
}
