// Package auth provides shared authentication utilities for CLI tools.
package auth

// MaskToken returns a masked version of a token for safe display.
// Shows first 4 and last 4 characters with "•••" in between.
// Tokens shorter than 12 chars show first 2 and last 2.
// Empty tokens return "(none)".
func MaskToken(token string) string {
	if token == "" {
		return "(none)"
	}
	if len(token) < 12 {
		if len(token) <= 4 {
			return "••••"
		}
		return token[:2] + "•••" + token[len(token)-2:]
	}
	return token[:4] + "•••" + token[len(token)-4:]
}
