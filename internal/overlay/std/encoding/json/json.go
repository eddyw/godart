//go:build dart

// Package json writes the JSON form of a Go value, and it reads such a form.
//
// The compiler cannot read the real source. That source reads the fields of a
// struct through the package reflect, and the subset holds no general reflect.
//
// The compiler writes a type descriptor for each named type, and that descriptor
// holds the name, the json tag, and a read of each field. The Dart side walks
// that descriptor, thus this package needs no code generator.
package json

import (
	"bytes"
	"errors"
	"strconv"
	"strings"
	"syscall/dart"
)

// Marshal gives the JSON form of a value.
//
// A struct gives an object. The name of each member comes from the json tag of
// the field, and the name of the field gives it when the tag holds none. A tag
// of `-` drops the field, and `omitempty` drops a field that holds the zero
// value.
//
// A type that holds the method MarshalJSON writes its own form.
func Marshal(v any) ([]byte, error) {
	text, cause := dart.MarshalJSON(v)
	if cause != "" {
		return nil, marshalCause(cause)
	}
	return []byte(text), nil
}

// MarshalIndent gives the JSON form with 1 line for each member. Each line
// starts with the prefix, and each level adds the indent.
func MarshalIndent(v any, prefix, indent string) ([]byte, error) {
	b, err := Marshal(v)
	if err != nil {
		return nil, err
	}
	var out bytes.Buffer
	if err := Indent(&out, b, prefix, indent); err != nil {
		return nil, err
	}
	return out.Bytes(), nil
}

// Unmarshal reads the JSON form into the value that v points to.
//
// A JSON object fills a struct. The name of each member matches the json tag of
// a field, and the name of the field matches when the tag holds none. A match
// that differs only in case counts, exactly as in Go. A member with no field
// stays out.
//
// A JSON value fills an empty interface with the types of Go: a bool, a
// float64, a string, a []interface{}, a map[string]interface{}, or nil.
//
// A type that holds the method UnmarshalJSON reads its own form.
func Unmarshal(data []byte, v any) error {
	return unmarshal(data, v, false, false)
}

// unmarshal reads 1 value. The flag strict reports a member that matches no
// field of the target struct.
func unmarshal(data []byte, v any, strict, useNumber bool) error {
	n, err := scanValue(data, true)
	if err != nil {
		if err == errEnd {
			return &SyntaxError{
				msg:    "unexpected end of JSON input",
				Offset: int64(len(data)),
			}
		}
		return err
	}
	for i, c := range data[n:] {
		if !isSpace(c) {
			return &SyntaxError{
				msg:    "invalid character " + quoteChar(c) + " after top-level value",
				Offset: int64(n + i + 1),
			}
		}
	}
	if cause := dart.UnmarshalJSON(v, string(data), strict, useNumber); cause != "" {
		return unmarshalCause(cause)
	}
	return nil
}

// A Marshaler writes its own JSON form. Marshal calls the method for a value
// whose type holds it.
type Marshaler interface {
	MarshalJSON() ([]byte, error)
}

// An Unmarshaler reads its own JSON form. Unmarshal calls the method for a
// value whose type holds it.
type Unmarshaler interface {
	UnmarshalJSON([]byte) error
}

// HTMLEscape writes the JSON text of src into dst with `<`, `>`, `&`, U+2028,
// and U+2029 in their escaped form. A page that holds the text inside a script
// element then reads 1 value.
func HTMLEscape(dst *bytes.Buffer, src []byte) {
	for i := 0; i < len(src); {
		c := src[i]
		switch {
		case c == '<' || c == '>' || c == '&':
			dst.WriteString(`\u00`)
			dst.WriteByte(hexDigit(c >> 4))
			dst.WriteByte(hexDigit(c & 0xF))
			i++
		case c == 0xE2 && i+2 < len(src) && src[i+1] == 0x80 &&
			src[i+2]&^1 == 0xA8:
			dst.WriteString(`\u202`)
			dst.WriteByte(hexDigit(src[i+2] & 1))
			i += 3
		default:
			dst.WriteByte(c)
			i++
		}
	}
}

func hexDigit(v byte) byte {
	const digits = "0123456789abcdef"
	return digits[v]
}

// An InvalidUTF8Error names a string that holds no valid UTF-8. Go marks the
// type as deprecated, and no path of this package gives it.
type InvalidUTF8Error struct {
	S string
}

