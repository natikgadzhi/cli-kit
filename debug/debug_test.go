package debug

import (
	"bytes"
	"os"
	"strings"
	"sync"
	"testing"

	"github.com/spf13/cobra"
)

func TestEnabledDefault(t *testing.T) {
	Disable()
	if Enabled() {
		t.Error("Enabled() = true, want false by default")
	}
}

func TestEnableDisableToggle(t *testing.T) {
	Disable()

	Enable()
	if !Enabled() {
		t.Error("Enabled() = false after Enable()")
	}

	Disable()
	if Enabled() {
		t.Error("Enabled() = true after Disable()")
	}
}

func TestLogWhenDisabled(t *testing.T) {
	Disable()

	// Capture stderr.
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	origStderr := os.Stderr
	os.Stderr = w

	Log("should not appear")

	w.Close()
	var buf bytes.Buffer
	buf.ReadFrom(r)
	os.Stderr = origStderr

	if buf.Len() != 0 {
		t.Errorf("Log when disabled produced output: %q", buf.String())
	}
}

func TestLogWhenEnabled(t *testing.T) {
	Disable()
	Enable()
	defer Disable()

	// Capture stderr.
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	origStderr := os.Stderr
	os.Stderr = w

	Log("hello %s", "world")

	w.Close()
	var buf bytes.Buffer
	buf.ReadFrom(r)
	os.Stderr = origStderr

	got := buf.String()
	want := "Debug: hello world\n"
	if got != want {
		t.Errorf("Log output = %q, want %q", got, want)
	}
}

func TestLogFormatting(t *testing.T) {
	Disable()
	Enable()
	defer Disable()

	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	origStderr := os.Stderr
	os.Stderr = w

	Log("count=%d name=%s", 42, "test")

	w.Close()
	var buf bytes.Buffer
	buf.ReadFrom(r)
	os.Stderr = origStderr

	got := buf.String()
	if !strings.HasPrefix(got, "Debug: ") {
		t.Errorf("Log output %q does not start with 'Debug: '", got)
	}
	if !strings.Contains(got, "count=42") {
		t.Errorf("Log output %q missing 'count=42'", got)
	}
	if !strings.Contains(got, "name=test") {
		t.Errorf("Log output %q missing 'name=test'", got)
	}
}

func TestRegisterFlagAddsDebugFlag(t *testing.T) {
	cmd := &cobra.Command{Use: "test"}
	RegisterFlag(cmd)

	f := cmd.PersistentFlags().Lookup("debug")
	if f == nil {
		t.Fatal("--debug flag not registered")
	}
	if f.DefValue != "false" {
		t.Errorf("--debug default = %q, want 'false'", f.DefValue)
	}
}

func TestRegisterFlagEnablesDebugOnExecution(t *testing.T) {
	Disable()

	cmd := &cobra.Command{
		Use: "test",
		RunE: func(cmd *cobra.Command, args []string) error {
			return nil
		},
	}
	RegisterFlag(cmd)

	cmd.SetArgs([]string{"--debug"})
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}

	if !Enabled() {
		t.Error("Enabled() = false after executing with --debug")
	}
	Disable()
}

func TestRegisterFlagChainsExistingPreRunE(t *testing.T) {
	Disable()

	existingCalled := false
	cmd := &cobra.Command{
		Use: "test",
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			existingCalled = true
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			return nil
		},
	}
	RegisterFlag(cmd)

	cmd.SetArgs([]string{"--debug"})
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}

	if !existingCalled {
		t.Error("existing PersistentPreRunE was not called")
	}
	if !Enabled() {
		t.Error("Enabled() = false after executing with --debug")
	}
	Disable()
}

func TestConcurrentAccess(t *testing.T) {
	Disable()

	var wg sync.WaitGroup
	// Run concurrent Enable/Disable/Enabled/Log calls to test thread safety.
	for i := 0; i < 100; i++ {
		wg.Add(4)
		go func() {
			defer wg.Done()
			Enable()
		}()
		go func() {
			defer wg.Done()
			Disable()
		}()
		go func() {
			defer wg.Done()
			_ = Enabled()
		}()
		go func() {
			defer wg.Done()
			Log("concurrent message %d", i)
		}()
	}
	wg.Wait()

	// If we get here without a race detector panic, concurrency is safe.
	Disable()
}
