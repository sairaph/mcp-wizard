package budget_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/sairaph/mcp-wizard/budget"
)

func TestCount(t *testing.T) {
	count, err := budget.Count("hello world")
	if err != nil {
		t.Fatal(err)
	}
	if count != 2 {
		t.Errorf("Count: got %d, want 2", count)
	}
}

func TestFitLinesFromStart(t *testing.T) {
	lines := []string{"first", "second", "third", "fourth", "fifth"}
	kept, omitted, err := budget.FitLines(lines, 4, false)
	if err != nil {
		t.Fatal(err)
	}
	if len(kept) == 0 || kept[0] != "first" {
		t.Errorf("fromStart should keep the oldest lines, got %v", kept)
	}
	if omitted != len(lines)-len(kept) {
		t.Errorf("omitted: got %d, want %d", omitted, len(lines)-len(kept))
	}
}

func TestFitLinesFromEnd(t *testing.T) {
	lines := []string{"first", "second", "third", "fourth", "fifth"}
	kept, omitted, err := budget.FitLines(lines, 4, true)
	if err != nil {
		t.Fatal(err)
	}
	if len(kept) == 0 || kept[len(kept)-1] != "fifth" {
		t.Errorf("fromEnd should keep the newest lines, got %v", kept)
	}
	if omitted != len(lines)-len(kept) {
		t.Errorf("omitted: got %d, want %d", omitted, len(lines)-len(kept))
	}
}

func TestFitLinesOversizedLine(t *testing.T) {
	huge := strings.Repeat("word ", 5_000)
	kept, omitted, err := budget.FitLines([]string{huge, "second"}, 10, true)
	if err != nil {
		t.Fatal(err)
	}
	if len(kept) != 1 {
		t.Fatalf("got %d lines, want exactly 1", len(kept))
	}
	if kept[0] != "second" {
		t.Errorf("fromEnd should have kept the last line")
	}
	if omitted != 1 {
		t.Errorf("omitted: got %d, want 1", omitted)
	}
}

func TestFitLinesEmpty(t *testing.T) {
	kept, omitted, err := budget.FitLines(nil, 100, true)
	if err != nil || len(kept) != 0 || omitted != 0 {
		t.Errorf("empty input should be a no-op, got %v %d %v", kept, omitted, err)
	}
}

func TestTruncateBasic(t *testing.T) {
	text := strings.Repeat("alpha beta ", 500)
	prefix, tokens, truncated, err := budget.Truncate(text, 50, 1<<20)
	if err != nil {
		t.Fatal(err)
	}
	if !truncated || tokens > 50 {
		t.Errorf("token limit not honoured: %d tokens, truncated=%v", tokens, truncated)
	}
	if !strings.HasPrefix(text, prefix) {
		t.Error("the result must be a prefix of the input")
	}
}

func TestTruncateByteLimit(t *testing.T) {
	text := strings.Repeat("alpha beta ", 500)
	prefix, _, truncated, err := budget.Truncate(text, 1_000_000, 100)
	if err != nil {
		t.Fatal(err)
	}
	if !truncated || len(prefix) > 100 {
		t.Errorf("byte limit not honoured: %d bytes, truncated=%v", len(prefix), truncated)
	}
}

func TestTruncateValidUTF8(t *testing.T) {
	text := strings.Repeat("こんにちは ", 200)
	prefix, tokens, truncated, err := budget.Truncate(text, 30, 500)
	if err != nil {
		t.Fatal(err)
	}
	if !truncated || tokens > 30 {
		t.Errorf("token limit not honoured: %d tokens, truncated=%v", tokens, truncated)
	}
	if !strings.HasPrefix(text, prefix) {
		t.Error("the result must be a prefix of the input")
	}
}

func TestPaginateBasic(t *testing.T) {
	records := make([]string, 40)
	for i := range records {
		records[i] = strings.Repeat("record ", 10)
	}
	render := func(items []string) (string, error) { return strings.Join(items, "\n"), nil }

	seen := 0
	var totalPages int
	for page := 1; ; page++ {
		window, pages, err := budget.Paginate(records, page, 100, render)
		if err != nil {
			t.Fatal(err)
		}
		totalPages = pages
		if len(window) == 0 {
			break
		}
		seen += len(window)
		if page > pages {
			t.Fatal("paginate returned records beyond the last page")
		}
	}
	if totalPages < 2 {
		t.Fatalf("a 100-token budget should need several pages, got %d", totalPages)
	}
	if seen != len(records) {
		t.Errorf("pagination lost records: saw %d of %d", seen, len(records))
	}
}

func TestPaginateOversizedRecord(t *testing.T) {
	records := []string{"small", strings.Repeat("huge ", 2_000), "small"}
	render := func(items []string) (string, error) { return strings.Join(items, "\n"), nil }

	seen := 0
	for page := 1; page <= 10; page++ {
		window, pages, err := budget.Paginate(records, page, 20, render)
		if err != nil {
			t.Fatal(err)
		}
		seen += len(window)
		if page >= pages {
			break
		}
	}
	if seen != len(records) {
		t.Errorf("every record must be delivered, saw %d of %d", seen, len(records))
	}
}

func TestPaginateEmpty(t *testing.T) {
	window, pages, err := budget.Paginate(nil, 1, 100, func(items []string) (string, error) { return "", nil })
	if err != nil || len(window) != 0 || pages != 0 {
		t.Errorf("empty input should be a no-op, got %v %d %v", window, pages, err)
	}
}

func TestPaginateOutOfRange(t *testing.T) {
	records := []string{"one", "two", "three"}
	render := func(items []string) (string, error) { return strings.Join(items, "\n"), nil }

	window, pages, err := budget.Paginate(records, 999, 10, render)
	if err != nil {
		t.Fatal(err)
	}
	if pages < 1 {
		t.Fatalf("should report total pages, got %d", pages)
	}
	if len(window) != 0 {
		t.Errorf("out-of-range page should return nil window, got %v", window)
	}
}

func TestPaginateRenderError(t *testing.T) {
	records := []string{"one", "two"}
	render := func(items []string) (string, error) {
		return "", errors.New("render failure")
	}
	_, _, err := budget.Paginate(records, 1, 100, render)
	if err == nil {
		t.Fatal("expected render error to propagate")
	}
}
