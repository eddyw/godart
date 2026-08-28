package main

import (
	"errors"
	"fmt"
)

type notFound struct{ name string }

func (e *notFound) Error() string { return e.name + " not found" }

type wrapError struct {
	text string
	err  error
}

func (e *wrapError) Error() string { return e.text + ": " + e.err.Error() }

func (e *wrapError) Unwrap() error { return e.err }

var errBase = errors.New("the base error")

func main() {
	fmt.Println("new:", errBase.Error())

	wrapped := &wrapError{text: "outer", err: errBase}
	fmt.Println("wrap:", wrapped.Error())
	fmt.Println("unwrap:", errors.Unwrap(wrapped) == errBase)
	fmt.Println("is:", errors.Is(wrapped, errBase), errors.Is(errBase, wrapped))

	missing := &notFound{name: "config"}
	chain := &wrapError{text: "read", err: missing}
	var target *notFound
	fmt.Println("as:", errors.As(chain, &target), target.name)

	var other *wrapError
	fmt.Println("as of the outer:", errors.As(chain, &other), other.text)

	typed, ok := errors.AsType[*notFound](chain)
	fmt.Println("astype:", ok, typed.name)
	_, missed := errors.AsType[*notFound](errBase)
	fmt.Println("astype of a plain error:", missed)

	joined := errors.Join(errBase, missing, nil)
	fmt.Println("join:", joined.Error())
	fmt.Println("join is:", errors.Is(joined, errBase), errors.Is(joined, missing))
	var fromJoin *notFound
	fmt.Println("join as:", errors.As(joined, &fromJoin), fromJoin.name)
	fmt.Println("join of nil:", errors.Join(nil, nil) == nil)

	fmt.Println("errorf:", fmt.Errorf("wrapped: %w", errBase).Error())
}
