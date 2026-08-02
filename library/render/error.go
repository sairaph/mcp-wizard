package render

import (
	"errors"
	"reflect"
)

// Error is the structured error shape for MCP tool results.
type Error struct {
	Code    string         `yaml:"code"`
	Message string         `yaml:"message"`
	Hint    string         `yaml:"hint,omitempty"`
	Fields  map[string]any `yaml:"fields,omitempty"`
}

// Standard error codes. Consumers may define additional domain codes.
const (
	CodeNotFound     = "not_found"
	CodeInvalidInput = "invalid_input"
	CodeAuth         = "authentication"
	CodeForbidden    = "forbidden"
	CodeRateLimited  = "rate_limited"
	CodeAmbiguous    = "ambiguous"
	CodeConflict     = "conflict"
	CodeUnavailable  = "unavailable"
	CodeInternal     = "internal_error"
)

// ErrorMapping maps a Go error type to a render Error via errors.As.
type ErrorMapping struct {
	// Target is a pointer to a concrete error type, e.g. &MyNotFoundError{}.
	Target any
	// Code is the error code to use when this error type is matched.
	Code string
	// Hint is a static hint string to include in the error.
	Hint string
	// Extract is an optional function to pull dynamic fields from the error.
	// If nil, the error's Error() string is used as the message.
	Extract func(err error) (message string, fields map[string]any)
}

// Classify maps a Go error to a render.Error using errors.As against the
// provided table. Unrecognised errors map to CodeInternal with a generic hint.
func Classify(err error, table []ErrorMapping) Error {
	if err == nil {
		return Error{}
	}
	for _, mapping := range table {
		if mapping.Target != nil {
			tv := reflect.ValueOf(mapping.Target)
			if tv.Kind() == reflect.Ptr {
				dbl := reflect.New(tv.Type())
				if errors.As(err, dbl.Interface()) {
					message := err.Error()
					var fields map[string]any
				if mapping.Extract != nil {
					msg, f := mapping.Extract(err)
					if msg != "" {
						message = msg
					}
					if f != nil {
						fields = f
					}
				}
					return Error{
						Code:    mapping.Code,
						Message: message,
						Hint:    mapping.Hint,
						Fields:  fields,
					}
				}
			}
		}
	}
	return Error{
		Code:    CodeInternal,
		Message: err.Error(),
		Hint:    "Retry the call; if it keeps failing, run `<binary> doctor`.",
	}
}
