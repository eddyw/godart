//go:build dart

// Package filepath holds the path operations of the file system.
//
// The separator is always `/`. Go selects the separator with a build tag of the
// operating system, and the emitted Dart names the system only at run time.
// docs/parity.md holds that gap.
package filepath

import (
	"errors"
	"io/fs"
	"os"
	"path"
	"strings"
	"syscall/dart"
	"unicode/utf8"
)

const (
	Separator     = '/'
	ListSeparator = ':'
)

// ErrBadPattern names a pattern that Match cannot read. Go declares 1 such
// value for each of the 2 packages, thus errors.Is of a pattern error of path
// finds this value not.
var ErrBadPattern = errors.New("syntax error in pattern")

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

// Match reports whether name matches the shell file name pattern.
// The pattern syntax is:
//
//	pattern:
//		{ term }
//	term:
//		'*'         matches any sequence of non-Separator characters
//		'?'         matches any single non-Separator character
//		'[' [ '^' ] { character-range } ']'
//		            character class (must be non-empty)
//		c           matches character c (c != '*', '?', '\\', '[')
//		'\\' c      matches character c (except on Windows)
//
//	character-range:
//		c           matches character c (c != '\\', '-', ']')
//		'\\' c      matches character c (except on Windows)
//		lo '-' hi   matches character c for lo <= c <= hi
//
// Path segments in the pattern must be separated by [Separator].
//
// Match requires pattern to match all of name, not just a substring.
// The only possible returned error is [ErrBadPattern], when pattern
// is malformed.
//
// On Windows, escaping is disabled. Instead, '\\' is treated as
// path separator.
func Match(pattern, name string) (matched bool, err error) {
Pattern:
	for len(pattern) > 0 {
		var star bool
		var chunk string
		star, chunk, pattern = scanChunk(pattern)
		if star && chunk == "" {
			// Trailing * matches rest of string unless it has a /.
			return !strings.Contains(name, string(Separator)), nil
		}
		// Look for match at current position.
		t, ok, err := matchChunk(chunk, name)
		// if we're the last chunk, make sure we've exhausted the name
		// otherwise we'll give a false result even if we could still match
		// using the star
		if ok && (len(t) == 0 || len(pattern) > 0) {
			name = t
			continue
		}
		if err != nil {
			return false, err
		}
		if star {
			// Look for match skipping i+1 bytes.
			// Cannot skip /.
			for i := 0; i < len(name) && name[i] != Separator; i++ {
				t, ok, err := matchChunk(chunk, name[i+1:])
				if ok {
					// if we're the last chunk, make sure we exhausted the name
					if len(pattern) == 0 && len(t) > 0 {
						continue
					}
					name = t
					continue Pattern
				}
				if err != nil {
					return false, err
				}
			}
		}
		return false, nil
	}
	return len(name) == 0, nil
}

// scanChunk gets the next segment of pattern, which is a non-star string
// possibly preceded by a star.
func scanChunk(pattern string) (star bool, chunk, rest string) {
	for len(pattern) > 0 && pattern[0] == '*' {
		pattern = pattern[1:]
		star = true
	}
	inrange := false
	for i := 0; i < len(pattern); i++ {
		switch pattern[i] {
		case '\\':
			// error check handled in matchChunk: bad pattern.
			if true /* the surface holds no Windows target */ && i+1 < len(pattern) {
				i++
			}
		case '[':
			inrange = true
		case ']':
			inrange = false
		case '*':
			if !inrange {
				return star, pattern[:i], pattern[i:]
			}
		}
	}
	return star, pattern, ""
}

// matchChunk checks whether chunk matches the beginning of s.
// If so, it returns the remainder of s (after the match).
// Chunk is all single-character operators: literals, char classes, and ?.
func matchChunk(chunk, s string) (rest string, ok bool, err error) {
	// failed records whether the match has failed.
	// After the match fails, the loop continues on processing chunk,
	// checking that the pattern is well-formed but no longer reading s.
	failed := false
	for len(chunk) > 0 {
		failed = failed || len(s) == 0
		switch chunk[0] {
		case '[':
			// character class
			var r rune
			if !failed {
				var n int
				r, n = utf8.DecodeRuneInString(s)
				s = s[n:]
			}
			chunk = chunk[1:]
			// possibly negated
			negated := false
			if len(chunk) > 0 && chunk[0] == '^' {
				negated = true
				chunk = chunk[1:]
			}
			// parse all ranges
			match := false
			nrange := 0
			for {
				if len(chunk) > 0 && chunk[0] == ']' && nrange > 0 {
					chunk = chunk[1:]
					break
				}
				var lo, hi rune
				if lo, chunk, err = getEsc(chunk); err != nil {
					return "", false, err
				}
				hi = lo
				if chunk[0] == '-' {
					if hi, chunk, err = getEsc(chunk[1:]); err != nil {
						return "", false, err
					}
				}
				match = match || lo <= r && r <= hi
				nrange++
			}
			failed = failed || match == negated

		case '?':
			if !failed {
				failed = s[0] == Separator
				_, n := utf8.DecodeRuneInString(s)
				s = s[n:]
			}
			chunk = chunk[1:]

		case '\\':
			if true /* the surface holds no Windows target */ {
				chunk = chunk[1:]
				if len(chunk) == 0 {
					return "", false, ErrBadPattern
				}
			}
			fallthrough

		default:
			if !failed {
				failed = chunk[0] != s[0]
				s = s[1:]
			}
			chunk = chunk[1:]
		}
	}
	if failed {
		return "", false, nil
	}
	return s, true, nil
}

