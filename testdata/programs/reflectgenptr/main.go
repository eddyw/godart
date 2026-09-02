package main

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"reflect"
)

// stamp holds a decoder with a pointer receiver, thus only *stamp implements
// GobDecoder.
type stamp struct{ N int }

func (s stamp) GobEncode() ([]byte, error) { return []byte{byte(s.N)}, nil }

func (s *stamp) GobDecode(b []byte) error {
	s.N = int(b[0]) * 2
	return nil
}

type plain struct{ N int }

var decoderType = reflect.TypeOf((*gob.GobDecoder)(nil)).Elem()
var encoderType = reflect.TypeOf((*gob.GobEncoder)(nil)).Elem()

// decodes reports whether a pointer to the type parameter holds GobDecode. The
// descriptor of that pointer stands inside the generic function.
func decodes[T any]() bool {
	var v T
	return reflect.TypeOf(&v).Implements(decoderType)
}

// text names the type of a pointer to the type parameter.
func text[T any]() string {
	var v T
	t := reflect.TypeOf(&v)
	return fmt.Sprint(t, " ", t.Kind(), " ", t.NumMethod())
}

// roundTrip sends a value through gob and gives the value again.
func roundTrip[T any](in T) (T, error) {
	var buf bytes.Buffer
	var out T
	if err := gob.NewEncoder(&buf).Encode(&in); err != nil {
		return out, err
	}
	err := gob.NewDecoder(&buf).Decode(&out)
	return out, err
}

func main() {
	fmt.Println(decodes[stamp](), decodes[plain](), decodes[int]())
	fmt.Println(text[stamp]())
	fmt.Println(text[plain]())
	fmt.Println(reflect.TypeOf(&stamp{}).Implements(decoderType))
	fmt.Println(reflect.TypeOf(stamp{}).Implements(encoderType))

	out, err := roundTrip(stamp{N: 3})
	fmt.Println(out.N, err)
}
