package main

import (
	"fmt"
	"os"
	"path/filepath"
)

func cause(err error) string {
	if err == nil {
		return "ok"
	}
	if path, ok := err.(*os.PathError); ok {
		return path.Err.Error()
	}
	return err.Error()
}

func main() {
	dir, err := os.MkdirTemp("", "gart-openflags-")
	if err != nil {
		panic(err)
	}
	defer os.RemoveAll(dir)
	index := 0
	for _, existing := range []bool{false, true} {
		for _, access := range []int{os.O_RDONLY, os.O_WRONLY, os.O_RDWR} {
			for bits := 0; bits < 32; bits++ {
				flags := access
				for i, flag := range []int{os.O_CREATE, os.O_EXCL, os.O_TRUNC, os.O_APPEND, os.O_SYNC} {
					if bits&(1<<i) != 0 {
						flags |= flag
					}
				}
				name := filepath.Join(dir, fmt.Sprint(index))
				if existing {
					if err := os.WriteFile(name, []byte("old"), 0600); err != nil {
						panic(err)
					}
				}
				file, err := os.OpenFile(name, flags, 0600)
				fmt.Printf("%d open=%s", index, cause(err))
				index++
				if err != nil {
					fmt.Println()
					continue
				}
				n, err := file.Write(nil)
				fmt.Printf(" emptywrite=%d,%s", n, cause(err))
				n, err = file.WriteAt(nil, 1)
				fmt.Printf(" emptyat=%d,%s", n, cause(err))
				bytes := make([]byte, 8)
				n, err = file.Read(bytes)
				fmt.Printf(" read=%d,%q,%s", n, bytes[:n], cause(err))
				offset, err := file.Seek(0, 0)
				fmt.Printf(" seek=%d,%s", offset, cause(err))
				n, err = file.Write([]byte("new"))
				fmt.Printf(" write=%d,%s", n, cause(err))
				n, err = file.WriteAt([]byte("AT"), 1)
				fmt.Printf(" writeat=%d,%s", n, cause(err))
				fmt.Printf(" close=%s", cause(file.Close()))
				data, err := os.ReadFile(name)
				fmt.Printf(" data=%q,%s\n", data, cause(err))
			}
		}
	}
}
