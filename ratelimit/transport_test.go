package ratelimit

import (
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"
)

// noopSleep replaces time.Sleep so tests run instantly.
func noopSleep(time.Duration) {}

// recordingSleep captures the durations passed to the timer function.
type recordingSleep struct {
	mu     sync.Mutex
	delays []time.Duration
}

func (r *recordingSleep) sleep(d time.Duration) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.delays = append(r.delays, d)
}

func (r *recordingSleep) getDelays() []time.Duration {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make([]time.Duration, len(r.delays))
	copy(out, r.delays)
	return out
}

// statusSequenceHandler returns an httptest.Server that responds with the
// given status codes in sequence. Once all codes are exhausted, it returns 200.
func statusSequenceHandler(statuses []int, retryAfter string) *httptest.Server {
	var mu sync.Mutex
	idx := 0

	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		var code int
		if idx < len(statuses) {
			code = statuses[idx]
			idx++
		} else {
			code = http.StatusOK
		}
		mu.Unlock()

		if retryAfter != "" && (code == http.StatusTooManyRequests || code >= 500) {
			w.Header().Set("Retry-After", retryAfter)
		}
		w.WriteHeader(code)
		w.Write([]byte("ok")) //nolint:errcheck
	}))
}

func TestRetryOn429ThenSuccess(t *testing.T) {
	srv := statusSequenceHandler([]int{429, 429}, "")
	defer srv.Close()

	tr := &RetryTransport{
		Base:       srv.Client().Transport,
		MaxRetries: 5,
		BaseDelay:  1 * time.Millisecond,
		MaxDelay:   100 * time.Millisecond,
		RetryOn5xx: true,
		timerFunc:  noopSleep,
	}

	req, _ := http.NewRequest("GET", srv.URL, nil)
	resp, err := tr.RoundTrip(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}
	resp.Body.Close()
}

func TestRetryAfterIntegerSeconds(t *testing.T) {
	srv := statusSequenceHandler([]int{429}, "3")
	defer srv.Close()

	rec := &recordingSleep{}
	tr := &RetryTransport{
		Base:       srv.Client().Transport,
		MaxRetries: 5,
		BaseDelay:  1 * time.Millisecond, // very small so Retry-After wins
		MaxDelay:   100 * time.Second,
		RetryOn5xx: true,
		timerFunc:  rec.sleep,
	}

	req, _ := http.NewRequest("GET", srv.URL, nil)
	resp, err := tr.RoundTrip(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}
	resp.Body.Close()

	delays := rec.getDelays()
	if len(delays) != 1 {
		t.Fatalf("expected 1 delay, got %d", len(delays))
	}
	// Retry-After is 3 seconds, which should be larger than the computed
	// backoff (1ms * 2^0 * jitter), so the delay should be 3s.
	if delays[0] < 3*time.Second {
		t.Errorf("expected delay >= 3s (from Retry-After), got %v", delays[0])
	}
}

func TestRetryAfterHTTPDate(t *testing.T) {
	// Use a date 5 seconds in the future.
	futureTime := time.Now().Add(5 * time.Second)
	retryAfterDate := futureTime.UTC().Format(http.TimeFormat)

	srv := statusSequenceHandler([]int{429}, retryAfterDate)
	defer srv.Close()

	rec := &recordingSleep{}
	tr := &RetryTransport{
		Base:       srv.Client().Transport,
		MaxRetries: 5,
		BaseDelay:  1 * time.Millisecond,
		MaxDelay:   100 * time.Second,
		RetryOn5xx: true,
		timerFunc:  rec.sleep,
	}

	req, _ := http.NewRequest("GET", srv.URL, nil)
	resp, err := tr.RoundTrip(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}
	resp.Body.Close()

	delays := rec.getDelays()
	if len(delays) != 1 {
		t.Fatalf("expected 1 delay, got %d", len(delays))
	}
	// The Retry-After date is ~5s in the future, so delay should be > 3s
	// (allowing some slack for timing).
	if delays[0] < 3*time.Second {
		t.Errorf("expected delay >= 3s (from Retry-After HTTP-date), got %v", delays[0])
	}
}

