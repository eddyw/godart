// The program proves that a method that spells uint8 satisfies an interface
// that spells byte, and that rune and int32 hold the same rule.
package main

import (
	"fmt"
	"io"
	"strings"
)

type reader struct{ inner io.Reader }

func (r *reader) Read(b []uint8) (int, error) { return r.inner.Read(b) }

type taker interface {
	Take(r rune) string
	Give() []byte
}

type box struct{ seen []int32 }

func (b *box) Take(r int32) string {
	b.seen = append(b.seen, r)
	return fmt.Sprintf("%c", r)
}

func (b *box) Give() []uint8 {
	out := make([]byte, 0, len(b.seen))
	for _, r := range b.seen {
		out = append(out, byte(r))
	}
	return out
}

func main() {
	var r io.Reader = &reader{inner: strings.NewReader("abc")}
	out, err := io.ReadAll(r)
	fmt.Printf("%q %v\n", string(out), err)

	var t taker = &box{}
	fmt.Println(t.Take('x'), t.Take('y'))
	fmt.Printf("%q\n", string(t.Give()))

	var w io.Writer = &sink{}
	n, err := w.Write([]uint8("hello"))
	fmt.Println(n, err)
}

type sink struct{}

func (s *sink) Write(p []uint8) (int, error) {
	fmt.Printf("sink got %q\n", string(p))
	return len(p), nil
}
