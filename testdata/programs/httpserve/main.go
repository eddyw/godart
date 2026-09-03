// httpserve shows the fields of an http.Server: the 4 timeouts, the limit of
// the head, the hook of the state of a connection, the 2 context hooks, and
// the 2 ways to end the server.
package main

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"
)

type key string

const tag key = "tag"

func main() {
	states()
	contexts()
	headLimit()
	valueCount()
	headerTimeout()
	idleTimeout()
	shutdown()
	closeNow()
	packageServe()
}

// listen binds a port of the loopback address and gives the listener and the
// base of each URL.
func listen() (net.Listener, string) {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		panic(err)
	}
	return l, "http://" + l.Addr().String()
}

// states reads the hook ConnState of 1 connection that answers 1 request.
func states() {
	var mu sync.Mutex
	var seen []string
	done := make(chan struct{})
	server := &http.Server{
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			io.WriteString(w, "ok")
		}),
		ConnState: func(c net.Conn, s http.ConnState) {
			mu.Lock()
			seen = append(seen, s.String())
			last := s == http.StateClosed
			mu.Unlock()
			if last {
				close(done)
			}
		},
	}
	server.SetKeepAlivesEnabled(false)
	l, base := listen()
	go server.Serve(l)
	res, err := http.Get(base + "/")
	if err != nil {
		fmt.Println("states:", err)
		return
	}
	body, _ := io.ReadAll(res.Body)
	res.Body.Close()
	<-done
	mu.Lock()
	fmt.Printf("states %v body=%q close=%v\n", seen, string(body), res.Close)
	mu.Unlock()
	server.Close()
}

// contexts reads BaseContext, ConnContext, ServerContextKey, and
// LocalAddrContextKey.
func contexts() {
	server := &http.Server{}
	server.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		_, ok := ctx.Value(http.ServerContextKey).(*http.Server)
		addr, _ := ctx.Value(http.LocalAddrContextKey).(net.Addr)
		fmt.Fprintf(w, "tag=%v server=%v addr=%v", ctx.Value(tag), ok, addr != nil)
	})
	server.BaseContext = func(l net.Listener) context.Context {
		return context.WithValue(context.Background(), tag, "base")
	}
	server.ConnContext = func(ctx context.Context, c net.Conn) context.Context {
		return context.WithValue(ctx, tag, ctx.Value(tag).(string)+"+conn")
	}
	l, base := listen()
	go server.Serve(l)
	res, err := http.Get(base + "/")
	show("contexts", res, err)
	server.Close()
}

// headLimit reads MaxHeaderBytes. A head that holds more gives the status 431.
func headLimit() {
	server := &http.Server{
		MaxHeaderBytes: 2048,
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			io.WriteString(w, "ok")
		}),
	}
	l, _ := listen()
	go server.Serve(l)
	conn, err := net.Dial("tcp", l.Addr().String())
	if err != nil {
		fmt.Println("headLimit:", err)
		return
	}
	head := "GET / HTTP/1.1\r\nHost: x\r\nX-Long: " + strings.Repeat("a", 16384) + "\r\n\r\n"
	io.WriteString(conn, head)
	line, err := bufio.NewReader(conn).ReadString('\n')
	fmt.Printf("headLimit %q err=%v\n", strings.TrimRight(line, "\r\n"), err != nil)
	conn.Close()
	server.Close()
}

// valueCount reads MaxHeaderValueCount. A head that holds more lines gives the
// status 431.
func valueCount() {
	server := &http.Server{
		MaxHeaderValueCount: 5,
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			io.WriteString(w, "ok")
		}),
	}
	l, _ := listen()
	go server.Serve(l)
	for _, count := range []int{4, 5} {
		conn, err := net.Dial("tcp", l.Addr().String())
		if err != nil {
			fmt.Println("valueCount:", err)
			return
		}
		head := "GET / HTTP/1.1\r\nHost: x\r\n"
		for i := 0; i < count; i++ {
			head += fmt.Sprintf("X-%d: v\r\n", i)
		}
		io.WriteString(conn, head+"\r\n")
		line, _ := bufio.NewReader(conn).ReadString('\n')
		fmt.Printf("valueCount %d %q\n", count, strings.TrimRight(line, "\r\n"))
		conn.Close()
	}
	fmt.Println("valueCount default:", http.DefaultMaxHeaderValueCount)
	server.Close()
}

