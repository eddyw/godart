package driver_test

import (
	"github.com/eddyw/godart/internal/driver"
	"path/filepath"
	"testing"
)

func TestReflectedOpaquePointerKinds(t *testing.T) {
	dir, out := t.TempDir(), t.TempDir()
	writeFile(t, filepath.Join(dir, "go.mod"), "module example.com/reflectedpointer\n\ngo 1.27\n")
	writeFile(t, filepath.Join(dir, "main.go"), `package main
import("fmt";"reflect";"runtime";"unsafe")
type S struct { First int; Pad byte; Values [3]int; Last int }
type Scalar struct {First,Middle,Last int}
func allocate[T any]() *T {return new(T)}
func pinPanics(value any)(out bool){defer func(){out=recover()!=nil}();var p runtime.Pinner;p.Pin(value);p.Unpin();return}
func invalid(v reflect.Value)(out string){defer func(){out=fmt.Sprintf("%T",recover())}();v.UnsafePointer();return}
func query(f func()bool)(ok bool){defer func(){if recover()!=nil{ok=false}}();return f()}
func opaque(v any)unsafe.Pointer{return reflect.ValueOf(v).UnsafePointer()}
func function(){}
func invalidPointer(v reflect.Value)(out string){defer func(){out=fmt.Sprintf("%T",recover())}();v.Pointer();return}
func main(){
 namedScalar:=new(Scalar);namedFirst:=reflect.ValueOf(&namedScalar.First).Pointer();namedOther:=new(int);*namedOther=7;namedOtherNumber:=reflect.ValueOf(namedOther).Pointer();namedLast:=reflect.ValueOf(&namedScalar.Last).Pointer()
 fmt.Println("named allocation span",namedOtherNumber<namedFirst || namedOtherNumber>namedLast,*namedOther==7)
 genericNamed:=allocate[Scalar]();genericNamedFirst:=reflect.ValueOf(&genericNamed.First).Pointer();genericNamedOther:=new(int);*genericNamedOther=13;genericNamedOtherNumber:=reflect.ValueOf(genericNamedOther).Pointer();genericNamedLast:=reflect.ValueOf(&genericNamed.Last).Pointer()
 fmt.Println("generic named span",genericNamedOtherNumber<genericNamedFirst || genericNamedOtherNumber>genericNamedLast,*genericNamedOther==13)
 genericArray:=allocate[[3]int]();genericFirst:=reflect.ValueOf(&genericArray[0]).Pointer();genericOther:=new(int);*genericOther=11;genericOtherNumber:=reflect.ValueOf(genericOther).Pointer();genericLast:=reflect.ValueOf(&genericArray[2]).Pointer()
 fmt.Println("generic array span",genericOtherNumber<genericFirst || genericOtherNumber>genericLast,*genericOther==11)
 scalar:=new(struct{First,Middle,Last int});scalarFirst:=reflect.ValueOf(&scalar.First).Pointer();otherScalar:=new(int);*otherScalar=5;otherNumber:=reflect.ValueOf(otherScalar).Pointer();scalarLast:=reflect.ValueOf(&scalar.Last).Pointer()
 fmt.Println("scalar allocation span",otherNumber<scalarFirst || otherNumber>scalarLast,*otherScalar==5)
 shape:=reflect.StructOf([]reflect.StructField{{Name:"First",Type:reflect.TypeOf(0)},{Name:"Padding",Type:reflect.ArrayOf(128,reflect.TypeOf(byte(0)))},{Name:"Last",Type:reflect.TypeOf(0)}})
 dynamic:=reflect.New(shape).Elem();dynamicFirst:=dynamic.Field(0).Addr();dynamicLast:=dynamic.Field(2).Addr()
 fmt.Println("dynamic offsets",dynamicLast.Pointer()-dynamicFirst.Pointer()==shape.Field(2).Offset,dynamicFirst.UnsafePointer()==dynamic.Addr().UnsafePointer())
 paddingPointer:=dynamic.Field(1).Index(0).Addr().UnsafePointer();replacement:=[128]byte{71};dynamic.Field(1).Set(reflect.ValueOf(replacement))
 fmt.Println("dynamic assignment",dynamic.Field(1).Index(0).Uint()==71,paddingPointer==dynamic.Field(1).Index(0).Addr().UnsafePointer())
 data:=make([]byte,65536);firstData:=reflect.ValueOf(&data[0]).Pointer();lastData:=reflect.ValueOf(&data[len(data)-1]).Pointer();code:=reflect.ValueOf(function).Pointer()
 fmt.Println("code outside data",code<firstData || code>lastData)
 fmt.Println("invalid numeric kind",invalidPointer(reflect.ValueOf(3)),invalidPointer(reflect.Value{}))
 var nilfn func();var nilmap map[int]int;var nilslice []int
 fmt.Println("function values",query(func()bool{return opaque(nilfn)==nil && opaque(function)!=nil && opaque(function)==opaque(function)}))
 maps:=map[int]int{1:2};fmt.Println("map values",query(func()bool{return opaque(nilmap)==nil && opaque(maps)!=nil && opaque(maps)==opaque(maps)}))
 slice:=make([]int,3,4);empty:=make([]int,0)
 fmt.Println("slice values",query(func()bool{return opaque(nilslice)==nil && opaque(empty)!=nil && opaque(slice)==unsafe.Pointer(&slice[0]) && opaque(slice[1:])==unsafe.Pointer(&slice[1]) && reflect.ValueOf(slice).Pointer()==reflect.ValueOf(&slice[0]).Pointer()}))
 text:=string([]byte{'a','b','c','d'})
 fmt.Println("string values",query(func()bool{return opaque(text)!=nil && opaque(text)==opaque(text[:2]) && opaque(text[1:])==opaque(text[1:3]) && reflect.ValueOf(text[1:]).Pointer()-reflect.ValueOf(text).Pointer()==1}))
 fmt.Println("invalid kind",invalid(reflect.ValueOf(3)),invalid(reflect.Value{}))
 s:=new(S);values:=reflect.ValueOf(&s.Values);first:=reflect.ValueOf(&s.Values[0]);last:=reflect.ValueOf(&s.Values[2]);root:=reflect.ValueOf(s)
 fmt.Println("pointer alignment",root.Pointer()%uintptr(unsafe.Alignof(*s))==0,first.Pointer()%uintptr(unsafe.Alignof(s.Values[0]))==0)
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
