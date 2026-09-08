// Package budget provides deterministic o200k_base token accounting so tool
// output can be capped by tokens rather than by bytes or line counts.
package budget

import (
	"errors"
	"fmt"
	"math"
	"sync"
	"unicode/utf8"

	"github.com/tiktoken-go/tokenizer/codec"
)

var (
	codecOnce sync.Once
	encoder   *codec.Codec
	encErr    error
)

func encoding() (*codec.Codec, error) {
	codecOnce.Do(func() {
		defer func() {
			if r := recover(); r != nil {
				encErr = fmt.Errorf("tokenizer init panic: %v", r)
			}
		}()
		encoder = codec.NewO200kBase()
		if encoder == nil {
			encErr = errors.New("codec.NewO200kBase returned nil")
		}
	})
	if encErr != nil {
		return nil, encErr
	}
	return encoder, nil
}

// Count returns the exact number of o200k_base tokens in text.
func Count(text string) (int, error) {
	enc, err := encoding()
	if err != nil {
		return 0, err
	}
	return enc.Count(text)
}

// FitLines returns the longest run of lines that fits tokenLimit.
//
// fromEnd selects which end survives: true keeps the newest lines and drops
// the oldest (it_tail), false keeps the oldest and drops the newest (it_head).
// A single line larger than the budget is still returned, because returning
// nothing would be a worse answer than returning one oversized line.
func FitLines(lines []string, tokenLimit int, fromEnd bool) (kept []string, omitted int, err error) {
	if len(lines) == 0 {
		return nil, 0, nil
	}
	if tokenLimit <= 0 {
		tokenLimit = 1
	}

	// Per-line costs, including the newline that joins them.
	costs := make([]int, len(lines))
	for i, line := range lines {
		n, err := Count(line + "\n")
		if err != nil {
			return nil, 0, err
		}
		costs[i] = n
	}

	used, count := 0, 0
	if fromEnd {
		for i := len(lines) - 1; i >= 0; i-- {
			if count > 0 && used+costs[i] > tokenLimit {
				break
			}
			used += costs[i]
			count++
		}
		return lines[len(lines)-count:], len(lines) - count, nil
	}
	for i := range lines {
		if count > 0 && used+costs[i] > tokenLimit {
			break
		}
		used += costs[i]
		count++
	}
	return lines[:count], len(lines) - count, nil
}

// Truncate returns a valid UTF-8 prefix bounded by both tokens and bytes.
// The returned token count describes the returned prefix, not either limit.
func Truncate(text string, tokenLimit, byteLimit int) (prefix string, tokens int, truncated bool, err error) {
	if !utf8.ValidString(text) {
		return "", 0, false, errors.New("token budget input is not valid UTF-8")
	}
	if tokenLimit <= 0 && byteLimit <= 0 {
		n, err := Count(text)
		if err != nil {
			return "", 0, false, err
		}
		return text, n, false, nil
	}
	if tokenLimit <= 0 {
		tokenLimit = math.MaxInt
	}
	if byteLimit <= 0 {
		byteLimit = len(text)
	}

	bounded := text
	if len(bounded) > byteLimit {
		bounded = validPrefix(bounded, byteLimit)
		truncated = true
	}
	enc, err := encoding()
	if err != nil {
		return "", 0, false, err
	}

	probeLimit := 4 << 10
	if tokenLimit <= (int(^uint(0)>>1))/8 && tokenLimit*8 > probeLimit {
		probeLimit = tokenLimit * 8
	}
	probeLimit = min(probeLimit, len(bounded))
	candidate := validPrefix(bounded, probeLimit)
	var ids []uint
	for {
		ids, _, err = enc.Encode(candidate)
		if err != nil {
			return "", 0, false, err
		}
		if len(ids) > tokenLimit || len(candidate) == len(bounded) {
			break
		}
		next := min(len(bounded), max(len(candidate)+1, len(candidate)*2))
		candidate = validPrefix(bounded, next)
	}
	if len(ids) <= tokenLimit {
		return candidate, len(ids), truncated || len(candidate) != len(text), nil
	}

	truncated = true
	// Find the longest valid prefix of candidate within the token budget by
	// bisecting on byte length. Joining the encoder's pieces would not do:
	// byte-fallback tokens for control or invalid bytes do not round-trip to
	// the original text, so the result would not be a prefix.
	lo, hi := 0, len(candidate)
	for lo < hi {
		mid := (lo + hi + 1) / 2
		p := validPrefix(candidate, mid)
		n, err := enc.Count(p)
		if err != nil {
			return "", 0, false, err
		}
		if n <= tokenLimit {
			lo = len(p)
			if len(p) < mid {
				// validPrefix backed off a partial rune; keep bisecting above it.
				lo = mid
				if lo > hi {
					lo = hi
				}
			}
		} else {
			hi = mid - 1
		}
	}
	prefix = validPrefix(candidate, lo)
	tokens, err = enc.Count(prefix)
	if err != nil {
		return "", 0, false, err
	}
	for tokens > tokenLimit && prefix != "" {
		_, size := utf8.DecodeLastRuneInString(prefix)
		prefix = prefix[:len(prefix)-size]
		tokens, err = enc.Count(prefix)
		if err != nil {
			return "", 0, false, err
		}
	}
	return prefix, tokens, true, nil
}

func validPrefix(text string, limit int) string {
	if limit <= 0 {
		return ""
	}
	prefix := text
	if limit < len(prefix) {
		prefix = prefix[:limit]
	}
	for !utf8.ValidString(prefix) {
		prefix = prefix[:len(prefix)-1]
	}
	return prefix
}
