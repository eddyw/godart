package main

import (
	"fmt"
	"net"
	"time"
)

func main() {
	for _, host := range []string{"127.0.0.1", "::1"} {
		probe, err := net.ListenPacket("udp", net.JoinHostPort(host, "0"))
		if err != nil {
			panic(err)
		}
		addr := probe.LocalAddr().String()
		probe.Close()
		c, err := net.Dial("udp", addr)
		if err != nil {
			panic(err)
		}
		c.SetDeadline(time.Now().Add(500 * time.Millisecond))
		_, err = c.Write([]byte("one"))
		if err != nil {
			panic(err)
		}
		buffer := make([]byte, 8)
		_, err = c.Read(buffer)
		timeout := false
		if e, ok := err.(interface{ Timeout() bool }); ok {
			timeout = e.Timeout()
		}
		fmt.Println(host, "refused", err != nil, !timeout)
		server, err := net.ListenPacket("udp", addr)
		if err != nil {
			panic(err)
		}
		c.SetDeadline(time.Now().Add(time.Second))
		server.SetDeadline(time.Now().Add(time.Second))
		_, err = c.Write([]byte("two"))
		if err != nil {
			panic(err)
		}
		n, peer, err := server.ReadFrom(buffer)
		if err != nil {
			panic(err)
		}
		fmt.Println("reuse", string(buffer[:n]))
		_, err = server.WriteTo([]byte("ok"), peer)
		if err != nil {
			panic(err)
		}
		n, err = c.Read(buffer)
		if err != nil {
			panic(err)
		}
		fmt.Println("reply", string(buffer[:n]))
		c.SetReadDeadline(time.Now().Add(20 * time.Millisecond))
		_, err = c.Read(buffer)
		e, ok := err.(interface{ Timeout() bool })
		fmt.Println("deadline", ok && e.Timeout())
		c.Close()
		server.Close()
	}
}
