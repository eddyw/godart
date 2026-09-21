package main

import (
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

func main() {
	root, err := os.MkdirTemp("", "gart-statsys-")
	must(err)
	defer os.RemoveAll(root)
	path := filepath.Join(root, "file")
	must(os.WriteFile(path, []byte("data"), 0600))
	info, err := os.Stat(path)
	must(err)
	raw := info.Sys().(*syscall.Stat_t)
	fmt.Println("owner", raw.Uid == uint32(os.Getuid()), raw.Gid == uint32(os.Getgid()))
	fmt.Println("metadata", raw.Size == 4, raw.Mode&0777 == 0600, raw.Mode&0170000 == 0100000)
	fmt.Println("allocation", raw.Blksize > 0, raw.Blocks > 0)
	fmt.Println("mtime", rawModTime(raw).Equal(info.ModTime()))
	fmt.Println("pointer", info.Sys() == info.Sys())
	must(os.Link(path, filepath.Join(root, "link")))
	linked, err := os.Stat(filepath.Join(root, "link"))
	must(err)
	linkRaw := linked.Sys().(*syscall.Stat_t)
	fmt.Println("snapshot", raw.Nlink == 1, linkRaw.Nlink == 2)
	fmt.Println("identity", raw.Dev == linkRaw.Dev, raw.Ino == linkRaw.Ino, os.SameFile(info, linked))
	oldMode, oldSize := info.Mode(), info.Size()
	raw.Dev++
	raw.Mode = 0
	raw.Size = 99
	fmt.Println("mutable identity", !os.SameFile(info, linked))
	fmt.Println("snapshot fields", info.Mode() == oldMode, info.Size() == oldSize)
	must(os.Symlink("file", filepath.Join(root, "symbolic")))
	symbolic, err := os.Lstat(filepath.Join(root, "symbolic"))
	must(err)
	symbolicRaw := symbolic.Sys().(*syscall.Stat_t)
	fmt.Println("link metadata", symbolicRaw.Ino != linkRaw.Ino, symbolicRaw.Mode&0170000 == 0120000, symbolicRaw.Size == 4)
	file, err := os.Open(path)
	must(err)
	must(os.Remove(path))
	must(os.Remove(filepath.Join(root, "link")))
	opened, err := file.Stat()
	must(err)
	openedRaw := opened.Sys().(*syscall.Stat_t)
	fmt.Println("unlinked", openedRaw.Nlink == 0, openedRaw.Ino == linkRaw.Ino, openedRaw.Size == 4)
	must(file.Close())
}
