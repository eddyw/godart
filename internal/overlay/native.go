package overlay

// native lists each standard library package that the compiler reads from the
// real Go source. Such a package needs no overlay, because it already stays
// inside the subset.
//
// A package enters this list when stage 2 accepts each file of it, and when
// each package that it imports also stays in the surface. A directory of the
// same path inside std then patches 1 file of the package by its base name.
var native = []string{
	"cmp",
	"container/list",
	"io",
	"path",
	"math",
	"math/bits",
	"unicode",
}

// NativePaths gives the standard library packages that the compiler reads from
// the real Go source.
func NativePaths() []string {
	out := make([]string, len(native))
	copy(out, native)
	return out
}

// Surface gives each standard library package that the compiler writes Dart
// code for. It joins the overlays and the native packages.
func Surface() []string {
	return append(Paths(), native...)
}
