// The witness of each method of a nil *os.File. Go tests the receiver first
// and it gives ErrInvalid.
package main

import (
	"fmt"
	"os"
	"time"
)

func main() {
	var f *os.File
	n, err := f.Write([]byte("x"))
	fmt.Println("write", n, err)
	n, err = f.Read(make([]byte, 1))
	fmt.Println("read", n, err)
	fmt.Println("close", f.Close())
	off, err := f.Seek(0, 0)
	fmt.Println("seek", off, err)
	fmt.Println("sync", f.Sync())
	_, err = f.Stat()
	fmt.Println("stat", err)
	n, err = f.WriteString("x")
	fmt.Println("writestring", n, err)
	fmt.Println("truncate", f.Truncate(0))
	n, err = f.WriteAt([]byte("x"), 0)
	fmt.Println("writeat", n, err)
	n, err = f.ReadAt(make([]byte, 1), 0)
	fmt.Println("readat", n, err)
	fmt.Println("chmod", f.Chmod(0o644))
	_, err = f.ReadDir(1)
	fmt.Println("readdir", err)
	_, err = f.Readdirnames(1)
	fmt.Println("readdirnames", err)
	fmt.Println("chdir", f.Chdir())
	fmt.Println("chown", f.Chown(-1, -1))
	fmt.Println("deadline", f.SetDeadline(time.Time{}))
	fmt.Println("read deadline", f.SetReadDeadline(time.Time{}))
	fmt.Println("write deadline", f.SetWriteDeadline(time.Time{}))
	fmt.Println("errors is", err == os.ErrInvalid)
}
