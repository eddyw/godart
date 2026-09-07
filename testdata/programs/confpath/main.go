// The program proves the packages path and path/filepath against real Go:
// each function on a table of paths that holds an empty string, a dot, a
// double dot, and a trailing slash, Match with each pattern form, Rel, Glob
// and WalkDir over a temporary directory with a sorted output.
package main

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
)

func row(label string, value any) {
	fmt.Printf("%-30s %v\n", label, value)
}

var paths = []string{
	"", ".", "..", "/", "//", "a", "a/", "a/b", "a/b/", "/a", "/a/b", "/a/b/", "a//b", "a/./b", "a/../b", "./a", "../a",
	"a/..", "a/b/..", "a/b/../..", "a/b/../../..", "/..", "/../a", "a/.", "./", "../", "a.txt", "a.tar.gz", ".hidden",
	"dir/.hidden", "a/b.c/d", "/a/b.c", ".", "..a", "a..", "日本/語.txt", "a\\b", "a b/c d",
}

func table() {
	fmt.Printf("%-16s %-14s %-10s %-8s %-8s %-8s %-14s\n", "path", "Clean", "Base", "Dir", "Ext", "IsAbs", "Split")
	for _, p := range paths {
		d, f := path.Split(p)
		fmt.Printf("%-16q %-14q %-10q %-8q %-8q %-8v %q,%q\n", p, path.Clean(p), path.Base(p), path.Dir(p), path.Ext(p), path.IsAbs(p), d, f)
	}
	fmt.Println("-- filepath --")
	fmt.Printf("%-16s %-14s %-10s %-8s %-8s %-8s %-8s %-14s\n", "path", "Clean", "Base", "Dir", "Ext", "IsAbs", "IsLocal", "Split")
	for _, p := range paths {
		d, f := filepath.Split(p)
		fmt.Printf("%-16q %-14q %-10q %-8q %-8q %-8v %-8v %q,%q\n", p, filepath.Clean(p), filepath.Base(p), filepath.Dir(p), filepath.Ext(p), filepath.IsAbs(p), filepath.IsLocal(p), d, f)
	}
}

func joins() {
	cases := [][]string{
		{"a", "b"}, {"a/", "b"}, {"a", "/b"}, {"", "b"}, {"a", ""}, {"", ""}, {}, {"/", "a"}, {"a", "..", "b"},
		{"a", ".", "b"}, {"a//", "//b"}, {"..", "a"}, {"a", "b", "c", "d"}, {"/a", "../.."}, {"", "", "a"}, {"a/b", "../../.."},
	}
	for _, c := range cases {
		fmt.Printf("Join %-28q path=%-10q filepath=%q\n", c, path.Join(c...), filepath.Join(c...))
	}
	for _, p := range []string{"", ".", "a", "a/b", "/a", "a/b/", "日本/語"} {
		row(fmt.Sprintf("ToSlash %q", p), fmt.Sprintf("%q %q", filepath.ToSlash(p), filepath.FromSlash(p)))
	}
	for _, p := range []string{"", "/", "a", "/a/b", "C:\\x", "//host/share/x"} {
		row(fmt.Sprintf("VolumeName %q", p), fmt.Sprintf("%q", filepath.VolumeName(p)))
	}
	for _, p := range []string{"", "a", "a:b", "a:b:c", ":a", "a:", "::", "\"quoted\":b"} {
		row(fmt.Sprintf("SplitList %q", p), fmt.Sprintf("%q", filepath.SplitList(p)))
	}
	row("Separator", string(filepath.Separator))
	row("ListSeparator", string(filepath.ListSeparator))
	for _, p := range []string{"a", "a/b", "", ".", "..", "/a", "a/../b", "a/../../b", "日本"} {
		v, err := filepath.Localize(p)
		row(fmt.Sprintf("Localize %q", p), fmt.Sprintf("%q %v", v, err))
	}
}

