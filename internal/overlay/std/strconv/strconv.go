//go:build dart

// Package strconv converts between a Go string and a basic value.
package strconv

import (
	"errors"
	"syscall/dart"
	"unicode/utf8"
)

// IntSize is the count of bits of an int. A Dart int holds 64 bits on the VM
// and with AOT.
const IntSize = 64

// ErrSyntax reports text that holds no value of the type.
var ErrSyntax = errors.New("invalid syntax")

// ErrRange reports a value outside the range of the type.
var ErrRange = errors.New("value out of range")

// A NumError names the function, the text, and the reason of a failure.
type NumError struct {
	Func string
	Num  string
	Err  error
}

func (e *NumError) Error() string {
	return "strconv." + e.Func + ": parsing " + quote(e.Num) + ": " + e.Err.Error()
}

func (e *NumError) Unwrap() error { return e.Err }

func quote(s string) string {
	return dart.Format("q", s)
}

// Itoa gives the decimal text of i.
func Itoa(i int) string {
	return FormatInt(int64(i), 10)
}

// FormatInt gives the text of i in the base.
func FormatInt(i int64, base int) string {
	if base < 2 || base > 36 {
		panic("strconv: illegal base")
	}
	if i == 0 {
		return "0"
	}
	negative := i < 0
	digits := "0123456789abcdefghijklmnopqrstuvwxyz"
	out := ""
	v := i
	if negative {
		v = -v
	}
	for v > 0 {
		out = string(digits[v%int64(base)]) + out
		v /= int64(base)
	}
	if negative {
		return "-" + out
	}
	return out
}

// AppendInt adds the text of i in the base to dst.
func AppendInt(dst []byte, i int64, base int) []byte {
	return append(dst, FormatInt(i, base)...)
}

// AppendUint adds the text of i in the base to dst.
func AppendUint(dst []byte, i uint64, base int) []byte {
	return append(dst, FormatUint(i, base)...)
}

// AppendBool adds "true" or "false" to dst.
func AppendBool(dst []byte, b bool) []byte {
	return append(dst, FormatBool(b)...)
}

// AppendFloat adds the text of f to dst.
func AppendFloat(dst []byte, f float64, verb byte, precision, bitSize int) []byte {
	return append(dst, FormatFloat(f, verb, precision, bitSize)...)
}

// AppendQuote adds the quoted form of s to dst.
func AppendQuote(dst []byte, s string) []byte {
	return append(dst, Quote(s)...)
}

// FormatBool gives "true" or "false".
func FormatBool(b bool) string {
	if b {
		return "true"
	}
	return "false"
}

// FormatFloat gives the text of f. The verb is one of 'f', 'e', and 'g', and a
// negative precision gives the shortest text.
func FormatFloat(f float64, verb byte, precision, bitSize int) string {
	form := string(rune(verb))
	if precision >= 0 {
		form = "." + Itoa(precision) + form
	}
	return dart.Format(form, f)
}

// Atoi reads a decimal int from text.
func Atoi(text string) (int, error) {
	v, err := ParseInt(text, 10, 64)
	if err != nil {
		return 0, &NumError{Func: "Atoi", Num: text, Err: ErrSyntax}
	}
	return int(v), nil
}

// ParseInt reads an int from text in the base.
func ParseInt(text string, base, bitSize int) (int64, error) {
	if len(text) == 0 {
		return 0, &NumError{Func: "ParseInt", Num: text, Err: ErrSyntax}
	}
	i := 0
	negative := false
	if text[0] == '+' || text[0] == '-' {
		negative = text[0] == '-'
		i = 1
	}
	out, err := parseDigits(text, i, base, "ParseInt")
	if err != nil {
		return 0, err
	}
	if negative {
		return -out, nil
	}
	return out, nil
}

// ParseUint reads an unsigned value from text.
func ParseUint(text string, base, bitSize int) (uint64, error) {
	out, err := parseDigits(text, 0, base, "ParseUint")
	if err != nil {
		return 0, err
	}
	return uint64(out), nil
}

