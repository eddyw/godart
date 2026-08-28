package main

import "fmt"

// serve accepts 1 connection, reads the request, and writes the answer. It
// gives a report of the work, because only main writes to standard output.
func serve(listener int) string {
	c, cause := accept(listener)
	if cause != "" {
		return "accept failed: " + cause
	}
	buf := make([]byte, 8)
	n, cause := read(c, buf)
	if cause != "" {
		return "read failed: " + cause
	}
	request := string(buf[:n])
	_, cause = write(c, []byte("pong"))
	if cause != "" {
		return "write failed: " + cause
	}
	if closeConn(c) != "" {
		return "close failed"
	}
	return "got " + request
}

func main() {
	listener, port, cause := listen()
	if cause != "" {
		fmt.Println("listen failed:", cause)
		return
	}
	report := make(chan string, 1)
	go func() { report <- serve(listener) }()

	c, cause := dial(port)
	if cause != "" {
		fmt.Println("dial failed:", cause)
		return
	}
	n, cause := write(c, []byte("ping"))
	fmt.Println("wrote", n, cause == "")
	buf := make([]byte, 8)
	n, cause = read(c, buf)
	fmt.Println("read", string(buf[:n]), cause == "")
	fmt.Println("close", closeConn(c) == "")
	fmt.Println("server", <-report)
	fmt.Println("listener close", closeListener(listener) == "")
}
