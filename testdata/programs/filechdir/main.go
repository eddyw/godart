package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"syscall"
)

func main() {
	old, err := os.Getwd()
	if err != nil {
		panic(err)
	}
	root, err := os.MkdirTemp("", "gart-filechdir-")
	if err != nil {
		panic(err)
	}
	defer os.RemoveAll(root)
	defer os.Chdir(old)
	if err := os.Chdir(root); err != nil {
		panic(err)
	}
	if err := os.Mkdir("original", 0700); err != nil {
		panic(err)
	}
	if err := os.WriteFile("original/marker", []byte("original"), 0600); err != nil {
		panic(err)
	}
	dir, err := os.Open("original")
	if err != nil {
		panic(err)
	}
	defer dir.Close()
	if err := os.Rename("original", "moved"); err != nil {
		panic(err)
	}
	if err := os.Mkdir("original", 0700); err != nil {
		panic(err)
	}
	if err := os.WriteFile("original/marker", []byte("replacement"), 0600); err != nil {
		panic(err)
	}
	if err := os.Chdir(old); err != nil {
		panic(err)
	}
	if err := dir.Chdir(); err != nil {
		panic(err)
	}
	value, err := os.ReadFile("marker")
	if err != nil {
		panic(err)
	}
	if string(value) != "original" {
		panic("directory handle followed its old path")
	}
	actual, err := os.Getwd()
	if err != nil {
		panic(err)
	}
	want, err := filepath.EvalSymlinks(filepath.Join(root, "moved"))
	if err != nil {
		panic(err)
	}
	if actual != want {
		panic("current directory does not match the open descriptor")
	}
	if err := os.Setenv("PWD", filepath.Join(root, "original")); err != nil {
		panic(err)
	}
	if wd, err := os.Getwd(); err != nil || wd != want {
		panic("Getwd accepted a different directory")
	}
	alias := filepath.Join(root, "alias")
	if err := os.Symlink(want, alias); err != nil {
		panic(err)
	}
	if err := os.Setenv("PWD", alias); err != nil {
		panic(err)
	}
	if wd, err := os.Getwd(); err != nil || wd != alias {
		panic("Getwd lost the valid PWD alias")
	}
	file, err := os.Open("marker")
	if err != nil {
		panic(err)
	}
	if err := file.Chdir(); !errors.Is(err, syscall.ENOTDIR) {
		panic(fmt.Sprint("regular file: ", err))
	}
	file.Close()
	if err := file.Chdir(); !errors.Is(err, os.ErrClosed) {
		panic(fmt.Sprint("closed file: ", err))
	}
	var absent *os.File
	if err := absent.Chdir(); !errors.Is(err, os.ErrInvalid) {
		panic(fmt.Sprint("nil file: ", err))
	}
	if err := os.Stdout.Chdir(); !errors.Is(err, syscall.ENOTDIR) {
		panic(fmt.Sprint("standard stream: ", err))
	}
	fmt.Println("directory identity and errors match")
}
