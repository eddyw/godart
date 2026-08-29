package main

import (
	"fmt"
	"os"
	"path/filepath"
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

func main() {
	pure()
	relative()
	onDisk()
}
