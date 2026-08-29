package main

import (
	"context"
	"errors"
	"fmt"
	"net"
	"os"
	"time"
)

// The program reads the real net API with no build tag. The go command takes
// the real package, and gart takes the overlay. Each print holds no address of
// the system, because a free port and a temporary path differ at each run.

func stop(what string, err error) {
	if err != nil {
		fmt.Println(what, "failed:", err)
		os.Exit(1)
	}
}

// timedOut reports whether the error names a deadline that passed.
func timedOut(err error) bool {
	e, ok := err.(net.Error)
	return ok && e.Timeout()
}

func udp() {
	server, err := net.ListenPacket("udp", "127.0.0.1:0")
	stop("listen udp", err)
	defer server.Close()
	client, err := net.ListenPacket("udp", "127.0.0.1:0")
	stop("listen udp client", err)
	defer client.Close()

	fmt.Println("udp network:", server.LocalAddr().Network())

	n, err := client.WriteTo([]byte("ping"), server.LocalAddr())
	stop("write udp", err)
	fmt.Println("udp write:", n)

	buf := make([]byte, 16)
	n, from, err := server.ReadFrom(buf)
	stop("read udp", err)
	fmt.Println("udp read:", n, string(buf[:n]), from.Network())

	n, err = server.WriteTo([]byte("pong"), from)
	stop("write udp back", err)
	n, _, err = client.ReadFrom(buf)
	stop("read udp back", err)
	fmt.Println("udp back:", n, string(buf[:n]))

	stop("set deadline", client.SetReadDeadline(time.Now().Add(20*time.Millisecond)))
	_, _, err = client.ReadFrom(buf)
	fmt.Println("udp deadline:", timedOut(err))
}

func udpDial() {
	server, err := net.ListenPacket("udp", "127.0.0.1:0")
	stop("listen udp", err)
	defer server.Close()

	c, err := net.Dial("udp", server.LocalAddr().String())
	stop("dial udp", err)
	defer c.Close()

	n, err := c.Write([]byte("hi"))
	stop("write udp", err)
	fmt.Println("udp dial write:", n)

	buf := make([]byte, 16)
	n, from, err := server.ReadFrom(buf)
	stop("read udp", err)
	fmt.Println("udp dial read:", n, string(buf[:n]))

	_, err = server.WriteTo([]byte("ho"), from)
	stop("write udp back", err)
	n, err = c.Read(buf)
	stop("read udp back", err)
	fmt.Println("udp dial back:", n, string(buf[:n]))
}

func unix() {
	dir, err := os.MkdirTemp("", "n")
	stop("temp dir", err)
	defer os.RemoveAll(dir)
	path := dir + "/s"

	ln, err := net.Listen("unix", path)
	stop("listen unix", err)
	fmt.Println("unix network:", ln.Addr().Network())

	go func() {
		s, err := ln.Accept()
		if err != nil {
			return
		}
		buf := make([]byte, 8)
		n, err := s.Read(buf)
		if err != nil {
			return
		}
		s.Write(buf[:n])
		s.Close()
	}()

	c, err := net.Dial("unix", path)
	stop("dial unix", err)
	_, err = c.Write([]byte("unix"))
	stop("write unix", err)
	buf := make([]byte, 8)
	n, err := c.Read(buf)
	stop("read unix", err)
	fmt.Println("unix read:", string(buf[:n]))
	c.Close()

	stop("close unix", ln.Close())
	_, err = os.Stat(path)
	fmt.Println("unix file removed:", os.IsNotExist(err))
}

func deadline() {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	stop("listen tcp", err)
	defer ln.Close()

	go func() {
		s, err := ln.Accept()
		if err != nil {
			return
		}
		// The server writes nothing, thus the read of the client waits.
		buf := make([]byte, 1)
		s.Read(buf)
	}()

	c, err := net.Dial("tcp", ln.Addr().String())
	stop("dial tcp", err)
	defer c.Close()

	stop("set write deadline", c.SetWriteDeadline(time.Now().Add(time.Second)))
	stop("set read deadline", c.SetReadDeadline(time.Now().Add(20*time.Millisecond)))
	buf := make([]byte, 4)
	n, err := c.Read(buf)
	fmt.Println("tcp deadline:", n, timedOut(err))
}

func cancel() {
	ctx, stopCtx := context.WithCancel(context.Background())
	stopCtx()
	c, err := (&net.Dialer{}).DialContext(ctx, "tcp", "127.0.0.1:1")
	fmt.Println("cancel:", c == nil, errors.Is(err, context.Canceled))

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	stop("listen tcp", err)
	defer ln.Close()
	open, err := (&net.Dialer{Timeout: 5 * time.Second}).Dial("tcp", ln.Addr().String())
	fmt.Println("dialer:", err == nil, open != nil)
	if open != nil {
		open.Close()
	}

	timed, err := net.DialTimeout("tcp", ln.Addr().String(), 5*time.Second)
	fmt.Println("timeout dial:", err == nil, timed != nil)
	if timed != nil {
		timed.Close()
	}
}

func main() {
	udp()
	udpDial()
	unix()
	deadline()
	cancel()
}
