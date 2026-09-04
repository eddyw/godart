// The program proves that a type parameter over a byte slice and a string
// reads its length, its index, its slice, and its 2 conversions.
package main

import "fmt"

type Bytes interface{ ~[]byte | ~string }

func head[T Bytes](v T, n int) T {
	if len(v) < n {
		return v
	}
	return v[:n]
}

func firstByte[T Bytes](v T) byte {
	if len(v) == 0 {
		return 0
	}
	return v[0]
}

func text[T Bytes](v T) string {
	return string(v)
}

func raw[T Bytes](v T) []byte {
	return []byte(v)
}

func middle[T Bytes](v T, a, b int) T {
	return v[a:b]
}

type name string
type blob []byte

func main() {
	s := "hello, world"
	b := []byte(s)

	fmt.Println(head(s, 5), string(head(b, 5)))
	fmt.Println(firstByte(s), firstByte(b))
	fmt.Println(text(s), text(b))
	fmt.Println(len(raw(s)), len(raw(b)))
	fmt.Println(middle(s, 7, 12), string(middle(b, 7, 12)))

	var n name = "named text"
	var q blob = []byte("named bytes")
	fmt.Println(head(n, 5), string(head(q, 5)))
	fmt.Println(text(n), text(q))
	fmt.Println(firstByte(n), firstByte(q))

	fmt.Println(head("ab", 5), string(head([]byte("ab"), 5)))
	fmt.Println(firstByte(""), firstByte([]byte{}))
}
