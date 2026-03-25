// Package version provides a standard version command and --version flag for CLI tools.
//
// Tools set Version, Commit, and Date via ldflags at build time.
// The version output is always JSON, regardless of the output format flag.
package version

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

// Info holds version information. Fields are set via ldflags at build time.
type Info struct {
	Version string `json:"version"`
	Commit  string `json:"commit"`
	Date    string `json:"date"`
}

// NewCommand creates a "version" subcommand that prints version info as JSON.
func NewCommand(info *Info) *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print version information",
		RunE: func(cmd *cobra.Command, args []string) error {
			return printJSON(info)
		},
	}
}

// SetupFlag configures the root command's --version flag to print version info.
// This makes `tool --version` work in addition to `tool version`.
func SetupFlag(rootCmd *cobra.Command, info *Info) {
	rootCmd.Version = info.Version
	rootCmd.SetVersionTemplate(versionTemplate(info))
}

func printJSON(info *Info) error {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(info)
}

func versionTemplate(info *Info) string {
	b, _ := json.MarshalIndent(info, "", "  ")
	return fmt.Sprintf("%s\n", string(b))
}
