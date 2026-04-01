// Package ratelimit provides an HTTP retry transport with exponential backoff,
// jitter, and Retry-After header support.
package ratelimit

import (
	"net/http"
	"strconv"
	"strings"
	"time"
)

// ParseRetryAfter parses a Retry-After header value.
// It supports two formats:
//   - Integer seconds: "120"
//   - HTTP-date: "Fri, 31 Dec 1999 23:59:59 GMT" (any format recognized by http.ParseTime)
//
// Returns 0 if the value is empty or cannot be parsed.
func ParseRetryAfter(value string) time.Duration {
	value = strings.TrimSpace(value)
	if value == "" {
		return 0
	}

	// Try integer seconds first.
	if secs, err := strconv.Atoi(value); err == nil && secs > 0 {
		return time.Duration(secs) * time.Second
	}

	// Try HTTP-date (supports RFC 1123, RFC 850, and ANSI C asctime formats).
	if t, err := http.ParseTime(value); err == nil {
		d := time.Until(t)
		if d > 0 {
			return d
		}
	}

	return 0
}
