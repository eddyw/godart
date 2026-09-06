//go:build dart

// Package strings gives the operations on a Go string.
package strings

import "errors"

import (
	"io"
	"iter"
	"unicode"
	"unicode/utf8"
)

// Index gives the position of the first substr in s, or -1.
//
// The scan reads the bytes. A comparison of 2 substrings builds 1 Go string for
// each position of the text, and each build allocates a view and a header.
func Index(s, substr string) int {
	n := len(substr)
	if n == 0 {
		return 0
	}
	if n > len(s) {
		return -1
	}
	first := substr[0]
	for i := 0; i+n <= len(s); i++ {
		if s[i] == first && matchAt(s, substr, i) {
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
	if n > len(s) {
		return -1
	}
	first := substr[0]
	for i := len(s) - n; i >= 0; i-- {
		if s[i] == first && matchAt(s, substr, i) {
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

// LastIndexByte gives the position of the last c in s, or -1.
func LastIndexByte(s string, c byte) int {
	for i := len(s) - 1; i >= 0; i-- {
		if s[i] == c {
			return i
		}
	}
	return -1
}

// Contains reports whether substr is inside s.
// Compare gives a negative value, 0, or a positive value. It orders by byte.
//
// The 2 operators < and > give the same result. The function exists because a
// caller needs a value of the type func(a, b string) int.
func Compare(a, b string) int {
	if a < b {
		return -1
	}
	if a > b {
		return 1
	}
	return 0
}

// IndexRune gives the index of the first rune r, in bytes. The result is -1
// when the text holds no such rune.
func IndexRune(s string, r rune) int {
	for i, c := range s {
		if c == r {
			return i
		}
	}
	return -1
}

// ContainsAny reports whether the text holds 1 rune of the set.
func ContainsAny(s, chars string) bool {
	for _, c := range s {
		if IndexRune(chars, c) >= 0 {
			return true
		}
	}
	return false
}

// IndexAny gives the index of the first rune of the set, in bytes.
func IndexAny(s, chars string) int {
	for i, c := range s {
		if IndexRune(chars, c) >= 0 {
			return i
		}
	}
	return -1
}

// asciiSet is a 256-byte lookup table for fast ASCII character membership testing.
// Each element corresponds to an ASCII character value, with true indicating the
// character is in the set. Using bool instead of byte allows the compiler to
// eliminate the comparison instruction, as bool values are guaranteed to be 0 or 1.
//
// The full 256-element table is used rather than a 128-element table to avoid
// additional operations in the lookup path. Alternative approaches were tested:
//   - [128]bool with explicit bounds check (if c >= 128): introduces branches
//     that cause pipeline stalls, resulting in ~70% slower performance
//   - [128]bool with masking (c&0x7f): eliminates bounds checks but the AND
//     operation still costs ~10% performance compared to direct indexing
//
// The 256-element array allows direct indexing with no bounds checks, no branches,
// and no masking operations, providing optimal performance. The additional 128 bytes
// of memory is a worthwhile tradeoff for the simpler, faster code.
type asciiSet [256]bool

// makeASCIISet creates a set of ASCII characters and reports whether all
// characters in chars are ASCII.
func makeASCIISet(chars string) (as asciiSet, ok bool) {
	for i := 0; i < len(chars); i++ {
		c := chars[i]
		if c >= utf8.RuneSelf {
			return as, false
		}
		as[c] = true
	}
	return as, true
}

// contains reports whether c is inside the set.
func (as *asciiSet) contains(c byte) bool {
	return as[c]
}

// Trim returns a slice of the string s with all leading and
// trailing Unicode code points contained in cutset removed.
func Trim(s, cutset string) string {
	if s == "" || cutset == "" {
		return s
	}
	if len(cutset) == 1 && cutset[0] < utf8.RuneSelf {
		return trimLeftByte(trimRightByte(s, cutset[0]), cutset[0])
	}
	if as, ok := makeASCIISet(cutset); ok {
		return trimLeftASCII(trimRightASCII(s, &as), &as)
	}
	return trimLeftUnicode(trimRightUnicode(s, cutset), cutset)
}

// TrimLeft returns a slice of the string s with all leading
// Unicode code points contained in cutset removed.
//
// To remove a prefix, use [TrimPrefix] instead.
func TrimLeft(s, cutset string) string {
	if s == "" || cutset == "" {
		return s
	}
	if len(cutset) == 1 && cutset[0] < utf8.RuneSelf {
		return trimLeftByte(s, cutset[0])
	}
	if as, ok := makeASCIISet(cutset); ok {
		return trimLeftASCII(s, &as)
	}
	return trimLeftUnicode(s, cutset)
}

func trimLeftByte(s string, c byte) string {
	for len(s) > 0 && s[0] == c {
		s = s[1:]
	}
	return s
}

func trimLeftASCII(s string, as *asciiSet) string {
	for len(s) > 0 {
		if !as.contains(s[0]) {
			break
		}
		s = s[1:]
	}
	return s
}

func trimLeftUnicode(s, cutset string) string {
	for len(s) > 0 {
		r, n := utf8.DecodeRuneInString(s)
		if !ContainsRune(cutset, r) {
			break
		}
		s = s[n:]
	}
	return s
}

// TrimRight returns a slice of the string s, with all trailing
// Unicode code points contained in cutset removed.
//
// To remove a suffix, use [TrimSuffix] instead.
func TrimRight(s, cutset string) string {
	if s == "" || cutset == "" {
		return s
	}
	if len(cutset) == 1 && cutset[0] < utf8.RuneSelf {
		return trimRightByte(s, cutset[0])
	}
	if as, ok := makeASCIISet(cutset); ok {
		return trimRightASCII(s, &as)
	}
	return trimRightUnicode(s, cutset)
}

func trimRightByte(s string, c byte) string {
	for len(s) > 0 && s[len(s)-1] == c {
		s = s[:len(s)-1]
	}
	return s
}

func trimRightASCII(s string, as *asciiSet) string {
	for len(s) > 0 {
		if !as.contains(s[len(s)-1]) {
			break
		}
		s = s[:len(s)-1]
	}
	return s
}

func trimRightUnicode(s, cutset string) string {
	for len(s) > 0 {
		r, n := rune(s[len(s)-1]), 1
		if r >= utf8.RuneSelf {
			r, n = utf8.DecodeLastRuneInString(s)
		}
		if !ContainsRune(cutset, r) {
			break
		}
		s = s[:len(s)-n]
	}
	return s
}

// CutPrefix removes the prefix and reports whether the text held it.
func CutPrefix(s, prefix string) (string, bool) {
	if !HasPrefix(s, prefix) {
		return s, false
	}
	return s[len(prefix):], true
}

// CutSuffix removes the suffix and reports whether the text held it.
func CutSuffix(s, suffix string) (string, bool) {
	if !HasSuffix(s, suffix) {
		return s, false
	}
	return s[:len(s)-len(suffix)], true
}

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
	width := len(substr)
	first := substr[0]
	for i+width <= len(s) {
		if s[i] == first && matchAt(s, substr, i) {
			n++
			i += width
			continue
		}
		i++
	}
	return n
}

// Repeat gives s a count of times.
//
// A concatenation in a loop builds a fresh text at each step, thus the cost
// stands at the square of the result. The build below doubles the text, and it
// reaches the length in a logarithm of the count. TestUnmarshalMaxDepth of
// encoding/json asks for 3,000,000 parts.
func Repeat(s string, count int) string {
	if count < 0 {
		panic("strings: negative Repeat count")
	}
	if count == 0 || len(s) == 0 {
		return ""
	}
	total := len(s) * count
	if total/count != len(s) {
		panic("strings: Repeat output length overflow")
	}
	out := make([]byte, 0, total)
	out = append(out, s...)
	for len(out) < total {
		n := len(out)
		if n > total-n {
			n = total - n
		}
		out = append(out, out[:n]...)
	}
	return string(out)
}

// Join puts sep between each element of elems.
//
// The call reads the length of the result first, and Grow then holds the whole
// text in 1 buffer. A concatenation of 1 element at a time builds a fresh text
// at each step, thus its cost stands at the square of the result: 2,000
// elements took 37.5 milliseconds and they take 0.9 now. That form costs 17
// percent more for the 6 elements of the case str.join.100k of bench, and a
// library must hold no square cost. Go itself holds this form.
func Join(elems []string, sep string) string {
	switch len(elems) {
	case 0:
		return ""
	case 1:
		return elems[0]
	}
	n := len(sep) * (len(elems) - 1)
	for _, elem := range elems {
		n += len(elem)
	}
	var b Builder
	b.Grow(n)
	b.WriteString(elems[0])
	for _, s := range elems[1:] {
		b.WriteString(sep)
		b.WriteString(s)
	}
	return b.String()
}

// matchAt reports whether sep stands in s at the position i. The caller must
// hold i plus the length of sep inside s.
//
// A comparison of 2 substrings builds 1 Go string for each read, and a scan of
// a text reads each position. matchAt thus reads the bytes and it allocates
// nothing.
func matchAt(s, sep string, i int) bool {
	for k := 0; k < len(sep); k++ {
		if s[i+k] != sep[k] {
			return false
		}
	}
	return true
}

// Split cuts s at each sep.
func Split(s, sep string) []string {
	if len(sep) == 0 {
		// Go slices s at each rune boundary, thus a byte that holds no rune
		// stays as it is. A build from the rune would give the replacement
		// character.
		out := []string{}
		for i := 0; i < len(s); {
			_, width := utf8.DecodeRuneInString(s[i:])
			out = append(out, s[i:i+width])
			i += width
		}
		return out
	}
	if len(sep) > len(s) {
		return []string{s}
	}
	// The count of the parts stands in front of the walk, thus the storage of
	// the result takes 1 allocation. A growth of the slice copies each part
	// again, and a line of 6 parts took 4 such copies.
	out := make([]string, 0, Count(s, sep)+1)
	width := len(sep)
	first := sep[0]
	start := 0
	i := 0
	for i+width <= len(s) {
		if s[i] != first || !matchAt(s, sep, i) {
			i++
			continue
		}
		out = append(out, s[start:i])
		i += width
		start = i
	}
	return append(out, s[start:])
}

// splitSeq is SplitSeq or SplitAfterSeq, configured by how many
// bytes of sep to include in the results (none or all).
func splitSeq(s, sep string, sepSave int) iter.Seq[string] {
	return func(yield func(string) bool) {
		if len(sep) == 0 {
			for len(s) > 0 {
				_, size := utf8.DecodeRuneInString(s)
				if !yield(s[:size]) {
					return
				}
				s = s[size:]
			}
			return
		}
		for {
			i := Index(s, sep)
			if i < 0 {
				break
			}
			frag := s[:i+sepSave]
			if !yield(frag) {
				return
			}
			s = s[i+len(sep):]
		}
		yield(s)
	}
}

// SplitSeq returns an iterator over all substrings of s separated by sep.
// The iterator yields the same strings that would be returned by [Split](s, sep),
// but without constructing the slice.
// It returns a single-use iterator.
func SplitSeq(s, sep string) iter.Seq[string] {
	return splitSeq(s, sep, 0)
}

// SplitAfterSeq returns an iterator over substrings of s split after each instance of sep.
// The iterator yields the same strings that would be returned by [SplitAfter](s, sep),
// but without constructing the slice.
// It returns a single-use iterator.
func SplitAfterSeq(s, sep string) iter.Seq[string] {
	return splitSeq(s, sep, len(sep))
}

// FieldsSeq returns an iterator over substrings of s split around runs of
// whitespace characters, as defined by [unicode.IsSpace].
// The iterator yields the same strings that would be returned by [Fields](s),
// but without constructing the slice.
func FieldsSeq(s string) iter.Seq[string] {
	return func(yield func(string) bool) {
		start := -1
		for i := 0; i < len(s); {
			size := 1
			r := rune(s[i])
			isSpace := asciiSpace[s[i]] != 0
			if r >= utf8.RuneSelf {
				r, size = utf8.DecodeRuneInString(s[i:])
				isSpace = unicode.IsSpace(r)
			}
			if isSpace {
				if start >= 0 {
					if !yield(s[start:i]) {
						return
					}
					start = -1
				}
			} else if start < 0 {
				start = i
			}
			i += size
		}
		if start >= 0 {
			yield(s[start:])
		}
	}
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
	if n == 0 {
		return s
	}
	if len(old) == 0 {
		// An empty old stands at the front of s and after each rune, thus a
		// text of k runes takes up to k plus 1 replacements.
		most := utf8.RuneCountInString(s) + 1
		if n < 0 || n > most {
			n = most
		}
		var out Builder
		out.WriteString(new)
		count := 1
		i := 0
		for i < len(s) {
			_, width := utf8.DecodeRuneInString(s[i:])
			out.WriteString(s[i : i+width])
			i += width
			if count < n {
				out.WriteString(new)
				count++
			}
		}
		return out.String()
	}
	if len(old) > len(s) {
		return s
	}
	// The count of the replacements gives the size of the result, thus the
	// builder takes 1 allocation. A growth of the storage copies each byte
	// again, and a text of 45 bytes took 6 such copies.
	most := Count(s, old)
	if n > 0 && n < most {
		most = n
	}
	if most == 0 {
		return s
	}
	var out Builder
	out.Grow(len(s) + most*(len(new)-len(old)))
	width := len(old)
	first := old[0]
	count := 0
	i := 0
	start := 0
	for i+width <= len(s) {
		if s[i] != first || !matchAt(s, old, i) {
			i++
			continue
		}
		out.WriteString(s[start:i])
		out.WriteString(new)
		i += width
		start = i
		count++
		if count == most {
			break
		}
	}
	out.WriteString(s[start:])
	return out.String()
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

// IndexFunc gives the position of the first rune that satisfies f, or -1.
func IndexFunc(s string, f func(rune) bool) int {
	for i, c := range s {
		if f(c) {
			return i
		}
	}
	return -1
}

// TrimLeftFunc removes each leading rune that satisfies f.
func TrimLeftFunc(s string, f func(rune) bool) string {
	for i, c := range s {
		if !f(c) {
			return s[i:]
		}
	}
	return ""
}

// TrimRightFunc removes each trailing rune that satisfies f.
func TrimRightFunc(s string, f func(rune) bool) string {
	end := len(s)
	for end > 0 {
		start := end - 1
		for start > 0 && s[start]&0xC0 == 0x80 {
			start--
		}
		c, _ := decodeFirst(s[start:end])
		if !f(c) {
			break
		}
		end = start
	}
	return s[:end]
}

var asciiSpace = [256]uint8{'\t': 1, '\n': 1, '\v': 1, '\f': 1, '\r': 1, ' ': 1}

// Fields splits the string s around each instance of one or more consecutive white space
// characters, as defined by [unicode.IsSpace], returning a slice of substrings of s or an
// empty slice if s contains only white space. Every element of the returned slice is
// non-empty. Unlike [Split], leading and trailing runs of white space characters
// are discarded.
func Fields(s string) []string {
	// First count the fields.
	// This is an exact count if s is ASCII, otherwise it is an approximation.
	n := 0
	wasSpace := 1
	// setBits is used to track which bits are set in the bytes of s.
	setBits := uint8(0)
	for i := 0; i < len(s); i++ {
		r := s[i]
		setBits |= r
		isSpace := int(asciiSpace[r])
		n += wasSpace & ^isSpace
		wasSpace = isSpace
	}

	if setBits >= utf8.RuneSelf {
		// Some runes in the input string are not ASCII.
		return FieldsFunc(s, unicode.IsSpace)
	}
	// ASCII fast path
	a := make([]string, n)
	na := 0
	fieldStart := 0
	i := 0
	// Skip spaces in the front of the input.
	for i < len(s) && asciiSpace[s[i]] != 0 {
		i++
	}
	fieldStart = i
	for i < len(s) {
		if asciiSpace[s[i]] == 0 {
			i++
			continue
		}
		a[na] = s[fieldStart:i]
		na++
		i++
		// Skip spaces in between fields.
		for i < len(s) && asciiSpace[s[i]] != 0 {
			i++
		}
		fieldStart = i
	}
	if fieldStart < len(s) { // Last field might end at EOF.
		a[na] = s[fieldStart:]
	}
	return a
}

// TrimSpace returns a slice (substring) of the string s,
// with all leading and trailing white space removed,
// as defined by Unicode.
func TrimSpace(s string) string {
	// Fast path for ASCII: look for the first ASCII non-space byte.
	for lo, c := range []byte(s) {
		if c >= utf8.RuneSelf {
			// If we run into a non-ASCII byte, fall back to the
			// slower unicode-aware method on the remaining bytes.
			return TrimFunc(s[lo:], unicode.IsSpace)
		}
		if asciiSpace[c] != 0 {
			continue
		}
		s = s[lo:]
		// Now look for the first ASCII non-space byte from the end.
		for hi := len(s) - 1; hi >= 0; hi-- {
			c := s[hi]
			if c >= utf8.RuneSelf {
				return TrimRightFunc(s[:hi+1], unicode.IsSpace)
			}
			if asciiSpace[c] == 0 {
				// At this point, s[:hi+1] starts and ends with ASCII
				// non-space bytes, so we're done. Non-ASCII cases have
				// already been handled above.
				return s[:hi+1]
			}
		}
	}
	return ""
}

// decodeFirst reads the first rune of the text with its size in bytes.
func decodeFirst(s string) (rune, int) {
	for _, c := range s {
		return c, len(string(c))
	}
	return 0, 0
}

// FieldsFunc cuts s at each run of runes that f accepts. It gives the runs of
// runes that f rejects.
func FieldsFunc(s string, f func(rune) bool) []string {
	out := []string{}
	start := -1
	for i, r := range s {
		if f(r) {
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
	isASCII, hasLower := true, false
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c >= utf8.RuneSelf {
			isASCII = false
			break
		}
		hasLower = hasLower || ('a' <= c && c <= 'z')
	}
	if isASCII {
		if !hasLower {
			return s
		}
		b := make([]byte, len(s))
		for i := 0; i < len(s); i++ {
			c := s[i]
			if 'a' <= c && c <= 'z' {
				c -= 'a' - 'A'
			}
			b[i] = c
		}
		return string(b)
	}
	return Map(unicode.ToUpper, s)
}

// ToLower gives the text in lower case.
func ToLower(s string) string {
	isASCII, hasUpper := true, false
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c >= utf8.RuneSelf {
			isASCII = false
			break
		}
		hasUpper = hasUpper || ('A' <= c && c <= 'Z')
	}
	if isASCII {
		if !hasUpper {
			return s
		}
		b := make([]byte, len(s))
		for i := 0; i < len(s); i++ {
			c := s[i]
			if 'A' <= c && c <= 'Z' {
				c += 'a' - 'A'
			}
			b[i] = c
		}
		return string(b)
	}
	return Map(unicode.ToLower, s)
}

// ToLowerSpecial returns a copy of the string s with all Unicode letters mapped to their
// lower case using the case mapping specified by c.
func ToLowerSpecial(c unicode.SpecialCase, s string) string {
	return Map(c.ToLower, s)
}

// ToUpperSpecial returns a copy of the string s with all Unicode letters mapped to their
// upper case using the case mapping specified by c.
func ToUpperSpecial(c unicode.SpecialCase, s string) string {
	return Map(c.ToUpper, s)
}

// ToTitleSpecial returns a copy of the string s with all Unicode letters mapped to their
// Unicode title case, giving priority to the special casing rules.
func ToTitleSpecial(c unicode.SpecialCase, s string) string {
	return Map(c.ToTitle, s)
}

// Map returns a copy of the string s with all its characters modified
// according to the mapping function. If mapping returns a negative value, the character is
// dropped from the string with no replacement.
func Map(mapping func(rune) rune, s string) string {
	// In the worst case, the string can grow when mapped, making
	// things unpleasant. But it's so rare we barge in assuming it's
	// fine. It could also shrink but that falls out naturally.

	// The output buffer b is initialized on demand, the first
	// time a character differs.
	var b Builder

	for i, c := range s {
		r := mapping(c)
		if r == c && c != utf8.RuneError {
			continue
		}

		var width int
		if c == utf8.RuneError {
			c, width = utf8.DecodeRuneInString(s[i:])
			if width != 1 && r == c {
				continue
			}
		} else {
			width = utf8.RuneLen(c)
		}

		b.Grow(len(s) + utf8.UTFMax)
		b.WriteString(s[:i])
		if r >= 0 {
			b.WriteRune(r)
		}

		s = s[i+width:]
		break
	}

	// Fast path for unchanged input
	if b.Cap() == 0 { // didn't call b.Grow above
		return s
	}

	for _, c := range s {
		r := mapping(c)

		if r >= 0 {
			// common case
			// Due to inlining, it is more performant to determine if WriteByte should be
			// invoked rather than always call WriteRune
			if r < utf8.RuneSelf {
				b.WriteByte(byte(r))
			} else {
				// r is not an ASCII rune.
				b.WriteRune(r)
			}
		}
	}

	return b.String()
}

// EqualFold reports whether s and t are equal under simple Unicode case
// folding.
//
// A lower case of the 2 texts gives a different answer. Simple folding relates
// the long s to s, the final sigma to sigma, and the theta symbol to theta, and
// no case mapping does.
func EqualFold(s, t string) bool {
	i := 0
	wide := -1
	for i < len(s) && i < len(t) {
		sr := s[i]
		tr := t[i]
		if sr|tr >= utf8.RuneSelf {
			wide = i
			break
		}
		if tr == sr {
			i++
			continue
		}
		if tr < sr {
			tr, sr = sr, tr
		}
		if 'A' <= sr && sr <= 'Z' && tr == sr+'a'-'A' {
			i++
			continue
		}
		return false
	}
	if wide < 0 {
		return len(s) == len(t)
	}
	return foldTail(s[wide:], t[wide:])
}

// foldTail reads the rest of the 2 texts rune by rune. EqualFold reads the
// ASCII part first, because that part needs no decode.
func foldTail(s, t string) bool {
	for _, sr := range s {
		if len(t) == 0 {
			return false
		}
		var tr rune
		if t[0] < utf8.RuneSelf {
			tr, t = rune(t[0]), t[1:]
		} else {
			r, size := utf8.DecodeRuneInString(t)
			tr, t = r, t[size:]
		}
		if tr == sr {
			continue
		}
		if tr < sr {
			tr, sr = sr, tr
		}
		if tr < utf8.RuneSelf {
			if 'A' <= sr && sr <= 'Z' && tr == sr+'a'-'A' {
				continue
			}
			return false
		}
		// The fold of a rune is a cycle. The walk stops at the start again, or
		// at a rune that is not below the larger 1.
		r := unicode.SimpleFold(sr)
		for r != sr && r < tr {
			r = unicode.SimpleFold(r)
		}
		if r == tr {
			continue
		}
		return false
	}
	return len(t) == 0
}

// Cut cuts s at the first sep.
func Cut(s, sep string) (before, after string, found bool) {
	i := Index(s, sep)
	if i < 0 {
		return s, "", false
	}
	return s[:i], s[i+len(sep):], true
}

// A Builder holds the parts of a string.
type Builder struct {
	// addr holds the receiver of the first write. A copy of a Builder by
	// value keeps the address of the source, thus a write through the copy
	// panics, exactly as in Go.
	addr  *Builder
	parts []byte
}

func (b *Builder) copyCheck() {
	if b.addr == nil {
		b.addr = b
	} else if b.addr != b {
		panic("strings: illegal use of non-zero Builder copied by value")
	}
}

func (b *Builder) WriteString(s string) (int, error) {
	b.copyCheck()
	b.parts = append(b.parts, s...)
	return len(s), nil
}

// Write adds the bytes. The method makes a Builder an io.Writer.
func (b *Builder) Write(p []byte) (int, error) {
	b.copyCheck()
	b.parts = append(b.parts, p...)
	return len(p), nil
}

// Grow reserves room for n more bytes.
//
// A Dart list grows on its own, and each growth copies the whole storage. A
// caller that holds the size of the result thus takes 1 allocation here.
func (b *Builder) Grow(n int) {
	b.copyCheck()
	if n < 0 {
		panic("strings.Builder.Grow: negative count")
	}
	if cap(b.parts)-len(b.parts) >= n {
		return
	}
	next := make([]byte, len(b.parts), 2*cap(b.parts)+n)
	copy(next, b.parts)
	b.parts = next
}

func (b *Builder) WriteByte(c byte) error {
	b.copyCheck()
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

// Cap gives the count of bytes that the Builder holds room for.
func (b *Builder) Cap() int { return cap(b.parts) }

func (b *Builder) Reset() {
	b.addr = nil
	b.parts = nil
}

// A Reader implements the [io.Reader], [io.ReaderAt], [io.ByteReader], [io.ByteScanner],
// [io.RuneReader], [io.RuneScanner], [io.Seeker], and [io.WriterTo] interfaces by reading
// from a string.
// The zero value for Reader operates like a Reader of an empty string.
type Reader struct {
	s        string
	i        int64 // current reading index
	prevRune int   // index of previous rune; or < 0
}

// Len returns the number of bytes of the unread portion of the
// string.
func (r *Reader) Len() int {
	if r.i >= int64(len(r.s)) {
		return 0
	}
	return int(int64(len(r.s)) - r.i)
}

// Size returns the original length of the underlying string.
// Size is the number of bytes available for reading via [Reader.ReadAt].
// The returned value is always the same and is not affected by calls
// to any other method.
func (r *Reader) Size() int64 { return int64(len(r.s)) }

// Read implements the [io.Reader] interface.
func (r *Reader) Read(b []byte) (n int, err error) {
	if r.i >= int64(len(r.s)) {
		return 0, io.EOF
	}
	r.prevRune = -1
	n = copy(b, r.s[r.i:])
	r.i += int64(n)
	return
}

// ReadAt implements the [io.ReaderAt] interface.
func (r *Reader) ReadAt(b []byte, off int64) (n int, err error) {
	// cannot modify state - see io.ReaderAt
	if off < 0 {
		return 0, errors.New("strings.Reader.ReadAt: negative offset")
	}
	if off >= int64(len(r.s)) {
		return 0, io.EOF
	}
	n = copy(b, r.s[off:])
	if n < len(b) {
		err = io.EOF
	}
	return
}

// ReadByte implements the [io.ByteReader] interface.
func (r *Reader) ReadByte() (byte, error) {
	r.prevRune = -1
	if r.i >= int64(len(r.s)) {
		return 0, io.EOF
	}
	b := r.s[r.i]
	r.i++
	return b, nil
}

// UnreadByte implements the [io.ByteScanner] interface.
func (r *Reader) UnreadByte() error {
	if r.i <= 0 {
		return errors.New("strings.Reader.UnreadByte: at beginning of string")
	}
	r.prevRune = -1
	r.i--
	return nil
}

// ReadRune implements the [io.RuneReader] interface.
func (r *Reader) ReadRune() (ch rune, size int, err error) {
	if r.i >= int64(len(r.s)) {
		r.prevRune = -1
		return 0, 0, io.EOF
	}
	r.prevRune = int(r.i)
	ch, size = utf8.DecodeRuneInString(r.s[r.i:])
	r.i += int64(size)
	return
}

// UnreadRune implements the [io.RuneScanner] interface.
func (r *Reader) UnreadRune() error {
	if r.i <= 0 {
		return errors.New("strings.Reader.UnreadRune: at beginning of string")
	}
	if r.prevRune < 0 {
		return errors.New("strings.Reader.UnreadRune: previous operation was not ReadRune")
	}
	r.i = int64(r.prevRune)
	r.prevRune = -1
	return nil
}

// Seek implements the [io.Seeker] interface.
func (r *Reader) Seek(offset int64, whence int) (int64, error) {
	r.prevRune = -1
	var abs int64
	switch whence {
	case io.SeekStart:
		abs = offset
	case io.SeekCurrent:
		abs = r.i + offset
	case io.SeekEnd:
		abs = int64(len(r.s)) + offset
	default:
		return 0, errors.New("strings.Reader.Seek: invalid whence")
	}
	if abs < 0 {
		return 0, errors.New("strings.Reader.Seek: negative position")
	}
	r.i = abs
	return abs, nil
}

// WriteTo implements the [io.WriterTo] interface.
func (r *Reader) WriteTo(w io.Writer) (n int64, err error) {
	r.prevRune = -1
	if r.i >= int64(len(r.s)) {
		return 0, nil
	}
	s := r.s[r.i:]
	m, err := io.WriteString(w, s)
	if m > len(s) {
		panic("strings.Reader.WriteTo: invalid WriteString count")
	}
	r.i += int64(m)
	n = int64(m)
	if m != len(s) && err == nil {
		err = io.ErrShortWrite
	}
	return
}

// Reset resets the [Reader] to be reading from s.
func (r *Reader) Reset(s string) { *r = Reader{s, 0, -1} }

// NewReader returns a new [Reader] reading from s.
// It is similar to [bytes.NewBufferString] but more efficient and non-writable.
func NewReader(s string) *Reader { return &Reader{s, 0, -1} }
