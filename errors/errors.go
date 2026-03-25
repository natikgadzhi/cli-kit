// Package errors provides consistent HTTP error handling for CLI tools.
//
// It maps HTTP status codes to user-friendly error messages, handles auth
// verification on 401/403, and supports partial result patterns for 429 errors.
package errors

import (
	"encoding/json"
	"fmt"
	"os"
)

// Exit codes used across all tools.
const (
	ExitSuccess   = 0
	ExitError     = 1
	ExitAuthError = 2
)

// CLIError represents a structured error with an exit code and optional suggestion.
type CLIError struct {
	Message    string `json:"error"`
	Code       int    `json:"code,omitempty"`
	Suggestion string `json:"suggestion,omitempty"`
	ExitCode   int    `json:"-"`
}

func (e *CLIError) Error() string {
	return e.Message
}

// NewCLIError creates a new CLIError with the given message and exit code.
func NewCLIError(exitCode int, message string) *CLIError {
	return &CLIError{
		Message:  message,
		ExitCode: exitCode,
	}
}

// WithCode sets the HTTP status code on the error.
func (e *CLIError) WithCode(code int) *CLIError {
	e.Code = code
	return e
}

// WithSuggestion sets a suggestion message on the error.
func (e *CLIError) WithSuggestion(s string) *CLIError {
	e.Suggestion = s
	return e
}

// PrintError writes an error to stderr in the appropriate format.
// If jsonFormat is true, writes structured JSON. Otherwise writes plain text.
func PrintError(err *CLIError, jsonFormat bool) {
	if jsonFormat {
		enc := json.NewEncoder(os.Stderr)
		enc.Encode(err)
		return
	}
	fmt.Fprintf(os.Stderr, "Error: %s\n", err.Message)
	if err.Suggestion != "" {
		fmt.Fprintf(os.Stderr, "%s\n", err.Suggestion)
	}
}

// PrintWarning writes a warning to stderr.
func PrintWarning(msg string, jsonFormat bool) {
	if jsonFormat {
		enc := json.NewEncoder(os.Stderr)
		enc.Encode(map[string]string{"warning": msg})
		return
	}
	fmt.Fprintf(os.Stderr, "Warning: %s\n", msg)
}

// AuthChecker is a function that verifies whether current credentials are valid.
// It returns true if auth is valid, false otherwise.
type AuthChecker func() (valid bool, err error)

// HandleHTTPError processes an HTTP error status code and returns a CLIError.
// For 401/403, it optionally runs an auth check to provide better guidance.
// toolName is used in suggestion messages (e.g., "run 'fm auth login'").
func HandleHTTPError(statusCode int, resource string, toolName string, checkAuth AuthChecker) *CLIError {
	switch {
	case statusCode == 401 || statusCode == 403:
		return handleAuthError(statusCode, resource, toolName, checkAuth)
	case statusCode == 429:
		return handleRateLimit(resource)
	case statusCode >= 500:
		return handleServerError(statusCode, resource)
	default:
		return &CLIError{
			Message:  fmt.Sprintf("HTTP %d for %s", statusCode, resource),
			Code:     statusCode,
			ExitCode: ExitError,
		}
	}
}

func handleAuthError(statusCode int, resource string, toolName string, checkAuth AuthChecker) *CLIError {
	cliErr := &CLIError{
		Message:  fmt.Sprintf("access denied to %s (HTTP %d)", resource, statusCode),
		Code:     statusCode,
		ExitCode: ExitAuthError,
	}

	if checkAuth != nil {
		valid, _ := checkAuth()
		if !valid {
			cliErr.Suggestion = fmt.Sprintf("Authentication check failed — your token may have expired.\nRun %q to re-authenticate.", toolName+" auth login")
		} else {
			cliErr.Suggestion = "Your credentials are valid, but you may not have permission to access this resource."
		}
	} else {
		cliErr.Suggestion = fmt.Sprintf("Run %q to check your credentials.", toolName+" auth check")
	}
	return cliErr
}

func handleRateLimit(resource string) *CLIError {
	return &CLIError{
		Message:  fmt.Sprintf("rate limited while accessing %s (HTTP 429)", resource),
		Code:     429,
		ExitCode: ExitSuccess, // Partial data is still valid
	}
}

func handleServerError(statusCode int, resource string) *CLIError {
	return &CLIError{
		Message:  fmt.Sprintf("server error accessing %s (HTTP %d)", resource, statusCode),
		Code:     statusCode,
		ExitCode: ExitError,
	}
}

// PartialResult wraps results that may be incomplete due to errors.
type PartialResult[T any] struct {
	Partial bool   `json:"partial,omitempty"`
	Results []T    `json:"results"`
	Error   string `json:"error,omitempty"`
}

// NewPartialResult creates a PartialResult marked as incomplete with the given error.
func NewPartialResult[T any](results []T, err string) PartialResult[T] {
	return PartialResult[T]{
		Partial: true,
		Results: results,
		Error:   err,
	}
}

// NewCompleteResult creates a PartialResult with all results (not partial).
func NewCompleteResult[T any](results []T) PartialResult[T] {
	return PartialResult[T]{
		Results: results,
	}
}
