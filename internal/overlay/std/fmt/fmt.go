//go:build dart

// Package fmt writes values in the format of Go.
//
// The Dart runtime writes 1 value, and this package reads the format text and
// applies the width.
package fmt

import "syscall/dart"

// Sprint returns the text of the operands. It puts 1 space between 2 operands
// when neither one is a string.
func Sprint(a ...any) string {
	out := ""
	for i, v := range a {
		if i > 0 && dart.Kind(a[i-1]) != "string" && dart.Kind(v) != "string" {
			out += " "
		}
		out += dart.Format("v", v)
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
		out += dart.Format("v", v)
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
		for i < len(format) && isFlag(format[i]) {
			verb += string(rune(format[i]))
			i++
		}
		width := 0
		for i < len(format) && isDigit(format[i]) {
			width = width*10 + int(format[i]-'0')
			i++
		}
		if i < len(format) && format[i] == '.' {
			verb += "."
			i++
			for i < len(format) && isDigit(format[i]) {
				verb += string(rune(format[i]))
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
		text := dart.Format(verb+string(rune(letter)), a[arg])
		arg++
		out += pad(text, width, verb)
	}
	out += format[start:]
	return out
}

// Errorf returns an error that holds the formatted text.
func Errorf(format string, a ...any) error {
	return &formatError{text: Sprintf(format, a...)}
}

type formatError struct {
	text string
}

func (e *formatError) Error() string {
	return e.text
}

func isFlag(c byte) bool {
	return c == '+' || c == '-' || c == '#' || c == ' ' || c == '0'
}

func isDigit(c byte) bool {
	return c >= '0' && c <= '9'
}

// pad gives the text the width of the verb. The flag `-` puts the space after
// the text.
func pad(text string, width int, verb string) string {
	n := width - runeCount(text)
	if n <= 0 {
		return text
	}
	space := ""
	for range n {
		space += " "
	}
	if contains(verb, '-') {
		return text + space
	}
	return space + text
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
