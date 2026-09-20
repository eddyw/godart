package main

import (
	"fmt"
	"runtime"
)

func main() {
	runtime.Caller(0)
	for _, bad := range []uintptr{0, ^uintptr(0), ^uintptr(0) >> 1, uintptr(1) << 63} {
		f := runtime.FuncForPC(bad)
		fmt.Printf("%t %q\n", f == nil, f.Name())
	}
}
