package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"syscall"
)

func must(err error) {
	if err != nil {
		panic(err)
	}
}
func panics(f func()) (result bool) {
	defer func() { result = recover() != nil }()
	f()
	return
}

func main() {
	fmt.Println(os.Stdin.Fd() == 0, os.Stdout.Fd() == 1, os.Stderr.Fd() == 2)
	var absent *os.File
	fmt.Println(absent.Fd() == ^uintptr(0))
	var zero os.File
	fmt.Println(panics(func() { zero.Fd() }), errors.Is(zero.Close(), syscall.EINVAL))
	root, err := os.MkdirTemp("", "gart-filefd-")
	must(err)
	defer os.RemoveAll(root)
	first, err := os.Create(filepath.Join(root, "first"))
	must(err)
	copy := *first
	fd := first.Fd()
	fmt.Println(fd > 2, fd == copy.Fd(), fd == first.Fd())
	must(first.Close())
	second, err := os.Create(filepath.Join(root, "second"))
	must(err)
	defer second.Close()
	_, err = copy.Write([]byte("wrong"))
	fmt.Println(errors.Is(err, os.ErrClosed), errors.Is(copy.Close(), os.ErrClosed))
	fmt.Println(first.Fd() == ^uintptr(0), copy.Fd() == ^uintptr(0))
	_, err = second.Write([]byte("right"))
	must(err)
	must(second.Sync())
	data, err := os.ReadFile(second.Name())
	must(err)
	fmt.Println(string(data))
	dir, err := os.Open(root)
	must(err)
	dirCopy := *dir
	dirFD := dir.Fd()
	fmt.Println(dirFD > 2, dirCopy.Fd() == dirFD)
	fmt.Println(dir.Fd() == dirFD)
	must(dirCopy.Close())
	fmt.Println(dir.Fd() == ^uintptr(0), errors.Is(dir.Close(), os.ErrClosed))
}
