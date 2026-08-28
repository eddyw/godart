package main

import (
	"fmt"
	"net"
)

// serve accepts 1 connection, reads the request, and writes the answer. Only
// main writes to standard output, thus the order of the lines stays equal.
func serve(l net.Listener, report chan string) {
	c, err := l.Accept()
	if err != nil {
		report <- "accept failed: " + err.Error()
		return
	}
	buf := make([]byte, 16)
	n, err := c.Read(buf)
	if err != nil {
		report <- "read failed: " + err.Error()
		return
	}
	request := string(buf[:n])
	if _, err := c.Write([]byte("pong")); err != nil {
		report <- "write failed: " + err.Error()
		return
	}
	if err := c.Close(); err != nil {
		report <- "close failed: " + err.Error()
		return
	}
	report <- "got " + request
}

func main() {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		fmt.Println("listen failed:", err)
		return
	}
	fmt.Println("network", l.Addr().Network())

	report := make(chan string, 1)
	go serve(l, report)

	c, err := net.Dial("tcp", l.Addr().String())
	if err != nil {
		fmt.Println("dial failed:", err)
		return
	}
	n, err := c.Write([]byte("ping"))
	fmt.Println("wrote", n, err)
	buf := make([]byte, 16)
	n, err = c.Read(buf)
	fmt.Println("read", string(buf[:n]), err)
	fmt.Println("close", c.Close())
	fmt.Println("server", <-report)
	fmt.Println("listener close", l.Close())

	host, port, err := net.SplitHostPort("example.com:443")
	fmt.Println("split", host, port, err)
	fmt.Println("join", net.JoinHostPort(host, port))

	// A UDP dial needs no handshake, thus Go accepts it. The 2 builds only
	// agree on a network name that Go also rejects.
	if _, err := net.Dial("sctp", "127.0.0.1:1"); err != nil {
		fmt.Println("an unknown network gives an error")
	}
}
