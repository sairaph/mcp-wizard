package render

import "fmt"

// PageMeta is the frontmatter shape for paginated list results.
type PageMeta struct {
	Page       int `yaml:"page"`
	Total      int `yaml:"total"`
	TotalPages int `yaml:"total_pages"`
}

// NextPageHint returns "Next: page=N." or "" if this is the last page.
func NextPageHint(meta PageMeta) string {
	if meta.TotalPages <= 1 || meta.Page >= meta.TotalPages-1 {
		return ""
	}
	return fmt.Sprintf(" Next: page=%d.", meta.Page+1)
}
