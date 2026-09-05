package main

import (
	"strings"
	"testing"

	"github.com/eddyw/godart/internal/overlay"
)

// TestSurfaceCount reports the count of the import paths of the surface, by
// group. docs/support.md holds the numbers, thus a change of the surface reads
// this test.
func TestSurfaceCount(t *testing.T) {
	seen := map[string]bool{}
	for _, p := range overlay.Surface() {
		seen[p] = true
	}
	std, internal, vendored, interop := 0, 0, 0, 0
	for p := range seen {
		switch {
		case p == overlay.IntrinsicPath:
			interop++
		case strings.HasPrefix(p, "vendor/"):
			vendored++
		case p == "internal" || strings.HasPrefix(p, "internal/") ||
			strings.Contains(p, "/internal/") || strings.HasSuffix(p, "/internal"):
			internal++
		default:
			std++
		}
	}
	got := [5]int{len(seen), std, interop, internal, vendored}
	want := [5]int{254, 164, 1, 77, 12}
	if got != want {
		t.Errorf("the surface holds %v, and docs/support.md names %v.\n"+
			"The order is the whole count, the standard packages, the interop "+
			"package, the internal packages, and the vendored packages.\n"+
			"Write the new numbers into the section \"The measure\" of "+
			"docs/support.md and into the table of docs/parity.md.", got, want)
	}
}
