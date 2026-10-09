package driver_test

import (
	"path/filepath"
	"testing"

	"github.com/eddyw/godart/internal/driver"
)

func TestReflectOpaquePointerSet(t *testing.T) {
	dir, out := t.TempDir(), t.TempDir()
	writeFile(t, filepath.Join(dir, "go.mod"), "module example.com/reflectpointerset\n\ngo 1.27\n")
	writeFile(t, filepath.Join(dir, "main.go"), `package main
import("fmt";"reflect";"unsafe")
type NamedPointer unsafe.Pointer
type Fields struct { Pointer unsafe.Pointer; Named NamedPointer; private unsafe.Pointer }
type Allocation struct { First int; Last int }
func failure(v reflect.Value) (text string) {
 defer func(){ value:=recover();text=fmt.Sprintf("%T %v",value,value) }()
 v.SetPointer(nil)
 return "no panic"
}
func main(){
 allocation:=&Allocation{First:17,Last:29}
 first,last:=unsafe.Pointer(&allocation.First),unsafe.Pointer(&allocation.Last)
 var pointer unsafe.Pointer
 cell:=reflect.ValueOf(&pointer).Elem()
 cell.SetPointer(first)
 fmt.Println("cell",pointer==first,cell.UnsafePointer()==first,cell.Interface().(unsafe.Pointer)==first)
 cell.SetPointer(last)
 fmt.Println("offset",pointer==last,pointer!=first)
 cell.SetPointer(nil)
 fmt.Println("nil",pointer==nil,cell.IsNil())
 var named NamedPointer
 namedCell:=reflect.ValueOf(&named).Elem()
 namedCell.SetPointer(first)
 fmt.Println("named",named==NamedPointer(first),namedCell.Type()==reflect.TypeOf(named),namedCell.Interface().(NamedPointer)==NamedPointer(first))
 namedCell.SetPointer(nil)
 fmt.Println("named nil",named==nil,namedCell.IsNil())
 fields:=Fields{}
 field:=reflect.ValueOf(&fields).Elem()
 field.Field(0).SetPointer(first)
 field.Field(1).SetPointer(last)
 fmt.Println("fields",fields.Pointer==first,fields.Named==NamedPointer(last))
 pointers:=make([]unsafe.Pointer,2)
 reflect.ValueOf(pointers).Index(0).SetPointer(first)
 reflect.ValueOf(pointers).Index(1).SetPointer(last)
 fmt.Println("slice",pointers[0]==first,pointers[1]==last)
 var array [2]NamedPointer
 arrayValue:=reflect.ValueOf(&array).Elem()
 arrayValue.Index(0).SetPointer(first)
 arrayValue.Index(1).SetPointer(last)
 fmt.Println("array",array[0]==NamedPointer(first),array[1]==NamedPointer(last))
 fmt.Println("alias",fields.Pointer==unsafe.Pointer(allocation),fields.Named==NamedPointer(unsafe.Pointer(&allocation.Last)),allocation.First==17,allocation.Last==29)
 var boxed any=first
 for _,v:=range []reflect.Value{reflect.Value{},reflect.ValueOf(first),reflect.ValueOf(new(int)).Elem(),field.Field(2),reflect.ValueOf(&boxed).Elem(),reflect.ValueOf(map[int]unsafe.Pointer{0:first}).MapIndex(reflect.ValueOf(0))}{fmt.Println(failure(v))}
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
				t.Fatalf("Go: %+v\nGart: %+v", want, got)
			}
		})
	}
}
