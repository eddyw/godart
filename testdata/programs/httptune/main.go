// httptune shows the tuning fields of an http.Transport: the limits of the
// pool, the 2 timeouts of an exchange, the sizes of the buffers, the copy of a
// transport, and the RoundTripper of another scheme.
package main

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

func main() {
	clone()
	registerProtocol()
	idlePool()
	maxConns()
	idleTimeout()
	headerTimeout()
	expectContinue()
	buffers()
	headLimit()
	fileProtocol()
	redirectBody()
	customDial()
}

// listen binds a port of the loopback address.
func listen(handler http.Handler) (*http.Server, string, net.Listener) {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		panic(err)
	}
	server := &http.Server{Handler: handler}
	go server.Serve(l)
	return server, "http://" + l.Addr().String(), l
}

// ok answers each request with a short body.
var ok = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
	io.WriteString(w, "ok")
})

// clone reads Transport.Clone.
func clone() {
	t := &http.Transport{
		MaxIdleConns:          7,
		MaxIdleConnsPerHost:   3,
		MaxConnsPerHost:       5,
		IdleConnTimeout:       2 * time.Second,
		ResponseHeaderTimeout: 3 * time.Second,
		ExpectContinueTimeout: 4 * time.Second,
		WriteBufferSize:       1024,
		ReadBufferSize:        2048,
		DisableKeepAlives:     true,
		DisableCompression:    true,
		ForceAttemptHTTP2:     true,
		ProxyConnectHeader:    http.Header{"X-Proxy": {"1"}},
	}
	c := t.Clone()
	c.MaxIdleConns = 9
	c.ProxyConnectHeader.Set("X-Proxy", "2")
	fmt.Println("clone idle", t.MaxIdleConns, c.MaxIdleConns)
	fmt.Println("clone perHost", c.MaxIdleConnsPerHost, "conns", c.MaxConnsPerHost)
	fmt.Println("clone timeouts", c.IdleConnTimeout, c.ResponseHeaderTimeout,
		c.ExpectContinueTimeout)
	fmt.Println("clone buffers", c.WriteBufferSize, c.ReadBufferSize)
	fmt.Println("clone flags", c.DisableKeepAlives, c.DisableCompression,
		c.ForceAttemptHTTP2)
	fmt.Println("clone header", t.ProxyConnectHeader.Get("X-Proxy"),
		c.ProxyConnectHeader.Get("X-Proxy"))
	fmt.Println("default per host", http.DefaultMaxIdleConnsPerHost)
}

// A memoryRT answers each request of its scheme with no network.
type memoryRT struct{ body string }

func (m memoryRT) RoundTrip(r *http.Request) (*http.Response, error) {
	if r.URL.Path == "/skip" {
		return nil, http.ErrSkipAltProtocol
	}
	return &http.Response{
		StatusCode:    200,
		Status:        "200 OK",
		Proto:         "HTTP/1.1",
		ProtoMajor:    1,
		ProtoMinor:    1,
		Header:        http.Header{},
		Body:          io.NopCloser(strings.NewReader(m.body + " " + r.URL.Path)),
		ContentLength: -1,
		Request:       r,
	}, nil
}

// registerProtocol reads Transport.RegisterProtocol and ErrSkipAltProtocol.
func registerProtocol() {
	t := &http.Transport{}
	t.RegisterProtocol("memory", memoryRT{body: "from memory"})
	res, err := t.RoundTrip(mustRequest("GET", "memory://host/one"))
	if err != nil {
		fmt.Println("protocol:", err)
		return
	}
	body, _ := io.ReadAll(res.Body)
	res.Body.Close()
	fmt.Printf("protocol %d %q\n", res.StatusCode, string(body))
	// A result of ErrSkipAltProtocol gives the request back to the transport,
	// and the transport holds no such scheme.
	_, err = t.RoundTrip(mustRequest("GET", "memory://host/skip"))
	fmt.Println("protocol skip:", err != nil)
	fmt.Println("protocol twice:", trap(func() {
		t.RegisterProtocol("memory", memoryRT{})
	}))
}

// idlePool reads MaxIdleConns and MaxIdleConnsPerHost.
func idlePool() {
	server, base, l := listen(ok)
	defer server.Close()
	defer l.Close()
	t := &http.Transport{MaxIdleConnsPerHost: 1}
	// 2 requests that stand open at the same time need 2 connections, and the
	// pool keeps 1 of them.
	hold := make(chan struct{})
	var wg sync.WaitGroup
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			res, err := t.RoundTrip(mustRequest("GET", base+"/"))
			if err != nil {
				fmt.Println("idlePool:", err)
				return
			}
			<-hold
			io.ReadAll(res.Body)
			res.Body.Close()
		}()
	}
	time.Sleep(100 * time.Millisecond)
	close(hold)
	wg.Wait()
	fmt.Println("idlePool done")
	t.CloseIdleConnections()
}

