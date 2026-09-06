//go:build dart

// Package fmt writes values in the format of Go.
//
// The Dart runtime writes 1 value, and this package reads the format text and
// applies the width.
package fmt

import (
	"errors"
	"io"
	"strconv"
	"strings"
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
	var out strings.Builder
	arg := 0
	start := 0
	i := 0
	for i < len(format) {
		if format[i] != '%' {
			i++
			continue
		}
		out.WriteString(format[start:i])
		i++
		if i < len(format) && format[i] == '%' {
			out.WriteString("%")
			i++
			start = i
			continue
		}
		// The bits of the flags travel to the runtime, thus no verb text
		// stands in this loop. A build of the text costs 1 Go string for each
		// verb, and the runtime scans that text again for each flag.
		flagStart := i
		bits := 0
		for i < len(format) && isFlag(format[i]) {
			bits |= flagBit(format[i])
			i++
		}
		flagEnd := i
		// The parts of the verb stand in a local each. A spec is an object of
		// Dart, and only a Format method reads one.
		hasWidth := false
		prec := 0
		hasPrec := false
		// starMinus marks a star width below 0, which Go reads as the flag -.
		starMinus := false
		// good marks an argument index that names an argument of the call. Go
		// writes %!verb(BADINDEX) for each other 1.
		good := true
		var afterIndex bool
		arg, i, afterIndex = argNumber(arg, format, i, len(a), &good)
		width := 0
		if i < len(format) && format[i] == '*' {
			i++
			var isInt bool
			width, isInt, arg = intFromArg(a, arg)
			if width < 0 {
				width = -width
				bits |= flagMinus
				starMinus = true
			}
			hasWidth = isInt
			if !isInt {
				width = 0
				out.WriteString("%!(BADWIDTH)")
			}
			afterIndex = false
		} else {
			var present bool
			width, present, i = parseNum(format, i)
			hasWidth = present
			if afterIndex && present {
				good = false
			}
		}
		if i+1 < len(format) && format[i] == '.' {
			i++
			if afterIndex {
				good = false
			}
			arg, i, afterIndex = argNumber(arg, format, i, len(a), &good)
			if i < len(format) && format[i] == '*' {
				i++
				var isInt bool
				prec, isInt, arg = intFromArg(a, arg)
				hasPrec = isInt && prec >= 0
				if !hasPrec {
					// Go writes the report and drops the precision, for a
					// value of a wrong type and for a value below 0.
					prec = 0
					out.WriteString("%!(BADPREC)")
				}
				afterIndex = false
			} else {
				var present bool
				prec, present, i = parseNum(format, i)
				hasPrec = true
				if afterIndex && present {
					good = false
				}
			}
		}
		if !afterIndex {
			arg, i, _ = argNumber(arg, format, i, len(a), &good)
		}
		if i >= len(format) {
			out.WriteString("%!(NOVERB)")
			start = i
			break
		}
		letter := format[i]
		i++
		start = i
		if !good {
			out.WriteString("%!" + string(rune(letter)) + "(BADINDEX)")
			continue
		}
		if arg >= len(a) {
			out.WriteString("%!" + string(rune(letter)) + "(MISSING)")
			continue
		}
		if f, ok := a[arg].(Formatter); ok {
			// The method owns the width, the precision, and each flag, thus
			// the text of the flags stands here and in no other place.
			st := &state{spec: spec{
				flags:    flagText(format, flagStart, flagEnd, starMinus),
				width:    width,
				hasWidth: hasWidth,
				prec:     prec,
				hasPrec:  hasPrec,
			}}
			f.Format(st, rune(letter))
			out.WriteString(string(st.buf))
			arg++
			continue
		}
		value := a[arg]
		if hasPrec {
			value = truncate(value, letter, prec)
		}
		if letter == 'U' {
			flags := flagText(format, flagStart, flagEnd, starMinus)
			out.WriteString(pad(unicodeText(value, flags), width, bits, letter))
			arg++
			continue
		}
		// The verb q of a rune reads the escape table of Go. strconv holds it,
		// and the Dart side holds no such table.
		if letter == 'q' {
			if r, ok := runeOf(value); ok {
				text := strconv.QuoteRune(r)
				if bits&flagPlus != 0 {
					text = strconv.QuoteRuneToASCII(r)
				}
				out.WriteString(pad(text, width, bits, letter))
				arg++
				continue
			}
			if t, ok := value.(string); ok {
				text := strconv.Quote(t)
				if bits&flagPlus != 0 {
					text = strconv.QuoteToASCII(t)
				}
				out.WriteString(pad(text, width, bits, letter))
				arg++
				continue
			}
		}
		want := -1
		if hasPrec {
			want = prec
		}
		// The runtime owns the width and the precision digits here, because Go
		// writes each 1 on the leaves of a value and not on the whole text.
		arg++
		out.WriteString(dart.FormatVerb(int(letter), bits, width, want, value))
	}
	out.WriteString(format[start:])
	return out.String()
}

