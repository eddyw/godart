//go:build dart

// Package strings gives the operations on a Go string.
package strings

import "syscall/dart"

// Index gives the position of the first substr in s, or -1.
func Index(s, substr string) int {
	n := len(substr)
	if n == 0 {
		return 0
	}
	if n > len(s) {
		return -1
	}
	for i := 0; i+n <= len(s); i++ {
		if s[i:i+n] == substr {
			return i
		}
	}
	return -1
}

// LastIndex gives the position of the last substr in s, or -1.
func LastIndex(s, substr string) int {
	n := len(substr)
	if n == 0 {
		return len(s)
	}
	for i := len(s) - n; i >= 0; i-- {
		if s[i:i+n] == substr {
			return i
		}
	}
	return -1
}

// IndexByte gives the position of the first c in s, or -1.
func IndexByte(s string, c byte) int {
	for i := 0; i < len(s); i++ {
		if s[i] == c {
			return i
		}
	}
	return -1
}

// Contains reports whether substr is inside s.
func Contains(s, substr string) bool {
	return Index(s, substr) >= 0
}

// ContainsRune reports whether r is inside s.
func ContainsRune(s string, r rune) bool {
	for _, c := range s {
		if c == r {
			return true
		}
	}
	return false
}

// HasPrefix reports whether s starts with prefix.
func HasPrefix(s, prefix string) bool {
	return len(s) >= len(prefix) && s[0:len(prefix)] == prefix
}

// HasSuffix reports whether s ends with suffix.
func HasSuffix(s, suffix string) bool {
	return len(s) >= len(suffix) && s[len(s)-len(suffix):] == suffix
}

// Count gives the count of substr inside s.
func Count(s, substr string) int {
	if len(substr) == 0 {
		n := 1
		for range s {
			n++
		}
		return n
	}
	n := 0
	i := 0
	for i+len(substr) <= len(s) {
		if s[i:i+len(substr)] == substr {
			n++
			i += len(substr)
			continue
		}
		i++
	}
	return n
}

// Repeat gives s a count of times.
func Repeat(s string, count int) string {
	if count < 0 {
		panic("strings: negative Repeat count")
	}
	out := ""
	for range count {
		out += s
	}
	return out
}

// Join puts sep between each element of elems.
func Join(elems []string, sep string) string {
	out := ""
	for i, v := range elems {
		if i > 0 {
			out += sep
		}
		out += v
	}
	return out
}

// Split cuts s at each sep.
func Split(s, sep string) []string {
	if len(sep) == 0 {
		out := []string{}
		for _, r := range s {
			out = append(out, string(r))
		}
		return out
	}
	out := []string{}
	start := 0
	i := 0
	for i+len(sep) <= len(s) {
		if s[i:i+len(sep)] != sep {
			i++
			continue
		}
		out = append(out, s[start:i])
		i += len(sep)
		start = i
	}
	return append(out, s[start:])
}

// SplitN cuts s at each sep, and it gives at most n parts.
func SplitN(s, sep string, n int) []string {
	if n == 0 {
		return nil
	}
	parts := Split(s, sep)
	if n < 0 || len(parts) <= n {
		return parts
	}
	out := parts[:n-1]
	rest := parts[n-1]
	for i := n; i < len(parts); i++ {
		rest += sep + parts[i]
	}
	return append(out, rest)
}

// Replace changes the first n occurrences of old to new. A negative n changes
// each one.
func Replace(s, old, new string, n int) string {
	if len(old) == 0 || n == 0 {
		return s
	}
	out := ""
	count := 0
	i := 0
	start := 0
	for i+len(old) <= len(s) {
		if s[i:i+len(old)] != old {
			i++
			continue
		}
		out += s[start:i] + new
		i += len(old)
		start = i
		count++
		if n > 0 && count == n {
			break
		}
	}
	return out + s[start:]
}

// ReplaceAll changes each old to new.
func ReplaceAll(s, old, new string) string {
	return Replace(s, old, new, -1)
}

// TrimPrefix removes prefix from the start of s.
func TrimPrefix(s, prefix string) string {
	if HasPrefix(s, prefix) {
		return s[len(prefix):]
	}
	return s
}

// TrimSuffix removes suffix from the end of s.
func TrimSuffix(s, suffix string) string {
	if len(suffix) > 0 && HasSuffix(s, suffix) {
		return s[:len(s)-len(suffix)]
	}
	return s
}

// TrimSpace removes the space at the start and at the end of s.
func TrimSpace(s string) string {
	start := 0
	for start < len(s) && isSpace(s[start]) {
		start++
	}
	end := len(s)
	for end > start && isSpace(s[end-1]) {
		end--
	}
	return s[start:end]
}

// Trim removes each byte of cutset from the start and from the end of s.
func Trim(s, cutset string) string {
	start := 0
	for start < len(s) && IndexByte(cutset, s[start]) >= 0 {
		start++
	}
	end := len(s)
	for end > start && IndexByte(cutset, s[end-1]) >= 0 {
		end--
	}
	return s[start:end]
}

// Fields cuts s at each run of space.
func Fields(s string) []string {
	out := []string{}
	start := -1
	for i := 0; i < len(s); i++ {
		if isSpace(s[i]) {
			if start >= 0 {
				out = append(out, s[start:i])
				start = -1
			}
			continue
		}
		if start < 0 {
			start = i
		}
	}
	if start >= 0 {
		out = append(out, s[start:])
	}
	return out
}

// ToUpper gives the text in upper case.
func ToUpper(s string) string {
	return dart.Upper(s)
}

// ToLower gives the text in lower case.
func ToLower(s string) string {
	return dart.Lower(s)
}

// EqualFold reports whether s and t are equal in lower case.
func EqualFold(s, t string) bool {
	return dart.Lower(s) == dart.Lower(t)
}

// Cut cuts s at the first sep.
func Cut(s, sep string) (before, after string, found bool) {
	i := Index(s, sep)
	if i < 0 {
		return s, "", false
	}
	return s[:i], s[i+len(sep):], true
}

func isSpace(c byte) bool {
	return c == ' ' || c == '\t' || c == '\n' || c == '\r' || c == '\v' || c == '\f'
}

// A Builder holds the parts of a string.
type Builder struct {
	parts []byte
}

func (b *Builder) WriteString(s string) (int, error) {
	b.parts = append(b.parts, s...)
	return len(s), nil
}

func (b *Builder) WriteByte(c byte) error {
	b.parts = append(b.parts, c)
	return nil
}

func (b *Builder) WriteRune(r rune) (int, error) {
	return b.WriteString(string(r))
}

func (b *Builder) String() string {
	return string(b.parts)
}

func (b *Builder) Len() int {
	return len(b.parts)
}

func (b *Builder) Reset() {
	b.parts = nil
}
