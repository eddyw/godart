package main

import (
	"strings"
	"testing"

	"github.com/eddyw/godart/internal/overlay"
)

// TestSurfaceCount detects an unreviewed change to package classifications.
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
	want := [5]int{264, 165, 1, 86, 12}
	if got != want {
		t.Errorf("the surface holds %v, expected %v.\n"+
			"The order is the whole count, the standard packages, the interop "+
			"package, the internal packages, and the vendored packages.\n"+
			"Review the package surface before an expectation update.", got, want)
	}
}