// The bit of each flag of a verb. The Dart runtime reads the same numbers.
const (
	flagMinus = 1 << iota
	flagPlus
	flagSharp
	flagSpace
	flagZero
)

// flagBit gives the bit of 1 flag byte, and 0 for each other byte.
func flagBit(c byte) int {
	switch c {
	case '-':
		return flagMinus
	case '+':
		return flagPlus
	case '#':
		return flagSharp
	case ' ':
		return flagSpace
	case '0':
		return flagZero
	}
	return 0
}

// flagText gives the flags of a verb as text. Only a Format method, the verb U,
// and a report read it, thus the hot path of Sprintf never builds it.
func flagText(format string, start, end int, minus bool) string {
	out := ""
	if start < end {
		out = format[start:end]
	}
	if minus {
		out += "-"
	}
	return out
}

// unicodeText writes the code point of a value, as the verb U of Go does. The
// flag # adds the character of a printable rune.
//
// strconv.IsPrint reads the tables of Go, thus the answer matches Go for each
// rune. The Dart side holds no such table.
func unicodeText(v any, flags string) string {
	r, ok := runeOf(v)
	if !ok {
		// Go names the type of the value in the report of a wrong verb.
		return "%!U(" + dart.Kind(v) + "=" + dart.Format("v", v) + ")"
	}
	digits := strings.ToUpper(strconv.FormatInt(int64(r), 16))
	for len(digits) < 4 {
		digits = "0" + digits
	}
	out := "U+" + digits
	if contains(flags, '#') && strconv.IsPrint(r) {
		out += " '" + string(r) + "'"
	}
	return out
}

// runeOf reads a code point from an integer value of any width.
func runeOf(v any) (rune, bool) {
	switch n := v.(type) {
	case rune:
		return n, true
	case int:
		return rune(n), true
	case int8:
		return rune(n), true
	case int16:
		return rune(n), true
	case int64:
		return rune(n), true
	case uint:
		return rune(n), true
	case uint8:
		return rune(n), true
	case uint16:
		return rune(n), true
	case uint32:
		return rune(n), true
	case uint64:
		return rune(n), true
	}
	return 0, false
}

// truncate cuts a text argument to the precision. Go reads the precision of a
// string verb as a count of runes, and it cuts the value before it writes it,
// thus `%.3q` of "abcdef" gives `"abc"`.
func truncate(v any, letter byte, prec int) any {
	switch letter {
	case 's', 'q', 'v', 'x', 'X':
	default:
		return v
	}
	switch t := v.(type) {
	case string:
		return cutRunes(t, prec)
	case []byte:
		return []byte(cutRunes(string(t), prec))
	}
	return v
}

// cutRunes gives the first n runes of a text.
func cutRunes(s string, n int) string {
	count := 0
	for i := range s {
		if count == n {
			return s[:i]
		}
		count++
	}
	return s
}

// parseNum reads a decimal number at the position. It gives the number, a flag
// that reports a digit, and the position after it.
func parseNum(format string, i int) (int, bool, int) {
	num := 0
	present := false
	for i < len(format) && isDigit(format[i]) {
		num = num*10 + int(format[i]-'0')
		present = true
		i++
	}
	return num, present, i
}

// argNumber reads the form [n] of an explicit argument index. Go counts from 1,
// thus the result is n minus 1. An index that names no argument sets good to
// false, and the verb then writes BADINDEX.
func argNumber(arg int, format string, i, count int, good *bool) (int, int, bool) {
	if i >= len(format) || format[i] != '[' {
		return arg, i, false
	}
	end := -1
	for j := i + 1; j < len(format); j++ {
		if format[j] == ']' {
			end = j
			break
		}
	}
	if end < 0 {
		*good = false
		return arg, i + 1, false
	}
	num, present, after := parseNum(format, i+1)
	if !present || after != end {
		*good = false
		return arg, end + 1, false
	}
	index := num - 1
	if index < 0 || index >= count {
		*good = false
		return arg, end + 1, false
	}
	return index, end + 1, true
}

// intFromArg reads the argument of a star width or of a star precision. Go
// accepts each integer type, and it reports BADWIDTH for each other type.
func intFromArg(a []any, arg int) (int, bool, int) {
	if arg >= len(a) {
		return 0, false, arg
	}
	v := a[arg]
	arg++
	switch n := v.(type) {
	case int:
		return n, true, arg
	case int8:
		return int(n), true, arg
	case int16:
		return int(n), true, arg
	case int32:
		return int(n), true, arg
	case int64:
		return int(n), true, arg
	case uint:
		return int(n), true, arg
	case uint8:
		return int(n), true, arg
	case uint16:
		return int(n), true, arg
	case uint32:
		return int(n), true, arg
	case uint64:
		return int(n), true, arg
	}
	return 0, false, arg
}

