package installer

import (
	"context"
	"errors"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/sairaph/mcp-wizard/flow"
	"github.com/sairaph/mcp-wizard/secret"
	"github.com/sairaph/mcp-wizard/tui"
)

// LoginConfig describes a credential collection flow.
type LoginConfig struct {
	ID        string
	Label     string
	Skippable bool
	Stages    []LoginStage
	Store     secret.Store
}

// LoginStage is one step in a multi-stage login flow.
type LoginStage struct {
	Prompt string
	Fields []LoginField
	Submit func(ctx context.Context, sess *secret.Session) error
}

// LoginField is a single credential field.
type LoginField struct {
	Name     string
	Label    string
	Masked   bool
	Validate func(string) error
}

// LoginState is embedded in consumer state for login.
type LoginState struct {
	Stage      int
	Session    *secret.Session
	Skipped    bool
	Input      string
	Cursor     int
	Submitting bool
}

// LoginStep returns a flow.Step for credential collection.
func LoginStep[T any](config LoginConfig, stateFn func(*T) *LoginState) flow.Step[T] {
	return &loginStep[T]{
		config:  config,
		stateFn: stateFn,
	}
}

type loginStep[T any] struct {
	config  LoginConfig
	stateFn func(*T) *LoginState
}

type skipLoginMsg struct{}

func (s *loginStep[T]) ID() string { return s.config.ID }

func (s *loginStep[T]) Title(state *T) string {
	lState := s.stateFn(state)
	if lState == nil {
		return s.config.Label
	}
	if lState.Stage >= 0 && lState.Stage < len(s.config.Stages) {
		stage := s.config.Stages[lState.Stage]
		if stage.Prompt != "" {
			return s.config.Label + " \u2014 " + stage.Prompt
		}
	}
	return s.config.Label
}

func (s *loginStep[T]) Hints(state *T) []struct{ Key, Label string } {
	lState := s.stateFn(state)
	if lState == nil {
		return nil
	}
	if lState.Submitting {
		return nil
	}
	if lState.Stage == -1 {
		return []struct{ Key, Label string }{
			{Key: "\u2191\u2193", Label: "move"},
			{Key: "enter", Label: "select"},
			{Key: "esc", Label: "skip"},
		}
	}
	return []struct{ Key, Label string }{
		{Key: "enter", Label: "confirm"},
		{Key: "esc", Label: "skip"},
	}
}

type submitResultMsg struct {
	stage int
	err   error
}

type persistResultMsg struct {
	err error
}

func (s *loginStep[T]) Init(state *T) tea.Cmd {
	lState := s.stateFn(state)
	if lState == nil {
		return nil
	}

	// Check if already logged in (session exists).
	if s.config.Store != nil {
		sess, exists, err := s.config.Store.Load(context.Background())
		if err == nil && exists && sess.GetString("email") != "" {
			lState.Session = sess
			lState.Skipped = false
			return func() tea.Msg { return skipLoginMsg{} }
		}
	}

	lState.Stage = -1 // Show the "Sign in / Skip for now" screen first
	lState.Session = secret.NewSession()
	lState.Input = ""
	lState.Cursor = 0
	lState.Submitting = false
	return nil
}

