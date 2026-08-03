package paginator_test

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/sairaph/mcp-wizard/app"
	"github.com/sairaph/mcp-wizard/app/paginator"
)

func TestCursorMovement(t *testing.T) {
	m := paginator.New("test", []string{"a", "b", "c"}, 0, 5, 3)
	if m.Cursor != 0 {
		t.Fatalf("expected cursor 0, got %d", m.Cursor)
	}
	m.Update(tea.KeyMsg{Type: tea.KeyDown})
	if m.Cursor != 1 {
		t.Fatalf("expected cursor 1 after down, got %d", m.Cursor)
	}
	m.Update(tea.KeyMsg{Type: tea.KeyDown})
	if m.Cursor != 2 {
		t.Fatalf("expected cursor 2 after down, got %d", m.Cursor)
	}
	m.Update(tea.KeyMsg{Type: tea.KeyDown})
	if m.Cursor != 2 {
		t.Fatalf("expected cursor 2 at bottom, got %d", m.Cursor)
	}
	m.Update(tea.KeyMsg{Type: tea.KeyUp})
	if m.Cursor != 1 {
		t.Fatalf("expected cursor 1 after up, got %d", m.Cursor)
	}
	m.Update(tea.KeyMsg{Type: tea.KeyUp})
	m.Update(tea.KeyMsg{Type: tea.KeyUp})
	if m.Cursor != 0 {
		t.Fatalf("expected cursor 0 at top, got %d", m.Cursor)
	}
}

func TestCursorMovementKeys(t *testing.T) {
	m := paginator.New("test", []string{"x", "y", "z"}, 0, 5, 3)
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	if m.Cursor != 1 {
		t.Fatalf("expected cursor 1 after j, got %d", m.Cursor)
	}
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}})
	if m.Cursor != 0 {
		t.Fatalf("expected cursor 0 after k, got %d", m.Cursor)
	}
}

func TestPageNavigationNext(t *testing.T) {
	m := paginator.New("test", []string{"a", "b"}, 0, 2, 10)
	if !m.HasNext() {
		t.Fatal("expected HasNext=true")
	}
	cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}})
	if cmd == nil {
		t.Fatal("expected non-nil cmd for next page")
	}
	msg := cmd()
	action, ok := msg.(app.ActionMsg)
	if !ok {
		t.Fatalf("expected ActionMsg, got %T", msg)
	}
	if action.Source != "paginator" || action.Value != "next" {
		t.Fatalf("expected paginator/next action, got %s/%s", action.Source, action.Value)
	}
}

func TestPageNavigationPrev(t *testing.T) {
	m := paginator.New("test", []string{"a", "b"}, 1, 2, 10)
	if !m.HasPrev() {
		t.Fatal("expected HasPrev=true")
	}
	cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'p'}})
	if cmd == nil {
		t.Fatal("expected non-nil cmd for prev page")
	}
	msg := cmd()
	action, ok := msg.(app.ActionMsg)
	if !ok {
		t.Fatalf("expected ActionMsg, got %T", msg)
	}
	if action.Source != "paginator" || action.Value != "prev" {
		t.Fatalf("expected paginator/prev action, got %s/%s", action.Source, action.Value)
	}
}

func TestPageNavigationRight(t *testing.T) {
	m := paginator.New("test", []string{"a"}, 0, 1, 3)
	cmd := m.Update(tea.KeyMsg{Type: tea.KeyRight})
	if cmd == nil {
		t.Fatal("expected non-nil cmd for right key")
	}
}

func TestPageNavigationLeft(t *testing.T) {
	m := paginator.New("test", []string{"a"}, 1, 1, 3)
	cmd := m.Update(tea.KeyMsg{Type: tea.KeyLeft})
	if cmd == nil {
		t.Fatal("expected non-nil cmd for left key")
	}
}

