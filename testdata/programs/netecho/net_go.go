//go:build !dart

package main

import (
	"io"
	"net"
	"strconv"
)

// The Go build holds the same handle model as the interop package, thus the
// program keeps 1 shape. A read at the end of the stream gives 0 with no
// cause, exactly as the interop does.

var listeners = map[int]net.Listener{}
var conns = map[int]net.Conn{}
var handles int

func listen() (int, int, string) {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return -1, 0, err.Error()
	}
	handles++
	listeners[handles] = l
	return handles, l.Addr().(*net.TCPAddr).Port, ""
}

func accept(listener int) (int, string) {
	c, err := listeners[listener].Accept()
	if err != nil {
		return -1, err.Error()
	}
	handles++
	conns[handles] = c
	return handles, ""
}

func closeListener(listener int) string {
	if err := listeners[listener].Close(); err != nil {
		return err.Error()
	}
	delete(listeners, listener)
	return ""
}

func dial(port int) (int, string) {
	c, err := net.Dial("tcp", net.JoinHostPort("127.0.0.1", strconv.Itoa(port)))
	if err != nil {
		return -1, err.Error()
	}
	handles++
	conns[handles] = c
	return handles, ""
}

func write(c int, p []byte) (int, string) {
	n, err := conns[c].Write(p)
	if err != nil {
		return n, err.Error()
	}
	return n, ""
}

func read(c int, p []byte) (int, string) {
	n, err := conns[c].Read(p)
	if err != nil && err != io.EOF {
		return n, err.Error()
	}
	return n, ""
}

func closeConn(c int) string {
	if err := conns[c].Close(); err != nil {
		return err.Error()
	}
	delete(conns, c)
	return ""
}
