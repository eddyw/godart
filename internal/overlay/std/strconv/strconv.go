//go:build dart

// Package strconv converts between a Go string and a basic value.
package strconv

import (
	"errors"
	"syscall/dart"
)

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
	if i >= len(text) {
		return 0, &NumError{Func: "ParseInt", Num: text, Err: ErrSyntax}
	}
	var out int64
	for ; i < len(text); i++ {
		digit := digitValue(text[i])
		if digit < 0 || digit >= base {
			return 0, &NumError{Func: "ParseInt", Num: text, Err: ErrSyntax}
		}
		out = out*int64(base) + int64(digit)
	}
	if negative {
		return -out, nil
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
