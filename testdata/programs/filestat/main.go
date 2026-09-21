package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

type wrappedInfo struct{ os.FileInfo }

func must(err error) {
	if err != nil {
		panic(err)
	}
}

func stat(path string) os.FileInfo {
	i, err := os.Stat(path)
	must(err)
	return i
}

func main() {
	if len(os.Args) == 2 {
		path := os.Args[1]
		info := stat(path)
		link, err := os.Lstat(path)
		must(err)
		f, err := os.Open(path)
		must(err)
		opened, err := f.Stat()
		must(err)
		must(f.Close())
		for _, entry := range []os.FileInfo{info, link, opened} {
			fmt.Println(entry.ModTime().Unix(), entry.ModTime().Nanosecond())
		}
		return
	}
	root, err := os.MkdirTemp("", "gart-filestat-")
	must(err)
	defer os.RemoveAll(root)
	a, b := filepath.Join(root, "a"), filepath.Join(root, "b")
	stamp := time.Unix(1700000000, 0)
	for _, path := range []string{a, b} {
		must(os.WriteFile(path, []byte("old"), 0600))
		must(os.Chtimes(path, stamp, stamp))
	}
	ia, ib := stat(a), stat(b)
	fmt.Println("distinct", !os.SameFile(ia, ib))
	fmt.Println("snapshot", os.SameFile(ia, stat(a)))
	fmt.Println("wrapper", !os.SameFile(wrappedInfo{ia}, wrappedInfo{ia}))
	fmt.Println("nil", !os.SameFile(nil, ia))
	fmt.Println("permissions", ia.Mode() == 0600)
	fmt.Println("time", ia.ModTime().Equal(stamp))
	f, err := os.Open(a)
	must(err)
	defer f.Close()
	moved := filepath.Join(root, "moved")
	must(os.Rename(a, moved))
	must(os.WriteFile(a, []byte("replacement"), 0640))
	opened, err := f.Stat()
	must(err)
	fmt.Println("descriptor", os.SameFile(ia, opened), opened.Size() == 3, opened.Name() == "a")
	fmt.Println("replacement", !os.SameFile(ia, stat(a)))
	link := filepath.Join(root, "hard")
	must(os.Link(moved, link))
	fmt.Println("hard link", os.SameFile(ia, stat(link)))
	must(os.Remove(moved))
	must(os.Remove(link))
	removed, err := f.Stat()
	must(err)
	fmt.Println("unlinked", os.SameFile(ia, removed), removed.Size() == 3)
	for _, name := range []string{"first", "second"} {
		must(os.Symlink("a", filepath.Join(root, name)))
	}
	l1, err := os.Lstat(filepath.Join(root, "first"))
	must(err)
	l2, err := os.Lstat(filepath.Join(root, "second"))
	must(err)
	fmt.Println("symlinks", !os.SameFile(l1, l2), l1.Mode()&os.ModeSymlink != 0, l1.Size() == 1, !l1.ModTime().IsZero())
	fmt.Println("follow", os.SameFile(stat(a), stat(filepath.Join(root, "first"))))
	dir, err := os.Open(root)
	must(err)
	di, err := dir.Stat()
	must(err)
	must(dir.Close())
	fmt.Println("directory", di.IsDir(), os.SameFile(di, stat(root)))
	dev, err := os.Open(os.DevNull)
	must(err)
	dv, err := dev.Stat()
	must(err)
	must(dev.Close())
	fmt.Println("device", dv.Mode()&os.ModeCharDevice != 0, dv.Mode()&os.ModeDevice != 0, !dv.Mode().IsRegular())
	must(f.Close())
	_, err = f.Stat()
	fmt.Println("closed", errors.Is(err, os.ErrClosed))
	var absent *os.File
	_, err = absent.Stat()
	fmt.Println("absent", errors.Is(err, os.ErrInvalid))
}
