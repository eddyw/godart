// The witness of errors.As with an As method and with a bad target.
package main

import (
	"errors"
	"fmt"
	"io/fs"
)

type wrapped struct{}

func (wrapped) Error() string { return "wrapped" }

func (wrapped) As(target any) bool {
	if p, ok := target.(**fs.PathError); ok {
		*p = &fs.PathError{Op: "op", Path: "p", Err: errors.New("cause")}
		return true
	}
	return false
}

type myErr struct{ code int }

func (m *myErr) Error() string { return fmt.Sprintf("my %d", m.code) }

func try(name string, f func()) {
	defer func() {
		if e := recover(); e != nil {
			fmt.Printf("%s: panic %v\n", name, e)
			return
		}
		fmt.Printf("%s: no panic\n", name)
	}()
	f()
}

func main() {
	var pe *fs.PathError
	fmt.Println("as method", errors.As(fmt.Errorf("w: %w", wrapped{}), &pe), pe)

	var me *myErr
	fmt.Println("as chain", errors.As(fmt.Errorf("a: %w", &myErr{7}), &me), me)

	try("nil target", func() { errors.As(errors.New("x"), nil) })
	try("no pointer", func() { errors.As(errors.New("x"), 1) })
	var n int
	try("no error type", func() { errors.As(errors.New("x"), &n) })

	fmt.Println("is", errors.Is(fmt.Errorf("w: %w", fs.ErrNotExist), fs.ErrNotExist))
}
