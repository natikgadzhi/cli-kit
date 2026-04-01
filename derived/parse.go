package derived

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"strings"

	"gopkg.in/yaml.v3"
)

// delimiter is the YAML frontmatter delimiter.
var delimiter = []byte("---")

// Parse splits a markdown file with YAML frontmatter into metadata and body.
// Returns nil meta if no frontmatter delimiters (---) are found.
func Parse(content []byte) (meta map[string]any, body []byte, err error) {
	trimmed := bytes.TrimLeft(content, "\n\r")
	if !bytes.HasPrefix(trimmed, delimiter) {
		return nil, content, nil
	}

	// Find the end of the opening delimiter line.
	rest := trimmed[len(delimiter):]
	// The opening delimiter must be followed by a newline.
	idx := bytes.IndexByte(rest, '\n')
	if idx == -1 {
		return nil, content, nil
	}
	// Check that nothing but whitespace follows the opening "---" on the same line.
	if strings.TrimSpace(string(rest[:idx])) != "" {
		return nil, content, nil
	}
	rest = rest[idx+1:]

	// Find the closing delimiter.
	closingIdx := bytes.Index(rest, delimiter)
	if closingIdx == -1 {
		return nil, content, nil
	}

	// Verify the closing delimiter is on its own line.
	if closingIdx > 0 && rest[closingIdx-1] != '\n' {
		return nil, content, nil
	}

	yamlContent := rest[:closingIdx]
	afterClosing := rest[closingIdx+len(delimiter):]

	// The closing delimiter must be followed by a newline or end of content.
	if len(afterClosing) > 0 {
		nlIdx := bytes.IndexByte(afterClosing, '\n')
		if nlIdx == -1 {
			// Closing delimiter is last line with no trailing newline.
			if strings.TrimSpace(string(afterClosing)) != "" {
				return nil, content, nil
			}
			afterClosing = nil
		} else {
			if strings.TrimSpace(string(afterClosing[:nlIdx])) != "" {
				return nil, content, nil
			}
			afterClosing = afterClosing[nlIdx+1:]
		}
	}

	// Parse YAML. Empty frontmatter (---\n---) yields nil meta.
	yamlTrimmed := bytes.TrimSpace(yamlContent)
	if len(yamlTrimmed) == 0 {
		return nil, afterClosing, nil
	}

	meta = make(map[string]any)
	if err := yaml.Unmarshal(yamlContent, &meta); err != nil {
		return nil, content, fmt.Errorf("parsing frontmatter YAML: %w", err)
	}

	return meta, afterClosing, nil
}

// Render produces a markdown file with YAML frontmatter from a metadata map and body.
// If meta is nil or empty, returns body unchanged.
func Render(meta map[string]any, body []byte) []byte {
	if len(meta) == 0 {
		return body
	}

	yamlBytes, err := yaml.Marshal(meta)
	if err != nil {
		// This should not happen for map[string]any values that came from Parse.
		panic(fmt.Sprintf("frontmatter: failed to marshal YAML: %v", err))
	}

	var buf bytes.Buffer
	buf.Write(delimiter)
	buf.WriteByte('\n')
	buf.Write(yamlBytes)
	buf.Write(delimiter)
	buf.WriteByte('\n')
	buf.Write(body)
	return buf.Bytes()
}

// ContentHash computes SHA-256 of content (with leading/trailing whitespace trimmed),
// returning "sha256:<hex>".
func ContentHash(content []byte) string {
	h := sha256.Sum256(bytes.TrimSpace(content))
	return fmt.Sprintf("sha256:%x", h)
}
