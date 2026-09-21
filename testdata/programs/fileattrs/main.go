package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
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
	pathAttributes(root)
}

func pathAttributes(root string) {
	path := filepath.Join(root, "attributes")
	must(os.WriteFile(path, nil, 0600))
	link := filepath.Join(root, "link")
	must(os.Symlink(path, link))
	broken := filepath.Join(root, "broken")
	must(os.Symlink(filepath.Join(root, "absent"), broken))
	loop := filepath.Join(root, "loop")
	must(os.Symlink(loop, loop))
	for _, call := range []struct {
		name string
		fn   func(string) error
	}{
		{"chmod", func(path string) error { return os.Chmod(path, 0640) }},
		{"chown", func(path string) error { return os.Chown(path, -1, -1) }},
		{"lchown", func(path string) error { return os.Lchown(path, -1, -1) }},
	} {
		for _, item := range []struct {
			path string
			want syscall.Errno
		}{
			{filepath.Join(root, "absent"), syscall.ENOENT},
			{filepath.Join(path, "child"), syscall.ENOTDIR},
			{path + "\x00suffix", syscall.EINVAL},
		} {
			err := call.fn(item.path)
			pe, ok := err.(*os.PathError)
			fmt.Println(call.name, item.want, ok && pe.Op == call.name && pe.Path == item.path && pe.Err == item.want)
		}
	}
	must(os.Chmod(link, 0640))
	fmt.Println("link mode", mode(path) == 0640)
	fmt.Println("broken ownership", os.Lchown(broken, -1, -1) == nil, errors.Is(os.Chown(broken, -1, -1), syscall.ENOENT))
	fmt.Println("loop ownership", os.Lchown(loop, -1, -1) == nil, errors.Is(os.Chown(loop, -1, -1), syscall.ELOOP))
	fmt.Println("loop mode", errors.Is(os.Chmod(loop, 0600), syscall.ELOOP))
	fmt.Println("negative owner", errors.Is(os.Chown(path, -2, -1), syscall.EPERM))
	if runtime.GOOS == "linux" {
		raw := filepath.Join(root, "raw-\xff")
		must(os.Mkdir(raw, 0700))
		must(os.Chmod(raw, 0500))
		must(os.Chown(raw, -1, -1))
		must(os.Lchown(raw, -1, -1))
		fmt.Println("raw directory", mode(raw) == 0500)
		must(os.Chmod(raw, 0700))
	}
	oldPath, present := os.LookupEnv("PATH")
	defer func() {
		if present {
			must(os.Setenv("PATH", oldPath))
		} else {
			must(os.Unsetenv("PATH"))
		}
	}()
	must(os.Setenv("PATH", ""))
	must(os.Chmod(path, 0600))
	must(os.Chown(path, os.Getuid(), os.Getgid()))
	must(os.Lchown(link, os.Getuid(), os.Getgid()))
	fmt.Println("no commands", mode(path) == 0600)
}
