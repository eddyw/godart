// Command httputiluse proves that the package net/http/httputil works.
package main

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"net/http/httputil"
	"net/url"
	"strings"
)

// mask replaces each port that follows the loopback address, because the
// system gives a free port that differs at each run.
func mask(text string) string {
	const host = "127.0.0.1:"
	var b strings.Builder
	for {
		i := strings.Index(text, host)
		if i < 0 {
			b.WriteString(text)
			return b.String()
		}
		b.WriteString(text[:i+len(host)])
		b.WriteString("PORT")
		text = text[i+len(host):]
		j := 0
		for j < len(text) && text[j] >= '0' && text[j] <= '9' {
			j++
		}
		text = text[j:]
	}
}

// show prints a dump line by line, thus the end of each line stays visible.
func show(title string, dump []byte) {
	fmt.Println("--", title)
	for _, line := range strings.Split(strings.TrimRight(string(dump), "\r\n"), "\n") {
		fmt.Printf("   %q\n", mask(strings.TrimRight(line, "\r")))
	}
}

func backend(w http.ResponseWriter, r *http.Request) {
	body, _ := io.ReadAll(r.Body)
	w.Header().Set("X-Backend", "yes")
	w.Header().Set("X-Forwarded-Seen", r.Header.Get("X-Forwarded-For"))
	w.Header().Set("X-Through-Seen", r.Header.Get("X-Through"))
	w.Header().Set("Content-Type", "text/plain")
	w.WriteHeader(http.StatusTeapot)
	fmt.Fprintf(w, "backend saw %s %s host=%s body=%q",
		r.Method, r.URL.Path, r.Host, string(body))
}

func proxyRun() {
	origin := httptest.NewServer(http.HandlerFunc(backend))
	defer origin.Close()
	target, err := url.Parse(origin.URL)
	if err != nil {
		fmt.Println("parse:", err)
		return
	}
	proxy := &httputil.ReverseProxy{
		Rewrite: func(pr *httputil.ProxyRequest) {
			pr.SetURL(target)
			pr.SetXForwarded()
			pr.Out.Header.Set("X-Through", "rewrite")
		},
		ModifyResponse: func(res *http.Response) error {
			res.Header.Set("X-Modified", "yes")
			return nil
		},
	}
	front := httptest.NewServer(proxy)
	defer front.Close()

	res, err := http.Post(front.URL+"/deep/path?q=1", "text/plain",
		strings.NewReader("payload"))
	if err != nil {
		fmt.Println("post:", err)
		return
	}
	body, _ := io.ReadAll(res.Body)
	res.Body.Close()
	fmt.Println("-- the answer through the proxy")
	fmt.Println("   status:", res.StatusCode)
	fmt.Println("   X-Backend:", res.Header.Get("X-Backend"))
	fmt.Println("   X-Modified:", res.Header.Get("X-Modified"))
	fmt.Println("   X-Forwarded-Seen:", res.Header.Get("X-Forwarded-Seen"))
	fmt.Println("   X-Through-Seen:", res.Header.Get("X-Through-Seen"))
	fmt.Println("   body:", mask(string(body)))

	// NewSingleHostReverseProxy joins the path of the target and the path of
	// the request.
	single := httputil.NewSingleHostReverseProxy(target)
	front2 := httptest.NewServer(single)
	defer front2.Close()
	res, err = http.Get(front2.URL + "/single")
	if err != nil {
		fmt.Println("get:", err)
		return
	}
	body, _ = io.ReadAll(res.Body)
	res.Body.Close()
	fmt.Println("-- the answer of NewSingleHostReverseProxy")
	fmt.Println("   status:", res.StatusCode)
	fmt.Println("   body:", mask(string(body)))

	// An error of the backend reaches ErrorHandler.
	dead := &httputil.ReverseProxy{
		Director: func(req *http.Request) {
			req.URL.Scheme = "http"
			req.URL.Host = "127.0.0.1:1"
		},
		ErrorHandler: func(w http.ResponseWriter, r *http.Request, err error) {
			w.WriteHeader(http.StatusBadGateway)
			io.WriteString(w, "the backend gave no answer")
		},
	}
	front3 := httptest.NewServer(dead)
	defer front3.Close()
	res, err = http.Get(front3.URL + "/gone")
	if err != nil {
		fmt.Println("get:", err)
		return
	}
	body, _ = io.ReadAll(res.Body)
	res.Body.Close()
	fmt.Println("-- the handler of an error")
	fmt.Println("   status:", res.StatusCode)
	fmt.Println("   body:", string(body))
}

