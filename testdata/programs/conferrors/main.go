// The program proves the error rules of Go: errors.New builds a value that
// compares by identity, fmt.Errorf with %w wraps another error, errors.Is
// walks the chain, errors.As finds a value of a type, errors.Join holds many
// errors, and a nil pointer inside an error interface is not a nil error.
package main

import (
	"errors"
	"fmt"
	"os"
	"strconv"
)

func row(label string, value any) {
	fmt.Printf("%-34s %v\n", label, value)
}

var (
	errBase   = errors.New("base failure")
	errOther  = errors.New("other failure")
	errSame   = errors.New("base failure")
	errSentry = errors.New("sentry")
)

func newAndCompare() {
	row("the text", errBase.Error())
	row("printed", errBase)
	row("2 names of 1 value", errBase == errBase)
	row("2 values of 1 text", errBase == errSame)
	row("Is of itself", errors.Is(errBase, errBase))
	row("Is of an equal text", errors.Is(errBase, errSame))
	row("Is of another", errors.Is(errBase, errOther))
	row("Is of nil", errors.Is(nil, errBase))
	row("Is nil nil", errors.Is(nil, nil))
	row("the type", fmt.Sprintf("%T", errBase))
	// An error inside a map and inside a slice.
	m := map[error]int{errBase: 1}
	row("as a map key", m[errBase])
	row("as a map gap", m[errOther])
	list := []error{errBase, errOther}
	row("in a slice", list)
	// The empty text.
	empty := errors.New("")
	row("an empty text", fmt.Sprintf("%q", empty.Error()))
	row("printed empty", fmt.Sprintf("%q", fmt.Sprint(empty)))
}

func wrapping() {
	wrapped := fmt.Errorf("read the file: %w", errBase)
	row("the text", wrapped.Error())
	row("Is of the base", errors.Is(wrapped, errBase))
	row("Is of another", errors.Is(wrapped, errOther))
	row("Unwrap", errors.Unwrap(wrapped))
	row("Unwrap of the base", errors.Unwrap(errBase))
	row("the type", fmt.Sprintf("%T", wrapped))

	// A chain of 3 levels.
	level2 := fmt.Errorf("layer 2: %w", wrapped)
	level3 := fmt.Errorf("layer 3: %w", level2)
	row("the whole text", level3.Error())
	row("Is at 3 levels", errors.Is(level3, errBase))
	row("Unwrap 1", errors.Unwrap(level3))
	row("Unwrap 2", errors.Unwrap(errors.Unwrap(level3)))
	row("Unwrap 3", errors.Unwrap(errors.Unwrap(errors.Unwrap(level3))))
	row("Unwrap 4", errors.Unwrap(errors.Unwrap(errors.Unwrap(errors.Unwrap(level3)))))

	// A format with no %w gives no chain.
	plain := fmt.Errorf("read the file: %v", errBase)
	row("with %%v", plain.Error())
	row("Is with %%v", errors.Is(plain, errBase))
	row("Unwrap with %%v", errors.Unwrap(plain))
	row("the type with %%v", fmt.Sprintf("%T", plain))

	// 2 verbs of %w in 1 call.
	two := fmt.Errorf("both: %w and %w", errBase, errOther)
	row("2 wraps", two.Error())
	row("Is of the first", errors.Is(two, errBase))
	row("Is of the second", errors.Is(two, errOther))
	row("the type of 2 wraps", fmt.Sprintf("%T", two))
	row("Unwrap of 2 wraps", errors.Unwrap(two))

	// A wrap of a nil error stands in conf-blocked/conffmtwrapnil. Go writes
	// the bad verb form there. Read REPORT-conform.md, item 13.
	row("Is of nil", errors.Is(errBase, nil))
}

// The custom error types prove Is, As, and Unwrap of a type of the program.
type codeError struct {
	Code int
	Msg  string
}

func (c *codeError) Error() string { return "code " + strconv.Itoa(c.Code) + ": " + c.Msg }

type valueError struct{ Field string }

func (v valueError) Error() string { return "bad field " + v.Field }

type wrapError struct {
	Msg   string
	Inner error
}

func (w *wrapError) Error() string { return w.Msg + ": " + w.Inner.Error() }

func (w *wrapError) Unwrap() error { return w.Inner }

// matchError holds an Is method of its own.
type matchError struct{ Kind string }

func (m matchError) Error() string { return "kind " + m.Kind }

func (m matchError) Is(target error) bool {
	other, ok := target.(matchError)
	return ok && (other.Kind == m.Kind || other.Kind == "any")
}

