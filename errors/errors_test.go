package errors

import (
	"strings"
	"testing"
)

func TestCLIError(t *testing.T) {
	err := NewCLIError(ExitError, "something went wrong").
		WithCode(500).
		WithSuggestion("try again later")

	if err.Error() != "something went wrong" {
		t.Errorf("Error() = %q, want %q", err.Error(), "something went wrong")
	}
	if err.Code != 500 {
		t.Errorf("Code = %d, want 500", err.Code)
	}
	if err.Suggestion != "try again later" {
		t.Errorf("Suggestion = %q", err.Suggestion)
	}
	if err.ExitCode != ExitError {
		t.Errorf("ExitCode = %d, want %d", err.ExitCode, ExitError)
	}
}

func TestHandleHTTPError_AuthDenied(t *testing.T) {
	// Auth check returns invalid
	err := HandleHTTPError(403, "/api/resource", "fm", func() (bool, error) {
		return false, nil
	})
	if err.ExitCode != ExitAuthError {
		t.Errorf("ExitCode = %d, want %d", err.ExitCode, ExitAuthError)
	}
	if !strings.Contains(err.Message, "access denied") {
		t.Errorf("expected 'access denied' in message, got %q", err.Message)
	}
	if !strings.Contains(err.Suggestion, "auth login") {
		t.Errorf("expected 'auth login' in suggestion, got %q", err.Suggestion)
	}
}

func TestHandleHTTPError_AuthValid(t *testing.T) {
	err := HandleHTTPError(403, "/api/resource", "fm", func() (bool, error) {
		return true, nil
	})
	if !strings.Contains(err.Suggestion, "permission") {
		t.Errorf("expected permission hint in suggestion, got %q", err.Suggestion)
	}
}

func TestHandleHTTPError_RateLimit(t *testing.T) {
	err := HandleHTTPError(429, "/api/search", "fm", nil)
	if err.ExitCode != ExitSuccess {
		t.Errorf("429 should exit with success (partial data valid), got %d", err.ExitCode)
	}
	if err.Code != 429 {
		t.Errorf("Code = %d, want 429", err.Code)
	}
}

func TestHandleHTTPError_ServerError(t *testing.T) {
	err := HandleHTTPError(502, "/api/data", "fm", nil)
	if err.ExitCode != ExitError {
		t.Errorf("ExitCode = %d, want %d", err.ExitCode, ExitError)
	}
	if !strings.Contains(err.Message, "502") {
		t.Errorf("expected status code in message, got %q", err.Message)
	}
}

func TestPartialResult(t *testing.T) {
	pr := NewPartialResult([]string{"a", "b"}, "rate limited after page 2")
	if !pr.Partial {
		t.Error("expected Partial=true")
	}
	if len(pr.Results) != 2 {
		t.Errorf("expected 2 results, got %d", len(pr.Results))
	}
	if pr.Error != "rate limited after page 2" {
		t.Errorf("Error = %q", pr.Error)
	}
}

func TestCompleteResult(t *testing.T) {
	cr := NewCompleteResult([]string{"a", "b", "c"})
	if cr.Partial {
		t.Error("expected Partial=false")
	}
	if len(cr.Results) != 3 {
		t.Errorf("expected 3 results, got %d", len(cr.Results))
	}
}
