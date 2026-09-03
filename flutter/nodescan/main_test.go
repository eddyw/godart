package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestScanReadsEachName proves the 3 answers that a regular expression over
// the source cannot give: a name that an import alias carries, a name of a
// comment or of a string, and a name that only a second package of the module
// writes.
func TestScanReadsEachName(t *testing.T) {
	dir := t.TempDir()
	write := func(name, body string) {
		t.Helper()
		path := filepath.Join(dir, name)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	write("go.mod", "module probe.dev/app\n\ngo 1.27.0\n")
	write("ui/ui.go", `package ui

type Node interface{ isNode() }

type Text struct{ Value string }
type Card struct{ Child Node }
type Missed struct{ Child Node }
type Unused struct{ Child Node }

func (Text) isNode()   {}
func (Card) isNode()   {}
func (Missed) isNode() {}
func (Unused) isNode() {}
`)
	write("main.go", `package app

import (
	gui "probe.dev/app/ui"
	"probe.dev/app/inner"
)

// The comment names ui.Unused, and no code names that type.
const label = "ui.Unused"

// Build gives the tree of the application.
func Build() gui.Node {
	_ = label
	return gui.Card{Child: gui.Text{Value: "hello"}}
}

var _ = inner.Deep
`)
	write("inner/inner.go", `package inner

import "probe.dev/app/ui"

// Deep builds a node of a second package of the module.
var Deep = ui.Missed{}
`)

	got, err := scan(dir, "probe.dev/app/ui")
	if err != nil {
		t.Fatal(err)
	}
	want := "Card Missed Node Text"
	if strings.Join(got, " ") != want {
		t.Errorf("scan gave %q, and it must give %q", strings.Join(got, " "), want)
	}
}

// TestScanReportsABadPackage proves that the tool fails loudly. A silent empty
// answer would give a host with no case, and each node would then fail at run
// time.
func TestScanReportsABadPackage(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "go.mod"),
		[]byte("module probe.dev/bad\n\ngo 1.27.0\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "bad.go"),
		[]byte("package bad\n\nfunc F() { return 1 }\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := scan(dir, "probe.dev/bad/ui"); err == nil {
		t.Error("scan gave no error for a package that does not compile")
	}
}
