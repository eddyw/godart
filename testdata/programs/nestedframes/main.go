package main

import (
	"fmt"
	"runtime"
)

func main() {
	fmt.Println(func() bool { drop(func() bool { runtime.Gosched(); return true }); return true }())
}
