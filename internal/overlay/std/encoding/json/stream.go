//go:build dart

package json

import (
	"bytes"
	"errors"
	"io"
)

// A Decoder reads a stream of JSON values from an input stream.
type Decoder struct {
	r      io.Reader
	buf    []byte
	off    int
	done   bool
	strict bool
	number bool
	read   int64
	// state and stack hold the position of Token inside the value. Read
	// token.go.
	state int
	stack []int
}

// NewDecoder gives a Decoder that reads from r.
func NewDecoder(r io.Reader) *Decoder { return &Decoder{r: r} }

// DisallowUnknownFields makes Decode report a member of the input that matches
// no field of the target struct.
func (d *Decoder) DisallowUnknownFields() { d.strict = true }

// Decode reads the next JSON value of the stream into the value that v points
// to. It gives io.EOF after the last value.
//
// A call of Token before Decode leaves the reader inside an array or inside an
// object, and Decode then reads the next element of that value.
func (d *Decoder) Decode(v any) error {
	if err := d.prepare(); err != nil {
		return err
	}
	n, err := d.next()
	if err != nil {
		return err
	}
	data := d.buf[d.off : d.off+n]
	d.off += n
	d.read += int64(n)
	if err := unmarshal(data, v, d.strict, d.number); err != nil {
		return err
	}
	d.valueEnd()
	return nil
}

// More reports whether the stream holds another element of the current array or
// object.
func (d *Decoder) More() bool {
	c, ok := d.peek()
	return ok && c != ']' && c != '}'
}

// Buffered gives a reader of the bytes that Decode read from the stream and did
// not use.
func (d *Decoder) Buffered() io.Reader {
	return bytes.NewReader(d.buf[d.off:])
}

// InputOffset gives the count of bytes of the stream that Decode used.
func (d *Decoder) InputOffset() int64 { return d.read }

// UseNumber makes Decode read a number into a Number, and not into a float64,
// when the target is an empty interface.
func (d *Decoder) UseNumber() { d.number = true }

// next gives the length of the next value of the buffer, and it reads more
// bytes until the buffer holds that value.
func (d *Decoder) next() (int, error) {
	for {
		if d.off < len(d.buf) {
			n, err := scanValue(d.buf[d.off:], d.done)
			if err == nil {
				return n, nil
			}
			if err != errEnd {
				return 0, err
			}
		}
		if d.done {
			if !d.hasValue() {
				return 0, io.EOF
			}
			return 0, &SyntaxError{
				msg:    "unexpected end of JSON input",
				Offset: d.read + int64(len(d.buf)-d.off),
			}
		}
		if err := d.fill(); err != nil {
			return 0, err
		}
	}
}

// hasValue reports whether the rest of the buffer holds a byte that is not a
// space.
func (d *Decoder) hasValue() bool {
	for _, c := range d.buf[d.off:] {
		if !isSpace(c) {
			return true
		}
	}
	return false
}

// peek gives the next byte of the stream that is not a space, and it leaves
// that byte in the buffer.
func (d *Decoder) peek() (byte, bool) {
	for {
		for d.off < len(d.buf) && isSpace(d.buf[d.off]) {
			d.off++
			d.read++
		}
		if d.off < len(d.buf) {
			return d.buf[d.off], true
		}
		if d.done {
			return 0, false
		}
		if err := d.fill(); err != nil {
			return 0, false
		}
	}
}

func (d *Decoder) fill() error {
	if d.off > 0 {
		d.buf = append(d.buf[:0], d.buf[d.off:]...)
		d.off = 0
	}
	chunk := make([]byte, 4096)
	n, err := d.r.Read(chunk)
	d.buf = append(d.buf, chunk[:n]...)
	if err == io.EOF {
		d.done = true
		return nil
	}
	return err
}

// An Encoder writes a stream of JSON values to an output stream.
type Encoder struct {
	w      io.Writer
	prefix string
	indent string
	err    error
}

// NewEncoder gives an Encoder that writes to w.
func NewEncoder(w io.Writer) *Encoder { return &Encoder{w: w} }

// SetIndent makes Encode write 1 line for each member. Each line starts with
// the prefix, and each level adds the indent. An empty indent gives the compact
// form again.
func (e *Encoder) SetIndent(prefix, indent string) {
	e.prefix = prefix
	e.indent = indent
}

// SetEscapeHTML is present, and it changes nothing. The encoder writes `<`,
// `>`, and `&` as themselves. Read docs/parity.md.
func (e *Encoder) SetEscapeHTML(on bool) {}

// Encode writes the JSON form of v and a newline.
func (e *Encoder) Encode(v any) error {
	if e.err != nil {
		return e.err
	}
	b, err := Marshal(v)
	if err != nil {
		return err
	}
	if e.prefix != "" || e.indent != "" {
		var out bytes.Buffer
		if err := Indent(&out, b, e.prefix, e.indent); err != nil {
			return err
		}
		b = out.Bytes()
	}
	b = append(b, '\n')
	if _, err := e.w.Write(b); err != nil {
		e.err = err
		return err
	}
	return nil
}

// A RawMessage holds a JSON value in its own bytes. It waits for a later
// decode, and it writes its bytes without a change.
type RawMessage []byte

// MarshalJSON gives the bytes of m.
func (m RawMessage) MarshalJSON() ([]byte, error) {
	if m == nil {
		return []byte("null"), nil
	}
	return m, nil
}

// UnmarshalJSON writes the bytes of the value into m.
func (m *RawMessage) UnmarshalJSON(data []byte) error {
	if m == nil {
		return errors.New("json.RawMessage: UnmarshalJSON on nil pointer")
	}
	*m = append((*m)[0:0], data...)
	return nil
}
