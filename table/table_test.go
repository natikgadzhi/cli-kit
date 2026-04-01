package table

import (
	"bytes"
	"strings"
	"testing"
	"unicode/utf8"
)

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
	if !strings.HasPrefix(lines[0], "┌") || !strings.HasSuffix(lines[0], "┐") {
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
	if !strings.HasPrefix(lines[5], "└") || !strings.HasSuffix(lines[5], "┘") {
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
		w := utf8.RuneCountInString(line)
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
		w := utf8.RuneCountInString(line)
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
