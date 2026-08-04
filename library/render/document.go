package render

import (
	"bytes"
	"fmt"
	"reflect"

	"gopkg.in/yaml.v3"
)

const MaxBytes = 1 << 20 // 1 MiB hard ceiling

// Document is a YAML frontmatter + Markdown body output.
// Front must be a struct or a concrete typed value - map[string]any
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
		v := reflect.ValueOf(d.Front)
		for v.Kind() == reflect.Ptr {
			if v.IsNil() {
				return "", fmt.Errorf("render: Front must not be a nil pointer")
			}
			v = v.Elem()
		}
		if v.Kind() == reflect.Map {
			return "", fmt.Errorf("render: Front must be a struct, not a map (non-deterministic YAML key order)")
		}
	}

	var buf bytes.Buffer
	buf.WriteString("---\n")
	if d.Front != nil {
		enc := yaml.NewEncoder(&buf)
		enc.SetIndent(2)
		if err := enc.Encode(d.Front); err != nil {
			enc.Close()
			return "", fmt.Errorf("render frontmatter: %w", err)
		}
		if err := enc.Close(); err != nil {
			return "", fmt.Errorf("close yaml encoder: %w", err)
		}
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
