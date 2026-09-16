package parity

import (
	"regexp"
	"testing"

	harness "github.com/eddyw/godart/internal/overlay/std/testing"
)

func TestSkipPreservesParents(t *testing.T) {
	pattern := "^TestA$/^excluded$|^TestB$"
	for name, want := range map[string]bool{
		"TestA": false, "TestA/kept": false, "TestA/excluded": true,
		"TestA/excluded/child": true, "TestB": true, "TestC": false,
	} {
		if got := harness.SkipName(name, pattern, regexp.MatchString); got != want {
			t.Errorf("%s: got %v, want %v", name, got, want)
		}
	}
}
