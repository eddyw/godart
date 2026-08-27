//go:build dart

// Package errors gives the error values of the standard library.
package errors

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

// Unwrap returns the error inside err, or nil.
func Unwrap(err error) error {
	inner, ok := err.(wrapped)
	if !ok {
		return nil
	}
	return inner.Unwrap()
}

// Is reports whether err or an error inside it is equal to target.
func Is(err, target error) bool {
	for err != nil {
		if err == target {
			return true
		}
		err = Unwrap(err)
	}
	return false
}
