// Package table renders bordered tables with box-drawing characters.
//
// Tables automatically adapt to terminal width by iteratively shrinking
// the widest columns. Values that exceed the column width are truncated
// with an ellipsis ("…") by default, or word-wrapped across multiple
// visual lines when the column is marked via WrapColumns.
//
// Usage:
//
//	t := table.New()
//	t.Header("Name", "Status", "Description")
//	t.WrapColumns(2) // wrap the Description column instead of truncating
//	t.Row("alpha", "active", "First item")
//	t.Row("beta", "inactive", "Second item")
//	t.Flush()
package table

import (
	"fmt"
	"io"
	"os"
	"strings"
	"unicode/utf8"

	"golang.org/x/term"
)

// visibleLen returns the number of visible characters (runes) in s,
// ignoring ANSI CSI sequences (e.g. \033[1m) and OSC 8 hyperlinks
// (e.g. \033]8;params;uri\033\).
func visibleLen(s string) int {
	return len([]rune(stripANSI(s)))
}

// stripANSI removes all ANSI CSI sequences and OSC sequences from s,
// returning only the visible text.
func stripANSI(s string) string {
	var b strings.Builder
	b.Grow(len(s))
	i := 0
	for i < len(s) {
		if s[i] == '\033' && i+1 < len(s) {
			if s[i+1] == '[' {
				j := i + 2
				for j < len(s) && (s[j] < '@' || s[j] > '~') {
					j++
				}
				if j < len(s) {
					j++
				}
				i = j
				continue
			}
			if s[i+1] == ']' {
				j := i + 2
				for j < len(s) {
					if s[j] == '\a' {
						j++
						break
					}
					if s[j] == '\033' && j+1 < len(s) && s[j+1] == '\\' {
						j += 2
						break
					}
					j++
				}
				i = j
				continue
			}
		}
		b.WriteByte(s[i])
		i++
	}
	return b.String()
}

// Hyperlink returns an OSC 8 hyperlink sequence that renders text as
// a clickable link to url in supporting terminals.
func Hyperlink(url, text string) string {
	return "\033]8;;" + url + "\033\\" + text + "\033]8;;\033\\"
}

// Table renders a bordered ASCII table to an io.Writer.
type Table struct {
	out     io.Writer
	headers []string
	rows    [][]string
	widths  []int

	// RowBorders adds a separator line between each data row.
	RowBorders bool

	// wrapColumns holds the set of column indices marked for word-wrapping.
	wrapColumns map[int]bool

	// termWidthFunc is used to determine the terminal width.
	// It can be overridden for testing.
	termWidthFunc func() int
}

// New creates a new bordered table that writes to stdout.
func New() *Table {
	return &Table{out: os.Stdout, termWidthFunc: defaultTerminalWidth}
}

// NewWriter creates a new bordered table with a custom destination.
func NewWriter(out io.Writer) *Table {
	return &Table{out: out, termWidthFunc: defaultTerminalWidth}
}

// Header sets the column headers (uppercased for display).
func (t *Table) Header(columns ...string) {
	t.headers = make([]string, len(columns))
	t.widths = make([]int, len(columns))
	for i, c := range columns {
		t.headers[i] = strings.ToUpper(c)
		t.widths[i] = visibleLen(t.headers[i])
	}
}

// Row adds a data row.
func (t *Table) Row(values ...string) {
	t.rows = append(t.rows, values)
	for i, v := range values {
		if i < len(t.widths) && visibleLen(v) > t.widths[i] {
			t.widths[i] = visibleLen(v)
		}
	}
}

// WrapColumns marks the given column indices for word-wrapping. Content
// in a wrapped column that exceeds the column width is broken across
// multiple visual lines instead of being truncated with an ellipsis.
// Non-wrapped columns on a wrapped row are padded with blanks on
// continuation lines.
//
// Wrapped columns still participate in fit-to-terminal shrinking — a
// narrow terminal just causes earlier wrapping. Embedded newlines in
// the source string are preserved as hard line breaks.
func (t *Table) WrapColumns(indices ...int) {
	if t.wrapColumns == nil {
		t.wrapColumns = make(map[int]bool)
	}
	for _, i := range indices {
		t.wrapColumns[i] = true
	}
}

// Flush renders the table, shrinking columns to fit the terminal if needed.
func (t *Table) Flush() error {
	if len(t.headers) == 0 {
		return nil
	}
	t.fitToTerminal()
	fmt.Fprintln(t.out, t.line("╭", "┬", "╮"))
	fmt.Fprintln(t.out, t.formatRow(t.headers, true))
	fmt.Fprintln(t.out, t.line("├", "┼", "┤"))
	for i, row := range t.rows {
		fmt.Fprintln(t.out, t.formatRow(row, false))
		if t.RowBorders && i < len(t.rows)-1 {
			fmt.Fprintln(t.out, t.line("├", "┼", "┤"))
		}
	}
	fmt.Fprintln(t.out, t.line("╰", "┴", "╯"))
	return nil
}

