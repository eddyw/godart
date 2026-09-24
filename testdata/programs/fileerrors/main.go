package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"syscall"
)

func main() {
	root, err := os.MkdirTemp("", "file-errors-")
	if err != nil {
		panic(err)
	}
	defer os.RemoveAll(root)
	seekErrors(root)
	path := filepath.Join(root, "file")
	if err := os.WriteFile(path, []byte("abc"), 0600); err != nil {
		panic(err)
	}
	for _, kind := range []string{"read-only", "write-only", "read-write", "directory", "closed", "nil", "append", "closed-append"} {
		for _, method := range []string{"Read", "Write", "ReadAt", "WriteAt"} {
			for _, size := range []int{0, 1} {
				offsets := []int64{0}
				if method == "ReadAt" || method == "WriteAt" {
					offsets = []int64{-1, 0}
				}
				for _, off := range offsets {
					target, flags := path, os.O_RDONLY
					switch kind {
					case "write-only":
						flags = os.O_WRONLY
					case "read-write":
						flags = os.O_RDWR
					case "directory":
						target = root
					case "append", "closed-append":
						flags = os.O_WRONLY | os.O_APPEND
					}
					var f *os.File
					if kind != "nil" {
						f, err = os.OpenFile(target, flags, 0)
						if err != nil {
							panic(err)
						}
					}
					if kind == "closed" || kind == "closed-append" {
						f.Close()
					}
					b := make([]byte, size)
					var n int
					switch method {
					case "Read":
						n, err = f.Read(b)
					case "Write":
						n, err = f.Write(b)
					case "ReadAt":
						n, err = f.ReadAt(b, off)
					case "WriteAt":
						n, err = f.WriteAt(b, off)
					}
					op, cause := "", fmt.Sprintf("%T:%v", err, err)
					matches := true
					var pe *os.PathError
					var code syscall.Errno
					if errors.As(err, &pe) {
						op = pe.Op
						cause = fmt.Sprintf("%T:%v", pe.Err, pe.Err)
						matches = pe.Path == target
					}
					numeric := errors.As(err, &code)
					fmt.Println(kind, method, size, off, n, err == nil, op, cause, numeric, int(code), matches)
					if f != nil && kind != "closed" && kind != "closed-append" {
						f.Close()
					}
				}
			}
		}
	}
}

func seekErrors(root string) {
	path := filepath.Join(root, "seek")
	if err := os.WriteFile(path, []byte("abc"), 0600); err != nil {
		panic(err)
	}
	for _, kind := range []string{"regular", "device", "closed", "nil"} {
		for _, c := range []struct {
			offset int64
			whence int
		}{{0, 0}, {2, 0}, {-1, 0}, {1, 1}, {-6, 2}, {-1, 2}, {0, -1}, {0, 9}, {1 << 33, 0}} {
			target := path
			if kind == "device" {
				target = os.DevNull
			}
			var f *os.File
			var err error
			if kind != "nil" {
				f, err = os.Open(target)
				if err != nil {
					panic(err)
				}
			}
			if kind == "closed" {
				f.Close()
			}
			pos, err := f.Seek(c.offset, c.whence)
			op, cause, matches := "", fmt.Sprintf("%T:%v", err, err), true
			var pe *os.PathError
			var code syscall.Errno
			if errors.As(err, &pe) {
				op = pe.Op
				cause = fmt.Sprintf("%T:%v", pe.Err, pe.Err)
				matches = pe.Path == target
			}
			numeric := errors.As(err, &code)
			fmt.Println(kind, "Seek", c.offset, c.whence, pos, err == nil, op, cause, numeric, int(code), matches)
			if f != nil && kind != "closed" {
				f.Close()
			}
		}
	}
}
