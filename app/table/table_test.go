package table_test

import (
	"fmt"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/sairaph/mcp-wizard/app"
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
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")})
	if m.Cursor != 1 {
		t.Fatalf("Cursor after j = %d, want 1", m.Cursor)
	}

	// Move up
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("k")})
	if m.Cursor != 0 {
		t.Fatalf("Cursor after k = %d, want 0", m.Cursor)
	}

	// up at top does nothing
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("k")})
	if m.Cursor != 0 {
		t.Fatalf("Cursor at top after k = %d, want 0", m.Cursor)
	}

	// Move to last
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")})
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")})
	if m.Cursor != 2 {
		t.Fatalf("Cursor at last = %d, want 2", m.Cursor)
	}

	// down at bottom does nothing
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")})
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

	m.Update(tea.KeyMsg{Type: tea.KeyUp})
	if m.Cursor != 0 {
		t.Fatalf("Cursor after up = %d, want 0", m.Cursor)
	}

	m.Update(tea.KeyMsg{Type: tea.KeyDown})
	if m.Cursor != 1 {
		t.Fatalf("Cursor after down = %d, want 1", m.Cursor)
	}
}

func TestEnterReturnsAction(t *testing.T) {
	m := table.New("", []table.Column{
		{Name: "Name", Width: 10},
	}, []table.Row{
		{"Alice"},
		{"Bob"},
	})

	cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil {
		t.Fatal("expected cmd on enter, got nil")
	}
	msg := cmd()
	am, ok := msg.(app.ActionMsg)
	if !ok {
		t.Fatalf("expected app.ActionMsg, got %T", msg)
	}
	if am.Source != "table" {
		t.Fatalf("expected source 'table', got %q", am.Source)
	}
	if am.Value != "select" {
		t.Fatalf("expected value 'select', got %q", am.Value)
	}
	if am.Data != 0 {
		t.Fatalf("expected data 0, got %v", am.Data)
	}
}

func TestEnterOnEmptyRows(t *testing.T) {
	m := table.New("", []table.Column{
		{Name: "Name", Width: 10},
	}, []table.Row{})

	cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd != nil {
		t.Fatalf("expected nil cmd on empty rows, got %v", cmd)
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

	m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
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

	// Paging on data smaller than the viewport clamps to the last and first
	// rows instead of running past them.
	m.Update(tea.KeyMsg{Type: tea.KeyPgDown})
	if m.Cursor != len(m.Rows)-1 {
		t.Fatalf("pgdown on small data: Cursor = %d, want %d", m.Cursor, len(m.Rows)-1)
	}
	m.Update(tea.KeyMsg{Type: tea.KeyPgUp})
	if m.Cursor != 0 {
		t.Fatalf("pgup on small data: Cursor = %d, want 0", m.Cursor)
	}
}

func TestSortIndicatorInView(t *testing.T) {
	m := table.New("", []table.Column{
		{Name: "Name", Width: 10},
	}, []table.Row{
		{"Alice"},
	})

	// Unsorted by default: no indicator.
	v := m.View()
	if strings.Contains(v, "\u25bc") || strings.Contains(v, "\u25b2") {
		t.Fatalf("Expected no sort indicator for insertion order, got: %q", v)
	}

	m.Sort(0, true)
	v = m.View()
	if !strings.Contains(v, "\u25b2") {
		t.Fatalf("Expected ascending sort indicator in View, got: %q", v)
	}

	m.Sort(0, false)
	v = m.View()
	if !strings.Contains(v, "\u25bc") {
		t.Fatalf("Expected descending sort indicator in View, got: %q", v)
	}
}

func TestSortOrdersRows(t *testing.T) {
	m := table.New("", []table.Column{
		{Name: "Name", Width: 10},
		{Name: "Age", Width: 4},
	}, []table.Row{
		{"Charlie", "3"},
		{"Alice", "2"},
		{"Bob"}, // short row sorts as empty in column 1
	})

	m.Sort(0, true)
	if m.Rows[0][0] != "Alice" || m.Rows[1][0] != "Bob" || m.Rows[2][0] != "Charlie" {
		t.Fatalf("ascending by Name: got %v", m.Rows)
	}
	m.Sort(0, false)
	if m.Rows[0][0] != "Charlie" || m.Rows[2][0] != "Alice" {
		t.Fatalf("descending by Name: got %v", m.Rows)
	}
	m.Sort(1, true)
	if m.Rows[0][0] != "Bob" || m.Rows[1][0] != "Alice" || m.Rows[2][0] != "Charlie" {
		t.Fatalf("ascending by Age with short row first: got %v", m.Rows)
	}
	m.Sort(-1, false)
	if m.Rows[0][0] != "Charlie" || m.Rows[1][0] != "Alice" || m.Rows[2][0] != "Bob" {
		t.Fatalf("insertion order restored: got %v", m.Rows)
	}
	if m.SortBy != -1 {
		t.Fatalf("SortBy = %d, want -1", m.SortBy)
	}
}

func TestSortKeyCycles(t *testing.T) {
	m := table.New("", []table.Column{
		{Name: "A", Width: 4},
		{Name: "B", Width: 4},
	}, []table.Row{{"2", "x"}, {"1", "y"}})

	press := func() { m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'s'}}) }

	press()
	if m.SortBy != 0 || !m.SortAsc || m.Rows[0][0] != "1" {
		t.Fatalf("first s: want col 0 asc, got SortBy=%d asc=%v rows=%v", m.SortBy, m.SortAsc, m.Rows)
	}
	press()
	if m.SortBy != 0 || m.SortAsc || m.Rows[0][0] != "2" {
		t.Fatalf("second s: want col 0 desc, got SortBy=%d asc=%v rows=%v", m.SortBy, m.SortAsc, m.Rows)
	}
	press()
	if m.SortBy != 1 || !m.SortAsc {
		t.Fatalf("third s: want col 1 asc, got SortBy=%d asc=%v", m.SortBy, m.SortAsc)
	}
	press()
	press()
	if m.SortBy != -1 || m.Rows[0][0] != "2" {
		t.Fatalf("after cycling every column: want insertion order, got SortBy=%d rows=%v", m.SortBy, m.Rows)
	}
}

