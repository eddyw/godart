//go:build dart

package json

import (
	"errors"
	"math"
	"sort"
	"strconv"
	"unicode/utf16"
	"unicode/utf8"
)

// A Kind names the kind of a JSON value. The value is the first byte of the
// value in the grammar, and a number always takes the digit 0.
//
// Go holds this type in the package encoding/json/jsontext, and that package
// stays outside the surface. RawMessage.Kind gives a kind, thus the name
// stands here. A program compares a kind with a character, for example
// `m.Kind() == '{'`.
type Kind byte

// String gives the name of the kind.
func (k Kind) String() string {
	switch k {
	case 0:
		return "invalid"
	case 'n':
		return "null"
	case 'f':
		return "false"
	case 't':
		return "true"
	case '"':
		return "string"
	case '0':
		return "number"
	case '{', '}', '[', ']':
		return string([]byte{byte(k)})
	}
	return "<invalid jsontext.Kind: " + strconv.QuoteRune(rune(k)) + ">"
}

// Kind gives the kind of the first token of the value. A value that holds no
// token gives the invalid kind 0.
func (m RawMessage) Kind() Kind {
	i := 0
	for i < len(m) && isSpace(m[i]) {
		i++
	}
	if i == len(m) {
		return 0
	}
	switch c := m[i]; c {
	case 'n', 'f', 't', '"', '{', '}', '[', ']':
		return Kind(c)
	case '-', '0', '1', '2', '3', '4', '5', '6', '7', '8', '9':
		return '0'
	}
	return 0
}

// Clone gives a copy of the value.
func (m RawMessage) Clone() RawMessage {
	if m == nil {
		return nil
	}
	out := make(RawMessage, len(m))
	copy(out, m)
	return out
}

// String gives the text of the value. A nil value gives the JSON null.
func (m RawMessage) String() string {
	if m == nil {
		return "null"
	}
	return string(m)
}

// IsValid reports whether the value holds 1 JSON value of RFC 7493.
//
// The check asks for valid UTF-8, for an escape that names a code point of
// Unicode, and for a unique name in each object.
func (m RawMessage) IsValid() bool {
	_, err := reformat(m, rawOpts{})
	return err == nil
}

// Format writes the minimal form of the value in place. It drops each space
// and it gives each string its shortest text.
//
// The check of IsValid runs first, thus a value outside RFC 7493 gives an
// error and no write.
func (m *RawMessage) Format() error { return m.apply(rawOpts{}) }

// Compact drops each space of the value in place.
//
// The call keeps the text of each string and of each number, thus it also
// holds a value with a duplicate name and a value with invalid UTF-8.
func (m *RawMessage) Compact() error {
	return m.apply(rawOpts{rawStrings: true, allowDup: true, allowBadUTF8: true})
}

// Indent writes 1 line for each member of an object and for each element of an
// array. Each level takes 1 tab.
//
// The call keeps the text of each string and of each number, exactly as
// Compact does.
func (m *RawMessage) Indent() error {
	return m.apply(rawOpts{
		expand: true, indent: "\t",
		rawStrings: true, allowDup: true, allowBadUTF8: true,
	})
}

// Canonicalize writes the form of RFC 8785 in place. It sorts the members of
// each object by name, it writes each number as a 64-bit float, and it drops
// each space.
func (m *RawMessage) Canonicalize() error {
	return m.apply(rawOpts{reorder: true, canonNumbers: true})
}

// apply writes the new form of the value. A value that already holds that form
// keeps its buffer, exactly as Go does.
func (m *RawMessage) apply(opt rawOpts) error {
	if m == nil {
		return errors.New("json.RawMessage: the pointer is nil")
	}
	out, err := reformat(*m, opt)
	if err != nil {
		return err
	}
	if string(out) == string(*m) {
		return nil
	}
	*m = append((*m)[:0], out...)
	return nil
}

// rawOpts holds the form that a reformat writes. The zero value is the form of
// Format: no space, the shortest text of each string, and the rules of
// RFC 7493.
type rawOpts struct {
	// expand puts each member of an object and each element of an array on a
	// line of its own.
	expand bool
	indent string
	// rawStrings keeps the text of each string of the input.
	rawStrings bool
	// allowDup holds an object that names 1 member 2 times.
	allowDup bool
	// allowBadUTF8 holds a string that is no valid UTF-8.
	allowBadUTF8 bool
	// reorder sorts the members of each object by name, per RFC 8785.
	reorder bool
	// canonNumbers writes each number as a 64-bit float, per RFC 8785.
	canonNumbers bool
}