// parseDigits reads the digits of a value. A base of 0 reads the prefix of the
// text, exactly as Go does, and it accepts an underscore between 2 digits.
func parseDigits(text string, i, base int, name string) (int64, error) {
	if i >= len(text) {
		return 0, &NumError{Func: name, Num: text, Err: ErrSyntax}
	}
	underscore := false
	if base == 0 {
		underscore = true
		base = 10
		if text[i] == '0' {
			if i+1 == len(text) {
				return 0, nil
			}
			switch text[i+1] {
			case 'x', 'X':
				base, i = 16, i+2
			case 'b', 'B':
				base, i = 2, i+2
			case 'o', 'O':
				base, i = 8, i+2
			default:
				base, i = 8, i+1
			}
		}
	}
	if i >= len(text) {
		return 0, &NumError{Func: name, Num: text, Err: ErrSyntax}
	}
	var out int64
	digits := 0
	for ; i < len(text); i++ {
		if underscore && text[i] == '_' {
			continue
		}
		digit := digitValue(text[i])
		if digit < 0 || digit >= base {
			return 0, &NumError{Func: name, Num: text, Err: ErrSyntax}
		}
		out = out*int64(base) + int64(digit)
		digits++
	}
	if digits == 0 {
		return 0, &NumError{Func: name, Num: text, Err: ErrSyntax}
	}
	return out, nil
}

// ParseBool reads a bool from text.
func ParseBool(text string) (bool, error) {
	switch text {
	case "1", "t", "T", "true", "TRUE", "True":
		return true, nil
	case "0", "f", "F", "false", "FALSE", "False":
		return false, nil
	}
	return false, &NumError{Func: "ParseBool", Num: text, Err: ErrSyntax}
}

// ParseFloat reads a float from text.
func ParseFloat(text string, bitSize int) (float64, error) {
	v, ok := dart.ParseFloat(text)
	if !ok {
		return 0, &NumError{Func: "ParseFloat", Num: text, Err: ErrSyntax}
	}
	return v, nil
}

// Quote gives the Go text of s inside double quotes.
func Quote(s string) string {
	return dart.Format("q", s)
}

// FormatUint gives the text of i in the base.
func FormatUint(i uint64, base int) string {
	if base < 2 || base > 36 {
		panic("strconv: illegal base")
	}
	if i == 0 {
		return "0"
	}
	digits := "0123456789abcdefghijklmnopqrstuvwxyz"
	out := ""
	v := i
	for v > 0 {
		out = string(digits[int(v%uint64(base))]) + out
		v /= uint64(base)
	}
	return out
}

// CanBackquote reports whether s fits inside back quotes with no change.
func CanBackquote(s string) bool {
	for _, r := range s {
		if r == 0xFEFF || r == utf8.RuneError {
			return false
		}
		if r > 0x7F {
			continue
		}
		if r < ' ' && r != '\t' || r == '`' || r == 0x7F {
			return false
		}
	}
	return true
}

// QuoteToASCII gives the Go text of s inside double quotes. Each rune outside
// the printable range of ASCII takes an escape form.
//
// An invalid byte of UTF-8 reaches the escape of the replacement rune. Go
// writes the byte itself in the \x form.
func QuoteToASCII(s string) string {
	out := "\""
	for _, r := range s {
		switch r {
		case '"':
			out += "\\\""
		case '\\':
			out += "\\\\"
		case '\a':
			out += "\\a"
		case '\b':
			out += "\\b"
		case '\f':
			out += "\\f"
		case '\n':
			out += "\\n"
		case '\r':
			out += "\\r"
		case '\t':
			out += "\\t"
		case '\v':
			out += "\\v"
		default:
			if r >= ' ' && r < 0x7F {
				out += string(r)
				continue
			}
			out += escape(r)
		}
	}
	return out + "\""
}

// escape writes 1 rune in the \x form, the \u form, or the \U form of Go. Go
// takes the \x form for a control character of ASCII only.
func escape(r rune) string {
	if r < ' ' || r == 0x7F {
		return "\\x" + hex(uint64(r), 2)
	}
	if r < 0x10000 {
		return "\\u" + hex(uint64(r), 4)
	}
	return "\\U" + hex(uint64(r), 8)
}

