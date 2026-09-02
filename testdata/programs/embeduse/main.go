package main

import (
	"embed"
	"fmt"
	"io"
	"io/fs"
)

//go:embed static/hello.txt
var greeting string

//go:embed logo.bin
var logo []byte

//go:embed static
var assets embed.FS

//go:embed all:static
var everything embed.FS

// count reads the embedded text before any init of the package runs, thus the
// value must stand at the declaration.
var count = len(greeting)

func main() {
	fmt.Printf("%q %d\n", greeting, count)
	fmt.Println(len(logo), logo)

	fmt.Println("assets:", walk(assets))
	fmt.Println("everything:", walk(everything))

	text, err := assets.ReadFile("static/hello.txt")
	fmt.Printf("%q %v\n", text, err)

	_, err = assets.ReadFile("static/.hidden.txt")
	fmt.Println(err)

	_, err = everything.ReadFile("static/.hidden.txt")
	fmt.Println(err)

	entries, err := assets.ReadDir("static")
	fmt.Println(err)
	for _, e := range entries {
		info, err := e.Info()
		fmt.Println(e.Name(), e.IsDir(), e.Type(), info.Size(), err)
	}

	file, err := assets.Open("static/data/nested.txt")
	fmt.Println(err)
	body, err := io.ReadAll(file)
	fmt.Printf("%q %v\n", body, err)
	fmt.Println(file.Close())

	sub, err := fs.Sub(assets, "static")
	fmt.Println(err)
	text, err = fs.ReadFile(sub, "second.txt")
	fmt.Printf("%q %v\n", text, err)

	matches, err := fs.Glob(assets, "static/*.txt")
	fmt.Println(matches, err)

	_, err = assets.Open("static/missing.txt")
	fmt.Println(err)
}

// walk gives the name of each file and each directory of the file system.
func walk(f embed.FS) []string {
	var out []string
	err := fs.WalkDir(f, ".", func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			out = append(out, p+"/")
			return nil
		}
		out = append(out, p)
		return nil
	})
	if err != nil {
		return []string{err.Error()}
	}
	return out
}
