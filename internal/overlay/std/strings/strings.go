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

// TrimLeft removes each leading rune of the set.
func TrimLeft(s, cutset string) string {
	for len(s) > 0 {
		r, size := decodeFirst(s)
		if IndexRune(cutset, r) < 0 {
			break
		}
		s = s[size:]
	}
	return s
}

// TrimRight removes each trailing rune of the set.
func TrimRight(s, cutset string) string {
	for len(s) > 0 {
		r, size := decodeLast(s)
		if IndexRune(cutset, r) < 0 {
			break
		}
		s = s[:len(s)-size]
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

// SplitSeq cuts s at each sep, and it gives each part as a sequence. An empty
// sep gives each rune as its own part.
func SplitSeq(s, sep string) iter.Seq[string] {
	return func(yield func(string) bool) {
		if len(sep) == 0 {
			for i := 0; i < len(s); {
				_, width := utf8.DecodeRuneInString(s[i:])
				part := s[i : i+width]
				i += width
				if !yield(part) {
					return
				}
			}
			return
		}
		width := len(sep)
		first := sep[0]
		start := 0
		i := 0
		for i+width <= len(s) {
			if s[i] != first || !matchAt(s, sep, i) {
				i++
				continue
			}
			if !yield(s[start:i]) {
				return
			}
			i += width
			start = i
		}
		yield(s[start:])
	}
}

// FieldsSeq cuts s at each run of space, and it gives each field as a
// sequence.
func FieldsSeq(s string) iter.Seq[string] {
	return func(yield func(string) bool) {
		start := -1
		for i := 0; i < len(s); i++ {
			if isSpace(s[i]) {
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
// decodeFirst reads the first rune of the text with its size in bytes.
func decodeFirst(s string) (rune, int) {
	for _, c := range s {
		return c, len(string(c))
	}
	return 0, 0
}

// decodeLast reads the last rune of the text with its size in bytes.
func decodeLast(s string) (rune, int) {
	last := rune(0)
	size := 0
	for _, c := range s {
		last = c
		size = len(string(c))
	}
	return last, size
}

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

// ToLowerSpecial gives the text in lower case, with the rules of the case
// range list.
func ToLowerSpecial(c unicode.SpecialCase, s string) string {
	return mapRunes(c.ToLower, s)
}

// ToUpperSpecial gives the text in upper case, with the rules of the case range
// list.
func ToUpperSpecial(c unicode.SpecialCase, s string) string {
	return mapRunes(c.ToUpper, s)
}

// ToTitleSpecial gives the text in title case, with the rules of the case range
// list.
func ToTitleSpecial(c unicode.SpecialCase, s string) string {
	return mapRunes(c.ToTitle, s)
}

// Map gives the text with f on each rune. A negative result drops the rune.
func Map(f func(rune) rune, s string) string {
	out := ""
	for _, r := range s {
		if v := f(r); v >= 0 {
			out += string(v)
		}
	}
	return out
}

func mapRunes(f func(rune) rune, s string) string {
	out := ""
	for _, r := range s {
		out += string(f(r))
	}
	return out
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

// Write adds the bytes. The method makes a Builder an io.Writer.
func (b *Builder) Write(p []byte) (int, error) {
	b.parts = append(b.parts, p...)
	return len(p), nil
}

// Grow reserves room for n more bytes.
//
// A Dart list grows on its own, and each growth copies the whole storage. A
// caller that holds the size of the result thus takes 1 allocation here.
func (b *Builder) Grow(n int) {
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

// Reader reads a text. It holds an io.Reader, an io.ByteReader, an
// io.RuneReader, and an io.Seeker.
type Reader struct {
	text string
	// pos is the offset of the next read, in bytes.
	pos int
}

// NewReader gives a Reader of the text.
func NewReader(s string) *Reader { return &Reader{text: s} }

// Len gives the count of bytes that a read still gives.
func (r *Reader) Len() int {
	if r.pos >= len(r.text) {
		return 0
	}
	return len(r.text) - r.pos
}

// Size gives the count of bytes of the whole text.
func (r *Reader) Size() int64 { return int64(len(r.text)) }

// Read reads bytes into the slice. It gives the error of the end of the text
// when the text holds no more byte.
func (r *Reader) Read(p []byte) (int, error) {
	if r.pos >= len(r.text) {
		return 0, io.EOF
	}
	n := copy(p, r.text[r.pos:])
	r.pos += n
	return n, nil
}

// ReadByte reads 1 byte.
func (r *Reader) ReadByte() (byte, error) {
	if r.pos >= len(r.text) {
		return 0, io.EOF
	}
	c := r.text[r.pos]
	r.pos++
	return c, nil
}

// ReadRune reads 1 rune with its size in bytes.
func (r *Reader) ReadRune() (rune, int, error) {
	if r.pos >= len(r.text) {
		return 0, 0, io.EOF
	}
	for _, c := range r.text[r.pos:] {
		size := len(string(c))
		r.pos += size
		return c, size, nil
	}
	return 0, 0, io.EOF
}

// Reset gives the Reader a new text.
// ReadAt reads at the offset. It does not move the read position of the
// reader.
func (r *Reader) ReadAt(p []byte, off int64) (int, error) {
	if off < 0 {
		return 0, errors.New("strings.Reader.ReadAt: negative offset")
	}
	if off >= int64(len(r.text)) {
		return 0, io.EOF
	}
	n := copy(p, r.text[off:])
	if n < len(p) {
		return n, io.EOF
	}
	return n, nil
}

func (r *Reader) Reset(s string) {
	r.text = s
	r.pos = 0
}

// Seek moves the read position and gives the new offset from the start. The
// whence holds 0 for the start, 1 for the position, and 2 for the end.
func (r *Reader) Seek(offset int64, whence int) (int64, error) {
	target := offset
	switch whence {
	case 1:
		target = int64(r.pos) + offset
	case 2:
		target = int64(len(r.text)) + offset
	}
	if target < 0 {
		return 0, errNegative
	}
	r.pos = int(target)
	return target, nil
}

var errNegative = errors.New("strings.Reader.Seek: negative position")
