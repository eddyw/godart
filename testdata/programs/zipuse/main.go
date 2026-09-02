package main

import (
	"archive/zip"
	"bytes"
	"fmt"
	"io"
)

func main() {
	var buf bytes.Buffer
	w := zip.NewWriter(&buf)
	for _, f := range []struct{ name, body string }{
		{"readme.txt", "the first file"},
		{"dir/second.txt", "the second file, which holds more text"},
	} {
		fw, err := w.Create(f.name)
		if err != nil {
			fmt.Println("create:", err)
			return
		}
		if _, err := io.WriteString(fw, f.body); err != nil {
			fmt.Println("write:", err)
			return
		}
	}
	if err := w.Close(); err != nil {
		fmt.Println("close:", err)
		return
	}

	r, err := zip.NewReader(bytes.NewReader(buf.Bytes()), int64(buf.Len()))
	if err != nil {
		fmt.Println("read:", err)
		return
	}
	for _, f := range r.File {
		rc, err := f.Open()
		if err != nil {
			fmt.Println("open:", err)
			return
		}
		body, err := io.ReadAll(rc)
		rc.Close()
		if err != nil {
			fmt.Println("readall:", err)
			return
		}
		fmt.Printf("%s method=%d size=%d body=%q\n", f.Name, f.Method, f.UncompressedSize64, body)
	}

	if _, err := zip.NewReader(bytes.NewReader([]byte("not a zip at all")), 16); err != nil {
		fmt.Println("a broken archive gives:", err)
	}
}
