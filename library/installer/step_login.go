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
	Field  LoginField
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
	Message    string
}

// LoginStep returns a flow.Step for credential collection.
func LoginStep[T any](ctx context.Context, config LoginConfig, stateFn func(*T) *LoginState) flow.Step[T] {
	return &loginStep[T]{
		config:  config,
		stateFn: stateFn,
		ctx:     ctx,
	}
}

type loginStep[T any] struct {
	config  LoginConfig
	stateFn func(*T) *LoginState
	ctx     context.Context
}

type skipLoginMsg struct{}

func (s *loginStep[T]) ID() string { return s.config.ID }

func (s *loginStep[T]) Title(state *T) string {
	if s.stateFn == nil {
		return s.config.Label
	}
	lState := s.stateFn(state)
	if lState == nil {
		return s.config.Label
	}
	if lState.Stage >= 0 && lState.Stage < len(s.config.Stages) {
		stage := s.config.Stages[lState.Stage]
		if stage.Prompt != "" {
			return s.config.Label + " - " + stage.Prompt
		}
	}
	return s.config.Label
}

func (s *loginStep[T]) Hints(state *T) []struct{ Key, Label string } {
	if s.stateFn == nil {
		return nil
	}
	lState := s.stateFn(state)
	if lState == nil {
		return nil
	}
	if lState.Submitting {
		return nil
	}
	if lState.Stage == -1 {
		hints := []struct{ Key, Label string }{
			{Key: "\u2191\u2193", Label: "move"},
			{Key: "enter", Label: "select"},
		}
		if s.config.Skippable {
			hints = append(hints, struct{ Key, Label string }{Key: "esc", Label: "skip"})
		} else {
			hints = append(hints, struct{ Key, Label string }{Key: "esc", Label: "back"})
		}
		return hints
	}
	return []struct{ Key, Label string }{
		{Key: "enter", Label: "confirm"},
		{Key: "esc", Label: "back"},
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
	if s.stateFn == nil {
		return nil
	}
	lState := s.stateFn(state)
	if lState == nil {
		return nil
	}

	// Check if already logged in (session exists).
	if s.config.Store != nil {
		sess, exists, err := s.config.Store.Load(s.ctx)
		if err == nil && exists && sess != nil {
			hasSession := false
			for _, stage := range s.config.Stages {
				if sess.GetString(stage.Field.Name) != "" {
					hasSession = true
					break
				}
			}
			if hasSession {
				lState.Session = sess
				lState.Skipped = false
				return func() tea.Msg { return skipLoginMsg{} }
			}
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
	if s.stateFn == nil {
		return flow.Fail, nil
	}
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
				lState.Message = msg.err.Error()
				var retry secret.RetryableError
				if errors.As(msg.err, &retry) {
					return flow.Continue, nil
				}
				return flow.Fail, nil
			}
			return s.advanceStage(lState)

		case persistResultMsg:
			lState.Submitting = false
			if msg.err != nil {
				lState.Message = "failed to save credentials: " + msg.err.Error()
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
				if !s.config.Skippable && lState.Cursor > 0 {
					lState.Cursor = 0
				}
			case "down", "j":
				lState.Cursor = 1 - lState.Cursor
				if !s.config.Skippable && lState.Cursor > 0 {
					lState.Cursor = 0
				}
			case "enter":
				if lState.Cursor == 0 {
					lState.Stage = 0
					lState.Input = ""
					return flow.Continue, nil
				}
				if s.config.Skippable {
					lState.Skipped = true
					return flow.Next, nil
				}
				return flow.Quit, nil
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
		if lState.Stage < 0 || lState.Stage >= len(s.config.Stages) {
			return flow.Next, nil
		}
		stage := s.config.Stages[lState.Stage]
		switch msg.String() {
		case "enter":
			allValid := true
			if stage.Field.Validate != nil {
				if err := stage.Field.Validate(lState.Input); err != nil {
					allValid = false
					lState.Message = err.Error()
				}
			}
			if !allValid {
				return flow.Continue, nil
			}

			// Store the current field value.
			if lState.Session == nil {
				lState.Session = secret.NewSession()
			}
			lState.Session.Set(stage.Field.Name, lState.Input)

			// Submit if there's a submit function.
			if stage.Submit != nil {
				lState.Submitting = true
				stageIdx := lState.Stage
				sess := lState.Session
				return flow.Continue, func() tea.Msg {
					err := stage.Submit(s.ctx, sess)
					return submitResultMsg{stage: stageIdx, err: err}
				}
			}
			return s.advanceStage(lState)

		case "esc":
			if lState.Stage >= 0 {
				lState.Stage = -1
				lState.Input = ""
				lState.Message = ""
				return flow.Continue, nil
			}
			if s.config.Skippable {
				lState.Skipped = true
				return flow.Next, nil
			}
			return flow.Back, nil

		case "q", "ctrl+c":
			return flow.Quit, nil

		case "backspace":
			runes := []rune(lState.Input)
			if len(runes) > 0 {
				lState.Input = string(runes[:len(runes)-1])
			}

		default:
			if len(msg.Runes) > 0 {
				lState.Input += string(msg.Runes)
			}
		}
	}
	return flow.Continue, nil
}

func advanceStage(lState *LoginState, stages int) {
	if lState.Stage+1 >= stages {
		return
	}
	lState.Stage++
	lState.Input = ""
	lState.Message = ""
	lState.Submitting = false
}

func (s *loginStep[T]) advanceStage(lState *LoginState) (flow.Directive, tea.Cmd) {
	if lState.Stage+1 < len(s.config.Stages) {
		advanceStage(lState, len(s.config.Stages))
		return flow.Continue, nil
	}
	// All stages done -- persist.
	if s.config.Store != nil {
		lState.Submitting = true
		return flow.Continue, func() tea.Msg {
			err := s.config.Store.Save(s.ctx, lState.Session)
			return persistResultMsg{err: err}
		}
	}
	return flow.Next, nil
}

func (s *loginStep[T]) View(state *T) string {
	if s.stateFn == nil {
		return ""
	}
	lState := s.stateFn(state)
	if lState == nil {
		return ""
	}

	theme := tui.DefaultTheme
	var content string

	if lState.Stage == -1 {
		// Sign in / Skip choice.
		options := []string{"Sign in"}
		if s.config.Skippable {
			options = append(options, "Skip for now")
		}
		content = tui.ActionMenu(theme, lState.Cursor, options...)
		content += "\n" + tui.Footer(theme, tui.Hints(theme,
			tui.Hint{Key: "\u2191\u2193", Label: "move"},
			tui.Hint{Key: "enter", Label: "select"},
		))
		if s.config.Skippable {
			content += "\n" + tui.Footer(theme, tui.Hints(theme,
				tui.Hint{Key: "esc", Label: "skip"},
			))
		}
	} else if lState.Stage >= 0 && lState.Stage < len(s.config.Stages) {
		// Input stage.
		stage := s.config.Stages[lState.Stage]

		masked := stage.Field.Masked

		content = "  " + tui.TextInput(lState.Input, "type here...", masked) + "\n"
		if lState.Message != "" {
			content += "\n" + theme.Styles().Err.Render("  " + lState.Message) + "\n"
		}
		content += "\n" + tui.Footer(theme, tui.Hints(theme,
			tui.Hint{Key: "enter", Label: "confirm"},
			tui.Hint{Key: "esc", Label: "back"},
		))
	} else {
		content = "  Signed in.\n"
	}

	return tui.Section(theme, s.Title(state), content)
}
