// The program proves the text that a panic writes for its value. Go calls
// Error on a value that implements error, and String on a value that
// implements Stringer. Read preprintpanics of runtime/panic.go.
//
// The deferred call recovers the first panic and starts a second one, thus the
// report holds the whole chain and each of the 2 texts.
package main

import (
	"errors"
	"fmt"
)

type shown struct{ n int }

func (s shown) String() string { return fmt.Sprintf("shown(%d)", s.n) }

func main() {
	defer func() {
		recover()
		panic(errors.New("boom"))
	}()
	fmt.Println("start")
	panic(shown{3})
}
