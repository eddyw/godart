// Copyright 2009, 2010, 2024 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build dart

package os

import (
	"errors"
	"slices"
)

const rootMaxSymlinks = 8

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
