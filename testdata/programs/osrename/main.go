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
func exists(path string) bool { _, err := os.Lstat(path); return err == nil }

func main() {
	root, err := os.MkdirTemp("", "gart-rename-")
	must(err)
	defer os.RemoveAll(root)
	from, to := filepath.Join(root, "from"), filepath.Join(root, "to")
	must(os.Mkdir(from, 0700))
	must(os.WriteFile(filepath.Join(from, "value"), []byte("keep"), 0600))
	file, err := os.Open(from)
	must(err)
	defer file.Close()
	fd := file.Fd()
	must(os.Rename(from, to))
	fmt.Println(!exists(from), exists(filepath.Join(to, "value")), file.Fd() == fd)
	must(os.Mkdir(from, 0700))
	fmt.Println(errors.Is(os.Rename(to, from), syscall.EEXIST))
	fmt.Println(exists(from), exists(filepath.Join(to, "value")))
	fmt.Println(errors.Is(os.Rename(from, from), syscall.EEXIST))
	must(os.Rename(from, root+"/./from"))
	fmt.Println(exists(from))
	fmt.Println(errors.Is(os.Rename(filepath.Join(root, "absent"), from), os.ErrNotExist))
	must(os.Remove(from))
	must(os.Symlink(to, from))
	link := filepath.Join(root, "link")
	must(os.Rename(from, link))
	target, err := os.Readlink(link)
	must(err)
	fmt.Println(!exists(from), target == to, exists(filepath.Join(to, "value")))
	must(os.Symlink(filepath.Join(root, "absent"), from))
	broken := filepath.Join(root, "broken")
	must(os.Rename(from, broken))
	fmt.Println(!exists(from), exists(broken))
	must(os.Symlink(to, from))
	fmt.Println(errors.Is(os.Rename(from, to), syscall.EEXIST))
	fmt.Println(exists(from), exists(to))
}