func matches() {
	type mc struct{ pattern, name string }
	cases := []mc{
		{"abc", "abc"}, {"*", "abc"}, {"*c", "abc"}, {"a*", "a"}, {"a*", "abc"}, {"a*", "ab/c"}, {"a*/b", "abc/b"}, {"a*/b", "a/c/b"},
		{"a*b*c*d*e*/f", "axbxcxdxe/f"}, {"a*b*c*d*e*/f", "axbxcxdxexxx/f"}, {"a*b*c*d*e*/f", "axbxcxdxe/xxx/f"}, {"a*b?c*x", "abxbbxdbxebxczzx"},
		{"a*b?c*x", "abxbbxdbxebxczzy"}, {"ab[c]", "abc"}, {"ab[b-d]", "abc"}, {"ab[e-g]", "abc"}, {"ab[^c]", "abc"}, {"ab[^b-d]", "abc"},
		{"ab[^e-g]", "abc"}, {"a\\*b", "a*b"}, {"a\\*b", "ab"}, {"a?b", "a☺b"}, {"a[^a]b", "a☺b"}, {"a???b", "a☺b"}, {"a[^a][^a][^a]b", "a☺b"},
		{"[a-ζ]*", "α"}, {"*[a-ζ]", "A"}, {"a?b", "a/b"}, {"a*b", "a/b"}, {"[\\]a]", "]"}, {"[\\-]", "-"}, {"[x\\-]", "x"}, {"[x\\-]", "-"},
		{"[x\\-]", "z"}, {"[\\-x]", "x"}, {"[\\-x]", "-"}, {"[\\-x]", "a"}, {"[a-b-c]", "a"}, {"[", "a"}, {"[^", "a"}, {"[^bc", "a"}, {"a[", "a"},
		{"a[", "ab"}, {"a[", "x"}, {"a/b[", "x"}, {"*x", "xxx"}, {"", ""}, {"", "a"}, {"*", ""}, {"?", ""}, {"a\\", "a"}, {"[]a]", "]"},
		{"[]", "a"}, {"*.go", "main.go"}, {"*.go", "dir/main.go"}, {"*/*.go", "dir/main.go"}, {"**/*.go", "dir/main.go"}, {"[a-]", "-"},
	}
	for _, c := range cases {
		pm, perr := path.Match(c.pattern, c.name)
		fm, ferr := filepath.Match(c.pattern, c.name)
		fmt.Printf("Match %-18q %-20q path=%-5v %-24v filepath=%-5v %v\n", c.pattern, c.name, pm, perr, fm, ferr)
	}
	_, err := path.Match("[", "a")
	row("ErrBadPattern", errors.Is(err, path.ErrBadPattern))
	_, err = filepath.Match("[", "a")
	row("filepath ErrBadPattern", errors.Is(err, filepath.ErrBadPattern))
}

func rels() {
	cases := [][2]string{
		{"a/b", "a/b"}, {"a/b/.", "a/b"}, {"a/b", "a/b/."}, {"./a/b", "a/b"}, {"a/b", "./a/b"}, {"ab/cd", "ab/cde"}, {"ab/cd", "ab/c"},
		{"a/b", "a/b/c/d"}, {"a/b", "a/b/../c"}, {"a/b/../c", "a/b"}, {"a/b/c", "a/c/d"}, {"a/b", "c/d"}, {"a/b/c/d", "a/b"}, {"a/b/c/d", "a/b/"},
		{"a/b/c/d/", "a/b"}, {"a/b/c/d/", "a/b/"}, {"../../a/b", "../../a/b/c/d"}, {"/a/b", "/a/b"}, {"/a/b/.", "/a/b"}, {"/a/b", "/a/b/."},
		{"/ab/cd", "/ab/cde"}, {"/a/b", "/a/b/c/d"}, {"/a/b", "/a/b/../c"}, {"/a/b/../c", "/a/b"}, {"/a/b/c", "/a/c/d"}, {"/a/b", "/c/d"},
		{"/a/b/c/d", "/a/b"}, {"/../../a/b", "/../../a/b/c/d"}, {".", "a/b"}, {".", ".."}, {"..", "."}, {"..", "a"}, {"../..", ".."},
		{"a", "../a"}, {"a/b", "../a/b"}, {"/", "/a"}, {"/a", "/"}, {"a", "/a"}, {"/a", "a"}, {"", "a"}, {"a", ""}, {"", ""}, {"..", "a/b/.."},
	}
	for _, c := range cases {
		r, err := filepath.Rel(c[0], c[1])
		fmt.Printf("Rel %-14q %-16q %-10q %v\n", c[0], c[1], r, err)
	}
	abs, err := filepath.Abs("/already/abs")
	row("Abs of an absolute path", fmt.Sprintf("%q %v", abs, err))
	abs, err = filepath.Abs("/a/../b/./c/")
	row("Abs cleans", fmt.Sprintf("%q %v", abs, err))
	abs, err = filepath.Abs("relative")
	row("Abs of a relative path", fmt.Sprintf("%v %v", filepath.IsAbs(abs), err))
	row("Abs ends with the name", filepath.Base(abs))
	row("HasPrefix", filepath.HasPrefix("/a/b", "/a"))
	row("HasPrefix false", filepath.HasPrefix("/a/b", "/b"))
}