func TestSetRowsKeepsSort(t *testing.T) {
	m := table.New("", []table.Column{{Name: "N", Width: 4}}, nil)
	m.Sort(0, true)
	m.SetRows([]table.Row{{"b"}, {"a"}})
	if m.Rows[0][0] != "a" {
		t.Fatalf("SetRows must re-apply the active sort, got %v", m.Rows)
	}
}

func TestCursorFollowsIntoViewport(t *testing.T) {
	var rows []table.Row
	for i := 0; i < 40; i++ {
		rows = append(rows, table.Row{fmt.Sprintf("row%02d", i)})
	}
	m := table.New("", []table.Column{{Name: "N", Width: 8}}, rows)
	m.Update(tea.WindowSizeMsg{Width: 80, Height: 14}) // viewport height 10

	for i := 0; i < 25; i++ {
		m.Update(tea.KeyMsg{Type: tea.KeyDown})
	}
	if m.Cursor != 25 {
		t.Fatalf("Cursor = %d, want 25", m.Cursor)
	}
	if !strings.Contains(m.View(), "row25") {
		t.Fatalf("cursor row must be visible after moving down, view:\n%s", m.View())
	}
	m.Update(tea.KeyMsg{Type: tea.KeyHome})
	if v := m.View(); !strings.Contains(v, "row00") || !strings.Contains(v, "N ") {
		t.Fatalf("home must scroll back to the first row with the header visible, view:\n%s", v)
	}
	for i := 0; i < 30; i++ {
		m.Update(tea.KeyMsg{Type: tea.KeyDown})
	}
	for i := 0; i < 30; i++ {
		m.Update(tea.KeyMsg{Type: tea.KeyUp})
	}
	if v := m.View(); !strings.Contains(v, "N ") {
		t.Fatalf("scrolling back up to the first row must reveal the header, view:\n%s", v)
	}
	m.Update(tea.KeyMsg{Type: tea.KeyEnd})
	if !strings.Contains(m.View(), "row39") {
		t.Fatalf("end must scroll to the last row, view:\n%s", m.View())
	}
	m.Update(tea.KeyMsg{Type: tea.KeyPgUp})
	if m.Cursor != 29 {
		t.Fatalf("pgup moves the cursor one viewport up, Cursor = %d, want 29", m.Cursor)
	}
	if !strings.Contains(m.View(), "row29") {
		t.Fatalf("cursor row must be visible after pgup, view:\n%s", m.View())
	}
}
