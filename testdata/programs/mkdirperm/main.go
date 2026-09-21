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

func mode(path string) os.FileMode {
	info, err := os.Stat(path)
	must(err)
	return info.Mode()
}

func main() {
	root, err := os.MkdirTemp("", "gart-mkdirperm-")
	must(err)
	defer os.RemoveAll(root)
	fmt.Printf("temporary %04o\n", mode(root).Perm())
	for i, permission := range []os.FileMode{0700, 0750, 0777, 0000, 0700 | os.ModeSticky} {
		path := filepath.Join(root, fmt.Sprint(i))
		must(os.Mkdir(path, permission))
		fmt.Printf("mode %d %04o %t\n", i, mode(path).Perm(), mode(path)&os.ModeSticky != 0)
		fmt.Println("existing", errors.Is(os.Mkdir(path, 0777), os.ErrExist))
		must(os.Chmod(path, 0700))
	}
	child := filepath.Join(root, "parent", "child")
	must(os.MkdirAll(child, 0700))
	fmt.Printf("parents %04o %04o\n", mode(filepath.Dir(child)).Perm(), mode(child).Perm())
	must(os.MkdirAll(child, 0777))
	fmt.Printf("retained %04o\n", mode(child).Perm())
	file := filepath.Join(root, "file")
	must(os.WriteFile(file, nil, 0600))
	fmt.Println("file", errors.Is(os.Mkdir(file, 0700), os.ErrExist))
	fmt.Println("parent", errors.Is(os.Mkdir(filepath.Join(root, "absent", "child"), 0700), os.ErrNotExist))
	fmt.Println("not directory", errors.Is(os.Mkdir(filepath.Join(file, "child"), 0700), syscall.ENOTDIR))
	fmt.Println("empty", errors.Is(os.Mkdir("", 0700), os.ErrNotExist))
	fmt.Println("NUL", errors.Is(os.Mkdir(root+"\x00suffix", 0700), syscall.EINVAL))
	link := filepath.Join(root, "link")
	must(os.Symlink("missing", link))
	fmt.Println("symlink", errors.Is(os.Mkdir(link, 0700), os.ErrExist))
}
