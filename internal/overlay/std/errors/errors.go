//go:build dart

// Package errors gives the error values of the standard library.
package errors

import "syscall/dart"

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

// Is reports whether err or an error inside it is equal to target.
//
// An error that holds an Is method decides for itself, and an error that wraps
// a list gives each error of the list.
func Is(err, target error) bool {
	if target == nil {
		return err == target
	}
	for {
		if err == target {
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
			for _, inner := range x.Unwrap() {
				if Is(inner, target) {
					return true
				}
			}
			return false
		default:
			return false
		}
	}
}

// As finds the first error of the chain that holds the type of the target, and
// it writes that error to the target.
//
// The target must be a pointer to a type that holds the error interface. Go
// also accepts a pointer to an interface type, and such a target stays outside
// the subset: the runtime library holds no method name of an interface type.
func As(err error, target any) bool {
	if target == nil {
		panic("errors: target cannot be nil")
	}
	for {
		if dart.AssignError(err, target) {
			return true
		}
		switch x := err.(type) {
		case wrapped:
			err = x.Unwrap()
			if err == nil {
				return false
			}
		case multi:
			for _, inner := range x.Unwrap() {
				if As(inner, target) {
					return true
				}
			}
			return false
		default:
			return false
		}
	}
}

// AsType finds the first error of the chain that holds the type E. Go 1.27 gives
// this form, and it needs no pointer.
func AsType[E error](err error) (E, bool) {
	for {
		if e, ok := any(err).(E); ok {
			return e, true
		}
		switch x := err.(type) {
		case wrapped:
			err = x.Unwrap()
			if err == nil {
				var zero E
				return zero, false
			}
		case multi:
			for _, inner := range x.Unwrap() {
				if e, ok := AsType[E](inner); ok {
					return e, true
				}
			}
			var zero E
			return zero, false
		default:
			var zero E
			return zero, false
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
