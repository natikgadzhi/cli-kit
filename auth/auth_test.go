package auth

import (
	"errors"
	"os"
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"github.com/zalando/go-keyring"
)

func init() {
	// Replace OS keychain with in-memory mock for all tests in this package.
	keyring.MockInit()
}

func TestMain(m *testing.M) {
	os.Exit(m.Run())
}

func TestResolveToken_FlagPriority(t *testing.T) {
	// Put tokens in all sources; flag should win.
	t.Setenv("TEST_TOKEN", "env-token")
	_ = keyring.Set("test-service", "api-token", "keychain-token")

	token, source, err := ResolveToken(TokenSource{
		FlagValue:       "flag-token",
		EnvVar:          "TEST_TOKEN",
		KeychainService: "test-service",
		KeychainKey:     "api-token",
	})
	if err != nil {
		t.Fatalf("ResolveToken() error: %v", err)
	}
	if token != "flag-token" {
		t.Errorf("token = %q, want %q", token, "flag-token")
	}
	if source != SourceFlag {
		t.Errorf("source = %q, want %q", source, SourceFlag)
	}
}

func TestResolveToken_EnvFallback(t *testing.T) {
	// Token in env and keychain, but no flag.
	t.Setenv("TEST_TOKEN", "env-token")
	_ = keyring.Set("test-service", "api-token", "keychain-token")

	token, source, err := ResolveToken(TokenSource{
		FlagValue:       "",
		EnvVar:          "TEST_TOKEN",
		KeychainService: "test-service",
		KeychainKey:     "api-token",
	})
	if err != nil {
		t.Fatalf("ResolveToken() error: %v", err)
	}
	if token != "env-token" {
		t.Errorf("token = %q, want %q", token, "env-token")
	}
	if source != SourceEnvironment {
		t.Errorf("source = %q, want %q", source, SourceEnvironment)
	}
}

func TestResolveToken_KeychainFallback(t *testing.T) {
	// Token only in keychain.
	t.Setenv("TEST_TOKEN", "")
	_ = keyring.Set("test-service", "api-token", "keychain-token")

	token, source, err := ResolveToken(TokenSource{
		FlagValue:       "",
		EnvVar:          "TEST_TOKEN",
		KeychainService: "test-service",
		KeychainKey:     "api-token",
	})
	if err != nil {
		t.Fatalf("ResolveToken() error: %v", err)
	}
	if token != "keychain-token" {
		t.Errorf("token = %q, want %q", token, "keychain-token")
	}
	if source != SourceKeychain {
		t.Errorf("source = %q, want %q", source, SourceKeychain)
	}
}

func TestResolveToken_NoToken(t *testing.T) {
	// No token in any source.
	t.Setenv("TEST_TOKEN", "")
	_ = keyring.Delete("test-service", "api-token")

	_, _, err := ResolveToken(TokenSource{
		FlagValue:       "",
		EnvVar:          "TEST_TOKEN",
		KeychainService: "test-service",
		KeychainKey:     "api-token",
	})
	if err == nil {
		t.Fatal("ResolveToken() expected error, got nil")
	}
	if !errors.Is(err, ErrNoToken) {
		t.Errorf("error = %v, want ErrNoToken", err)
	}
}

func TestResolveToken_NoToken_ActionableMessage(t *testing.T) {
	// Verify the error message includes the env var name as guidance.
	t.Setenv("MY_APP_TOKEN", "")
	_ = keyring.Delete("myapp", "token")

	_, _, err := ResolveToken(TokenSource{
		EnvVar:          "MY_APP_TOKEN",
		KeychainService: "myapp",
		KeychainKey:     "token",
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	msg := err.Error()
	if !errors.Is(err, ErrNoToken) {
		t.Errorf("error should wrap ErrNoToken, got: %v", err)
	}
	// The message should mention the env var for actionable guidance.
	if got := msg; got == "" {
		t.Error("error message is empty")
	}
	wantSubstr := "MY_APP_TOKEN"
	if !strings.Contains(msg, wantSubstr) {
		t.Errorf("error message %q should contain %q", msg, wantSubstr)
	}
}

func TestResolveToken_NoEnvVarConfigured(t *testing.T) {
	// When EnvVar is empty, env step is skipped entirely.
	_ = keyring.Delete("test-service", "api-token")

	_, _, err := ResolveToken(TokenSource{
		FlagValue:       "",
		EnvVar:          "",
		KeychainService: "test-service",
		KeychainKey:     "api-token",
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, ErrNoToken) {
		t.Errorf("error = %v, want ErrNoToken", err)
	}
}

func TestStoreToken(t *testing.T) {
	err := StoreToken("test-service", "api-token", "stored-token")
	if err != nil {
		t.Fatalf("StoreToken() error: %v", err)
	}

	// Verify it can be retrieved.
	got, err := keyring.Get("test-service", "api-token")
	if err != nil {
		t.Fatalf("keyring.Get() error: %v", err)
	}
	if got != "stored-token" {
		t.Errorf("stored token = %q, want %q", got, "stored-token")
	}
}

func TestDeleteToken(t *testing.T) {
	// Store a token first.
	_ = keyring.Set("test-service", "api-token", "to-delete")

	err := DeleteToken("test-service", "api-token")
	if err != nil {
		t.Fatalf("DeleteToken() error: %v", err)
	}

	// Verify it's gone.
	_, err = keyring.Get("test-service", "api-token")
	if err == nil {
		t.Error("expected error after delete, got nil")
	}
}

func TestDeleteToken_NotFound(t *testing.T) {
	// Ensure no token exists.
	_ = keyring.Delete("test-service", "api-token")

	// Should not error when deleting a non-existent token.
	err := DeleteToken("test-service", "api-token")
	if err != nil {
		t.Fatalf("DeleteToken() error on non-existent: %v", err)
	}
}

func TestRegisterFlag(t *testing.T) {
	cmd := &cobra.Command{Use: "test"}
	RegisterFlag(cmd, "MY_TOKEN")

	flag := cmd.PersistentFlags().Lookup("token")
	if flag == nil {
		t.Fatal("expected --token flag to be registered")
	}
	if flag.DefValue != "" {
		t.Errorf("default value = %q, want empty", flag.DefValue)
	}
	// Description should mention the env var.
	if !strings.Contains(flag.Usage, "MY_TOKEN") {
		t.Errorf("flag usage %q should mention env var MY_TOKEN", flag.Usage)
	}
}

func TestRegisterFlag_NoEnvVar(t *testing.T) {
	cmd := &cobra.Command{Use: "test"}
	RegisterFlag(cmd, "")

	flag := cmd.PersistentFlags().Lookup("token")
	if flag == nil {
		t.Fatal("expected --token flag to be registered")
	}
	if flag.Usage != "API token" {
		t.Errorf("flag usage = %q, want %q", flag.Usage, "API token")
	}
}

func TestMaskToken(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"normal token", "fmu1-abcdef123456", "fmu1-abc..."},
		{"short token", "abc", "****"},
		{"exactly 8", "12345678", "****"},
		{"empty", "", "****"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := MaskToken(tt.input)
			if got != tt.want {
				t.Errorf("MaskToken(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

