package main

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

func pure() {
	fmt.Println(filepath.Join("a", "b", "..", "c"), filepath.Join("/x", "y/"))
	fmt.Println(filepath.Clean("a//b/./c/.."), filepath.Clean(""), filepath.Clean("/../a"))
	d, f := filepath.Split("/a/b/c.txt")
	fmt.Println(d, f)
	fmt.Println(filepath.Base("/a/b/c.txt"), filepath.Dir("/a/b/c.txt"),
		filepath.Ext("/a/b/c.txt"), filepath.Ext("noext"))
	fmt.Println(filepath.IsAbs("/a"), filepath.IsAbs("a"),
		filepath.IsLocal("a/b"), filepath.IsLocal("../a"), filepath.IsLocal("/a"))
	fmt.Println(filepath.ToSlash("a/b"), filepath.FromSlash("a/b"),
		filepath.VolumeName("/a/b") == "")
	fmt.Println(filepath.SplitList("a:b:c"), len(filepath.SplitList("")))

	ok, err := filepath.Match("*.go", "main.go")
	fmt.Println(ok, err)
	ok, err = filepath.Match("a[", "a")
	fmt.Println(ok, err)
}

func relative() {
	cases := [][2]string{
		{"/a/b", "/a/b/c/d"},
		{"/a/b", "/a/c"},
		{"/a/b", "/a/b"},
		{"a/b", "a/b/c"},
		{"/a", "b"},
		{"../a", "b"},
	}
	for _, c := range cases {
		r, err := filepath.Rel(c[0], c[1])
		fmt.Printf("%s -> %s = %q %v\n", c[0], c[1], r, err)
	}
}

func onDisk() {
	dir, err := os.MkdirTemp("", "glob")
	if err != nil {
		fmt.Println("temp:", err)
		return
	}
	defer os.RemoveAll(dir)
	for _, n := range []string{"b.go", "a.go", "c.txt"} {
		if err := os.WriteFile(filepath.Join(dir, n), []byte("x"), 0o644); err != nil {
			fmt.Println("write:", err)
			return
		}
	}
	names, err := filepath.Glob(filepath.Join(dir, "*.go"))
	fmt.Println(err, len(names))
	for _, n := range names {
		fmt.Println(filepath.Base(n))
	}
	one, err := filepath.Glob(filepath.Join(dir, "a.go"))
	fmt.Println(len(one), err)
	none, err := filepath.Glob(filepath.Join(dir, "zz.go"))
	fmt.Println(len(none), err)

	abs, err := filepath.Abs(filepath.Join(dir, "a.go"))
	fmt.Println(strings.HasSuffix(abs, "a.go"), filepath.IsAbs(abs), err)
	rel, err := filepath.Abs("x/y")
	fmt.Println(filepath.IsAbs(rel), err)
}

func tree() {
	root, err := os.MkdirTemp("", "walk")
	if err != nil {
		fmt.Println("temp:", err)
		return
	}
	defer os.RemoveAll(root)
	for _, d := range []string{"a", "a/b", "c"} {
		if err := os.MkdirAll(filepath.Join(root, d), 0o755); err != nil {
			fmt.Println("mkdir:", err)
			return
		}
	}
	for _, f := range []string{"top.txt", "a/one.go", "a/b/two.go", "c/three.txt"} {
		if err := os.WriteFile(filepath.Join(root, f), []byte("x"), 0o644); err != nil {
			fmt.Println("write:", err)
			return
		}
	}

	var seen []string
	err = filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(root, p)
		seen = append(seen, fmt.Sprintf("%s:%v:%v", rel, d.IsDir(), d.Type().IsDir()))
		return nil
	})
	sort.Strings(seen)
	fmt.Println(err, len(seen))
	for _, s := range seen {
		fmt.Println(" ", s)
	}

	var skipped []string
	err = filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() && filepath.Base(p) == "a" {
			return filepath.SkipDir
		}
		rel, _ := filepath.Rel(root, p)
		skipped = append(skipped, rel)
		return nil
	})
	sort.Strings(skipped)
	fmt.Println("skip dir:", err, skipped)

	count := 0
	err = filepath.Walk(root, func(p string, info fs.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			count++
		}
		if count == 2 {
			return filepath.SkipAll
		}
		return nil
	})
	fmt.Println("skip all:", err, count)

	entries, err := os.ReadDir(root)
	fmt.Println("read dir:", err, len(entries))
	names := []string{}
	for _, e := range entries {
		info, err := e.Info()
		if err != nil {
			fmt.Println("info:", err)
			return
		}
		names = append(names, fmt.Sprintf("%s:%v:%v", e.Name(), e.IsDir(), info.Size()))
	}
	sort.Strings(names)
	fmt.Println(names)

	info, err := os.Stat(filepath.Join(root, "top.txt"))
	fmt.Println(err, info.Name(), info.Size(), info.IsDir(), info.Mode().IsRegular())
	dinfo, err := os.Stat(filepath.Join(root, "a"))
	fmt.Println(err, dinfo.IsDir(), dinfo.Mode().IsDir(), dinfo.Mode().Perm() != 0)
}

func main() {
	pure()
	relative()
	onDisk()
	tree()
}
