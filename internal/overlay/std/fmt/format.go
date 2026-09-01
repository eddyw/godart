//go:build dart

package fmt

import "syscall/dart"

// The Formatter interface of Go.
//
// A type that holds the method Format writes its own text under each verb.
// Sprintf calls that method in place of the writer of the runtime, and the
// method then owns the width, the precision, and each flag.

// State holds the state of 1 call of Format. The method writes its text to it.
type State interface {
	// Write writes the bytes of the text.
	Write(b []byte) (n int, err error)
	// Width gives the width of the verb, and whether the verb holds one.
	Width() (wid int, ok bool)
	// Precision gives the precision of the verb, and whether the verb holds
	// one.
	Precision() (prec int, ok bool)
	// Flag reports whether the verb holds the flag.
	Flag(c int) bool
}

// Formatter writes the text of a value under each verb of Sprintf.
type Formatter interface {
	Format(f State, verb rune)
}

// FormatString gives the verb of a State again, with each flag, the width, and
// the precision. A Format method that falls back to another verb reads it.
func FormatString(state State, verb rune) string {
	out := "%"
	for _, c := range []byte{'-', '+', '#', ' ', '0'} {
		if state.Flag(int(c)) {
			out += string(rune(c))
		}
	}
	if w, ok := state.Width(); ok {
		out += itoa(w)
	}
	if p, ok := state.Precision(); ok {
		out += "." + itoa(p)
	}
	return out + string(verb)
}

// spec holds the parts of 1 verb of a format text.
type spec struct {
	flags    string
	width    int
	hasWidth bool
	prec     int
	hasPrec  bool
}

// state carries 1 spec into a Format method, and it holds the text that the
// method writes.
type state struct {
	spec spec
	buf  []byte
}

func (s *state) Write(b []byte) (int, error) {
	s.buf = append(s.buf, b...)
	return len(b), nil
}

func (s *state) Width() (int, bool) { return s.spec.width, s.spec.hasWidth }

func (s *state) Precision() (int, bool) { return s.spec.prec, s.spec.hasPrec }

func (s *state) Flag(c int) bool {
	if c < 0 || c > 0xFF {
		return false
	}
	return contains(s.spec.flags, byte(c))
}

func itoa(v int) string {
	if v == 0 {
		return "0"
	}
	neg := v < 0
	if neg {
		v = -v
	}
	digits := ""
	for v > 0 {
		digits = string(rune('0'+v%10)) + digits
		v /= 10
	}
	if neg {
		return "-" + digits
	}
	return digits
}

// plain writes 1 value under the verb v. A type that holds the method Format
// writes its own text.
func plain(v any) string {
	if f, ok := v.(Formatter); ok {
		st := &state{}
		f.Format(st, 'v')
		return string(st.buf)
	}
	return dart.Format("v", v)
}
