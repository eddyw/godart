package driver_test

import (
	"github.com/eddyw/godart/internal/driver"
	"path/filepath"
	"testing"
)

func TestInlineSequencePointerCalls(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "go.mod"), "module example.com/inlinepointers\n\ngo 1.27\n")
	writeFile(t, filepath.Join(dir, "main.go"), `package main

import "fmt"

type row struct {
	N int
	Data [2]int
}

var order []string

func sequence(v []row) []row {
	order = append(order, "sequence")
	return v
}

func index() int {
	order = append(order, "index")
	ch := make(chan int)
	go func() { ch <- 1 }()
	return <-ch
}

func generic[T ~[2]row | ~[]row](v T) *row {
	return &v[1]
}

func report(name string, call func()) {
	defer func() { fmt.Println(name, recover()) }()
	call()
}

func main() {
	s := []row{{N: 3}, {N: 5}, {N: 7}}
	p := &sequence(s)[index()]
	fmt.Println("order", order)
	fmt.Println("slice identity", p == &s[1], p != &s[2])
	p.N = 11
	fmt.Println("slice write", s[1].N)
	a := (*[2]row)(s[1:])
	fmt.Println("view identity", p == &a[0], p == generic(s[:2]))
	a[0] = row{N: 13}
	fmt.Println("view assignment", p.N, s[1].N)
	b := [2]row{{N: 17}, {N: 19}}
	q := &b[1]
	fmt.Println("array identity", q == &b[1], q != &b[0])
	b = [2]row{{N: 23}, {N: 29}}
	fmt.Println("array assignment", q.N, generic(b).N)
	var nested [2][2]int
	n := &nested[1]
	fmt.Println("nested identity", n == &nested[1], n != &nested[0])
	*n = [2]int{31, 37}
	fmt.Println("nested write", nested[1])
	nested = [2][2]int{{41, 43}, {47, 53}}
	fmt.Println("nested assignment", *n)
	z := make([][0]int, 2)
	fmt.Println("zero element", &z[1] != nil, len(*(&z[1])))
	var negative = -1
	var beyond = 2
	var huge = uint64(1<<64 - 1)
	report("negative slice", func() { _ = &s[negative] })
	report("array bound", func() { _ = &b[beyond] })
	report("unsigned bound", func() { _ = &s[huge] })
	report("nil slice", func() { var v []row; _ = &v[0] })
	report("nil array", func() { var v *[2]row; _ = &v[0] })
}`)
	want := programOutcome(t, dir, "go", "run", ".")
	out := t.TempDir()
	if err := driver.Build(driver.Options{Dir: dir, Patterns: []string{"."}, Out: out, Runtime: runtimeDir(t), TrimPath: true}); err != nil {
		t.Fatal(err)
	}
	prepareDart(t, out)
	for _, target := range []string{"vm", "aot", "js", "wasm"} {
		t.Run(target, func(t *testing.T) {
			var got programResult
			if target == "vm" {
				got = programOutcome(t, dir, "dart", filepath.Join(out, "bin/main.dart"))
			} else {
				got = runUpstreamTarget(t, target, out, "bin/main.dart", dir, []string{})
			}
			if want.Exit != 0 || got.Exit != 0 || want.Text != got.Text {
				t.Fatalf("Go:\n%+v\nGart:\n%+v", want, got)
			}
		})
	}
}