// maxConns reads MaxConnsPerHost. The second request waits for the first one.
func maxConns() {
	var live, top int
	var mu sync.Mutex
	server, base, l := listen(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		live++
		if live > top {
			top = live
		}
		mu.Unlock()
		time.Sleep(80 * time.Millisecond)
		mu.Lock()
		live--
		mu.Unlock()
		io.WriteString(w, "ok")
	}))
	defer server.Close()
	defer l.Close()
	t := &http.Transport{MaxConnsPerHost: 1, DisableKeepAlives: true}
	var wg sync.WaitGroup
	for i := 0; i < 3; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			res, err := t.RoundTrip(mustRequest("GET", base+"/"))
			if err != nil {
				fmt.Println("maxConns:", err)
				return
			}
			io.ReadAll(res.Body)
			res.Body.Close()
		}()
	}
	wg.Wait()
	mu.Lock()
	fmt.Println("maxConns top:", top)
	mu.Unlock()
}

// idleTimeout reads IdleConnTimeout. A connection of the pool that stood open
// too long never carries a second request.
func idleTimeout() {
	var count int
	var mu sync.Mutex
	server, base, l := listen(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, "ok")
	}))
	defer server.Close()
	defer l.Close()
	server.ConnState = func(c net.Conn, s http.ConnState) {
		if s == http.StateNew {
			mu.Lock()
			count++
			mu.Unlock()
		}
	}
	t := &http.Transport{IdleConnTimeout: 100 * time.Millisecond}
	get(t, base)
	get(t, base)
	mu.Lock()
	fmt.Println("idleTimeout reuse:", count)
	mu.Unlock()
	time.Sleep(250 * time.Millisecond)
	get(t, base)
	mu.Lock()
	fmt.Println("idleTimeout after:", count)
	mu.Unlock()
	t.CloseIdleConnections()
}

// headerTimeout reads ResponseHeaderTimeout.
func headerTimeout() {
	server, base, l := listen(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(400 * time.Millisecond)
		io.WriteString(w, "late")
	}))
	defer server.Close()
	defer l.Close()
	t := &http.Transport{ResponseHeaderTimeout: 100 * time.Millisecond}
	_, err := t.RoundTrip(mustRequest("GET", base+"/"))
	fmt.Println("headerTimeout:", err != nil)
	t.CloseIdleConnections()
}

// expectContinue reads ExpectContinueTimeout. The server answers 100, thus the
// body reaches the wire.
func expectContinue() {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		fmt.Println("expectContinue:", err)
		return
	}
	defer l.Close()
	go func() {
		conn, err := l.Accept()
		if err != nil {
			return
		}
		defer conn.Close()
		reader := bufio.NewReader(conn)
		for {
			line, err := reader.ReadString('\n')
			if err != nil {
				return
			}
			if strings.TrimRight(line, "\r\n") == "" {
				break
			}
		}
		io.WriteString(conn, "HTTP/1.1 100 Continue\r\n\r\n")
		body := make([]byte, 5)
		io.ReadFull(reader, body)
		io.WriteString(conn, "HTTP/1.1 200 OK\r\nContent-Length: "+
			fmt.Sprint(len(body))+"\r\n\r\n"+string(body))
	}()
	req := mustRequest("POST", "http://"+l.Addr().String()+"/")
	req.Body = io.NopCloser(strings.NewReader("hello"))
	req.ContentLength = 5
	req.Header.Set("Expect", "100-continue")
	t := &http.Transport{ExpectContinueTimeout: time.Second}
	res, err := t.RoundTrip(req)
	if err != nil {
		fmt.Println("expectContinue:", err)
		return
	}
	got, _ := io.ReadAll(res.Body)
	res.Body.Close()
	fmt.Printf("expectContinue %d %q\n", res.StatusCode, string(got))
}

// buffers reads WriteBufferSize and ReadBufferSize. A small buffer changes no
// answer.
func buffers() {
	server, base, l := listen(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		fmt.Fprintf(w, "%d bytes", len(body))
	}))
	defer server.Close()
	defer l.Close()
	t := &http.Transport{WriteBufferSize: 16, ReadBufferSize: 16}
	req := mustRequest("POST", base+"/")
	req.Body = io.NopCloser(strings.NewReader(strings.Repeat("x", 300)))
	req.ContentLength = 300
	res, err := t.RoundTrip(req)
	if err != nil {
		fmt.Println("buffers:", err)
		return
	}
	body, _ := io.ReadAll(res.Body)
	res.Body.Close()
	fmt.Printf("buffers %q\n", string(body))
	t.CloseIdleConnections()
}

// headLimit reads MaxResponseHeaderBytes.
func headLimit() {
	server, base, l := listen(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Long", strings.Repeat("a", 2048))
		io.WriteString(w, "ok")
	}))
	defer server.Close()
	defer l.Close()
	t := &http.Transport{MaxResponseHeaderBytes: 512}
	_, err := t.RoundTrip(mustRequest("GET", base+"/"))
	fmt.Println("headLimit:", err != nil)
	t.CloseIdleConnections()
}

