//go:build dart

package reflect

// The 2 header types of the package reflect.
//
// Go marks each 1 as older than unsafe.Slice and unsafe.String, and it writes
// that no program can use them safely. Each field holds a plain number, thus
// the Dart form carries the same value that Go carries, and a read of a field
// gives the answer of Go.
//
// The subset rejects an import of the package unsafe. Thus no program of gart
// converts a slice or a string into a header, and no header of gart names the
// memory of a value.

// A SliceHeader is the shape of a slice value at run time.
type SliceHeader struct {
	Data uintptr
	Len  int
	Cap  int
}

// A StringHeader is the shape of a string value at run time.
type StringHeader struct {
	Data uintptr
	Len  int
}