func TestPageNavigationNoNextOnLastPage(t *testing.T) {
	m := paginator.New("test", []string{"a"}, 1, 1, 2)
	if m.HasNext() {
		t.Fatal("expected HasNext=false on last page")
	}
	cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}})
	if cmd != nil {
		t.Fatal("expected nil cmd (no next page)")
	}
}

func TestPageNavigationNoPrevOnFirstPage(t *testing.T) {
	m := paginator.New("test", []string{"a"}, 0, 1, 2)
	if m.HasPrev() {
		t.Fatal("expected HasPrev=false on first page")
	}
	cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'p'}})
	if cmd != nil {
		t.Fatal("expected nil cmd (no prev page)")
	}
}

func TestSelectItem(t *testing.T) {
	m := paginator.New("test", []string{"item0", "item1"}, 0, 5, 2)
	cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil {
		t.Fatal("expected non-nil cmd for enter")
	}
	msg := cmd()
	action, ok := msg.(app.ActionMsg)
	if !ok {
		t.Fatalf("expected ActionMsg, got %T", msg)
	}
	if action.Source != "paginator" || action.Value != "select" {
		t.Fatalf("expected paginator/select action, got %s/%s", action.Source, action.Value)
	}
	if action.Data != 0 {
		t.Fatalf("expected Data=0, got %v", action.Data)
	}
}

func TestSelectOnEmptyList(t *testing.T) {
	m := paginator.New("test", nil, 0, 5, 0)
	cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd != nil {
		t.Fatal("expected nil cmd for empty list")
	}
}

func TestTotalPages(t *testing.T) {
	tests := []struct {
		total, perPage, want int
	}{
		{0, 5, 1},
		{1, 5, 1},
		{5, 5, 1},
		{6, 5, 2},
		{10, 5, 2},
		{11, 5, 3},
		{10, 0, 1},
	}
	for _, tc := range tests {
		m := paginator.New("t", nil, 0, tc.perPage, tc.total)
		got := m.TotalPages()
		if got != tc.want {
			t.Errorf("TotalPages(total=%d, perPage=%d) = %d, want %d", tc.total, tc.perPage, got, tc.want)
		}
	}
}

func TestViewContainsTitle(t *testing.T) {
	m := paginator.New("My Cards", []string{"card1", "card2"}, 0, 5, 2)
	v := m.View()
	if !strings.Contains(v, "My Cards") {
		t.Fatal("view should contain title")
	}
}

func TestViewContainsFooter(t *testing.T) {
	m := paginator.New("test", []string{"a"}, 0, 1, 3)
	v := m.View()
	if !strings.Contains(v, "page 1/3") {
		t.Fatal("view should contain page info")
	}
}

func TestViewNextPrevIndicators(t *testing.T) {
	// First page with next
	m := paginator.New("test", []string{"a"}, 0, 1, 3)
	v := m.View()
	if !strings.Contains(v, "n next") {
		t.Fatal("view should show n next on first page")
	}
	if strings.Contains(v, "p prev") {
		t.Fatal("view should NOT show p prev on first page")
	}

	// Middle page
	m2 := paginator.New("test", []string{"a"}, 1, 1, 3)
	v2 := m2.View()
	if !strings.Contains(v2, "n next") {
		t.Fatal("view should show n next on middle page")
	}
	if !strings.Contains(v2, "p prev") {
		t.Fatal("view should show p prev on middle page")
	}

	// Last page
	m3 := paginator.New("test", []string{"a"}, 2, 1, 3)
	v3 := m3.View()
	if strings.Contains(v3, "n next") {
		t.Fatal("view should NOT show n next on last page")
	}
	if !strings.Contains(v3, "p prev") {
		t.Fatal("view should show p prev on last page")
	}
}

func TestInitReturnsNil(t *testing.T) {
	m := paginator.New("t", nil, 0, 5, 0)
	if cmd := m.Init(); cmd != nil {
		t.Fatal("Init should return nil")
	}
}
