package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

func main() {
	root, err := os.MkdirTemp("", "directory-errors-")
	if err != nil {
		panic(err)
	}
	defer os.RemoveAll(root)
	path := filepath.Join(root, "file")
	if err := os.WriteFile(path, nil, 0600); err != nil {
		panic(err)
	}
	for _, state := range []string{"nil", "closed-file", "closed-dir", "read-closed-dir", "regular"} {
		for _, method := range []string{"ReadDir", "Readdir", "Readdirnames"} {
			for _, count := range []int{-1, 0, 1} {
				var f *os.File
				if state != "nil" {
					target := path
					if state == "closed-dir" || state == "read-closed-dir" {
						target = root
					}
					f, err = os.Open(target)
					if err != nil {
						panic(err)
					}
					if state == "read-closed-dir" {
						if _, err := f.Readdirnames(1); err != nil {
							panic(err)
						}
					}
					if state != "regular" {
						f.Close()
					}
				}
				var isNil bool
				var n int
				switch method {
				case "ReadDir":
					x, e := f.ReadDir(count)
					isNil, n, err = x == nil, len(x), e
				case "Readdir":
					x, e := f.Readdir(count)
					isNil, n, err = x == nil, len(x), e
				case "Readdirnames":
					x, e := f.Readdirnames(count)
					isNil, n, err = x == nil, len(x), e
				}
				op := ""
				cause := ""
				pathMatches := true
				var pe *os.PathError
				if errors.As(err, &pe) {
					op = pe.Op
					pathMatches = pe.Path == f.Name()
					cause = fmt.Sprintf("%T:%v", pe.Err, pe.Err)
				}
				fmt.Println(state, method, count, isNil, n, errors.Is(err, os.ErrInvalid), errors.Is(err, os.ErrClosed), op, cause, pathMatches)
				if state == "regular" {
					f.Close()
				}
			}
		}
	}
}
