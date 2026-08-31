package main

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
)

func main() {
	links()
	localize()
	prefixes()
}

// links proves os.Symlink, os.Lstat, filepath.EvalSymlinks, and a walk over a
// tree that holds a link. The temporary directory differs on each run, thus the
// output holds relative paths only.
func links() {
	dir, err := os.MkdirTemp("", "symlink")
	if err != nil {
		fmt.Println("temp:", err)
		return
	}
	defer os.RemoveAll(dir)
	if err := os.WriteFile(filepath.Join(dir, "f.txt"), []byte("hello"), 0o644); err != nil {
		fmt.Println("write:", err)
		return
	}
	if err := os.Mkdir(filepath.Join(dir, "sub"), 0o755); err != nil {
		fmt.Println("mkdir:", err)
		return
	}
	if err := os.WriteFile(filepath.Join(dir, "sub", "inner.txt"), []byte("in"), 0o644); err != nil {
		fmt.Println("write:", err)
		return
	}
	if err := os.Symlink("f.txt", filepath.Join(dir, "link")); err != nil {
		fmt.Println("symlink:", err)
		return
	}
	if err := os.Symlink("sub", filepath.Join(dir, "dl")); err != nil {
		fmt.Println("symlink:", err)
		return
	}
	err = os.Symlink("f.txt", filepath.Join(dir, "link"))
	fmt.Println("again:", err != nil)

	li, err := os.Lstat(filepath.Join(dir, "link"))
	if err != nil {
		fmt.Println("lstat:", err)
		return
	}
	fmt.Println("lstat:", li.Mode()&fs.ModeSymlink != 0, li.IsDir(), li.Size())
	si, err := os.Stat(filepath.Join(dir, "link"))
	if err != nil {
		fmt.Println("stat:", err)
		return
	}
	fmt.Println("stat:", si.Mode()&fs.ModeSymlink != 0, si.IsDir(), si.Size())
	di, err := os.Lstat(filepath.Join(dir, "dl"))
	if err != nil {
		fmt.Println("lstat dl:", err)
		return
	}
	fmt.Println("lstat dl:", di.Mode()&fs.ModeSymlink != 0, di.IsDir())

	entries, err := os.ReadDir(dir)
	if err != nil {
		fmt.Println("readdir:", err)
		return
	}
	for _, e := range entries {
		fmt.Println("entry:", e.Name(), e.IsDir(), e.Type()&fs.ModeSymlink != 0)
	}

	root, err := filepath.EvalSymlinks(dir)
	if err != nil {
		fmt.Println("eval root:", err)
		return
	}
	show := func(p string) {
		out, err := filepath.EvalSymlinks(p)
		if err != nil {
			fmt.Println("eval failed:", err != nil)
			return
		}
		rel, err := filepath.Rel(root, out)
		fmt.Println("eval:", rel, err)
	}
	show(filepath.Join(dir, "link"))
	show(filepath.Join(dir, "dl"))
	show(filepath.Join(dir, "dl", "inner.txt"))

	wd, err := os.Getwd()
	if err != nil {
		fmt.Println("wd:", err)
		return
	}
	defer os.Chdir(wd)
	if err := os.Chdir(dir); err != nil {
		fmt.Println("chdir:", err)
		return
	}
	rel := func(p string) {
		out, err := filepath.EvalSymlinks(p)
		fmt.Println("rel:", out, err)
	}
	rel("link")
	rel("dl")
	rel("sub/../link")
	_, err = filepath.EvalSymlinks("nope")
	fmt.Println("missing:", err != nil)

	walkErr := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, relErr := filepath.Rel(dir, path)
		if relErr != nil {
			return relErr
		}
		fmt.Println("walk:", rel, d.IsDir(), d.Type()&fs.ModeSymlink != 0)
		return nil
	})
	fmt.Println("walked:", walkErr)
}

func localize() {
	for _, p := range []string{"a/b", "a", "a/b/../c", "/a", "", "./a"} {
		out, err := filepath.Localize(p)
		fmt.Printf("localize %q = %q %v\n", p, out, err)
	}
}

func prefixes() {
	fmt.Println(filepath.HasPrefix("/a/b", "/a"),
		filepath.HasPrefix("/ab", "/a/"),
		filepath.HasPrefix("x", ""))
}
