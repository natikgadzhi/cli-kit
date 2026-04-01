package config

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/cobra"
)

// testConfig is a sample config struct used across tests.
type testConfig struct {
	Name    string `toml:"name"`
	Port    int    `toml:"port"`
	Verbose bool   `toml:"verbose"`
}

func TestLoad(t *testing.T) {
	t.Run("valid TOML", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "config.toml")
		content := `name = "myapp"
port = 8080
verbose = true
`
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			t.Fatal(err)
		}

		var cfg testConfig
		if err := Load(path, &cfg); err != nil {
			t.Fatalf("Load() returned error: %v", err)
		}

		if cfg.Name != "myapp" {
			t.Errorf("Name = %q, want %q", cfg.Name, "myapp")
		}
		if cfg.Port != 8080 {
			t.Errorf("Port = %d, want %d", cfg.Port, 8080)
		}
		if !cfg.Verbose {
			t.Error("Verbose = false, want true")
		}
	})

	t.Run("missing file returns wrapped os.ErrNotExist", func(t *testing.T) {
		var cfg testConfig
		err := Load("/nonexistent/path/config.toml", &cfg)
		if err == nil {
			t.Fatal("Load() returned nil error for missing file")
		}
		if !errors.Is(err, os.ErrNotExist) {
			t.Errorf("error does not wrap os.ErrNotExist: %v", err)
		}
	})

	t.Run("invalid TOML returns parse error", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "bad.toml")
		if err := os.WriteFile(path, []byte("not valid [[[toml"), 0644); err != nil {
			t.Fatal(err)
		}

		var cfg testConfig
		err := Load(path, &cfg)
		if err == nil {
			t.Fatal("Load() returned nil error for invalid TOML")
		}
		// Should not be ErrNotExist since the file exists
		if errors.Is(err, os.ErrNotExist) {
			t.Error("parse error should not wrap os.ErrNotExist")
		}
	})

	t.Run("tilde expansion in path", func(t *testing.T) {
		// Write a file to a known location under home
		home, err := os.UserHomeDir()
		if err != nil {
			t.Skip("cannot determine home directory")
		}

		dir := filepath.Join(home, ".config", "cli-kit-test-load")
		if err := os.MkdirAll(dir, 0755); err != nil {
			t.Fatal(err)
		}
		defer os.RemoveAll(dir)

		path := filepath.Join(dir, "config.toml")
		if err := os.WriteFile(path, []byte(`name = "tildetest"`), 0644); err != nil {
			t.Fatal(err)
		}

		var cfg testConfig
		if err := Load("~/.config/cli-kit-test-load/config.toml", &cfg); err != nil {
			t.Fatalf("Load() with tilde path returned error: %v", err)
		}
		if cfg.Name != "tildetest" {
			t.Errorf("Name = %q, want %q", cfg.Name, "tildetest")
		}
	})
}

func TestDefaultPath(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Skip("cannot determine home directory")
	}

	got := DefaultPath("mytool")
	want := filepath.Join(home, ".config", "mytool", "config.toml")
	if got != want {
		t.Errorf("DefaultPath(%q) = %q, want %q", "mytool", got, want)
	}
}

func TestExpandTilde(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Skip("cannot determine home directory")
	}

	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "bare tilde",
			input: "~",
			want:  home,
		},
		{
			name:  "tilde with path",
			input: "~/Documents/config.toml",
			want:  filepath.Join(home, "Documents", "config.toml"),
		},
		{
			name:  "absolute path unchanged",
			input: "/etc/config.toml",
			want:  "/etc/config.toml",
		},
		{
			name:  "relative path unchanged",
			input: "configs/local.toml",
			want:  "configs/local.toml",
		},
		{
			name:  "tilde in middle unchanged",
			input: "/home/~user/file",
			want:  "/home/~user/file",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ExpandTilde(tt.input)
			if err != nil {
				t.Fatalf("ExpandTilde(%q) returned error: %v", tt.input, err)
			}
			if got != tt.want {
				t.Errorf("ExpandTilde(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestRegisterFlag(t *testing.T) {
	cmd := &cobra.Command{Use: "test"}
	RegisterFlag(cmd, "mytool")

	flag := cmd.PersistentFlags().Lookup("config")
	if flag == nil {
		t.Fatal("RegisterFlag() did not add --config flag")
	}

	want := DefaultPath("mytool")
	if flag.DefValue != want {
		t.Errorf("--config default = %q, want %q", flag.DefValue, want)
	}
}
