package errors

import (
	"errors"
	"fmt"
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

func TestClassifyError_Nil(t *testing.T) {
	action := classifyError(nil)
	if action != nil {
		t.Errorf("expected nil action for nil error, got %+v", action)
	}
}

func TestClassifyError_CLIError(t *testing.T) {
	cliErr := &CLIError{
		Message:    "auth failed",
		Suggestion: "run 'fm auth login'",
		ExitCode:   ExitAuthError,
	}
	action := classifyError(cliErr)
	if action == nil {
		t.Fatal("expected non-nil action")
	}
	if action.Message != "auth failed" {
		t.Errorf("Message = %q, want %q", action.Message, "auth failed")
	}
	if action.Suggestion != "run 'fm auth login'" {
		t.Errorf("Suggestion = %q, want %q", action.Suggestion, "run 'fm auth login'")
	}
	if action.ExitCode != ExitAuthError {
		t.Errorf("ExitCode = %d, want %d", action.ExitCode, ExitAuthError)
	}
}

func TestClassifyError_WrappedCLIError(t *testing.T) {
	cliErr := &CLIError{
		Message:  "wrapped error",
		ExitCode: ExitAuthError,
	}
	// Wrap the CLIError with fmt.Errorf so errors.As must unwrap it.
	wrapped := fmt.Errorf("outer: %w", cliErr)
	action := classifyError(wrapped)
	if action == nil {
		t.Fatal("expected non-nil action for wrapped CLIError")
	}
	if action.ExitCode != ExitAuthError {
		t.Errorf("ExitCode = %d, want %d", action.ExitCode, ExitAuthError)
	}
	if action.Message != "wrapped error" {
		t.Errorf("Message = %q, want %q", action.Message, "wrapped error")
	}
}

func TestClassifyError_PlainError(t *testing.T) {
	action := classifyError(fmt.Errorf("something broke"))
	if action == nil {
		t.Fatal("expected non-nil action")
	}
	if action.Message != "something broke" {
		t.Errorf("Message = %q, want %q", action.Message, "something broke")
	}
	if action.Suggestion != "" {
		t.Errorf("Suggestion = %q, want empty", action.Suggestion)
	}
	if action.ExitCode != ExitError {
		t.Errorf("ExitCode = %d, want %d", action.ExitCode, ExitError)
	}
}

func TestWrap(t *testing.T) {
	original := fmt.Errorf("connection refused")
	wrapped := Wrap(original, "could not reach server", "check your network connection")

	if wrapped.Message != "could not reach server" {
		t.Errorf("Message = %q, want %q", wrapped.Message, "could not reach server")
	}
	if wrapped.Suggestion != "check your network connection" {
		t.Errorf("Suggestion = %q, want %q", wrapped.Suggestion, "check your network connection")
	}
	if wrapped.ExitCode != ExitError {
		t.Errorf("ExitCode = %d, want %d", wrapped.ExitCode, ExitError)
	}
	if wrapped.Err != original {
		t.Error("Err does not point to original error")
	}
}

func TestWrapAuth(t *testing.T) {
	original := fmt.Errorf("token expired")
	wrapped := WrapAuth(original, "authentication failed", "run 'fm auth login'")

	if wrapped.ExitCode != ExitAuthError {
		t.Errorf("ExitCode = %d, want %d", wrapped.ExitCode, ExitAuthError)
	}
	if wrapped.Message != "authentication failed" {
		t.Errorf("Message = %q, want %q", wrapped.Message, "authentication failed")
	}
	if wrapped.Suggestion != "run 'fm auth login'" {
		t.Errorf("Suggestion = %q, want %q", wrapped.Suggestion, "run 'fm auth login'")
	}
	if wrapped.Err != original {
		t.Error("Err does not point to original error")
	}
}

func TestUnwrap(t *testing.T) {
	original := fmt.Errorf("underlying issue")
	cliErr := Wrap(original, "user message", "")

	if cliErr.Unwrap() != original {
		t.Error("Unwrap() did not return the original error")
	}

	// errors.Is should work through the chain.
	if !errors.Is(cliErr, original) {
		t.Error("errors.Is failed to find original error through CLIError")
	}
}

func TestUnwrap_Nil(t *testing.T) {
	cliErr := NewCLIError(ExitError, "no underlying error")
	if cliErr.Unwrap() != nil {
		t.Errorf("Unwrap() = %v, want nil", cliErr.Unwrap())
	}
}
