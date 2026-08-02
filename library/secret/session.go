// Package secret provides credential lifecycle management: storage, loading,
// and typed credential bags for login flows.
package secret

// Session is a typed credential bag. It carries both user-entered values
// (email, token) and server-issued values (CSRF tokens, cookies, session IDs)
// between login stages and to persistent storage.
type Session struct {
	Values map[string]any
}

// NewSession returns an empty session.
func NewSession() *Session {
	return &Session{Values: make(map[string]any)}
}

// Set stores a value by key.
func (s *Session) Set(key string, value any) {
	if s == nil {
		return
	}
	if s.Values == nil {
		s.Values = make(map[string]any)
	}
	s.Values[key] = value
}

// Get retrieves a value by key. Returns nil if not found.
func (s *Session) Get(key string) any {
	if s == nil || s.Values == nil {
		return nil
	}
	return s.Values[key]
}

// GetString retrieves a string value by key. Returns "" if not found or not a string.
func (s *Session) GetString(key string) string {
	v := s.Get(key)
	if v == nil {
		return ""
	}
	str, ok := v.(string)
	if !ok {
		return ""
	}
	return str
}

// Clone returns a shallow copy of the session.
func (s *Session) Clone() *Session {
	if s == nil || s.Values == nil {
		return NewSession()
	}
	cp := NewSession()
	for k, v := range s.Values {
		cp.Values[k] = v
	}
	return cp
}
