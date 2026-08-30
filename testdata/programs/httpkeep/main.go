package main

import (
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"strings"
	"sync/atomic"
)

// countListener counts each connection that the server accepts.
type countListener struct {
	inner net.Listener
	count *int64
}

func (l *countListener) Accept() (net.Conn, error) {
	c, err := l.inner.Accept()
	if err == nil {
		atomic.AddInt64(l.count, 1)
	}
	return c, err
}

func (l *countListener) Close() error   { return l.inner.Close() }
func (l *countListener) Addr() net.Addr { return l.inner.Addr() }

func main() {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		fmt.Println("listen:", err)
		os.Exit(1)
	}
	var conns int64
	mux := http.NewServeMux()
	mux.HandleFunc("/hello", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "hello")
	})
	mux.HandleFunc("/echo", func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		fmt.Fprintf(w, "echo %d", len(body))
	})
	mux.HandleFunc("/skip", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "skip")
	})
	server := &http.Server{Handler: mux}
	go server.Serve(&countListener{inner: listener, count: &conns})

	base := "http://" + listener.Addr().String()
	client := &http.Client{}

	// 3 requests in a row share 1 connection.
	for i := 0; i < 3; i++ {
		fmt.Println(get(client, base+"/hello"), atomic.LoadInt64(&conns))
	}

	// A POST reads its body on the server, thus the connection stands open.
	res, err := client.Post(base+"/echo", "text/plain", strings.NewReader("abcdef"))
	if err != nil {
		fmt.Println("post:", err)
		os.Exit(1)
	}
	body, _ := io.ReadAll(res.Body)
	res.Body.Close()
	fmt.Println(string(body), atomic.LoadInt64(&conns))

	// A handler that reads no body leaves 6 bytes. The server reads them.
	res, err = client.Post(base+"/skip", "text/plain", strings.NewReader("abcdef"))
	if err != nil {
		fmt.Println("skip:", err)
		os.Exit(1)
	}
	body, _ = io.ReadAll(res.Body)
	res.Body.Close()
	fmt.Println(string(body), atomic.LoadInt64(&conns))

	// A client with no keep alive opens 1 connection for each request.
	off := &http.Client{Transport: &http.Transport{DisableKeepAlives: true}}
	for i := 0; i < 2; i++ {
		fmt.Println(get(off, base+"/hello"), atomic.LoadInt64(&conns))
	}

	client.CloseIdleConnections()
	server.Close()
}

func get(c *http.Client, target string) string {
	res, err := c.Get(target)
	if err != nil {
		return "error " + err.Error()
	}
	body, err := io.ReadAll(res.Body)
	res.Body.Close()
	if err != nil {
		return "read " + err.Error()
	}
	return string(body)
}
