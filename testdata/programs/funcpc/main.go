package main

import (
	"example.com/funcpc/helper"
	"fmt"
	"path/filepath"
	"reflect"
	"runtime"
)

func named(n int) int { return n + 1 }

func main() {
	runtime.Caller(0)
	for _, bad := range []uintptr{0, ^uintptr(0), ^uintptr(0) >> 1, uintptr(1) << 63} {
		f := runtime.FuncForPC(bad)
		fmt.Printf("%t %q\n", f == nil, f.Name())
	}
}

func init() {
	for _, value := range []any{named, helper.Named} {
		pc := reflect.ValueOf(value).Pointer()
		fn := runtime.FuncForPC(pc)
		if fn == nil {
			panic("missing function metadata")
		}
		file, line := fn.FileLine(pc)
		fmt.Println(fn.Name(), filepath.Base(file), line > 0, fn.Entry() == pc, fn == runtime.FuncForPC(pc))
	}
}
