package derived

import (
	"bytes"
	"fmt"
	"strings"
	"testing"
)

func TestParse(t *testing.T) {
	tests := []struct {
		name       string
		input      string
		wantMeta   map[string]any
		wantBody   string
		wantErr    bool
		nilMeta    bool
	}{
		{
			name: "with frontmatter",
			input: "---\ntitle: Hello\ntags: [a, b]\n---\n# Body\n\nContent here.\n",
			wantMeta: map[string]any{
				"title": "Hello",
				"tags":  []any{"a", "b"},
			},
			wantBody: "# Body\n\nContent here.\n",
		},
		{
			name:     "no frontmatter",
			input:    "# Just a heading\n\nSome text.\n",
			wantBody: "# Just a heading\n\nSome text.\n",
			nilMeta:  true,
		},
		{
			name:     "empty frontmatter",
			input:    "---\n---\n# Body\n",
			wantBody: "# Body\n",
			nilMeta:  true,
		},
		{
			name:    "malformed YAML",
			input:   "---\n: :\n  - :\n  bad: [yaml\n---\nBody\n",
			wantErr: true,
		},
		{
			name: "frontmatter with leading newlines",
			input: "\n\n---\nkey: value\n---\nBody\n",
			wantMeta: map[string]any{
				"key": "value",
			},
			wantBody: "Body\n",
		},
		{
			name:     "only opening delimiter",
			input:    "---\nkey: value\nno closing\n",
			wantBody: "---\nkey: value\nno closing\n",
			nilMeta:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			meta, body, err := Parse([]byte(tt.input))

			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if tt.nilMeta {
				if meta != nil {
					t.Errorf("expected nil meta, got %v", meta)
				}
			} else {
				if meta == nil {
					t.Fatal("expected non-nil meta, got nil")
				}
				for k, want := range tt.wantMeta {
					got, ok := meta[k]
					if !ok {
						t.Errorf("missing key %q in meta", k)
						continue
					}
					// Compare string representations for simplicity with slices.
					if fmt.Sprintf("%v", got) != fmt.Sprintf("%v", want) {
						t.Errorf("meta[%q] = %v, want %v", k, got, want)
					}
				}
			}

			if string(body) != tt.wantBody {
				t.Errorf("body = %q, want %q", string(body), tt.wantBody)
			}
		})
	}
}

func TestRender(t *testing.T) {
	tests := []struct {
		name string
		meta map[string]any
		body string
		want string
	}{
		{
			name: "with meta and body",
			meta: map[string]any{
				"title": "Hello",
			},
			body: "# Content\n",
			want: "---\ntitle: Hello\n---\n# Content\n",
		},
		{
			name: "nil meta returns body only",
			meta: nil,
			body: "# Content\n",
			want: "# Content\n",
		},
		{
			name: "empty meta returns body only",
			meta: map[string]any{},
			body: "# Content\n",
			want: "# Content\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Render(tt.meta, []byte(tt.body))
			if string(got) != tt.want {
				t.Errorf("Render() = %q, want %q", string(got), tt.want)
			}
		})
	}
}

func TestRoundTrip(t *testing.T) {
	originalMeta := map[string]any{
		"title":  "Test Document",
		"author": "tester",
	}
	originalBody := []byte("# Hello World\n\nThis is the body.\n")

	rendered := Render(originalMeta, originalBody)

	parsedMeta, parsedBody, err := Parse(rendered)
	if err != nil {
		t.Fatalf("Parse after Render failed: %v", err)
	}

	if parsedMeta["title"] != "Test Document" {
		t.Errorf("round-trip title = %v, want %q", parsedMeta["title"], "Test Document")
	}
	if parsedMeta["author"] != "tester" {
		t.Errorf("round-trip author = %v, want %q", parsedMeta["author"], "tester")
	}

	if !bytes.Equal(parsedBody, originalBody) {
		t.Errorf("round-trip body = %q, want %q", string(parsedBody), string(originalBody))
	}

	// Modify meta and re-render.
	parsedMeta["status"] = "reviewed"
	reRendered := Render(parsedMeta, parsedBody)

	if !strings.Contains(string(reRendered), "status: reviewed") {
		t.Error("re-rendered output should contain added field")
	}
	if !strings.Contains(string(reRendered), "# Hello World") {
		t.Error("re-rendered output should preserve body")
	}
}

func TestContentHash(t *testing.T) {
	content := []byte("Hello, world!")

	hash1 := ContentHash(content)
	hash2 := ContentHash(content)
	if hash1 != hash2 {
		t.Errorf("ContentHash not deterministic: %q != %q", hash1, hash2)
	}

	if !strings.HasPrefix(hash1, "sha256:") {
		t.Errorf("ContentHash should start with 'sha256:', got %q", hash1)
	}

	// Whitespace trimming: leading/trailing whitespace should not affect the hash.
	hashTrimmed := ContentHash([]byte("  Hello, world!  \n"))
	if hash1 != hashTrimmed {
		t.Errorf("ContentHash should trim whitespace: %q != %q", hash1, hashTrimmed)
	}

	// Different content should produce a different hash.
	hashDiff := ContentHash([]byte("Different content"))
	if hash1 == hashDiff {
		t.Error("different content should produce different hashes")
	}
}
