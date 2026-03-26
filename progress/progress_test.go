package progress

import (
	"bytes"
	"strings"
	"testing"
	"time"
)

func TestNewCounterJSON(t *testing.T) {
	p := NewCounter("Fetching", "json")
	if _, ok := p.(*noop); !ok {
		t.Error("expected noop indicator for json format")
	}
}

func TestNewCounterTable(t *testing.T) {
	p := NewCounter("Fetching", "table")
	if _, ok := p.(*Counter); !ok {
		t.Error("expected Counter indicator for table format")
	}
}

func TestNewSpinnerJSON(t *testing.T) {
	p := NewSpinner("Loading", "json")
	if _, ok := p.(*noop); !ok {
		t.Error("expected noop indicator for json format")
	}
}

func TestNewSpinnerTable(t *testing.T) {
	p := NewSpinner("Loading", "table")
	defer p.Finish()
	if _, ok := p.(*Spinner); !ok {
		t.Error("expected Spinner indicator for table format")
	}
}

func TestCounterLifecycle(t *testing.T) {
	var buf bytes.Buffer
	c := &Counter{label: "Test", active: true, w: &buf}
	// Should not panic
	c.Update(10)
	c.Update(20)
	c.Finish()
	// After finish, update should be no-op
	c.Update(30)
}

func TestCounterSetMessage(t *testing.T) {
	var buf bytes.Buffer
	c := &Counter{label: "First", active: true, w: &buf}
	c.Update(1)
	if !strings.Contains(buf.String(), "First") {
		t.Error("expected output to contain 'First'")
	}
	buf.Reset()
	c.SetMessage("Second")
	c.Update(2)
	if !strings.Contains(buf.String(), "Second") {
		t.Error("expected output to contain 'Second' after SetMessage")
	}
	c.Finish()
}

func TestSpinnerLifecycle(t *testing.T) {
	var buf bytes.Buffer
	s := &Spinner{label: "Test", active: true, w: &buf, done: make(chan struct{})}
	s.Update(0)
	s.Update(0)
	s.Update(0)
	if s.frame != 3 {
		t.Errorf("frame = %d, want 3", s.frame)
	}
	s.Finish()
	s.Update(0)
	if s.frame != 3 {
		t.Error("frame should not advance after Finish")
	}
}

func TestSpinnerAutoAnimates(t *testing.T) {
	var buf bytes.Buffer
	s := &Spinner{label: "Auto", active: true, w: &buf, done: make(chan struct{})}
	go s.run()

	// Wait enough time for several frames (~350ms should yield at least 3 frames).
	time.Sleep(350 * time.Millisecond)
	s.Finish()

	s.mu.Lock()
	frames := s.frame
	s.mu.Unlock()

	if frames < 3 {
		t.Errorf("expected at least 3 auto-animated frames, got %d", frames)
	}

	output := buf.String()
	if !strings.Contains(output, "Auto") {
		t.Error("expected spinner output to contain label 'Auto'")
	}
}

func TestSpinnerSetMessage(t *testing.T) {
	var buf bytes.Buffer
	s := &Spinner{label: "Before", active: true, w: &buf, done: make(chan struct{})}
	s.Update(0)
	if !strings.Contains(buf.String(), "Before") {
		t.Error("expected output to contain 'Before'")
	}
	buf.Reset()
	s.SetMessage("After")
	s.Update(0)
	if !strings.Contains(buf.String(), "After") {
		t.Error("expected output to contain 'After' after SetMessage")
	}
	s.Finish()
}

func TestSpinnerFinishStopsGoroutine(t *testing.T) {
	var buf bytes.Buffer
	s := &Spinner{label: "Stop", active: true, w: &buf, done: make(chan struct{})}
	go s.run()

	time.Sleep(150 * time.Millisecond)
	s.Finish()

	s.mu.Lock()
	frameAtFinish := s.frame
	s.mu.Unlock()

	// Wait a bit more and confirm no more frames are written.
	time.Sleep(250 * time.Millisecond)
	s.mu.Lock()
	frameAfterWait := s.frame
	s.mu.Unlock()

	if frameAfterWait != frameAtFinish {
		t.Errorf("expected no more frames after Finish, got %d (was %d at finish)", frameAfterWait, frameAtFinish)
	}
}

func TestNoopDoesNothing(t *testing.T) {
	n := &noop{}
	// Just verify these don't panic
	n.Update(100)
	n.SetMessage("anything")
	n.Finish()
}
