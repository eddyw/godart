//go:build dart

// Package fmt writes values in the format of Go.
//
// The Dart runtime writes 1 value, and this package reads the format text and
// applies the width.
package fmt

import (
	"io"
	"syscall/dart"
)

// Sprint returns the text of the operands. It puts 1 space between 2 operands
// when neither one is a string.
func Sprint(a ...any) string {
	out := ""
	for i, v := range a {
		if i > 0 && dart.Kind(a[i-1]) != "string" && dart.Kind(v) != "string" {
			out += " "
		}
		out += plain(v)
	}
	return out
}

// Sprintln returns the text of the operands, with 1 space between 2 operands
// and a newline at the end.
func Sprintln(a ...any) string {
	out := ""
	for i, v := range a {
		if i > 0 {
			out += " "
		}
		out += plain(v)
	}
	return out + "\n"
}

// Print writes the operands to standard output.
func Print(a ...any) (int, error) {
	text := Sprint(a...)
	dart.Print(text)
	return len(text), nil
}

// Println writes the operands and a newline to standard output.
func Println(a ...any) (int, error) {
	text := Sprintln(a...)
	dart.Print(text)
	return len(text), nil
}

// Printf writes the formatted text to standard output.
func Printf(format string, a ...any) (int, error) {
	text := Sprintf(format, a...)
	dart.Print(text)
	return len(text), nil
}

// Stringer holds the method that gives the text of a value.
type Stringer interface {
	String() string
}

// GoStringer holds the method that gives the Go syntax of a value.
type GoStringer interface {
	GoString() string
}

// Append writes each value after b. It adds a space between 2 values when
// neither is a string.
func Append(b []byte, a ...any) []byte { return append(b, Sprint(a...)...) }

// Appendln writes each value after b with a space between 2 values, and a
// newline at the end.
func Appendln(b []byte, a ...any) []byte { return append(b, Sprintln(a...)...) }

// Appendf writes the format with each value after b.
func Appendf(b []byte, format string, a ...any) []byte {
	return append(b, Sprintf(format, a...)...)
}

// Fprint writes each value to w. It adds a space between 2 values when neither
// is a string.
func Fprint(w io.Writer, a ...any) (int, error) {
	return w.Write([]byte(Sprint(a...)))
}

// Fprintln writes each value to w with a space between 2 values, and a newline
// at the end.
func Fprintln(w io.Writer, a ...any) (int, error) {
	return w.Write([]byte(Sprintln(a...)))
}

// Fprintf writes the format with each value to w.
func Fprintf(w io.Writer, format string, a ...any) (int, error) {
	return w.Write([]byte(Sprintf(format, a...)))
}

// Sprintf returns the formatted text.
func Sprintf(format string, a ...any) string {
	out := ""
	arg := 0
	start := 0
	i := 0
	for i < len(format) {
		if format[i] != '%' {
			i++
			continue
		}
		out += format[start:i]
		i++
		if i < len(format) && format[i] == '%' {
			out += "%"
			i++
			start = i
			continue
		}
		verb := ""
		var sp spec
		for i < len(format) && isFlag(format[i]) {
			verb += string(rune(format[i]))
			sp.flags += string(rune(format[i]))
			i++
		}
		width := 0
		for i < len(format) && isDigit(format[i]) {
			width = width*10 + int(format[i]-'0')
			sp.hasWidth = true
			i++
		}
		sp.width = width
		if i < len(format) && format[i] == '.' {
			verb += "."
			sp.hasPrec = true
			i++
			for i < len(format) && isDigit(format[i]) {
				verb += string(rune(format[i]))
				sp.prec = sp.prec*10 + int(format[i]-'0')
				i++
			}
		}
		if i >= len(format) {
			out += "%!(NOVERB)"
			start = i
			break
		}
		letter := format[i]
		i++
		start = i
		if arg >= len(a) {
			out += "%!" + string(rune(letter)) + "(MISSING)"
			continue
		}
		if f, ok := a[arg].(Formatter); ok {
			// The method owns the width, the precision, and each flag.
			st := &state{spec: sp}
			f.Format(st, rune(letter))
			out += string(st.buf)
			arg++
			continue
		}
		text := dart.Format(verb+string(rune(letter)), a[arg])
		arg++
		out += pad(text, width, verb)
	}
	out += format[start:]
	return out
}

// Errorf returns an error that holds the formatted text.
func Errorf(format string, a ...any) error {
	// The verb %w writes the error and keeps it. Sprintf holds no such verb,
	// thus the format takes %v in its place.
	plain := ""
	var held error
	for i := 0; i < len(format); i++ {
		if format[i] != '%' || i+1 >= len(format) {
			plain += string(format[i])
			continue
		}
		j := i + 1
		for j < len(format) && (isFlag(format[j]) || isDigit(format[j]) || format[j] == '.') {
			j++
		}
		if j < len(format) && format[j] == 'w' {
			if k := verbIndex(format, i); k >= 0 && k < len(a) {
				if err, ok := a[k].(error); ok && held == nil {
					held = err
				}
			}
			plain += format[i:j] + "v"
			i = j
			continue
		}
		plain += format[i : j+1]
		i = j
	}
	return &formatError{text: Sprintf(plain, a...), err: held}
}

// verbIndex counts the verbs in front of the position, thus it gives the index
// of the argument of the verb at that position.
func verbIndex(format string, pos int) int {
	n := 0
	for i := 0; i < pos; i++ {
		if format[i] != '%' || i+1 >= len(format) {
			continue
		}
		j := i + 1
		for j < len(format) && (isFlag(format[j]) || isDigit(format[j]) || format[j] == '.') {
			j++
		}
		if j < len(format) && format[j] == '%' {
			i = j
			continue
		}
		n++
		i = j
	}
	return n
}

type formatError struct {
	text string
	err  error
}

func (e *formatError) Error() string {
	return e.text
}

// Unwrap gives the error of the verb %w. It gives nil when the format holds no
// such verb.
func (e *formatError) Unwrap() error { return e.err }

func isFlag(c byte) bool {
	return c == '+' || c == '-' || c == '#' || c == ' ' || c == '0'
}

func isDigit(c byte) bool {
	return c >= '0' && c <= '9'
}

// pad gives the text the width of the verb. The flag `-` puts the fill after
// the text, and the flag `0` writes a zero in place of each space.
func pad(text string, width int, verb string) string {
	n := width - runeCount(text)
	if n <= 0 {
		return text
	}
	if contains(verb, '-') {
		return text + fill(n, ' ')
	}
	if contains(verb, '0') {
		// The sign stands in front of the zeros.
		sign := ""
		if len(text) > 0 && (text[0] == '-' || text[0] == '+') {
			sign, text = text[:1], text[1:]
		}
		return sign + fill(n, '0') + text
	}
	return fill(n, ' ') + text
}

func fill(n int, c byte) string {
	out := ""
	for range n {
		out += string(rune(c))
	}
	return out
}

func runeCount(text string) int {
	n := 0
	for range text {
		n++
	}
	return n
}

func contains(text string, c byte) bool {
	for i := 0; i < len(text); i++ {
		if text[i] == c {
			return true
		}
	}
	return false
}
