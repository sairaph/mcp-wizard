package budget

// Paginate groups complete records into pages by the token count of
// render(records). A record that exceeds the budget by itself remains the sole
// record of its page rather than being dropped.
//
// The render callback receives a SLICE of records (the proposed window),
// not one record at a time, because YAML frontmatter overhead is per-page.
func Paginate[T any](records []T, requestedPage, tokenLimit int, render func([]T) (string, error)) (window []T, totalPages int, err error) {
	if len(records) == 0 {
		return nil, 0, nil
	}
	if tokenLimit <= 0 {
		tokenLimit = 1
	}
	starts := []int{0}
	start := 0
	for end := 1; end <= len(records); end++ {
		representation, err := render(records[start:end])
		if err != nil {
			return nil, 0, err
		}
		used, err := Count(representation)
		if err != nil {
			return nil, 0, err
		}
		if end-start > 1 && used > tokenLimit {
			start = end - 1
			starts = append(starts, start)
		}
	}

	totalPages = len(starts)
	if requestedPage < 1 || requestedPage > totalPages {
		return nil, totalPages, nil
	}
	start = starts[requestedPage-1]
	end := len(records)
	if requestedPage < totalPages {
		end = starts[requestedPage]
	}
	return append([]T(nil), records[start:end]...), totalPages, nil
}
