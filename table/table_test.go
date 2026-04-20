package table

import (
	"bytes"
	"regexp"
	"strings"
	"testing"
	"unicode/utf8"
)

var ansiRegex = regexp.MustCompile(`\x1b\[[0-9;]*m`)

func displayWidth(s string) int {
	// Use visibleLen for lines that may contain OSC 8 hyperlinks too.
	return visibleLen(s)
}

func displayWidthLegacy(s string) int {
	return utf8.RuneCountInString(ansiRegex.ReplaceAllString(s, ""))
}

func TestBasicRendering(t *testing.T) {
	var buf bytes.Buffer
	tbl := NewWriter(&buf)
	tbl.Header("Name", "Status")
	tbl.Row("alpha", "active")
	tbl.Row("beta", "inactive")
	tbl.Flush()

	out := buf.String()
	lines := strings.Split(strings.TrimRight(out, "\n"), "\n")

	// Expect 6 lines: top border, header, separator, 2 data rows, bottom border.
	if len(lines) != 6 {
		t.Fatalf("expected 6 lines, got %d:\n%s", len(lines), out)
	}

	// Top border uses box-drawing characters.
	if !strings.HasPrefix(lines[0], "╭") || !strings.HasSuffix(lines[0], "╮") {
		t.Errorf("top border malformed: %q", lines[0])
	}

	// Header row should contain uppercased headers.
	if !strings.Contains(lines[1], "NAME") || !strings.Contains(lines[1], "STATUS") {
		t.Errorf("header row missing uppercased headers: %q", lines[1])
	}

	// Separator between header and data.
	if !strings.HasPrefix(lines[2], "├") || !strings.HasSuffix(lines[2], "┤") {
		t.Errorf("separator malformed: %q", lines[2])
	}

	// Data rows.
	if !strings.Contains(lines[3], "alpha") || !strings.Contains(lines[3], "active") {
		t.Errorf("first data row missing values: %q", lines[3])
	}
	if !strings.Contains(lines[4], "beta") || !strings.Contains(lines[4], "inactive") {
		t.Errorf("second data row missing values: %q", lines[4])
	}

	// Bottom border.
	if !strings.HasPrefix(lines[5], "╰") || !strings.HasSuffix(lines[5], "╯") {
		t.Errorf("bottom border malformed: %q", lines[5])
	}

	// All data rows should start and end with │.
	for _, i := range []int{1, 3, 4} {
		if !strings.HasPrefix(lines[i], "│") || !strings.HasSuffix(lines[i], "│") {
			t.Errorf("row %d should be bordered with │: %q", i, lines[i])
		}
	}
}

func TestHeadersUppercased(t *testing.T) {
	var buf bytes.Buffer
	tbl := NewWriter(&buf)
	tbl.Header("first name", "last name")
	tbl.Row("John", "Doe")
	tbl.Flush()

	out := buf.String()
	if !strings.Contains(out, "FIRST NAME") {
		t.Error("expected 'FIRST NAME' in output")
	}
	if !strings.Contains(out, "LAST NAME") {
		t.Error("expected 'LAST NAME' in output")
	}
}

func TestColumnShrinking(t *testing.T) {
	var buf bytes.Buffer
	tbl := NewWriter(&buf)
	// Override the terminal width function to simulate a narrow terminal.
	tbl.termWidthFunc = func() int { return 30 }

	tbl.Header("Name", "Description")
	tbl.Row("a", "This is a very long description that should be truncated")
	tbl.Flush()

	out := buf.String()
	lines := strings.Split(strings.TrimRight(out, "\n"), "\n")

	// Every line should be at most 30 display columns wide.
	for i, line := range lines {
		w := displayWidth(line)
		if w > 30 {
			t.Errorf("line %d exceeds terminal width 30: display_width=%d %q", i, w, line)
		}
	}

	// The long description should have been truncated (contains ellipsis).
	if !strings.Contains(out, "…") {
		t.Error("expected truncation ellipsis in output")
	}
}

func TestColumnShrinkingMultipleColumns(t *testing.T) {
	var buf bytes.Buffer
	tbl := NewWriter(&buf)
	tbl.termWidthFunc = func() int { return 40 }

	tbl.Header("A", "B", "C")
	tbl.Row("short", strings.Repeat("x", 50), strings.Repeat("y", 50))
	tbl.Flush()

	out := buf.String()
	lines := strings.Split(strings.TrimRight(out, "\n"), "\n")

	for i, line := range lines {
		w := displayWidth(line)
		if w > 40 {
			t.Errorf("line %d exceeds terminal width 40: display_width=%d %q", i, w, line)
		}
	}
}

