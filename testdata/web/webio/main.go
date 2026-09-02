package main

import (
	"fmt"
	"net/http"
	"os"
)

// The web target holds no file and no socket. This program reads each shape
// of that boundary, thus the run of a web build proves that the whole io
// surface compiles and that each operation reports an honest cause.
func main() {
	fmt.Println("goos:", os.Getenv("GART_WEBIO") == "")
	fmt.Println("pid:", os.Getpid())

	if _, err := os.Open("/etc/hosts"); err != nil {
		fmt.Println("open:", err)
	} else {
		fmt.Println("open: no error")
	}
	if err := os.Mkdir("/x", 0o755); err != nil {
		fmt.Println("mkdir:", err)
	} else {
		fmt.Println("mkdir: no error")
	}
	if _, err := http.Get("http://127.0.0.1:1/"); err != nil {
		fmt.Println("get: an error")
	} else {
		fmt.Println("get: no error")
	}
}
