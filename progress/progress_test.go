package progress

import (
	"sync"
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
	if _, ok := p.(*Spinner); !ok {
		t.Error("expected Spinner indicator for table format")
	}
}

func TestCounterLifecycle(t *testing.T) {
	c := &Counter{label: "Test", active: true}
	// Should not panic
	c.Update(10)
	c.Update(20)
	c.Finish()
	// After finish, update should be no-op
	c.Update(30)
}

func TestCounterSetLabel(t *testing.T) {
	c := &Counter{label: "Test", active: true}
	c.SetLabel("New label")
	c.mu.Lock()
	if c.label != "New label" {
		t.Errorf("label = %q, want %q", c.label, "New label")
	}
	c.mu.Unlock()
	c.Finish()
}

func TestCounterStartIsNoop(t *testing.T) {
	c := &Counter{label: "Test", active: true}
	// Start should not panic and is a no-op for Counter.
	c.Start()
	c.Finish()
}

func TestSpinnerManualLifecycle(t *testing.T) {
	s := &Spinner{label: "Test", active: true}
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

func TestSpinnerStartFinishLifecycle(t *testing.T) {
	s := NewSpinner("Loading", "table").(*Spinner)
	s.tickInterval = 10 * time.Millisecond
	s.Start()

	// Let the goroutine tick a few times.
	time.Sleep(50 * time.Millisecond)

	s.mu.Lock()
	frames := s.frame
	s.mu.Unlock()

	if frames == 0 {
		t.Error("expected spinner to have advanced at least one frame after Start()")
	}

	s.Finish()

	// After Finish, the goroutine should have stopped. Record frame and wait.
	s.mu.Lock()
	framesAfterFinish := s.frame
	s.mu.Unlock()

	time.Sleep(30 * time.Millisecond)

	s.mu.Lock()
	framesLater := s.frame
	s.mu.Unlock()

	if framesLater != framesAfterFinish {
		t.Errorf("spinner continued ticking after Finish: %d -> %d", framesAfterFinish, framesLater)
	}
}

func TestSpinnerSetLabel(t *testing.T) {
	s := &Spinner{label: "Initial", active: true}
	s.SetLabel("Updated")
	s.mu.Lock()
	if s.label != "Updated" {
		t.Errorf("label = %q, want %q", s.label, "Updated")
	}
	s.mu.Unlock()
}

func TestSpinnerFinishIdempotent(t *testing.T) {
	s := &Spinner{label: "Test", active: true}
	s.Start()
	// Calling Finish multiple times must not panic.
	s.Finish()
	s.Finish()
	s.Finish()
}

func TestSpinnerStartIdempotent(t *testing.T) {
	s := NewSpinner("Loading", "table").(*Spinner)
	s.tickInterval = 10 * time.Millisecond
	s.Start()
	s.Start() // Second call should be a no-op.
	time.Sleep(30 * time.Millisecond)
	s.Finish()
}

func TestNoopStartSetLabelFinish(t *testing.T) {
	n := &noop{}
	// All methods should be no-ops and not panic.
	n.Start()
	n.SetLabel("anything")
	n.Update(100)
	n.Finish()
	n.Finish() // idempotent
}

func TestNoopViaNewSpinnerJSON(t *testing.T) {
	p := NewSpinner("Loading", "json")
	// All methods should work without panic.
	p.Start()
	p.SetLabel("changed")
	p.Update(0)
	p.Finish()
	p.Finish()
}

func TestSpinnerConcurrentAccess(t *testing.T) {
	s := NewSpinner("Loading", "table").(*Spinner)
	s.tickInterval = 10 * time.Millisecond
	s.Start()

	var wg sync.WaitGroup

	// Concurrently set labels from multiple goroutines.
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			for j := 0; j < 20; j++ {
				s.SetLabel("label from goroutine")
				time.Sleep(time.Millisecond)
			}
		}(i)
	}

	// Concurrently call Update from another goroutine.
	wg.Add(1)
	go func() {
		defer wg.Done()
		for j := 0; j < 20; j++ {
			s.Update(0)
			time.Sleep(time.Millisecond)
		}
	}()

	wg.Wait()
	s.Finish()
}

func TestSpinnerNotStartedByDefault(t *testing.T) {
	s := NewSpinner("Loading", "table").(*Spinner)
	if s.stopCh != nil {
		t.Error("expected stopCh to be nil before Start()")
	}
	// Should work in manual mode without Start().
	s.Update(0)
	s.Update(0)
	if s.frame != 2 {
		t.Errorf("frame = %d, want 2", s.frame)
	}
	s.Finish()
}

func TestIndicatorInterfaceCompliance(t *testing.T) {
	// Verify all types implement the Indicator interface.
	var _ Indicator = &Spinner{}
	var _ Indicator = &Counter{}
	var _ Indicator = &noop{}
}
