package overlay_test

import (
	"slices"
	"testing"

	"github.com/eddyw/godart/internal/overlay"
)

func TestPackages(t *testing.T) {
	paths := overlay.Paths()
	for _, want := range []string{"errors", "fmt", "syscall/dart"} {
		if !slices.Contains(paths, want) {
			t.Errorf("the overlay holds %v, want the package %s", paths, want)
		}
	}
	for _, p := range overlay.Packages() {
		if len(p.Files) == 0 {
			t.Errorf("the package %s holds no file", p.Path)
		}
		for _, f := range p.Files {
			if len(f.Content) == 0 {
				t.Errorf("the file %s of %s is empty", f.Name, p.Path)
			}
		}
	}
}

// TestStubs proves that a stub package stays out of the curated surface. The
// load replaces such a package, and the verifier still rejects an import of it.
func TestStubs(t *testing.T) {
	paths := overlay.Paths()
	stubs := map[string]bool{}
	for _, p := range overlay.Packages() {
		if p.Stub {
			stubs[p.Path] = true
		}
	}
	for _, want := range []string{"reflect", "testing/internal/testdeps"} {
		if !stubs[want] {
			t.Errorf("the overlay holds no stub of %s", want)
		}
		if slices.Contains(paths, want) {
			t.Errorf("the surface holds the stub %s", want)
		}
	}
	for _, want := range []string{"os", "testing"} {
		if !slices.Contains(paths, want) {
			t.Errorf("the surface holds %v, want the package %s", paths, want)
		}
	}
}