func (s *loginStep[T]) Update(msg tea.Msg, state *T) (flow.Directive, tea.Cmd) {
	lState := s.stateFn(state)
	if lState == nil {
		return flow.Fail, nil
	}

	switch msg.(type) {
	case skipLoginMsg:
		return flow.Skip, nil
	}

	if lState.Submitting {
		switch msg := msg.(type) {
		case submitResultMsg:
			lState.Submitting = false
			if msg.err != nil {
				var retry secret.RetryableError
				if errors.As(msg.err, &retry) {
					return flow.Continue, nil
				}
				return flow.Continue, nil
			}
			return s.advanceStage(lState)

		case persistResultMsg:
			lState.Submitting = false
			if msg.err != nil {
				return flow.Continue, nil
			}
			return flow.Next, nil
		}
		return flow.Continue, nil
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		// Initial choice screen (Sign in / Skip).
		if lState.Stage == -1 {
			switch msg.String() {
			case "up", "k":
				lState.Cursor = 1 - lState.Cursor
			case "down", "j":
				lState.Cursor = 1 - lState.Cursor
			case "enter":
				if lState.Cursor == 1 || !s.config.Skippable {
					lState.Stage = 0
					lState.Input = ""
					return flow.Continue, nil
				}
				lState.Skipped = true
				return flow.Next, nil
			case "esc":
				if s.config.Skippable {
					lState.Skipped = true
					return flow.Next, nil
				}
				return flow.Quit, nil
			case "q", "ctrl+c":
				return flow.Quit, nil
			}
			return flow.Continue, nil
		}

		// Input screen.
		stage := s.config.Stages[lState.Stage]
		switch msg.String() {
		case "enter":
			allValid := true
			for _, f := range stage.Fields {
				if f.Validate != nil {
					if err := f.Validate(lState.Input); err != nil {
						allValid = false
					}
				}
			}
			if !allValid || lState.Input == "" {
				return flow.Continue, nil
			}

			// Store the current field value.
			if len(stage.Fields) > 0 {
				lState.Session.Set(stage.Fields[0].Name, lState.Input)
			}

			// Submit if there's a submit function.
			if stage.Submit != nil {
				lState.Submitting = true
				stageIdx := lState.Stage
				sess := lState.Session
				return flow.Continue, func() tea.Msg {
					err := stage.Submit(context.Background(), sess)
					return submitResultMsg{stage: stageIdx, err: err}
				}
			}
			return s.advanceStage(lState)

		case "esc":
			if s.config.Skippable {
				lState.Skipped = true
				return flow.Next, nil
			}
			return flow.Back, nil

		case "q", "ctrl+c":
			return flow.Quit, nil

		case "backspace":
			if len(lState.Input) > 0 {
				lState.Input = lState.Input[:len(lState.Input)-1]
			}

		default:
			if len(msg.Runes) > 0 {
				lState.Input += string(msg.Runes)
			}
		}
	}
	return flow.Continue, nil
}

func (s *loginStep[T]) advanceStage(lState *LoginState) (flow.Directive, tea.Cmd) {
	if lState.Stage+1 < len(s.config.Stages) {
		lState.Stage++
		lState.Input = ""
		return flow.Continue, nil
	}
	// All stages done -- persist.
	if s.config.Store != nil {
		lState.Submitting = true
		return flow.Continue, func() tea.Msg {
			err := s.config.Store.Save(context.Background(), lState.Session)
			return persistResultMsg{err: err}
		}
	}
	return flow.Next, nil
}

func (s *loginStep[T]) View(state *T) string {
	lState := s.stateFn(state)
	if lState == nil {
		return ""
	}

	theme := tui.DefaultTheme
	var content string

	if lState.Stage == -1 {
		// Sign in / Skip choice.
		options := []string{"Sign in", "Skip for now"}
		content = tui.ActionMenu(theme, lState.Cursor, options...)
		content += "\n" + tui.Footer(theme, tui.Hints(theme,
			tui.Hint{Key: "\u2191\u2193", Label: "move"},
			tui.Hint{Key: "enter", Label: "select"},
			tui.Hint{Key: "esc", Label: "skip"},
		))
	} else if lState.Stage < len(s.config.Stages) {
		// Input stage.
		stage := s.config.Stages[lState.Stage]

		masked := false
		if lState.Stage < len(s.config.Stages) && len(stage.Fields) > 0 {
			masked = stage.Fields[0].Masked
		}

		content = "  " + tui.TextInput(lState.Input, "type here...", masked) + "\n"
		content += "\n" + tui.Footer(theme, tui.Hints(theme,
			tui.Hint{Key: "enter", Label: "confirm"},
			tui.Hint{Key: "esc", Label: "skip"},
		))
	} else {
		content = "  Signed in.\n"
	}

	return tui.Section(theme, s.Title(state), content)
}
