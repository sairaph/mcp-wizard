package installer

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/sairaph/mcp-wizard/flow"
	"github.com/sairaph/mcp-wizard/tui"
)

// TransportState is embedded in consumer state for transport selection.
type TransportState struct {
	Transport string // "stdio" or "http"
	HTTPAddr  string // listen address (only for http)
	Stage     int    // 0 = transport choice, 1 = address input
	Cursor    int
	Input     string
}

// TransportStep returns a flow.Step that lets the user choose between
// stdio (local) and HTTP (network) transport for the MCP server.
func TransportStep[T any](stateFn func(*T) *TransportState) flow.Step[T] {
	return &transportStep[T]{stateFn: stateFn}
}

type transportStep[T any] struct {
	stateFn func(*T) *TransportState
}

func (s *transportStep[T]) ID() string { return "transport" }

func (s *transportStep[T]) Title(state *T) string {
	if s.stateFn == nil {
		return "Server transport"
	}
	return "Server transport - how should AI clients connect?"
}

func (s *transportStep[T]) Hints(state *T) []struct{ Key, Label string } {
	if s.stateFn == nil {
		return nil
	}
	ts := s.stateFn(state)
	if ts == nil {
		return nil
	}
	if ts.Stage == 1 {
		return []struct{ Key, Label string }{
			{Key: "enter", Label: "confirm"},
			{Key: "esc", Label: "back"},
		}
	}
	return []struct{ Key, Label string }{
		{Key: "↑↓", Label: "move"},
		{Key: "enter", Label: "select"},
	}
}

func (s *transportStep[T]) Init(state *T) tea.Cmd {
	if s.stateFn == nil {
		return nil
	}
	ts := s.stateFn(state)
	if ts == nil {
		return nil
	}
	if ts.Transport == "" {
		ts.Transport = "stdio"
	}
	ts.Stage = 0
	return nil
}

func (s *transportStep[T]) Update(msg tea.Msg, state *T) (flow.Directive, tea.Cmd) {
	if s.stateFn == nil {
		return flow.Fail, nil
	}
	ts := s.stateFn(state)
	if ts == nil {
		return flow.Fail, nil
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		if ts.Stage == 0 {
			switch msg.String() {
			case "up", "k":
				if ts.Cursor > 0 {
					ts.Cursor--
				}
			case "down", "j":
				if ts.Cursor < 1 {
					ts.Cursor++
				}
			case "enter":
				if ts.Cursor == 0 {
					ts.Transport = "stdio"
					return flow.Next, nil
				}
				ts.Transport = "http"
				ts.Stage = 1
				ts.Input = "127.0.0.1:8080"
				return flow.Continue, nil
			case "q", "ctrl+c":
				return flow.Quit, nil
			}
		} else {
			switch msg.String() {
			case "q", "ctrl+c":
				return flow.Quit, nil
			case "enter":
				if ts.Input == "" {
					ts.Input = "127.0.0.1:8080"
				}
				ts.HTTPAddr = ts.Input
				return flow.Next, nil
			case "esc":
				ts.Stage = 0
				return flow.Continue, nil
			case "backspace":
				if len(ts.Input) > 0 {
					runes := []rune(ts.Input)
					ts.Input = string(runes[:len(runes)-1])
				}
			default:
				if len(msg.Runes) > 0 {
					ts.Input += string(msg.Runes)
				}
			}
		}
	}
	return flow.Continue, nil
}

func (s *transportStep[T]) View(state *T) string {
	if s.stateFn == nil {
		return ""
	}
	ts := s.stateFn(state)
	if ts == nil {
		return ""
	}

	if ts.Stage == 0 {
		options := []string{"stdio (local) - AI client spawns the server", "HTTP (network) - server listens on a TCP port"}
		var out strings.Builder
		out.WriteString(tui.Section(tui.DefaultTheme, s.Title(state), ""))
		for i, opt := range options {
			prefix := "  "
			if i == ts.Cursor {
				prefix = tui.DefaultTheme.Styles().Cursor.Render("> ")
			}
			dot := tui.DefaultTheme.Styles().Off.Render("○")
			if i == ts.Cursor {
				dot = tui.DefaultTheme.Styles().On.Render("●")
			}
			fmt.Fprintf(&out, "%s %s %s\n", prefix, dot, opt)
		}
		out.WriteString("\n" + tui.Footer(tui.DefaultTheme, "↑↓ move · enter select · q cancel"))
		return out.String()
	}

	var out strings.Builder
	out.WriteString(tui.Section(tui.DefaultTheme, "HTTP address - what port should the server listen on?", ""))
	fmt.Fprintf(&out, "  %s_\n", ts.Input)
	out.WriteString("\n" + tui.Footer(tui.DefaultTheme, "enter confirm · esc back"))
	return out.String()
}
