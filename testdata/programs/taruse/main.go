package main

import (
	"archive/tar"
	"bytes"
	"fmt"
	"io"
)

func main() {
	var buf bytes.Buffer
	w := tar.NewWriter(&buf)
	for _, f := range []struct {
		name string
		body string
		mode int64
	}{
		{"readme.txt", "the first file", 0o644},
		{"dir/second.txt", "the second file, which holds more text", 0o600},
	} {
		h := &tar.Header{
			Name:     f.name,
			Mode:     f.mode,
			Size:     int64(len(f.body)),
			Typeflag: tar.TypeReg,
			Format:   tar.FormatPAX,
		}
		if err := w.WriteHeader(h); err != nil {
			fmt.Println("header:", err)
			return
		}
		if _, err := io.WriteString(w, f.body); err != nil {
			fmt.Println("write:", err)
			return
		}
	}
	if err := w.Close(); err != nil {
		fmt.Println("close:", err)
		return
	}

	r := tar.NewReader(bytes.NewReader(buf.Bytes()))
	for {
		h, err := r.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			fmt.Println("next:", err)
			return
		}
		body, err := io.ReadAll(r)
		if err != nil {
			fmt.Println("readall:", err)
			return
		}
		fmt.Printf("%s mode=%o size=%d body=%q\n", h.Name, h.Mode, h.Size, body)
	}
}