func TestTruncateWithEllipsis(t *testing.T) {
	tests := []struct {
		input  string
		maxLen int
		want   string
	}{
		{"hello", 10, "hello"},       // Fits, no truncation.
		{"hello", 5, "hello"},        // Exact fit.
		{"hello world", 5, "hell…"},  // Truncated with ellipsis.
		{"hello", 3, "he…"},          // Short truncation.
		{"hello", 1, "…"},            // Very short: just ellipsis.
		{"hello", 0, "…"},            // Zero width: just ellipsis.
		{"", 5, ""},                   // Empty string.
		{"abcdef", 4, "abc…"},        // Standard truncation.
	}

	for _, tt := range tests {
		got := Truncate(tt.input, tt.maxLen)
		if got != tt.want {
			t.Errorf("Truncate(%q, %d) = %q, want %q", tt.input, tt.maxLen, got, tt.want)
		}
	}
}

func TestEmptyTable(t *testing.T) {
	var buf bytes.Buffer
	tbl := NewWriter(&buf)
	tbl.Flush()

	out := buf.String()
	if out != "" {
		t.Errorf("expected empty output for table with no headers, got %q", out)
	}
}

func TestEmptyTableWithHeaders(t *testing.T) {
	var buf bytes.Buffer
	tbl := NewWriter(&buf)
	tbl.Header("Name", "Value")
	tbl.Flush()

	out := buf.String()
	lines := strings.Split(strings.TrimRight(out, "\n"), "\n")

	// Expect 4 lines: top border, header, separator, bottom border (no data rows).
	if len(lines) != 4 {
		t.Fatalf("expected 4 lines for headers-only table, got %d:\n%s", len(lines), out)
	}

	if !strings.Contains(lines[1], "NAME") {
		t.Error("expected header row with NAME")
	}
}

func TestNewWriterCustomWriter(t *testing.T) {
	var buf bytes.Buffer
	tbl := NewWriter(&buf)
	tbl.Header("Col")
	tbl.Row("val")
	tbl.Flush()

	out := buf.String()
	if !strings.Contains(out, "COL") {
		t.Error("expected header in custom writer output")
	}
	if !strings.Contains(out, "val") {
		t.Error("expected row in custom writer output")
	}
}

func TestNoTerminalWidth(t *testing.T) {
	var buf bytes.Buffer
	tbl := NewWriter(&buf)
	// Simulate no terminal (returns 0).
	tbl.termWidthFunc = func() int { return 0 }

	tbl.Header("Name", "Description")
	longDesc := strings.Repeat("x", 200)
	tbl.Row("test", longDesc)
	tbl.Flush()

	out := buf.String()
	// When no terminal width is detected, no shrinking should occur.
	if !strings.Contains(out, longDesc) {
		t.Error("expected full long description when terminal width is 0 (no shrinking)")
	}
}

func TestNegativeTerminalWidth(t *testing.T) {
	var buf bytes.Buffer
	tbl := NewWriter(&buf)
	tbl.termWidthFunc = func() int { return -1 }

	tbl.Header("A")
	tbl.Row("value")
	tbl.Flush()

	out := buf.String()
	if !strings.Contains(out, "value") {
		t.Error("expected full value when terminal width is negative")
	}
}

func TestTableWidth(t *testing.T) {
	tbl := &Table{
		headers:       []string{"AB", "CD"},
		widths:        []int{2, 2},
		termWidthFunc: func() int { return 0 },
	}

	// │ AB │ CD │ = 1 + (2+2+1) + (2+2+1) = 1 + 5 + 5 = 11
	got := tbl.tableWidth()
	want := 11
	if got != want {
		t.Errorf("tableWidth() = %d, want %d", got, want)
	}
}

func TestRowWithFewerColumns(t *testing.T) {
	var buf bytes.Buffer
	tbl := NewWriter(&buf)
	tbl.Header("A", "B", "C")
	tbl.Row("only-one")
	tbl.Flush()

	out := buf.String()
	if !strings.Contains(out, "only-one") {
		t.Error("expected partial row to render")
	}
	// Should not panic with fewer values than headers.
}

