package driver_test

import (
	"path/filepath"
	"testing"

	"github.com/eddyw/godart/internal/driver"
)

func TestFunctionValueLifetime(t *testing.T) {
	dir, out := t.TempDir(), t.TempDir()
	writeFile(t, filepath.Join(dir, "go.mod"), "module example.com/functionlifetime\n\ngo 1.27\n")
	writeFile(t, filepath.Join(dir, "main.go"), `package main
import ("fmt"; "reflect"; "runtime"; "time"; "weak")
type item struct { value int; data [256]byte }
//go:noinline
func closureCapture() (func() int, weak.Pointer[item]) {
 p := &item{value: 7}
 return func() int { return p.value }, weak.Make(p)
}
func (p *item) Value() int { return p.value }
func methodCapture() (func() int, weak.Pointer[item]) {
 p := &item{value: 7}
 f := reflect.ValueOf(p).MethodByName("Value").Interface().(func() int)
 return f, weak.Make(p)
}
type genericItem[T ~int] struct { value T; data [256]byte }
func genericCapture[T ~int](value T) (func() int, weak.Pointer[genericItem[T]]) {
 p := &genericItem[T]{value: value}
 return func() int { return int(p.value) }, weak.Make(p)
}
func genericBind() (func() int, weak.Pointer[genericItem[int]]) { return genericCapture(7) }
type sliceItem[S ~[]E, E ~int] struct { value S }
func sliceCapture[S ~[]E, E ~int](value S) (func() int, weak.Pointer[sliceItem[S, E]]) {
 p := &sliceItem[S, E]{value: value}
 return func() int { return int(p.value[0]) }, weak.Make(p)
}
func sliceBind() (func() int, weak.Pointer[sliceItem[[]int, int]]) { return sliceCapture([]int{7}) }
func aliasCapture[T ~int](value T) (func() int, weak.Pointer[struct{ a T; b T; data [256]byte }]) {
 type Alias = T
 p := &struct{ a Alias; b T; data [256]byte }{a: value, b: value}
 return func() int { return int(p.a) }, weak.Make(p)
}
func aliasBind() (func() int, weak.Pointer[struct{ a int; b int; data [256]byte }]) { return aliasCapture(7) }
func useCapture[P any](bind func() (func() int, weak.Pointer[P])) weak.Pointer[P] {
 f, w := bind()
 runtime.GC()
 if w.Value() == nil || f() != 7 { panic("lost live capture") }
 pc := reflect.ValueOf(f).Pointer()
 fn := runtime.FuncForPC(pc)
 if fn == nil || fn.Name() == "" { panic("missing closure metadata") }
 runtime.KeepAlive(f)
 return w
}
func exercise[P any](bind func() (func() int, weak.Pointer[P])) {
 w := useCapture(bind)
 deadline := time.Now().Add(4*time.Second)
 for w.Value() != nil {
  if time.Now().After(deadline) { panic("metadata retained capture") }
  runtime.GC()
  time.Sleep(10*time.Millisecond)
 }
 fmt.Println("released function capture")
}
func main() {
 exercise(closureCapture)
 exercise(methodCapture)
 exercise(genericBind)
 exercise(sliceBind)
 exercise(aliasBind)
}
`)
	want := runProgram(t, dir, "go", "run", ".")
	if err := driver.Build(driver.Options{Dir: dir, Patterns: []string{"."}, Out: out, Runtime: runtimeDir(t), TrimPath: true}); err != nil {
		t.Fatal(err)
	}
	prepareDart(t, out)
	if got := runProgram(t, dir, "dart", filepath.Join(out, "bin/main.dart")); got != want {
		t.Fatalf("Go:\n%s\nGart:\n%s", want, got)
	}
	for _, target := range []string{"aot", "js", "wasm"} {
		t.Run(target, func(t *testing.T) {
			got := runLifetimeTarget(t, target, out, "bin/main.dart")
			if got.Exit != 0 || got.Text != want {
				t.Fatalf("%s: exit %d\nGo:\n%s\nGart:\n%s", target, got.Exit, want, got.Text)
			}
		})
	}
}
