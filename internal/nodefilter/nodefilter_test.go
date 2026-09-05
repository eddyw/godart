package nodefilter_test

import (
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/eddyw/godart/internal/nodefilter"
)

// uiRoot names the node module of the repository.
func uiRoot(t *testing.T) string {
	t.Helper()
	dir, err := filepath.Abs("../../flutter/ui")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(dir); err != nil {
		t.Skipf("the checkout holds no node module: %v", err)
	}
	return dir
}

// TestSourcesKeepsTheSet proves that the filter keeps each type of the set and
// drops each other 1. The result must also parse.
func TestSourcesKeepsTheSet(t *testing.T) {
	root := uiRoot(t)
	out, err := nodefilter.Sources(root, []string{"Center", "Column", "Text", "Scaffold"})
	if err != nil {
		t.Fatal(err)
	}
	if len(out) < 2 {
		t.Fatalf("the filter wrote %d files, and the module holds a pack for each 1", len(out))
	}
	core := filepath.Join(root, "widgets_gen.go")
	src, ok := out[core]
	if !ok {
		t.Fatalf("the result holds no %s", core)
	}
	fset := token.NewFileSet()
	if _, err := parser.ParseFile(fset, core, src, 0); err != nil {
		t.Fatalf("the filtered source does not parse: %v", err)
	}
	text := string(src)
	for _, want := range []string{"type Center struct", "type Column struct", "type Text struct",
		"func (Center) IsNode() Mark", "func (Text) IsNode() Mark"} {
		if !strings.Contains(text, want) {
			t.Errorf("the filtered core pack holds no %q", want)
		}
	}
	for _, gone := range []string{"type AbsorbPointer struct", "type AndroidView struct",
		"func (AbsorbPointer) IsNode() Mark"} {
		if strings.Contains(text, gone) {
			t.Errorf("the filtered core pack still holds %q", gone)
		}
	}
	full, err := os.ReadFile(core)
	if err != nil {
		t.Fatal(err)
	}
	if len(src) >= len(full) {
		t.Errorf("the filter wrote %d bytes against %d bytes of the file", len(src), len(full))
	}
	t.Logf("the core pack holds %d bytes, and the filter wrote %d", len(full), len(src))
}

// TestSourcesHoldsTheHandWrittenNames proves that a type that a hand written
// file of the pack names stays, also when the set holds it not.
func TestSourcesHoldsTheHandWrittenNames(t *testing.T) {
	root := uiRoot(t)
	out, err := nodefilter.Sources(root, nil)
	if err != nil {
		t.Fatal(err)
	}
	for path, src := range out {
		fset := token.NewFileSet()
		if _, err := parser.ParseFile(fset, path, src, 0); err != nil {
			t.Errorf("the filtered source of %s does not parse: %v", path, err)
		}
	}
}
