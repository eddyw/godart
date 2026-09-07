// The program proves the half close of a TCP connection. The server reads up
// to the end of the stream, thus only the close of the write side of the
// client ends that read, and the client still reads the answer after it.
package main

import (
	"fmt"
	"io"
	"net"
)

// serve accepts 1 connection, reads it up to its end, writes the count of the
// bytes, and closes it. It gives a report, because only main writes to the
// standard output.
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
	if _, err := fmt.Fprintf(c, "%d bytes\n", len(data)); err != nil {
		return "write: " + err.Error()
	}
	return "served"
}

func main() {
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
	for _, line := range []string{"one\n", "two\n", "three\n"} {
		if _, err := tcp.Write([]byte(line)); err != nil {
			fmt.Println("write:", err)
			return
		}
	}
	if err := tcp.CloseWrite(); err != nil {
		fmt.Println("close write:", err)
		return
	}
	_, err = tcp.Write([]byte("late\n"))
	fmt.Println("a write after the half close fails:", err != nil)
	answer, err := io.ReadAll(tcp)
	if err != nil {
		fmt.Println("read:", err)
		return
	}
	fmt.Print(string(answer))
	fmt.Println(<-report)
	fmt.Println("close:", tcp.Close())
}