func dumpRun() {
	body := strings.NewReader("dump body")
	req, err := http.NewRequest("PUT", "http://example.com/path?x=1", body)
	if err != nil {
		fmt.Println("request:", err)
		return
	}
	req.Header.Set("Content-Type", "text/plain")
	req.Header.Set("X-Order", "one")
	req.ContentLength = 9

	dump, err := httputil.DumpRequest(req, true)
	if err != nil {
		fmt.Println("dump:", err)
		return
	}
	show("DumpRequest", dump)

	// The body of the request stands again after a dump, thus DumpRequestOut
	// reads the same bytes.
	dump, err = httputil.DumpRequestOut(req, true)
	if err != nil {
		fmt.Println("dump out:", err)
		return
	}
	show("DumpRequestOut", dump)

	raw := "HTTP/1.1 201 Created\r\nContent-Length: 5\r\n" +
		"Content-Type: text/plain\r\n\r\nhello"
	res, err := http.ReadResponse(bufio.NewReader(strings.NewReader(raw)), req)
	if err != nil {
		fmt.Println("read response:", err)
		return
	}
	dump, err = httputil.DumpResponse(res, true)
	if err != nil {
		fmt.Println("dump response:", err)
		return
	}
	show("DumpResponse", dump)
	rest, _ := io.ReadAll(res.Body)
	res.Body.Close()
	fmt.Println("   the body stands again:", string(rest))
}

func chunkRun() {
	var out bytes.Buffer
	w := httputil.NewChunkedWriter(&out)
	if _, err := io.WriteString(w, "first "); err != nil {
		fmt.Println("write:", err)
		return
	}
	if _, err := io.WriteString(w, "second"); err != nil {
		fmt.Println("write:", err)
		return
	}
	if err := w.Close(); err != nil {
		fmt.Println("close:", err)
		return
	}
	out.WriteString("\r\n")
	fmt.Println("-- the chunked form")
	fmt.Printf("   wire=%q\n", out.String())
	back, err := io.ReadAll(httputil.NewChunkedReader(&out))
	if err != nil {
		fmt.Println("read:", err)
		return
	}
	fmt.Printf("   back=%q\n", string(back))
}

// persistRun drives the deprecated pair ClientConn and ServerConn over 1 pipe.
func persistRun() {
	client, server := net.Pipe()
	done := make(chan string, 1)
	go func() {
		sc := httputil.NewServerConn(server, bufio.NewReader(server))
		req, err := sc.Read()
		if err != nil {
			done <- "read: " + err.Error()
			return
		}
		res := &http.Response{
			StatusCode:    http.StatusOK,
			Proto:         "HTTP/1.1",
			ProtoMajor:    1,
			ProtoMinor:    1,
			Header:        http.Header{"Content-Type": []string{"text/plain"}},
			Body:          io.NopCloser(strings.NewReader("pong")),
			ContentLength: 4,
			Request:       req,
		}
		if err := sc.Write(req, res); err != nil {
			done <- "write: " + err.Error()
			return
		}
		sc.Close()
		done <- "server saw " + req.Method + " " + req.URL.Path
	}()

	cc := httputil.NewClientConn(client, bufio.NewReader(client))
	req, err := http.NewRequest("GET", "http://example.com/ping", nil)
	if err != nil {
		fmt.Println("request:", err)
		return
	}
	res, err := cc.Do(req)
	if err != nil {
		fmt.Println("do:", err)
		return
	}
	body, _ := io.ReadAll(res.Body)
	res.Body.Close()
	cc.Close()
	fmt.Println("-- ClientConn and ServerConn")
	fmt.Println("  ", <-done)
	fmt.Println("   status:", res.StatusCode, "body:", string(body))
	fmt.Println("   ErrPersistEOF:", httputil.ErrPersistEOF)
	fmt.Println("   ErrClosed:", httputil.ErrClosed)
	fmt.Println("   ErrPipeline:", httputil.ErrPipeline)
	fmt.Println("   ErrLineTooLong:", httputil.ErrLineTooLong)

	// NewProxyClientConn writes the whole URL in the request line.
	c2, s2 := net.Pipe()
	go func() {
		buf := make([]byte, 64)
		n, _ := s2.Read(buf)
		done <- strings.SplitN(string(buf[:n]), "\r\n", 2)[0]
		s2.Close()
	}()
	pc := httputil.NewProxyClientConn(c2, bufio.NewReader(c2))
	req, _ = http.NewRequest("GET", "http://example.com/proxied", nil)
	pc.Write(req)
	fmt.Println("   the request line of a proxy:", <-done)
	pc.Close()
}

// poolRun proves that a ReverseProxy takes the buffer of a BufferPool.
type pool struct{ taken int }

func (p *pool) Get() []byte  { p.taken++; return make([]byte, 32*1024) }
func (p *pool) Put(b []byte) {}

func poolRun() {
	origin := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, "pooled body")
	}))
	defer origin.Close()
	target, _ := url.Parse(origin.URL)
	p := &pool{}
	var box httputil.BufferPool = p
	proxy := httputil.NewSingleHostReverseProxy(target)
	proxy.BufferPool = box
	front := httptest.NewServer(proxy)
	defer front.Close()
	res, err := http.Get(front.URL + "/pool")
	if err != nil {
		fmt.Println("get:", err)
		return
	}
	body, _ := io.ReadAll(res.Body)
	res.Body.Close()
	fmt.Println("-- the buffer pool")
	fmt.Println("   body:", string(body))
	fmt.Println("   the pool gave a buffer:", p.taken > 0)
}

func main() {
	proxyRun()
	dumpRun()
	chunkRun()
	persistRun()
	poolRun()
}