func TestRetryOn5xxEnabled(t *testing.T) {
	srv := statusSequenceHandler([]int{503, 502}, "")
	defer srv.Close()

	tr := &RetryTransport{
		Base:       srv.Client().Transport,
		MaxRetries: 5,
		BaseDelay:  1 * time.Millisecond,
		MaxDelay:   100 * time.Millisecond,
		RetryOn5xx: true,
		timerFunc:  noopSleep,
	}

	req, _ := http.NewRequest("GET", srv.URL, nil)
	resp, err := tr.RoundTrip(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200 after retrying 5xx, got %d", resp.StatusCode)
	}
	resp.Body.Close()
}

func TestRetryOn5xxDisabled(t *testing.T) {
	srv := statusSequenceHandler([]int{503}, "")
	defer srv.Close()

	tr := &RetryTransport{
		Base:       srv.Client().Transport,
		MaxRetries: 5,
		BaseDelay:  1 * time.Millisecond,
		MaxDelay:   100 * time.Millisecond,
		RetryOn5xx: false,
		timerFunc:  noopSleep,
	}

	req, _ := http.NewRequest("GET", srv.URL, nil)
	resp, err := tr.RoundTrip(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.StatusCode != http.StatusServiceUnavailable {
		t.Errorf("expected 503 (no retry), got %d", resp.StatusCode)
	}
	resp.Body.Close()
}

func TestMaxRetriesExhausted(t *testing.T) {
	// Return 429 every time - more than MaxRetries.
	statuses := make([]int, 10)
	for i := range statuses {
		statuses[i] = 429
	}
	srv := statusSequenceHandler(statuses, "")
	defer srv.Close()

	tr := &RetryTransport{
		Base:       srv.Client().Transport,
		MaxRetries: 3,
		BaseDelay:  1 * time.Millisecond,
		MaxDelay:   100 * time.Millisecond,
		RetryOn5xx: true,
		timerFunc:  noopSleep,
	}

	req, _ := http.NewRequest("GET", srv.URL, nil)
	resp, err := tr.RoundTrip(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Should return the last 429 response, not swallow it.
	if resp.StatusCode != http.StatusTooManyRequests {
		t.Errorf("expected 429 after exhausted retries, got %d", resp.StatusCode)
	}
	resp.Body.Close()
}

func TestOnRetryCallback(t *testing.T) {
	srv := statusSequenceHandler([]int{429, 503}, "")
	defer srv.Close()

	type retryRecord struct {
		attempt    int
		delay      time.Duration
		statusCode int
	}
	var mu sync.Mutex
	var records []retryRecord

	tr := &RetryTransport{
		Base:       srv.Client().Transport,
		MaxRetries: 5,
		BaseDelay:  1 * time.Millisecond,
		MaxDelay:   100 * time.Millisecond,
		RetryOn5xx: true,
		timerFunc:  noopSleep,
		OnRetry: func(attempt int, delay time.Duration, statusCode int) {
			mu.Lock()
			defer mu.Unlock()
			records = append(records, retryRecord{attempt, delay, statusCode})
		},
	}

	req, _ := http.NewRequest("GET", srv.URL, nil)
	resp, err := tr.RoundTrip(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}
	resp.Body.Close()

	mu.Lock()
	defer mu.Unlock()
	if len(records) != 2 {
		t.Fatalf("expected 2 OnRetry calls, got %d", len(records))
	}
	if records[0].attempt != 0 || records[0].statusCode != 429 {
		t.Errorf("first retry: attempt=%d status=%d, expected 0/429", records[0].attempt, records[0].statusCode)
	}
	if records[1].attempt != 1 || records[1].statusCode != 503 {
		t.Errorf("second retry: attempt=%d status=%d, expected 1/503", records[1].attempt, records[1].statusCode)
	}
	if records[0].delay <= 0 {
		t.Error("expected positive delay on first retry")
	}
}

func TestJitterWithinBounds(t *testing.T) {
	// Run many iterations to verify jitter stays within +/-25%.
	tr := &RetryTransport{
		BaseDelay: 1 * time.Second,
		MaxDelay:  60 * time.Second,
	}

	for attempt := 0; attempt < 4; attempt++ {
		base := float64(tr.BaseDelay) * float64(int(1)<<uint(attempt)) // BaseDelay * 2^attempt
		if base > float64(tr.MaxDelay) {
			base = float64(tr.MaxDelay)
		}
		low := time.Duration(base * 0.75)
		high := time.Duration(base * 1.25)

		for i := 0; i < 1000; i++ {
			d := tr.backoffDelay(attempt, http.Header{})
			if d < low || d > high {
				t.Errorf("attempt %d, iteration %d: delay %v outside [%v, %v]",
					attempt, i, d, low, high)
			}
		}
	}
}

func TestParseRetryAfter_IntegerSeconds(t *testing.T) {
	d := ParseRetryAfter("120")
	if d != 120*time.Second {
		t.Errorf("expected 120s, got %v", d)
	}
}

func TestParseRetryAfter_Zero(t *testing.T) {
	d := ParseRetryAfter("0")
	if d != 0 {
		t.Errorf("expected 0, got %v", d)
	}
}

func TestParseRetryAfter_Negative(t *testing.T) {
	d := ParseRetryAfter("-5")
	if d != 0 {
		t.Errorf("expected 0 for negative, got %v", d)
	}
}

func TestParseRetryAfter_Empty(t *testing.T) {
	d := ParseRetryAfter("")
	if d != 0 {
		t.Errorf("expected 0, got %v", d)
	}
}

func TestParseRetryAfter_Garbage(t *testing.T) {
	d := ParseRetryAfter("not-a-number")
	if d != 0 {
		t.Errorf("expected 0, got %v", d)
	}
}

func TestParseRetryAfter_HTTPDate(t *testing.T) {
	future := time.Now().Add(10 * time.Second).UTC().Format(http.TimeFormat)
	d := ParseRetryAfter(future)
	// Should be roughly 10s (allow 8-12s for timing slack).
	if d < 8*time.Second || d > 12*time.Second {
		t.Errorf("expected ~10s for HTTP-date, got %v", d)
	}
}

func TestParseRetryAfter_PastHTTPDate(t *testing.T) {
	past := time.Now().Add(-10 * time.Second).UTC().Format(http.TimeFormat)
	d := ParseRetryAfter(past)
	if d != 0 {
		t.Errorf("expected 0 for past date, got %v", d)
	}
}

func TestNewRetryTransport_Defaults(t *testing.T) {
	tr := NewRetryTransport(nil)
	if tr.Base == nil {
		t.Error("expected Base to default to http.DefaultTransport")
	}
	if tr.MaxRetries != DefaultMaxRetries {
		t.Errorf("MaxRetries = %d, want %d", tr.MaxRetries, DefaultMaxRetries)
	}
	if tr.BaseDelay != DefaultBaseDelay {
		t.Errorf("BaseDelay = %v, want %v", tr.BaseDelay, DefaultBaseDelay)
	}
	if tr.MaxDelay != DefaultMaxDelay {
		t.Errorf("MaxDelay = %v, want %v", tr.MaxDelay, DefaultMaxDelay)
	}
	if !tr.RetryOn5xx {
		t.Error("expected RetryOn5xx=true by default")
	}
}

func TestZeroValueDefaults(t *testing.T) {
	// A zero-value RetryTransport should still work after applyDefaults.
	tr := &RetryTransport{}
	tr.applyDefaults()
	if tr.Base == nil {
		t.Error("expected Base to be set")
	}
	if tr.MaxRetries != DefaultMaxRetries {
		t.Errorf("MaxRetries = %d, want %d", tr.MaxRetries, DefaultMaxRetries)
	}
	if tr.timerFunc == nil {
		t.Error("expected timerFunc to be set")
	}
}
