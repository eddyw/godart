package main

import (
	"fmt"
	"io"
	"os"
	"strings"
)

func main() {
	fmt.Println(io.EOF != nil, io.SeekStart, io.SeekEnd)

	tmp := os.TempDir() + "/gart_iop"
	os.RemoveAll(tmp)
	os.MkdirAll(tmp, 0o755)
	path := tmp + "/a.txt"
	os.WriteFile(path, []byte("hello world\n"), 0o644)

	f, err := os.Open(path)
	if err != nil {
		fmt.Println("open", err)
		return
	}
	all, err := io.ReadAll(f)
	fmt.Println(string(all), err == nil)
	f.Close()

	// A copy from a file into a builder.
	f2, _ := os.Open(path)
	var b strings.Builder
	n, err := io.Copy(&b, f2)
	fmt.Println(n, b.String() == "hello world\n", err == nil)
	f2.Close()

	// A reader of a text.
	r := strings.NewReader("abcdef")
	buf := make([]byte, 3)
	k, err := io.ReadFull(r, buf)
	fmt.Println(k, string(buf), err == nil)

	// A limited reader.
	lr := io.LimitReader(strings.NewReader("abcdef"), 2)
	got, _ := io.ReadAll(lr)
	fmt.Println(string(got))

	// A write of a text into a file through the io interface.
	w, _ := os.Create(tmp + "/b.txt")
	m, err := io.WriteString(w, "written\n")
	fmt.Println(m, err == nil)
	w.Close()
	out, _ := os.ReadFile(tmp + "/b.txt")
	fmt.Println(string(out) == "written\n")

	fmt.Println(io.ReadAll(strings.NewReader("")))

	os.RemoveAll(tmp)
}
