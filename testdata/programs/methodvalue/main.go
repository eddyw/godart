package main

import (
	"fmt"
	"io"
)

// mover holds its step in a field of a function type. The compressor of
// compress/flate holds the same shape.
type mover struct {
	step func(*mover)
	w    io.Writer
	n    int
}

// send blocks on the pipe, and it raises the count after the write.
func (m *mover) send() {
	m.w.Write([]byte{1})
	m.n++
}

type sender interface{ send() }

func main() {
	r, w := io.Pipe()
	go func() {
		defer w.Close()
		m := &mover{w: w}
		m.step = (*mover).send
		m.step(m)
		fmt.Println("after the expression", m.n)
		var s sender = m
		f := s.send
		f()
		fmt.Println("after the value", m.n)
	}()
	b, err := io.ReadAll(r)
	fmt.Println("bytes", len(b), err)
}
