// The witness of errors.As with a pointer to an interface type. Go writes the
// first error of the chain that holds the interface into the target, and the
// error of Errorf holds it already.
//
// Go:   true / outer: code 500: boom
// gart: panic: runtime error: errors.As with a pointer to an interface type
//
//	is outside the subset
package main

import (
	"errors"
	"fmt"
)

type codeError struct{ Code int }

func (c *codeError) Error() string { return fmt.Sprint("code ", c.Code, ": boom") }

func main() {
	err := fmt.Errorf("outer: %w", &codeError{500})
	var target interface{ Error() string }
	fmt.Println(errors.As(err, &target))
	fmt.Println(target.Error())
}
