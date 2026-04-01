// Package progress provides progress indicators for CLI tools.
//
// Progress indicators write to stderr and only display in table output mode.
// In JSON mode, they are completely suppressed to avoid polluting output.
package progress

import (
	"fmt"
	"os"
	"sync"
	"time"
)

// Indicator is the interface for all progress indicators.
type Indicator interface {
	// Start launches a background goroutine that auto-ticks the indicator.
	// For Spinner, this animates frames at ~100ms intervals.
	// Callers may use the old manual Update() pattern instead of Start().
	Start()

	// SetLabel changes the indicator's display label. Thread-safe.
	SetLabel(label string)

	// Update manually advances the indicator. For Counter, current is the count
	// to display. For Spinner, the parameter is ignored and the frame advances.
	Update(current int)

	// Finish stops any background goroutine and clears the progress line.
	// Safe to call multiple times.
	Finish()
}

// Counter shows a running count of items processed.
// Example: "Fetching messages... 47"
type Counter struct {
	label  string
	active bool
	mu     sync.Mutex
}

// NewCounter creates a new counter progress indicator.
// If format is "json", returns a no-op indicator that produces no output.
func NewCounter(label string, format string) Indicator {
	if format == "json" {
		return &noop{}
	}
	return &Counter{label: label, active: true}
}

// Start is a no-op for Counter. Counter requires explicit Update(n) calls
// to display the count.
func (c *Counter) Start() {}

// SetLabel changes the counter's display label. Thread-safe.
func (c *Counter) SetLabel(label string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.label = label
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
	label        string
	active       bool
	frame        int
	tickInterval time.Duration // interval between auto-tick frames; 0 means use DefaultTickInterval
	mu           sync.Mutex
	stopCh       chan struct{} // closed by Finish to stop the auto-tick goroutine
}

var spinnerFrames = []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}

// DefaultTickInterval is the interval between spinner frame advances.
var DefaultTickInterval = 100 * time.Millisecond

// NewSpinner creates a new spinner progress indicator.
// If format is "json", returns a no-op indicator.
// The spinner is not started automatically; call Start() to begin auto-ticking
// or use Update() for manual frame control.
func NewSpinner(label string, format string) Indicator {
	if format == "json" {
		return &noop{}
	}
	return &Spinner{label: label, active: true}
}

// Start launches a background goroutine that auto-advances the spinner frame
// every DefaultTickInterval. The goroutine stops when Finish() is called.
func (s *Spinner) Start() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.active || s.stopCh != nil {
		// Already started or already finished.
		return
	}
	s.stopCh = make(chan struct{})
	go s.run()
}

// run is the auto-tick loop. It ticks at the spinner's tick interval until
// stopCh is closed.
func (s *Spinner) run() {
	interval := s.tickInterval
	if interval == 0 {
		interval = DefaultTickInterval
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-s.stopCh:
			return
		case <-ticker.C:
			s.mu.Lock()
			if !s.active {
				s.mu.Unlock()
				return
			}
			frame := spinnerFrames[s.frame%len(spinnerFrames)]
			fmt.Fprintf(os.Stderr, "\r%s %s", frame, s.label)
			s.frame++
			s.mu.Unlock()
		}
	}
}

// SetLabel changes the spinner's display label. Thread-safe.
// Useful for showing status changes like "Rate limited, retrying in 5s...".
func (s *Spinner) SetLabel(label string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.label = label
}

// Update advances the spinner by one frame. The current parameter is ignored.
// This is the manual mode: callers can use Update() in a loop instead of Start().
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

// Finish stops the auto-tick goroutine (if running) and clears the spinner
// line from stderr. Safe to call multiple times.
func (s *Spinner) Finish() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.active {
		return
	}
	s.active = false
	if s.stopCh != nil {
		close(s.stopCh)
	}
	fmt.Fprintf(os.Stderr, "\r\033[K")
}

// noop is a silent progress indicator used in JSON mode.
type noop struct{}

func (n *noop) Start()              {}
func (n *noop) SetLabel(_ string)   {}
func (n *noop) Update(_ int)        {}
func (n *noop) Finish()             {}