func (e *InvalidUTF8Error) Error() string {
	return "json: invalid UTF-8 in string: " + strconv.Quote(e.S)
}

// An UnmarshalFieldError names a member of the input that matches no field of
// the target. Go marks the type as deprecated, and no path of this package
// gives it.
type UnmarshalFieldError struct {
	Key   string
	Field string
}

func (e *UnmarshalFieldError) Error() string {
	return "json: cannot unmarshal object key " + strconv.Quote(e.Key) +
		" into unexported field " + e.Field
}

// A Number holds the text of a JSON number. A field of this type keeps each
// digit of the input.
type Number string

// String gives the text of the number.
func (n Number) String() string { return string(n) }

// Float64 reads the number as a float.
func (n Number) Float64() (float64, error) {
	return strconv.ParseFloat(string(n), 64)
}

// Int64 reads the number as an integer.
func (n Number) Int64() (int64, error) {
	return strconv.ParseInt(string(n), 10, 64)
}

// A SyntaxError names a JSON text that holds no value.
type SyntaxError struct {
	msg string

	// Offset is the count of bytes that the reader took before the error.
	Offset int64
}

func (e *SyntaxError) Error() string { return e.msg }

// An UnmarshalTypeError names a JSON value that does not fit its target.
//
// Go holds the field Type of the type reflect.Type. This package cannot build
// such a value, thus the text of the type stands in an unexported field.
// docs/parity.md holds that gap.
type UnmarshalTypeError struct {
	// Value names the JSON value, for example `string` or `number`.
	Value string

	// Struct names the struct that holds the field, and Field names the path of
	// the field inside it.
	Struct string
	Field  string

	typeText string
}

func (e *UnmarshalTypeError) Error() string {
	if e.Struct != "" || e.Field != "" {
		return "json: cannot unmarshal " + e.Value + " into Go struct field " +
			e.Struct + "." + e.Field + " of type " + e.typeText
	}
	return "json: cannot unmarshal " + e.Value + " into Go value of type " + e.typeText
}

// An InvalidUnmarshalError names a target that Unmarshal cannot write.
type InvalidUnmarshalError struct {
	text string
}

func (e *InvalidUnmarshalError) Error() string {
	return "json: Unmarshal(" + e.text + ")"
}

// An UnsupportedTypeError names a type that holds no JSON form.
type UnsupportedTypeError struct {
	typeText string
}

func (e *UnsupportedTypeError) Error() string {
	return "json: unsupported type: " + e.typeText
}

// An UnsupportedValueError names a value that holds no JSON form, for example
// a float that holds no number.
type UnsupportedValueError struct {
	Str string
}

func (e *UnsupportedValueError) Error() string {
	return "json: unsupported value: " + e.Str
}

// A MarshalerError holds the cause that the method MarshalJSON of a type gave.
type MarshalerError struct {
	Err error

	typeText string
}

func (e *MarshalerError) Error() string {
	return "json: error calling MarshalJSON for type " + e.typeText + ": " +
		e.Err.Error()
}

func (e *MarshalerError) Unwrap() error { return e.Err }

// marshalCause builds the error of a cause that the Dart walk gives. A cause
// that starts with a null byte holds the code and the parts of a typed error.
func marshalCause(cause string) error {
	parts := causeParts(cause)
	if parts == nil {
		return errors.New(cause)
	}
	switch parts[0] {
	case "unsupportedType":
		return &UnsupportedTypeError{typeText: parts[1]}
	case "unsupportedValue":
		return &UnsupportedValueError{Str: parts[1]}
	case "marshaler":
		return &MarshalerError{typeText: parts[1], Err: errors.New(parts[2])}
	}
	return errors.New(cause)
}

func unmarshalCause(cause string) error {
	parts := causeParts(cause)
	if parts == nil {
		return errors.New(cause)
	}
	switch parts[0] {
	case "type":
		return &UnmarshalTypeError{
			Value:    parts[1],
			typeText: parts[2],
			Struct:   parts[3],
			Field:    parts[4],
		}
	case "unknown":
		return errors.New(`json: unknown field "` + parts[1] + `"`)
	case "invalid":
		return &InvalidUnmarshalError{text: parts[1]}
	}
	return errors.New(cause)
}

// causeParts splits a structured cause. It gives nil for a plain text.
func causeParts(cause string) []string {
	if cause == "" || cause[0] != 0 {
		return nil
	}
	return strings.Split(cause[1:], "\x00")
}
