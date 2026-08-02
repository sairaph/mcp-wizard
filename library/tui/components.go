package tui

import (
	"fmt"
	"strings"
)

type CheckboxItem struct {
	ID   string
	Name string
}

func CheckboxList(theme Theme, items []CheckboxItem, cursor int, selected map[string]bool, selectable func(int) bool, status func(int) string, showAll bool, hidden int) string {
	s := theme.Styles()
	var out strings.Builder

	if len(items) == 0 {
		out.WriteString(s.Dim.Render(theme.Indent + theme.Copy.NoClients + "\n"))
		return out.String()
	}

	for i, item := range items {
		cur := " "
		if i == cursor && selectable(i) {
			cur = s.Cursor.Render(">")
		}
		mark := s.Off.Render("\u25cb")
		if !selectable(i) {
			mark = s.Dim.Render("\u00b7")
		} else if selected[item.ID] {
			mark = s.On.Render("\u25cf")
		}
		line := fmt.Sprintf("%-22s %s", item.Name, s.Dim.Render(status(i)))
		if !selectable(i) {
			line = s.Dim.Render(fmt.Sprintf("%-22s ", item.Name)) + s.Hint.Render(status(i))
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

type toggleItem struct {
	ID      string
	Label   string
	Tier    string
	Checked bool
}

func ToggleList(theme Theme, items []toggleItem, cursor int) string {
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
		line := fmt.Sprintf("%-26s %s", item.ID, s.Dim.Render("("+item.Tier+")"))
		fmt.Fprintf(&out, " %s %s %s\n", pointer, mark, line)
	}
	return out.String()
}
