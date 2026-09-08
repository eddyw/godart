// The program proves the half close of the read side of a connection.
// CloseRead ends the receive direction of the socket: each read of this side
// then gives io.EOF, and the write side stays open.
//
// The server writes nothing after the CloseRead of the client. A BSD system
// answers a byte that arrives after the shutdown with a reset, thus a write of
// the server would end the connection at another moment in each run.
//
// A second CloseRead of the same connection stays out of the program. Go
// writes the call of the system again and the system reports ENOTCONN, and
// RawSocket.shutdown of Dart takes a second call with no report.
package main

import (
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
)

// serve accepts 1 connection, reads it up to its end, and reports the bytes.
// Only main writes to the standard output.
func serve(l net.Listener) string {
	c, err := l.Accept()
	if err != nil {
		return "accept: " + err.Error()
	}
	defer c.Close()
	data, err := io.ReadAll(c)
	if err != nil {
		return "read: " + err.Error()
	}
	return fmt.Sprintf("the server read %q", string(data))
}

// halfRead runs the whole proof over 1 connection that stands open.
type halfRead interface {
	net.Conn
	CloseRead() error
	CloseWrite() error
}

func prove(name string, c halfRead, report chan string) {
	fmt.Println("--", name)
	fmt.Println("   close read:", c.CloseRead())
	n, err := c.Read(make([]byte, 8))
	fmt.Println("   a read after the half close gives", n, "and io.EOF:", err == io.EOF)
	n, err = c.Write([]byte("hello\n"))
	fmt.Println("   a write after the half close passes:", n == 6 && err == nil)
	fmt.Println("   close write:", c.CloseWrite())
	fmt.Println("  ", <-report)
	fmt.Println("   close:", c.Close())
}

func tcp() {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		fmt.Println("listen:", err)
		return
	}
	defer l.Close()
	report := make(chan string, 1)
	go func() { report <- serve(l) }()

	c, err := net.Dial("tcp", l.Addr().String())
	if err != nil {
		fmt.Println("dial:", err)
		return
	}
	prove("a TCP connection", c.(*net.TCPConn), report)
}

func unixSocket() {
	dir, err := os.MkdirTemp("", "gh")
	if err != nil {
		fmt.Println("temp dir:", err)
		return
	}
	defer os.RemoveAll(dir)
	path := filepath.Join(dir, "s")

	l, err := net.Listen("unix", path)
	if err != nil {
		fmt.Println("listen unix:", err)
		return
	}
	defer l.Close()
	report := make(chan string, 1)
	go func() { report <- serve(l) }()

	c, err := net.Dial("unix", path)
	if err != nil {
		fmt.Println("dial unix:", err)
		return
	}
	prove("a connection of the Unix domain", c.(*net.UnixConn), report)
}

// closed proves that each half close of a connection that no longer stands
// gives an error.
func closed() {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		fmt.Println("listen:", err)
		return
	}
	defer l.Close()
	report := make(chan string, 1)
	go func() { report <- serve(l) }()

	c, err := net.Dial("tcp", l.Addr().String())
	if err != nil {
		fmt.Println("dial:", err)
		return
	}
	tcp := c.(*net.TCPConn)
	fmt.Println("-- a connection that ended")
	fmt.Println("   close:", tcp.Close())
	fmt.Println("   close read gives an error:", tcp.CloseRead() != nil)
	fmt.Println("   close write gives an error:", tcp.CloseWrite() != nil)
	fmt.Println("  ", <-report)
}

func main() {
	tcp()
	unixSocket()
	closed()
}
