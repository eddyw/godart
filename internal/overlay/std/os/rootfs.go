// Copyright 2024 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build dart

package os

import (
	"io"
	"io/fs"
	"slices"
	"strings"
)

type rootFS struct{ root *Root }

func (f rootFS) Open(name string) (fs.File, error) {
	if !fs.ValidPath(name) {
		return nil, &PathError{Op: "open", Path: name, Err: ErrInvalid}
	}
	file, err := f.root.Open(name)
	if err != nil {
		return nil, err
	}
	return file, nil
}

func (f rootFS) ReadDir(name string) ([]DirEntry, error) {
	if !fs.ValidPath(name) {
		return nil, &PathError{Op: "readdir", Path: name, Err: ErrInvalid}
	}
	file, err := f.root.Open(name)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	dirs, err := file.ReadDir(-1)
	slices.SortFunc(dirs, func(a, b DirEntry) int {
		return strings.Compare(a.Name(), b.Name())
	})
	return dirs, err
}

func (f rootFS) ReadFile(name string) ([]byte, error) {
	if !fs.ValidPath(name) {
		return nil, &PathError{Op: "readfile", Path: name, Err: ErrInvalid}
	}
	file, err := f.root.Open(name)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	return io.ReadAll(file)
}

func (f rootFS) ReadLink(name string) (string, error) {
	if !fs.ValidPath(name) {
		return "", &PathError{Op: "readlink", Path: name, Err: ErrInvalid}
	}
	return f.root.Readlink(name)
}

func (f rootFS) Stat(name string) (fs.FileInfo, error) {
	if !fs.ValidPath(name) {
		return nil, &PathError{Op: "stat", Path: name, Err: ErrInvalid}
	}
	return f.root.Stat(name)
}

func (f rootFS) Lstat(name string) (fs.FileInfo, error) {
	if !fs.ValidPath(name) {
		return nil, &PathError{Op: "lstat", Path: name, Err: ErrInvalid}
	}
	return f.root.Lstat(name)
}
