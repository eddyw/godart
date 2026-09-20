package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"syscall"
	"time"
)

func main() {
	d, err := os.MkdirTemp("", "gart-path-probe-")
	if err != nil {
		panic(err)
	}
	defer os.RemoveAll(d)
	f := filepath.Join(d, "keep")
	if err := os.WriteFile(f, []byte("keep"), 0600); err != nil {
		panic(err)
	}
	p := f + "\x00"
	to := filepath.Join(d, "new")
	steps := []struct {
		name string
		run  func() error
	}{
		{"readdir", func() error { _, e := os.ReadDir(p); return e }},
		{"evalsymlinks", func() error { _, e := filepath.EvalSymlinks(p); return e }},
		{"read", func() error { _, e := os.ReadFile(p); return e }},
		{"write", func() error { return os.WriteFile(p, []byte("bad"), 0600) }},
		{"stat", func() error { _, e := os.Stat(p); return e }},
		{"lstat", func() error { _, e := os.Lstat(p); return e }},
		{"remove", func() error { return os.Remove(p) }},
		{"mkdir", func() error { return os.Mkdir(p, 0700) }},
		{"mkdirall", func() error { return os.MkdirAll(p, 0700) }},
		{"chmod", func() error { return os.Chmod(p, 0600) }},
		{"chown", func() error { return os.Chown(p, -1, -1) }},
		{"lchown", func() error { return os.Lchown(p, -1, -1) }},
		{"chtimes", func() error { return os.Chtimes(p, time.Time{}, time.Time{}) }},
		{"truncate", func() error { return os.Truncate(p, 0) }},
		{"chdir", func() error { return os.Chdir(p) }},
		{"readlink", func() error { _, e := os.Readlink(p); return e }},
		{"link-old", func() error { return os.Link(p, to) }},
		{"link-new", func() error { return os.Link(f, to+"\x00") }},
		{"symlink-old", func() error { return os.Symlink(p, to) }},
		{"symlink-new", func() error { return os.Symlink(f, to+"\x00") }},
		{"rename-old", func() error { return os.Rename(p, to) }},
		{"rename-new", func() error { return os.Rename(f, to+"\x00") }},
	}
	for _, s := range steps {
		e := s.run()
		var pe *os.PathError
		var le *os.LinkError
		if errors.As(e, &pe) {
			fmt.Println(s.name, errors.Is(e, syscall.EINVAL), pe.Op, pe.Path == p)
		} else if errors.As(e, &le) {
			fmt.Println(s.name, errors.Is(e, syscall.EINVAL), le.Op, le.Old == p || le.Old == f, le.New == to || le.New == to+"\x00")
		} else {
			fmt.Println(s.name, "unexpected error", e)
		}
	}
	b, e := os.ReadFile(f)
	fmt.Println("retained", string(b), e)
	names, e := os.ReadDir(d)
	if e != nil {
		panic(e)
	}
	fmt.Println("entries", len(names), names[0].Name())
}
