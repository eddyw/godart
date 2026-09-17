package main

import (
	"fmt"
	"io"
	"strings"
)

// A select statement of 1 goroutine meets a select statement of another one.
// io.Pipe holds that shape.
func rendezvous() {
	done := make(chan struct{})
	wrCh := make(chan []byte)
	rdCh := make(chan int)
	out := make(chan int)

	go func() {
		select {
		case <-done:
			out <- -1
			return
		default:
		}
		select {
		case bw := <-wrCh:
			rdCh <- len(bw)
			out <- len(bw)
		case <-done:
			out <- -1
		}
	}()

	b := []byte("hello")
	n := 0
	for once := true; once || len(b) > 0; once = false {
		select {
		case wrCh <- b:
			nw := <-rdCh
			b = b[nw:]
			n += nw
		case <-done:
			fmt.Println("done")
			return
		}
	}
	fmt.Println("wrote", n, "read", <-out)
}

func pipe() {
	r, w := io.Pipe()
	type result struct {
		n   int
		err error
	}
	done := make(chan result)
	go func() {
		n, err := w.Write([]byte("hello, world"))
		done <- result{n, err}
	}()
	buf := make([]byte, 64)
	n, err := r.Read(buf)
	fmt.Println("read", n, err, string(buf[:n]))
	written := <-done
	fmt.Println("wrote", written.n, written.err)
	fmt.Println("done", written.n)
	w.Close()
	r.Close()
}

func copyThrough() {
	r, w := io.Pipe()
	go func() {
		io.Copy(w, strings.NewReader("a longer text that crosses the pipe"))
		w.Close()
	}()
	text, err := io.ReadAll(r)
	fmt.Println(string(text), err)
}

func closeFirst() {
	r, w := io.Pipe()
	r.Close()
	n, err := w.Write([]byte("x"))
	fmt.Println("after close", n, err)
	m, err := r.Read(make([]byte, 4))
	fmt.Println("read after close", m, err)
}

func main() {
	rendezvous()
	pipe()
	copyThrough()
	closeFirst()
}