func walk() {
	dir, err := os.MkdirTemp("", "confpath-")
	if err != nil {
		row("MkdirTemp", err)
		return
	}
	defer os.RemoveAll(dir)
	files := []string{"a.txt", "b.go", "sub/c.txt", "sub/d.go", "sub/deep/e.txt", "sub/deep/f.md", "other/g.txt", ".hidden", "z"}
	for _, f := range files {
		full := filepath.Join(dir, f)
		os.MkdirAll(filepath.Dir(full), 0o755)
		os.WriteFile(full, []byte(f), 0o644)
	}
	os.MkdirAll(filepath.Join(dir, "empty"), 0o755)
	rel := func(list []string) []string {
		out := make([]string, 0, len(list))
		for _, p := range list {
			r, _ := filepath.Rel(dir, p)
			out = append(out, r)
		}
		sort.Strings(out)
		return out
	}
	for _, pattern := range []string{"*.txt", "*", "*/*.go", "sub/*/*", "*/*", "none*", ".*", "[a-b].*", "sub/deep/?.md", "[", "**"} {
		matches, err := filepath.Glob(filepath.Join(dir, pattern))
		row(fmt.Sprintf("Glob %q", pattern), fmt.Sprintf("%v %v", rel(matches), err))
	}
	matches, err := filepath.Glob(filepath.Join(dir, "missing", "*"))
	row("Glob in a missing dir", fmt.Sprintf("%v %v nil=%v", matches, err, matches == nil))
	matches, err = filepath.Glob(filepath.Join(dir, "a.txt"))
	row("Glob of a literal", fmt.Sprintf("%v %v", rel(matches), err))
	matches, err = filepath.Glob(filepath.Join(dir, "nothing.txt"))
	row("Glob of a missing literal", fmt.Sprintf("%v %v", matches, err))
	var walked []string
	err = filepath.WalkDir(dir, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		r, _ := filepath.Rel(dir, p)
		kind := "f"
		if d.IsDir() {
			kind = "d"
		}
		walked = append(walked, kind+":"+r)
		return nil
	})
	row("WalkDir err", err)
	for _, w := range walked {
		fmt.Println("WalkDir", w)
	}
	walked = nil
	err = filepath.WalkDir(dir, func(p string, d fs.DirEntry, err error) error {
		r, _ := filepath.Rel(dir, p)
		if d.IsDir() && d.Name() == "sub" {
			return filepath.SkipDir
		}
		if !d.IsDir() {
			walked = append(walked, r)
		}
		return nil
	})
	row("WalkDir SkipDir", fmt.Sprintf("%v %v", walked, err))
	walked = nil
	err = filepath.WalkDir(dir, func(p string, d fs.DirEntry, err error) error {
		r, _ := filepath.Rel(dir, p)
		walked = append(walked, r)
		if r == "b.go" {
			return filepath.SkipAll
		}
		return nil
	})
	row("WalkDir SkipAll", fmt.Sprintf("%v %v", walked, err))
	walked = nil
	err = filepath.Walk(dir, func(p string, info fs.FileInfo, err error) error {
		if err != nil {
			return err
		}
		r, _ := filepath.Rel(dir, p)
		if !info.IsDir() {
			walked = append(walked, fmt.Sprintf("%s=%d", r, info.Size()))
		}
		return nil
	})
	row("Walk", fmt.Sprintf("%v %v", walked, err))
	err = filepath.WalkDir(filepath.Join(dir, "missing"), func(p string, d fs.DirEntry, err error) error {
		r, _ := filepath.Rel(dir, p)
		walked = []string{r, fmt.Sprint(d == nil), strings.ReplaceAll(err.Error(), dir, "$DIR")}
		return err
	})
	row("WalkDir missing", fmt.Sprintf("%v %v", walked, strings.ReplaceAll(err.Error(), dir, "$DIR")))
	walked = nil
	err = filepath.WalkDir(filepath.Join(dir, "a.txt"), func(p string, d fs.DirEntry, err error) error {
		r, _ := filepath.Rel(dir, p)
		walked = append(walked, fmt.Sprintf("%s dir=%v err=%v", r, d.IsDir(), err))
		return nil
	})
	row("WalkDir of a file", fmt.Sprintf("%v %v", walked, err))
	errStop := errors.New("stop here")
	err = filepath.WalkDir(dir, func(p string, d fs.DirEntry, err error) error {
		if d.Name() == "d.go" {
			return errStop
		}
		return nil
	})
	row("WalkDir custom error", fmt.Sprintf("%v %v", err, err == errStop))
	// The temporary directory of macOS stands behind a symbolic link, thus
	// the base of the result is the only stable part.
	resolved, err := filepath.EvalSymlinks(dir)
	row("EvalSymlinks of the dir", fmt.Sprintf("%v %v", filepath.IsAbs(resolved), err))
	target, err := filepath.EvalSymlinks(filepath.Join(dir, "sub", "..", "a.txt"))
	r, _ := filepath.Rel(resolved, target)
	row("EvalSymlinks of a plain path", fmt.Sprintf("%v %v", r, err))
	_, err = filepath.EvalSymlinks(filepath.Join(dir, "missing"))
	row("EvalSymlinks missing", fmt.Sprintf("%v %v", errors.Is(err, fs.ErrNotExist), filepath.Base(err.(*fs.PathError).Path)))
}

func main() {
	fmt.Println("== the table ==")
	table()
	fmt.Println("== Join and the slash functions ==")
	joins()
	fmt.Println("== Match ==")
	matches()
	fmt.Println("== Rel and Abs ==")
	rels()
	fmt.Println("== Glob and WalkDir ==")
	walk()
}
