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
	"slices"
	"strings"
	"syscall"
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
// Go swaps it, thus the walk of a child reads it and not os.Lstat.
var lstat = os.Lstat

// WalkDir walks the tree that starts at root, and it runs fn for each file and
// each directory. It reads no symbolic link.
//
// The body below holds the walk of the real path.go of Go. The order of the
// calls and the answer of each error rule thus agree with Go.
func WalkDir(root string, fn fs.WalkDirFunc) error {
	info, err := os.Lstat(root)
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
			// The walk left the directory, thus the caller of this call
			// continues.
			err = nil
		}
		return err
	}

	dirs, err := os.ReadDir(name)
	if err != nil {
		// The read of the directory failed after the first call of fn, thus
		// the second call names the cause.
		err = fn(name, d, err)
		if err != nil {
			if err == SkipDir && d.IsDir() {
				err = nil
			}
			return err
		}
	}

	for _, e := range dirs {
		child := Join(name, e.Name())
		if err := walkDir(child, e, fn); err != nil {
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
	info, err := os.Lstat(root)
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

	names, err := readDirNames(name)
	err1 := fn(name, info, err)
	// A read that failed holds the walk out of the directory, and an answer
	// of fn that is not nil asks the walk to leave the directory or to stop.
	// The walk gives the answer of fn back in each of the 2 cases, and the
	// caller reads SkipDir and SkipAll there.
	if err != nil || err1 != nil {
		return err1
	}

	for _, entry := range names {
		child := Join(name, entry)
		childInfo, err := lstat(child)
		if err != nil {
			if err := fn(child, childInfo, err); err != nil && err != SkipDir {
				return err
			}
		} else {
			err = walk(child, childInfo, fn)
			if err != nil {
				if !childInfo.IsDir() || err != SkipDir {
					return err
				}
			}
		}
	}
	return nil
}

// readDirNames gives the name of each entry of the directory, ordered.
func readDirNames(dirname string) ([]string, error) {
	f, err := os.Open(dirname)
	if err != nil {
		return nil, err
	}
	names, err := f.Readdirnames(-1)
	f.Close()
	if err != nil {
		return nil, err
	}
	slices.Sort(names)
	return names, nil
}

// Glob gives each path that fits the pattern. It reports no error for a path
// that it cannot read.
func Glob(pattern string) ([]string, error) {
	return globWithLimit(pattern)
}

// pathSeparatorsLimit bounds the count of the elements of a pattern. Go holds
// the same bound, and CVE-2022-30632 gave it: a pattern of 4 million
// separators filled the stack before the bound stood there.
const pathSeparatorsLimit = 10000

// globWithLimit walks the elements of the pattern from the last one to the
// first one, and it then reads the directories from the first one to the last
// one. The real source of Go calls itself for each element, and a chain of
// 10,000 calls fills the fixed stack of the Dart VM. The list files below
// holds the same state as that chain.
func globWithLimit(pattern string) ([]string, error) {
	// files holds the last element of each level, from the last level to the
	// first one.
	var files []string
	var matches []string
	p := pattern
	for {
		if len(files) == pathSeparatorsLimit {
			return nil, ErrBadPattern
		}
		if _, err := Match(p, ""); err != nil {
			return nil, err
		}
		if !hasMeta(p) {
			// Go reads Lstat here, thus a symbolic link with no target still
			// fits a pattern that holds no meta character.
			if _, err := os.Lstat(p); err == nil {
				matches = []string{p}
			}
			break
		}
		dir, file := Split(p)
		dir = cleanGlobPath(dir)
		if !hasMeta(dir) {
			m, err := glob(dir, file, nil)
			if err != nil {
				return nil, err
			}
			matches = m
			break
		}
		if dir == p {
			return nil, ErrBadPattern
		}
		files = append(files, file)
		p = dir
	}
	for i := len(files) - 1; i >= 0; i-- {
		var next []string
		for _, d := range matches {
			var err error
			next, err = glob(d, files[i], next)
			if err != nil {
				return nil, err
			}
		}
		matches = next
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
// The body holds the walk of the real symlink.go of Go, with the parts of
// Windows and of Plan 9 out. The separator is always `/` here, thus the volume
// name is empty and the special case of `.` under Windows stays out.
//
// A walk of the elements gives the answers of Go that 1 call of the resolver
// of the platform cannot give: the error of a missing element names the
// element and not the whole path, a non directory that another element follows
// gives syscall.ENOTDIR, a chain above 255 links gives its own error, and the
// result of a relative path stays relative.
func EvalSymlinks(path string) (string, error) {
	volLen := 0
	if len(path) > 0 && path[0] == Separator {
		volLen = 1
	}
	vol := path[:volLen]
	dest := vol
	linksWalked := 0
	for start, end := volLen, volLen; start < len(path); start = end {
		for start < len(path) && path[start] == Separator {
			start++
		}
		end = start
		for end < len(path) && path[end] != Separator {
			end++
		}

		// The next element of the path stands in path[start:end].
		if end == start {
			// No element stays.
			break
		} else if path[start:end] == "." {
			// The element `.` names the same directory.
			continue
		} else if path[start:end] == ".." {
			// Go back 1 element when the result holds one.
			r := len(dest) - 1
			for ; r >= volLen; r-- {
				if dest[r] == Separator {
					break
				}
			}
			if r < volLen || dest[r+1:] == ".." {
				// The result holds no separator, or it ends with a `..` that
				// the walk had to keep. The new `..` joins it.
				if len(dest) > volLen {
					dest += string(Separator)
				}
				dest += ".."
			} else {
				dest = dest[:r]
			}
			continue
		}

		// An ordinary element joins the result.
		if len(dest) > volLen && dest[len(dest)-1] != Separator {
			dest += string(Separator)
		}
		dest += path[start:end]

		fi, err := os.Lstat(dest)
		if err != nil {
			return "", err
		}
		if fi.Mode()&fs.ModeSymlink == 0 {
			if !fi.Mode().IsDir() && end < len(path) {
				return "", syscall.ENOTDIR
			}
			continue
		}

		// The element is a symbolic link.
		linksWalked++
		if linksWalked > 255 {
			return "", errors.New("EvalSymlinks: too many links")
		}
		link, err := os.Readlink(dest)
		if err != nil {
			return "", err
		}
		path = link + path[end:]
		if len(link) > 0 && link[0] == Separator {
			// The link names an absolute path.
			dest = link[:1]
			end = 1
			vol = link[:1]
			volLen = 1
		} else {
			// The link names a relative path, thus it replaces the last
			// element of the result.
			r := len(dest) - 1
			for ; r >= volLen; r-- {
				if dest[r] == Separator {
					break
				}
			}
			if r < volLen {
				dest = vol
			} else {
				dest = dest[:r]
			}
			end = 0
		}
	}
	return Clean(dest), nil
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
