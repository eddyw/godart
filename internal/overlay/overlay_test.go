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
