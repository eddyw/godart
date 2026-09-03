//go:build dart

package runtime

import "syscall/dart"

// A Pinner holds a set of pinned objects.
//
// A pin of Go does 2 things: the collector moves the object no more, and it
// frees the object no more. The subset holds no cgo and no address of a
// machine, thus no program reads the position of an object and only the second
// half reaches it. The Pinner keeps a reference of each pinned object, and the
// collector of Dart frees no object that a reference names.
//
// Go ends the program when a Pinner reaches the collector with a pin that no
// Unpin dropped. A cleanup of Dart reads no field of the object that it
// watches, thus gart writes no such report. Read docs/parity.md.
type Pinner struct {
	pinned []any
}

// Pin keeps an object alive up to the call of Unpin.
//
// The argument must be a pointer. A nil argument and an argument of another
// kind each panic, exactly as they do in Go.
func (p *Pinner) Pin(pointer any) {
	if pointer == nil {
		panic(errorString("runtime.Pinner: argument is nil"))
	}
	if dart.Kind(pointer) != "ptr" {
		panic(errorString("runtime.Pinner: argument is not a pointer: " +
			dart.Format("T", pointer)))
	}
	p.pinned = append(p.pinned, pointer)
}

// Unpin drops each object of the Pinner. The Pinner takes a new pin after the
// call.
func (p *Pinner) Unpin() { p.pinned = nil }

// An errorString is a run time error that the runtime builds. Go holds the
// same type and the same text.
type errorString string

func (e errorString) RuntimeError() {}

func (e errorString) Error() string { return "runtime error: " + string(e) }
