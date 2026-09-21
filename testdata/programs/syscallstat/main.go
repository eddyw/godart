package main

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"syscall"
)

func must(err error) {
	if err != nil {
		panic(err)
	}
}
func same(path string, follow bool, stat *syscall.Stat_t) bool {
	var info os.FileInfo
	var err error
	if follow {
		info, err = os.Stat(path)
	} else {
		info, err = os.Lstat(path)
	}
	must(err)
	return reflect.DeepEqual(stat, info.Sys())
}
func main() {
	root, err := os.MkdirTemp("", "gart-syscallstat-")
	must(err)
	defer os.RemoveAll(root)
	path := filepath.Join(root, "file")
	must(os.WriteFile(path, []byte("original"), 0600))
	var stat syscall.Stat_t
	must(syscall.Stat(path, &stat))
	fmt.Println("stat", same(path, true, &stat), stat.Size == 8, stat.Mode&0777 == 0600)
	must(syscall.Lstat(path, &stat))
	fmt.Println("lstat", same(path, false, &stat))
	link := filepath.Join(root, "link")
	must(os.Symlink("file", link))
	must(syscall.Lstat(link, &stat))
	fmt.Println("symlink", same(link, false, &stat), stat.Mode&syscall.S_IFMT == syscall.S_IFLNK)
	must(syscall.Stat(link, &stat))
	fmt.Println("follow", same(link, true, &stat), stat.Mode&syscall.S_IFMT == syscall.S_IFREG)
	f, err := os.Open(path)
	must(err)
	defer f.Close()
	fd := int(f.Fd())
	must(syscall.Fstat(fd, &stat))
	info, err := f.Stat()
	must(err)
	fmt.Println("descriptor", reflect.DeepEqual(&stat, info.Sys()))
	original := stat.Ino
	moved := filepath.Join(root, "moved")
	must(os.Rename(path, moved))
	must(os.WriteFile(path, []byte("replacement"), 0600))
	must(syscall.Fstat(fd, &stat))
	fmt.Println("renamed", stat.Ino == original, stat.Size == 8)
	must(os.Remove(moved))
	must(syscall.Fstat(fd, &stat))
	fmt.Println("unlinked", stat.Ino == original, stat.Size == 8, stat.Nlink == 0)
	inspectErrors := len(os.Args) > 1 && os.Args[1] == "nil"
	if inspectErrors {
		fmt.Println("nil", syscall.Stat(path, nil) == syscall.EFAULT, syscall.Lstat(path, nil) == syscall.EFAULT, syscall.Fstat(fd, nil) == syscall.EFAULT)
		fmt.Println("nil precedence", syscall.Stat(path+"/missing", nil) == syscall.ENOTDIR, syscall.Stat(path+"\x00", nil) == syscall.EINVAL, syscall.Fstat(-1, nil) == syscall.EBADF)
	}
	before := stat
	fmt.Println("missing", syscall.Stat(filepath.Join(root, "missing"), &stat) == syscall.ENOENT)
	if inspectErrors {
		fmt.Println("missing buffer", reflect.DeepEqual(before, stat))
	}
	fmt.Println("NUL", syscall.Lstat(path+"\x00", &stat) == syscall.EINVAL)
	if inspectErrors {
		fmt.Println("NUL buffer", reflect.DeepEqual(before, stat))
	}
	must(f.Close())
	fmt.Println("closed", syscall.Fstat(fd, &stat) == syscall.EBADF)
	if inspectErrors {
		fmt.Println("closed buffer", reflect.DeepEqual(before, stat))
	}
	must(syscall.Stat(root, &stat))
	fmt.Println("directory", same(root, true, &stat), stat.Mode&syscall.S_IFMT == syscall.S_IFDIR)
}
