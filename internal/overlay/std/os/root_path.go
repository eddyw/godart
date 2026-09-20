// Copyright 2009, 2010, 2024 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build dart

package os

import (
	"errors"
	"slices"
	"syscall"
)

const rootMaxSymlinks = 8

func checkPathEscapesInternal(r *Root, name string, lstat bool) error {
	if r.closed {
		return ErrClosed
	}
	parts, endsInSlash, err := splitPathInRoot(name, nil, nil)
	if err != nil {
		return err
	}

	i := 0
	symlinks := 0
	base := r.real
	for i < len(parts) {
		if parts[i] == ".." {
			// Resolve one or more parent ("..") path components.
			end := i + 1
			for end < len(parts) && parts[end] == ".." {
				end++
			}
			count := end - i
			if count > i {
				return errPathEscapes
			}
			parts = slices.Delete(parts, i-count, end)
			i -= count
			base = r.real
			for j := range i {
				base = joinPath(base, parts[j])
			}
			continue
		}

		part := parts[i]
		if i == len(parts)-1 {
			if lstat && !endsInSlash {
				break
			}
		}

		next := joinPath(base, part)
		fi, err := Lstat(next)
		if err != nil {
			if IsNotExist(err) {
				// Later parent components can reach existing symlinks before MkdirAll creates this path.
				base = next
				i++
				continue
			}
			return underlyingError(err)
		}
		if fi.Mode()&ModeSymlink != 0 {
			link, err := Readlink(next)
			if err != nil {
				return errPathEscapes
			}
			symlinks++
			if symlinks > rootMaxSymlinks {
				return errors.New("too many symlinks")
			}
			newparts, newEndsInSlash, err := splitPathInRoot(link, parts[:i], parts[i+1:])
			if err != nil {
				return err
			}
			if i == len(parts)-1 && newEndsInSlash {
				endsInSlash = true
			}
			parts = newparts
			continue
		}
		if !fi.IsDir() && i < len(parts)-1 {
			return syscall.ENOTDIR
		}

		base = next
		i++
	}
	return nil
}

func splitPathInRoot(s string, prefix, suffix []string) (_ []string, endsInSlash bool, err error) {
	if len(s) == 0 {
		return nil, false, errors.New("empty path")
	}
	if IsPathSeparator(s[0]) {
		return nil, false, errPathEscapes
	}

	parts := slices.Clone(prefix)
	i, j := 0, 1
	for {
		if j < len(s) && !IsPathSeparator(s[j]) {
			// Keep looking for the end of this component.
			j++
			continue
		}
		parts = append(parts, s[i:j])
		// Advance to the next component, or end of the path.
		partEnd := j
		for j < len(s) && IsPathSeparator(s[j]) {
			j++
		}
		if j == len(s) {
			// If this is the last path component,
			// check for any trailing path separators.
			if len(s[partEnd:]) > 0 {
				endsInSlash = true
			}
			break
		}
		if parts[len(parts)-1] == "." {
			// Remove "." components, except at the end.
			parts = parts[:len(parts)-1]
		}
		i = j
	}
	if len(suffix) > 0 && len(parts) > 0 && parts[len(parts)-1] == "." {
		// Remove a trailing "." component if we're joining to a suffix.
		parts = parts[:len(parts)-1]
	}
	parts = append(parts, suffix...)
	return parts, endsInSlash, nil
}

func joinPath(dir, name string) string {
	if len(dir) > 0 && IsPathSeparator(dir[len(dir)-1]) {
		return dir + name
	}
	return dir + string(PathSeparator) + name
}

func endsWithDot(path string) bool {
	if path == "." {
		return true
	}
	if len(path) >= 2 && path[len(path)-1] == '.' && IsPathSeparator(path[len(path)-2]) {
		return true
	}
	return false
}
