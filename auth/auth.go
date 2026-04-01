// Package auth provides a token resolution chain for CLI tools.
//
// Tokens are resolved in priority order: CLI flag > environment variable > OS keychain.
// This implements the standard auth pattern described in CLI_STANDARDS.md section 3.
package auth

import (
	"errors"
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/zalando/go-keyring"
)

// Source constants identify where a token was resolved from.
const (
	SourceFlag        = "flag"
	SourceEnvironment = "environment"
	SourceKeychain    = "keychain"
)

// ErrNoToken is returned when no token is found in any source.
var ErrNoToken = fmt.Errorf("no authentication token found")

// TokenSource defines where to look for an auth token.
type TokenSource struct {
	FlagValue       string // value from --token flag (highest priority)
	EnvVar          string // environment variable name (e.g., "FM_API_TOKEN")
	KeychainService string // OS keychain service name
	KeychainKey     string // OS keychain account/key
}

// ResolveToken returns the token from the highest-priority available source.
// Priority: flag value > env var > OS keychain.
//
// Returns the token, its source identifier, and any error.
// Returns ErrNoToken if no source has a token.
func ResolveToken(src TokenSource) (token string, source string, err error) {
	// 1. Flag value has highest priority.
	if src.FlagValue != "" {
		return src.FlagValue, SourceFlag, nil
	}

	// 2. Environment variable.
	if src.EnvVar != "" {
		if v := os.Getenv(src.EnvVar); v != "" {
			return v, SourceEnvironment, nil
		}
	}

	// 3. OS keychain.
	if src.KeychainService != "" && src.KeychainKey != "" {
		t, err := keyring.Get(src.KeychainService, src.KeychainKey)
		if err == nil && t != "" {
			return t, SourceKeychain, nil
		}
		// If the keychain returned an error other than "not found", it may be
		// inaccessible (e.g. locked, unavailable in CI). Wrap and return that.
		if err != nil && !errors.Is(err, keyring.ErrNotFound) {
			return "", "", fmt.Errorf("could not access system keychain: %w", err)
		}
	}

	// Build an actionable error message.
	msg := "no authentication token found"
	if src.EnvVar != "" {
		msg += fmt.Sprintf(". Set %s, pass --token, or run your tool's auth login command", src.EnvVar)
	} else {
		msg += ". Pass --token or run your tool's auth login command"
	}
	return "", "", fmt.Errorf("%s: %w", msg, ErrNoToken)
}

// StoreToken saves a token to the OS keychain.
func StoreToken(service, key, token string) error {
	return keyring.Set(service, key, token)
}

// DeleteToken removes a token from the OS keychain.
// If the token does not exist, DeleteToken returns nil.
func DeleteToken(service, key string) error {
	err := keyring.Delete(service, key)
	if err != nil {
		if errors.Is(err, keyring.ErrNotFound) {
			return nil
		}
		return err
	}
	return nil
}

// RegisterFlag adds --token as a persistent string flag on the given command.
// The flag description mentions the env var as a fallback.
func RegisterFlag(cmd *cobra.Command, envVar string) {
	desc := "API token"
	if envVar != "" {
		desc += fmt.Sprintf(" (or set %s env var)", envVar)
	}
	cmd.PersistentFlags().String("token", "", desc)
}

// MaskToken returns a masked version of the token for display.
// Shows the first 8 characters followed by "...".
// Tokens of 8 characters or fewer are fully masked as "****".
func MaskToken(token string) string {
	if len(token) <= 8 {
		return "****"
	}
	return token[:8] + "..."
}
