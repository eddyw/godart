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
	syncTruncateErrors(root)
	pathTruncateErrors(root)
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

func syncTruncateErrors(root string) {
	path := filepath.Join(root, "attributes")
	if err := os.WriteFile(path, []byte("abc"), 0600); err != nil {
		panic(err)
	}
	for _, kind := range []string{"read-only", "write-only", "read-write", "directory", "device", "closed", "nil"} {
		for _, method := range []string{"Sync", "Truncate"} {
			sizes := []int64{0}
			if method == "Truncate" {
				sizes = []int64{-1, 0, 7}
			}
			for _, size := range sizes {
				target, flags := path, os.O_RDONLY
				switch kind {
				case "write-only":
					flags = os.O_WRONLY
				case "read-write":
					flags = os.O_RDWR
				case "directory":
					target = root
				case "device":
					target = os.DevNull
				}
				var f *os.File
				var err error
				if kind != "nil" {
					f, err = os.OpenFile(target, flags, 0)
					if err != nil {
						panic(err)
					}
				}
				if kind == "closed" {
					f.Close()
				}
				if method == "Sync" {
					err = f.Sync()
				} else {
					err = f.Truncate(size)
				}
				op, cause, matches := "", fmt.Sprintf("%T:%v", err, err), true
				var pe *os.PathError
				var code syscall.Errno
				if errors.As(err, &pe) {
					op = pe.Op
					cause = fmt.Sprintf("%T:%v", pe.Err, pe.Err)
					matches = pe.Path == target
				}
				numeric := errors.As(err, &code)
				fmt.Println(kind, method, size, err == nil, op, cause, numeric, int(code), matches)
				if f != nil && kind != "closed" {
					f.Close()
				}
			}
		}
	}
}

func pathTruncateErrors(root string) {
	file := filepath.Join(root, "path-truncate")
	if err := os.WriteFile(file, []byte("abc"), 0200); err != nil {
		panic(err)
	}
	link := filepath.Join(root, "link")
	if err := os.Symlink(file, link); err != nil {
		panic(err)
	}
	dangling := filepath.Join(root, "dangling")
	if err := os.Symlink(filepath.Join(root, "missing"), dangling); err != nil {
		panic(err)
	}
	raw := filepath.Join(root, "raw-"+string([]byte{255}))
	paths := []string{file, root, os.DevNull, link, dangling, filepath.Join(root, "missing"), "", file + "/", file + "/child", file + "\x00", raw}
	for i, path := range paths {
		for _, size := range []int64{-1, 0, 7} {
			err := os.Truncate(path, size)
			var pe *os.PathError
			var code syscall.Errno
			op, cause, matches := "", fmt.Sprintf("%T:%v", err, err), true
			if errors.As(err, &pe) {
				op = pe.Op
				cause = fmt.Sprintf("%T:%v", pe.Err, pe.Err)
				matches = pe.Path == path
			}
			numeric := errors.As(err, &code)
			fmt.Println(i, size, err == nil, op, cause, numeric, int(code), matches)
		}
	}
}
