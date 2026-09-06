// The witness of errors.As with a target of a struct type that is not a
// pointer. The method Error of the type reads a value receiver, thus the
// error holds the struct itself.
//
// Go:   true / name
// gart: false / (an empty text)
package main

import (
	"errors"
	"fmt"
)

type valueError struct{ Field string }

func (v valueError) Error() string { return "bad field " + v.Field }

func main() {
	var err error = valueError{"name"}
	var target valueError
	fmt.Println(errors.As(err, &target))
	fmt.Println(target.Field)
}
