//go:build dart

package json

import (
	"bytes"
	"errors"
)

// errEnd marks a value that the buffer does not hold to its end. A Decoder
// reads more bytes and scans again.
var errEnd = errors.New("unexpected end of JSON input")

// scanValue gives the length of the first JSON value of data.
//
// The value starts at the first byte that is not a space. The flag atEOF tells
// the scanner that no more bytes follow, thus a number at the end of the buffer
// is complete.
func scanValue(data []byte, atEOF bool) (int, error) {
	s := &scanner{data: data, atEOF: atEOF}
	if err := s.value(); err != nil {
		return 0, err
	}
	return s.i, nil
}

// Valid reports whether data holds 1 JSON value and nothing else.
func Valid(data []byte) bool {
	n, err := scanValue(data, true)
	if err != nil {
		return false
	}
	for _, c := range data[n:] {
		if !isSpace(c) {
			return false
		}
	}
	return true
}

// Compact writes the JSON value of src into dst with no space between the
// members.
func Compact(dst *bytes.Buffer, src []byte) error {
	return rewrite(dst, src, "", "", false)
}

// Indent writes the JSON value of src into dst with 1 line for each member.
// Each line starts with the prefix, and each level adds the indent.
func Indent(dst *bytes.Buffer, src []byte, prefix, indent string) error {
	return rewrite(dst, src, prefix, indent, true)
}

func rewrite(dst *bytes.Buffer, src []byte, prefix, indent string, pretty bool) error {
	start := dst.Len()
	s := &scanner{
		data:  src,
		atEOF: true,
		out:   &former{dst: dst, prefix: prefix, indent: indent, pretty: pretty},
	}
	if err := s.value(); err != nil {
		dst.Truncate(start)
		return err
	}
	for i, c := range src[s.i:] {
		if !isSpace(c) {
			dst.Truncate(start)
			return &SyntaxError{
				msg:    "invalid character " + quoteChar(c) + " after top-level value",
				Offset: int64(s.i + i + 1),
			}
		}
	}
	return nil
}

// former writes the compact form or the indented form of the value that the
// scanner walks. A scanner with no former reads the value and writes nothing.
type former struct {
	dst    *bytes.Buffer
	prefix string
	indent string
	pretty bool
	depth  int
}

func (f *former) open(c byte) {
	f.dst.WriteByte(c)
	f.depth++
}

// item stands in front of each element of an array and of each member of an
// object.
func (f *former) item(first bool) {
	if !first {
		f.dst.WriteByte(',')
	}
	f.line()
}

func (f *former) close(c byte, empty bool) {
	f.depth--
	if !empty {
		f.line()
	}
	f.dst.WriteByte(c)
}

func (f *former) line() {
	if !f.pretty {
		return
	}
	f.dst.WriteByte('\n')
	f.dst.WriteString(f.prefix)
	for i := 0; i < f.depth; i++ {
		f.dst.WriteString(f.indent)
	}
}

func (f *former) colon() {
	f.dst.WriteByte(':')
	if f.pretty {
		f.dst.WriteByte(' ')
	}
}

func (f *former) raw(b []byte) { f.dst.Write(b) }

// scanner walks 1 JSON value of a buffer. It reports the end of the value, and
// it writes each part into the former when the caller gives one.
type scanner struct {
	data  []byte
	i     int
	atEOF bool
	out   *former
}

func (s *scanner) space() {
	for s.i < len(s.data) && isSpace(s.data[s.i]) {
		s.i++
	}
}

func (s *scanner) value() error {
	s.space()
	if s.i >= len(s.data) {
		return errEnd
	}
	c := s.data[s.i]
	switch {
	case c == '{':
		return s.object()
	case c == '[':
		return s.array()
	case c == '"':
		return s.text()
	case c == 't':
		return s.word("true")
	case c == 'f':
		return s.word("false")
	case c == 'n':
		return s.word("null")
	case c == '-' || (c >= '0' && c <= '9'):
		return s.number()
	}
	return s.fail("looking for beginning of value")
}

func (s *scanner) object() error {
	s.emitOpen('{')
	s.i++
	s.space()
	if s.i >= len(s.data) {
		return errEnd
	}
	if s.data[s.i] == '}' {
		s.i++
		s.emitClose('}', true)
		return nil
	}
	first := true
	for {
		s.space()
		if s.i >= len(s.data) {
			return errEnd
		}
		if s.data[s.i] != '"' {
			return s.fail("looking for beginning of object key string")
		}
		s.emitItem(first)
		first = false
		if err := s.text(); err != nil {
			return err
		}
		s.space()
		if s.i >= len(s.data) {
			return errEnd
		}
		if s.data[s.i] != ':' {
			return s.fail("after object key")
		}
		s.i++
		s.emitColon()
		if err := s.value(); err != nil {
			return err
		}
		s.space()
		if s.i >= len(s.data) {
			return errEnd
		}
		switch s.data[s.i] {
		case ',':
			s.i++
		case '}':
			s.i++
			s.emitClose('}', false)
			return nil
		default:
			return s.fail("after object key:value pair")
		}
	}
}

