package version

import (
	"testing"

	"github.com/spf13/cobra"
)

func TestNewCommand(t *testing.T) {
	info := &Info{Version: "1.0.0", Commit: "abc123", Date: "2026-01-01T00:00:00Z"}
	cmd := NewCommand(info)

	if cmd.Use != "version" {
		t.Errorf("Use = %q, want 'version'", cmd.Use)
	}
}

func TestSetupFlag(t *testing.T) {
	info := &Info{Version: "2.0.0", Commit: "def456", Date: "2026-06-01T00:00:00Z"}
	rootCmd := &cobra.Command{Use: "test"}
	SetupFlag(rootCmd, info)

	if rootCmd.Version != "2.0.0" {
		t.Errorf("Version = %q, want '2.0.0'", rootCmd.Version)
	}
}
