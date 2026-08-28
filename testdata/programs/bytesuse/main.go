// Package main uses the package bytes. The compiler reads the real Go source of
// that package, with 1 patch of buffer.go.
package main

import (
	"bytes"
	"fmt"
)

func main() {
	var b bytes.Buffer
	b.WriteString("hello")
	b.WriteByte(' ')
	b.Write([]byte("world"))
	fmt.Println(b.String(), b.Len())

	fmt.Println(bytes.Contains([]byte("seafood"), []byte("foo")))
	fmt.Println(bytes.Equal([]byte("a"), []byte("a")))
	fmt.Println(bytes.Compare([]byte("a"), []byte("b")))
	fmt.Println(bytes.Index([]byte("chicken"), []byte("ken")))
	fmt.Println(bytes.LastIndex([]byte("go gopher"), []byte("go")))
	fmt.Println(bytes.Count([]byte("cheese"), []byte("e")))

	fmt.Println(string(bytes.ToUpper([]byte("abc"))))
	fmt.Println(string(bytes.ToLower([]byte("ABC"))))
	fmt.Println(string(bytes.TrimSpace([]byte("  pad  "))))
	fmt.Println(string(bytes.Repeat([]byte("ab"), 3)))
	fmt.Println(string(bytes.Join([][]byte{[]byte("a"), []byte("b")}, []byte("-"))))

	parts := bytes.Split([]byte("a,b,c"), []byte(","))
	for _, p := range parts {
		fmt.Print(string(p), " ")
	}
	fmt.Println()

	fmt.Println(bytes.HasPrefix([]byte("gopher"), []byte("go")))
	fmt.Println(bytes.HasSuffix([]byte("gopher"), []byte("her")))
	fmt.Println(string(bytes.ReplaceAll([]byte("aaa"), []byte("a"), []byte("b"))))

	r := bytes.NewReader([]byte("stream"))
	buf := make([]byte, 3)
	n, err := r.Read(buf)
	fmt.Println(n, string(buf[:n]), err)

	var grow bytes.Buffer
	for i := 0; i < 100; i++ {
		grow.WriteString("x")
	}
	fmt.Println(grow.Len(), grow.Cap() >= 100)
}
