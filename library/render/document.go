package render

import (
	"bytes"
	"fmt"

	"gopkg.in/yaml.v3"
)

const MaxBytes = 1 << 20 // 1 MiB hard ceiling

// Document is a YAML frontmatter + Markdown body output.
// Front must be a struct or a concrete typed value — map[string]any
// produces non-deterministic key order in YAML and is rejected at runtime.
type Document struct {
	Front   any
	Body    string
	IsError bool
}

// String renders the document as "---\nyaml\n---\nbody".
// Returns an error if Front is a map (non-deterministic YAML key order)
// or if the output exceeds MaxBytes.
func (d Document) String() (string, error) {
	if d.Front != nil {
		if _, isMap := d.Front.(map[string]any); isMap {
			return "", fmt.Errorf("render: Front must be a struct, not map[string]any (non-deterministic YAML key order)")
		}
	}

	var buf bytes.Buffer
	buf.WriteString("---\n")
	if d.Front != nil {
		enc := yaml.NewEncoder(&buf)
		enc.SetIndent(2)
		if err := enc.Encode(d.Front); err != nil {
			return "", fmt.Errorf("render frontmatter: %w", err)
		}
		enc.Close()
	}
	buf.WriteString("---")
	if d.Body != "" {
		buf.WriteString("\n")
		buf.WriteString(d.Body)
	}
	buf.WriteString("\n")

	if buf.Len() > MaxBytes {
		return "", fmt.Errorf("render: document exceeds %d bytes", MaxBytes)
	}
	return buf.String(), nil
}
