//go:build dart

package main

import "syscall/dart"

// The 7 functions below hold the network access of the Dart build. Each one
// carries a handle of the runtime, and the Go build holds the same shape.

func listen() (int, int, string) {
	return dart.ListenTCP("127.0.0.1", 0, false)
}

func accept(listener int) (int, string) {
	return dart.Accept(listener)
}

func closeListener(listener int) string {
	return dart.ListenClose(listener)
}

func dial(port int) (int, string) {
	return dart.DialTCP("127.0.0.1", port)
}

func write(c int, p []byte) (int, string) {
	return dart.ConnWrite(c, p)
}

func read(c int, p []byte) (int, string) {
	return dart.ConnRead(c, p)
}

func closeConn(c int) string {
	return dart.ConnClose(c)
}
