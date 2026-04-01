package debug

import (
	"fmt"
	"os"
	"sync"

	"github.com/spf13/cobra"
)

var (
	enabled bool
	mu      sync.RWMutex
)

// Enable turns on debug logging to stderr.
func Enable() {
	mu.Lock()
	defer mu.Unlock()
	enabled = true
}

// Enabled returns whether debug mode is active.
func Enabled() bool {
	mu.RLock()
	defer mu.RUnlock()
	return enabled
}

// Disable turns off debug logging. Useful for resetting state in tests.
func Disable() {
	mu.Lock()
	defer mu.Unlock()
	enabled = false
}

// Log prints a debug message to stderr if debug mode is enabled.
// Format: "Debug: <message>\n"
func Log(format string, args ...any) {
	if !Enabled() {
		return
	}
	msg := fmt.Sprintf(format, args...)
	fmt.Fprintf(os.Stderr, "Debug: %s\n", msg)
}

// RegisterFlag adds --debug as a persistent flag on the given command
// and wires it to Enable() via PersistentPreRunE.
func RegisterFlag(cmd *cobra.Command) {
	cmd.PersistentFlags().Bool("debug", false, "Enable debug logging to stderr")

	// Chain with any existing PersistentPreRunE.
	existing := cmd.PersistentPreRunE
	cmd.PersistentPreRunE = func(cmd *cobra.Command, args []string) error {
		if d, _ := cmd.Flags().GetBool("debug"); d {
			Enable()
		}
		if existing != nil {
			return existing(cmd, args)
		}
		return nil
	}
}
