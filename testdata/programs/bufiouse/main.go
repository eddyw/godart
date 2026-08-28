package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

func lines() {
	r := bufio.NewReader(strings.NewReader("first\nsecond\nthird"))
	for {
		line, err := r.ReadString('\n')
		fmt.Printf("line %q %v\n", line, err)
		if err != nil {
			return
		}
	}
}

func words() {
	s := bufio.NewScanner(strings.NewReader("alpha beta\ngamma\n"))
	s.Split(bufio.ScanWords)
	for s.Scan() {
		fmt.Println("word", s.Text())
	}
	fmt.Println("scan error", s.Err())
}

func peek() {
	r := bufio.NewReader(strings.NewReader("peek me"))
	p, err := r.Peek(4)
	fmt.Println("peek", string(p), err)
	b, _ := r.ReadByte()
	fmt.Println("byte", string(b))
	if err := r.UnreadByte(); err != nil {
		fmt.Println("unread error", err)
	}
	rest, _ := r.ReadString('\n')
	fmt.Println("rest", rest)
}

func buffer() {
	w := bufio.NewWriterSize(os.Stdout, 8)
	fmt.Fprint(w, "abc")
	fmt.Println("buffered", w.Buffered(), "available", w.Available())
	w.WriteString("de")
	w.WriteByte('f')
	w.WriteRune('g')
	w.Flush()
	fmt.Println()
}

// readWriter reads through the embedded pointer of bufio.ReadWriter. The method
// set of that struct holds each method of the 2 fields.
func readWriter() {
	rw := bufio.NewReadWriter(
		bufio.NewReader(strings.NewReader("zy")),
		bufio.NewWriter(os.Stdout),
	)
	b, _ := rw.ReadByte()
	fmt.Println("readwriter", string(b))
	if _, err := rw.WriteString("through the writer\n"); err != nil {
		fmt.Println("write error", err)
	}
	rw.Flush()
}

func main() {
	lines()
	words()
	peek()
	buffer()
	readWriter()
}
