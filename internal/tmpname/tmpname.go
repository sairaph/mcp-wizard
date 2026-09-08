// Package tmpname generates short random suffixes for temporary file names.
package tmpname

import (
	"crypto/rand"
	"time"
)

const letters = "abcdefghijklmnopqrstuvwxyz0123456789"

// Suffix returns n random lowercase alphanumeric characters. It falls back
// to clock bits if the system random source fails, which is acceptable
// because callers also rely on O_EXCL or on the rename target not existing.
func Suffix(n int) string {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		now := time.Now().UnixNano()
		for i := range b {
			b[i] = byte(now >> uint(i*8%64))
		}
	}
	for i := range b {
		b[i] = letters[int(b[i])%len(letters)]
	}
	return string(b)
}