// hex writes a value in the base 16, with a minimum count of digits.
func hex(v uint64, width int) string {
	out := FormatUint(v, 16)
	for len(out) < width {
		out = "0" + out
	}
	return out
}

func digitValue(c byte) int {
	switch {
	case c >= '0' && c <= '9':
		return int(c - '0')
	case c >= 'a' && c <= 'z':
		return int(c-'a') + 10
	case c >= 'A' && c <= 'Z':
		return int(c-'A') + 10
	}
	return -1
}

// UnquoteChar decodes the first character or byte in the escaped string
// or character literal represented by the string s.
// It returns four values:
//
//  1. value, the decoded Unicode code point or byte value;
//  2. multibyte, a boolean indicating whether the decoded character requires a multibyte UTF-8 representation;
//  3. tail, the remainder of the string after the character; and
//  4. an error that will be nil if the character is syntactically valid.
//
// The second argument, quote, specifies the type of literal being parsed
// and therefore which escaped quote character is permitted.
// If set to a single quote, it permits the sequence \' and disallows unescaped '.
// If set to a double quote, it permits \" and disallows unescaped ".
// If set to zero, it does not permit either escape and allows both quote characters to appear unescaped.
func UnquoteChar(s string, quote byte) (value rune, multibyte bool, tail string, err error) {
	// easy cases
	if len(s) == 0 {
		err = ErrSyntax
		return
	}
	switch c := s[0]; {
	case c == quote && (quote == '\'' || quote == '"'):
		err = ErrSyntax
		return
	case c >= utf8.RuneSelf:
		r, size := utf8.DecodeRuneInString(s)
		return r, true, s[size:], nil
	case c != '\\':
		return rune(s[0]), false, s[1:], nil
	}

	// hard case: c is backslash
	if len(s) <= 1 {
		err = ErrSyntax
		return
	}
	c := s[1]
	s = s[2:]

	switch c {
	case 'a':
		value = '\a'
	case 'b':
		value = '\b'
	case 'f':
		value = '\f'
	case 'n':
		value = '\n'
	case 'r':
		value = '\r'
	case 't':
		value = '\t'
	case 'v':
		value = '\v'
	case 'x', 'u', 'U':
		n := 0
		switch c {
		case 'x':
			n = 2
		case 'u':
			n = 4
		case 'U':
			n = 8
		}
		var v rune
		if len(s) < n {
			err = ErrSyntax
			return
		}
		for j := 0; j < n; j++ {
			x, ok := unhex(s[j])
			if !ok {
				err = ErrSyntax
				return
			}
			v = v<<4 | x
		}
		s = s[n:]
		if c == 'x' {
			// single-byte string, possibly not UTF-8
			value = v
			break
		}
		if !utf8.ValidRune(v) {
			err = ErrSyntax
			return
		}
		value = v
		multibyte = true
	case '0', '1', '2', '3', '4', '5', '6', '7':
		v := rune(c) - '0'
		if len(s) < 2 {
			err = ErrSyntax
			return
		}
		for j := 0; j < 2; j++ { // one digit already; two more
			x := rune(s[j]) - '0'
			if x < 0 || x > 7 {
				err = ErrSyntax
				return
			}
			v = (v << 3) | x
		}
		s = s[2:]
		if v > 255 {
			err = ErrSyntax
			return
		}
		value = v
	case '\\':
		value = '\\'
	case '\'', '"':
		if c != quote {
			err = ErrSyntax
			return
		}
		value = rune(c)
	default:
		err = ErrSyntax
		return
	}
	tail = s
	return
}

// Unquote interprets s as a single-quoted, double-quoted,
// or backquoted Go string literal, returning the string value
// that s quotes.  (If s is single-quoted, it would be a Go
// character literal; Unquote returns the corresponding
// one-character string. For an empty character literal
// Unquote returns the empty string.)
func Unquote(s string) (string, error) {
	out, rem, err := unquote(s, true)
	if len(rem) > 0 {
		return "", ErrSyntax
	}
	return out, err
}

