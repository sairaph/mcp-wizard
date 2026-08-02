package tui

import "github.com/charmbracelet/lipgloss"

type Colors struct {
	Primary   lipgloss.Color
	Secondary lipgloss.Color
	Success   lipgloss.Color
	Error     lipgloss.Color
	Warning   lipgloss.Color
}

var DefaultColors = Colors{
	Primary:   lipgloss.Color("81"),
	Secondary: lipgloss.Color("244"),
	Success:   lipgloss.Color("42"),
	Error:     lipgloss.Color("203"),
	Warning:   lipgloss.Color("214"),
}

type Copy struct {
	Title     string
	Move      string
	Toggle    string
	AllNone   string
	ShowAll   string
	Continue  string
	Back      string
	Cancel    string
	Select    string
	Confirm   string
	Finish    string
	Edit      string
	Restore   string
	NoClients string
}

var DefaultCopy = Copy{
	Title:     "setup",
	Move:      "\u2191\u2193 move",
	Toggle:    "space toggle",
	AllNone:   "a all/none",
	ShowAll:   "v show all",
	Continue:  "enter continue",
	Back:      "esc back",
	Cancel:    "q cancel",
	Select:    "enter select",
	Confirm:   "enter confirm",
	Finish:    "enter to finish",
	Edit:      "enter edit",
	Restore:   "r restore defaults",
	NoClients: "No AI clients detected.",
}

type Theme struct {
	Colors    Colors
	Copy      Copy
	Indent    string
	WrapWidth int
}

var DefaultTheme = Theme{
	Colors:    DefaultColors,
	Copy:      DefaultCopy,
	Indent:    "  ",
	WrapWidth: 76,
}

type themeStyles struct {
	Title   lipgloss.Style
	Dim     lipgloss.Style
	Cursor  lipgloss.Style
	On      lipgloss.Style
	Off     lipgloss.Style
	Err     lipgloss.Style
	Hint    lipgloss.Style
	Footer  lipgloss.Style
}

func (t Theme) Styles() themeStyles {
	return themeStyles{
		Title:  lipgloss.NewStyle().Bold(true).Foreground(t.Colors.Primary),
		Dim:    lipgloss.NewStyle().Foreground(t.Colors.Secondary),
		Cursor: lipgloss.NewStyle().Foreground(t.Colors.Primary),
		On:     lipgloss.NewStyle().Foreground(t.Colors.Success),
		Off:    lipgloss.NewStyle().Foreground(t.Colors.Secondary),
		Err:    lipgloss.NewStyle().Foreground(t.Colors.Error),
		Hint:   lipgloss.NewStyle().Foreground(t.Colors.Warning),
		Footer: lipgloss.NewStyle().Foreground(t.Colors.Secondary),
	}
}
