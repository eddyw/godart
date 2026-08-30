package main

import (
	"bytes"
	"compress/flate"
	"compress/gzip"
	"fmt"
	"io"
	"strings"
)

func main() {
	text := strings.Repeat("the quick brown fox jumps over the lazy dog. ", 40)

	var buf bytes.Buffer
	w := gzip.NewWriter(&buf)
	w.Name = "text.txt"
	w.Comment = "a note"
	if _, err := io.WriteString(w, text); err != nil {
		fmt.Println("write:", err)
		return
	}
	if err := w.Close(); err != nil {
		fmt.Println("close:", err)
		return
	}
	fmt.Println("smaller", buf.Len() < len(text))

	r, err := gzip.NewReader(bytes.NewReader(buf.Bytes()))
	if err != nil {
		fmt.Println("reader:", err)
		return
	}
	out, err := io.ReadAll(r)
	if err != nil {
		fmt.Println("read:", err)
		return
	}
	fmt.Println("equal", string(out) == text, r.Name, r.Comment)
	fmt.Println("close", r.Close())

	// Each level of flate gives a stream that the reader reads back.
	for _, level := range []int{flate.NoCompression, flate.BestSpeed, 5, flate.BestCompression, flate.HuffmanOnly} {
		var raw bytes.Buffer
		fw, err := flate.NewWriter(&raw, level)
		if err != nil {
			fmt.Println("level:", err)
			return
		}
		io.WriteString(fw, text)
		fw.Close()
		back, err := io.ReadAll(flate.NewReader(bytes.NewReader(raw.Bytes())))
		if err != nil {
			fmt.Println("flate read:", err)
			return
		}
		fmt.Println("level", level, len(raw.Bytes()), string(back) == text)
	}

	// A dictionary gives a smaller stream for a text that holds it.
	var dictBuf bytes.Buffer
	dict := []byte("the quick brown fox")
	dw, err := flate.NewWriterDict(&dictBuf, 6, dict)
	if err != nil {
		fmt.Println("dict:", err)
		return
	}
	io.WriteString(dw, text)
	dw.Close()
	dr := flate.NewReaderDict(bytes.NewReader(dictBuf.Bytes()), dict)
	back, err := io.ReadAll(dr)
	fmt.Println("dict", string(back) == text, err)

	// A stream that holds no valid header gives an error.
	_, err = gzip.NewReader(strings.NewReader("not a stream"))
	fmt.Println("bad", err)

	// The empty text passes through.
	var empty bytes.Buffer
	ew := gzip.NewWriter(&empty)
	ew.Close()
	er, err := gzip.NewReader(bytes.NewReader(empty.Bytes()))
	if err != nil {
		fmt.Println("empty:", err)
		return
	}
	none, err := io.ReadAll(er)
	fmt.Println("empty", len(none), err)
}
