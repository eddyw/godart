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