// Errorf returns an error that holds the formatted text.
func Errorf(format string, a ...any) error {
	// The verb %w writes the error and keeps it. Sprintf holds no such verb,
	// thus the format takes %v in its place.
	plain := ""
	var held []error
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
			// The verb w takes an operand that holds the interface error.
			// Each other operand keeps the letter w, thus Sprintf writes the
			// bad verb form of Go: %!w(<nil>) and %!w(string=a text).
			wraps := false
			if k := verbIndex(format, i); k >= 0 && k < len(a) {
				if err, ok := a[k].(error); ok && err != nil {
					held = append(held, err)
					wraps = true
				}
			}
			if wraps {
				plain += format[i:j] + "v"
			} else {
				plain += format[i : j+1]
			}
			i = j
			continue
		}
		plain += format[i : j+1]
		i = j
	}
	text := Sprintf(plain, a...)
	// Go gives 3 types, and each 1 reaches a program through %T and through a
	// type switch. A format with no %w gives the type of errors.New.
	switch len(held) {
	case 0:
		return errors.New(text)
	case 1:
		return &wrapError{text: text, err: held[0]}
	}
	return &wrapErrors{text: text, errs: held}
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

// wrapError holds the error of 1 verb %w. Go gives this type the same name.
type wrapError struct {
	text string
	err  error
}

func (e *wrapError) Error() string { return e.text }

// Unwrap gives the error of the verb %w.
func (e *wrapError) Unwrap() error { return e.err }

// wrapErrors holds the errors of more than 1 verb %w. Go gives this type the
// same name, and its Unwrap gives a list.
type wrapErrors struct {
	text string
	errs []error
}

func (e *wrapErrors) Error() string { return e.text }

func (e *wrapErrors) Unwrap() []error { return e.errs }

func isFlag(c byte) bool {
	return c == '+' || c == '-' || c == '#' || c == ' ' || c == '0'
}

func isDigit(c byte) bool {
	return c >= '0' && c <= '9'
}

// pad gives the text the width of the verb. The flag `-` puts the fill after
// the text, and the flag `0` writes a zero in place of each space.
func pad(text string, width int, bits int, letter byte) string {
	// A width of 0 leaves the text as it stands. The walk of the runes below
	// reads each byte, thus the test stands in front of it.
	if width <= 0 {
		return text
	}
	// Go turns the flag 0 off for the verb U, thus a width there gives spaces.
	if letter != 'U' && bits&flagZero != 0 && bits&flagMinus == 0 {
		return zeroPad(text, width, bits, letter)
	}
	n := width - runeCount(text)
	if n <= 0 {
		return text
	}
	if bits&flagMinus != 0 {
		return text + fill(n, ' ')
	}
	return fill(n, ' ') + text
}

// zeroPad writes the zeros of the flag 0.
//
// Go puts the zeros after the sign and after the base prefix of the flag #,
// and it counts the sign against the width and the prefix not. The prefix of
// an octal is 1 zero, and Go writes that zero only when the digits hold none,
// thus the padding gives it.
func zeroPad(text string, width int, bits int, letter byte) string {
	sign := ""
	if len(text) > 0 && (text[0] == '-' || text[0] == '+' || text[0] == ' ') {
		sign, text = text[:1], text[1:]
	}
	alt := bits&flagSharp != 0
	prefix := ""
	octal := alt && letter == 'o'
	switch {
	case octal:
		// dart.Format already wrote the leading zero of the octal. The width
		// covers the digits alone, thus the zero comes off first.
		if len(text) > 1 && text[0] == '0' {
			text = text[1:]
		}
	case alt || letter == 'O':
		// The verb O writes its prefix with no flag, thus the width covers the
		// digits alone for that verb too.
		prefix, text = cutBase(text)
	}
	if n := width - runeCount(sign) - runeCount(text); n > 0 {
		text = fill(n, '0') + text
	}
	if octal && (len(text) == 0 || text[0] != '0') {
		text = "0" + text
	}
	return sign + prefix + text
}

// cutBase splits the 2 character base prefix that the flag # writes from the
// digits behind it.
func cutBase(text string) (string, string) {
	if len(text) >= 2 && text[0] == '0' {
		switch text[1] {
		case 'x', 'X', 'b', 'B', 'o', 'O':
			return text[:2], text[2:]
		}
	}
	return "", text
}

func fill(n int, c byte) string {
	return strings.Repeat(string(rune(c)), n)
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
