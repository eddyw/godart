//go:build dart

// Package filepath holds the path operations of the file system.
//
// The separator is always `/`. Go selects the separator with a build tag of the
// operating system, and the emitted Dart names the system only at run time.
// docs/parity.md holds that gap.
//
// The walk of a directory tree stays out. `Walk` and `WalkDir` each read the
// package io/fs, and that package stays outside the surface.
package filepath

import (
	"errors"
	"os"
	"path"
	"strings"
)

const (
	Separator     = '/'
	ListSeparator = ':'
)

// ErrBadPattern names a pattern that Match cannot read.
var ErrBadPattern = path.ErrBadPattern

// Clean gives the shortest path of equal meaning.
func Clean(p string) string {
	if p == "" {
		return "."
	}
	return path.Clean(p)
}

// ToSlash gives the path with `/` between 2 elements. The separator is already
// `/`, thus the result is the argument.
func ToSlash(p string) string { return p }

// FromSlash gives the path with the separator of the system between 2 elements.
// The separator is already `/`, thus the result is the argument.
func FromSlash(p string) string { return p }

// SplitList cuts a list of paths at each list separator.
func SplitList(p string) []string {
	if p == "" {
		return []string{}
	}
	return strings.Split(p, string(ListSeparator))
}

// Split cuts the path at the last separator. The directory holds that
// separator, and the file holds no separator.
func Split(p string) (string, string) {
	i := strings.LastIndexByte(p, Separator)
	return p[:i+1], p[i+1:]
}

// Join builds a path of the elements, with a separator between 2 of them.
func Join(elem ...string) string {
	return path.Join(elem...)
}

// Ext gives the suffix of the last element, from the last dot.
func Ext(p string) string { return path.Ext(p) }

// Base gives the last element of the path.
func Base(p string) string { return path.Base(p) }

// Dir gives each element of the path except the last one.
func Dir(p string) string { return path.Dir(p) }

// VolumeName gives the volume of the path. A path of this system holds none.
func VolumeName(p string) string { return "" }

// IsAbs reports whether the path starts at the root.
func IsAbs(p string) bool { return path.IsAbs(p) }

// IsLocal reports whether the path stays inside the directory that holds it.
func IsLocal(p string) bool {
	if p == "" || IsAbs(p) {
		return false
	}
	clean := Clean(p)
	return clean != ".." && !strings.HasPrefix(clean, "../")
}

// Match reports whether the name fits the pattern.
func Match(pattern, name string) (bool, error) {
	return path.Match(pattern, name)
}

// Abs gives the absolute form of the path. A relative path takes the working
// directory in front of it.
func Abs(p string) (string, error) {
	if IsAbs(p) {
		return Clean(p), nil
	}
	wd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	return Join(wd, p), nil
}

// Rel gives the path of targPath relative to basePath. The 2 paths must both
// start at the root, or both start at the working directory.
func Rel(basePath, targPath string) (string, error) {
	base := Clean(basePath)
	targ := Clean(targPath)
	if targ == base {
		return ".", nil
	}
	if base == "." {
		base = ""
	}
	baseSlashed := len(base) > 0 && base[0] == Separator
	targSlashed := len(targ) > 0 && targ[0] == Separator
	if baseSlashed != targSlashed {
		return "", errors.New("Rel: can't make " + targPath + " relative to " + basePath)
	}
	// Move to the first element that the 2 paths do not share.
	bl := len(base)
	tl := len(targ)
	b0, bi, t0, ti := 0, 0, 0, 0
	for {
		for bi < bl && base[bi] != Separator {
			bi++
		}
		for ti < tl && targ[ti] != Separator {
			ti++
		}
		if targ[t0:ti] != base[b0:bi] {
			break
		}
		if bi < bl {
			bi++
		}
		if ti < tl {
			ti++
		}
		b0 = bi
		t0 = ti
	}
	if base[b0:bi] == ".." {
		return "", errors.New("Rel: can't make " + targPath + " relative to " + basePath)
	}
	if b0 != bl {
		// The base holds elements that the target does not, thus the result
		// goes up before it goes down.
		seps := strings.Count(base[b0:bl], string(Separator))
		out := ".."
		for i := 0; i < seps; i++ {
			out += string(Separator) + ".."
		}
		if t0 != tl {
			out += string(Separator) + targ[t0:]
		}
		return Clean(out), nil
	}
	return targ[t0:], nil
}

// Glob gives each path that fits the pattern. It reports no error for a path
// that it cannot read.
func Glob(pattern string) ([]string, error) {
	return globWithLimit(pattern, 0)
}

// pathSeparatorsLimit bounds the depth of the walk of a pattern. A deeper walk
// would fill the stack.
const pathSeparatorsLimit = 10000

func globWithLimit(pattern string, depth int) ([]string, error) {
	if depth == pathSeparatorsLimit {
		return nil, ErrBadPattern
	}
	if _, err := Match(pattern, ""); err != nil {
		return nil, err
	}
	if !hasMeta(pattern) {
		if _, err := os.Stat(pattern); err != nil {
			return nil, nil
		}
		return []string{pattern}, nil
	}
	dir, file := Split(pattern)
	dir = cleanGlobPath(dir)
	if !hasMeta(dir) {
		return glob(dir, file, nil)
	}
	if dir == pattern {
		return nil, ErrBadPattern
	}
	m, err := globWithLimit(dir, depth+1)
	if err != nil {
		return nil, err
	}
	var matches []string
	for _, d := range m {
		matches, err = glob(d, file, matches)
		if err != nil {
			return nil, err
		}
	}
	return matches, nil
}

func cleanGlobPath(p string) string {
	switch p {
	case "":
		return "."
	case string(Separator):
		return p
	}
	return p[:len(p)-1]
}

func hasMeta(p string) bool {
	return strings.ContainsAny(p, `*?[\`)
}

func glob(dir, pattern string, matches []string) ([]string, error) {
	fi, err := os.Stat(dir)
	if err != nil || !fi.IsDir() {
		// A path that the program cannot read gives no match and no error.
		return matches, nil
	}
	names, err := os.ReadNames(dir)
	if err != nil {
		return matches, nil
	}
	sortNames(names)
	for _, n := range names {
		matched, err := Match(pattern, n)
		if err != nil {
			return matches, err
		}
		if matched {
			matches = append(matches, Join(dir, n))
		}
	}
	return matches, nil
}

// sortNames orders the names of a directory. Glob gives the matches in that
// order, exactly as Go does.
func sortNames(names []string) {
	for i := 1; i < len(names); i++ {
		for j := i; j > 0 && names[j] < names[j-1]; j-- {
			names[j], names[j-1] = names[j-1], names[j]
		}
	}
}