// reformat reads 1 JSON value and writes its new form.
func reformat(src []byte, opt rawOpts) ([]byte, error) {
	f := &rawFormatter{src: src, opt: opt}
	out, err := f.value(0)
	if err != nil {
		return nil, err
	}
	for f.i < len(f.src) {
		if !isSpace(f.src[f.i]) {
			return nil, errors.New("jsontext: invalid character " +
				quoteChar(f.src[f.i]) + " after top-level value")
		}
		f.i++
	}
	return out, nil
}

// rawFormatter walks 1 JSON value of a buffer and writes its new form.
type rawFormatter struct {
	src []byte
	i   int
	opt rawOpts
}

func (f *rawFormatter) space() {
	for f.i < len(f.src) && isSpace(f.src[f.i]) {
		f.i++
	}
}

// line writes the newline and the indent of 1 level.
func (f *rawFormatter) line(out []byte, depth int) []byte {
	if !f.opt.expand {
		return out
	}
	out = append(out, '\n')
	for k := 0; k < depth; k++ {
		out = append(out, f.opt.indent...)
	}
	return out
}

var errRawEnd = errors.New("jsontext: unexpected end of JSON input")

// value reads 1 JSON value and gives its new form.
func (f *rawFormatter) value(depth int) ([]byte, error) {
	f.space()
	if f.i >= len(f.src) {
		return nil, errRawEnd
	}
	switch c := f.src[f.i]; {
	case c == '{':
		return f.object(depth)
	case c == '[':
		return f.array(depth)
	case c == '"':
		out, _, err := f.text()
		return out, err
	case c == 't':
		return f.word("true")
	case c == 'f':
		return f.word("false")
	case c == 'n':
		return f.word("null")
	case c == '-' || (c >= '0' && c <= '9'):
		return f.number()
	}
	return nil, errors.New("jsontext: invalid character " +
		quoteChar(f.src[f.i]) + " at start of value")
}

// member holds 1 member of an object. The sort of RFC 8785 reads the name.
type member struct {
	name []byte
	text []byte
}

func (f *rawFormatter) object(depth int) ([]byte, error) {
	f.i++
	f.space()
	if f.i >= len(f.src) {
		return nil, errRawEnd
	}
	if f.src[f.i] == '}' {
		f.i++
		return []byte("{}"), nil
	}
	var members []member
	seen := map[string]bool{}
	for {
		f.space()
		if f.i >= len(f.src) {
			return nil, errRawEnd
		}
		if f.src[f.i] != '"' {
			return nil, errors.New("jsontext: invalid character " +
				quoteChar(f.src[f.i]) + " at start of object name")
		}
		key, plain, err := f.text()
		if err != nil {
			return nil, err
		}
		if !f.opt.allowDup {
			if seen[string(plain)] {
				return nil, errors.New(
					"jsontext: duplicate object member name " + strconv.Quote(string(plain)))
			}
			seen[string(plain)] = true
		}
		f.space()
		if f.i >= len(f.src) || f.src[f.i] != ':' {
			if f.i >= len(f.src) {
				return nil, errRawEnd
			}
			return nil, errors.New("jsontext: invalid character " +
				quoteChar(f.src[f.i]) + " after object name")
		}
		f.i++
		value, err := f.value(depth + 1)
		if err != nil {
			return nil, err
		}
		text := append([]byte{}, key...)
		text = append(text, ':')
		if f.opt.expand {
			text = append(text, ' ')
		}
		text = append(text, value...)
		members = append(members, member{name: plain, text: text})
		f.space()
		if f.i >= len(f.src) {
			return nil, errRawEnd
		}
		switch f.src[f.i] {
		case ',':
			f.i++
		case '}':
			f.i++
			return f.joinObject(members, depth), nil
		default:
			return nil, errors.New("jsontext: invalid character " +
				quoteChar(f.src[f.i]) + " after object value")
		}
	}
}

// joinObject writes the members of 1 object.
func (f *rawFormatter) joinObject(members []member, depth int) []byte {
	if f.opt.reorder {
		sort.SliceStable(members, func(a, b int) bool {
			if c := compareUTF16(members[a].name, members[b].name); c != 0 {
				return c < 0
			}
			return compareUTF16(members[a].text, members[b].text) < 0
		})
	}
	out := []byte{'{'}
	for k, m := range members {
		if k > 0 {
			out = append(out, ',')
		}
		out = f.line(out, depth+1)
		out = append(out, m.text...)
	}
	out = f.line(out, depth)
	return append(out, '}')
}

