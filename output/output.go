// Package output provides consistent output formatting for CLI tools.
//
// It supports two formats: "json" (structured JSON) and "table" (human-readable aligned columns).
// By default, it detects whether stdout is a TTY and chooses accordingly:
// TTY → table, piped/redirected → json.
//
// Usage:
//
//	output.RegisterFlag(rootCmd)
//	format := output.Resolve(cmd)
//	output.Print(format, data)
package output

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/spf13/cobra"
	"golang.org/x/term"
)

const (
	FormatJSON  = "json"
	FormatTable = "table"
)

// RegisterFlag adds the -o/--output flag to a Cobra command's persistent flags.
func RegisterFlag(cmd *cobra.Command) {
	cmd.PersistentFlags().StringP("output", "o", "", "Output format: json, table (default: auto-detected based on TTY)")
}

// Resolve returns the output format for the given command.
// If the user explicitly set -o, that value is used.
// Otherwise, it detects TTY on stdout: TTY → table, non-TTY → json.
func Resolve(cmd *cobra.Command) string {
	f := cmd.Flags().Lookup("output")
	if f == nil {
		f = cmd.PersistentFlags().Lookup("output")
	}
	if f != nil && f.Changed {
		val := strings.ToLower(f.Value.String())
		if val == FormatJSON || val == FormatTable {
			return val
		}
		// Invalid value — fall through to default
		fmt.Fprintf(os.Stderr, "Warning: unknown output format %q, using auto-detection\n", val)
	}
	return detectDefault()
}

// IsTable returns true if the format is table.
func IsTable(format string) bool {
	return format == FormatTable
}

// IsJSON returns true if the format is json.
func IsJSON(format string) bool {
	return format == FormatJSON
}

func detectDefault() string {
	if term.IsTerminal(int(os.Stdout.Fd())) {
		return FormatTable
	}
	return FormatJSON
}

// PrintJSON writes data as pretty-printed JSON to stdout.
func PrintJSON(data any) error {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(data)
}

// Table provides a simple tabwriter-based table output.
type Table struct {
	w *tabwriter.Writer
}

// NewTable creates a new table writer that writes to stdout.
func NewTable() *Table {
	return NewTableWriter(os.Stdout)
}

// NewTableWriter creates a new table writer with a custom destination.
func NewTableWriter(out io.Writer) *Table {
	return &Table{
		w: tabwriter.NewWriter(out, 0, 4, 2, ' ', 0),
	}
}

// Header writes a header row. Values are tab-separated and uppercased.
func (t *Table) Header(columns ...string) {
	upper := make([]string, len(columns))
	for i, c := range columns {
		upper[i] = strings.ToUpper(c)
	}
	fmt.Fprintln(t.w, strings.Join(upper, "\t"))
}

// Row writes a data row. Values are tab-separated.
func (t *Table) Row(values ...string) {
	fmt.Fprintln(t.w, strings.Join(values, "\t"))
}

// Flush flushes the underlying tabwriter.
func (t *Table) Flush() error {
	return t.w.Flush()
}

// Print writes data in the specified format. For JSON it marshals to stdout.
// For table, the caller provides a TableRenderer that knows how to render the data.
func Print(format string, data any, renderer TableRenderer) error {
	if format == FormatJSON {
		return PrintJSON(data)
	}
	if renderer == nil {
		// Fallback to JSON if no table renderer provided
		return PrintJSON(data)
	}
	t := NewTable()
	renderer.RenderTable(t)
	return t.Flush()
}

// TableRenderer is implemented by types that can render themselves as a table.
type TableRenderer interface {
	RenderTable(t *Table)
}
