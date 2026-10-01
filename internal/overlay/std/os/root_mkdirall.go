// Copyright 2024 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build dart

package os

import (
	"errors"
	"runtime"
	"slices"
	"strings"
	"syscall"
)

// MkdirAll makes a directory of the root and each parent of it.
func (r *Root) MkdirAll(name string, perm FileMode) error {
	if perm&0o777 != perm {
		return &PathError{Op: "mkdirat", Path: name, Err: errors.New("unsupported file mode")}
	}
	if r.closed {
		return &PathError{Op: "mkdirat", Path: name, Err: ErrClosed}
	}
	parts, endsInSlash, err := splitPathInRoot(name, nil, nil)
	if err != nil {
		return rootError("mkdirat", name, err)
	}
	resolveSlash := endsInSlash && runtime.GOOS != "linux" && runtime.GOOS != "windows" && runtime.GOOS != "openbsd"
	base := r.real
	i, steps, restarts, symlinks := 0, 0, 0, 0
	for {
		steps++
		// Match the native traversal bound for repeated parent components.
		if steps > 255 && restarts > 8 {
			return rootError("mkdirat", name, syscall.ENAMETOOLONG)
		}
		if parts[i] == ".." {
			restarts++
			end := i + 1
			for end < len(parts) && parts[end] == ".." {
				end++
			}
			count := end - i
			if count > i {
				return rootError("mkdirat", name, errPathEscapes)
			}
			parts = slices.Delete(parts, i-count, end)
			if len(parts) == 0 {
				parts = []string{"."}
			}
			i, base = 0, r.real
			continue
		}
		last := i == len(parts)-1
		path := joinPath(base, parts[i])
		component := strings.Join(parts[:i+1], string(PathSeparator))
		if last && !resolveSlash {
			return r.mkdirAllLast(path, name, component, perm)
		}
		info, err := Lstat(path)
		if !last && IsNotExist(err) {
			if cause := Mkdir(path, perm); cause != nil && !IsExist(cause) {
				return rootError("mkdirat", component, cause)
			}
			info, err = Lstat(path)
		}
		if err != nil {
			if last {
				if IsNotExist(err) {
					return r.mkdirAllLast(path, name, component, perm)
				}
				return rootError("mkdirat", name, err)
			}
			return rootError("openat", component, err)
		}
		if info.Mode()&ModeSymlink != 0 {
			link, err := Readlink(path)
			if err != nil {
				return rootError("mkdirat", name, err)
			}
			symlinks++
			if symlinks > rootMaxSymlinks {
				return rootError("mkdirat", name, syscall.ELOOP)
			}
			newparts, newEndsInSlash, err := splitPathInRoot(link, parts[:i], parts[i+1:])
			if err != nil {
				return rootError("mkdirat", name, err)
			}
			if last && newEndsInSlash {
				resolveSlash = runtime.GOOS != "linux" && runtime.GOOS != "windows" && runtime.GOOS != "openbsd"
			}
			if len(newparts) < i || !slices.Equal(parts[:i], newparts[:i]) {
				i, base = 0, r.real
			}
			parts = newparts
			continue
		}
		if !info.IsDir() {
			if last {
				return rootError("mkdirat", name, syscall.ENOTDIR)
			}
			return rootError("openat", component, syscall.ENOTDIR)
		}
		if last {
			return r.mkdirAllLast(path, name, component, perm)
		}
		f, err := Open(path)
		if err != nil {
			return rootError("openat", component, err)
		}
		f.Close()
		base = path
		i++
	}
}

func (r *Root) mkdirAllLast(path, name, component string, perm FileMode) error {
	err := underlyingError(Mkdir(path, perm))
	if err == syscall.EEXIST {
		info, cause := Lstat(path)
		if cause == nil {
			if info.IsDir() {
				return nil
			}
			if info.Mode()&ModeSymlink != 0 {
				info, cause = r.Stat(name)
				if cause == nil {
					if info.IsDir() {
						return nil
					}
					err = syscall.ENOTDIR
				} else if !IsNotExist(cause) {
					err = cause
				}
			}
		}
	}
	if err != nil {
		return &PathError{Op: "mkdirat", Path: component, Err: err}
	}
	return nil
}
