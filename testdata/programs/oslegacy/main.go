package main

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"syscall"
)

type custom struct{ error }

func (e custom) Is(target error) bool { return e.error == target }
func (e custom) Unwrap() error        { return e.error }

type timeout struct{}

func (timeout) Error() string { return "timeout" }
func (timeout) Timeout() bool { return true }

func main() {
	for _, err := range []error{
		nil, fs.ErrExist, fs.ErrPermission, fs.ErrNotExist,
		syscall.EEXIST, syscall.EACCES, syscall.ENOENT, syscall.ENOTDIR, syscall.EPERM,
		custom{fs.ErrPermission}, custom{fs.ErrNotExist},
		&os.PathError{Err: fs.ErrPermission}, &os.LinkError{Err: fs.ErrExist},
		&os.SyscallError{Err: syscall.ENOENT},
		&os.PathError{Err: &os.SyscallError{Err: syscall.ENOENT}},
		timeout{}, custom{timeout{}}, &os.PathError{Err: timeout{}},
		&os.SyscallError{Err: timeout{}},
	} {
		fmt.Println(os.IsExist(err), os.IsNotExist(err), os.IsPermission(err), os.IsTimeout(err), errors.Is(err, errors.ErrUnsupported))
	}
	dir, err := os.MkdirTemp("", "oslegacy-")
	if err != nil {
		panic(err)
	}
	defer os.RemoveAll(dir)
	path := filepath.Join(dir, "file")
	if err := os.WriteFile(path, []byte("keep"), 0600); err != nil {
		panic(err)
	}
	for _, name := range []string{path + "\x00", filepath.Join(dir, "absent") + "\x00"} {
		for _, flag := range []int{os.O_RDONLY, os.O_RDWR, os.O_CREATE | os.O_RDWR, os.O_CREATE | os.O_WRONLY | os.O_TRUNC} {
			f, err := os.OpenFile(name, flag, 0600)
			if f != nil {
				f.Close()
			}
			fmt.Println(f == nil, errors.Is(err, syscall.EINVAL))
		}
	}
	data, err := os.ReadFile(path)
	fmt.Println(string(data), err)
}
