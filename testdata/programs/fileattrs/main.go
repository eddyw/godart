package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

func must(err error) {
	if err != nil {
		panic(err)
	}
}

func mode(path string) os.FileMode {
	info, err := os.Stat(path)
	must(err)
	return info.Mode().Perm()
}

func main() {
	root, err := os.MkdirTemp("", "gart-fileattrs-")
	must(err)
	defer os.RemoveAll(root)
	path := filepath.Join(root, "file")
	must(os.WriteFile(path, []byte("old"), 0600))
	f, err := os.Open(path)
	must(err)
	defer f.Close()
	moved := filepath.Join(root, "moved")
	must(os.Rename(path, moved))
	must(os.WriteFile(path, []byte("replacement"), 0600))
	must(f.Chmod(0400))
	fmt.Println("renamed", mode(moved) == 0400, mode(path) == 0600)
	must(os.Remove(moved))
	must(os.Remove(path))
	must(f.Chmod(0600))
	must(f.Chown(-1, -1))
	info, err := f.Stat()
	must(err)
	fmt.Println("unlinked", info.Mode().Perm() == 0600, info.Size() == 3)
	dirPath := filepath.Join(root, "dir")
	must(os.Mkdir(dirPath, 0700))
	dir, err := os.Open(dirPath)
	must(err)
	defer dir.Close()
	dirMoved := filepath.Join(root, "moved-dir")
	must(os.Rename(dirPath, dirMoved))
	must(os.Mkdir(dirPath, 0700))
	must(dir.Chmod(0500))
	must(dir.Chown(-1, -1))
	fmt.Println("directory", mode(dirMoved) == 0500, mode(dirPath) == 0700)
	must(dir.Chmod(0700))
	must(f.Close())
	fmt.Println("closed", errors.Is(f.Chmod(0600), os.ErrClosed), errors.Is(f.Chown(-1, -1), os.ErrClosed))
	var absent *os.File
	fmt.Println("absent", errors.Is(absent.Chmod(0600), os.ErrInvalid), errors.Is(absent.Chown(-1, -1), os.ErrInvalid))
}