func (f *rawFormatter) array(depth int) ([]byte, error) {
	f.i++
	f.space()
	if f.i >= len(f.src) {
		return nil, errRawEnd
	}
	if f.src[f.i] == ']' {
		f.i++
		return []byte("[]"), nil
	}
	out := []byte{'['}
	first := true
	for {
		if !first {
			out = append(out, ',')
		}
		out = f.line(out, depth+1)
		first = false
		value, err := f.value(depth + 1)
		if err != nil {
			return nil, err
		}
		out = append(out, value...)
		f.space()
		if f.i >= len(f.src) {
			return nil, errRawEnd
		}
		switch f.src[f.i] {
		case ',':
			f.i++
		case ']':
			f.i++
			out = f.line(out, depth)
			return append(out, ']'), nil
		default:
			return nil, errors.New("jsontext: invalid character " +
				quoteChar(f.src[f.i]) + " after array element")
		}
	}
}

// text reads 1 JSON string. The first result is the text of the output, and
// the second result is the value of the string with each escape read.
func (f *rawFormatter) text() ([]byte, []byte, error) {
	start := f.i
	plain, err := f.unquote()
	if err != nil {
		return nil, nil, err
	}
	if f.opt.rawStrings {
		return f.src[start:f.i], plain, nil
	}
	return appendQuoteJSON(nil, plain), plain, nil
}

// unquote reads 1 JSON string and gives its value. It reports each escape that
// names no code point of Unicode.
func (f *rawFormatter) unquote() ([]byte, error) {
	f.i++
	out := []byte{}
	for {
		if f.i >= len(f.src) {
			return nil, errRawEnd
		}
		c := f.src[f.i]
		switch {
		case c == '"':
			f.i++
			return out, nil
		case c == '\\':
			r, err := f.escape()
			if err != nil {
				return nil, err
			}
			out = utf8.AppendRune(out, r)
		case c < 0x20:
			return nil, errors.New("jsontext: invalid character " +
				quoteChar(c) + " within string")
		case c < utf8.RuneSelf:
			out = append(out, c)
			f.i++
		default:
			r, size := utf8.DecodeRune(f.src[f.i:])
			if r == utf8.RuneError && size == 1 {
				if !f.opt.allowBadUTF8 {
					return nil, errors.New("jsontext: invalid UTF-8 within string")
				}
				out = append(out, c)
				f.i++
				continue
			}
			out = append(out, f.src[f.i:f.i+size]...)
			f.i += size
		}
	}
}

// escape reads 1 escape of a JSON string and gives its code point. A pair of
// surrogates gives 1 code point.
func (f *rawFormatter) escape() (rune, error) {
	if f.i+1 >= len(f.src) {
		return 0, errRawEnd
	}
	c := f.src[f.i+1]
	switch c {
	case '"', '\\', '/':
		f.i += 2
		return rune(c), nil
	case 'b':
		f.i += 2
		return '\b', nil
	case 'f':
		f.i += 2
		return '\f', nil
	case 'n':
		f.i += 2
		return '\n', nil
	case 'r':
		f.i += 2
		return '\r', nil
	case 't':
		f.i += 2
		return '\t', nil
	case 'u':
		first, err := f.hex4()
		if err != nil {
			return 0, err
		}
		if !utf16.IsSurrogate(rune(first)) {
			return rune(first), nil
		}
		if f.i+1 < len(f.src) && f.src[f.i] == '\\' && f.src[f.i+1] == 'u' {
			second, err := f.hex4()
			if err != nil {
				return 0, err
			}
			r := utf16.DecodeRune(rune(first), rune(second))
			if r != utf8.RuneError {
				return r, nil
			}
			if f.opt.allowBadUTF8 {
				return utf8.RuneError, nil
			}
			return 0, errRawSurrogate
		}
		if f.opt.allowBadUTF8 {
			return utf8.RuneError, nil
		}
		return 0, errRawSurrogate
	}
	return 0, errors.New("jsontext: invalid escape sequence " +
		strconv.Quote(string(f.src[f.i:f.i+2])) + " within string")
}

var errRawSurrogate = errors.New("jsontext: invalid UTF-8 within string")

// hex4 reads the 4 digits of a \u escape and gives their value.
func (f *rawFormatter) hex4() (int, error) {
	if f.i+6 > len(f.src) {
		return 0, errRawEnd
	}
	value := 0
	for k := 2; k < 6; k++ {
		c := f.src[f.i+k]
		if !isHex(c) {
			return 0, errors.New("jsontext: invalid escape sequence " +
				strconv.Quote(string(f.src[f.i:f.i+k+1])) + " within string")
		}
		value = value*16 + hexValue(c)
	}
	f.i += 6
	return value, nil
}

func hexValue(c byte) int {
	switch {
	case c >= '0' && c <= '9':
		return int(c - '0')
	case c >= 'a' && c <= 'f':
		return int(c-'a') + 10
	}
	return int(c-'A') + 10
}

