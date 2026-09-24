package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
)

func main() {
	dir, err := os.MkdirTemp("", "directory-cursor-")
	if err != nil {
		panic(err)
	}
	defer os.RemoveAll(dir)
	parent := filepath.Join(dir, "before")
	if err := os.Mkdir(parent, 0700); err != nil {
		panic(err)
	}
	for i := 0; i < 600; i++ {
		if err := os.WriteFile(filepath.Join(parent, fmt.Sprintf("entry-%04d", i)), nil, 0600); err != nil {
			panic(err)
		}
	}
	f, err := os.Open(parent)
	if err != nil {
		panic(err)
	}
	defer f.Close()
	names, err := f.Readdirnames(1)
	if err != nil {
		panic(err)
	}
	entries, err := f.ReadDir(2)
	if err != nil {
		panic(err)
	}
	for _, entry := range entries {
		names = append(names, entry.Name())
	}
	infos, err := f.Readdir(3)
	if err != nil {
		panic(err)
	}
	for _, info := range infos {
		names = append(names, info.Name())
	}
	moved := filepath.Join(dir, "after")
	if err := os.Rename(parent, moved); err != nil {
		panic(err)
	}
	for {
		next, err := f.Readdirnames(7)
		names = append(names, next...)
		if err == io.EOF {
			break
		}
		if err != nil {
			panic(err)
		}
	}
	sort.Strings(names)
	valid := len(names) == 600
	for i, name := range names {
		valid = valid && name == fmt.Sprintf("entry-%04d", i)
	}
	fmt.Println("mixed-and-renamed", len(names), valid)
	if _, err := f.Seek(0, 0); err != nil {
		panic(err)
	}
	names, err = f.Readdirnames(0)
	if err != nil {
		panic(err)
	}
	fmt.Println("rewind", len(names))
	names, err = f.Readdirnames(1)
	fmt.Println("eof", len(names), names != nil, err == io.EOF)
	infos, err = f.Readdir(1)
	fmt.Println("info-eof", len(infos), infos != nil, err == io.EOF)
	entries, err = f.ReadDir(1)
	fmt.Println("entry-eof", len(entries), entries != nil, err == io.EOF)
}