func customTypes() {
	var err error = &codeError{404, "not found"}
	row("the text", err.Error())
	var target *codeError
	row("As", errors.As(err, &target))
	row("the code", target.Code)
	var other valueError
	row("As of a wrong type", errors.As(err, &other))

	// As with a target of a struct type that is not a pointer stands in
	// conf-blocked/conferrorsasvalue. Read REPORT-conform.md, item 11.
	// AsType of Go 1.27 finds the same value with no pointer.
	var verr error = valueError{"name"}
	vtarget, ok := errors.AsType[valueError](verr)
	row("AsType of a value type", fmt.Sprint(vtarget.Field, " ", ok))
	row("the text of it", verr.Error())

	// As walks a chain.
	chain := fmt.Errorf("outer: %w", &wrapError{"middle", &codeError{500, "boom"}})
	row("the chain text", chain.Error())
	target = nil
	row("As through a chain", errors.As(chain, &target))
	row("the code of the chain", target.Code)
	var wtarget *wrapError
	row("As of the middle", errors.As(chain, &wtarget))
	row("the middle text", wtarget.Msg)

	// Is with a method of the type.
	var kind error = matchError{"read"}
	row("Is of the same kind", errors.Is(kind, matchError{"read"}))
	row("Is of any", errors.Is(kind, matchError{"any"}))
	row("Is of another kind", errors.Is(kind, matchError{"write"}))
	wrappedKind := fmt.Errorf("outer: %w", kind)
	row("Is through a wrap", errors.Is(wrappedKind, matchError{"any"}))

	// As with a pointer to an interface type stands in
	// conf-blocked/conferrorsasiface. Read REPORT-conform.md, item 12.
	// AsType of Go 1.27 needs no pointer, thus it reads here.
	got, ok := errors.AsType[*codeError](chain)
	row("AsType", fmt.Sprint(got.Code, " ", ok))
	_, ok = errors.AsType[valueError](chain)
	row("AsType of a wrong type", ok)

	// Is and As of nil.
	row("As of nil", errors.As(error(nil), &target))
	row("Is of a nil chain", errors.Is(error(nil), errBase))
}

func joins() {
	a := errors.New("a")
	b := errors.New("b")
	j := errors.Join(a, b)
	row("the text", fmt.Sprintf("%q", j.Error()))
	row("Is of a", errors.Is(j, a))
	row("Is of b", errors.Is(j, b))
	row("Is of another", errors.Is(j, errBase))
	row("the type", fmt.Sprintf("%T", j))
	row("Unwrap of a Join", errors.Unwrap(j))

	// A Join of nil values gives nil.
	row("Join of nothing", errors.Join() == nil)
	row("Join of nil", errors.Join(nil, nil) == nil)
	row("Join of 1 and nil", errors.Join(a, nil).Error())
	row("Join of nil and 1", errors.Join(nil, b).Error())

	// A Join inside a Join.
	inner := errors.Join(a, b)
	outer := errors.Join(inner, errBase)
	row("a nested Join", fmt.Sprintf("%q", outer.Error()))
	row("Is of the inner", errors.Is(outer, a))
	row("Is of the base", errors.Is(outer, errBase))

	// A Join with a custom type, and As.
	withCode := errors.Join(a, &codeError{7, "seven"})
	var target *codeError
	row("As of a Join", errors.As(withCode, &target))
	row("the code", target.Code)

	// A Join inside a wrap.
	wrapped := fmt.Errorf("outer: %w", j)
	row("a wrap of a Join", fmt.Sprintf("%q", wrapped.Error()))
	row("Is through the wrap", errors.Is(wrapped, b))
}

// nilInIface proves that a nil pointer inside an error interface is not nil.
type nilError struct{}

func (n *nilError) Error() string {
	if n == nil {
		return "nilError(nil)"
	}
	return "nilError"
}

func returnsNilPointer() error {
	var p *nilError
	return p
}

func returnsNil() error { return nil }

func nilInIface() {
	e := returnsNilPointer()
	row("a typed nil is not nil", e == nil)
	row("the text", e.Error())
	row("a plain nil is nil", returnsNil() == nil)
	row("Is of a typed nil", errors.Is(e, nil))
	var p *nilError
	row("compares to the pointer", e == error(p))
	// The safe test reads the pointer.
	row("the pointer is nil", p == nil)
	// A wrap of a typed nil.
	w := fmt.Errorf("outer: %w", e)
	row("a wrap of a typed nil", w.Error())
	row("Unwrap of it", errors.Unwrap(w) == nil)
	// A nil error in a slice and in a map.
	list := []error{nil, errBase}
	row("nil in a slice", list[0] == nil)
	row("printed", list)
}

// stdErrors proves a sentinel of the standard library.
func stdErrors() {
	_, err := os.Open("/a/path/that/holds/no/file")
	row("Is of ErrNotExist", errors.Is(err, os.ErrNotExist))
	var pathErr *os.PathError
	row("As of a PathError", errors.As(err, &pathErr))
	row("the operation", pathErr.Op)
	_, err = strconv.Atoi("xyz")
	row("the text of Atoi", err.Error())
	row("Is of ErrSyntax", errors.Is(err, strconv.ErrSyntax))
	var numErr *strconv.NumError
	row("As of a NumError", errors.As(err, &numErr))
	row("the function", numErr.Func)
	_, err = strconv.Atoi("99999999999999999999")
	row("Is of ErrRange", errors.Is(err, strconv.ErrRange))
	row("the text of a range", err.Error())
}

func main() {
	fmt.Println("== errors.New ==")
	newAndCompare()
	fmt.Println("== a wrap ==")
	wrapping()
	fmt.Println("== a custom type ==")
	customTypes()
	fmt.Println("== errors.Join ==")
	joins()
	fmt.Println("== a nil error ==")
	nilInIface()
	fmt.Println("== an error of the library ==")
	stdErrors()
}
