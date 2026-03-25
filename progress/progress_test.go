package progress

import (
	"testing"
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

func TestSpinnerLifecycle(t *testing.T) {
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

func TestNoopDoesNothing(t *testing.T) {
	n := &noop{}
	// Just verify these don't panic
	n.Update(100)
	n.Finish()
}
