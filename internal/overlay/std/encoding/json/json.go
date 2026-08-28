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
	"errors"
	"syscall/dart"
)

// Marshal gives the JSON form of a value.
//
// A struct gives an object. The name of each member comes from the json tag of
// the field, and the name of the field gives it when the tag holds none. A tag
// of `-` drops the field, and `omitempty` drops a field that holds the zero
// value.
func Marshal(v any) ([]byte, error) {
	text, cause := dart.MarshalJSON(v, "")
	if cause != "" {
		return nil, &UnsupportedTypeError{Text: cause}
	}
	return []byte(text), nil
}

// MarshalIndent gives the JSON form with 1 line for each member.
//
// Go takes a prefix and an indent. The Dart encoder holds no prefix, thus this
// function rejects a prefix that is not empty.
func MarshalIndent(v any, prefix, indent string) ([]byte, error) {
	if prefix != "" {
		return nil, errors.New("json: gart writes no prefix")
	}
	if indent == "" {
		indent = " "
	}
	text, cause := dart.MarshalJSON(v, indent)
	if cause != "" {
		return nil, &UnsupportedTypeError{Text: cause}
	}
	return []byte(text), nil
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
func Unmarshal(data []byte, v any) error {
	if cause := dart.UnmarshalJSON(v, string(data)); cause != "" {
		return &SyntaxError{Text: cause}
	}
	return nil
}

// SyntaxError names a JSON text that holds no value of the target type.
type SyntaxError struct {
	Text string
}

func (e *SyntaxError) Error() string { return e.Text }

// UnsupportedTypeError names a value that holds no JSON form.
type UnsupportedTypeError struct {
	Text string
}

func (e *UnsupportedTypeError) Error() string { return e.Text }
