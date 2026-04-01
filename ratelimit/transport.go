package ratelimit

import (
	"io"
	"math"
	"math/rand"
	"net/http"
	"time"
)

const (
	// DefaultMaxRetries is the maximum number of retry attempts before giving up.
	DefaultMaxRetries = 5

	// DefaultBaseDelay is the initial delay before the first retry.
	DefaultBaseDelay = 1 * time.Second

	// DefaultMaxDelay is the upper bound on any single retry delay.
	DefaultMaxDelay = 60 * time.Second
)

// RetryTransport wraps an http.RoundTripper and retries requests that receive
// HTTP 429 (Too Many Requests) or 5xx responses with exponential backoff and
// jitter. It respects Retry-After headers when present.
//
// Zero-valued fields use sensible defaults when RoundTrip is called:
//   - Base defaults to http.DefaultTransport
//   - MaxRetries defaults to 5
//   - BaseDelay defaults to 1s
//   - MaxDelay defaults to 60s
//   - RetryOn5xx defaults to true (per CLI_STANDARDS)
type RetryTransport struct {
	// Base is the underlying transport. Defaults to http.DefaultTransport.
	Base http.RoundTripper

	// MaxRetries is the maximum number of retry attempts. Defaults to 5.
	MaxRetries int

	// BaseDelay is the initial delay before the first retry. Defaults to 1s.
	BaseDelay time.Duration

	// MaxDelay is the upper bound on any single retry delay. Defaults to 60s.
	MaxDelay time.Duration

	// RetryOn5xx controls whether HTTP 5xx responses are retried.
	// Since the zero value (false) differs from the desired default (true),
	// callers must set this explicitly to false to disable 5xx retries.
	// Use NewRetryTransport to get the recommended defaults.
	RetryOn5xx bool

	// OnRetry is an optional callback invoked before each retry sleep.
	// Arguments: the attempt number (0-based), the delay about to be applied,
	// and the HTTP status code that triggered the retry.
	OnRetry func(attempt int, delay time.Duration, statusCode int)

	// timerFunc is called instead of time.Sleep during retries. Override in
	// tests to avoid real sleeps. Defaults to time.Sleep.
	timerFunc func(time.Duration)

	// defaultsApplied tracks whether defaults have been filled in.
	defaultsApplied bool
}

// NewRetryTransport creates a RetryTransport with the recommended defaults.
// Pass nil for base to use http.DefaultTransport.
func NewRetryTransport(base http.RoundTripper) *RetryTransport {
	if base == nil {
		base = http.DefaultTransport
	}
	return &RetryTransport{
		Base:       base,
		MaxRetries: DefaultMaxRetries,
		BaseDelay:  DefaultBaseDelay,
		MaxDelay:   DefaultMaxDelay,
		RetryOn5xx: true,
	}
}

// applyDefaults fills in zero-valued fields with sensible defaults.
func (t *RetryTransport) applyDefaults() {
	if t.defaultsApplied {
		return
	}
	if t.Base == nil {
		t.Base = http.DefaultTransport
	}
	if t.MaxRetries == 0 {
		t.MaxRetries = DefaultMaxRetries
	}
	if t.BaseDelay == 0 {
		t.BaseDelay = DefaultBaseDelay
	}
	if t.MaxDelay == 0 {
		t.MaxDelay = DefaultMaxDelay
	}
	if t.timerFunc == nil {
		t.timerFunc = time.Sleep
	}
	t.defaultsApplied = true
}

// RoundTrip executes the request and retries on retryable status codes.
// It drains and closes the response body before each retry. If all retries
// are exhausted, the last response is returned (not swallowed).
func (t *RetryTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	t.applyDefaults()

	var resp *http.Response
	var err error

	for attempt := 0; attempt <= t.MaxRetries; attempt++ {
		resp, err = t.Base.RoundTrip(req)
		if err != nil {
			return nil, err
		}

		if !t.isRetryable(resp.StatusCode) {
			return resp, nil
		}

		// Last attempt - return the response as-is.
		if attempt == t.MaxRetries {
			return resp, nil
		}

		delay := t.backoffDelay(attempt, resp.Header)

		// Honour Retry-After: if the header specifies a longer delay, use it.
		if ra := ParseRetryAfter(resp.Header.Get("Retry-After")); ra > delay {
			delay = ra
		}

		if t.OnRetry != nil {
			t.OnRetry(attempt, delay, resp.StatusCode)
		}

		// Drain and close the body before retrying so the underlying TCP
		// connection can be reused by HTTP keep-alive.
		io.Copy(io.Discard, resp.Body) //nolint:errcheck // best-effort drain
		resp.Body.Close()

		// Sleep (or call the injected timer function).
		t.timerFunc(delay)
	}

	// Unreachable: the loop always returns.
	return resp, err
}

// isRetryable returns true if the status code should trigger a retry.
func (t *RetryTransport) isRetryable(statusCode int) bool {
	if statusCode == http.StatusTooManyRequests {
		return true
	}
	if t.RetryOn5xx && statusCode >= 500 && statusCode < 600 {
		return true
	}
	return false
}

// backoffDelay computes the exponential backoff delay for the given attempt.
// The formula is BaseDelay * 2^attempt, capped at MaxDelay, with +/-25% jitter.
func (t *RetryTransport) backoffDelay(attempt int, header http.Header) time.Duration {
	delay := float64(t.BaseDelay) * math.Pow(2, float64(attempt))
	if delay > float64(t.MaxDelay) {
		delay = float64(t.MaxDelay)
	}

	// Apply +/-25% jitter: multiply by a random factor in [0.75, 1.25].
	jitter := 0.75 + 0.5*rand.Float64() //nolint:gosec // jitter does not need crypto rand
	delay *= jitter

	if delay < 0 {
		delay = float64(t.BaseDelay)
	}

	return time.Duration(delay)
}