// unquote parses a quoted string at the start of the input,
// returning the parsed prefix, the remaining suffix, and any parse errors.
// If unescape is true, the parsed prefix is unescaped,
// otherwise the input prefix is provided verbatim.
func unquote(in string, unescape bool) (out, rem string, err error) {
	// Determine the quote form and optimistically find the terminating quote.
	if len(in) < 2 {
		return "", in, ErrSyntax
	}
	quote := in[0]
	end := index(in[1:], quote)
	if end < 0 {
		return "", in, ErrSyntax
	}
	end += 2 // position after terminating quote; may be wrong if escape sequences are present

	switch quote {
	case '`':
		switch {
		case !unescape:
			out = in[:end] // include quotes
		case !contains(in[:end], '\r'):
			out = in[len("`") : end-len("`")] // exclude quotes
		default:
			// Carriage return characters ('\r') inside raw string literals
			// are discarded from the raw string value.
			buf := make([]byte, 0, end-len("`")-len("\r")-len("`"))
			for i := len("`"); i < end-len("`"); i++ {
				if in[i] != '\r' {
					buf = append(buf, in[i])
				}
			}
			out = string(buf)
		}
		// NOTE: Prior implementations did not verify that raw strings consist
		// of valid UTF-8 characters and we continue to not verify it as such.
		// The Go specification does not explicitly require valid UTF-8,
		// but only mention that it is implicitly valid for Go source code
		// (which must be valid UTF-8).
		return out, in[end:], nil
	case '"', '\'':
		// Handle quoted strings without any escape sequences.
		if !contains(in[:end], '\\') && !contains(in[:end], '\n') {
			var valid bool
			switch quote {
			case '"':
				valid = utf8.ValidString(in[len(`"`) : end-len(`"`)])
			case '\'':
				r, n := utf8.DecodeRuneInString(in[len("'") : end-len("'")])
				valid = len("'")+n+len("'") == end && (r != utf8.RuneError || n != 1)
			}
			if valid {
				out = in[:end]
				if unescape {
					out = out[1 : end-1] // exclude quotes
				}
				return out, in[end:], nil
			}
		}

		// Handle quoted strings with escape sequences.
		var buf []byte
		in0 := in
		in = in[1:] // skip starting quote
		if unescape {
			buf = make([]byte, 0, 3*end/2) // try to avoid more allocations
		}
		for len(in) > 0 && in[0] != quote {
			// Process the next character,
			// rejecting any unescaped newline characters which are invalid.
			r, multibyte, rem, err := UnquoteChar(in, quote)
			if in[0] == '\n' || err != nil {
				return "", in0, ErrSyntax
			}
			in = rem

			// Append the character if unescaping the input.
			if unescape {
				if r < utf8.RuneSelf || !multibyte {
					buf = append(buf, byte(r))
				} else {
					buf = utf8.AppendRune(buf, r)
				}
			}

			// Single quoted strings must be a single character.
			if quote == '\'' {
				break
			}
		}

		// Verify that the string ends with a terminating quote.
		if !(len(in) > 0 && in[0] == quote) {
			return "", in0, ErrSyntax
		}
		in = in[1:] // skip terminating quote

		if unescape {
			return string(buf), in, nil
		}
		return in0[:len(in0)-len(in)], in, nil
	default:
		return "", in, ErrSyntax
	}
}

// contains reports whether the string holds the byte.
func contains(s string, c byte) bool {
	return index(s, c) != -1
}

// index gives the position of the byte in the string, and it gives -1 for a
// string that holds no such byte.
func index(s string, c byte) int {
	for i := 0; i < len(s); i++ {
		if s[i] == c {
			return i
		}
	}
	return -1
}

// unhex gives the value of a hexadecimal digit.
func unhex(b byte) (v rune, ok bool) {
	c := rune(b)
	switch {
	case '0' <= c && c <= '9':
		return c - '0', true
	case 'a' <= c && c <= 'f':
		return c - 'a' + 10, true
	case 'A' <= c && c <= 'F':
		return c - 'A' + 10, true
	}
	return 0, false
}
