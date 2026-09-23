package driver_test

import (
	"path/filepath"
	"testing"

	"github.com/eddyw/godart/internal/driver"
)

func TestWeakPointerLifetime(t *testing.T) {
	dir, out := t.TempDir(), t.TempDir()
	writeFile(t, filepath.Join(dir, "go.mod"), "module example.com/weaklocations\n\ngo 1.27\n")
	writeFile(t, filepath.Join(dir, "main.go"), `package main
import ("fmt"; "reflect"; "runtime"; "time"; "weak")
type item struct { A int; B int; data [256]byte }
var Global int
func field() weak.Pointer[int] {
 p := &item{A: 7, B: 9}
 w := weak.Make(&p.A)
 if w != weak.Make(&p.A) { panic("field identity") }
 if w == weak.Make(&p.B) { panic("distinct field identity") }
 if w != weak.Make(reflect.ValueOf(p).Elem().Field(0).Addr().Interface().(*int)) { panic("reflected field identity") }
 runtime.GC()
 if w.Value() == nil || *w.Value() != 7 { panic("lost live field") }
 *w.Value() = 11
 if p.A != 11 { panic("field write") }
 runtime.KeepAlive(p)
 return w
}
type view struct { A int; B int; data [256]byte }
func convertedField() weak.Pointer[int] {
 p := &item{A: 7, B: 9}
 v := (*view)(p)
 w := weak.Make(&v.A)
 if w != weak.Make(&p.A) { panic("converted field identity") }
 runtime.GC()
 if w.Value() == nil || *w.Value() != 7 { panic("lost converted field") }
 *w.Value() = 11
 if p.A != 11 || v.A != 11 { panic("converted field write") }
 runtime.KeepAlive(p)
 return w
}
func element() weak.Pointer[int] {
 p := make([]int, 32)
 p[4] = 7
 w := weak.Make(&p[4])
 if w != weak.Make(&p[4]) { panic("element identity") }
 if w == weak.Make(&p[5]) { panic("distinct element identity") }
 a := (*[8]int)(p[2:])
 if w != weak.Make(&a[2]) { panic("array view identity") }
 runtime.GC()
 if w.Value() == nil || *w.Value() != 7 { panic("lost live element") }
 *w.Value() = 11
 if p[4] != 11 { panic("element write") }
 runtime.KeepAlive(p)
 return w
}
func convertedPointer() weak.Pointer[view] {
 p := &item{A: 7, B: 9}
 v := (*view)(p)
 w := weak.Make(v)
 if w != weak.Make((*view)(p)) { panic("converted pointer identity") }
 runtime.GC()
 if w.Value() == nil || w.Value().A != 7 { panic("lost converted pointer") }
 w.Value().A = 11
 if p.A != 11 { panic("converted pointer write") }
 runtime.KeepAlive(p)
 return w
}
func convertedArray() weak.Pointer[[8]int] {
 p := make([]int, 32)
 p[2] = 7
 a := (*[8]int)(p[2:])
 w := weak.Make(a)
 if w != weak.Make((*[8]int)(p[2:])) { panic("converted array identity") }
 runtime.GC()
 if w.Value() == nil || w.Value()[0] != 7 { panic("lost converted array") }
 w.Value()[0] = 11
 if p[2] != 11 { panic("converted array write") }
 runtime.KeepAlive(p)
 return w
}
func exercise[T any](makePointer func() weak.Pointer[T]) {
 w := makePointer()
 deadline := time.Now().Add(4*time.Second)
 for w.Value() != nil {
  if time.Now().After(deadline) { panic("weak pointer retained owner") }
  runtime.GC()
  time.Sleep(10*time.Millisecond)
 }
 if w == (weak.Pointer[T]{}) { panic("lost dead identity") }
 fmt.Println("released weak location")
}
func main() {
 w := weak.Make(&Global)
 if w != weak.Make(&Global) { panic("global identity") }
 runtime.GC()
 if w.Value() != &Global { panic("lost global") }
 exercise(field)
 exercise(convertedField)
 exercise(convertedPointer)
 exercise(convertedArray)
 exercise(element)
 if w != weak.Make(&Global) || w.Value() != &Global { panic("lost global identity") }
}
`)
	want := runProgram(t, dir, "go", "run", ".")
	if err := driver.Build(driver.Options{Dir: dir, Patterns: []string{"."}, Out: out, Runtime: runtimeDir(t), TrimPath: true}); err != nil {
		t.Fatal(err)
	}
	prepareDart(t, out)
	for _, target := range []string{"vm", "aot", "js", "wasm"} {
		t.Run(target, func(t *testing.T) {
			got := runLifetimeTarget(t, target, out, "bin/main.dart")
			if got.Exit != 0 || got.Text != want {
				t.Fatalf("%s: exit %d\nGo:\n%s\nGart:\n%s", target, got.Exit, want, got.Text)
			}
		})
	}
}
