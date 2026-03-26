// Package progress provides progress indicators for CLI tools.
//
// Progress indicators write to stderr and only display in table output mode.
// In JSON mode, they are completely suppressed to avoid polluting output.
package progress

import (
	"fmt"
	"io"
	"os"
	"sync"
	"time"
)

// Indicator is the interface for all progress indicators.
type Indicator interface {
	Update(current int)
	SetMessage(msg string)
	Finish()
}

// Counter shows a running count of items processed.
// Example: "Fetching messages... 47"
type Counter struct {
	label  string
	active bool
	mu     sync.Mutex
	w      io.Writer
}

// NewCounter creates a new counter progress indicator.
// If format is "json", returns a no-op indicator that produces no output.
func NewCounter(label string, format string) Indicator {
	if format == "json" {
		return &noop{}
	}
	return &Counter{label: label, active: true, w: os.Stderr}
}

// Update displays the current count. Overwrites the previous line on stderr.
func (c *Counter) Update(current int) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if !c.active {
		return
	}
	fmt.Fprintf(c.w, "\r%s... %d", c.label, current)
}

// SetMessage updates the counter label text.
func (c *Counter) SetMessage(msg string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.label = msg
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
	fmt.Fprintf(c.w, "\r\033[K")
}

// Spinner shows a spinning indicator for operations with unknown total.
// It self-animates via a background goroutine that advances frames at ~100ms.
type Spinner struct {
	label  string
	active bool
	frame  int
	mu     sync.Mutex
	w      io.Writer
	done   chan struct{}
}

var spinnerFrames = []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}

// NewSpinner creates a new spinner progress indicator.
// If format is "json", returns a no-op indicator.
// The spinner starts self-animating immediately at ~100ms per frame.
func NewSpinner(label string, format string) Indicator {
	if format == "json" {
		return &noop{}
	}
	s := &Spinner{
		label:  label,
		active: true,
		w:      os.Stderr,
		done:   make(chan struct{}),
	}
	go s.run()
	return s
}

// run is the background goroutine that auto-advances the spinner frames.
func (s *Spinner) run() {
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()
	for {
		select {
		case <-s.done:
			return
		case <-ticker.C:
			s.tick()
		}
	}
}

// tick advances the spinner by one frame and renders it.
func (s *Spinner) tick() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.active {
		return
	}
	frame := spinnerFrames[s.frame%len(spinnerFrames)]
	fmt.Fprintf(s.w, "\r%s %s", frame, s.label)
	s.frame++
}

// Update manually advances the spinner by one frame. The current parameter is ignored.
// Note: the spinner auto-animates, so calling Update is not required.
func (s *Spinner) Update(_ int) {
	s.tick()
}

// SetMessage updates the spinner label text.
func (s *Spinner) SetMessage(msg string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.label = msg
}

// Finish stops the spinner animation and clears the line from stderr.
func (s *Spinner) Finish() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.active {
		return
	}
	s.active = false
	close(s.done)
	fmt.Fprintf(s.w, "\r\033[K")
}

// noop is a silent progress indicator used in JSON mode.
type noop struct{}

func (n *noop) Update(_ int)       {}
func (n *noop) SetMessage(_ string) {}
func (n *noop) Finish()            {}