// headerTimeout reads ReadHeaderTimeout. A client that sends no request loses
// its connection.
func headerTimeout() {
	server := &http.Server{
		ReadHeaderTimeout: 150 * time.Millisecond,
		ReadTimeout:       time.Second,
		WriteTimeout:      time.Second,
		Handler:           http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}),
	}
	l, _ := listen()
	go server.Serve(l)
	conn, err := net.Dial("tcp", l.Addr().String())
	if err != nil {
		fmt.Println("headerTimeout:", err)
		return
	}
	// The server holds the connection until the timeout, and it then ends it.
	conn.SetReadDeadline(time.Now().Add(3 * time.Second))
	_, err = conn.Read(make([]byte, 1))
	fmt.Println("headerTimeout closed:", err != nil)
	conn.Close()
	server.Close()
}

// idleTimeout reads IdleTimeout. A connection that answered 1 request and that
// sends no second request loses its connection.
func idleTimeout() {
	server := &http.Server{
		IdleTimeout: 150 * time.Millisecond,
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			io.WriteString(w, "ok")
		}),
	}
	l, _ := listen()
	go server.Serve(l)
	conn, err := net.Dial("tcp", l.Addr().String())
	if err != nil {
		fmt.Println("idleTimeout:", err)
		return
	}
	io.WriteString(conn, "GET / HTTP/1.1\r\nHost: x\r\n\r\n")
	reader := bufio.NewReader(conn)
	for {
		line, err := reader.ReadString('\n')
		if err != nil || strings.TrimRight(line, "\r\n") == "" {
			break
		}
	}
	io.ReadFull(reader, make([]byte, 2))
	conn.SetReadDeadline(time.Now().Add(3 * time.Second))
	_, err = reader.Read(make([]byte, 1))
	fmt.Println("idleTimeout closed:", err != nil)
	conn.Close()
	server.Close()
}

// shutdown reads Shutdown and RegisterOnShutdown.
func shutdown() {
	hook := make(chan string, 1)
	server := &http.Server{
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			io.WriteString(w, "before")
		}),
	}
	server.RegisterOnShutdown(func() { hook <- "hook ran" })
	l, base := listen()
	ended := make(chan error, 1)
	go func() { ended <- server.Serve(l) }()
	res, err := http.Get(base + "/")
	show("shutdown", res, err)
	http.DefaultClient.CloseIdleConnections()
	err = server.Shutdown(context.Background())
	fmt.Println("shutdown err:", err, "serve:", <-ended == http.ErrServerClosed)
	fmt.Println("shutdown:", <-hook)
	_, err = http.Get(base + "/")
	fmt.Println("shutdown after:", err != nil)
	fmt.Println("shutdown again:", server.ListenAndServe() == http.ErrServerClosed)
}

// closeNow reads Close. Serve then gives ErrServerClosed.
func closeNow() {
	server := &http.Server{
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}),
	}
	l, _ := listen()
	ended := make(chan error, 1)
	go func() { ended <- server.Serve(l) }()
	// The accept stands open, thus the close of the listener ends it.
	time.Sleep(50 * time.Millisecond)
	fmt.Println("close err:", server.Close(), "serve:", <-ended == http.ErrServerClosed)
}

// packageServe reads the function http.Serve.
func packageServe() {
	l, base := listen()
	go http.Serve(l, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, "package serve")
	}))
	res, err := http.Get(base + "/")
	show("packageServe", res, err)
	l.Close()
}

// show writes the answer of a request in 1 line.
func show(name string, res *http.Response, err error) {
	if err != nil {
		fmt.Println(name, "error:", err)
		return
	}
	defer res.Body.Close()
	body, _ := io.ReadAll(res.Body)
	fmt.Printf("%s %d %q\n", name, res.StatusCode, string(body))
}