func TestShrinkStopsAtMinWidth(t *testing.T) {
	var buf bytes.Buffer
	tbl := NewWriter(&buf)
	// Extremely narrow terminal.
	tbl.termWidthFunc = func() int { return 10 }

	tbl.Header("ABCD", "EFGH")
	tbl.Row(strings.Repeat("x", 100), strings.Repeat("y", 100))
	tbl.Flush()

	out := buf.String()
	// Table should still render without panicking.
	if len(out) == 0 {
		t.Error("expected some output even with extremely narrow terminal")
	}

	// Columns should not shrink below header width (or 4, whichever is larger).
	for _, w := range tbl.widths {
		if w < 4 {
			t.Errorf("column width %d is below minimum of 4", w)
		}
	}
}

func TestVisibleLen(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  int
	}{
		{"plain text", "hello", 5},
		{"empty", "", 0},
		{"bold", "\033[1mhello\033[0m", 5},
		{"color", "\033[31mred text\033[0m", 8},
		{"multiple SGR", "\033[1;31mbold red\033[0m", 8},
		{"OSC 8 hyperlink", "\033]8;;https://example.com\033\\click here\033]8;;\033\\", 10},
		{"nested: bold inside hyperlink", "\033]8;;https://x.com\033\\\033[1mBold Link\033[0m\033]8;;\033\\", 9},
		{"mixed plain and ANSI", "pre\033[1mbold\033[0mpost", 11},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := visibleLen(tt.input)
			if got != tt.want {
				t.Errorf("visibleLen(%q) = %d, want %d", tt.input, got, tt.want)
			}
		})
	}
}

func TestStripANSI(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"plain", "hello", "hello"},
		{"bold", "\033[1mhello\033[0m", "hello"},
		{"hyperlink", "\033]8;;https://example.com\033\\click\033]8;;\033\\", "click"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := stripANSI(tt.input)
			if got != tt.want {
				t.Errorf("stripANSI(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestHyperlink(t *testing.T) {
	got := Hyperlink("https://example.com", "click")
	want := "\033]8;;https://example.com\033\\click\033]8;;\033\\"
	if got != want {
		t.Errorf("Hyperlink() = %q, want %q", got, want)
	}
	// Visible length should be just the text.
	if vl := visibleLen(got); vl != 5 {
		t.Errorf("visibleLen(Hyperlink()) = %d, want 5", vl)
	}
}

func TestTruncateWithANSI(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		maxLen int
		want   string
	}{
		{
			"ANSI fits, returned as-is",
			"\033[1mhello\033[0m",
			5,
			"\033[1mhello\033[0m",
		},
		{
			"ANSI too long, stripped and truncated",
			"\033[1mhello world\033[0m",
			5,
			"hell…",
		},
		{
			"hyperlink fits",
			Hyperlink("https://example.com", "hi"),
			5,
			Hyperlink("https://example.com", "hi"),
		},
		{
			"hyperlink too long, stripped and truncated",
			Hyperlink("https://example.com", "click here now"),
			5,
			"clic…",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Truncate(tt.input, tt.maxLen)
			if got != tt.want {
				t.Errorf("Truncate(%q, %d) = %q, want %q", tt.input, tt.maxLen, got, tt.want)
			}
		})
	}
}

func TestTableWithHyperlinks(t *testing.T) {
	var buf bytes.Buffer
	tbl := NewWriter(&buf)
	tbl.termWidthFunc = func() int { return 0 } // no shrinking

	link := Hyperlink("https://example.com", "Example")
	tbl.Header("Name", "Link")
	tbl.Row("alpha", link)
	tbl.Row("beta", "plain text")
	tbl.Flush()

	out := buf.String()
	lines := strings.Split(strings.TrimRight(out, "\n"), "\n")

	// All lines should have the same visible width (columns aligned).
	widths := make([]int, len(lines))
	for i, line := range lines {
		widths[i] = visibleLen(line)
	}
	for i := 1; i < len(widths); i++ {
		if widths[i] != widths[0] {
			t.Errorf("line %d visible width %d != line 0 visible width %d\nline 0: %q\nline %d: %q",
				i, widths[i], widths[0], lines[0], i, lines[i])
		}
	}

	// The hyperlink content should be present.
	if !strings.Contains(out, "Example") {
		t.Error("expected hyperlink text 'Example' in output")
	}
}

