// The witness of the 2 pattern sentinels, of the index of a short buffer of
// utf8.EncodeRune, and of a failure of the writer of text/tabwriter.
package main

import (
	"errors"
	"fmt"
	"path"
	"path/filepath"
	"text/tabwriter"
	"unicode/utf8"
)

type failWriter struct{}

func (failWriter) Write(p []byte) (int, error) { return 0, errors.New("the writer failed") }

func try(name string, f func()) {
	defer func() {
		if e := recover(); e != nil {
			fmt.Printf("%s: panic %v\n", name, e)
			return
		}
		fmt.Printf("%s: no panic\n", name)
	}()
	f()
}

func main() {
	fmt.Println("same value", path.ErrBadPattern == filepath.ErrBadPattern)
	_, perr := path.Match("[", "a")
	_, ferr := filepath.Match("[", "a")
	fmt.Println("path is filepath", errors.Is(perr, filepath.ErrBadPattern))
	fmt.Println("filepath is path", errors.Is(ferr, path.ErrBadPattern))
	fmt.Println("path is path", errors.Is(perr, path.ErrBadPattern))

	try("encode", func() { utf8.EncodeRune(make([]byte, 1), '日') })
	try("encode 2", func() { utf8.EncodeRune(make([]byte, 2), '日') })
	try("append", func() { _ = utf8.AppendRune(nil, '日') })
	b := make([]byte, 4)
	fmt.Println("encode ok", utf8.EncodeRune(b, '日'), b)

	w := tabwriter.NewWriter(failWriter{}, 1, 1, 1, ' ', 0)
	fmt.Fprintln(w, "a\tb")
	fmt.Println("flush", w.Flush())
}
