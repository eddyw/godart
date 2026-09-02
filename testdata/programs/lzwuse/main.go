package main

import (
	"bytes"
	"compress/lzw"
	"fmt"
	"io"
)

func main() {
	text := []byte("the same words the same words the same words the same words")
	for _, order := range []lzw.Order{lzw.LSB, lzw.MSB} {
		var buf bytes.Buffer
		w := lzw.NewWriter(&buf, order, 8)
		if _, err := w.Write(text); err != nil {
			fmt.Println("write:", err)
			return
		}
		if err := w.Close(); err != nil {
			fmt.Println("close:", err)
			return
		}
		packed := buf.Len()

		r := lzw.NewReader(bytes.NewReader(buf.Bytes()), order, 8)
		back, err := io.ReadAll(r)
		if err != nil {
			fmt.Println("read:", err)
			return
		}
		fmt.Printf("order=%d plain=%d packed=%d equal=%v\n",
			order, len(text), packed, bytes.Equal(text, back))
	}
}
