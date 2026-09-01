package main

import (
	"fmt"
	"io/fs"
)

func main() {
	fmt.Println(fs.ModeDir, fs.ModePerm, fs.FileMode(0o755))
	fmt.Println(fs.ValidPath("a/b"), fs.ValidPath("../a"))
	fmt.Println(fs.ErrNotExist)
}
