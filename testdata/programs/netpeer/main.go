package main

import (
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
)

func main() {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		fmt.Println("listen:", err)
		return
	}
	addr := ln.Addr().(*net.TCPAddr)
	fmt.Println("listener", addr.Network(), addr.IP, addr.Port > 0)

	done := make(chan string, 1)
	go func() {
		c, err := ln.Accept()
		if err != nil {
			done <- "accept: " + err.Error()
			return
		}
		local := c.LocalAddr().(*net.TCPAddr)
		remote := c.RemoteAddr().(*net.TCPAddr)
		io.WriteString(c, "hi")
		c.Close()
		done <- fmt.Sprint("server ", local.IP, " ", local.Port == addr.Port,
			" peer ", remote.IP, " ", remote.Port > 0)
	}()

	c, err := net.Dial("tcp", addr.String())
	if err != nil {
		fmt.Println("dial:", err)
		return
	}
	cl := c.LocalAddr().(*net.TCPAddr)
	cr := c.RemoteAddr().(*net.TCPAddr)
	fmt.Println("client", cl.IP, cl.Port > 0, "peer", cr.IP, cr.Port == addr.Port)
	body, _ := io.ReadAll(c)
	fmt.Println("read", string(body))
	c.Close()
	fmt.Println(<-done)
	ln.Close()

	// A server fills Request.RemoteAddr.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		host, _, err := net.SplitHostPort(r.RemoteAddr)
		fmt.Fprintf(w, "%s %v", host, err)
	}))
	res, err := http.Get(srv.URL)
	if err != nil {
		fmt.Println("get:", err)
		return
	}
	text, _ := io.ReadAll(res.Body)
	res.Body.Close()
	srv.Close()
	fmt.Println("remote", strings.TrimSpace(string(text)))
}
