package driver_test

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/eddyw/godart/internal/driver"
)

// //gart:dart names 1 Dart symbol for a Go function declaration with no body.
// The value boundary of such a call holds no descriptor, thus stage 2 rejects
// each type that needs a conversion table.

// buildDart compiles a module of 1 file and gives the report of the build.
func buildDart(t *testing.T, main string) error {
	t.Helper()
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "go.mod"), "module darttest\n\ngo 1.27\n")
	writeFile(t, filepath.Join(dir, "main.go"), main)
	return driver.Build(driver.Options{
		Dir:      dir,
		Patterns: []string{"./..."},
		Out:      t.TempDir(),
		Runtime:  runtimeDir(t),
	})
}

func TestDartDeclarationAccepts(t *testing.T) {
	err := buildDart(t, `package main

import "fmt"

//gart:dart dart:math max
func dartMax(a, b float64) float64

//gart:dart dart:core print
func dartPrint(s string)

func main() {
	fmt.Println(dartMax(1, 2))
	dartPrint("hello")
}
`)
	if err != nil {
		t.Fatalf("the build rejected a correct declaration:\n%v", err)
	}
}

func TestDartDeclarationRejectsASlice(t *testing.T) {
	err := buildDart(t, `package main

//gart:dart dart:convert base64Encode
func encode(b []byte) string

func main() { _ = encode(nil) }
`)
	mustReject(t, err, "carries no form of it")
}

func TestDartDeclarationRejectsTwoResults(t *testing.T) {
	err := buildDart(t, `package main

//gart:dart dart:core int.parse
func parse(s string) (int, error)

func main() { _, _ = parse("1") }
`)
	mustReject(t, err, "and 2 results")
}

func TestDartDeclarationRejectsAMethod(t *testing.T) {
	err := buildDart(t, `package main

type box struct{}

//gart:dart dart:math max
func (box) big(a, b float64) float64

func main() { _ = box{}.big(1, 2) }
`)
	mustReject(t, err, "and only a function carries it")
}

func TestDartDeclarationRejectsAVariadic(t *testing.T) {
	err := buildDart(t, `package main

//gart:dart dart:math max
func big(v ...float64) float64

func main() { _ = big(1, 2) }
`)
	mustReject(t, err, "and a variadic parameter")
}

// A declaration with no marker still reaches the rejection of stage 2, thus
// the marker widens the subset and it does not open it.
func TestBodylessWithNoMarkerStaysRejected(t *testing.T) {
	err := buildDart(t, `package main

func hidden(a float64) float64

func main() { _ = hidden(1) }
`)
	mustReject(t, err, "holds no body")
}

func mustReject(t *testing.T, err error, want string) {
	t.Helper()
	if err == nil {
		t.Fatalf("the build accepted the declaration, and it holds %q", want)
	}
	if !strings.Contains(err.Error(), want) {
		t.Errorf("the report holds no %q:\n%v", want, err)
	}
}

func TestDartErrorSurvivesDeferredWait(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "go.mod"), "module darterror\n\ngo 1.27\n")
	writeFile(t, filepath.Join(dir, "main.go"), `package main
import ("fmt"; "runtime")
//gart:dart dart:core int.parse
func parse(s string) int
func main() {
 defer func() { runtime.Gosched(); fmt.Println("defer", recover()) }()
 fmt.Println(parse("invalid"))
}
`)
	out := t.TempDir()
	if err := driver.Build(driver.Options{Dir: dir, Patterns: []string{"./..."}, Out: out, Runtime: runtimeDir(t)}); err != nil {
		t.Fatal(err)
	}
	prepareDart(t, out)
	result := programOutcome(t, out, "dart", "bin/main.dart")
	if result.Exit == 0 || !strings.Contains(result.Text, "FormatException") || strings.Count(result.Text, "defer <nil>") != 1 {
		t.Fatalf("the deferred wait lost the Dart error or its finalizer: %+v", result)
	}
}
