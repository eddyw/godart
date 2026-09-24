package main

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"sort"
)

func main() {
	root, err := os.MkdirTemp("", "directory-metadata-")
	if err != nil {
		panic(err)
	}
	defer os.RemoveAll(root)
	gone := filepath.Join(root, "gone")
	if err := os.Mkdir(gone, 0700); err != nil {
		panic(err)
	}
	for _, name := range []string{"a", "b", "c"} {
		if err := os.WriteFile(filepath.Join(gone, name), nil, 0600); err != nil {
			panic(err)
		}
	}
	g, err := os.Open(gone)
	if err != nil {
		panic(err)
	}
	defer g.Close()
	if _, err := g.Readdirnames(1); err != nil {
		panic(err)
	}
	for _, name := range []string{"a", "b", "c"} {
		if err := os.Remove(filepath.Join(gone, name)); err != nil {
			panic(err)
		}
	}
	removed, err := g.Readdir(-1)
	fmt.Println("removed-buffered-entries", len(removed), err == nil)
	before := filepath.Join(root, "before")
	if err := os.Mkdir(before, 0700); err != nil {
		panic(err)
	}
	if err := os.Mkdir(filepath.Join(before, "child"), 0700); err != nil {
		panic(err)
	}
	if err := os.WriteFile(filepath.Join(before, "file"), []byte("a"), 0600); err != nil {
		panic(err)
	}
	if err := os.Symlink("file", filepath.Join(before, "link")); err != nil {
		panic(err)
	}
	f, err := os.Open(before)
	if err != nil {
		panic(err)
	}
	defer f.Close()
	entries, err := f.ReadDir(-1)
	if err != nil {
		panic(err)
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].Name() < entries[j].Name() })
	after := filepath.Join(root, "after")
	if err := os.Rename(before, after); err != nil {
		panic(err)
	}
	for _, entry := range entries {
		_, err := entry.Info()
		fmt.Println("type-after-rename", entry.Name(), entry.IsDir(), entry.Type(), err != nil)
	}
	if _, err := f.Seek(0, 0); err != nil {
		panic(err)
	}
	infos, err := f.Readdir(-1)
	fmt.Println("info-after-rename", len(infos), err == nil)
	sort.Slice(infos, func(i, j int) bool { return infos[i].Name() < infos[j].Name() })
	for _, info := range infos {
		actual, err := os.Lstat(filepath.Join(after, info.Name()))
		if err != nil {
			panic(err)
		}
		fmt.Println("info", info.Name(), info.Mode().Type(), os.SameFile(info, actual), reflect.DeepEqual(info.Sys(), actual.Sys()))
	}
	entries, err = os.ReadDir(after)
	if err != nil {
		panic(err)
	}
	for _, entry := range entries {
		if entry.Name() != "file" {
			continue
		}
		first, err := entry.Info()
		if err != nil {
			panic(err)
		}
		if err := os.WriteFile(filepath.Join(after, "file"), []byte("changed"), 0600); err != nil {
			panic(err)
		}
		second, err := entry.Info()
		if err != nil {
			panic(err)
		}
		fmt.Println("fresh-info", first.Size(), second.Size())
	}

}
