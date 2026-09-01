package main

import (
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
)

func main() {
	base, err := os.MkdirTemp("", "root")
	if err != nil {
		fmt.Println("temp:", err)
		return
	}
	defer os.RemoveAll(base)

	dir := filepath.Join(base, "tree")
	if err := os.MkdirAll(filepath.Join(dir, "sub"), 0o755); err != nil {
		fmt.Println("mkdir:", err)
		return
	}
	if err := os.WriteFile(filepath.Join(base, "secret.txt"), []byte("no"), 0o644); err != nil {
		fmt.Println("write:", err)
		return
	}
	if err := os.WriteFile(filepath.Join(dir, "in.txt"), []byte("yes"), 0o644); err != nil {
		fmt.Println("write:", err)
		return
	}

	r, err := os.OpenRoot(dir)
	if err != nil {
		fmt.Println("openroot:", err)
		return
	}
	fmt.Println("name ok:", r.Name() == dir)

	// A name inside the root works.
	b, err := r.ReadFile("in.txt")
	fmt.Println(string(b), err)
	fmt.Println(r.WriteFile("out.txt", []byte("written"), 0o644))
	b, err = r.ReadFile("out.txt")
	fmt.Println(string(b), err)

	info, err := r.Stat("sub")
	fmt.Println(info.IsDir(), err)
	fmt.Println(r.Mkdir("fresh", 0o755))
	fmt.Println(r.Remove("fresh"))

	f, err := r.Open("in.txt")
	fmt.Println(err)
	text, _ := io.ReadAll(f)
	fmt.Println(string(text), f.Close())

	// A name that leaves the root gives an error.
	_, err = r.ReadFile("../secret.txt")
	fmt.Println("dotdot:", err != nil)
	_, err = r.Open("../secret.txt")
	fmt.Println("dotdot open:", err != nil)

	// A symbolic link that leaves the root gives an error.
	if err := os.Symlink(filepath.Join(base, "secret.txt"), filepath.Join(dir, "escape")); err != nil {
		fmt.Println("symlink:", err)
		return
	}
	_, err = r.ReadFile("escape")
	fmt.Println("escape:", err != nil)

	// A link inside the root works.
	if err := os.Symlink("in.txt", filepath.Join(dir, "inside")); err != nil {
		fmt.Println("symlink:", err)
		return
	}
	b, err = r.ReadFile("inside")
	fmt.Println(string(b), err)

	// The FS of the root reads the tree.
	fsys := r.FS()
	b, err = fs.ReadFile(fsys, "in.txt")
	fmt.Println(string(b), err)
	_, err = fs.ReadFile(fsys, "../secret.txt")
	fmt.Println("fs dotdot:", err != nil)

	// OpenInRoot does the same in 1 call.
	f, err = os.OpenInRoot(dir, "in.txt")
	fmt.Println(err)
	f.Close()
	_, err = os.OpenInRoot(dir, "../secret.txt")
	fmt.Println("openinroot:", err != nil)

	// A closed root rejects each operation.
	fmt.Println(r.Close())
	_, err = r.ReadFile("in.txt")
	fmt.Println("closed:", err != nil)
}
