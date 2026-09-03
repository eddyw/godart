//go:build dart

package strings

import (
	"errors"
	"io"
	"iter"
	"unicode"
	"unicode/utf8"
)

// errUnread reports an unread that stands at the start of a Reader.
var errUnread = errors.New("strings.Reader: at beginning of string")

// Clone gives a copy of s. A Dart string holds no shared array, thus the copy
// is the string itself.
func Clone(s string) string { return s }

// ContainsFunc reports whether s holds a rune that f accepts.
func ContainsFunc(s string, f func(rune) bool) bool { return IndexFunc(s, f) >= 0 }

// LastIndexFunc gives the index of the last rune of s that f accepts, and it
// gives -1 for a text that holds no such rune.
func LastIndexFunc(s string, f func(rune) bool) int {
	for i := len(s); i > 0; {
		r, size := utf8.DecodeLastRuneInString(s[:i])
		i -= size
		if f(r) {
			return i
		}
	}
	return -1
}

// LastIndexAny gives the index of the last rune of s that chars holds, and it
// gives -1 for a text that holds no such rune.
func LastIndexAny(s, chars string) int {
	if chars == "" {
		return -1
	}
	return LastIndexFunc(s, func(r rune) bool { return ContainsRune(chars, r) })
}

// TrimFunc drops each rune of the 2 ends of s that f accepts.
func TrimFunc(s string, f func(rune) bool) string {
	return TrimRightFunc(TrimLeftFunc(s, f), f)
}

// CutLast cuts s at the last sep. The 2 parts stand before and after that sep,
// and the flag reports a text that holds sep.
func CutLast(s, sep string) (before, after string, found bool) {
	if i := LastIndex(s, sep); i >= 0 {
		return s[:i], s[i+len(sep):], true
	}
	return s, "", false
}

// SplitAfter cuts s after each sep, thus each part keeps its sep.
func SplitAfter(s, sep string) []string { return SplitAfterN(s, sep, -1) }

// SplitAfterN cuts s after each sep, and it gives a maximum of n parts. A
// negative n holds no limit, and a value of 0 gives nil.
func SplitAfterN(s, sep string, n int) []string {
	if n == 0 {
		return nil
	}
	if sep == "" {
		return explode(s, n)
	}
	out := []string{}
	for n < 0 || len(out) < n-1 {
		i := Index(s, sep)
		if i < 0 {
			break
		}
		out = append(out, s[:i+len(sep)])
		s = s[i+len(sep):]
	}
	return append(out, s)
}

// SplitAfterSeq cuts s after each sep, and it gives each part as a sequence.
func SplitAfterSeq(s, sep string) iter.Seq[string] {
	return func(yield func(string) bool) {
		for {
			i := Index(s, sep)
			if i < 0 || sep == "" {
				yield(s)
				return
			}
			if !yield(s[:i+len(sep)]) {
				return
			}
			s = s[i+len(sep):]
		}
	}
}

// explode cuts s into its runes, and it gives a maximum of n parts.
func explode(s string, n int) []string {
	count := utf8.RuneCountInString(s)
	if n < 0 || n > count {
		n = count
	}
	out := make([]string, n)
	for i := 0; i < n-1; i++ {
		_, size := utf8.DecodeRuneInString(s)
		out[i] = s[:size]
		s = s[size:]
	}
	if n > 0 {
		out[n-1] = s
	}
	return out
}

// FieldsFuncSeq cuts s at each run of runes that f accepts, and it gives each
// part as a sequence.
func FieldsFuncSeq(s string, f func(rune) bool) iter.Seq[string] {
	return func(yield func(string) bool) {
		start := -1
		for i, r := range s {
			if f(r) {
				if start >= 0 {
					if !yield(s[start:i]) {
						return
					}
					start = -1
				}
				continue
			}
			if start < 0 {
				start = i
			}
		}
		if start >= 0 {
			yield(s[start:])
		}
	}
}

// Lines gives each line of s as a sequence. A line keeps its end of line, and
// the last line keeps none when s holds none.
func Lines(s string) iter.Seq[string] {
	return func(yield func(string) bool) {
		for len(s) > 0 {
			i := IndexByte(s, '\n')
			if i < 0 {
				yield(s)
				return
			}
			if !yield(s[:i+1]) {
				return
			}
			s = s[i+1:]
		}
	}
}

// ToTitle gives s with each rune in its title form.
func ToTitle(s string) string { return Map(unicode.ToTitle, s) }

// Title gives s with the first rune of each word in its title form.
//
// Go marks this function as deprecated, because it does not honor the rules of
// a word boundary of Unicode.
func Title(s string) string {
	previous := ' '
	return Map(func(r rune) rune {
		if isSeparator(previous) {
			previous = r
			return unicode.ToTitle(r)
		}
		previous = r
		return r
	}, s)
}

// isSeparator reports a rune that stands between 2 words.
func isSeparator(r rune) bool {
	if r <= 0x7F {
		switch {
		case '0' <= r && r <= '9':
			return false
		case 'a' <= r && r <= 'z':
			return false
		case 'A' <= r && r <= 'Z':
			return false
		case r == '_':
			return false
		}
		return true
	}
	if unicode.IsLetter(r) || unicode.IsDigit(r) {
		return false
	}
	return unicode.IsSpace(r)
}

// ToValidUTF8 replaces each run of bytes that no rune holds with the text of
// replacement.
func ToValidUTF8(s, replacement string) string {
	var b Builder
	invalid := false
	for i := 0; i < len(s); {
		c := s[i]
		if c < utf8.RuneSelf {
			i++
			invalid = false
			b.WriteByte(c)
			continue
		}
		_, size := utf8.DecodeRuneInString(s[i:])
		if size > 1 {
			invalid = false
			b.WriteString(s[i : i+size])
			i += size
			continue
		}
		i++
		if !invalid {
			invalid = true
			b.WriteString(replacement)
		}
	}
	return b.String()
}

// Cap gives the count of bytes that the Builder holds room for. A Dart list
// grows on its own, thus the value is the length.
func (b *Builder) Cap() int { return len(b.parts) }

// UnreadByte gives the last byte of a read back to the Reader.
func (r *Reader) UnreadByte() error {
	if r.pos <= 0 {
		return errUnread
	}
	r.pos--
	return nil
}

// UnreadRune gives the last rune of a read back to the Reader.
func (r *Reader) UnreadRune() error {
	if r.pos <= 0 {
		return errUnread
	}
	_, size := utf8.DecodeLastRuneInString(r.text[:r.pos])
	r.pos -= size
	return nil
}

// WriteTo writes the rest of the text to w.
//
// Go names the type of w io.Writer. The parameter names the same method set,
// and Go holds the 2 types equal.
func (r *Reader) WriteTo(w interface {
	Write(p []byte) (int, error)
}) (int64, error) {
	if r.pos >= len(r.text) {
		return 0, nil
	}
	rest := r.text[r.pos:]
	n, err := w.Write([]byte(rest))
	r.pos += n
	if err == nil && n < len(rest) {
		err = io.ErrShortWrite
	}
	return int64(n), err
}
