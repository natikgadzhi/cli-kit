# cli-kit

Shared Go packages for building consistent, well-behaved CLI tools. Implements the conventions defined in [CLI_STANDARDS.md](https://github.com/natikgadzhi/template/blob/main/CLI_STANDARDS.md).

## Requirement

**If your Go CLI tool uses cli-kit, it must use cli-kit for all features that cli-kit covers.** Do not reimplement table rendering, progress indicators, error handling, retry logic, config loading, auth resolution, or debug logging locally when cli-kit provides them. This keeps tools consistent and avoids drift.

If cli-kit doesn't cover a feature you need, either extend cli-kit or discuss adding it.

## Installation

```
go get github.com/natikgadzhi/cli-kit
```

## Tools using cli-kit

| Tool | Description | Repo |
|------|-------------|------|
| [fm](https://github.com/natikgadzhi/fm) | FastMail CLI | `github.com/natikgadzhi/fm` |
| [gdrive-cli](https://github.com/natikgadzhi/gdrive-cli) | Google Drive CLI | `github.com/natikgadzhi/gdrive-cli` |
| [slack-cli](https://github.com/natikgadzhi/slack-cli) | Slack CLI | `github.com/natikgadzhi/slack-cli` |
| [claude-utils](https://github.com/natikgadzhi/claude-utils) | Obsidian vault sync | `github.com/natikgadzhi/claude-utils` |

## Packages

### table — Bordered tables with terminal-width adaptation

Renders tables with rounded corners, bold headers, and automatic column shrinking. Truncates overflowing values with `…`.

```go
import "github.com/natikgadzhi/cli-kit/table"

t := table.New()
t.Header("Name", "Status", "Last Sync")
t.Row("weekly-review", "0 conflicts", "31 Mar 2026 15:04")
t.Flush()
```

Output:
```
╭────────────────┬──────────────┬───────────────────╮
│ NAME           │ STATUS       │ LAST SYNC         │
├────────────────┼──────────────┼───────────────────┤
│ weekly-review  │ 0 conflicts  │ 31 Mar 2026 15:04 │
╰────────────────┴──────────────┴───────────────────╯
```

Use `table.NewWriter(w)` to write to a custom `io.Writer`. Use `table.Truncate(s, n)` for standalone truncation.

**Source:** [`table/table.go`](table/table.go)

### output — Output format flag and TTY detection

Registers `-o/--output` flag on a Cobra command. Auto-detects TTY: table for interactive terminals, JSON for pipes.

```go
import "github.com/natikgadzhi/cli-kit/output"

output.RegisterFlag(rootCmd)

format := output.Resolve(cmd)        // "json" or "table"
if output.IsJSON(format) {
    output.PrintJSON(data)
} else {
    t := table.New()
    // ... render table
}
```

**Source:** [`output/output.go`](output/output.go)

### progress — Spinners and counters

Auto-ticking spinner and manual counter. Suppressed in JSON mode. Thread-safe.

```go
import "github.com/natikgadzhi/cli-kit/progress"

// Spinner: auto-animates in a goroutine
sp := progress.NewSpinner("Searching...", format)
sp.Start()
sp.SetLabel("Rate limited, retrying in 5s...")  // update mid-operation
sp.Finish()                                      // clears the line

// Counter: caller updates the count
c := progress.NewCounter("Fetching emails", format)
c.Update(42)
c.Finish()
```

**Source:** [`progress/progress.go`](progress/progress.go)

### ratelimit — HTTP retry transport

`http.RoundTripper` with exponential backoff, ±25% jitter, and `Retry-After` header parsing. Retries on HTTP 429 and 5xx.

```go
import "github.com/natikgadzhi/cli-kit/ratelimit"

client := &http.Client{
    Transport: ratelimit.NewRetryTransport(http.DefaultTransport),
}

// Or configure:
rt := &ratelimit.RetryTransport{
    Base:       http.DefaultTransport,
    MaxRetries: 3,
    BaseDelay:  2 * time.Second,
    RetryOn5xx: true,
    OnRetry: func(attempt int, delay time.Duration, status int) {
        spinner.SetLabel(fmt.Sprintf("Rate limited, retry %d...", attempt))
    },
}
```

**Source:** [`ratelimit/transport.go`](ratelimit/transport.go), [`ratelimit/retry_after.go`](ratelimit/retry_after.go)

### errors — Structured CLI errors

Exit codes, HTTP error classification, partial results, and `ExitWithError` for the standard `Execute()` pattern.

```go
import "github.com/natikgadzhi/cli-kit/errors"

// Standard root command pattern:
func Execute() {
    if err := rootCmd.Execute(); err != nil {
        errors.ExitWithError(err)  // prints error + suggestion, exits with code
    }
}

// Wrap errors with user-facing context:
errors.Wrap(err, "upload failed", "Check your network connection")
errors.WrapAuth(err, "token expired", "Run 'mytool auth login'")

// HTTP status → CLIError:
errors.HandleHTTPError(resp.StatusCode, "/api/users", "mytool", authChecker)
```

Exit codes: `ExitSuccess` (0), `ExitError` (1), `ExitAuthError` (2).

**Source:** [`errors/errors.go`](errors/errors.go)

### derived — Derived data directory and frontmatter

Manages `~/.local/share/lambdal/derived/{tool}/` for cached markdown files with YAML frontmatter.

```go
import "github.com/natikgadzhi/cli-kit/derived"

// Write a derived file:
fm := derived.NewFrontmatter("mytool", "issue", "123", "https://...", "mytool issues get 123")
content := derived.FormatFile(fm, body)

// Parse a file with frontmatter:
meta, body, err := derived.Parse(fileContent)

// Render modified frontmatter back:
output := derived.Render(meta, body)

// Content fingerprint (SHA-256, whitespace-trimmed):
hash := derived.ContentHash(body)  // "sha256:abc123..."
```

**Source:** [`derived/derived.go`](derived/derived.go), [`derived/parse.go`](derived/parse.go)

### config — TOML config loading

Loads `~/.config/{tool}/config.toml` with tilde expansion. Registers `--config` flag.

```go
import "github.com/natikgadzhi/cli-kit/config"

type MyConfig struct {
    Vault   string `toml:"vault"`
    Debug   bool   `toml:"debug"`
}

config.RegisterFlag(rootCmd, "mytool")

var cfg MyConfig
err := config.Load(config.DefaultPath("mytool"), &cfg)
```

**Source:** [`config/config.go`](config/config.go)

### auth — Token resolution and masking

Three-source token resolution: CLI flag → env var → OS keychain. Cross-platform keychain via `go-keyring`.

```go
import "github.com/natikgadzhi/cli-kit/auth"

// Resolve token from highest-priority source:
token, source, err := auth.ResolveToken(auth.TokenSource{
    FlagValue:       flagVal,
    EnvVar:          "MYTOOL_TOKEN",
    KeychainService: "mytool",
    KeychainKey:     "api-token",
})

// Store/delete in keychain:
auth.StoreToken("mytool", "api-token", token)
auth.DeleteToken("mytool", "api-token")

// Safe display:
auth.MaskToken("ghp_abc123xyz789")  // "ghp_•••z789"

// Register --token flag:
auth.RegisterFlag(rootCmd, "MYTOOL_TOKEN")
```

**Source:** [`auth/auth.go`](auth/auth.go)

### debug — Standardized debug logging

Writes `Debug: <message>` to stderr when `--debug` is passed. Thread-safe.

```go
import "github.com/natikgadzhi/cli-kit/debug"

debug.RegisterFlag(rootCmd)

debug.Log("fetching page %d of %d", page, total)
debug.Log("rate limited, waiting %s", delay)
```

**Source:** [`debug/debug.go`](debug/debug.go)

### version — Version command and flag

Standard `version` subcommand and `--version` flag. Outputs JSON with version, commit, and build date.

```go
import "github.com/natikgadzhi/cli-kit/version"

info := &version.Info{Version: Version, Commit: Commit, Date: Date}
rootCmd.AddCommand(version.NewCommand(info))
version.SetupFlag(rootCmd, info)
```

**Source:** [`version/version.go`](version/version.go)
