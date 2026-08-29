package overlay

import (
	"strings"
	"sync"
)

// native lists each standard library package that the compiler reads from the
// real Go source. Such a package needs no overlay, because it already stays
// inside the subset.
//
// A package enters this list when stage 2 accepts each file of it, and when
// each package that it imports also stays in the surface. A directory of the
// same path inside std then patches 1 file of the package by its base name.
var native = []string{
	"bufio",
	"bytes",
	"cmp",
	"container/list",
	"encoding/base64",
	"internal/byteorder",
	"context",
	"io",
	"path",
	"math",
	"math/bits",
	"math/rand",
	"math/rand/v2",
	"internal/chacha8rand",
	"unicode",
	"container/heap",
	"container/ring",
	"encoding/ascii85",
	"encoding/csv",
	"encoding/hex",
	"hash",
	"hash/adler32",
	"hash/crc32",
	"hash/fnv",
	"net/netip",
	"net/url",
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

// Outside reports whether a path names a standard library package that the
// curated surface does not hold. The compiler writes no Dart code for such a
// package, and the verifier rejects an import of it.
//
// The first element of a module path always holds a dot, and a standard path
// never does. A path with the suffix _test names an external test package. The
// go command reserves that suffix, thus such a package belongs to the build.
func Outside(pkgPath string) bool {
	if strings.HasSuffix(pkgPath, "_test") {
		return false
	}
	first, _, _ := strings.Cut(pkgPath, "/")
	if strings.Contains(first, ".") {
		return false
	}
	return !surfaceSet()[pkgPath]
}

// surfaceSet holds each package of the surface. A walk of the embedded files
// for each call of Outside would cost too much.
var surfaceSet = sync.OnceValue(func() map[string]bool {
	out := map[string]bool{}
	for _, p := range Surface() {
		out[p] = true
	}
	return out
})
