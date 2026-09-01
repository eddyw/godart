//go:build dart

package json

import "io"

// The token reader of a Decoder.
//
// Token walks the input 1 piece at a time, and the state below names the place
// of the reader inside an array or inside an object. The 9 states are the
// states of Go, and Decode reads them too, thus a program can mix the 2 calls.

const (
	stateTopValue = iota
	stateArrayStart
	stateArrayValue
	stateArrayComma
	stateObjectStart
	stateObjectKey
	stateObjectColon
	stateObjectValue
	stateObjectComma
)

// A Delim is 1 of the 4 delimiters of JSON: [ ] { or }.
type Delim rune

func (d Delim) String() string { return string(d) }

// A Token holds 1 piece of a JSON value: a Delim, a bool, a float64, a Number,
// a string, or nil.
type Token any

// Token reads the next piece of the input and gives it.
//
// It gives a Delim for each bracket and each brace, and it gives the value of
// each other piece. The name of a member of an object comes as a string. The
// end of the input gives io.EOF.
func (d *Decoder) Token() (Token, error) {
	for {
		c, err := d.peekToken()
		if err != nil {
			return nil, err
		}
		switch c {
		case '[':
			if !d.valueAllowed() {
				return nil, d.tokenError(c)
			}
			d.take()
			d.stack = append(d.stack, d.state)
			d.state = stateArrayStart
			return Delim('['), nil
		case ']':
			if d.state != stateArrayStart && d.state != stateArrayComma {
				return nil, d.tokenError(c)
			}
			d.take()
			d.pop()
			return Delim(']'), nil
		case '{':
			if !d.valueAllowed() {
				return nil, d.tokenError(c)
			}
			d.take()
			d.stack = append(d.stack, d.state)
			d.state = stateObjectStart
			return Delim('{'), nil
		case '}':
			if d.state != stateObjectStart && d.state != stateObjectComma {
				return nil, d.tokenError(c)
			}
			d.take()
			d.pop()
			return Delim('}'), nil
		case ':':
			if d.state != stateObjectColon {
				return nil, d.tokenError(c)
			}
			d.take()
			d.state = stateObjectValue
		case ',':
			switch d.state {
			case stateArrayComma:
				d.take()
				d.state = stateArrayValue
			case stateObjectComma:
				d.take()
				d.state = stateObjectKey
			default:
				return nil, d.tokenError(c)
			}
		case '"':
			if d.state == stateObjectStart || d.state == stateObjectKey {
				var key string
				held := d.state
				d.state = stateTopValue
				err := d.Decode(&key)
				d.state = held
				if err != nil {
					return nil, err
				}
				d.state = stateObjectColon
				return key, nil
			}
			return d.tokenValue()
		default:
			return d.tokenValue()
		}
	}
}

// tokenValue reads 1 value that holds no delimiter.
func (d *Decoder) tokenValue() (Token, error) {
	if !d.valueAllowed() {
		c, err := d.peekToken()
		if err != nil {
			return nil, err
		}
		return nil, d.tokenError(c)
	}
	var v any
	held := d.number
	d.number = true
	err := d.Decode(&v)
	d.number = held
	if err != nil {
		return nil, err
	}
	if n, ok := v.(Number); ok && !d.number {
		f, err := n.Float64()
		if err != nil {
			return nil, err
		}
		return f, nil
	}
	return v, nil
}

// take steps over 1 byte that peekToken already found.
func (d *Decoder) take() {
	d.off++
	d.read++
}

// pop leaves an array or an object, and it marks the value as read.
func (d *Decoder) pop() {
	d.state = d.stack[len(d.stack)-1]
	d.stack = d.stack[:len(d.stack)-1]
	d.valueEnd()
}

// valueAllowed reports whether a value stands at the place of the reader.
func (d *Decoder) valueAllowed() bool {
	switch d.state {
	case stateTopValue, stateArrayStart, stateArrayValue, stateObjectValue:
		return true
	}
	return false
}

// valueEnd marks the end of 1 value, thus the next piece must be a comma or a
// closing delimiter.
func (d *Decoder) valueEnd() {
	switch d.state {
	case stateArrayStart, stateArrayValue:
		d.state = stateArrayComma
	case stateObjectValue:
		d.state = stateObjectComma
	}
}

// prepare steps over the comma or the colon that stands before the next value
// of a Decode that follows a Token.
func (d *Decoder) prepare() error {
	switch d.state {
	case stateArrayComma:
		if err := d.want(',', "expected comma after array element"); err != nil {
			return err
		}
		d.state = stateArrayValue
	case stateObjectColon:
		if err := d.want(':', "expected colon after object key"); err != nil {
			return err
		}
		d.state = stateObjectValue
	}
	return nil
}

func (d *Decoder) want(c byte, msg string) error {
	got, err := d.peekToken()
	if err != nil {
		return err
	}
	if got != c {
		return &SyntaxError{msg: msg, Offset: d.read}
	}
	d.take()
	return nil
}

// peekToken gives the next byte that is not a space, and it leaves that byte
// in the buffer.
func (d *Decoder) peekToken() (byte, error) {
	c, ok := d.peek()
	if !ok {
		return 0, io.EOF
	}
	return c, nil
}

func (d *Decoder) tokenError(c byte) error {
	msg := "invalid character " + quoteChar(c)
	switch d.state {
	case stateTopValue:
		msg += " looking for beginning of value"
	case stateArrayStart, stateArrayValue, stateObjectValue:
		msg += " looking for beginning of value"
	case stateArrayComma:
		msg += " after array element"
	case stateObjectStart, stateObjectKey:
		msg += " looking for beginning of object key string"
	case stateObjectColon:
		msg += " after object key"
	case stateObjectComma:
		msg += " after object key:value pair"
	}
	return &SyntaxError{msg: msg, Offset: d.read + 1}
}
