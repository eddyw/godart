package main

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
)

func main() {
	dir, err := os.MkdirTemp("", "dirfs")
	if err != nil {
		fmt.Println("temp:", err)
		return
	}
	defer os.RemoveAll(dir)

	must := func(err error) {
		if err != nil {
			fmt.Println("setup:", err)
		}
	}
	must(os.MkdirAll(filepath.Join(dir, "sub", "deep"), 0o755))
	must(os.WriteFile(filepath.Join(dir, "a.txt"), []byte("alpha"), 0o644))
	must(os.WriteFile(filepath.Join(dir, "b.md"), []byte("beta"), 0o644))
	must(os.WriteFile(filepath.Join(dir, "sub", "c.txt"), []byte("gamma"), 0o644))
	must(os.WriteFile(filepath.Join(dir, "sub", "deep", "d.txt"), []byte("delta"), 0o644))

	fsys := os.DirFS(dir)

	// fs.ReadFile through the ReadFileFS method.
	b, err := fs.ReadFile(fsys, "a.txt")
	fmt.Println(string(b), err)

	// fs.Stat through the StatFS method.
	info, err := fs.Stat(fsys, "sub")
	fmt.Println(info.Name(), info.IsDir(), err)

	// fs.ReadDir sorts each entry.
	entries, err := fs.ReadDir(fsys, ".")
	fmt.Println(err)
	for _, e := range entries {
		fmt.Println(" ", e.Name(), e.IsDir(), e.Type().IsRegular())
	}

	// fs.WalkDir walks the whole tree.
	var seen []string
	err = fs.WalkDir(fsys, ".", func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		seen = append(seen, p)
		return nil
	})
	sort.Strings(seen)
	fmt.Println(err, seen)

	// fs.Glob reads a pattern.
	hits, err := fs.Glob(fsys, "*.txt")
	fmt.Println(hits, err)

	// fs.Sub gives the tree under a name.
	sub, err := fs.Sub(fsys, "sub")
	fmt.Println(err)
	b, err = fs.ReadFile(sub, "c.txt")
	fmt.Println(string(b), err)

	// fs.Open gives a file that reads and closes.
	f, err := fsys.Open("b.md")
	fmt.Println(err)
	st, err := f.Stat()
	fmt.Println(st.Name(), st.Size(), err)
	buf := make([]byte, 4)
	n, err := f.Read(buf)
	fmt.Println(n, string(buf[:n]), err)
	fmt.Println(f.Close())

	// A name outside the tree gives an error.
	_, err = fs.ReadFile(fsys, "../outside")
	fmt.Println(err != nil)
	_, err = fs.ReadFile(fsys, "nothere.txt")
	fmt.Println(err != nil)

	// A symbolic link and its target.
	must(os.Symlink("a.txt", filepath.Join(dir, "link")))
	target, err := os.Readlink(filepath.Join(dir, "link"))
	fmt.Println(target, err)
	_, err = os.Readlink(filepath.Join(dir, "a.txt"))
	fmt.Println(err != nil)
}
