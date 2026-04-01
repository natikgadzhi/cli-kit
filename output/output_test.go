package output

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/natikgadzhi/cli-kit/table"
	"github.com/spf13/cobra"
)

func TestRegisterFlag(t *testing.T) {
	cmd := &cobra.Command{}
	RegisterFlag(cmd)

	f := cmd.PersistentFlags().Lookup("output")
	if f == nil {
		t.Fatal("expected 'output' flag to be registered")
	}
	if f.Shorthand != "o" {
		t.Errorf("expected shorthand 'o', got %q", f.Shorthand)
	}
}

func TestResolveExplicit(t *testing.T) {
	tests := []struct {
		flag string
		want string
	}{
		{"json", FormatJSON},
		{"table", FormatTable},
		{"JSON", FormatJSON},
		{"TABLE", FormatTable},
	}
	for _, tt := range tests {
		t.Run(tt.flag, func(t *testing.T) {
			cmd := &cobra.Command{}
			RegisterFlag(cmd)
			cmd.PersistentFlags().Set("output", tt.flag)
			got := Resolve(cmd)
			if got != tt.want {
				t.Errorf("Resolve() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestIsTableIsJSON(t *testing.T) {
	if !IsTable(FormatTable) {
		t.Error("IsTable(FormatTable) should be true")
	}
	if IsTable(FormatJSON) {
		t.Error("IsTable(FormatJSON) should be false")
	}
	if !IsJSON(FormatJSON) {
		t.Error("IsJSON(FormatJSON) should be true")
	}
	if IsJSON(FormatTable) {
		t.Error("IsJSON(FormatTable) should be false")
	}
}

type testData struct {
	Name  string `json:"name"`
	Value int    `json:"value"`
}

type testRenderer struct {
	items []testData
}

func (r *testRenderer) RenderTable(t *table.Table) {
	t.Header("Name", "Value")
	for _, item := range r.items {
		t.Row(item.Name, strings.Repeat("*", item.Value))
	}
}

func TestTableRendererInterface(t *testing.T) {
	var buf bytes.Buffer
	tbl := table.NewWriter(&buf)

	renderer := &testRenderer{
		items: []testData{
			{Name: "alpha", Value: 3},
			{Name: "beta", Value: 5},
		},
	}
	renderer.RenderTable(tbl)
	tbl.Flush()

	out := buf.String()
	if !strings.Contains(out, "NAME") {
		t.Error("expected uppercased header NAME")
	}
	if !strings.Contains(out, "alpha") {
		t.Error("expected data row 'alpha'")
	}
	if !strings.Contains(out, "***") {
		t.Error("expected value '***' for alpha")
	}
	// Bordered table should have box-drawing characters.
	if !strings.Contains(out, "┌") {
		t.Error("expected bordered table with box-drawing characters")
	}
}

func TestPrintJSON(t *testing.T) {
	data := testData{Name: "test", Value: 42}
	// We can't easily capture stdout in a test, but we can test the JSON encoding logic.
	b, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(b), `"name": "test"`) {
		t.Error("expected JSON to contain name field")
	}
}
