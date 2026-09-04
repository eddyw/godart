package main

import (
	"fmt"
	"io"
	"io/fs"
	"testing/fstest"
	"time"
)

func main() {
	when := time.Date(2026, 9, 4, 12, 0, 0, 0, time.UTC)
	fsys := fstest.MapFS{
		"greet.txt":     &fstest.MapFile{Data: []byte("hello"), Mode: 0o644, ModTime: when},
		"sub/inner.txt": &fstest.MapFile{Data: []byte("inner"), Mode: 0o644, ModTime: when},
		"sub/deep/x.md": &fstest.MapFile{Data: []byte("# x"), Mode: 0o644, ModTime: when},
	}

	data, err := fs.ReadFile(fsys, "greet.txt")
	fmt.Println("read:", string(data), err == nil)

	entries, err := fs.ReadDir(fsys, ".")
	fmt.Println("root:", err == nil)
	for _, e := range entries {
		fmt.Println("  entry:", e.Name(), e.IsDir())
	}

	info, err := fs.Stat(fsys, "sub/inner.txt")
	fmt.Println("stat:", info.Name(), info.Size(), info.Mode(), info.ModTime().UTC(), err == nil)

	matches, err := fs.Glob(fsys, "sub/*")
	fmt.Println("glob:", matches, err == nil)

	var walked []string
	fs.WalkDir(fsys, ".", func(path string, d fs.DirEntry, err error) error {
		walked = append(walked, path)
		return nil
	})
	fmt.Println("walk:", walked)

	sub, err := fs.Sub(fsys, "sub")
	fmt.Println("sub:", err == nil)
	inner, err := fs.ReadFile(sub, "inner.txt")
	fmt.Println("inner:", string(inner), err == nil)

	f, err := fsys.Open("greet.txt")
	fmt.Println("open:", err == nil)
	all, _ := io.ReadAll(f)
	fmt.Println("all:", string(all))
	f.Close()

	if err := fstest.TestFS(fsys, "greet.txt", "sub/inner.txt", "sub/deep/x.md"); err != nil {
		fmt.Println("TestFS:", err)
	} else {
		fmt.Println("TestFS: ok")
	}

	_, err = fsys.Open("absent.txt")
	fmt.Println("absent:", err != nil)
}
