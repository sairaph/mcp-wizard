package flow

// BaseState carries cross-cutting fields every step touches.
// Consumer state structs embed this.
type BaseState struct {
	Spinner  SpinnerState
	Width    int
	Height   int
	Message  string // transient, current-screen (recoverable error)
	Failure  error  // terminal error (causes Fail)
	Warning  string // non-fatal, surfaced in summary
	Settled  bool   // point of no return — cancel is normal exit, not failure
	NextStep string // target step ID when Directive is Jump
}

// SpinnerState tracks the spinner animation frame.
type SpinnerState struct {
	Frame  int
	Active bool
}
