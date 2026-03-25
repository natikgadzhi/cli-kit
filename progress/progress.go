// Package progress provides progress indicators for CLI tools.
//
// Progress indicators write to stderr and only display in table output mode.
// In JSON mode, they are completely suppressed to avoid polluting output.
package progress

import (
	"fmt"
	"os"
	"sync"
)

// Indicator is the interface for all progress indicators.
type Indicator interface {
	Update(current int)
	Finish()
}

// Counter shows a running count of items processed.
// Example: "Fetching messages... 47"
type Counter struct {
	label    string
	active   bool
	mu       sync.Mutex
}

// NewCounter creates a new counter progress indicator.
// If format is "json", returns a no-op indicator that produces no output.
func NewCounter(label string, format string) Indicator {
	if format == "json" {
		return &noop{}
	}
	return &Counter{label: label, active: true}
}

// Update displays the current count. Overwrites the previous line on stderr.
func (c *Counter) Update(current int) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if !c.active {
		return
	}
	fmt.Fprintf(os.Stderr, "\r%s... %d", c.label, current)
}

// Finish clears the progress line from stderr.
func (c *Counter) Finish() {
	c.mu.Lock()
	defer c.mu.Unlock()
	if !c.active {
		return
	}
	c.active = false
	// Clear the line
	fmt.Fprintf(os.Stderr, "\r\033[K")
}

// Spinner shows a spinning indicator for operations with unknown total.
type Spinner struct {
	label  string
	active bool
	frame  int
	mu     sync.Mutex
}

var spinnerFrames = []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}

// NewSpinner creates a new spinner progress indicator.
// If format is "json", returns a no-op indicator.
func NewSpinner(label string, format string) Indicator {
	if format == "json" {
		return &noop{}
	}
	return &Spinner{label: label, active: true}
}

// Update advances the spinner by one frame. The current parameter is ignored.
func (s *Spinner) Update(_ int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.active {
		return
	}
	frame := spinnerFrames[s.frame%len(spinnerFrames)]
	fmt.Fprintf(os.Stderr, "\r%s %s", frame, s.label)
	s.frame++
}

// Finish clears the spinner line from stderr.
func (s *Spinner) Finish() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.active {
		return
	}
	s.active = false
	fmt.Fprintf(os.Stderr, "\r\033[K")
}

// noop is a silent progress indicator used in JSON mode.
type noop struct{}

func (n *noop) Update(_ int) {}
func (n *noop) Finish()      {}
