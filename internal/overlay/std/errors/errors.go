//go:build dart

// Package errors gives the error values of the standard library.
package errors

import (
	"internal/reflectlite"
	"syscall/dart"
)

type errorString struct {
	text string
}

func (e *errorString) Error() string {
	return e.text
}

// New returns an error that holds the text.
func New(text string) error {
	return &errorString{text: text}
}

// ErrUnsupported reports an operation that the environment does not hold. A
// caller reads it with Is.
var ErrUnsupported = New("unsupported operation")

type wrapped interface {
	Unwrap() error
}

type multi interface {
	Unwrap() []error
}

// Unwrap returns the error inside err, or nil.
func Unwrap(err error) error {
	inner, ok := err.(wrapped)
	if !ok {
		return nil
	}
	return inner.Unwrap()
}

// Is reports whether any error in err's tree matches target.
//
// The tree consists of err itself, followed by the errors obtained by repeatedly
// calling its Unwrap() error or Unwrap() []error method. When err wraps multiple
// errors, Is examines err followed by a depth-first traversal of its children.
//
// An error is considered to match a target if it is equal to that target or if
// it implements a method Is(error) bool such that Is(target) returns true.
//
// A target that holds no comparison, for example a slice, never equals err, and
// a comparison of the 2 would panic. Go asks the type of the target first.
func Is(err, target error) bool {
	if err == nil || target == nil {
		return err == target
	}
	isComparable := reflectlite.TypeOf(target).Comparable()
	return is(err, target, isComparable)
}

func is(err, target error, targetComparable bool) bool {
	for {
		if targetComparable && err == target {
			return true
		}
		if x, ok := err.(interface{ Is(error) bool }); ok && x.Is(target) {
			return true
		}
		switch x := err.(type) {
		case wrapped:
			err = x.Unwrap()
			if err == nil {
				return false
			}
		case multi:
			for _, err := range x.Unwrap() {
				if is(err, target, targetComparable) {
					return true
				}
			}
			return false
		default:
			return false
		}
	}
}

// As finds the first error in err's tree that matches target, and if one is
// found, sets target to that error value and returns true. Otherwise, it
// returns false.
//
// An error matches target if the error's concrete type is assignable to the
// value pointed to by target, or if the error has a method As(any) bool such
// that As(target) returns true.
//
// As panics if target is not a non-nil pointer to either a type that
// implements error, or to any interface type.
func As(err error, target any) bool {
	if err == nil {
		return false
	}
	if target == nil {
		panic("errors: target cannot be nil")
	}
	dart.CheckErrorTarget(target)
	return as(err, target)
}

func as(err error, target any) bool {
	for {
		if dart.AssignError(err, target) {
			return true
		}
		if x, ok := err.(interface{ As(any) bool }); ok && x.As(target) {
			return true
		}
		switch x := err.(type) {
		case wrapped:
			err = x.Unwrap()
			if err == nil {
				return false
			}
		case multi:
			for _, err := range x.Unwrap() {
				if err == nil {
					continue
				}
				if as(err, target) {
					return true
				}
			}
			return false
		default:
			return false
		}
	}
}

// AsType finds the first error in err's tree that matches the type E, and if
// one is found, returns that error value and true. Otherwise, it returns the
// zero value of E and false.
//
// An error err matches the type E if the type assertion err.(E) holds, or if
// the error has a method As(any) bool such that err.As(target) returns true
// when target is a non-nil *E.
func AsType[E error](err error) (E, bool) {
	if err == nil {
		var zero E
		return zero, false
	}
	var pe *E
	return asType(err, &pe)
}

func asType[E error](err error, ppe **E) (_ E, _ bool) {
	for {
		if e, ok := err.(E); ok {
			return e, true
		}
		if x, ok := err.(interface{ As(any) bool }); ok {
			if *ppe == nil {
				*ppe = new(E)
			}
			if x.As(*ppe) {
				return **ppe, true
			}
		}
		switch x := err.(type) {
		case wrapped:
			err = x.Unwrap()
			if err == nil {
				return
			}
		case multi:
			for _, err := range x.Unwrap() {
				if err == nil {
					continue
				}
				if x, ok := asType(err, ppe); ok {
					return x, true
				}
			}
			return
		default:
			return
		}
	}
}

// joinError holds each error of a Join call.
type joinError struct {
	errs []error
}

func (e *joinError) Error() string {
	out := ""
	for i, err := range e.errs {
		if i > 0 {
			out += "\n"
		}
		out += err.Error()
	}
	return out
}

func (e *joinError) Unwrap() []error { return e.errs }

// Join gives 1 error that holds each error that is not nil. It gives nil when
// each error is nil.
func Join(errs ...error) error {
	n := 0
	for _, err := range errs {
		if err != nil {
			n++
		}
	}
	if n == 0 {
		return nil
	}
	out := &joinError{errs: make([]error, 0, n)}
	for _, err := range errs {
		if err != nil {
			out.errs = append(out.errs, err)
		}
	}
	return out
}
