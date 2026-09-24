package main

import (
	"fmt"
	"interfacemethodpc/lib"
	"path/filepath"
	"reflect"
	"runtime"
)

type Reader interface{ Read(int) int }
type Extended interface{ Reader }
type Generic[T any] interface{ Echo(T) T }
type Private interface{ hidden() int }
type Box int

func (b Box) hidden() int    { return int(b) }
func (b Box) Read(n int) int { return int(b) + n }
func show(fn any) {
	pc := reflect.ValueOf(fn).Pointer()
	f := runtime.FuncForPC(pc)
	if f == nil {
		fmt.Println("missing")
		return
	}
	file, line := f.FileLine(pc)
	fmt.Println(f.Name(), filepath.Base(file), line)
}
func main() {
	show(Reader.Read)
	show(Extended.Read)
	show((interface{ Read(int) int }).Read)
	show(error.Error)
	show(Generic[int].Echo)
	show(Generic[string].Echo)
	show(Private.hidden)
	show((interface{ hidden() int }).hidden)
	var h interface{ hidden() int } = Box(1)
	show(h.hidden)
	for _, value := range lib.Values() {
		show(value)
	}
	fmt.Println(reflect.ValueOf(Generic[int].Echo).Pointer() != reflect.ValueOf(Generic[string].Echo).Pointer())
	fmt.Println(Reader.Read(Box(2), 3), Extended.Read(Box(3), 4))
}
