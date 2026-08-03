package table_test

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/sairaph/mcp-wizard/app/table"
)

func TestNew(t *testing.T) {
	m := table.New("Test", []table.Column{
		{Name: "Name", Width: 10},
		{Name: "Age", Width: 5},
	}, []table.Row{
		{"Alice", "30"},
		{"Bob", "25"},
	})
	if m == nil {
		t.Fatal("New returned nil")
	}
	if m.Title != "Test" {
		t.Fatalf("Title = %q, want %q", m.Title, "Test")
	}
	if len(m.Columns) != 2 {
		t.Fatalf("len(Columns) = %d, want 2", len(m.Columns))
	}
	if len(m.Rows) != 2 {
		t.Fatalf("len(Rows) = %d, want 2", len(m.Rows))
	}
	if m.Cursor != 0 {
		t.Fatalf("Cursor = %d, want 0", m.Cursor)
	}
}

func TestSetRows(t *testing.T) {
	m := table.New("", []table.Column{
		{Name: "Name", Width: 10},
	}, []table.Row{
		{"Alice"},
	})
	m.SetRows([]table.Row{
		{"Bob"},
		{"Charlie"},
	})
	if len(m.Rows) != 2 {
		t.Fatalf("len(Rows) = %d, want 2", len(m.Rows))
	}
	if m.Rows[0][0] != "Bob" {
		t.Fatalf("Rows[0][0] = %q, want %q", m.Rows[0][0], "Bob")
	}
}

func TestSetRowsResetsCursor(t *testing.T) {
	m := table.New("", []table.Column{
		{Name: "Name", Width: 10},
	}, []table.Row{
		{"Alice"},
		{"Bob"},
	})
	m.Cursor = 1
	m.SetRows([]table.Row{})
	if m.Cursor != 0 {
		t.Fatalf("Cursor = %d, want 0", m.Cursor)
	}
}

func TestCursorMovement(t *testing.T) {
	m := table.New("", []table.Column{
		{Name: "Name", Width: 10},
	}, []table.Row{
		{"A"}, {"B"}, {"C"},
	})

	// Move down
	_, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")})
	if m.Cursor != 1 {
		t.Fatalf("Cursor after j = %d, want 1", m.Cursor)
	}

	// Move up
	_, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("k")})
	if m.Cursor != 0 {
		t.Fatalf("Cursor after k = %d, want 0", m.Cursor)
	}

	// up at top does nothing
	_, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("k")})
	if m.Cursor != 0 {
		t.Fatalf("Cursor at top after k = %d, want 0", m.Cursor)
	}

	// Move to last
	_, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")})
	_, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")})
	if m.Cursor != 2 {
		t.Fatalf("Cursor at last = %d, want 2", m.Cursor)
	}

	// down at bottom does nothing
	_, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")})
	if m.Cursor != 2 {
		t.Fatalf("Cursor at bottom after j = %d, want 2", m.Cursor)
	}
}

func TestCursorArrowKeys(t *testing.T) {
	m := table.New("", []table.Column{
		{Name: "Name", Width: 10},
	}, []table.Row{
		{"A"}, {"B"},
	})

	_, _ = m.Update(tea.KeyMsg{Type: tea.KeyUp})
	if m.Cursor != 0 {
		t.Fatalf("Cursor after up = %d, want 0", m.Cursor)
	}

	_, _ = m.Update(tea.KeyMsg{Type: tea.KeyDown})
	if m.Cursor != 1 {
		t.Fatalf("Cursor after down = %d, want 1", m.Cursor)
	}
}

func TestSortToggle(t *testing.T) {
	m := table.New("", []table.Column{
		{Name: "Name", Width: 10},
	}, []table.Row{
		{"Alice"},
	})

	// Default descending (Go zero value)
	if m.SortAsc {
		t.Fatal("SortAsc = true, want false")
	}

	// Toggle to ascending
	_, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("s")})
	if !m.SortAsc {
		t.Fatal("SortAsc after s = false, want true")
	}

	// Toggle back to descending
	_, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("s")})
	if m.SortAsc {
		t.Fatal("SortAsc after second s = true, want false")
	}
}

func TestEnterReturnsError(t *testing.T) {
	m := table.New("", []table.Column{
		{Name: "Name", Width: 10},
	}, []table.Row{
		{"Alice"},
		{"Bob"},
	})

	_, err := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if err == nil {
		t.Fatal("expected error on enter, got nil")
	}
	if !strings.Contains(err.Error(), "table:select:0") {
		t.Fatalf("error = %q, want to contain %q", err.Error(), "table:select:0")
	}
}

func TestEnterOnEmptyRows(t *testing.T) {
	m := table.New("", []table.Column{
		{Name: "Name", Width: 10},
	}, []table.Row{})

	_, err := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if err != nil {
		t.Fatalf("expected nil error on empty rows, got %v", err)
	}
}

func TestEmptyRows(t *testing.T) {
	m := table.New("Empty", []table.Column{
		{Name: "Name", Width: 10},
	}, []table.Row{})

	v := m.View()
	if !strings.Contains(v, "No data.") {
		t.Fatalf("View = %q, want to contain %q", v, "No data.")
	}
	if !strings.Contains(v, "Empty") {
		t.Fatalf("View = %q, want to contain title %q", v, "Empty")
	}
}

func TestViewOutput(t *testing.T) {
	m := table.New("People", []table.Column{
		{Name: "Name", Width: 10},
		{Name: "Age", Width: 5},
	}, []table.Row{
		{"Alice", "30"},
		{"Bob", "25"},
	})

	v := m.View()
	if !strings.Contains(v, "Name") {
		t.Fatalf("View missing column Name: %q", v)
	}
	if !strings.Contains(v, "Age") {
		t.Fatalf("View missing column Age: %q", v)
	}
	if !strings.Contains(v, "Alice") {
		t.Fatalf("View missing row Alice: %q", v)
	}
	if !strings.Contains(v, "Bob") {
		t.Fatalf("View missing row Bob: %q", v)
	}
	if !strings.Contains(v, "People") {
		t.Fatalf("View missing title: %q", v)
	}
}

func TestWindowSizeMsg(t *testing.T) {
	m := table.New("", []table.Column{
		{Name: "Name", Width: 10},
	}, []table.Row{
		{"Alice"},
	})

	_, _ = m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	if m.Width != 120 {
		t.Fatalf("Width = %d, want 120", m.Width)
	}
	if m.Height != 40 {
		t.Fatalf("Height = %d, want 40", m.Height)
	}
}

func TestPageUpDown(t *testing.T) {
	m := table.New("", []table.Column{
		{Name: "Name", Width: 10},
	}, []table.Row{
		{"A"}, {"B"}, {"C"},
	})

	// pgup should not panic on small data
	_, _ = m.Update(tea.KeyMsg{Type: tea.KeyPgUp})
	_, _ = m.Update(tea.KeyMsg{Type: tea.KeyPgDown})
}

func TestSortIndicatorInView(t *testing.T) {
	m := table.New("", []table.Column{
		{Name: "Name", Width: 10},
	}, []table.Row{
		{"Alice"},
	})

	// Default is descending (SortAsc = false)
	v := m.View()
	if !strings.Contains(v, "▼") {
		t.Fatalf("Expected descending sort indicator in View, got: %q", v)
	}

	m.SortAsc = true
	m.SetRows(m.Rows)
	v = m.View()
	if !strings.Contains(v, "▲") {
		t.Fatalf("Expected ascending sort indicator in View, got: %q", v)
	}
}