// fitToTerminal shrinks the widest columns to fit within the terminal width.
func (t *Table) fitToTerminal() {
	termWidth := t.termWidthFunc()
	if termWidth <= 0 {
		return
	}

	for t.tableWidth() > termWidth {
		// Find the widest column and shrink it by one.
		widest := 0
		for i := 1; i < len(t.widths); i++ {
			if t.widths[i] > t.widths[widest] {
				widest = i
			}
		}
		// Don't shrink below the header length or 4 chars (room for "x…").
		minWidth := visibleLen(t.headers[widest])
		if minWidth < 4 {
			minWidth = 4
		}
		if t.widths[widest] <= minWidth {
			break // can't shrink further
		}
		t.widths[widest]--
	}
}

// tableWidth returns the total rendered width of the table.
// Each column contributes: 1 (border) + 1 (pad) + width + 1 (pad), plus final border.
func (t *Table) tableWidth() int {
	// │ col1 │ col2 │ ... │ colN │
	// = len(widths) borders + len(widths) * (width + 2 padding) + 1 final border
	w := 1 // leading │
	for _, cw := range t.widths {
		w += cw + 2 + 1 // padding + content + trailing │
	}
	return w
}

func (t *Table) line(left, mid, right string) string {
	parts := make([]string, len(t.widths))
	for i, w := range t.widths {
		parts[i] = strings.Repeat("─", w+2)
	}
	return left + strings.Join(parts, mid) + right
}

// formatRow renders a row, producing multiple newline-joined lines when
// any wrap-enabled column requires wrapping. Headers never wrap.
func (t *Table) formatRow(values []string, bold bool) string {
	cellLines := make([][]string, len(t.widths))
	maxLines := 1
	for i, w := range t.widths {
		val := ""
		if i < len(values) {
			val = values[i]
		}
		if !bold && t.wrapColumns[i] {
			cellLines[i] = wrapVisible(val, w)
		} else {
			cellLines[i] = []string{Truncate(val, w)}
		}
		if len(cellLines[i]) > maxLines {
			maxLines = len(cellLines[i])
		}
	}

	var b strings.Builder
	for line := 0; line < maxLines; line++ {
		if line > 0 {
			b.WriteByte('\n')
		}
		parts := make([]string, len(t.widths))
		for i, w := range t.widths {
			val := ""
			if line < len(cellLines[i]) {
				val = cellLines[i][line]
			}
			padWidth := w + (utf8.RuneCountInString(val) - visibleLen(val))
			cell := fmt.Sprintf(" %-*s ", padWidth, val)
			if bold {
				cell = " \033[1m" + fmt.Sprintf("%-*s", padWidth, val) + "\033[0m "
			}
			parts[i] = cell
		}
		b.WriteString("│" + strings.Join(parts, "│") + "│")
	}
	return b.String()
}

// wrapVisible word-wraps s to width visible characters. It splits on
// whitespace and packs words greedily; words longer than the column
// are hard-broken. Existing newlines produce hard line breaks. ANSI
// escapes are stripped before wrapping because preserving them across
// line breaks is error-prone and rarely useful in wrapped columns.
func wrapVisible(s string, width int) []string {
	if width <= 0 {
		return []string{s}
	}
	stripped := stripANSI(s)
	var out []string
	for _, para := range strings.Split(stripped, "\n") {
		words := strings.Fields(para)
		if len(words) == 0 {
			out = append(out, "")
			continue
		}
		var cur strings.Builder
		curLen := 0
		for _, word := range words {
			wLen := utf8.RuneCountInString(word)
			if wLen > width {
				if curLen > 0 {
					out = append(out, cur.String())
					cur.Reset()
					curLen = 0
				}
				runes := []rune(word)
				for len(runes) > width {
					out = append(out, string(runes[:width]))
					runes = runes[width:]
				}
				if len(runes) > 0 {
					cur.WriteString(string(runes))
					curLen = len(runes)
				}
				continue
			}
			if curLen == 0 {
				cur.WriteString(word)
				curLen = wLen
				continue
			}
			if curLen+1+wLen > width {
				out = append(out, cur.String())
				cur.Reset()
				cur.WriteString(word)
				curLen = wLen
				continue
			}
			cur.WriteByte(' ')
			cur.WriteString(word)
			curLen += 1 + wLen
		}
		if curLen > 0 {
			out = append(out, cur.String())
		}
	}
	return out
}

// Truncate shortens s to maxLen visible characters, replacing the last
// visible char with "…" if needed. If s contains ANSI escape sequences
// and must be truncated, all formatting is stripped and the plain text
// is truncated (this avoids cutting through escape sequences).
func Truncate(s string, maxLen int) string {
	vLen := visibleLen(s)
	if vLen <= maxLen {
		return s
	}
	if maxLen <= 1 {
		return "…"
	}
	plain := stripANSI(s)
	if len(plain) > maxLen-1 {
		plain = plain[:maxLen-1]
	}
	return plain + "…"
}

func defaultTerminalWidth() int {
	w, _, err := term.GetSize(int(os.Stdout.Fd()))
	if err != nil {
		return 0
	}
	return w
}
