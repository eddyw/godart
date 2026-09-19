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

	got, texts, err := scan(dir, "probe.dev/app/ui")
	if err != nil {
		t.Fatal(err)
	}
	want := "Card Missed Node Text"
	if strings.Join(got, " ") != want {
		t.Errorf("scan gave %q, and it must give %q", strings.Join(got, " "), want)
	}
	// The scan keeps each string constant that can name an icon. "ui.Unused"
	// holds a period, thus no icon carries that name.
	if strings.Join(texts, " ") != "hello" {
		t.Errorf("scan gave the strings %q, and it must give %q",
			strings.Join(texts, " "), "hello")
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
	if _, _, err := scan(dir, "probe.dev/bad/ui"); err == nil {
		t.Error("scan gave no error for a package that does not compile")
	}
}

func TestScanKeepsApplicationLibraryIdentity(t *testing.T) {
	dir := t.TempDir()
	files := map[string]string{
		"go.mod":        "module probe.dev/app\n\ngo 1.27.0\n",
		"ui/ui.go":      `package ui; type Mark struct{}; type Node interface{ IsNode() Mark }`,
		"one/widget.go": `package one; import "probe.dev/app/ui"; type Badge struct{}; func (Badge) IsNode() ui.Mark { return ui.Mark{} }`,
		"two/widget.go": `package two; import "probe.dev/app/ui"; type Badge struct{}; func (Badge) IsNode() ui.Mark { return ui.Mark{} }`,
		"main.go":       `package app; import ("probe.dev/app/one"; "probe.dev/app/two"); var A = one.Badge{}; var B = two.Badge{}`,
	}
	for name, body := range files {
		path := filepath.Join(dir, name)
		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(body), 0644); err != nil {
			t.Fatal(err)
		}
	}
	names, _, err := scan(dir, "probe.dev/app/ui")
	if err != nil {
		t.Fatal(err)
	}
	got := strings.Join(names, " ")
	for _, name := range []string{"probe.dev/app/one.Badge", "probe.dev/app/two.Badge"} {
		if !strings.Contains(got, name) {
			t.Fatalf("missing %s in %s", name, got)
		}
	}
}
