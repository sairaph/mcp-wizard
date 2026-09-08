package tui

import (
	"fmt"
	"strings"
)

// CheckboxItem is one row in a CheckboxList.
type CheckboxItem struct {
	ID   string
	Name string
}

// CheckboxList renders a multi-select list. selectable and status may be
// nil; hidden is the number of rows filtered out when showAll is false.
func CheckboxList(theme Theme, items []CheckboxItem, cursor int, selected map[string]bool, selectable func(int) bool, status func(int) string, showAll bool, hidden int) string {
	s := theme.Styles()
	var out strings.Builder

	if len(items) == 0 {
		out.WriteString(s.Dim.Render(theme.Indent + theme.Copy.NoClients + "\n"))
		return out.String()
	}

	selectableFn := selectable
	if selectableFn == nil {
		selectableFn = func(i int) bool { return true }
	}
	statusFn := status
	if statusFn == nil {
		statusFn = func(i int) string { return "" }
	}

	for i, item := range items {
		cur := " "
		if i == cursor {
			cur = s.Cursor.Render(">")
			if !selectableFn(i) {
				cur = s.Dim.Render(">")
			}
		}
		mark := s.Off.Render("\u25cb")
		if !selectableFn(i) {
			mark = s.Dim.Render("\u00b7")
		} else if selected[item.ID] {
			mark = s.On.Render("\u25cf")
		}
		line := fmt.Sprintf("%-22s %s", item.Name, s.Dim.Render(statusFn(i)))
		if !selectableFn(i) {
			line = s.Dim.Render(fmt.Sprintf("%-22s ", item.Name)) + s.Hint.Render(statusFn(i))
		}
		fmt.Fprintf(&out, " %s %s %s\n", cur, mark, line)
	}

	if hidden > 0 && !showAll {
		out.WriteString("\n" + s.Dim.Render(
			fmt.Sprintf("  press v to show %d client(s) that are not installed", hidden)))
	} else if showAll {
		out.WriteString("\n" + s.Dim.Render("  press v to hide clients that are not installed"))
	}
	return out.String()
}

// RadioList renders single-choice options with the cursor row marked.
func RadioList(theme Theme, cursor int, options ...string) string {
	s := theme.Styles()
	var out strings.Builder
	for i, option := range options {
		cur := " "
		dot := s.Off.Render("\u25cb")
		if i == cursor {
			cur = s.Cursor.Render(">")
			dot = s.On.Render("\u25cf")
		}
		fmt.Fprintf(&out, " %s %s %s\n", cur, dot, option)
	}
	return out.String()
}

// ActionMenu renders a plain list of actions with a cursor.
func ActionMenu(theme Theme, cursor int, options ...string) string {
	s := theme.Styles()
	var out strings.Builder
	for i, option := range options {
		pointer := " "
		if i == cursor {
			pointer = s.Cursor.Render(">")
		}
		fmt.Fprintf(&out, " %s %s\n", pointer, option)
	}
	return out.String()
}

// ToggleItem is one row in a ToggleList.
type ToggleItem struct {
	ID      string
	Label   string
	Tier    string
	Checked bool
}

// ToggleList renders on/off rows with a cursor, showing each item's Label
// (or ID when Label is empty) and tier in parentheses.
func ToggleList(theme Theme, items []ToggleItem, cursor int) string {
	s := theme.Styles()
	var out strings.Builder
	for i, item := range items {
		pointer := " "
		if i == cursor {
			pointer = s.Cursor.Render(">")
		}
		mark := s.Off.Render("\u25cb")
		if item.Checked {
			mark = s.On.Render("\u25cf")
		}
		label := item.Label
		if label == "" {
			label = item.ID
		}
		line := fmt.Sprintf("%-26s %s", label, s.Dim.Render("("+item.Tier+")"))
		fmt.Fprintf(&out, " %s %s %s\n", pointer, mark, line)
	}
	return out.String()
}
