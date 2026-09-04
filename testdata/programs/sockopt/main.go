// The program proves the socket options that dart:io reaches. Go writes each
// option on the socket of the system, and the runtime writes the same option
// through setRawOption with the number of the platform.
package main

import (
	"fmt"
	"net"
)

func main() {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		fmt.Println("listen", err)
		return
	}
	defer ln.Close()

	done := make(chan struct{})
	go func() {
		c, err := ln.Accept()
		if err == nil {
			c.Close()
		}
		close(done)
	}()

	c, err := net.Dial("tcp", ln.Addr().String())
	if err != nil {
		fmt.Println("dial", err)
		return
	}
	tcp := c.(*net.TCPConn)
	fmt.Println("nodelay", tcp.SetNoDelay(true))
	fmt.Println("keepalive", tcp.SetKeepAlive(true))
	fmt.Println("rcvbuf", tcp.SetReadBuffer(32768))
	fmt.Println("sndbuf", tcp.SetWriteBuffer(32768))
	c.Close()
	<-done

	pc, err := net.ListenPacket("udp", "127.0.0.1:0")
	if err != nil {
		fmt.Println("packet", err)
		return
	}
	udp := pc.(*net.UDPConn)
	fmt.Println("udp rcvbuf", udp.SetReadBuffer(32768))
	fmt.Println("udp sndbuf", udp.SetWriteBuffer(32768))
	pc.Close()
}