// appendQuoteJSON writes the shortest JSON text of a value of a string.
func appendQuoteJSON(dst []byte, s []byte) []byte {
	dst = append(dst, '"')
	for _, c := range s {
		switch c {
		case '"':
			dst = append(dst, '\\', '"')
		case '\\':
			dst = append(dst, '\\', '\\')
		case '\b':
			dst = append(dst, '\\', 'b')
		case '\f':
			dst = append(dst, '\\', 'f')
		case '\n':
			dst = append(dst, '\\', 'n')
		case '\r':
			dst = append(dst, '\\', 'r')
		case '\t':
			dst = append(dst, '\\', 't')
		default:
			if c < 0x20 {
				dst = append(dst, '\\', 'u', '0', '0',
					hexDigit(c>>4), hexDigit(c&0xF))
				continue
			}
			dst = append(dst, c)
		}
	}
	return append(dst, '"')
}

// number reads 1 JSON number and writes its new form.
func (f *rawFormatter) number() ([]byte, error) {
	start := f.i
	if f.i < len(f.src) && f.src[f.i] == '-' {
		f.i++
	}
	if !f.digits(true) {
		return nil, f.badNumber()
	}
	isFloat := false
	if f.i < len(f.src) && f.src[f.i] == '.' {
		isFloat = true
		f.i++
		if !f.digits(false) {
			return nil, f.badNumber()
		}
	}
	if f.i < len(f.src) && (f.src[f.i] == 'e' || f.src[f.i] == 'E') {
		isFloat = true
		f.i++
		if f.i < len(f.src) && (f.src[f.i] == '+' || f.src[f.i] == '-') {
			f.i++
		}
		if !f.digits(false) {
			return nil, f.badNumber()
		}
	}
	text := f.src[start:f.i]
	if !f.opt.canonNumbers {
		return append([]byte{}, text...), nil
	}
	// An integer of 15 digits or less already holds its canonical text, and
	// the number -0 always becomes 0.
	if string(text) != "-0" && !isFloat && len(text) < 16 {
		return append([]byte{}, text...), nil
	}
	value, _ := strconv.ParseFloat(string(text), 64)
	return appendCanonicalFloat(nil, value), nil
}

// digits reads a run of decimal digits. A leading zero takes no second digit,
// exactly as the grammar of JSON asks.
func (f *rawFormatter) digits(leading bool) bool {
	start := f.i
	for f.i < len(f.src) && f.src[f.i] >= '0' && f.src[f.i] <= '9' {
		f.i++
		if leading && f.src[start] == '0' {
			break
		}
	}
	return f.i > start
}

func (f *rawFormatter) badNumber() error {
	if f.i >= len(f.src) {
		return errRawEnd
	}
	return errors.New("jsontext: invalid character " +
		quoteChar(f.src[f.i]) + " within number")
}

// appendCanonicalFloat writes a 64-bit float in the text of ECMA-262, which
// RFC 8785 asks for.
func appendCanonicalFloat(dst []byte, value float64) []byte {
	switch {
	case value == 0:
		value = 0
	case math.IsInf(value, +1):
		value = math.MaxFloat64
	case math.IsInf(value, -1):
		value = -math.MaxFloat64
	}
	form := byte('f')
	abs := math.Abs(value)
	if abs != 0 && (abs < 1e-6 || abs >= 1e21) {
		form = 'e'
	}
	dst = strconv.AppendFloat(dst, value, form, -1, 64)
	if form == 'e' {
		// The text e-09 becomes e-9.
		n := len(dst)
		if n >= 4 && dst[n-4] == 'e' && dst[n-3] == '-' && dst[n-2] == '0' {
			dst[n-2] = dst[n-1]
			dst = dst[:n-1]
		}
	}
	return dst
}

// word reads a literal of the grammar: true, false, or null.
func (f *rawFormatter) word(want string) ([]byte, error) {
	for k := 0; k < len(want); k++ {
		if f.i+k >= len(f.src) {
			return nil, errRawEnd
		}
		if f.src[f.i+k] != want[k] {
			return nil, errors.New("jsontext: invalid character " +
				quoteChar(f.src[f.i+k]) + " within literal " + want)
		}
	}
	f.i += len(want)
	return []byte(want), nil
}

// compareUTF16 compares 2 texts as sequences of code units of UTF-16, which
// RFC 8785 asks for.
func compareUTF16(x, y []byte) int {
	a := utf16.Encode([]rune(string(x)))
	b := utf16.Encode([]rune(string(y)))
	for k := 0; k < len(a) && k < len(b); k++ {
		if a[k] != b[k] {
			if a[k] < b[k] {
				return -1
			}
			return 1
		}
	}
	switch {
	case len(a) < len(b):
		return -1
	case len(a) > len(b):
		return 1
	}
	return 0
}