func TestWrapColumns_BreaksLongText(t *testing.T) {
	var buf bytes.Buffer
	tbl := NewWriter(&buf)
	tbl.termWidthFunc = func() int { return 0 }

	tbl.Header("Name", "Detail")
	tbl.WrapColumns(1)
	tbl.Row("repo-a", "short")
	tbl.Row("repo-b", "this is a much longer message that should be wrapped across several lines in the detail column")
	tbl.Flush()

	out := buf.String()

	// No ellipsis should appear — wrap replaces truncation for this column.
	if strings.Contains(out, "…") {
		t.Errorf("wrapped column should not truncate with ellipsis:\n%s", out)
	}

	// Short rows still render as a single visual line between borders.
	if !strings.Contains(out, "short") {
		t.Errorf("expected short value to render:\n%s", out)
	}

	// Every line must be the same visible width (table alignment holds).
	lines := strings.Split(strings.TrimRight(out, "\n"), "\n")
	base := visibleLen(lines[0])
	for i, line := range lines {
		if visibleLen(line) != base {
			t.Errorf("line %d width %d != expected %d: %q", i, visibleLen(line), base, line)
		}
	}

	// The wrapped message must appear in full (joining the wrapped fragments
	// with a single space reproduces the original words in order).
	joined := strings.Join(strings.Fields(stripANSI(out)), " ")
	if !strings.Contains(joined, "this is a much longer message that should be wrapped across several lines in the detail column") {
		t.Errorf("wrapped content is missing or reordered:\n%s", out)
	}
}

func TestWrapColumns_PreservesNewlines(t *testing.T) {
	var buf bytes.Buffer
	tbl := NewWriter(&buf)
	tbl.termWidthFunc = func() int { return 0 }

	tbl.Header("Col")
	tbl.WrapColumns(0)
	tbl.Row("line one\nline two\nline three")
	tbl.Flush()

	out := buf.String()
	// Count visible rows between the header separator and the bottom border.
	lines := strings.Split(strings.TrimRight(out, "\n"), "\n")
	// top border, header, sep, 3 wrapped data lines, bottom border = 7.
	if len(lines) != 7 {
		t.Errorf("expected 7 lines for 3 hard-broken data lines, got %d:\n%s", len(lines), out)
	}
}

func TestWrapColumns_HardBreaksOverlongWord(t *testing.T) {
	var buf bytes.Buffer
	tbl := NewWriter(&buf)
	tbl.termWidthFunc = func() int { return 20 }

	tbl.Header("A", "B")
	tbl.WrapColumns(1)
	// A single 40-char token with no spaces — must hard-break.
	tbl.Row("x", strings.Repeat("z", 40))
	tbl.Flush()

	out := buf.String()
	lines := strings.Split(strings.TrimRight(out, "\n"), "\n")
	for i, line := range lines {
		if visibleLen(line) > 20 {
			t.Errorf("line %d exceeds terminal width 20: %d %q", i, visibleLen(line), line)
		}
	}
	if strings.Contains(out, "…") {
		t.Errorf("wrap column should not ellipsize:\n%s", out)
	}
}

func TestWrapColumns_OtherColumnsPadContinuationLines(t *testing.T) {
	var buf bytes.Buffer
	tbl := NewWriter(&buf)
	tbl.termWidthFunc = func() int { return 0 }

	tbl.Header("Name", "Detail")
	tbl.WrapColumns(1)
	tbl.Row("repo-a", "one two three four five six seven eight nine ten")
	tbl.Flush()

	out := buf.String()
	lines := strings.Split(strings.TrimRight(out, "\n"), "\n")

	// repo-a appears exactly once — continuation lines leave Name empty.
	count := strings.Count(out, "repo-a")
	if count != 1 {
		t.Errorf("expected repo-a to appear once (only on the first visual line), got %d:\n%s", count, out)
	}

	// All data lines share the same visible width.
	base := visibleLen(lines[0])
	for i, line := range lines {
		if visibleLen(line) != base {
			t.Errorf("line %d misaligned: width=%d want=%d %q", i, visibleLen(line), base, line)
		}
	}
}

func TestTableWithHyperlinksShrinking(t *testing.T) {
	var buf bytes.Buffer
	tbl := NewWriter(&buf)
	tbl.termWidthFunc = func() int { return 30 }

	link := Hyperlink("https://example.com", "A very long hyperlink text here")
	tbl.Header("Name", "Link")
	tbl.Row("a", link)
	tbl.Flush()

	out := buf.String()
	lines := strings.Split(strings.TrimRight(out, "\n"), "\n")

	for i, line := range lines {
		w := visibleLen(line)
		if w > 30 {
			t.Errorf("line %d exceeds terminal width 30: visible_width=%d %q", i, w, line)
		}
	}
}