// getEsc gets a possibly-escaped character from chunk, for a character class.
func getEsc(chunk string) (r rune, nchunk string, err error) {
	if len(chunk) == 0 || chunk[0] == '-' || chunk[0] == ']' {
		err = ErrBadPattern
		return
	}
	if chunk[0] == '\\' && true /* the surface holds no Windows target */ {
		chunk = chunk[1:]
		if len(chunk) == 0 {
			err = ErrBadPattern
			return
		}
	}
	r, n := utf8.DecodeRuneInString(chunk)
	if r == utf8.RuneError && n == 1 {
		err = ErrBadPattern
	}
	nchunk = chunk[n:]
	if len(nchunk) == 0 {
		err = ErrBadPattern
	}
	return
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

// SkipDir tells a walk to leave the directory that holds the current entry.
var SkipDir = fs.SkipDir

// SkipAll tells a walk to stop with no error.
var SkipAll = fs.SkipAll

// WalkFunc runs for each file and each directory of a walk of Walk.
type WalkFunc func(path string, info fs.FileInfo, err error) error

// lstat reads the metadata of a path and follows no symbolic link. A test of
// Go swaps it, thus each walk reads it and not os.Lstat.
var lstat = os.Lstat

// WalkDir walks the tree that starts at root, and it runs fn for each file and
// each directory. It reads no symbolic link.
func WalkDir(root string, fn fs.WalkDirFunc) error {
	info, err := lstat(root)
	if err != nil {
		err = fn(root, nil, err)
	} else {
		err = walkDir(root, fs.FileInfoToDirEntry(info), fn)
	}
	if err == SkipDir || err == SkipAll {
		return nil
	}
	return err
}

func walkDir(name string, d fs.DirEntry, fn fs.WalkDirFunc) error {
	if err := fn(name, d, nil); err != nil || !d.IsDir() {
		if err == SkipDir && d.IsDir() {
			// The walk leaves the directory, thus the caller of this call
			// continues.
			return nil
		}
		return err
	}
	entries, err := os.ReadDir(name)
	if err != nil {
		// The read of the directory failed after the first call of fn, thus
		// the second call names the cause.
		return fn(name, d, err)
	}
	for _, e := range entries {
		if err := walkDir(Join(name, e.Name()), e, fn); err != nil {
			if err == SkipDir {
				break
			}
			return err
		}
	}
	return nil
}

// Walk walks the tree that starts at root, and it runs fn for each file and
// each directory. It reads the metadata of each entry, thus WalkDir is faster.
func Walk(root string, fn WalkFunc) error {
	info, err := lstat(root)
	if err != nil {
		err = fn(root, nil, err)
	} else {
		err = walk(root, info, fn)
	}
	if err == SkipDir || err == SkipAll {
		return nil
	}
	return err
}

func walk(name string, info fs.FileInfo, fn WalkFunc) error {
	if !info.IsDir() {
		return fn(name, info, nil)
	}
	err := fn(name, info, nil)
	if err != nil {
		if err == SkipDir {
			return nil
		}
		return err
	}
	entries, err := os.ReadDir(name)
	if err != nil {
		return fn(name, info, err)
	}
	for _, e := range entries {
		child := Join(name, e.Name())
		childInfo, err := e.Info()
		if err != nil {
			if err := fn(child, nil, err); err != nil && err != SkipDir {
				return err
			}
			continue
		}
		if err := walk(child, childInfo, fn); err != nil {
			if err == SkipDir {
				break
			}
			return err
		}
	}
	return nil
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

// EvalSymlinks gives the path after each symbolic link. The path must exist.
//
// Go walks each element itself, and the result of a relative path stays
// relative. The Dart side gives the resolved path from the root, thus a
// relative argument takes Rel against the resolved working directory.
func EvalSymlinks(path string) (string, error) {
	resolved, cause := dart.ResolveSymlinks(path)
	if cause != "" {
		return "", &fs.PathError{Op: "lstat", Path: path, Err: fs.ErrNotExist}
	}
	if IsAbs(path) {
		return resolved, nil
	}
	wd, err := os.Getwd()
	if err != nil {
		return resolved, nil
	}
	wdResolved, cause := dart.ResolveSymlinks(wd)
	if cause != "" {
		return resolved, nil
	}
	rel, err := Rel(wdResolved, resolved)
	if err != nil {
		return resolved, nil
	}
	return rel, nil
}

// errInvalidPath names an argument that Localize rejects.
var errInvalidPath = errors.New("invalid path")

// Localize gives the path of the system for a path with `/` between 2
// elements. The argument must hold the form that fs.ValidPath accepts.
func Localize(path string) (string, error) {
	if !fs.ValidPath(path) {
		return "", errInvalidPath
	}
	for i := 0; i < len(path); i++ {
		if path[i] == 0 {
			return "", errInvalidPath
		}
	}
	return path, nil
}

// HasPrefix reports whether the path starts with the prefix. Go holds the
// function for old code only, and it respects no element boundary.
//
// Deprecated: HasPrefix does not respect path boundaries and
// does not ignore case when required.
func HasPrefix(p, prefix string) bool {
	return strings.HasPrefix(p, prefix)
}
