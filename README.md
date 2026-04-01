# cli-kit

Shared Go packages for building consistent, well-behaved CLI tools.

## Installation

```
go get github.com/natikgadzhi/cli-kit
```

## Packages

### output

Output format flag with TTY auto-detection. Registers `-o/--output` on a Cobra command and resolves to `"json"` or `"table"` based on whether stdout is a terminal.

```go
output.RegisterFlag(rootCmd)

format := output.Resolve(cmd)
if output.IsJSON(format) {
    output.PrintJSON(data)
} else {
    output.Print(format, data, myRenderer)
}
```

Types that implement `output.TableRenderer` can render themselves as bordered tables.

### table

Bordered tables with box-drawing characters that automatically shrink columns to fit the terminal width. Overflowing values are truncated with an ellipsis.

```go
t := table.New()
t.Header("Name", "Status", "Description")
t.Row("alpha", "active", "First item")
t.Row("beta", "inactive", "Second item")
t.Flush()
```

Use `table.NewWriter(w)` to write to a custom `io.Writer` instead of stdout.

### progress

Spinner and counter indicators that write to stderr. Automatically suppressed in JSON output mode.

```go
// Auto-ticking spinner
sp := progress.NewSpinner("Loading...", format)
sp.Start()
defer sp.Finish()

// Manual counter
c := progress.NewCounter("Fetching items", format)
defer c.Finish()
c.Update(42)
```

Both types implement the `progress.Indicator` interface and are safe for concurrent use.

### errors

Structured CLI errors with exit codes, HTTP error classification, and partial-result support.

```go
err := errors.NewCLIError(errors.ExitError, "something went wrong").
    WithSuggestion("Try again with --force")
errors.PrintError(err, jsonFormat)

// Wrap an underlying error with user-facing context
wrapped := errors.Wrap(err, "upload failed", "Check your network connection")

// HTTP status → CLIError with auth verification
cliErr := errors.HandleHTTPError(statusCode, "repos", "mytool", authChecker)

// Exit the process with the right code
errors.ExitWithError(err)
```

`PartialResult[T]` wraps results that may be incomplete due to rate-limiting or transient errors.

### derived

Derived data directory management and Markdown frontmatter utilities. Stores cached data as Markdown files with YAML frontmatter under `~/.local/share/lambdal/derived/{tool}/`.

```go
derived.RegisterFlag(rootCmd, "mytool")
dir := derived.Resolve(cmd, "mytool")
derived.EnsureDir(dir)

fm := derived.NewFrontmatter("mytool", "issue", "123", "https://...", "mytool issues get 123")
content := derived.FormatFile(fm, "# Issue 123\n...")
```

Includes `Parse` and `Render` for round-tripping frontmatter, and `ContentHash` for SHA-256 content fingerprinting.

### ratelimit

HTTP retry transport with exponential backoff, jitter, and Retry-After header parsing. Retries on 429 and 5xx responses.

```go
client := &http.Client{
    Transport: ratelimit.NewRetryTransport(nil),
}
```

Configure retries, delays, and an optional `OnRetry` callback:

```go
rt := &ratelimit.RetryTransport{
    Base:       http.DefaultTransport,
    MaxRetries: 3,
    BaseDelay:  2 * time.Second,
    RetryOn5xx: true,
    OnRetry: func(attempt int, delay time.Duration, status int) {
        log.Printf("retry #%d after %s (HTTP %d)", attempt, delay, status)
    },
}
```

### version

Standard `version` subcommand and `--version` flag. Version info is set via ldflags at build time and printed as JSON.

```go
info := &version.Info{Version: "1.0.0", Commit: "abc123", Date: "2025-01-01"}
rootCmd.AddCommand(version.NewCommand(info))
version.SetupFlag(rootCmd, info)
```

### debug

Debug logger that writes to stderr when `--debug` is passed. Thread-safe.

```go
debug.RegisterFlag(rootCmd)

debug.Log("fetching page %d", page)
```

### config

TOML config file loading with tilde expansion. Looks for `~/.config/{tool}/config.toml` by default.

```go
type Config struct {
    Token string `toml:"token"`
    Org   string `toml:"org"`
}

config.RegisterFlag(rootCmd, "mytool")

var cfg Config
err := config.Load(config.DefaultPath("mytool"), &cfg)
```

### auth

Token masking for safe display in terminal output.

```go
auth.MaskToken("ghp_abc123xyz789") // "ghp_•••z789"
auth.MaskToken("")                 // "(none)"
```

## CLI Standards

These packages implement the conventions defined in [CLI_STANDARDS.md](https://github.com/natikgadzhi/cli-tool-template/blob/main/CLI_STANDARDS.md) in the template repo.