func (s *scanner) array() error {
	s.emitOpen('[')
	s.i++
	s.space()
	if s.i >= len(s.data) {
		return errEnd
	}
	if s.data[s.i] == ']' {
		s.i++
		s.emitClose(']', true)
		return nil
	}
	first := true
	for {
		s.emitItem(first)
		first = false
		if err := s.value(); err != nil {
			return err
		}
		s.space()
		if s.i >= len(s.data) {
			return errEnd
		}
		switch s.data[s.i] {
		case ',':
			s.i++
		case ']':
			s.i++
			s.emitClose(']', false)
			return nil
		default:
			return s.fail("after array element")
		}
	}
}

func (s *scanner) text() error {
	start := s.i
	s.i++
	for s.i < len(s.data) {
		c := s.data[s.i]
		switch {
		case c == '"':
			s.i++
			s.emitRaw(start)
			return nil
		case c == '\\':
			s.i++
			if s.i >= len(s.data) {
				return errEnd
			}
			if s.data[s.i] == 'u' {
				if s.i+4 >= len(s.data) {
					return errEnd
				}
				for k := 1; k <= 4; k++ {
					if !isHex(s.data[s.i+k]) {
						s.i += k
						return s.fail("in \\u hexadecimal character escape")
					}
				}
				s.i += 4
			} else if !isEscape(s.data[s.i]) {
				return s.fail("in string escape code")
			}
			s.i++
		case c < 0x20:
			return s.fail("in string literal")
		default:
			s.i++
		}
	}
	return errEnd
}

func (s *scanner) word(want string) error {
	start := s.i
	for k := 0; k < len(want); k++ {
		if s.i >= len(s.data) {
			return errEnd
		}
		if s.data[s.i] != want[k] {
			return s.fail("in literal " + want)
		}
		s.i++
	}
	s.emitRaw(start)
	return nil
}

func (s *scanner) number() error {
	start := s.i
	if s.data[s.i] == '-' {
		s.i++
	}
	// The whole part holds no leading zero, thus a run that starts with 0 holds
	// that digit alone. A later 0 ends the value, and the caller reports the
	// next byte.
	if err := s.digits(true); err != nil {
		return err
	}
	if s.i < len(s.data) && s.data[s.i] == '.' {
		s.i++
		if err := s.digits(false); err != nil {
			return err
		}
	}
	if s.i < len(s.data) && (s.data[s.i] == 'e' || s.data[s.i] == 'E') {
		s.i++
		if s.i < len(s.data) && (s.data[s.i] == '+' || s.data[s.i] == '-') {
			s.i++
		}
		if err := s.digits(false); err != nil {
			return err
		}
	}
	s.emitRaw(start)
	return nil
}

// digits reads 1 digit or more. The flag whole marks the part of a number in
// front of the point, where a run that starts with 0 holds that digit alone.
func (s *scanner) digits(whole bool) error {
	if s.i >= len(s.data) {
		return errEnd
	}
	if !isDigit(s.data[s.i]) {
		return s.fail("in numeric literal")
	}
	if whole && s.data[s.i] == '0' {
		s.i++
	} else {
		for s.i < len(s.data) && isDigit(s.data[s.i]) {
			s.i++
		}
	}
	if s.i >= len(s.data) && !s.atEOF {
		return errEnd
	}
	return nil
}

func (s *scanner) fail(where string) error {
	if s.i >= len(s.data) {
		return errEnd
	}
	return &SyntaxError{
		msg:    "invalid character " + quoteChar(s.data[s.i]) + " " + where,
		Offset: int64(s.i) + 1,
	}
}

func (s *scanner) emitOpen(c byte) {
	if s.out != nil {
		s.out.open(c)
	}
}

func (s *scanner) emitClose(c byte, empty bool) {
	if s.out != nil {
		s.out.close(c, empty)
	}
}

func (s *scanner) emitItem(first bool) {
	if s.out != nil {
		s.out.item(first)
	}
}

func (s *scanner) emitColon() {
	if s.out != nil {
		s.out.colon()
	}
}

func (s *scanner) emitRaw(start int) {
	if s.out != nil {
		s.out.raw(s.data[start:s.i])
	}
}

func isSpace(c byte) bool {
	return c == ' ' || c == '\t' || c == '\n' || c == '\r'
}

func isDigit(c byte) bool { return c >= '0' && c <= '9' }

func isHex(c byte) bool {
	return isDigit(c) || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')
}

func isEscape(c byte) bool {
	switch c {
	case '"', '\\', '/', 'b', 'f', 'n', 'r', 't':
		return true
	}
	return false
}

// quoteChar writes a byte the way the error text of Go writes it.
func quoteChar(c byte) string {
	if c == '\'' {
		return `'\''`
	}
	if c == '"' {
		return `'"'`
	}
	if c < 0x20 || c >= 0x7f {
		return `'\x` + hexPair(c) + `'`
	}
	return "'" + string(rune(c)) + "'"
}

func hexPair(c byte) string {
	const digits = "0123456789abcdef"
	return string(rune(digits[c>>4])) + string(rune(digits[c&0xf]))
}