// fileProtocol reads NewFileTransport and NewFileTransportFS.
func fileProtocol() {
	dir, err := os.MkdirTemp("", "tune")
	if err != nil {
		fmt.Println("fileProtocol:", err)
		return
	}
	defer os.RemoveAll(dir)
	if err := os.WriteFile(filepath.Join(dir, "hello.txt"),
		[]byte("hello from a file\n"), 0o644); err != nil {
		fmt.Println("fileProtocol write:", err)
		return
	}
	t := &http.Transport{}
	t.RegisterProtocol("file", http.NewFileTransport(http.Dir(dir)))
	res, err := t.RoundTrip(mustRequest("GET", "file:///hello.txt"))
	if err != nil {
		fmt.Println("fileProtocol get:", err)
		return
	}
	body, _ := io.ReadAll(res.Body)
	res.Body.Close()
	fmt.Printf("file %d %q\n", res.StatusCode, string(body))

	missing, err := t.RoundTrip(mustRequest("GET", "file:///gone.txt"))
	if err != nil {
		fmt.Println("fileProtocol missing:", err)
		return
	}
	io.ReadAll(missing.Body)
	missing.Body.Close()
	fmt.Println("file missing", missing.StatusCode)

	t2 := &http.Transport{}
	t2.RegisterProtocol("file", http.NewFileTransportFS(os.DirFS(dir)))
	res2, err := t2.RoundTrip(mustRequest("GET", "file:///hello.txt"))
	if err != nil {
		fmt.Println("fileProtocol fs:", err)
		return
	}
	body2, _ := io.ReadAll(res2.Body)
	res2.Body.Close()
	fmt.Printf("file fs %d %q\n", res2.StatusCode, string(body2))
}

// redirectBody reads Request.Response and the body of a redirect of the status
// 307.
func redirectBody() {
	server, base, l := listen(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/from" {
			http.Redirect(w, r, "/to", http.StatusTemporaryRedirect)
			return
		}
		body, _ := io.ReadAll(r.Body)
		fmt.Fprintf(w, "%s %q response=%v", r.Method, string(body),
			r.URL.Path)
	}))
	defer server.Close()
	defer l.Close()
	client := &http.Client{CheckRedirect: func(r *http.Request, via []*http.Request) error {
		fmt.Println("redirect from", r.Response.StatusCode, "to", r.URL.Path,
			"method", r.Method, "via", len(via))
		return nil
	}}
	res, err := client.Post(base+"/from", "text/plain", strings.NewReader("payload"))
	if err != nil {
		fmt.Println("redirectBody:", err)
		return
	}
	body, _ := io.ReadAll(res.Body)
	res.Body.Close()
	fmt.Printf("redirectBody %d %q\n", res.StatusCode, string(body))
	client.CloseIdleConnections()
}

// customDial reads Transport.Dial, Transport.DialContext, and
// Transport.DialTLSContext. Each 1 opens the connection of a request, thus the
// URL of the request names a host that no name server holds.
//
// A request of the scheme https reaches Dial and DialContext no time, because
// the secure socket of Dart takes no socket that stands open. Such a request
// gives a clear error, and Go layers TLS over the socket instead.
func customDial() {
	server, _, l := listen(ok)
	defer server.Close()
	defer l.Close()
	real := l.Addr().String()

	calls := 0
	t := &http.Transport{DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
		calls++
		return net.Dial(network, real)
	}}
	show("dialContext", t, "http://elsewhere.invalid/")
	fmt.Println("dialContext calls:", calls)
	t.CloseIdleConnections()

	plain := 0
	t2 := &http.Transport{Dial: func(network, addr string) (net.Conn, error) {
		plain++
		return net.Dial(network, real)
	}}
	show("dial", t2, "http://elsewhere.invalid/")
	fmt.Println("dial calls:", plain)
	t2.CloseIdleConnections()

	t3 := &http.Transport{DialTLSContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
		return net.Dial(network, real)
	}}
	show("dialTLS", t3, "https://elsewhere.invalid/")
	t3.CloseIdleConnections()

	t4 := &http.Transport{DialTLS: func(network, addr string) (net.Conn, error) {
		return net.Dial(network, real)
	}}
	show("dialTLS plain", t4, "https://elsewhere.invalid/")
	t4.CloseIdleConnections()
}

// show sends 1 request over the transport and writes the answer in 1 line.
func show(name string, t *http.Transport, target string) {
	res, err := t.RoundTrip(mustRequest("GET", target))
	if err != nil {
		fmt.Println(name+":", err)
		return
	}
	body, _ := io.ReadAll(res.Body)
	res.Body.Close()
	fmt.Printf("%s %d %q\n", name, res.StatusCode, string(body))
}

// get sends 1 request and reads the whole answer.
func get(t *http.Transport, base string) {
	res, err := t.RoundTrip(mustRequest("GET", base+"/"))
	if err != nil {
		fmt.Println("get:", err)
		return
	}
	io.ReadAll(res.Body)
	res.Body.Close()
}

// mustRequest builds a request and stops the program for a bad target.
func mustRequest(method, target string) *http.Request {
	req, err := http.NewRequest(method, target, nil)
	if err != nil {
		panic(err)
	}
	return req
}

// trap runs the function and gives the text of the panic that it raises.
func trap(f func()) (text string) {
	defer func() {
		if p := recover(); p != nil {
			text = fmt.Sprint(p)
		}
	}()
	f()
	return "no panic"
}
