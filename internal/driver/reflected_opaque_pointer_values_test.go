package driver_test

import (
	"github.com/eddyw/godart/internal/driver"
	"path/filepath"
	"testing"
)

func TestReflectedOpaquePointerValues(t *testing.T) {
	dir, out := t.TempDir(), t.TempDir()
	writeFile(t, filepath.Join(dir, "go.mod"), "module example.com/reflectedpointer\n\ngo 1.27\n")
	writeFile(t, filepath.Join(dir, "main.go"), `package main
import("fmt";"reflect";"runtime";"unsafe")
type S struct { First int; Pad byte; Values [3]int; Last int }
func pinPanics(value any)(out bool){defer func(){out=recover()!=nil}();var p runtime.Pinner;p.Pin(value);p.Unpin();return}
func invalid(v reflect.Value)(out string){defer func(){out=fmt.Sprintf("%T",recover())}();v.UnsafePointer();return}
func main(){
 fmt.Println("invalid kind",invalid(reflect.ValueOf(3)),invalid(reflect.Value{}))
 s:=new(S);values:=reflect.ValueOf(&s.Values);first:=reflect.ValueOf(&s.Values[0]);last:=reflect.ValueOf(&s.Values[2]);root:=reflect.ValueOf(s)
 fmt.Println("pointer type",reflect.TypeOf(first.UnsafePointer())==reflect.TypeOf(unsafe.Pointer(nil)))
 fmt.Println("allocation aliases",root.UnsafePointer()==reflect.ValueOf(&s.First).UnsafePointer(),values.UnsafePointer()==first.UnsafePointer(),first.UnsafePointer()==unsafe.Pointer(&s.Values[0]))
 fmt.Println("numeric offsets",last.Pointer()-first.Pointer()==16,first.Pointer()==values.Pointer(),root.Pointer()==reflect.ValueOf(&s.First).Pointer())
 fmt.Println("opaque number",first.Pointer()==reflect.ValueOf(first.UnsafePointer()).Pointer(),last.Pointer()==reflect.ValueOf(last.UnsafePointer()).Pointer())
 fmt.Println("pointer format",fmt.Sprintf("%p",&s.Values[0])==fmt.Sprintf("%p",first.UnsafePointer()))
 m:=map[unsafe.Pointer]int{first.UnsafePointer():17};s.Values=[3]int{19,23,29};fmt.Println("assignment alias",m[unsafe.Pointer(&s.Values[0])],first.UnsafePointer()==unsafe.Pointer(&s.Values[0]))
 var typed *int;var opaque unsafe.Pointer;var channel chan int
 fmt.Println("nil values",reflect.ValueOf(typed).UnsafePointer()==nil,reflect.ValueOf(opaque).UnsafePointer()==nil,reflect.ValueOf(channel).UnsafePointer()==nil)
 c:=make(chan int);other:=make(chan int);fmt.Println("channel aliases",reflect.ValueOf(c).UnsafePointer()==reflect.ValueOf(c).UnsafePointer(),reflect.ValueOf(c).UnsafePointer()!=reflect.ValueOf(other).UnsafePointer(),reflect.ValueOf(c).Pointer()==reflect.ValueOf(reflect.ValueOf(c).UnsafePointer()).Pointer())
 fmt.Println("pin values",pinPanics(&s.First),pinPanics(first.UnsafePointer()),pinPanics(reflect.ValueOf(c).UnsafePointer()),pinPanics(typed),pinPanics(opaque),pinPanics(nil),pinPanics(3))
}

`)
	want := programOutcome(t, dir, "go", "run", ".")
	if want.Exit != 0 {
		t.Fatalf("native Go failed: %+v", want)
	}
	if err := driver.Build(driver.Options{Dir: dir, Patterns: []string{"."}, Out: out, Runtime: runtimeDir(t)}); err != nil {
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
			if got != want {
				t.Fatalf("Go:\n%s\nGart (exit %d):\n%s", want.Text, got.Exit, got.Text)
			}
		})
	}
}
