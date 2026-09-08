package render

import "fmt"

// PageMeta is the frontmatter shape for paginated list results.
// Page is 1-indexed, matching budget.Paginate: the first page is page 1 and
// the last page is page TotalPages.
type PageMeta struct {
	Page       int `yaml:"page"`
	Total      int `yaml:"total"`
	TotalPages int `yaml:"total_pages"`
}

// NextPageHint returns " Next: page=N." for the page after meta.Page, or ""
// when meta.Page is the last page.
func NextPageHint(meta PageMeta) string {
	if meta.TotalPages <= 1 || meta.Page >= meta.TotalPages || meta.Page < 1 {
		return ""
	}
	return fmt.Sprintf(" Next: page=%d.", meta.Page+1)
}
