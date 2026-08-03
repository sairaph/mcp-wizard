package app

// BaseModel is embedded by every app's model struct.
type BaseModel struct {
	Width   int
	Height  int
	Status  string // transient status line
	Failure string // fatal error
	Quit    bool   // set to true to exit the app
}
